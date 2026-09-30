// Package system provides system-level health checks
// [REQ:SYSTEM-GPU-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// GPUInfo contains information about a single GPU
type GPUInfo struct {
	Index           int     `json:"index"`
	Name            string  `json:"name"`
	MemoryTotal     uint64  `json:"memoryTotalMB"`
	MemoryUsed      uint64  `json:"memoryUsedMB"`
	MemoryFree      uint64  `json:"memoryFreeMB"`
	MemoryUsedPct   int     `json:"memoryUsedPercent"`
	Temperature     int     `json:"temperatureC"`
	UtilizationGPU  int     `json:"utilizationGPUPercent"`
	UtilizationMem  int     `json:"utilizationMemPercent"`
	PowerDraw       float64 `json:"powerDrawW"`
	PowerLimit      float64 `json:"powerLimitW"`
	FanSpeed        int     `json:"fanSpeedPercent"`
	DriverVersion   string  `json:"driverVersion,omitempty"`
	ComputeCapacity string  `json:"computeCapacity,omitempty"`
}

// GPUCheck monitors NVIDIA GPU health and utilization.
type GPUCheck struct {
	memoryWarning  int // Memory usage percentage to warn
	memoryCritical int // Memory usage percentage to go critical
	tempWarning    int // Temperature (C) to warn
	tempCritical   int // Temperature (C) to go critical
	hostCollector  hostSnapshotCollector
	executor       checks.CommandExecutor
	// placement answers whether the resources that asked for this card are
	// actually on it. nil leaves the check thresholds-only, which is what it
	// was before: a healthy card and an unused card looked identical.
	placement PlacementReporter
}

// GPUCheckOption configures a GPUCheck.
type GPUCheckOption func(*GPUCheck)

// WithGPUThresholds sets memory and temperature thresholds.
func WithGPUThresholds(memWarn, memCrit, tempWarn, tempCrit int) GPUCheckOption {
	return func(c *GPUCheck) {
		c.memoryWarning = memWarn
		c.memoryCritical = memCrit
		c.tempWarning = tempWarn
		c.tempCritical = tempCrit
	}
}

// WithGPUExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithGPUExecutor(executor checks.CommandExecutor) GPUCheckOption {
	return func(c *GPUCheck) {
		c.executor = executor
	}
}

// WithGPUPlacementReporter supplies the accelerator placement source.
func WithGPUPlacementReporter(reporter PlacementReporter) GPUCheckOption {
	return func(c *GPUCheck) {
		c.placement = reporter
	}
}

func WithGPUHostCollector(collector hostSnapshotCollector) GPUCheckOption {
	return func(c *GPUCheck) {
		c.hostCollector = collector
	}
}

// NewGPUCheck creates a GPU health check. Supply WithGPUPlacementReporter to
// make it answer the question a device-threshold check cannot: are the
// resources that asked for this card actually on it?
func NewGPUCheck(opts ...GPUCheckOption) *GPUCheck {
	c := &GPUCheck{
		memoryWarning:  80,
		memoryCritical: 95,
		tempWarning:    75,
		tempCritical:   85,
		hostCollector:  defaultHostSnapshotCollector{},
		executor:       checks.DefaultExecutor,
		placement:      CLIPlacementReporter{Executor: checks.DefaultExecutor},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GPUCheck) ID() string    { return "system-gpu" }
func (c *GPUCheck) Title() string { return "GPU Health" }
func (c *GPUCheck) Description() string {
	return "Monitors NVIDIA GPU memory, temperature and utilization, and reports every resource that declared an accelerator and is not on it"
}

func (c *GPUCheck) Importance() string {
	return "GPU health affects AI model performance, and an idle healthy card says nothing about whether the resources that asked for it are using it: a fleet on the CPU passes every threshold this check applies to the device itself"
}
func (c *GPUCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *GPUCheck) IntervalSeconds() int       { return 30 } // Check every 30 seconds for GPU
func (c *GPUCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *GPUCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}
	if checkOS != "linux" {
		result.Status = checks.StatusNotApplicable
		result.Message = "NVIDIA GPU check is not implemented on this platform"
		result.Details["platform"] = checkOS
		return result
	}

	collector := c.hostCollector
	if collector == nil {
		collector = defaultHostSnapshotCollector{}
	}
	snap, err := collector.Collect(ctx)
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to collect host GPU inventory"
		result.Details["error"] = err.Error()
		return result
	}

	if !snap.HasNvidiaGPU() {
		result.Status = checks.StatusOK
		result.Message = "No NVIDIA GPU detected"
		result.Details["hasGPU"] = false
		result.Details["note"] = "This is normal for systems without NVIDIA GPUs"
		return result
	}

	gpus := gpuInfoFromSnapshot(snap)
	if len(gpus) == 0 {
		result.Status = checks.StatusOK
		result.Message = "No GPUs found"
		result.Details["hasGPU"] = false
		return result
	}

	result.Details["hasGPU"] = true
	result.Details["gpuCount"] = len(gpus)
	result.Details["gpus"] = gpus
	result.Details["memoryWarningThreshold"] = c.memoryWarning
	result.Details["memoryCriticalThreshold"] = c.memoryCritical
	result.Details["tempWarningThreshold"] = c.tempWarning
	result.Details["tempCriticalThreshold"] = c.tempCritical

	// Determine overall status based on worst GPU
	var worstStatus checks.Status = checks.StatusOK
	var worstMessage string
	var subChecks []checks.SubCheck

	for _, gpu := range gpus {
		// Check memory usage
		memStatus := "ok"
		if gpu.MemoryUsedPct >= c.memoryCritical {
			memStatus = "critical"
			if worstStatus != checks.StatusCritical {
				worstStatus = checks.StatusCritical
				worstMessage = fmt.Sprintf("GPU %d memory critical: %d%% used", gpu.Index, gpu.MemoryUsedPct)
			}
		} else if gpu.MemoryUsedPct >= c.memoryWarning {
			memStatus = "warning"
			if worstStatus == checks.StatusOK {
				worstStatus = checks.StatusWarning
				worstMessage = fmt.Sprintf("GPU %d memory warning: %d%% used", gpu.Index, gpu.MemoryUsedPct)
			}
		}

		// Check temperature
		tempStatus := "ok"
		if gpu.Temperature >= c.tempCritical {
			tempStatus = "critical"
			if worstStatus != checks.StatusCritical {
				worstStatus = checks.StatusCritical
				worstMessage = fmt.Sprintf("GPU %d temperature critical: %d°C", gpu.Index, gpu.Temperature)
			}
		} else if gpu.Temperature >= c.tempWarning {
			tempStatus = "warning"
			if worstStatus == checks.StatusOK {
				worstStatus = checks.StatusWarning
				worstMessage = fmt.Sprintf("GPU %d temperature warning: %d°C", gpu.Index, gpu.Temperature)
			}
		}

		subChecks = append(subChecks,
			checks.SubCheck{
				Name:   fmt.Sprintf("gpu%d-memory", gpu.Index),
				Passed: memStatus != "critical",
				Detail: fmt.Sprintf("%d%% used (%d MB free)", gpu.MemoryUsedPct, gpu.MemoryFree),
			},
			checks.SubCheck{
				Name:   fmt.Sprintf("gpu%d-temperature", gpu.Index),
				Passed: tempStatus != "critical",
				Detail: fmt.Sprintf("%d°C", gpu.Temperature),
			},
		)
	}

	// Calculate overall score (average of memory and temp scores)
	totalScore := 0
	for _, gpu := range gpus {
		memScore := 100 - gpu.MemoryUsedPct
		tempScore := 100
		if gpu.Temperature > 0 {
			// Normalize temp to 0-100 scale (0°C = 100, 100°C = 0)
			tempScore = 100 - gpu.Temperature
			if tempScore < 0 {
				tempScore = 0
			}
		}
		totalScore += (memScore + tempScore) / 2
	}
	avgScore := totalScore / len(gpus)
	if avgScore < 0 {
		avgScore = 0
	}

	result.Metrics = &checks.HealthMetrics{
		Score:     &avgScore,
		SubChecks: subChecks,
	}

	result.Status = worstStatus
	if worstStatus == checks.StatusOK {
		result.Message = fmt.Sprintf("%d GPU(s) healthy", len(gpus))
	} else {
		result.Message = worstMessage
	}

	return applyPlacement(ctx, c.placement, result)
}

// RecoveryActions returns available recovery actions for GPU issues.
// [REQ:HEAL-ACTION-001]
func (c *GPUCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasGPU := false
	if lastResult != nil {
		if v, ok := lastResult.Details["hasGPU"].(bool); ok {
			hasGPU = v
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "gpu-status",
			Name:        "GPU Status",
			Description: "Show detailed GPU status and processes",
			Dangerous:   false,
			Available:   hasGPU,
		},
		{
			ID:          "gpu-processes",
			Name:        "GPU Processes",
			Description: "List processes using the GPU",
			Dangerous:   false,
			Available:   hasGPU,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *GPUCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "gpu-status":
		return c.executeGPUStatus(ctx, start)

	case "gpu-processes":
		return c.executeGPUProcesses(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeGPUStatus shows detailed GPU status
func (c *GPUCheck) executeGPUStatus(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "gpu-status",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== GPU Status ===\n\n")

	snap, err := c.collectHostSnapshot(ctx)
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to get GPU status"
		return result
	}
	payload := struct {
		GPUs         []sharedhost.GPU        `json:"gpus"`
		GPUProcesses []sharedhost.GPUProcess `json:"gpu_processes,omitempty"`
		Statuses     map[string]string       `json:"probe_statuses,omitempty"`
	}{
		GPUs:         snap.GPUs,
		GPUProcesses: snap.GPUProcesses,
		Statuses:     snap.ProbeStatuses,
	}
	if encoded, err := json.MarshalIndent(payload, "", "  "); err == nil {
		outputBuilder.Write(encoded)
		outputBuilder.WriteString("\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "GPU status retrieved"
	return result
}

// executeGPUProcesses lists processes using the GPU
func (c *GPUCheck) executeGPUProcesses(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "gpu-processes",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== GPU Processes ===\n\n")

	snap, err := c.collectHostSnapshot(ctx)
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to list GPU processes"
		return result
	}
	if len(snap.GPUProcesses) == 0 {
		outputBuilder.WriteString("No GPU processes reported by host inventory\n")
	} else if encoded, err := json.MarshalIndent(snap.GPUProcesses, "", "  "); err == nil {
		outputBuilder.Write(encoded)
		outputBuilder.WriteString("\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "GPU processes listed"
	return result
}

// MarshalJSON implements custom JSON marshaling for GPUInfo
func (g GPUInfo) MarshalJSON() ([]byte, error) {
	type Alias GPUInfo
	return json.Marshal(Alias(g))
}

func gpuInfoFromSnapshot(snap sharedhost.Snapshot) []GPUInfo {
	gpus := make([]GPUInfo, 0, len(snap.GPUs))
	for _, gpu := range snap.GPUs {
		if gpu.Source != "nvidia-smi" {
			continue
		}
		info := GPUInfo{
			Index:           gpu.Index,
			Name:            gpu.Name,
			MemoryTotal:     gpu.VRAMBytes / 1024 / 1024,
			MemoryUsed:      gpu.VRAMUsedBytes / 1024 / 1024,
			UtilizationGPU:  int(gpu.UtilizationPercent),
			UtilizationMem:  int(gpu.MemoryUtilizationPercent),
			DriverVersion:   gpu.DriverVersion,
			ComputeCapacity: "",
		}
		if info.MemoryTotal >= info.MemoryUsed {
			info.MemoryFree = info.MemoryTotal - info.MemoryUsed
		}
		if info.MemoryTotal > 0 {
			info.MemoryUsedPct = int((info.MemoryUsed * 100) / info.MemoryTotal)
		}
		if gpu.TemperatureC != nil {
			info.Temperature = int(*gpu.TemperatureC)
		}
		if gpu.PowerDrawW != nil {
			info.PowerDraw = *gpu.PowerDrawW
		}
		if gpu.PowerLimitW != nil {
			info.PowerLimit = *gpu.PowerLimitW
		}
		if gpu.FanSpeedPercent != nil {
			info.FanSpeed = int(*gpu.FanSpeedPercent)
		}
		gpus = append(gpus, info)
	}
	return gpus
}

func (c *GPUCheck) collectHostSnapshot(ctx context.Context) (sharedhost.Snapshot, error) {
	collector := c.hostCollector
	if collector == nil {
		collector = defaultHostSnapshotCollector{}
	}
	return collector.Collect(ctx)
}

// Ensure GPUCheck implements HealableCheck
var _ checks.HealableCheck = (*GPUCheck)(nil)
