package collectors

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// MemoryCollector collects memory metrics
type MemoryCollector struct {
	BaseCollector
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		BaseCollector: NewBaseCollector("memory", 10*time.Second),
	}
}

// Collect gathers memory metrics
func (c *MemoryCollector) Collect(ctx context.Context) (*MetricData, error) {
	snapshot, _ := hostinventory.Collect(ctx)
	memUsage := c.getMemoryUsage(snapshot)
	memDetails := c.getMemoryDetails(snapshot)
	swapInfo := c.getSwapUsage(snapshot)
	topProcesses, _ := GetTopProcessesByMemory(5)

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "memory",
		Values: map[string]interface{}{
			"usage_percent": memUsage,
			"total":         memDetails["total"],
			"used":          memDetails["used"],
			"available":     memDetails["available"],
			"cached":        memDetails["cached"],
			"buffers":       memDetails["buffers"],
			"swap":          swapInfo,
			"top_processes": topProcesses,
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
	if runtime.GOOS != "linux" {
		return []map[string]interface{}{}, nil
	}

	output, err := commandOutput(context.Background(), 2*time.Second, "bash", "-c",
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,rss --sort=-%%mem --no-headers | head -%d", limit))
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
		rssKB, _ := strconv.ParseFloat(fields[4], 64)

		processes = append(processes, map[string]interface{}{
			"pid":         pid,
			"name":        fields[1],
			"cpu_percent": cpuPercent,
			"mem_percent": memPercent,
			"memory_mb":   rssKB / 1024,
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
