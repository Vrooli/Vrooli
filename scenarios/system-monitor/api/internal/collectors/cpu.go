package collectors

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// CPUCollector collects CPU metrics
type CPUCollector struct {
	BaseCollector
	mu            sync.Mutex
	snapshots     SnapshotProvider
	deviceGraphs  DeviceGraphProvider
	rates         *counterRateTracker
	platformState platformCPUState
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		BaseCollector: NewBaseCollector("cpu", 10*time.Second),
		snapshots:     defaultSnapshotProvider(),
		rates:         newCounterRateTracker(),
	}
}

// SetSnapshotProvider injects the shared host-inventory provider so a cycle
// probes the host once across the cpu/memory/gpu collectors.
func (c *CPUCollector) SetSnapshotProvider(p SnapshotProvider) {
	if p != nil {
		c.snapshots = p
	}
}

// SetDeviceGraphProvider shares the cached thermal/device observation with the
// CPU lane; CPU collection must not trigger a second sensor enumeration.
func (c *CPUCollector) SetDeviceGraphProvider(p DeviceGraphProvider) {
	if p != nil {
		c.deviceGraphs = p
	}
}

// Collect gathers CPU metrics
func (c *CPUCollector) Collect(ctx context.Context) (*MetricData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	snapshot, _ := c.snapshots.Snapshot(ctx)
	reading := collectPlatformCPU(ctx, c, snapshot)
	if c.deviceGraphs != nil {
		thermal := c.deviceGraphs.DeviceGraph(ctx).SummarizeThermal()
		if reading.values == nil {
			reading.values = map[string]interface{}{}
		}
		reading.values["thermal_throttle_evidence_status"] = thermal.Status
		reading.values["thermal_throttle_evidence_reason"] = thermal.Reason
		reading.values["thermal_throttle_evidence_provenance"] = thermal.Provenance
		if thermal.Status == "measured" {
			reading.values["thermal_throttle_evidence"] = thermal.TemperatureC
			reading.values["thermal_trip_point_celsius"] = thermal.TripPointC
		}
	}
	cores := snapshot.CPU.Cores
	goarch := snapshot.Arch
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	goos := snapshot.OS
	if goos == "" {
		goos = "unknown"
	}

	data := &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "cpu",
		Values: map[string]interface{}{
			"cores": cores,
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
	}
	for key, value := range reading.values {
		data.Values[key] = value
	}
	ensureCPUObservationStates(data.Values, reading)
	return data, nil
}

func ensureCPUObservationStates(values map[string]interface{}, reading platformCPUReading) {
	for _, signal := range cpuSignalCatalog {
		statusKey := signal.Key + "_status"
		if _, exists := values[statusKey]; exists {
			continue
		}
		if _, exists := values[signal.Key]; exists {
			values[statusKey] = "measured"
			values[signal.Key+"_provenance"] = reading.provenance
			continue
		}
		if signal.Key == "usage_percent" {
			values[statusKey] = reading.status
			values[signal.Key+"_provenance"] = reading.provenance
			if reading.reason != "" {
				values[signal.Key+"_reason"] = reading.reason
			}
			continue
		}
		values[statusKey] = "unsupported"
		values[signal.Key+"_reason"] = "CPU signal backend is not enabled in this collector build"
		values[signal.Key+"_provenance"] = reading.provenance
	}
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
			"pid":                sample.PID,
			"name":               sample.Comm,
			"cpu_percent":        sample.CPUPct,
			"mem_percent":        memoryPercent(sample.RSSKB, totalKB),
			"threads":            sample.Threads,
			"cpu_seconds":        sample.CPUSeconds,
			"cpu_seconds_status": sample.CPUSecondsStatus,
			"cpu_seconds_reason": sample.CPUSecondsReason,
		})
	}

	return processes, nil
}

// GetTopProcessesByCPUSeconds ranks the same sampled process set by cumulative
// CPU time in the last sampling interval. It is intentionally separate from
// instantaneous CPU ranking so a steady low-percentage burner remains visible.
func GetTopProcessesByCPUSeconds(limit int) ([]map[string]interface{}, error) {
	samples, err := topProcessSamples(limit, func(a, b procsampler.ProcessSample) bool {
		if a.CPUSeconds != b.CPUSeconds {
			return a.CPUSeconds > b.CPUSeconds
		}
		return a.CPUPct > b.CPUPct
	})
	if err != nil {
		return nil, err
	}
	processes := make([]map[string]interface{}, 0, len(samples))
	for _, sample := range samples {
		processes = append(processes, map[string]interface{}{
			"pid": sample.PID, "name": sample.Comm, "cpu_percent": sample.CPUPct,
			"cpu_seconds": sample.CPUSeconds, "cpu_seconds_status": sample.CPUSecondsStatus,
			"cpu_seconds_reason": sample.CPUSecondsReason, "threads": sample.Threads,
		})
	}
	return processes, nil
}
