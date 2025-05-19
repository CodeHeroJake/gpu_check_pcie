# gpu_check_pcie
安装完服务器后，需要检查pcie通道数量是否正确，若不为pcie最大值，则将该显卡风扇设置为最大，便于物理上查找这张显卡


## 使用方式
### 拉取代码
```sh
git clone https://github.com/CodeHeroJake/gpu_check_pcie.git
```
### 安装依赖
```sh
cd gpu_check_pcie && go mod tidy
```
### 编译代码
```sh
go build -o gpu_check_pcie
```
### 运行程序（前提条件，已经安装nvidia-smi）
```sh
sudo ./gpu_check_pcie -p
```

### 程序help
```sh
./gpu_check_pcie -h
```
```txt
  -i int
        Specify GPU index, if not means all GPUs (default -1)
  -p    show PCIe width and speed
  -r    Reset fan speed to default
```

### 示例
1. 初始化所有风扇转速
```sh
sudo ./gpu_check_pcie -r
```
执行结果示意：
```
Number of devices: 2
Reset fan speed at all fans for GPU 0
Reset fan speed at all fans for GPU 1
```
2. 显示所有GPU的PCIe通道信息
所有未满足最大速率的GPU风扇转速设置为最大
```sh
sudo ./gpu_check_pcie -p
```
执行结果示意：
```txt
Number of devices: 2
GPU 0: UUID=GPU-b2f19a54-e865-3a37-6893-c17f0b8c0f74, LinkWidth=16(MAX:16), LinkGeneration=1(MAX:4,Pcie: 4)
GPU 1: UUID=GPU-6a5102da-2b6e-3f42-306c-7330bacca8fd, LinkWidth=16(MAX:16), LinkGeneration=1(MAX:4,Pcie: 4)
```

3. 手动将某个GPU的风扇转速设置为最大
```sh
sudo ./gpu_check_pcie -i 0
```
执行结果示意：
```txt
Number of devices: 2
Set 100% fan speed at all fans for GPU 1
```

4. 手动将某个GPU的风扇转速设置为默认值
```sh
sudo ./gpu_check_pcie -r -i 0
```
执行结果示意：
```txt
Number of devices: 2
Reset fan speed at all fans for GPU 0
```