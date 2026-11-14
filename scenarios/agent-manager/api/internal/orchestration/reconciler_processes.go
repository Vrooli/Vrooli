// Responsibility: inspect and clean up stale runner processes during reconciliation.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
)

// reconcilerProcessNames is the verified run-tag process subset shared by
// liveness, orphan discovery and cleanup. Grok has no verified Agent Manager
// run-tag protocol; Antigravity is not an Agent Manager runner. Descriptor
// binary names alone must never admit either to process discovery or cleanup.
var reconcilerProcessNames = []string{"claude", "codex", "opencode"}

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
				if r.processEffects.ProcessTag(run.RunnerPID) == legacy.ContinuationTag(runner.ContinueRequest{RunID: run.ID}) {
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
	for _, runnerName := range reconcilerProcessNames {
		if r.scanRunnerProcessByTag(runnerName, tag) {
			return true
		}
	}
	return false
}

// scanRunnerProcessByTag checks if a specific runner type has a process with the given tag.
func (r *Reconciler) scanRunnerProcessByTag(runnerName, tag string) bool {
	processes, err := r.processEffects.DiscoverRunnerProcesses(context.Background(), runnerName)
	if err != nil {
		return false
	}
	for _, process := range processes {
		pid, command := process.PID, process.Command
		// Existing executors predate store markers. Their exact runner/tag
		// evidence still establishes liveness; it never authorizes orphan
		// termination. Explicit evidence of another store remains excluded.
		if process.OwnerID != "" && process.OwnerID != r.config.StoreOwnerID {
			continue
		}

		// First check: does the command line have --tag with our specific tag?
		if cmdTag := extractTagFromCommand(command); cmdTag == tag {
			r.log().Debug("pid matched tag via --tag", "pid", pid, "tag", tag)
			return true
		}

		// Second check: does the process environment have the tag?
		if r.processEffects.ProcessTag(pid) == tag {
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

	for _, runnerName := range reconcilerProcessNames {
		for _, candidate := range r.scanRunnerProcesses(runnerName, knownTags) {
			retained, err := r.retainsInteractiveExecutor(ctx, candidate.Tag)
			if err != nil {
				r.log().Warn("orphan ownership lookup unavailable; process retained", "pid", candidate.PID, "tag", candidate.Tag, obs.KeyError, err.Error())
				continue
			}
			if !retained {
				orphans = append(orphans, candidate)
			}
		}
	}

	return orphans
}

// A terminal turn or a parked native goal can retain its provider for later
// continuation. Absence from heartbeat scanning is not permission to reap it.
func (r *Reconciler) retainsInteractiveExecutor(ctx context.Context, tag string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if r.runs == nil {
		return false, fmt.Errorf("run ownership repository is unavailable")
	}
	// Prefix lookup is the repository's maintained tag surface. Bound the read;
	// ambiguous/truncated results cannot prove absence of an exact owner.
	runs, err := r.runs.List(ctx, repository.RunListFilter{TagPrefix: tag, ListFilter: repository.ListFilter{Limit: 2}})
	if err != nil {
		return false, err
	}
	for _, candidate := range runs {
		if candidate.GetTag() != tag {
			continue
		}
		stored, err := r.runs.Get(ctx, candidate.ID)
		if err != nil {
			return false, err
		}
		if stored == nil {
			return false, fmt.Errorf("run owner disappeared during orphan classification")
		}
		if stored.ExecutionMode.Normalized() == domain.ExecutionModeInteractive && strings.TrimSpace(stored.WebConsoleSessionID) != "" && stored.Status != domain.RunStatusCancelled {
			if r.sessions == nil {
				return false, fmt.Errorf("retained interactive session owner is unavailable")
			}
			session, err := r.sessions.GetSession(ctx, stored.WebConsoleSessionID)
			if errors.Is(err, webconsole.ErrSessionNotFound) {
				continue
			} // Positive retirement evidence.
			if err != nil {
				return false, err
			}
			if session.ID != stored.WebConsoleSessionID {
				return false, fmt.Errorf("retained interactive session lookup changed identity")
			}
			if session.LaunchMode == "codex_app_server" && (session.NativeThreadID != stored.SessionID || session.NativeOwner == "" || session.ControlMode != "native_capable") {
				return false, fmt.Errorf("retained native owner binding is unknown")
			}
			return true, nil
		}
	}
	if len(runs) >= 2 {
		return false, fmt.Errorf("bounded tag lookup cannot prove complete ownership inventory")
	}
	return false, nil
}

// scanRunnerProcesses scans for processes of a specific runner type.
func (r *Reconciler) scanRunnerProcesses(runnerName string, knownTags map[string]*domain.Run) []OrphanProcess {
	var orphans []OrphanProcess

	// Look for processes with agent-manager tags
	// Tags are typically UUIDs or "scenario-taskid" format
	processes, err := r.processEffects.DiscoverRunnerProcesses(context.Background(), runnerName)
	if err != nil {
		return orphans
	}
	for _, process := range processes {
		pid, command := process.PID, process.Command
		// A matching tag or a missing row in this database is not ownership
		// evidence. Separate Agent Manager stores share the host process space.
		// Require the marker embedded by our launch owners before any other
		// orphan classification or termination decision.
		if r.config.StoreOwnerID == "" || process.OwnerID != r.config.StoreOwnerID {
			continue
		}

		// Resolve the tag from the command line, then the environment. codex
		// injects its tag only via CODEX_AGENT_TAG in the environment (never
		// argv), and `pgrep -af` reports argv but not the environment — so a
		// command-only read classifies every detached codex process as tagless
		// and skips it forever (reconciler_processes.go history: 40+ codex
		// processes survived weeks past their terminal runs). Mirror the
		// liveness scan, which already consults both (scanRunnerProcessByTag).
		tag := extractTagFromCommand(command)
		if tag == "" {
			tag = r.processEffects.ProcessTag(pid)
		}
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

func extractStoreOwnerIDFromEnv(pid int) string {
	envPath := fmt.Sprintf("/proc/%d/environ", pid)
	data, err := os.ReadFile(envPath)
	if err != nil || len(data) == 0 {
		return ""
	}
	prefix := phases.AgentManagerStoreOwnerEnv + "="
	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

// runnerProcessTag resolves the agent-manager tag for a scanned runner process,
// preferring the command line and falling back to the process environment.
// Env-only-tagged runners (codex sets CODEX_AGENT_TAG in the environment, never
// argv) are invisible to a command-line-only read, so any caller that must
// recognize every managed runner process — orphan detection in particular —
// has to consult both sources.
func runnerProcessTag(command string, pid int) string {
	if tag := extractTagFromCommand(command); tag != "" {
		return tag
	}
	return extractTagFromEnv(pid)
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
	return r.processEffects.ProcessStartTime(pid)
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
	for _, runnerName := range reconcilerProcessNames {
		processes, err := r.processEffects.DiscoverRunnerProcesses(ctx, runnerName)
		if err != nil {
			continue
		}
		for _, process := range processes {
			pid, command := process.PID, process.Command
			if r.config.StoreOwnerID == "" || process.OwnerID != r.config.StoreOwnerID {
				continue
			}
			if extractTagFromCommand(command) == tag || r.processEffects.ProcessTag(pid) == tag {
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
//
// Managed runners are launched as session leaders (Setsid), so the agent binary
// and everything it spawns (node, shells, cleanup handlers) share a single
// process group. Signalling only the leader PID leaves those children resident —
// the residue observed behind reaped codex leaders — so the force step targets
// the whole process group, matching the Terminator's KillProcessGroup behaviour.
func (r *Reconciler) killProcess(pid int) error {
	return r.processEffects.KillProcess(pid)
}

func killHostProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	defer process.Release()
	return terminateReconcilerProcess(pid, processTerminationEffects{
		groupID:   processGroupID,
		graceful:  func() error { return requestGracefulProcessStop(process) },
		alive:     platform.IsPIDRunning,
		wait:      time.Sleep,
		killGroup: func(group int) error { return platform.SignalProcessGroup(group, true) },
		confirmGroupExit: func(group int) error {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			return platform.WaitProcessGroupContext(ctx, group, 25*time.Millisecond)
		},
		killPID: process.Kill,
	})
}

type hostReconcilerProcessEffects struct{}

func (hostReconcilerProcessEffects) DiscoverRunnerProcesses(_ context.Context, runnerName string) ([]discoveredRunnerProcess, error) {
	cmd := exec.Command("pgrep", "-af", runnerName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var processes []discoveredRunnerProcess
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		pid, err := strconv.Atoi(parts[0])
		if err == nil {
			processes = append(processes, discoveredRunnerProcess{PID: pid, Command: parts[1], OwnerID: extractStoreOwnerIDFromEnv(pid)})
		}
	}
	return processes, nil
}
func (hostReconcilerProcessEffects) ProcessStartTime(pid int) time.Time {
	return hostProcessStartTime(pid, time.Now())
}
func (hostReconcilerProcessEffects) ProcessTag(pid int) string { return extractTagFromEnv(pid) }
func (hostReconcilerProcessEffects) ProcessAlive(pid int) bool {
	return platform.IsPIDRunning(pid)
}
func (hostReconcilerProcessEffects) AttachedIdentityAlive(hash string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
		if err != nil {
			continue
		}
		for _, value := range strings.Split(string(data), "\x00") {
			if strings.HasPrefix(value, "VROOLI_AGENT_IDENTITY_TOKEN=") {
				token := strings.TrimPrefix(value, "VROOLI_AGENT_IDENTITY_TOKEN=")
				if token != "" && identity.HashToken(token) == hash {
					return true
				}
			}
		}
	}
	return false
}
func (hostReconcilerProcessEffects) KillProcess(pid int) error { return killHostProcess(pid) }
func (hostReconcilerProcessEffects) CleanupRunnerResources(ctx context.Context) error {
	return cleanupHostResourceRegistries(ctx)
}

func hostProcessStartTime(pid int, now time.Time) time.Time {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return time.Time{}
	}
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return time.Time{}
	}
	startTicks, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return time.Time{}
	}
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}
	}
	uptimeFields := strings.Fields(string(uptimeData))
	if len(uptimeFields) == 0 {
		return time.Time{}
	}
	uptime, err := strconv.ParseFloat(uptimeFields[0], 64)
	if err != nil {
		return time.Time{}
	}
	processUptimeSeconds := float64(startTicks) / 100
	bootTime := now.Add(-time.Duration(uptime * float64(time.Second)))
	return bootTime.Add(time.Duration(processUptimeSeconds * float64(time.Second)))
}
