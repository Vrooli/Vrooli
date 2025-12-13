// Package vrooli provides Vrooli-specific health checks
// [REQ:STALE-LOCK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// StaleLockCheck detects stale port lock files that point to dead processes.
// These locks prevent scenarios from binding to their assigned ports.
type StaleLockCheck struct {
	warningThreshold  int // count of stale locks before warning
	criticalThreshold int // count of stale locks before critical
	stateReader       checks.VrooliStateReader
}

// StaleLockCheckOption configures a StaleLockCheck.
type StaleLockCheckOption func(*StaleLockCheck)

// WithStaleLockThresholds sets warning and critical thresholds (lock counts).
func WithStaleLockThresholds(warning, critical int) StaleLockCheckOption {
	return func(c *StaleLockCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithStaleLockStateReader sets the state reader (for testing).
// [REQ:TEST-SEAM-001]
func WithStaleLockStateReader(reader checks.VrooliStateReader) StaleLockCheckOption {
	return func(c *StaleLockCheck) {
		c.stateReader = reader
	}
}

// NewStaleLockCheck creates a stale lock check.
// Default thresholds: warning at 3 stale locks, critical at 10 stale locks
func NewStaleLockCheck(opts ...StaleLockCheckOption) *StaleLockCheck {
	c := &StaleLockCheck{
		warningThreshold:  3,
		criticalThreshold: 10,
		stateReader:       checks.DefaultVrooliStateReader,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *StaleLockCheck) ID() string    { return "vrooli-stale-locks" }
func (c *StaleLockCheck) Title() string { return "Stale Port Locks" }
func (c *StaleLockCheck) Description() string {
	return "Detects stale port lock files that prevent scenario startup"
}
func (c *StaleLockCheck) Importance() string {
	return "Stale locks block scenarios from binding to their ports, causing startup failures"
}
func (c *StaleLockCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *StaleLockCheck) IntervalSeconds() int       { return 60 }
func (c *StaleLockCheck) Platforms() []platform.Type { return nil } // All platforms

func (c *StaleLockCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	locks, err := c.stateReader.ListPortLocks()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read port locks"
		result.Details["error"] = err.Error()
		return result
	}

	// Find stale locks (PID not running)
	var staleLocks []staleLockInfo
	for _, lock := range locks {
		if lock.PID <= 0 || !checks.ProcessExists(lock.PID) {
			staleLocks = append(staleLocks, staleLockInfo{
				Port:     lock.Port,
				Scenario: lock.Scenario,
				PID:      lock.PID,
				FilePath: lock.FilePath,
			})
		}
	}

	staleCount := len(staleLocks)
	result.Details["staleCount"] = staleCount
	result.Details["totalLocks"] = len(locks)
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	if len(staleLocks) > 0 {
		// Limit to first 10 in details
		limit := 10
		if len(staleLocks) < limit {
			limit = len(staleLocks)
		}
		result.Details["staleLocks"] = staleLocks[:limit]
	}

	// Calculate score
	score := 100
	if staleCount > 0 {
		// -10 points per stale lock
		score = 100 - (staleCount * 10)
		if score < 0 {
			score = 0
		}
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "stale-lock-count",
				Passed: staleCount < c.criticalThreshold,
				Detail: fmt.Sprintf("%d stale port locks detected", staleCount),
			},
		},
	}

	switch {
	case staleCount >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Critical: %d stale port locks detected", staleCount)
	case staleCount >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Warning: %d stale port locks detected", staleCount)
	case staleCount > 0:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("%d stale port locks (below threshold)", staleCount)
	default:
		result.Status = checks.StatusOK
		result.Message = "No stale port locks detected"
	}

	return result
}

// staleLockInfo contains information about a stale port lock
type staleLockInfo struct {
	Port     int    `json:"port"`
	Scenario string `json:"scenario"`
	PID      int    `json:"pid"`
	FilePath string `json:"filePath"`
}

// RecoveryActions returns available recovery actions for stale lock cleanup
// [REQ:HEAL-ACTION-001]
func (c *StaleLockCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasStale := false
	if lastResult != nil {
		if count, ok := lastResult.Details["staleCount"].(int); ok {
			hasStale = count > 0
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "list",
			Name:        "List Stale Locks",
			Description: "Show all stale port lock files with their details",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "clean",
			Name:        "Clean Stale Locks",
			Description: "Remove all stale port lock files (safe - only removes files for dead processes)",
			Dangerous:   false, // Safe because we only remove locks for non-running PIDs
			Available:   hasStale,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *StaleLockCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "list":
		return c.executeList(ctx, start)
	case "clean":
		return c.executeClean(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeList returns a detailed list of stale port locks
func (c *StaleLockCheck) executeList(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "list",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	locks, err := c.stateReader.ListPortLocks()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Port Lock Status ===\n\n")

	staleCount := 0
	validCount := 0

	for _, lock := range locks {
		isStale := lock.PID <= 0 || !checks.ProcessExists(lock.PID)
		status := "VALID"
		if isStale {
			status = "STALE"
			staleCount++
		} else {
			validCount++
		}

		outputBuilder.WriteString(fmt.Sprintf("Port %d: %s\n", lock.Port, status))
		outputBuilder.WriteString(fmt.Sprintf("  Scenario: %s\n", lock.Scenario))
		outputBuilder.WriteString(fmt.Sprintf("  PID: %d\n", lock.PID))
		outputBuilder.WriteString(fmt.Sprintf("  File: %s\n", lock.FilePath))
		if isStale {
			outputBuilder.WriteString("  Fix: rm '" + lock.FilePath + "'\n")
		}
		outputBuilder.WriteString("\n")
	}

	outputBuilder.WriteString(fmt.Sprintf("Summary: %d valid, %d stale, %d total\n",
		validCount, staleCount, len(locks)))

	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Found %d stale locks out of %d total", staleCount, len(locks))
	result.Output = outputBuilder.String()
	return result
}

// executeClean removes all stale port lock files
func (c *StaleLockCheck) executeClean(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "clean",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	locks, err := c.stateReader.ListPortLocks()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Cleaning Stale Port Locks ===\n\n")

	cleaned := 0
	failed := 0

	for _, lock := range locks {
		if lock.PID <= 0 || !checks.ProcessExists(lock.PID) {
			outputBuilder.WriteString(fmt.Sprintf("Removing lock for port %d (scenario: %s, dead PID: %d)... ",
				lock.Port, lock.Scenario, lock.PID))

			if err := c.stateReader.RemovePortLock(lock); err != nil {
				outputBuilder.WriteString(fmt.Sprintf("FAILED: %v\n", err))
				failed++
			} else {
				outputBuilder.WriteString("OK\n")
				cleaned++
			}
		}
	}

	if cleaned == 0 && failed == 0 {
		outputBuilder.WriteString("No stale locks to clean.\n")
	}

	outputBuilder.WriteString(fmt.Sprintf("\nSummary: %d cleaned, %d failed\n", cleaned, failed))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()

	if failed > 0 {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to remove %d lock files", failed)
		result.Message = fmt.Sprintf("Partially cleaned: %d removed, %d failed", cleaned, failed)
	} else {
		result.Success = true
		result.Message = fmt.Sprintf("Cleaned %d stale port locks", cleaned)
	}

	return result
}

// Ensure StaleLockCheck implements HealableCheck
var _ checks.HealableCheck = (*StaleLockCheck)(nil)

// PortDiagnostics contains comprehensive information about a port conflict
type PortDiagnostics struct {
	Port          int               `json:"port"`
	Scenario      string            `json:"scenario"`
	LockExists    bool              `json:"lockExists"`
	LockInfo      *PortLockInfo     `json:"lockInfo,omitempty"`
	ProcessOnPort *ProcessOnPortInfo `json:"processOnPort,omitempty"`
	Recommendation string           `json:"recommendation"`
	IsRecoverable  bool             `json:"isRecoverable"`
}

// PortLockInfo contains lock file information
type PortLockInfo struct {
	Scenario  string `json:"scenario"`
	PID       int    `json:"pid"`
	IsStale   bool   `json:"isStale"`
	FilePath  string `json:"filePath"`
}

// ProcessOnPortInfo contains information about a process using a port
type ProcessOnPortInfo struct {
	PID            int    `json:"pid"`
	Command        string `json:"command"`
	IsVrooliManaged bool  `json:"isVrooliManaged"`
	IsTracked      bool   `json:"isTracked"`
	Scenario       string `json:"scenario,omitempty"`
	Step           string `json:"step,omitempty"`
}

// DiagnosePort provides comprehensive diagnostics for a port conflict.
// This is called when a scenario fails to start due to port already in use.
// It returns actionable information about:
// - Whether there's a stale lock file for the port
// - What process is actually using the port
// - Whether that process is a Vrooli-managed process
// - Whether the process is tracked or an orphan
// - Recommended action to resolve the conflict
func DiagnosePort(port int, scenario string, executor checks.CommandExecutor) (*PortDiagnostics, error) {
	diag := &PortDiagnostics{
		Port:     port,
		Scenario: scenario,
	}

	// Check for lock file
	stateReader := checks.DefaultVrooliStateReader
	locks, err := stateReader.ListPortLocks()
	if err == nil {
		for _, lock := range locks {
			if lock.Port == port {
				diag.LockExists = true
				diag.LockInfo = &PortLockInfo{
					Scenario: lock.Scenario,
					PID:      lock.PID,
					IsStale:  lock.PID <= 0 || !checks.ProcessExists(lock.PID),
					FilePath: lock.FilePath,
				}
				break
			}
		}
	}

	// Find what process is using the port
	portPID, err := GetProcessOnPort(port, executor)
	if err == nil && portPID > 0 {
		diag.ProcessOnPort = &ProcessOnPortInfo{
			PID: portPID,
		}

		// Get command name
		procReader := checks.DefaultProcReader
		procs, _ := procReader.ListProcesses()
		for _, proc := range procs {
			if proc.PID == portPID {
				diag.ProcessOnPort.Command = proc.Comm
				break
			}
		}

		// Check if Vrooli-managed
		diag.ProcessOnPort.IsVrooliManaged = checks.IsVrooliManagedProcess(portPID)

		// Get Vrooli-specific info
		if info := checks.GetVrooliProcessInfo(portPID); info != nil {
			diag.ProcessOnPort.Scenario = info["VROOLI_SCENARIO"]
			diag.ProcessOnPort.Step = info["VROOLI_STEP"]
		}

		// Check if tracked
		tracked, _ := stateReader.ListTrackedProcesses()
		for _, t := range tracked {
			if t.PID == portPID || t.PGID == portPID {
				diag.ProcessOnPort.IsTracked = true
				break
			}
		}
	}

	// Determine recommendation
	diag.IsRecoverable = true
	switch {
	case diag.LockInfo != nil && diag.LockInfo.IsStale && diag.ProcessOnPort == nil:
		// Stale lock with no process - just need to clean the lock
		diag.Recommendation = fmt.Sprintf("Remove stale lock file: rm '%s'", diag.LockInfo.FilePath)

	case diag.ProcessOnPort != nil && !diag.ProcessOnPort.IsTracked && diag.ProcessOnPort.IsVrooliManaged:
		// Orphaned Vrooli process - safe to kill
		diag.Recommendation = fmt.Sprintf("Kill orphaned Vrooli process: kill %d", diag.ProcessOnPort.PID)

	case diag.ProcessOnPort != nil && diag.ProcessOnPort.IsTracked:
		// Tracked process - need to stop the owning scenario
		if diag.ProcessOnPort.Scenario != "" {
			diag.Recommendation = fmt.Sprintf("Stop the owning scenario: vrooli scenario stop %s", diag.ProcessOnPort.Scenario)
		} else {
			diag.Recommendation = fmt.Sprintf("Process is tracked but scenario unknown. Consider: kill %d", diag.ProcessOnPort.PID)
		}

	case diag.ProcessOnPort != nil && !diag.ProcessOnPort.IsVrooliManaged:
		// External process - user needs to handle this
		diag.Recommendation = fmt.Sprintf("Port is used by external process (PID %d: %s). Stop it manually or use a different port.",
			diag.ProcessOnPort.PID, diag.ProcessOnPort.Command)
		diag.IsRecoverable = false

	case diag.LockInfo != nil && !diag.LockInfo.IsStale:
		// Lock exists for running process
		diag.Recommendation = fmt.Sprintf("Port is locked by scenario '%s'. Stop it first: vrooli scenario stop %s",
			diag.LockInfo.Scenario, diag.LockInfo.Scenario)

	default:
		diag.Recommendation = "Unable to determine cause. Try: vrooli clean locks && vrooli orphans kill"
	}

	return diag, nil
}

// AutoRecoverPort attempts to automatically recover a port conflict.
// It only performs safe operations and returns whether recovery was successful.
func AutoRecoverPort(port int, scenario string, executor checks.CommandExecutor) (bool, string, error) {
	diag, err := DiagnosePort(port, scenario, executor)
	if err != nil {
		return false, "", err
	}

	if !diag.IsRecoverable {
		return false, diag.Recommendation, nil
	}

	var actions []string

	// Clean stale lock if present
	if diag.LockInfo != nil && diag.LockInfo.IsStale {
		stateReader := checks.DefaultVrooliStateReader
		err := stateReader.RemovePortLock(checks.PortLock{
			Port:     diag.LockInfo.PID,
			FilePath: diag.LockInfo.FilePath,
		})
		if err == nil {
			actions = append(actions, fmt.Sprintf("Removed stale lock for port %d", port))
		}
	}

	// Kill orphaned process if present
	if diag.ProcessOnPort != nil && !diag.ProcessOnPort.IsTracked && diag.ProcessOnPort.IsVrooliManaged {
		err := KillProcess(diag.ProcessOnPort.PID)
		if err == nil {
			actions = append(actions, fmt.Sprintf("Killed orphaned process %d", diag.ProcessOnPort.PID))
		}
	}

	if len(actions) > 0 {
		return true, strings.Join(actions, "; "), nil
	}

	return false, diag.Recommendation, nil
}
