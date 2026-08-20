package collectors

import (
	"context"
	"runtime"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// MemoryCollector collects memory metrics
type MemoryCollector struct {
	BaseCollector
	snapshots SnapshotProvider
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		BaseCollector: NewBaseCollector("memory", 10*time.Second),
		snapshots:     defaultSnapshotProvider(),
	}
}

// SetSnapshotProvider injects the shared host-inventory provider.
func (c *MemoryCollector) SetSnapshotProvider(p SnapshotProvider) {
	if p != nil {
		c.snapshots = p
	}
}

// Collect gathers memory metrics
func (c *MemoryCollector) Collect(ctx context.Context) (*MetricData, error) {
	if collectorOS != runtime.GOOS {
		return unsupportedMetricData(c.GetName(), "memory"), nil
	}
	snapshot, _ := c.snapshots.Snapshot(ctx)
	reading := collectPlatformMemory(ctx, c, snapshot)
	topProcesses, _ := GetTopProcessesByMemory(5)
	values := map[string]interface{}{"top_processes": topProcesses}
	if reading.status == "measured" {
		values["usage_percent"] = reading.usage
		for key, value := range reading.details {
			values[key] = value
		}
		values["swap"] = reading.swap
	}
	if reading.status != "" {
		values["status"] = reading.status
	}
	if reading.reason != "" {
		values["reason"] = reading.reason
	}

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "memory",
		Values:        values,
		Tags: map[string]string{
			"os":     collectorOS,
			"source": reading.provenance,
		},
	}, nil
}

// getMemoryUsage returns memory usage percentage
func (c *MemoryCollector) getMemoryUsage(snapshot hostinventory.Snapshot) float64 {
	total := snapshot.Memory.TotalBytes
	available := snapshot.Memory.AvailableBytes
	if total <= 0 {
		return 0.0
	}
	usage := (float64(total-available) / float64(total)) * 100.0
	if usage < 0 {
		return 0.0
	}
	return usage
}

// getMemoryDetails returns detailed memory information
func (c *MemoryCollector) getMemoryDetails(snapshot hostinventory.Snapshot) map[string]int64 {
	details := map[string]int64{
		"total":     0,
		"used":      0,
		"available": 0,
		"cached":    0,
		"buffers":   0,
	}

	details["total"] = bytesToInt64(snapshot.Memory.TotalBytes)
	details["available"] = bytesToInt64(snapshot.Memory.AvailableBytes)
	details["buffers"] = bytesToInt64(snapshot.Memory.BuffersBytes)
	details["cached"] = bytesToInt64(snapshot.Memory.CachedBytes)
	if details["total"] > 0 && details["available"] > 0 {
		details["used"] = details["total"] - details["available"]
	}

	return details
}

// getSwapUsage returns swap usage information
func (c *MemoryCollector) getSwapUsage(snapshot hostinventory.Snapshot) map[string]interface{} {
	swapInfo := map[string]interface{}{
		"used":    int64(0),
		"total":   int64(0),
		"percent": float64(0),
	}

	total := bytesToInt64(snapshot.Swap.TotalBytes)
	free := bytesToInt64(snapshot.Swap.FreeBytes)
	used := total - free
	if used < 0 {
		used = 0
	}

	swapInfo["total"] = total
	swapInfo["used"] = used
	if total > 0 {
		swapInfo["percent"] = float64(used) / float64(total) * 100
	}

	return swapInfo
}

// GetTopProcessesByMemory returns top processes by memory usage
func GetTopProcessesByMemory(limit int) ([]map[string]interface{}, error) {
	samples, err := topProcessSamples(limit, func(a, b procsampler.ProcessSample) bool {
		if a.RSSKB != b.RSSKB {
			return a.RSSKB > b.RSSKB
		}
		return a.CPUPct > b.CPUPct
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
			"memory_mb":   float64(sample.RSSKB) / 1024,
		})
	}

	return processes, nil
}

// GetMemoryGrowthPatterns analyzes memory growth patterns
func GetMemoryGrowthPatterns() []map[string]interface{} {
	// This would require historical data tracking
	// For now, return mock data
	return []map[string]interface{}{
		{
			"process":            "scenario-api-1",
			"growth_mb_per_hour": 15.0,
			"risk_level":         "medium",
		},
		{
			"process":            "postgres",
			"growth_mb_per_hour": 2.0,
			"risk_level":         "low",
		},
	}
}

func bytesToInt64(value uint64) int64 {
	if value > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1)
	}
	return int64(value)
}
