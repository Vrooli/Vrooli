// Responsibility: inspect and clean up stale runner processes during reconciliation.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"context"
	"fmt"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// isProcessAlive checks if the process for a run is still running.
func (r *Reconciler) isProcessAlive(ctx context.Context, run *domain.Run) bool {
	tag := run.GetTag()
	runnerType := "unknown"
	if run.ResolvedConfig != nil {
		runnerType = string(run.ResolvedConfig.RunnerType)
	}

	r.log().Debug("isProcessAlive check",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		obs.KeyRunnerType, runnerType,
	)

	// Method 1: Check via runner if available
	if r.runners != nil && run.ResolvedConfig != nil {
		if rr, err := r.runners.Get(run.ResolvedConfig.RunnerType); err == nil {
			if legacy, ok := rr.(interface {
				ContinuationTag(runner.ContinueRequest) string
			}); ok && run.RunnerPID > 0 {
				// Existing continuations used a synthesized tag. Require both the
				// recorded process and its exact codec tag; PID liveness alone
				// could mistake an unrelated reused PID for this run.
				if extractTagFromEnv(run.RunnerPID) == legacy.ContinuationTag(runner.ContinueRequest{RunID: run.ID}) {
					return true
				}
			}
		}
	}

	// Method 2: Scan /proc for the process
	alive := r.scanForProcess(tag)
	r.log().Debug("isProcessAlive result",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		"alive", alive,
	)
	return alive
}

// scanForProcess checks if the runner process for a run is still alive.
//
// We intentionally avoid "pgrep -f <tag>" because it matches ANY process whose
// command line contains the tag string — including child processes (shells, tee,
// cleanup handlers) that inherited the tag via environment variables. These
// lingering children cause false positives that prevent the reconciler from
// detecting dead runs.
//
// Instead, we scan only for known runner executables (claude, codex, opencode)
// and verify they carry the tag via either:
//   - --tag <tag> in their command line arguments, OR
//   - *_AGENT_TAG=<tag> in their /proc/<pid>/environ
func (r *Reconciler) scanForProcess(tag string) bool {
	found := r.scanRunnerProcessesByTag(tag)
	if found {
		r.log().Debug("runner process found", "tag", tag)
	} else {
		r.log().Debug("no runner process found", "tag", tag)
	}
	return found
}

// scanRunnerProcessesByTag checks if any known runner process (claude, codex, opencode)
// is alive with the given tag. It checks both command-line --tag arguments and
// environment variables for precise matching.
func (r *Reconciler) scanRunnerProcessesByTag(tag string) bool {
	for _, runnerName := range []string{"claude", "codex", "opencode"} {
		if r.scanRunnerProcessByTag(runnerName, tag) {
			return true
		}
	}
	return false
}

// scanRunnerProcessByTag checks if a specific runner type has a process with the given tag.
func (r *Reconciler) scanRunnerProcessByTag(runnerName, tag string) bool {
	cmd := exec.Command("pgrep", "-af", runnerName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		command := parts[1]

		// First check: does the command line have --tag with our specific tag?
		if cmdTag := extractTagFromCommand(command); cmdTag == tag {
			r.log().Debug("pid matched tag via --tag", "pid", pid, "tag", tag)
			return true
		}

		// Second check: does the process environment have the tag?
		if extractTagFromEnv(pid) == tag {
			r.log().Debug("pid matched tag via env", "pid", pid, "tag", tag)
			return true
		}
	}

	return false
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
		startTime := r.getProcessStartTime(pid)

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
		if strings.HasPrefix(part, "CLAUDE_CODE_AGENT_TAG=") {
			return strings.TrimPrefix(part, "CLAUDE_CODE_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "CODEX_AGENT_TAG=") {
			return strings.TrimPrefix(part, "CODEX_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "OPENCODE_AGENT_TAG=") {
			return strings.TrimPrefix(part, "OPENCODE_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "AGENT_TAG=") {
			return strings.TrimPrefix(part, "AGENT_TAG=")
		}
		if part == "--tag" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "--tag=") {
			return strings.TrimPrefix(part, "--tag=")
		}
	}
	return ""
}

func extractTagFromEnv(pid int) string {
	envPath := fmt.Sprintf("/proc/%d/environ", pid)
	data, err := os.ReadFile(envPath)
	if err != nil || len(data) == 0 {
		return ""
	}

	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, "CLAUDE_CODE_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "CLAUDE_CODE_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "CODEX_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "CODEX_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "OPENCODE_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "OPENCODE_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "AGENT_TAG=") {
			return strings.TrimPrefix(entry, "AGENT_TAG=")
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
		"heartbeat-",
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
func (r *Reconciler) getProcessStartTime(pid int) time.Time {
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
	bootTime := r.now().Add(-time.Duration(uptime * float64(time.Second)))
	startTime := bootTime.Add(time.Duration(processUptimeSeconds * float64(time.Second)))

	return startTime
}

// handleOrphan handles an orphan process.
func (r *Reconciler) handleOrphan(ctx context.Context, orphan OrphanProcess, stats *ReconcileStats) {
	r.log().Info("orphan process detected",
		"pid", orphan.PID,
		"tag", orphan.Tag,
		"runningSince", orphan.StartTime.Format(time.RFC3339),
	)

	if !r.config.KillOrphans {
		// Just log it, don't kill
		return
	}

	// Kill the orphan process
	if err := r.killProcess(orphan.PID); err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("failed to kill orphan %d: %v", orphan.PID, err))
	} else {
		stats.OrphansKilled++
		r.log().Info("orphan process killed", "pid", orphan.PID, "tag", orphan.Tag)

		// Clean up resource registries to remove stale entries
		r.cleanupResourceRegistries(ctx)
	}
}

// killRunProcesses finds and kills all processes associated with a run's tag.
func (r *Reconciler) killRunProcesses(ctx context.Context, run *domain.Run) {
	tag := run.GetTag()

	// Find PIDs matching the tag via runner process scan
	for _, runnerName := range []string{"claude", "codex", "opencode"} {
		cmd := exec.Command("pgrep", "-af", runnerName)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(output), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				continue
			}
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}

			command := parts[1]
			if extractTagFromCommand(command) == tag || extractTagFromEnv(pid) == tag {
				r.log().Info("killing run process", "pid", pid, "tag", tag)
				if err := r.killProcess(pid); err != nil {
					r.log().Warn("kill PID failed", "pid", pid, obs.KeyError, err.Error())
				}
			}
		}
	}

	r.cleanupResourceRegistries(ctx)
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
