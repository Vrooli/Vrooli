package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// GPUCollector adapts shared host inventory GPU facts into system-monitor metrics.
type GPUCollector struct {
	BaseCollector
}

// NewGPUCollector constructs a GPU collector. The collector is disabled when the
// shared host inventory reports no NVIDIA probe tool.
func NewGPUCollector() *GPUCollector {
	collector := &GPUCollector{
		BaseCollector: NewBaseCollector("gpu", 15*time.Second),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	snapshot, err := hostinventory.Collect(ctx)
	if err != nil || !snapshot.RuntimeTools["nvidia-smi"].Present {
		collector.SetEnabled(false)
	}

	return collector
}

// Collect retrieves GPU metrics. When no GPUs are present, the collector emits an
// empty metric payload with a descriptive warning so downstream consumers can
// surface the absence without relying on mock data.
func (c *GPUCollector) Collect(ctx context.Context) (*MetricData, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gpu collector disabled")
	}

	snapshot, err := hostinventory.Collect(ctx)
	if err != nil {
		return nil, err
	}
	devices, summary, driverVersion, primaryModel := adaptGPUInventory(snapshot)
	warnings := append([]string{}, snapshot.Warnings...)
	if snapshot.ProbeStatuses["nvidia_gpu"] == "not_present" {
		warnings = append(warnings, "nvidia-smi binary not found")
	}

	values := map[string]interface{}{
		"devices":               devices,
		"summary":               summary,
		"driver_version":        driverVersion,
		"primary_model":         primaryModel,
		"total_usage_percent":   summary.AverageUtilizationPercent,
		"device_count":          summary.DeviceCount,
		"average_usage_percent": summary.AverageUtilizationPercent,
		"total_memory_mb":       summary.TotalMemoryMB,
		"used_memory_mb":        summary.UsedMemoryMB,
		"average_temperature_c": summary.AverageTemperatureC,
	}
	if len(warnings) > 0 {
		values["warnings"] = warnings
	}

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "gpu",
		Values:        values,
	}, nil
}

func adaptGPUInventory(snapshot hostinventory.Snapshot) ([]models.GPUDeviceMetrics, models.GPUSummary, string, string) {
	var devices []models.GPUDeviceMetrics
	var summary models.GPUSummary
	var driverVersion string
	var primaryModel string
	var tempSamples int

	processesByUUID := map[string][]models.GPUProcessInfo{}
	for _, process := range snapshot.GPUProcesses {
		processesByUUID[process.GPUUUID] = append(processesByUUID[process.GPUUUID], models.GPUProcessInfo{
			PID:          process.PID,
			ProcessName:  process.ProcessName,
			MemoryUsedMB: float64(process.UsedBytes) / 1024 / 1024,
		})
	}

	for _, gpu := range snapshot.GPUs {
		device := models.GPUDeviceMetrics{
			Index:             gpu.Index,
			UUID:              gpu.UUID,
			Name:              gpu.Name,
			Utilization:       gpu.UtilizationPercent,
			MemoryUtilization: gpu.MemoryUtilizationPercent,
			MemoryUsedMB:      float64(gpu.VRAMUsedBytes) / 1024 / 1024,
			MemoryTotalMB:     float64(gpu.VRAMBytes) / 1024 / 1024,
			TemperatureC:      gpu.TemperatureC,
			FanSpeedPercent:   gpu.FanSpeedPercent,
			PowerDrawW:        gpu.PowerDrawW,
			PowerLimitW:       gpu.PowerLimitW,
			SMClockMHz:        gpu.SMClockMHz,
			MemoryClockMHz:    gpu.MemoryClockMHz,
			Processes:         processesByUUID[gpu.UUID],
		}
		if driverVersion == "" {
			driverVersion = gpu.DriverVersion
		}
		if primaryModel == "" {
			primaryModel = gpu.Name
		}

		devices = append(devices, device)
		summary.DeviceCount++
		summary.TotalUtilizationPercent += device.Utilization
		summary.TotalMemoryMB += device.MemoryTotalMB
		summary.UsedMemoryMB += device.MemoryUsedMB
		if device.TemperatureC != nil {
			summary.AverageTemperatureC += *device.TemperatureC
			tempSamples++
		}
	}

	if summary.DeviceCount > 0 {
		summary.AverageUtilizationPercent = summary.TotalUtilizationPercent / float64(summary.DeviceCount)
		if tempSamples > 0 {
			summary.AverageTemperatureC = summary.AverageTemperatureC / float64(tempSamples)
		}
	} else {
		summary.AverageTemperatureC = 0
	}

	return devices, summary, driverVersion, primaryModel
}
