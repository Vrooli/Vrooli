package collectors

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PressureCollector reads the kernel's low-cost pressure evidence. It does
// not scan processes or fork commands; unsupported hosts surface an explicit
// degraded value so recovery consumers never confuse unavailable with clear.
type PressureCollector struct{ BaseCollector }

func NewPressureCollector() *PressureCollector {
	return &PressureCollector{BaseCollector: NewBaseCollector("pressure", 10*time.Second)}
}

func (c *PressureCollector) Collect(context.Context) (*MetricData, error) {
	values := map[string]interface{}{"available": false}
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/proc/pressure/memory"); err == nil {
			values["available"] = true
			memory := parsePSI(string(raw))
			values["memory"] = memory
			// Flatten the primary PSI values as numeric time-series keys while
			// retaining the complete structured payload for current inspection.
			for class, metrics := range memory {
				for window, value := range metrics {
					values["memory_psi_"+class+"_"+window] = value
				}
			}
		} else {
			values["degraded_reason"] = "memory PSI unavailable"
		}
		if raw, err := os.ReadFile("/proc/vmstat"); err == nil {
			values["oom_kill_count"] = vmStatValue(string(raw), "oom_kill")
			values["oom_count"] = vmStatValue(string(raw), "oom")
		}
	} else {
		values["degraded_reason"] = "PSI is only available on Linux"
	}
	return &MetricData{CollectorName: c.GetName(), Timestamp: time.Now(), Type: "pressure", Values: values}, nil
}

func parsePSI(raw string) map[string]map[string]float64 {
	out := map[string]map[string]float64{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		metrics := map[string]float64{}
		for _, field := range fields[1:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if value, err := strconv.ParseFloat(parts[1], 64); err == nil {
				metrics[parts[0]] = value
			}
		}
		out[fields[0]] = metrics
	}
	return out
}

func vmStatValue(raw, key string) int64 {
	prefix := key + " "
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, prefix)), 10, 64)
		if err == nil {
			return value
		}
	}
	return 0
}
