// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the Reconciler which handles orphan detection and stale run
// recovery. It runs as a background service that periodically:
// - Detects runs that appear stuck (no heartbeat for too long)
// - Cleans up orphaned processes that are running without corresponding database records
// - Recovers from agent-manager crashes by reconciling actual state with DB state
//
// RECONCILIATION LOOP:
//   1. List all "running" runs from database
//   2. Check each run's heartbeat - mark as stale if too old
//   3. Scan for orphan processes that aren't tracked in DB
//   4. Handle stale runs (mark failed or attempt recovery)
//   5. Handle orphans (kill or adopt)

package orchestration

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// ReconcilerConfig holds configuration for the reconciliation service.
type ReconcilerConfig struct {
	// Interval is how often to run reconciliation
	Interval time.Duration

	// StaleThreshold is how long without heartbeat before marking a run as stale
	StaleThreshold time.Duration

	// OrphanGracePeriod is how long to wait before killing orphan processes
	OrphanGracePeriod time.Duration

	// MaxStaleRuns is the maximum number of stale runs to process per cycle
	MaxStaleRuns int

	// KillOrphans determines whether to automatically kill orphan processes
	KillOrphans bool

	// AutoRecover determines whether to automatically recover stale runs
	AutoRecover bool
}

// DefaultReconcilerConfig returns sensible defaults.
func DefaultReconcilerConfig() ReconcilerConfig {
	return ReconcilerConfig{
		Interval:          30 * time.Second,
		StaleThreshold:    2 * time.Minute,
		OrphanGracePeriod: 5 * time.Minute,
		MaxStaleRuns:      10,
		KillOrphans:       false, // Conservative default - don't auto-kill
		AutoRecover:       false, // Conservative default - don't auto-recover
	}
}

// Reconciler manages orphan detection and stale run recovery.
type Reconciler struct {
	runs    repository.RunRepository
	runners runner.Registry

	config ReconcilerConfig

	// State
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
	doneCh       chan struct{}
	lastRunTime  time.Time
	lastRunStats ReconcileStats

	// Broadcaster for real-time updates
	broadcaster EventBroadcaster
}

// ReconcileStats contains statistics from a reconciliation cycle.
type ReconcileStats struct {
	Timestamp     time.Time
	Duration      time.Duration
	RunsChecked   int
	StaleRuns     int
	OrphansFound  int
	RunsRecovered int
	OrphansKilled int
	Errors        []string
}

// NewReconciler creates a new reconciler with the given dependencies.
func NewReconciler(
	runs repository.RunRepository,
	runners runner.Registry,
	opts ...ReconcilerOption,
) *Reconciler {
	r := &Reconciler{
		runs:    runs,
		runners: runners,
		config:  DefaultReconcilerConfig(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReconcilerOption configures the reconciler.
type ReconcilerOption func(*Reconciler)

// WithReconcilerConfig sets custom configuration.
func WithReconcilerConfig(cfg ReconcilerConfig) ReconcilerOption {
	return func(r *Reconciler) {
		r.config = cfg
	}
}

// WithReconcilerBroadcaster sets the event broadcaster.
func WithReconcilerBroadcaster(b EventBroadcaster) ReconcilerOption {
	return func(r *Reconciler) {
		r.broadcaster = b
	}
}

// Start begins the reconciliation loop.
func (r *Reconciler) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("reconciler already running")
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.loop(ctx)
	log.Printf("[reconciler] Started with interval=%v, staleThreshold=%v",
		r.config.Interval, r.config.StaleThreshold)
	return nil
}

// Stop gracefully stops the reconciliation loop.
func (r *Reconciler) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	close(r.stopCh)
	<-r.doneCh

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	log.Printf("[reconciler] Stopped")
	return nil
}

// IsRunning returns whether the reconciler is active.
func (r *Reconciler) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// LastStats returns the statistics from the last reconciliation cycle.
func (r *Reconciler) LastStats() ReconcileStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastRunStats
}

// RunOnce performs a single reconciliation cycle.
// This is useful for testing or manual triggering.
func (r *Reconciler) RunOnce(ctx context.Context) ReconcileStats {
	return r.reconcile(ctx)
}

// loop runs the reconciliation loop.
func (r *Reconciler) loop(ctx context.Context) {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.config.Interval)
	defer ticker.Stop()

	// Run once immediately on startup
	stats := r.reconcile(ctx)
	r.updateStats(stats)

	for {
		select {
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := r.reconcile(ctx)
			r.updateStats(stats)
		}
	}
}

// updateStats updates the last run statistics.
func (r *Reconciler) updateStats(stats ReconcileStats) {
	r.mu.Lock()
	r.lastRunTime = stats.Timestamp
	r.lastRunStats = stats
	r.mu.Unlock()

	// Log summary
	if stats.StaleRuns > 0 || stats.OrphansFound > 0 {
		log.Printf("[reconciler] cycle: checked=%d stale=%d orphans=%d recovered=%d killed=%d errors=%d",
			stats.RunsChecked, stats.StaleRuns, stats.OrphansFound,
			stats.RunsRecovered, stats.OrphansKilled, len(stats.Errors))
	}
}

// reconcile performs the actual reconciliation work.
func (r *Reconciler) reconcile(ctx context.Context) ReconcileStats {
	start := time.Now()
	stats := ReconcileStats{Timestamp: start}

	// Step 1: Get all runs marked as "running" in the database
	runningStatus := domain.RunStatusRunning
	dbRuns, err := r.runs.List(ctx, repository.RunListFilter{
		Status: &runningStatus,
	})
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to list runs: "+err.Error())
		stats.Duration = time.Since(start)
		return stats
	}
	stats.RunsChecked = len(dbRuns)

	// Also check "starting" status runs
	startingStatus := domain.RunStatusStarting
	startingRuns, err := r.runs.List(ctx, repository.RunListFilter{
		Status: &startingStatus,
	})
	if err == nil {
		dbRuns = append(dbRuns, startingRuns...)
		stats.RunsChecked = len(dbRuns)
	}

	// Build a map of known run tags for orphan detection
	knownTags := make(map[string]*domain.Run)
	for _, run := range dbRuns {
		knownTags[run.GetTag()] = run
	}

	// Step 2: Check each run for staleness
	for _, run := range dbRuns {
		if run.IsStale(r.config.StaleThreshold) {
			stats.StaleRuns++
			r.handleStaleRun(ctx, run, &stats)
		}
	}

	// Step 3: Scan for orphan processes
	orphans := r.detectOrphanProcesses(ctx, knownTags)
	stats.OrphansFound = len(orphans)

	// Step 4: Handle orphans
	for _, orphan := range orphans {
		r.handleOrphan(ctx, orphan, &stats)
	}

	stats.Duration = time.Since(start)
	return stats
}

// handleStaleRun handles a run that appears to have stalled.
func (r *Reconciler) handleStaleRun(ctx context.Context, run *domain.Run, stats *ReconcileStats) {
	// First, check if the process is actually still running
	processAlive := r.isProcessAlive(ctx, run)

	if !processAlive {
		// Process died but DB wasn't updated - mark as failed
		log.Printf("[reconciler] Run %s process not found, marking as failed", run.ID)
		r.markRunFailed(ctx, run, "process terminated unexpectedly (detected by reconciler)")
		return
	}

	// Process is alive but heartbeat is stale - could be legitimate slow work
	log.Printf("[reconciler] Run %s is stale (last heartbeat: %v) but process is alive",
		run.ID, run.LastHeartbeat)

	if r.config.AutoRecover {
		// Attempt to recover by updating heartbeat and continuing
		now := time.Now()
		run.LastHeartbeat = &now
		run.UpdatedAt = now
		if err := r.runs.Update(ctx, run); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to update run %s: %v", run.ID, err))
		} else {
			stats.RunsRecovered++
		}
	}
}

// markRunFailed marks a run as failed due to unexpected termination.
func (r *Reconciler) markRunFailed(ctx context.Context, run *domain.Run, reason string) {
	now := time.Now()
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = reason
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		log.Printf("[reconciler] Failed to update run %s status: %v", run.ID, err)
	}

	// Broadcast status change
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

// isProcessAlive checks if the process for a run is still running.
func (r *Reconciler) isProcessAlive(ctx context.Context, run *domain.Run) bool {
	tag := run.GetTag()

	// Method 1: Check via runner if available
	if r.runners != nil && run.ResolvedConfig != nil {
		if runner, err := r.runners.Get(run.ResolvedConfig.RunnerType); err == nil {
			// Try to detect via runner's internal tracking
			// This requires the runner to implement a status check method
			// For now, fall through to process scanning
			_ = runner
		}
	}

	// Method 2: Scan /proc for the process
	return r.scanForProcess(tag)
}

// scanForProcess scans the process list for a process with the given tag.
func (r *Reconciler) scanForProcess(tag string) bool {
	// Use pgrep to find processes with the tag in their command line
	cmd := exec.Command("pgrep", "-f", tag)
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 if no processes found
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// OrphanProcess represents a process that's running but not tracked in the database.
type OrphanProcess struct {
	PID       int
	Tag       string
	Command   string
	StartTime time.Time
}

// detectOrphanProcesses scans for agent processes not tracked in the database.
func (r *Reconciler) detectOrphanProcesses(ctx context.Context, knownTags map[string]*domain.Run) []OrphanProcess {
	var orphans []OrphanProcess

	// Scan for claude-code processes
	orphans = append(orphans, r.scanRunnerProcesses("claude", knownTags)...)

	// Scan for codex processes
	orphans = append(orphans, r.scanRunnerProcesses("codex", knownTags)...)

	// Scan for opencode processes
	orphans = append(orphans, r.scanRunnerProcesses("opencode", knownTags)...)

	return orphans
}

// scanRunnerProcesses scans for processes of a specific runner type.
func (r *Reconciler) scanRunnerProcesses(runnerName string, knownTags map[string]*domain.Run) []OrphanProcess {
	var orphans []OrphanProcess

	// Look for processes with agent-manager tags
	// Tags are typically UUIDs or "scenario-taskid" format
	cmd := exec.Command("pgrep", "-af", runnerName)
	output, err := cmd.Output()
	if err != nil {
		return orphans
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse PID and command
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		command := parts[1]

		// Extract tag from command line (look for --tag argument)
		tag := extractTagFromCommand(command)
		if tag == "" {
			continue
		}

		// Check if this tag is known
		if _, known := knownTags[tag]; known {
			continue // Not an orphan
		}

		// Check if it looks like an agent-manager managed process
		// (UUIDs or known prefixes like "ecosystem-", "test-genie-")
		if !looksLikeAgentManagerTag(tag) {
			continue // Not our process
		}

		// Get process start time
		startTime := getProcessStartTime(pid)

		// Only consider it an orphan if it's been running longer than grace period
		if time.Since(startTime) < r.config.OrphanGracePeriod {
			continue // Too new, might be a race condition
		}

		orphans = append(orphans, OrphanProcess{
			PID:       pid,
			Tag:       tag,
			Command:   command,
			StartTime: startTime,
		})
	}

	return orphans
}

// extractTagFromCommand extracts the --tag value from a command line.
func extractTagFromCommand(command string) string {
	parts := strings.Fields(command)
	for i, part := range parts {
		if part == "--tag" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "--tag=") {
			return strings.TrimPrefix(part, "--tag=")
		}
	}
	return ""
}

// looksLikeAgentManagerTag checks if a tag looks like it was created by agent-manager.
func looksLikeAgentManagerTag(tag string) bool {
	// Check if it's a UUID
	if _, err := uuid.Parse(tag); err == nil {
		return true
	}

	// Check for known prefixes
	knownPrefixes := []string{
		"ecosystem-",
		"test-genie-",
		"agent-manager-",
		"run-",
	}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(tag, prefix) {
			return true
		}
	}

	return false
}

// getProcessStartTime gets the start time of a process.
func getProcessStartTime(pid int) time.Time {
	// Read process start time from /proc/[pid]/stat
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return time.Time{}
	}

	// The start time is field 22 (0-indexed: 21)
	// It's in clock ticks since boot
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return time.Time{}
	}

	startTicks, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Get system boot time
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}
	}
	uptimeStr := strings.Fields(string(uptimeData))[0]
	uptime, err := strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return time.Time{}
	}

	// Get clock ticks per second (usually 100)
	clkTck := int64(100) // Default, could read from sysconf

	// Calculate process start time
	processUptimeSeconds := float64(startTicks) / float64(clkTck)
	bootTime := time.Now().Add(-time.Duration(uptime * float64(time.Second)))
	startTime := bootTime.Add(time.Duration(processUptimeSeconds * float64(time.Second)))

	return startTime
}

// handleOrphan handles an orphan process.
func (r *Reconciler) handleOrphan(ctx context.Context, orphan OrphanProcess, stats *ReconcileStats) {
	log.Printf("[reconciler] Found orphan process: PID=%d tag=%s running since %v",
		orphan.PID, orphan.Tag, orphan.StartTime)

	if !r.config.KillOrphans {
		// Just log it, don't kill
		return
	}

	// Kill the orphan process
	if err := r.killProcess(orphan.PID); err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("failed to kill orphan %d: %v", orphan.PID, err))
	} else {
		stats.OrphansKilled++
		log.Printf("[reconciler] Killed orphan process: PID=%d tag=%s", orphan.PID, orphan.Tag)
	}
}

// killProcess kills a process with retry and escalation.
func (r *Reconciler) killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Try SIGTERM first
	if err := process.Signal(os.Interrupt); err != nil {
		// Process might already be dead
		return nil
	}

	// Wait a short time for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	// Check if still running
	if err := process.Signal(nil); err != nil {
		// Process is dead
		return nil
	}

	// Force kill with SIGKILL
	return process.Kill()
}
