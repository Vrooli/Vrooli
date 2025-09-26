package collectors

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"system-monitor-api/internal/models"
)

// GPUCollector gathers metrics from NVIDIA GPUs using nvidia-smi.
type GPUCollector struct {
	BaseCollector
	nvidiaSMIPath string
}

// NewGPUCollector constructs a GPU collector. The collector is disabled automatically
// when the nvidia-smi binary is not available.
func NewGPUCollector() *GPUCollector {
	collector := &GPUCollector{
		BaseCollector: NewBaseCollector("gpu", 15*time.Second),
	}

	if path, err := exec.LookPath("nvidia-smi"); err == nil {
		collector.nvidiaSMIPath = path
	} else {
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

	devices, summary, driverVersion, primaryModel, warnings, err := c.queryGPUMetrics(ctx)
	if err != nil {
		return nil, err
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

func (c *GPUCollector) queryGPUMetrics(ctx context.Context) ([]models.GPUDeviceMetrics, models.GPUSummary, string, string, []string, error) {
	var warnings []string
	if c.nvidiaSMIPath == "" {
		return nil, models.GPUSummary{}, "", "", append(warnings, "nvidia-smi binary not found"), nil
	}

	queryArgs := []string{
		"--query-gpu=index,name,uuid,driver_version,utilization.gpu,utilization.memory,memory.total,memory.used,temperature.gpu,fan.speed,power.draw,power.limit,clocks.sm,clocks.mem",
		"--format=csv,noheader,nounits",
	}
	cmd := exec.CommandContext(ctx, c.nvidiaSMIPath, queryArgs...)
	output, err := cmd.Output()
	if err != nil {
		// Handle no-device scenarios gracefully
		if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "No devices were found") || strings.Contains(string(output), "No devices were found") {
				warnings = append(warnings, "nvidia-smi reported no GPU devices")
				return []models.GPUDeviceMetrics{}, models.GPUSummary{DeviceCount: 0}, "", "", warnings, nil
			}
		}
		return nil, models.GPUSummary{}, "", "", warnings, fmt.Errorf("gpu query failed: %w", err)
	}

	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.TrimLeadingSpace = true

	var devices []models.GPUDeviceMetrics
	var summary models.GPUSummary
	var tempSamples int
	var driverVersion string
	var primaryModel string

	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, models.GPUSummary{}, "", "", warnings, fmt.Errorf("failed parsing gpu metrics: %w", readErr)
		}
		if len(record) < 14 {
			warnings = append(warnings, fmt.Sprintf("unexpected gpu record length: %d", len(record)))
			continue
		}

		device, devWarnings := parseGPUDeviceRecord(record)
		if device == nil {
			warnings = append(warnings, devWarnings...)
			continue
		}
		warnings = append(warnings, devWarnings...)

		if driverVersion == "" {
			driverVersion = strings.TrimSpace(record[3])
		}
		if primaryModel == "" {
			primaryModel = device.Name
		}

		devices = append(devices, *device)
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

	// Attach process information per GPU
	processesByUUID, procWarnings := c.queryGPUProcesses(ctx)
	warnings = append(warnings, procWarnings...)
	for idx := range devices {
		if processes, exists := processesByUUID[devices[idx].UUID]; exists {
			devices[idx].Processes = processes
		}
	}

	return devices, summary, driverVersion, primaryModel, warnings, nil
}

func (c *GPUCollector) queryGPUProcesses(ctx context.Context) (map[string][]models.GPUProcessInfo, []string) {
	if c.nvidiaSMIPath == "" {
		return map[string][]models.GPUProcessInfo{}, []string{"nvidia-smi binary not found for process query"}
	}

	cmd := exec.CommandContext(ctx, c.nvidiaSMIPath, "--query-compute-apps=pid,process_name,used_memory,gpu_uuid", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		// When there are no running processes, nvidia-smi returns a zero exit status with a message on stdout.
		if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr == "" && strings.Contains(string(output), "No running processes found") {
				return map[string][]models.GPUProcessInfo{}, nil
			}
		}
		return map[string][]models.GPUProcessInfo{}, []string{fmt.Sprintf("gpu process query failed: %v", err)}
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || strings.Contains(trimmed, "No running processes found") {
		return map[string][]models.GPUProcessInfo{}, nil
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.TrimLeadingSpace = true

	processes := make(map[string][]models.GPUProcessInfo)
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return processes, []string{fmt.Sprintf("failed parsing gpu process record: %v", readErr)}
		}
		if len(record) < 4 {
			continue
		}

		pid, _ := strconv.Atoi(strings.TrimSpace(record[0]))
		name := strings.TrimSpace(record[1])
		memMB, _ := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		uuid := strings.TrimSpace(record[3])

		if uuid == "" {
			continue
		}

		proc := models.GPUProcessInfo{
			PID:          pid,
			ProcessName:  name,
			MemoryUsedMB: memMB,
		}
		processes[uuid] = append(processes[uuid], proc)
	}

	return processes, nil
}

func parseGPUDeviceRecord(record []string) (*models.GPUDeviceMetrics, []string) {
	var warnings []string

	trim := func(input string) string { return strings.TrimSpace(input) }

	index, err := strconv.Atoi(trim(record[0]))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("invalid gpu index: %v", err))
		return nil, warnings
	}

	name := trim(record[1])
	uuid := trim(record[2])

	utilization := parseFloat(trim(record[4]))
	memoryUtil := parseFloat(trim(record[5]))
	memoryTotal := parseFloat(trim(record[6]))
	memoryUsed := parseFloat(trim(record[7]))

	temperature := parseOptionalFloat(trim(record[8]))
	fanSpeed := parseOptionalFloat(trim(record[9]))
	powerDraw := parseOptionalFloat(trim(record[10]))
	powerLimit := parseOptionalFloat(trim(record[11]))
	smClock := parseOptionalFloat(trim(record[12]))
	memoryClock := parseOptionalFloat(trim(record[13]))

	device := &models.GPUDeviceMetrics{
		Index:             index,
		UUID:              uuid,
		Name:              name,
		Utilization:       utilization,
		MemoryUtilization: memoryUtil,
		MemoryUsedMB:      memoryUsed,
		MemoryTotalMB:     memoryTotal,
		TemperatureC:      temperature,
		FanSpeedPercent:   fanSpeed,
		PowerDrawW:        powerDraw,
		PowerLimitW:       powerLimit,
		SMClockMHz:        smClock,
		MemoryClockMHz:    memoryClock,
	}

	return device, warnings
}

func parseFloat(value string) float64 {
	if value == "" || strings.EqualFold(value, "N/A") {
		return 0
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseOptionalFloat(value string) *float64 {
	if value == "" || strings.EqualFold(value, "N/A") {
		return nil
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &v
}
