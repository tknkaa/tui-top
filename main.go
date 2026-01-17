package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func readProcStat(pid int) (uint64, uint64, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0, 0, fmt.Errorf("not enough fields in stat")
	}
	// The amount of time the process has spent in user mode
	// utime is the cumulative number of clock ticks that the process has spent executing in user mode since it started.
	utime, _ := strconv.ParseUint(fields[13], 10, 64)
	// The amount of time the process has spent in kernel mode
	stime, _ := strconv.ParseUint(fields[14], 10, 64)
	return utime, stime, nil
}

func readTotalCPU() (uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(data), "\n")
	fmt.Println("first line of /proc/stat", lines[0])
	fields := strings.Fields(lines[0])
	var total uint64
	for _, v := range fields[1:] {
		n, _ := strconv.ParseUint(v, 10, 64)
		total += n
	}
	return total, nil
}

func main() {
	entries, _ := os.ReadDir("/proc")
	pids := []int{}
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			pids = append(pids, pid)
		}
	}

	// First sample
	procTimes1 := make(map[int]uint64)
	total1, _ := readTotalCPU()
	for _, pid := range pids {
		utime, stime, err := readProcStat(pid)
		if err == nil {
			procTimes1[pid] = utime + stime
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Second sample
	procTimes2 := make(map[int]uint64)
	total2, _ := readTotalCPU()
	for _, pid := range pids {
		utime, stime, err := readProcStat(pid)
		if err == nil {
			procTimes2[pid] = utime + stime
		}
	}

	deltaTotal := total2 - total1
	type ProcInfo struct {
		PID  int
		Name string
		CPU  float64
	}
	procs := []ProcInfo{}
	for _, pid := range pids {
		commPath := fmt.Sprintf("/proc/%d/comm", pid)
		comm, err := os.ReadFile(commPath)
		if err != nil {
			continue
		}
		pt1, ok1 := procTimes1[pid]
		pt2, ok2 := procTimes2[pid]
		if ok1 && ok2 && deltaTotal > 0 {
			cpu := float64(pt2-pt1) / float64(deltaTotal) * 100
			procs = append(procs, ProcInfo{PID: pid, Name: strings.TrimSpace(string(comm)), CPU: cpu})
		}
	}

	// Sort by CPU usage descending
	// import "sort" at the top
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})

	for _, p := range procs {
		fmt.Printf("PID: %d, Name: %s, CPU: %.2f%%\n", p.PID, p.Name, p.CPU)
	}
}
