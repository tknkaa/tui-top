package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProcInfo struct {
	PID  int
	Name string
	CPU  float64
}

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
	fields := strings.Fields(lines[0])
	var total uint64
	for _, v := range fields[1:] {
		n, _ := strconv.ParseUint(v, 10, 64)
		total += n
	}
	return total, nil
}

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.table.View() + "\nPress q to quit.\n"
}

func getProcessList() []ProcInfo {
	entries, _ := os.ReadDir("/proc")
	pids := []int{}
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			pids = append(pids, pid)
		}
	}

	procTimes1 := make(map[int]uint64)
	total1, _ := readTotalCPU()
	for _, pid := range pids {
		utime, stime, err := readProcStat(pid)
		if err == nil {
			procTimes1[pid] = utime + stime
		}
	}

	time.Sleep(1 * time.Second)

	procTimes2 := make(map[int]uint64)
	total2, _ := readTotalCPU()
	for _, pid := range pids {
		utime, stime, err := readProcStat(pid)
		if err == nil {
			procTimes2[pid] = utime + stime
		}
	}

	deltaTotal := total2 - total1
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

	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})

	return procs
}

func main() {
	procs := getProcessList()

	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 20},
		{Title: "CPU(%)", Width: 10},
	}

	rows := []table.Row{}
	for _, p := range procs {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", p.PID),
			p.Name,
			fmt.Sprintf("%.2f", p.CPU),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{table: t}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
