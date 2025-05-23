package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type pcieInfo struct {
	Index             int
	UUID              string
	LinkWidth         int
	MaxLinkWidth      int
	LinkSpeed         int
	LinkGeneration    int
	MaxLinkGeneration int
	MaxPcieGeneration int
}

func main() {
	// CheckRoot()
	var reset bool
	var gpuIndex int
	var pcieWidth bool
	var miniFan bool
	var fanSpeed int

	flag.BoolVar(&reset, "r", false, "Reset fan speed to default")
	flag.IntVar(&gpuIndex, "i", -1, "Specify GPU index, if not means all GPUs")
	flag.BoolVar(&miniFan, "m", false, "mini gpu fans speed")
	flag.IntVar(&fanSpeed, "f", 100, "Specify fan speed, range from 0 to 100")
	flag.BoolVar(&pcieWidth, "p", false, "show PCIe width and speed")

	flag.Parse()

	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		fmt.Println("Init nvml ", nvml.ErrorString(ret))
		return
	}
	defer nvml.Shutdown() // 确保在程序结束时关闭NVML

	n, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		fmt.Println("DeviceGetCount ", nvml.ErrorString(ret))
		return
	}
	fmt.Println("Number of devices:", n)
	var errorPcieInfos []pcieInfo

	for i := 0; i < n; i++ {
		if gpuIndex != -1 && i != gpuIndex {
			continue // 跳过非指定的GPU
		}

		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			fmt.Println("DeviceGetHandleByIndex ", nvml.ErrorString(ret))
			continue // 继续处理下一个设备
		}
		if pcieWidth {
			pinfo, err := ScanGPUPcieInfo(device)
			if err != nil {
				fmt.Println("ScanGPUPcieInfo Eorro", err.Error())
				continue // 继续处理下一个设备
			}

			if pinfo != nil {
				errorPcieInfos = append(errorPcieInfos, *pinfo)
			}

		} else if reset {
			ResetGPU(device)
			// 重置风扇转速（假设重置为默认值）
			ResetGPUFanSpeed(device, -1)
		} else if miniFan {
			SetGPUFanSpeed(device, 0, -1)
		} else {
			SetGPUFanSpeed(device, fanSpeed, -1)
		}
	}
	if pcieWidth && len(errorPcieInfos) > 0 {
		fmt.Println()
		fmt.Println("-------- Error PcieInfos: -------------")
		for _, info := range errorPcieInfos {
			fmt.Printf("GPU %d: UUID=%s, LinkWidth=%d(MAX:%d), LinkGeneration=%d(MAX:%d,Pcie: %d)\n",
				info.Index, info.UUID, info.LinkWidth, info.MaxLinkWidth, info.LinkGeneration, info.MaxLinkGeneration, info.MaxPcieGeneration)
			device, _ := nvml.DeviceGetHandleByIndex(info.Index)
			SetGPUFanSpeed(device, 100, -1)
		}
	}
}

func CheckRoot() {
	if syscall.Getuid() != 0 {
		fmt.Println("Please run as root.")
		os.Exit(1)
	}
}

func ResetGPU(d nvml.Device) error {
	ret := d.SetPersistenceMode(nvml.FEATURE_ENABLED)
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to enable PersistenceMode: %v", nvml.ErrorString(ret))
	}

	ret = d.ResetGpuLockedClocks()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to reset GPU locked clocks: %v", nvml.ErrorString(ret))
	}

	ret = d.ResetMemoryLockedClocks()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to reset memory locked clocks: %v", nvml.ErrorString(ret))
	}
	return nil
}

// forEachFan iterates over the specified fan or all fans if fanIndex is -1.
func forEachFan(device nvml.Device, fanIndex int, action func(i int) (nvml.Return, error)) error {
	n, ret := device.GetNumFans()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to get number of fans: %v", nvml.ErrorString(ret))
	}
	if fanIndex > n {
		return fmt.Errorf("invalid fan index: %d, only get %d fans", fanIndex, n)
	}

	start, end := 0, n
	if fanIndex >= 0 {
		start, end = fanIndex, fanIndex+1
	}

	var errMsg string
	for i := start; i < end; i++ {
		ret, err := action(i)
		if ret != nvml.SUCCESS {
			errMsg = fmt.Sprintf("operation failed at fan %d: %v", i, nvml.ErrorString(ret))
			fmt.Println(errMsg)
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// SetGPUFanSpeed sets the fan speed of a GPU.
// fanSpeed is the fan speed in percentage, range from 0 to 100
// fanIndex start from 0, and -1 means all fans
func SetGPUFanSpeed(device nvml.Device, fanSpeed int, fanIndex int) error {
	if fanSpeed < 0 || fanSpeed > 100 {
		return fmt.Errorf("invalid fan speed: %d, must be in range 0 to 100", fanSpeed)
	}

	action := func(i int) (nvml.Return, error) {
		ret := device.SetFanSpeed_v2(i, fanSpeed)
		return ret, fmt.Errorf("unable to set %d%% fan speed at %d fan", fanSpeed, i)
	}

	err := forEachFan(device, fanIndex, action)
	if err != nil {
		fmt.Printf("Set fan speed at all fans for GPU %d failed: %v\n", fanIndex, err)
		return err
	}

	gpuIndex, _ := device.GetIndex()
	if fanIndex < 0 {
		fmt.Printf("Set %d%% fan speed at all fans for GPU %d\n", fanSpeed, gpuIndex)
	} else {
		fmt.Printf("Set %d%% fan speed at %d fan for GPU %d\n", fanSpeed, fanIndex, gpuIndex)
	}
	return nil
}

// ResetGPUFanSpeed resets the fan speed of a GPU to default.
func ResetGPUFanSpeed(device nvml.Device, fanIndex int) error {
	action := func(i int) (nvml.Return, error) {
		ret := device.SetDefaultFanSpeed_v2(i)
		return ret, fmt.Errorf("unable to reset fan speed at %d fan", i)
	}

	err := forEachFan(device, fanIndex, action)
	if err != nil {
		return err
	}

	gpuIndex, _ := device.GetIndex()
	if fanIndex < 0 {
		fmt.Printf("Reset fan speed at all fans for GPU %d\n", gpuIndex)
	} else {
		fmt.Printf("Reset fan speed at %d fan for GPU %d\n", fanIndex, gpuIndex)
	}
	return nil
}

func ScanGPUPcieInfo(device nvml.Device) (*pcieInfo, error) {
	uuid, ret := device.GetUUID()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get UUID %v", nvml.ErrorString(ret))
	}
	linkWidth, ret := device.GetCurrPcieLinkWidth()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get LinkWidth %v", nvml.ErrorString(ret))
	}
	linkSpeed, ret := device.GetPcieSpeed()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get LinkSpeed %v", nvml.ErrorString(ret))
	}
	maxLinkWidth, ret := device.GetMaxPcieLinkWidth()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get MaxLinkWidth %v", nvml.ErrorString(ret))
	}

	currentLinkGeneration, ret := device.GetCurrPcieLinkGeneration()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get CurrentLinkGeneration %v", nvml.ErrorString(ret))
	}

	maxLinkGeneration, ret := device.GetGpuMaxPcieLinkGeneration()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get MaxLinkGeneration %v", nvml.ErrorString(ret))
	}
	maxPcieGeration, ret := device.GetMaxPcieLinkGeneration()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get MaxPcieGeneration %v", nvml.ErrorString(ret))
	}

	index, _ := device.GetIndex()

	fmt.Printf("GPU %d: UUID=%s, LinkWidth=%d(MAX:%d), LinkGeneration=%d(MAX:%d,Pcie: %d)\n",
		index, uuid, linkWidth, maxLinkWidth, currentLinkGeneration, maxLinkGeneration, maxPcieGeration)

	if linkWidth < maxLinkWidth {
		return &pcieInfo{Index: index,
			UUID:              uuid,
			LinkWidth:         linkWidth,
			MaxLinkWidth:      maxLinkWidth,
			LinkSpeed:         linkSpeed,
			MaxLinkGeneration: maxLinkGeneration,
			LinkGeneration:    currentLinkGeneration,
			MaxPcieGeneration: maxPcieGeration}, nil
	}
	return nil, nil
}
