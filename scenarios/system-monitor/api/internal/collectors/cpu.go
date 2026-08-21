package collectors

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// CPUCollector collects CPU metrics
type CPUCollector struct {
	BaseCollector
	mu             sync.Mutex
	snapshots      SnapshotProvider
	lastCPUStats   *cpuStats
	lastSampleTime time.Time
	platformState  platformCPUState
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if collectorOS != runtime.GOOS {
		return unsupportedMetricData(c.GetName(), "cpu"), nil
	}
	snapshot, _ := c.snapshots.Snapshot(ctx)
	reading := collectPlatformCPU(ctx, c, snapshot)
	cores := snapshot.CPU.Cores
	goarch := snapshot.Arch
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	goos := snapshot.OS
	if goos == "" {
		goos = collectorOS
	}

	data := &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "cpu",
		Values: map[string]interface{}{
			"cores":      cores,
			"goroutines": runtime.NumGoroutine(),
		},
		Tags: map[string]string{
			"arch":   goarch,
			"os":     goos,
			"source": reading.provenance,
		},
	}
	if reading.status != "" {
		data.Values["status"] = reading.status
	}
	if reading.reason != "" {
		data.Values["reason"] = reading.reason
	}
	if reading.status == "measured" {
		data.Values["usage_percent"] = reading.usage
		if reading.loadAverage != nil {
			data.Values["load_average"] = reading.loadAverage
		}
		if reading.contextSwitches != nil {
			data.Values["context_switches"] = *reading.contextSwitches
		}
	}
	return data, nil
}

// getCPUUsage returns current CPU usage percentage using delta calculation
func (c *CPUCollector) getCPUUsage() float64 {
	if collectorOS != "linux" {
		return 0
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
	if current.user < c.lastCPUStats.user || current.nice < c.lastCPUStats.nice || current.system < c.lastCPUStats.system || current.idle < c.lastCPUStats.idle || current.iowait < c.lastCPUStats.iowait || current.irq < c.lastCPUStats.irq || current.softirq < c.lastCPUStats.softirq || current.steal < c.lastCPUStats.steal {
		c.lastCPUStats = current
		c.lastSampleTime = now
		return 0.0
	}
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
	return []float64{0.0, 0.0, 0.0}
}

// getContextSwitches returns the number of context switches
func (c *CPUCollector) getContextSwitches() int64 {
	if collectorOS != "linux" {
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
	samples, err := topProcessSamples(limit, func(a, b procsampler.ProcessSample) bool {
		if a.CPUPct != b.CPUPct {
			return a.CPUPct > b.CPUPct
		}
		return a.RSSKB > b.RSSKB
	})
	if err != nil {
		return nil, err
	}

	totalKB := totalMemoryKB()
	processes := make([]map[string]interface{}, 0, len(samples))
	for _, sample := range samples {
		processes = append(processes, map[string]interface{}{
			"pid":         sample.PID,
			"name":        sample.Comm,
			"cpu_percent": sample.CPUPct,
			"mem_percent": memoryPercent(sample.RSSKB, totalKB),
			"threads":     sample.Threads,
		})
	}

	return processes, nil
}
