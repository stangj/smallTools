package main

import (
	"encoding/json"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var Message string = `
# @Time    : 2024/04/18 23:14
# @Author  : TangHao
`

type ProcessInfo struct {
	Pid    int32
	Memory uint64
}
type ByMemory []ProcessInfo

func (a ByMemory) Len() int           { return len(a) }
func (a ByMemory) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMemory) Less(i, j int) bool { return a[i].Memory > a[j].Memory }

func checkCpu() string {
	res, _ := cpu.Percent(time.Second*time.Duration(5), false)
	resout := fmt.Sprintf("%.2f", res[0])
	roundedAverage, _ := strconv.ParseFloat(resout, 64)
	return "CPU使用率:  " + strconv.FormatFloat(roundedAverage, 'f', 2, 64) + "%"
}
func checkCpuLoad() string {
	if runtime.GOOS == "linux" {
		pcnt, err1 := cpu.Counts(false) // true时会获取逻辑核心数量,false时则会获取物理核心数量
		if err1 != nil {
			return "Check Cpu Error"
		}
		stat, err2 := load.Avg()
		if err2 != nil {
			return "Check Cpu Error"
		}
		load_cpu := float64(stat.Load5) / float64(pcnt)
		resout := fmt.Sprintf("%.2f", load_cpu)
		roundedAverage, _ := strconv.ParseFloat(resout, 64)
		return "CPU负载: " + strconv.FormatFloat(roundedAverage, 'f', 2, 64)
	}

	return ""
}

func checkMem() string {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return "无法获取内存信息"
	}
	// 计算内存使用率
	usedPercent := memInfo.UsedPercent
	return "内存使用率:  " + strconv.FormatFloat(usedPercent, 'f', 2, 64) + "%"

}

func checkPressMem() []ProcessInfo {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}

	var processInfoList []ProcessInfo

	// 遍历所有进程，获取内存占用
	for _, p := range processes {
		memInfo, err := p.MemoryInfo()
		if err != nil {
			// 忽略无法获取内存信息的进程
			continue
		}
		processInfoList = append(processInfoList, ProcessInfo{
			Pid:    p.Pid,
			Memory: memInfo.RSS,
		})
	}

	// 按内存占用排序
	sort.Sort(ByMemory(processInfoList))
	return processInfoList

}

// 获取磁盘Inode信息
func getInode() string {
	totaldiskslist := make([]map[string]string, 0)
	parts, err := disk.Partitions(true)
	if err != nil {
		return "Check Disk Error  Not Found disk "
	}
	if runtime.GOOS == "linux" {
		for _, part := range parts {
			totaldisks := make(map[string]string, 0)
			if strings.Contains(part.Device, "/dev/") && !strings.Contains(part.Device, "loop") && !strings.Contains(part.Device, "/boot/efi") && !strings.Contains(part.Device, "/dev/sda1 -") {
				partInfo, _ := disk.Usage(part.Mountpoint)
				if strings.Contains(part.Device, "dm-") {
					totaldisks[part.Mountpoint] = fmt.Sprintf("%v%%", math.Ceil(partInfo.InodesUsedPercent))
				} else {
					totaldisks[part.Device] = fmt.Sprintf("%v%%", math.Ceil(partInfo.InodesUsedPercent))
				}
				totaldiskslist = append(totaldiskslist, totaldisks)
			}
		}
	}
	jsonString, _ := json.Marshal(totaldiskslist)
	return string(jsonString)
}

// 获取磁盘信息
func getDiskInfo() string {
	totaldisks := make(map[string]string, 0)
	parts, err := disk.Partitions(true)
	if err != nil {
		return "Check Disk Error  Not Found disk "
	}
	if runtime.GOOS == "linux" {
		for _, part := range parts {
			if strings.Contains(part.Device, "/dev/") && !strings.Contains(part.Device, "loop") {
				partInfo, _ := disk.Usage(part.Mountpoint)
				if strings.Contains(part.Device, "dm-") {
					totaldisks[part.Mountpoint] = fmt.Sprintf("%v%%", math.Ceil(partInfo.UsedPercent))
				} else {
					totaldisks[part.Device] = fmt.Sprintf("%v%%", math.Ceil(partInfo.UsedPercent))
				}
			}
		}
	} else {
		for _, part := range parts {
			partInfo, _ := disk.Usage(part.Mountpoint)
			totaldisks[strings.TrimRight(part.Device, ":")] = fmt.Sprintf("%v%%", math.Ceil(partInfo.UsedPercent))
		}
	}
	jsonString, _ := json.Marshal(totaldisks)
	return string(jsonString)
}

func checkNet() []net.IOCountersStat {
	netStat, err := net.IOCounters(true)
	if err != nil {
		fmt.Println("无法获取网络IO信息：", err)
		return nil
	}
	return netStat
}
func main() {

	Cpu := checkCpu()
	CpuLa := checkCpuLoad()
	Mem := checkMem()
	CInode := getInode()
	CDisk := getDiskInfo()
	Proc := checkPressMem()
	Net := checkNet()

	fmt.Printf("%s\r\n", Message)
	fmt.Println("++++++++++++++++++++++++++++++CPU/CPU负载/内存信息++++++++++++++++++++++++++++++\n")
	fmt.Printf("%v\r\n%v\n%v\n", Cpu, CpuLa, Mem)
	fmt.Println()
	fmt.Println("++++++++++++++++++++++++++++++磁盘空间使用率/磁盘Inode使用率++++++++++++++++++++++++++++++\n")
	fmt.Printf("磁盘空间使用率:  %v\n磁盘Inode使用率:  %v\n", CDisk, CInode)
	fmt.Println()
	fmt.Println("++++++++++++++++++++++++++++++占用内存前五的进程信息如下:++++++++++++++++++++++++++++++")
	for i, v := range Proc[0:5] {
		fmt.Printf("%d. PID: %d, Memory Usage: %d MB\n", i+1, v.Pid, v.Memory/1024/1024)
	}
	fmt.Println()
	fmt.Println("++++++++++++++++++++++++++++++服务器的网络信息如下:++++++++++++++++++++++++++++++")
	for _, io := range Net {
		fmt.Printf("网络接口：%s\n", io.Name)
		fmt.Printf("接收兆字节数：%dMB\n", io.BytesRecv/1024/1024)
		fmt.Printf("发送兆字节数：%dMB\n", io.BytesSent/1024/1024)
		fmt.Printf("接收错误数：%d\n", io.Errin)
		fmt.Printf("发送错误数：%d\n", io.Errout)
		fmt.Println("-----------------------------")
	}
}

