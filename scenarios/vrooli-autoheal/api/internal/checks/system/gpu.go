// Package system provides system-level health checks
// [REQ:SYSTEM-GPU-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
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
// Uses nvidia-smi to gather metrics. Falls back gracefully if no GPU present.
type GPUCheck struct {
	memoryWarning  int // Memory usage percentage to warn
	memoryCritical int // Memory usage percentage to go critical
	tempWarning    int // Temperature (C) to warn
	tempCritical   int // Temperature (C) to go critical
	executor       checks.CommandExecutor
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

// NewGPUCheck creates a GPU health check.
// Default thresholds: memory warning 80%, critical 95%, temp warning 75C, critical 85C
func NewGPUCheck(opts ...GPUCheckOption) *GPUCheck {
	c := &GPUCheck{
		memoryWarning:  80,
		memoryCritical: 95,
		tempWarning:    75,
		tempCritical:   85,
		executor:       checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GPUCheck) ID() string    { return "system-gpu" }
func (c *GPUCheck) Title() string { return "GPU Health" }
func (c *GPUCheck) Description() string {
	return "Monitors NVIDIA GPU memory, temperature, and utilization for AI/ML workloads"
}
func (c *GPUCheck) Importance() string {
	return "GPU health affects AI model performance - overheating or memory exhaustion degrades ML inference"
}
func (c *GPUCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *GPUCheck) IntervalSeconds() int       { return 30 } // Check every 30 seconds for GPU
func (c *GPUCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *GPUCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Check if nvidia-smi is available
	_, err := c.executor.Output(ctx, "which", "nvidia-smi")
	if err != nil {
		result.Status = checks.StatusOK
		result.Message = "No NVIDIA GPU detected (nvidia-smi not found)"
		result.Details["hasGPU"] = false
		result.Details["note"] = "This is normal for systems without NVIDIA GPUs"
		return result
	}

	// Query GPU information using nvidia-smi JSON format
	output, err := c.executor.Output(ctx, "nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.used,memory.free,temperature.gpu,utilization.gpu,utilization.memory,power.draw,power.limit,fan.speed,driver_version,compute_cap",
		"--format=csv,noheader,nounits")

	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to query GPU information"
		result.Details["error"] = err.Error()
		result.Details["hasGPU"] = true
		return result
	}

	// Parse GPU information
	gpus, parseErr := c.parseGPUOutput(string(output))
	if parseErr != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to parse GPU information"
		result.Details["error"] = parseErr.Error()
		result.Details["rawOutput"] = string(output)
		return result
	}

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

	return result
}

// parseGPUOutput parses nvidia-smi CSV output into GPUInfo structs
func (c *GPUCheck) parseGPUOutput(output string) ([]GPUInfo, error) {
	var gpus []GPUInfo

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 13 {
			continue
		}

		gpu := GPUInfo{}

		// Parse each field with error handling
		if idx, err := strconv.Atoi(strings.TrimSpace(fields[0])); err == nil {
			gpu.Index = idx
		}
		gpu.Name = strings.TrimSpace(fields[1])

		if v, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64); err == nil {
			gpu.MemoryTotal = v
		}
		if v, err := strconv.ParseUint(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			gpu.MemoryUsed = v
		}
		if v, err := strconv.ParseUint(strings.TrimSpace(fields[4]), 10, 64); err == nil {
			gpu.MemoryFree = v
		}
		if v, err := strconv.Atoi(strings.TrimSpace(fields[5])); err == nil {
			gpu.Temperature = v
		}
		if v, err := strconv.Atoi(strings.TrimSpace(fields[6])); err == nil {
			gpu.UtilizationGPU = v
		}
		if v, err := strconv.Atoi(strings.TrimSpace(fields[7])); err == nil {
			gpu.UtilizationMem = v
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(fields[8]), 64); err == nil {
			gpu.PowerDraw = v
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(fields[9]), 64); err == nil {
			gpu.PowerLimit = v
		}
		if v, err := strconv.Atoi(strings.TrimSpace(fields[10])); err == nil {
			gpu.FanSpeed = v
		}
		gpu.DriverVersion = strings.TrimSpace(fields[11])
		gpu.ComputeCapacity = strings.TrimSpace(fields[12])

		// Calculate memory usage percentage
		if gpu.MemoryTotal > 0 {
			gpu.MemoryUsedPct = int((gpu.MemoryUsed * 100) / gpu.MemoryTotal)
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
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
		{
			ID:          "gpu-reset",
			Name:        "Reset GPU",
			Description: "Reset GPU to recover from errors (requires root, may interrupt workloads)",
			Dangerous:   true,
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

	case "gpu-reset":
		return c.executeGPUReset(ctx, start)

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

	// Full nvidia-smi output
	output, err := c.executor.CombinedOutput(ctx, "nvidia-smi")
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to get GPU status"
		return result
	}
	outputBuilder.Write(output)

	// Also get JSON format for programmatic access
	outputBuilder.WriteString("\n\n=== GPU Details (JSON) ===\n")
	jsonOutput, err := c.executor.Output(ctx, "nvidia-smi", "--query-gpu=timestamp,name,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw,power.limit,clocks.gr,clocks.mem,pstate", "--format=csv")
	if err == nil {
		outputBuilder.Write(jsonOutput)
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

	// Get compute processes
	output, err := c.executor.CombinedOutput(ctx, "nvidia-smi", "pmon", "-c", "1")
	if err != nil {
		// Try alternative approach
		output, err = c.executor.CombinedOutput(ctx, "nvidia-smi", "--query-compute-apps=pid,used_memory,gpu_name,process_name", "--format=csv")
		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to list GPU processes"
			return result
		}
	}
	outputBuilder.Write(output)

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "GPU processes listed"
	return result
}

// executeGPUReset attempts to reset the GPU
func (c *GPUCheck) executeGPUReset(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "gpu-reset",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== GPU Reset ===\n\n")

	// Try nvidia-smi --gpu-reset (requires root)
	output, err := c.executor.CombinedOutput(ctx, "sudo", "nvidia-smi", "--gpu-reset")
	outputBuilder.Write(output)

	if err != nil {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String()
		result.Success = false
		result.Error = err.Error()
		result.Message = "GPU reset failed (may require root privileges or GPU to be idle)"
		return result
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "GPU reset completed"
	return result
}

// MarshalJSON implements custom JSON marshaling for GPUInfo
func (g GPUInfo) MarshalJSON() ([]byte, error) {
	type Alias GPUInfo
	return json.Marshal(Alias(g))
}

// Ensure GPUCheck implements HealableCheck
var _ checks.HealableCheck = (*GPUCheck)(nil)
