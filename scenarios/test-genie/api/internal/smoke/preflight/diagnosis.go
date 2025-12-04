package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/smoke/orchestrator"
)

// HealthThresholds defines thresholds for health checks.
type HealthThresholds struct {
	// MaxChromeProcesses is the maximum acceptable Chrome process count.
	MaxChromeProcesses int
	// MaxMemoryPercent is the maximum acceptable memory usage percentage.
	MaxMemoryPercent float64
}

// DefaultHealthThresholds returns the standard health thresholds.
func DefaultHealthThresholds() HealthThresholds {
	return HealthThresholds{
		MaxChromeProcesses: 50,
		MaxMemoryPercent:   80.0,
	}
}

// DiagnoseBrowserlessFailure analyzes a browserless failure and returns structured diagnosis.
func (c *Checker) DiagnoseBrowserlessFailure(ctx context.Context, scenarioName string) *orchestrator.Diagnosis {
	diag, err := c.GetHealthDiagnostics(ctx)
	if err != nil {
		return &orchestrator.Diagnosis{
			Type:               orchestrator.DiagnosisOffline,
			Message:            "Browserless is unreachable",
			Recommendation:     "Start browserless: resource-browserless manage start\n  ↳ Then verify: vrooli resource browserless status",
			IsBrowserlessIssue: "true",
		}
	}

	thresholds := DefaultHealthThresholds()

	// Priority 1: Chrome process leak (most common cause)
	if diag.ChromeProcessCount > thresholds.MaxChromeProcesses {
		return &orchestrator.Diagnosis{
			Type: orchestrator.DiagnosisProcessLeak,
			Message: fmt.Sprintf("Chrome process leak detected (%d processes accumulated) - browserless cannot spawn new Chrome instances",
				diag.ChromeProcessCount),
			Recommendation: fmt.Sprintf(`Restart browserless to clear accumulated processes: docker restart vrooli-browserless
  ↳ Then verify: vrooli resource browserless status
  ↳ Then rerun: vrooli scenario ui-smoke %s
  ↳ Note: This is a known issue with browserless v2 - Chrome processes accumulate over time`, scenarioName),
			IsBrowserlessIssue: "true",
			Diagnostics:        diag,
		}
	}

	// Priority 2: Memory exhaustion
	if diag.MemoryUsagePercent > thresholds.MaxMemoryPercent {
		return &orchestrator.Diagnosis{
			Type: orchestrator.DiagnosisMemoryExhaustion,
			Message: fmt.Sprintf("Browserless memory exhaustion (%.1f%% used) - Chrome instances likely crashed",
				diag.MemoryUsagePercent),
			Recommendation: `Restart browserless: docker restart vrooli-browserless
  ↳ Then verify: vrooli resource browserless status
  ↳ If issue persists, the scenario may have memory-intensive UI operations`,
			IsBrowserlessIssue: "true",
			Diagnostics:        diag,
		}
	}

	// Priority 3: Chrome crashes detected
	if diag.ChromeCrashes > 0 {
		return &orchestrator.Diagnosis{
			Type: orchestrator.DiagnosisChromeCrashes,
			Message: fmt.Sprintf("Browserless instability detected - %d Chrome crash(es) in recent logs",
				diag.ChromeCrashes),
			Recommendation: fmt.Sprintf(`Restart browserless: docker restart vrooli-browserless
  ↳ Then verify: vrooli resource browserless status
  ↳ Then rerun: vrooli scenario ui-smoke %s`, scenarioName),
			IsBrowserlessIssue: "true",
			Diagnostics:        diag,
		}
	}

	// Priority 4: Degraded but no specific cause
	if diag.Status != "healthy" && diag.Status != "" {
		return &orchestrator.Diagnosis{
			Type:    orchestrator.DiagnosisDegraded,
			Message: "Browserless health degraded - check diagnostics",
			Recommendation: `Check health: vrooli resource browserless status
  ↳ Review logs: docker logs vrooli-browserless --tail 50
  ↳ Consider restart: docker restart vrooli-browserless`,
			IsBrowserlessIssue: "maybe",
			Diagnostics:        diag,
		}
	}

	// Priority 5: Unknown - might be scenario issue
	return &orchestrator.Diagnosis{
		Type:    orchestrator.DiagnosisUnknown,
		Message: "Browserless returned invalid response (status appears healthy)",
		Recommendation: fmt.Sprintf(`This may be a scenario UI issue rather than browserless
  ↳ Check scenario logs: vrooli scenario logs %s
  ↳ Verify UI serves correctly: curl http://localhost:$UI_PORT
  ↳ Check browserless: vrooli resource browserless status`, scenarioName),
		IsBrowserlessIssue: "false",
		Diagnostics:        diag,
	}
}

// GetHealthDiagnostics retrieves detailed health metrics from browserless.
func (c *Checker) GetHealthDiagnostics(ctx context.Context) (*orchestrator.HealthDiagnostics, error) {
	diag := &orchestrator.HealthDiagnostics{}

	// Get pressure/session info from browserless API
	pressureURL := c.browserlessURL + "/pressure"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pressureURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pressure request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("browserless pressure endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var pressure struct {
			Pressure struct {
				Running    int `json:"running"`
				Queued     int `json:"queued"`
				Concurrent int `json:"concurrent"`
			} `json:"pressure"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&pressure); err == nil {
			diag.RunningSessions = pressure.Pressure.Running
			diag.QueuedSessions = pressure.Pressure.Queued
			diag.MaxConcurrent = pressure.Pressure.Concurrent
		}
	}

	// Try to get detailed diagnostics from resource-browserless CLI
	if out, err := c.cmdExecutor.Execute(ctx, "resource-browserless", "status", "--format", "json"); err == nil {
		var status struct {
			Running       bool `json:"running"`
			Configuration struct {
				Port int `json:"port"`
			} `json:"configuration"`
			Health struct {
				ChromeProcesses int     `json:"chrome_processes"`
				MemoryPercent   float64 `json:"memory_percent"`
				ChromeCrashes   int     `json:"chrome_crashes"`
				Status          string  `json:"status"`
			} `json:"health"`
		}
		if err := json.Unmarshal(out, &status); err == nil {
			diag.ChromeProcessCount = status.Health.ChromeProcesses
			diag.MemoryUsagePercent = status.Health.MemoryPercent
			diag.ChromeCrashes = status.Health.ChromeCrashes
			diag.Status = status.Health.Status
			return diag, nil
		}
	}

	// Fallback: try to get Chrome process count via docker exec
	chromeCount, _ := c.getChromeProcessCount(ctx)
	diag.ChromeProcessCount = chromeCount

	// Fallback: try to get memory usage via docker stats
	memPercent, _ := c.getContainerMemoryPercent(ctx)
	diag.MemoryUsagePercent = memPercent

	// Fallback: try to get Chrome crash count from docker logs
	crashCount, _ := c.getChromeCrashCount(ctx)
	diag.ChromeCrashes = crashCount

	// Determine overall status based on metrics
	thresholds := DefaultHealthThresholds()
	if diag.ChromeProcessCount == 0 && diag.MemoryUsagePercent == 0 && diag.ChromeCrashes == 0 {
		diag.Status = "healthy" // Assume healthy if we can't get metrics
	} else if diag.ChromeProcessCount > thresholds.MaxChromeProcesses ||
		diag.MemoryUsagePercent > thresholds.MaxMemoryPercent ||
		diag.ChromeCrashes > 0 {
		diag.Status = "degraded"
	} else {
		diag.Status = "healthy"
	}

	return diag, nil
}

// getChromeProcessCount gets the number of Chrome processes in the browserless container.
func (c *Checker) getChromeProcessCount(ctx context.Context) (int, error) {
	// Try docker exec to count Chrome processes
	out, err := c.cmdExecutor.Execute(ctx, "docker", "exec", "vrooli-browserless", "pgrep", "-c", "chrome")
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return count, nil
}

// getContainerMemoryPercent gets the memory usage percentage for the browserless container.
func (c *Checker) getContainerMemoryPercent(ctx context.Context) (float64, error) {
	out, err := c.cmdExecutor.Execute(ctx, "docker", "stats", "--no-stream", "--format", "{{.MemPerc}}", "vrooli-browserless")
	if err != nil {
		return 0, err
	}

	// Parse "45.67%" format
	memStr := strings.TrimSpace(string(out))
	memStr = strings.TrimSuffix(memStr, "%")
	memPct, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, err
	}
	return memPct, nil
}

// getChromeCrashCount checks docker logs for Chrome crash messages.
// Returns the count of "Page crashed!" occurrences in recent logs.
func (c *Checker) getChromeCrashCount(ctx context.Context) (int, error) {
	// Get recent logs from browserless container
	out, err := c.cmdExecutor.Execute(ctx, "docker", "logs", "vrooli-browserless", "--tail", "100")
	if err != nil {
		return 0, err
	}

	// Count occurrences of "Page crashed!" which indicates Chrome crashes
	logs := string(out)
	count := strings.Count(logs, "Page crashed!")
	return count, nil
}

// IsHealthy checks if browserless is healthy based on thresholds.
func (c *Checker) IsHealthy(ctx context.Context) (bool, *orchestrator.HealthDiagnostics, error) {
	diag, err := c.GetHealthDiagnostics(ctx)
	if err != nil {
		return false, nil, err
	}

	thresholds := DefaultHealthThresholds()
	healthy := diag.ChromeProcessCount <= thresholds.MaxChromeProcesses &&
		diag.MemoryUsagePercent <= thresholds.MaxMemoryPercent

	return healthy, diag, nil
}

// RecoveryExecutor abstracts recovery command execution for testing.
type RecoveryExecutor interface {
	// RestartContainer restarts the browserless container.
	RestartContainer(ctx context.Context) error
	// CleanupProcesses attempts to clean up leaked Chrome processes.
	CleanupProcesses(ctx context.Context) error
}

// DefaultRecoveryExecutor implements RecoveryExecutor using docker commands.
type DefaultRecoveryExecutor struct {
	containerName string
}

// NewDefaultRecoveryExecutor creates a new DefaultRecoveryExecutor.
func NewDefaultRecoveryExecutor(containerName string) *DefaultRecoveryExecutor {
	return &DefaultRecoveryExecutor{containerName: containerName}
}

// RestartContainer restarts the browserless container.
func (r *DefaultRecoveryExecutor) RestartContainer(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "restart", r.containerName)
	return cmd.Run()
}

// CleanupProcesses attempts to kill leaked Chrome processes in the container.
func (r *DefaultRecoveryExecutor) CleanupProcesses(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Kill zombie Chrome processes
	cmd := exec.CommandContext(ctx, "docker", "exec", r.containerName, "pkill", "-9", "-f", "chrome")
	_ = cmd.Run() // Ignore errors - processes may not exist

	return nil
}


// EnsureHealthy checks browserless health and attempts auto-recovery if unhealthy.
// Returns the recovery result and any error.
func (c *Checker) EnsureHealthy(ctx context.Context, opts orchestrator.AutoRecoveryOptions) (*orchestrator.RecoveryResult, error) {
	result := &orchestrator.RecoveryResult{Attempted: false}

	// Get initial diagnostics
	healthy, diagBefore, err := c.IsHealthy(ctx)
	if err != nil {
		// Can't get diagnostics - browserless may be offline
		return result, fmt.Errorf("failed to check browserless health: %w", err)
	}
	result.DiagnosticsBefore = diagBefore

	if healthy {
		result.Success = true
		return result, nil
	}

	// Check shared mode - don't restart if other sessions are active
	if opts.SharedMode && diagBefore.RunningSessions > 0 {
		result.Action = fmt.Sprintf("skipped: shared mode with %d active session(s)", diagBefore.RunningSessions)
		return result, fmt.Errorf("browserless unhealthy (chrome=%d, mem=%.1f%%) but cannot recover: %s",
			diagBefore.ChromeProcessCount, diagBefore.MemoryUsagePercent, result.Action)
	}

	// Attempt recovery
	result.Attempted = true
	executor := NewDefaultRecoveryExecutor(opts.ContainerName)

	// Step 1: Try cleanup first
	result.Action = "cleanup"
	if err := executor.CleanupProcesses(ctx); err != nil {
		// Cleanup failed, but continue to restart
	}

	// Wait briefly for cleanup to take effect
	time.Sleep(2 * time.Second)

	// Check if cleanup was sufficient
	healthy, diagAfter, err := c.IsHealthy(ctx)
	if err == nil && healthy {
		result.Success = true
		result.DiagnosticsAfter = diagAfter
		result.Action = "cleanup succeeded"
		return result, nil
	}

	// Step 2: Cleanup wasn't enough, restart container
	result.Action = "restart"
	if err := executor.RestartContainer(ctx); err != nil {
		result.Error = fmt.Sprintf("restart failed: %v", err)
		return result, fmt.Errorf("browserless recovery failed: %w", err)
	}

	// Wait for container to be ready
	time.Sleep(3 * time.Second)

	// Verify recovery succeeded
	healthy, diagAfter, err = c.IsHealthy(ctx)
	result.DiagnosticsAfter = diagAfter
	if err != nil {
		result.Error = fmt.Sprintf("post-restart health check failed: %v", err)
		return result, fmt.Errorf("browserless recovery verification failed: %w", err)
	}

	if !healthy {
		result.Error = "still unhealthy after restart"
		return result, fmt.Errorf("browserless still unhealthy after restart (chrome=%d, mem=%.1f%%)",
			diagAfter.ChromeProcessCount, diagAfter.MemoryUsagePercent)
	}

	result.Success = true
	result.Action = "restart succeeded"
	return result, nil
}
