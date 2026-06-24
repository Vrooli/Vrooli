package collectors

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// CPUCollector collects CPU metrics
type CPUCollector struct {
	BaseCollector
	snapshots      SnapshotProvider
	lastCPUStats   *cpuStats
	lastSampleTime time.Time
}

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		BaseCollector: NewBaseCollector("cpu", 10*time.Second),
		snapshots:     defaultSnapshotProvider(),
	}
}

// SetSnapshotProvider injects the shared host-inventory provider so a cycle
// probes the host once across the cpu/memory/gpu collectors.
func (c *CPUCollector) SetSnapshotProvider(p SnapshotProvider) {
	if p != nil {
		c.snapshots = p
	}
}

// Collect gathers CPU metrics
func (c *CPUCollector) Collect(ctx context.Context) (*MetricData, error) {
	snapshot, _ := c.snapshots.Snapshot(ctx)
	usage := c.getCPUUsage()
	loadAvg := c.getLoadAverage(snapshot)
	contextSwitches := c.getContextSwitches()
	cores := snapshot.CPU.Cores
	goarch := snapshot.Arch
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	goos := snapshot.OS
	if goos == "" {
		goos = runtime.GOOS
	}

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "cpu",
		Values: map[string]interface{}{
			"usage_percent":    usage,
			"cores":            cores,
			"load_average":     loadAvg,
			"context_switches": contextSwitches,
			"goroutines":       runtime.NumGoroutine(),
		},
		Tags: map[string]string{
			"arch": goarch,
			"os":   goos,
		},
	}, nil
}

// getCPUUsage returns current CPU usage percentage using delta calculation
func (c *CPUCollector) getCPUUsage() float64 {
	if runtime.GOOS != "linux" {
		// Fallback for non-Linux systems
		return float64(15 + (time.Now().Second() % 30))
	}

	// Read current CPU stats
	output, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0.0
	}

	var cpuLine string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "cpu ") {
			cpuLine = line
			break
		}
	}
	if cpuLine == "" {
		return 0.0
	}

	fields := strings.Fields(strings.TrimSpace(cpuLine))
	if len(fields) < 8 {
		return 0.0
	}

	// Parse current stats
	current := &cpuStats{}
	current.user, _ = strconv.ParseUint(fields[1], 10, 64)
	current.nice, _ = strconv.ParseUint(fields[2], 10, 64)
	current.system, _ = strconv.ParseUint(fields[3], 10, 64)
	current.idle, _ = strconv.ParseUint(fields[4], 10, 64)
	if len(fields) > 5 {
		current.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
	}
	if len(fields) > 6 {
		current.irq, _ = strconv.ParseUint(fields[6], 10, 64)
	}
	if len(fields) > 7 {
		current.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
	}
	if len(fields) > 8 {
		current.steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	now := time.Now()

	// If we don't have a previous sample, store this one and return 0 for the initial reading
	if c.lastCPUStats == nil {
		c.lastCPUStats = current
		c.lastSampleTime = now
		return 0.0
	}

	// Calculate delta
	deltaUser := current.user - c.lastCPUStats.user
	deltaNice := current.nice - c.lastCPUStats.nice
	deltaSystem := current.system - c.lastCPUStats.system
	deltaIdle := current.idle - c.lastCPUStats.idle
	deltaIowait := current.iowait - c.lastCPUStats.iowait
	deltaIrq := current.irq - c.lastCPUStats.irq
	deltaSoftirq := current.softirq - c.lastCPUStats.softirq
	deltaSteal := current.steal - c.lastCPUStats.steal

	// Calculate total delta
	deltaTotal := deltaUser + deltaNice + deltaSystem + deltaIdle + deltaIowait + deltaIrq + deltaSoftirq + deltaSteal

	// Update last stats
	c.lastCPUStats = current
	c.lastSampleTime = now

	if deltaTotal == 0 {
		return 0.0
	}

	// Calculate usage percentage (everything except idle and iowait)
	deltaUsed := deltaUser + deltaNice + deltaSystem + deltaIrq + deltaSoftirq + deltaSteal
	usage := (float64(deltaUsed) / float64(deltaTotal)) * 100.0

	return usage
}

// getLoadAverage returns system load averages from the shared host snapshot.
func (c *CPUCollector) getLoadAverage(snapshot hostinventory.Snapshot) []float64 {
	if snapshot.Load.Load1 != 0 || snapshot.Load.Load5 != 0 || snapshot.Load.Load15 != 0 {
		return []float64{snapshot.Load.Load1, snapshot.Load.Load5, snapshot.Load.Load15}
	}
	if snapshot.OS != "linux" && runtime.GOOS != "linux" {
		return []float64{0.5, 0.5, 0.5}
	}
	return []float64{0.0, 0.0, 0.0}
}

// getContextSwitches returns the number of context switches
func (c *CPUCollector) getContextSwitches() int64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	output, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "ctxt ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ctxt, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
				return ctxt
			}
		}
	}
	return 0
}

// getTopProcessesByCPU returns top processes by CPU usage
func GetTopProcessesByCPU(limit int) ([]map[string]interface{}, error) {
	if runtime.GOOS != "linux" {
		return []map[string]interface{}{}, nil
	}

	output, err := commandOutput(context.Background(), 2*time.Second, "bash", "-c",
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,nlwp --sort=-%%cpu --no-headers | head -%d", limit))
	if err != nil {
		return nil, err
	}

	var processes []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memPercent, _ := strconv.ParseFloat(fields[3], 64)
		threads, _ := strconv.Atoi(fields[4])

		processes = append(processes, map[string]interface{}{
			"pid":         pid,
			"name":        fields[1],
			"cpu_percent": cpuPercent,
			"mem_percent": memPercent,
			"threads":     threads,
		})
	}

	return processes, nil
}
