// Package vrooli provides Vrooli-specific health checks
// [REQ:ORPHAN-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// VrooliProcessPatterns defines regex patterns that identify Vrooli-managed processes.
// Processes matching these patterns are candidates for orphan detection.
var VrooliProcessPatterns = regexp.MustCompile(
	`vrooli|/scenarios/[^/]+/(api|ui)|node_modules/.bin/vite|ecosystem-manager|picker-wheel`,
)

// DefaultGracePeriodSeconds is the default time to wait before considering a process an orphan.
// This prevents false positives for processes that are still starting up and haven't registered yet.
const DefaultGracePeriodSeconds = 30

// OrphanCheck detects Vrooli processes that are running but not tracked by the lifecycle system.
// These orphans can hold ports, consume resources, and prevent scenario restarts.
type OrphanCheck struct {
	warningThreshold   int     // count of orphans before warning
	criticalThreshold  int     // count of orphans before critical
	gracePeriodSeconds float64 // seconds to wait before considering a process an orphan
	stateReader        checks.VrooliStateReader
	procReader         checks.ProcReader
	executor           checks.CommandExecutor
}

// OrphanCheckOption configures an OrphanCheck.
type OrphanCheckOption func(*OrphanCheck)

// WithOrphanThresholds sets warning and critical thresholds (orphan counts).
func WithOrphanThresholds(warning, critical int) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithGracePeriod sets the grace period in seconds.
// Processes younger than this are not considered orphans.
func WithGracePeriod(seconds float64) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.gracePeriodSeconds = seconds
	}
}

// WithOrphanStateReader sets the state reader (for testing).
// [REQ:TEST-SEAM-001]
func WithOrphanStateReader(reader checks.VrooliStateReader) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.stateReader = reader
	}
}

// WithOrphanProcReader sets the proc reader (for testing).
// [REQ:TEST-SEAM-001]
func WithOrphanProcReader(reader checks.ProcReader) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.procReader = reader
	}
}

// WithOrphanExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithOrphanExecutor(executor checks.CommandExecutor) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.executor = executor
	}
}

// NewOrphanCheck creates an orphan process check.
// Default thresholds: warning at 3 orphans, critical at 10 orphans
// Default grace period: 30 seconds (processes younger than this are not considered orphans)
func NewOrphanCheck(opts ...OrphanCheckOption) *OrphanCheck {
	c := &OrphanCheck{
		warningThreshold:   3,
		criticalThreshold:  10,
		gracePeriodSeconds: DefaultGracePeriodSeconds,
		stateReader:        checks.DefaultVrooliStateReader,
		procReader:         checks.DefaultProcReader,
		executor:           checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *OrphanCheck) ID() string    { return "vrooli-orphans" }
func (c *OrphanCheck) Title() string { return "Orphan Processes" }
func (c *OrphanCheck) Description() string {
	return "Detects Vrooli processes running without lifecycle tracking"
}
func (c *OrphanCheck) Importance() string {
	return "Orphan processes can hold ports, consume resources, and prevent scenario restarts"
}
func (c *OrphanCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *OrphanCheck) IntervalSeconds() int       { return 120 }
func (c *OrphanCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *OrphanCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Get tracked PIDs from state files
	tracked, err := c.stateReader.ListTrackedProcesses()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read tracked processes"
		result.Details["error"] = err.Error()
		return result
	}

	// Build set of tracked PIDs and PGIDs
	trackedSet := make(map[int]bool)
	for _, proc := range tracked {
		if proc.PID > 0 {
			trackedSet[proc.PID] = true
		}
		if proc.PGID > 0 {
			trackedSet[proc.PGID] = true
		}
	}

	// Get all running processes
	allProcs, err := c.procReader.ListProcesses()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to list running processes"
		result.Details["error"] = err.Error()
		return result
	}

	// Find orphaned Vrooli processes
	var orphans []orphanProcessInfo
	var skippedYoung int // Count of processes skipped due to grace period
	for _, proc := range allProcs {
		// Skip non-Vrooli processes
		if !c.isVrooliProcess(proc) {
			continue
		}

		// Skip if this process or any ancestor is tracked
		if c.hasTrackedAncestor(proc.PID, allProcs, trackedSet) {
			continue
		}

		// Check process age - skip if within grace period
		age := checks.ProcessAge(proc.StartTime)
		if age > 0 && age < c.gracePeriodSeconds {
			// Process is too young - might still be starting up
			skippedYoung++
			continue
		}

		// This is an orphan (old enough to be considered one)
		orphans = append(orphans, orphanProcessInfo{
			PID:     proc.PID,
			PPID:    proc.PPid,
			Command: proc.Comm,
			Age:     age,
		})
	}

	orphanCount := len(orphans)
	result.Details["orphanCount"] = orphanCount
	result.Details["trackedCount"] = len(tracked)
	result.Details["skippedYoung"] = skippedYoung
	result.Details["gracePeriodSeconds"] = c.gracePeriodSeconds
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	if len(orphans) > 0 {
		// Limit to first 10 in details
		limit := 10
		if len(orphans) < limit {
			limit = len(orphans)
		}
		result.Details["orphans"] = orphans[:limit]
	}

	// Calculate score
	score := 100
	if orphanCount > 0 {
		// -10 points per orphan
		score = 100 - (orphanCount * 10)
		if score < 0 {
			score = 0
		}
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "orphan-count",
				Passed: orphanCount < c.criticalThreshold,
				Detail: fmt.Sprintf("%d orphan processes detected", orphanCount),
			},
		},
	}

	switch {
	case orphanCount >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Critical: %d orphan Vrooli processes detected", orphanCount)
	case orphanCount >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Warning: %d orphan Vrooli processes detected", orphanCount)
	case orphanCount > 0:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("%d orphan Vrooli processes (below threshold)", orphanCount)
	default:
		result.Status = checks.StatusOK
		result.Message = "No orphan Vrooli processes detected"
	}

	return result
}

// orphanProcessInfo contains information about an orphan process
type orphanProcessInfo struct {
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	Command string  `json:"command"`
	Age     float64 `json:"age"` // Age in seconds
}

// isVrooliProcess checks if a process matches Vrooli patterns
func (c *OrphanCheck) isVrooliProcess(proc checks.ProcessInfo) bool {
	// Check command name
	if VrooliProcessPatterns.MatchString(proc.Comm) {
		return true
	}

	// Skip self-check processes
	if strings.Contains(proc.Comm, "vrooli-autoheal") ||
		strings.Contains(proc.Comm, "orphan") ||
		strings.Contains(proc.Comm, "zombie-detector") {
		return false
	}

	return false
}

// hasTrackedAncestor checks if the process or any of its ancestors is tracked
func (c *OrphanCheck) hasTrackedAncestor(pid int, allProcs []checks.ProcessInfo, trackedSet map[int]bool) bool {
	// Build a map for quick PPID lookup
	ppidMap := make(map[int]int)
	for _, proc := range allProcs {
		ppidMap[proc.PID] = proc.PPid
	}

	// Walk up the process tree (max 10 levels to prevent infinite loops)
	currentPID := pid
	for depth := 0; depth < 10; depth++ {
		// Check if this PID is tracked
		if trackedSet[currentPID] {
			return true
		}

		// Get parent PID
		ppid, exists := ppidMap[currentPID]
		if !exists || ppid <= 1 || ppid == currentPID {
			break // Reached init or end of tree
		}

		currentPID = ppid
	}

	return false
}

// RecoveryActions returns available recovery actions for orphan cleanup
// [REQ:HEAL-ACTION-001]
func (c *OrphanCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasOrphans := false
	if lastResult != nil {
		if count, ok := lastResult.Details["orphanCount"].(int); ok {
			hasOrphans = count > 0
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "list",
			Name:        "List Orphans",
			Description: "Show all orphaned Vrooli processes with their details",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "kill",
			Name:        "Kill Orphans",
			Description: "Terminate all orphaned Vrooli processes (SIGTERM, then SIGKILL)",
			Dangerous:   true, // Dangerous! Could kill wrong processes
			Available:   hasOrphans,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *OrphanCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "list":
		return c.executeList(ctx, start)
	case "kill":
		return c.executeKill(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeList returns a detailed list of orphan processes
func (c *OrphanCheck) executeList(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "list",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Get tracked processes
	tracked, err := c.stateReader.ListTrackedProcesses()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	trackedSet := make(map[int]bool)
	for _, proc := range tracked {
		if proc.PID > 0 {
			trackedSet[proc.PID] = true
		}
		if proc.PGID > 0 {
			trackedSet[proc.PGID] = true
		}
	}

	// Get all processes
	allProcs, err := c.procReader.ListProcesses()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Orphan Process Detection ===\n\n")
	outputBuilder.WriteString(fmt.Sprintf("Tracked processes: %d\n", len(tracked)))
	outputBuilder.WriteString(fmt.Sprintf("Grace period: %.0f seconds\n", c.gracePeriodSeconds))
	outputBuilder.WriteString("Vrooli process patterns: vrooli, /scenarios/*/[api|ui], vite, ecosystem-manager, picker-wheel\n\n")

	orphanCount := 0
	youngCount := 0
	trackedVrooliCount := 0

	for _, proc := range allProcs {
		if !c.isVrooliProcess(proc) {
			continue
		}

		isOrphan := !c.hasTrackedAncestor(proc.PID, allProcs, trackedSet)
		age := checks.ProcessAge(proc.StartTime)

		var status string
		if !isOrphan {
			status = "TRACKED"
			trackedVrooliCount++
		} else if age > 0 && age < c.gracePeriodSeconds {
			status = "YOUNG (grace period)"
			youngCount++
		} else {
			status = "ORPHAN"
			orphanCount++
		}

		outputBuilder.WriteString(fmt.Sprintf("PID %d: %s\n", proc.PID, status))
		outputBuilder.WriteString(fmt.Sprintf("  Command: %s\n", proc.Comm))
		outputBuilder.WriteString(fmt.Sprintf("  Parent PID: %d\n", proc.PPid))
		if age > 0 {
			outputBuilder.WriteString(fmt.Sprintf("  Age: %.1f seconds\n", age))
		}
		if isOrphan && age >= c.gracePeriodSeconds {
			outputBuilder.WriteString(fmt.Sprintf("  Kill: kill %d\n", proc.PID))
		}
		outputBuilder.WriteString("\n")
	}

	outputBuilder.WriteString(fmt.Sprintf("Summary: %d tracked, %d orphaned, %d young (within grace period)\n", trackedVrooliCount, orphanCount, youngCount))

	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Found %d orphan processes", orphanCount)
	result.Output = outputBuilder.String()
	return result
}

// executeKill terminates all orphan processes
func (c *OrphanCheck) executeKill(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "kill",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Get tracked processes
	tracked, err := c.stateReader.ListTrackedProcesses()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	trackedSet := make(map[int]bool)
	for _, proc := range tracked {
		if proc.PID > 0 {
			trackedSet[proc.PID] = true
		}
		if proc.PGID > 0 {
			trackedSet[proc.PGID] = true
		}
	}

	// Get all processes
	allProcs, err := c.procReader.ListProcesses()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Killing Orphan Processes ===\n\n")
	outputBuilder.WriteString(fmt.Sprintf("Grace period: %.0f seconds (younger processes will be skipped)\n\n", c.gracePeriodSeconds))

	killed := 0
	failed := 0
	skippedYoung := 0

	for _, proc := range allProcs {
		if !c.isVrooliProcess(proc) {
			continue
		}

		if c.hasTrackedAncestor(proc.PID, allProcs, trackedSet) {
			continue
		}

		// Check process age - skip if within grace period
		age := checks.ProcessAge(proc.StartTime)
		if age > 0 && age < c.gracePeriodSeconds {
			outputBuilder.WriteString(fmt.Sprintf("Skipping PID %d (%s) - only %.1f seconds old (grace period)\n", proc.PID, proc.Comm, age))
			skippedYoung++
			continue
		}

		// This is an orphan old enough to kill
		outputBuilder.WriteString(fmt.Sprintf("Killing PID %d (%s, age: %.1fs)... ", proc.PID, proc.Comm, age))

		// First try SIGTERM
		if err := syscall.Kill(proc.PID, syscall.SIGTERM); err != nil {
			outputBuilder.WriteString(fmt.Sprintf("SIGTERM failed: %v, trying SIGKILL... ", err))
			if err := syscall.Kill(proc.PID, syscall.SIGKILL); err != nil {
				outputBuilder.WriteString(fmt.Sprintf("FAILED: %v\n", err))
				failed++
				continue
			}
		}

		// Wait a moment for the process to terminate
		time.Sleep(100 * time.Millisecond)

		// Verify it's gone
		if checks.ProcessExists(proc.PID) {
			// Try SIGKILL as last resort
			_ = syscall.Kill(proc.PID, syscall.SIGKILL)
			time.Sleep(100 * time.Millisecond)
			if checks.ProcessExists(proc.PID) {
				outputBuilder.WriteString("STILL RUNNING\n")
				failed++
				continue
			}
		}

		outputBuilder.WriteString("OK\n")
		killed++
	}

	if killed == 0 && failed == 0 && skippedYoung == 0 {
		outputBuilder.WriteString("No orphan processes to kill.\n")
	}

	outputBuilder.WriteString(fmt.Sprintf("\nSummary: %d killed, %d failed, %d skipped (grace period)\n", killed, failed, skippedYoung))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()

	if failed > 0 {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to kill %d processes", failed)
		result.Message = fmt.Sprintf("Partially killed: %d terminated, %d failed, %d skipped", killed, failed, skippedYoung)
	} else {
		result.Success = true
		if skippedYoung > 0 {
			result.Message = fmt.Sprintf("Killed %d orphan processes (%d skipped - within grace period)", killed, skippedYoung)
		} else {
			result.Message = fmt.Sprintf("Killed %d orphan processes", killed)
		}
	}

	return result
}

// GetOrphanPIDs returns a list of orphan PIDs for external use (e.g., diagnose-port)
// Respects the grace period - only returns PIDs of processes older than the grace period.
func (c *OrphanCheck) GetOrphanPIDs() ([]int, error) {
	tracked, err := c.stateReader.ListTrackedProcesses()
	if err != nil {
		return nil, err
	}

	trackedSet := make(map[int]bool)
	for _, proc := range tracked {
		if proc.PID > 0 {
			trackedSet[proc.PID] = true
		}
		if proc.PGID > 0 {
			trackedSet[proc.PGID] = true
		}
	}

	allProcs, err := c.procReader.ListProcesses()
	if err != nil {
		return nil, err
	}

	var orphanPIDs []int
	for _, proc := range allProcs {
		if !c.isVrooliProcess(proc) {
			continue
		}
		if c.hasTrackedAncestor(proc.PID, allProcs, trackedSet) {
			continue
		}
		// Skip processes within grace period
		age := checks.ProcessAge(proc.StartTime)
		if age > 0 && age < c.gracePeriodSeconds {
			continue
		}
		orphanPIDs = append(orphanPIDs, proc.PID)
	}

	return orphanPIDs, nil
}

// KillProcess kills a specific process by PID (exported for diagnose-port)
func KillProcess(pid int) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to kill PID %d", pid)
	}

	// Try SIGTERM first
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		return syscall.Kill(pid, syscall.SIGKILL)
	}

	// Wait for termination
	time.Sleep(100 * time.Millisecond)

	// If still running, use SIGKILL
	if checks.ProcessExists(pid) {
		return syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}

// GetProcessOnPort returns the PID of the process listening on a port (exported for diagnose-port)
func GetProcessOnPort(port int, executor checks.CommandExecutor) (int, error) {
	ctx := context.Background()
	output, err := executor.Output(ctx, "lsof", "-ti", fmt.Sprintf(":%d", port))
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found on port %d", port)
	}

	// Take first PID if multiple
	pids := strings.Fields(pidStr)
	if len(pids) == 0 {
		return 0, fmt.Errorf("no process found on port %d", port)
	}

	pid, err := strconv.Atoi(pids[0])
	if err != nil {
		return 0, fmt.Errorf("invalid PID %q: %w", pids[0], err)
	}

	return pid, nil
}

// Ensure OrphanCheck implements HealableCheck
var _ checks.HealableCheck = (*OrphanCheck)(nil)
