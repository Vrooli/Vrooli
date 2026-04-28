// Package process provides process tracking and cleanup for sandboxes.
//
// This package implements OT-P0-008 (Process/Session Tracking), enabling:
//   - Tracking of all PIDs spawned within a sandbox context
//   - Clean termination of tracked processes on sandbox stop/delete
//   - Prevention of orphaned processes
//   - Structured exit-info recording (exit code, terminating signal,
//     OOM-killed flag) so callers can distinguish runner-process success
//     from failure precisely.
//   - Per-process stdin pipe handles so the /processes/{pid}/stdin endpoint
//     can stream input into running processes.
//
// Design Notes:
//   - Process tracking is best-effort, not a security boundary
//   - Uses process groups for reliable cleanup of child processes
//   - Stores tracked PIDs in the sandbox metadata (not separate table)
//   - Supports multiple sessions (multiple exec calls per sandbox)
//   - Exit info is set exactly once via RecordExit; subsequent calls no-op.
package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// ExitInfo captures the terminal state of a tracked process. It is the
// canonical contract between the driver's wait reaper and the rest of the
// system (handlers, SSE consumers, sandbox launcher).
//
// Signal is non-zero only when the process was killed by a signal.
// OOMKilled is set by the wait reaper when the kernel reported the process
// was killed by the OOM killer.
type ExitInfo struct {
	ExitCode  int       `json:"exitCode"`
	Signal    int       `json:"signal,omitempty"`
	OOMKilled bool      `json:"oomKilled,omitempty"`
	StoppedAt time.Time `json:"stoppedAt"`
}

// TrackedProcess represents a process being tracked.
type TrackedProcess struct {
	// PID is the process ID.
	PID int `json:"pid"`

	// PGID is the process group ID (for killing children).
	PGID int `json:"pgid"`

	// SandboxID is the owning sandbox.
	SandboxID uuid.UUID `json:"sandboxId"`

	// Command is the command that was executed.
	Command string `json:"command"`

	// StartedAt is when the process was started.
	StartedAt time.Time `json:"startedAt"`

	// StoppedAt is when the process was stopped (if applicable).
	StoppedAt *time.Time `json:"stoppedAt,omitempty"`

	// ExitCode is the process exit code (if exited). Mirrors ExitInfo.ExitCode
	// for backwards-compatible JSON consumers; SignalNum and OOMKilled carry
	// the same precision as ExitInfo.
	ExitCode  *int  `json:"exitCode,omitempty"`
	SignalNum *int  `json:"signal,omitempty"`
	OOMKilled *bool `json:"oomKilled,omitempty"`

	// SessionID groups related processes.
	SessionID string `json:"sessionId,omitempty"`

	// stdin is the writable end of a pipe wired to the process's stdin
	// (when the process was started with WithStdin=true). Closed once on
	// EOF. Not exported in JSON.
	stdin io.WriteCloser

	// exitCh is closed when ExitInfo has been recorded for this process.
	// Subscribers (SSE consumers, Wait callers) <-chan to know when the
	// process has terminated. Created lazily on first Track().
	exitCh chan struct{}

	// exitMu guards stdin closure idempotency.
	stdinMu sync.Mutex
}

// IsRunning checks if the process is still running.
func (p *TrackedProcess) IsRunning() bool {
	process, err := os.FindProcess(p.PID)
	if err != nil {
		return false
	}
	// On Unix, Signal(0) checks if process exists
	return process.Signal(syscall.Signal(0)) == nil
}

// Status returns "running" while the wait reaper has not delivered an
// ExitInfo, otherwise "exited".
func (p *TrackedProcess) Status() string {
	if p.ExitCode != nil {
		return "exited"
	}
	return "running"
}

// ExitInfo returns the structured exit info if the process has terminated,
// or nil if it is still running.
func (p *TrackedProcess) ExitInfoOrNil() *ExitInfo {
	if p.ExitCode == nil {
		return nil
	}
	info := &ExitInfo{ExitCode: *p.ExitCode}
	if p.SignalNum != nil {
		info.Signal = *p.SignalNum
	}
	if p.OOMKilled != nil {
		info.OOMKilled = *p.OOMKilled
	}
	if p.StoppedAt != nil {
		info.StoppedAt = *p.StoppedAt
	}
	return info
}

// TrackerConfig holds configuration for process tracking.
type TrackerConfig struct {
	// GracePeriod is how long to wait after SIGTERM before sending SIGKILL.
	// Default: 100ms
	GracePeriod time.Duration

	// KillWait is how long to wait after SIGKILL for process to die.
	// Default: 50ms
	KillWait time.Duration
}

// DefaultTrackerConfig returns sensible defaults.
func DefaultTrackerConfig() TrackerConfig {
	return TrackerConfig{
		GracePeriod: 100 * time.Millisecond,
		KillWait:    50 * time.Millisecond,
	}
}

// Tracker manages process tracking for sandboxes.
type Tracker struct {
	mu        sync.RWMutex
	processes map[uuid.UUID][]*TrackedProcess // sandboxID -> processes
	config    TrackerConfig
}

// NewTracker creates a new process tracker with default config.
func NewTracker() *Tracker {
	return &Tracker{
		processes: make(map[uuid.UUID][]*TrackedProcess),
		config:    DefaultTrackerConfig(),
	}
}

// NewTrackerWithConfig creates a new process tracker with custom config.
func NewTrackerWithConfig(cfg TrackerConfig) *Tracker {
	if cfg.GracePeriod <= 0 {
		cfg.GracePeriod = 100 * time.Millisecond
	}
	if cfg.KillWait <= 0 {
		cfg.KillWait = 50 * time.Millisecond
	}
	return &Tracker{
		processes: make(map[uuid.UUID][]*TrackedProcess),
		config:    cfg,
	}
}

// Track adds a process to tracking for a sandbox.
func (t *Tracker) Track(sandboxID uuid.UUID, pid int, command string, sessionID string) (*TrackedProcess, error) {
	// Get process group ID
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Use PID as PGID fallback
		pgid = pid
	}

	proc := &TrackedProcess{
		PID:       pid,
		PGID:      pgid,
		SandboxID: sandboxID,
		Command:   command,
		StartedAt: time.Now(),
		SessionID: sessionID,
		exitCh:    make(chan struct{}),
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.processes[sandboxID] = append(t.processes[sandboxID], proc)
	return proc, nil
}

// SetStdin attaches a stdin writer to the tracked process. The writer is
// closed by CloseStdin or by Cleanup. Stdin can only be set once.
func (t *Tracker) SetStdin(sandboxID uuid.UUID, pid int, stdin io.WriteCloser) error {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}
	target.stdinMu.Lock()
	defer target.stdinMu.Unlock()
	if target.stdin != nil {
		return fmt.Errorf("process %d already has stdin attached", pid)
	}
	target.stdin = stdin
	return nil
}

// WriteStdin writes the given bytes to the process's stdin pipe. Returns
// an error if the process has no stdin attached.
func (t *Tracker) WriteStdin(sandboxID uuid.UUID, pid int, p []byte) (int, error) {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return 0, fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}
	target.stdinMu.Lock()
	stdin := target.stdin
	target.stdinMu.Unlock()
	if stdin == nil {
		return 0, fmt.Errorf("process %d has no stdin pipe (not started with WithStdin)", pid)
	}
	return stdin.Write(p)
}

// CloseStdin closes the process's stdin pipe (signaling EOF to the process).
// Idempotent: subsequent calls return nil.
func (t *Tracker) CloseStdin(sandboxID uuid.UUID, pid int) error {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}
	target.stdinMu.Lock()
	defer target.stdinMu.Unlock()
	if target.stdin == nil {
		return nil
	}
	err := target.stdin.Close()
	target.stdin = nil
	return err
}

// RecordExit stores ExitInfo on the tracked process and closes its exit
// channel. Idempotent: subsequent calls no-op.
//
// Called by the driver's wait reaper goroutine when cmd.Wait() returns.
// Closes the stdin pipe (if any) so the process doesn't block on stdin
// during teardown.
func (t *Tracker) RecordExit(sandboxID uuid.UUID, pid int, info ExitInfo) {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return
	}

	// Idempotency: if exit already recorded, no-op.
	t.mu.Lock()
	if target.ExitCode != nil {
		t.mu.Unlock()
		return
	}
	stoppedAt := info.StoppedAt
	if stoppedAt.IsZero() {
		stoppedAt = time.Now()
	}
	exitCode := info.ExitCode
	target.ExitCode = &exitCode
	if info.Signal != 0 {
		sig := info.Signal
		target.SignalNum = &sig
	}
	if info.OOMKilled {
		oom := true
		target.OOMKilled = &oom
	}
	target.StoppedAt = &stoppedAt
	exitCh := target.exitCh
	t.mu.Unlock()

	// Close stdin so any blocked-on-stdin process unblocks for teardown.
	target.stdinMu.Lock()
	if target.stdin != nil {
		_ = target.stdin.Close()
		target.stdin = nil
	}
	target.stdinMu.Unlock()

	// Notify subscribers.
	if exitCh != nil {
		// Guard against double-close in pathological cases.
		defer func() { _ = recover() }()
		close(exitCh)
	}
}

// ExitChannel returns a channel that closes when the process has exited
// and ExitInfo has been recorded. Returns a closed channel if the process
// has already exited or is not tracked.
func (t *Tracker) ExitChannel(sandboxID uuid.UUID, pid int) <-chan struct{} {
	closed := make(chan struct{})
	close(closed)

	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return closed
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if target.ExitCode != nil {
		return closed
	}
	return target.exitCh
}

// GetExitInfo returns the structured ExitInfo for a process, or nil if it
// has not yet exited.
func (t *Tracker) GetExitInfo(sandboxID uuid.UUID, pid int) *ExitInfo {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return target.ExitInfoOrNil()
}

// WaitForExit blocks until the process has exited and ExitInfo has been
// recorded, or until ctx is cancelled. Returns:
//   - (*ExitInfo, nil) once RecordExit has run for this process.
//   - (nil, ctx.Err()) if ctx fires before exit info is recorded — typical
//     when the caller uses a bounded timeout to avoid hanging an SSE
//     stream on a wait reaper that never returned.
//   - (nil, error) if the process is not tracked.
//
// Used by StreamProcessLogs to deterministically deliver `event: exit`
// frames even when the process exits before the SSE subscriber attached
// (the prior best-effort GetExitInfo lookup raced spawnExitReaper for
// fast-failing processes).
//
// DOC: see scenarios/workspace-sandbox/docs/internal/SEAMS.md #ProcessTracker
func (t *Tracker) WaitForExit(ctx context.Context, sandboxID uuid.UUID, pid int) (*ExitInfo, error) {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return nil, fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}

	// Already recorded? Return immediately.
	t.mu.RLock()
	if target.ExitCode != nil {
		info := target.ExitInfoOrNil()
		t.mu.RUnlock()
		return info, nil
	}
	exitCh := target.exitCh
	t.mu.RUnlock()

	select {
	case <-exitCh:
		t.mu.RLock()
		info := target.ExitInfoOrNil()
		t.mu.RUnlock()
		return info, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// findProcess returns the tracked process for (sandboxID, pid) or nil.
func (t *Tracker) findProcess(sandboxID uuid.UUID, pid int) *TrackedProcess {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, proc := range t.processes[sandboxID] {
		if proc.PID == pid {
			return proc
		}
	}
	return nil
}

// GetProcesses returns all tracked processes for a sandbox.
func (t *Tracker) GetProcesses(sandboxID uuid.UUID) []*TrackedProcess {
	t.mu.RLock()
	defer t.mu.RUnlock()

	procs := t.processes[sandboxID]
	result := make([]*TrackedProcess, len(procs))
	copy(result, procs)
	return result
}

// GetRunningProcesses returns only running processes for a sandbox.
func (t *Tracker) GetRunningProcesses(sandboxID uuid.UUID) []*TrackedProcess {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*TrackedProcess
	for _, proc := range t.processes[sandboxID] {
		if proc.IsRunning() {
			result = append(result, proc)
		}
	}
	return result
}

// GetActiveCount returns the count of running processes for a sandbox.
func (t *Tracker) GetActiveCount(sandboxID uuid.UUID) int {
	return len(t.GetRunningProcesses(sandboxID))
}

// KillAll terminates all tracked processes for a sandbox.
// Returns the count of processes killed and any errors encountered.
func (t *Tracker) KillAll(ctx context.Context, sandboxID uuid.UUID) (int, []error) {
	t.mu.Lock()
	procs := t.processes[sandboxID]
	t.mu.Unlock()

	var killed int
	var errors []error

	for _, proc := range procs {
		if !proc.IsRunning() {
			continue
		}

		// Try graceful termination first (SIGTERM)
		if err := t.killProcess(proc, syscall.SIGTERM); err != nil {
			errors = append(errors, err)
		}

		// Wait for graceful shutdown (configurable)
		time.Sleep(t.config.GracePeriod)

		// If still running, force kill (SIGKILL)
		if proc.IsRunning() {
			if err := t.killProcess(proc, syscall.SIGKILL); err != nil {
				errors = append(errors, err)
				// Still try direct PID kill as last resort
				if killErr := syscall.Kill(proc.PID, syscall.SIGKILL); killErr != nil {
					errors = append(errors, killErr)
				}
			}
		}

		// Give time for the process to die (configurable)
		time.Sleep(t.config.KillWait)

		// Check if actually dead now
		if !proc.IsRunning() {
			// Wait reaper will record real exit info; if for some reason
			// it hasn't (e.g., process not started by us), record a
			// best-effort placeholder so callers are unblocked.
			if t.GetExitInfo(sandboxID, proc.PID) == nil {
				t.RecordExit(sandboxID, proc.PID, ExitInfo{
					ExitCode:  -1,
					Signal:    int(syscall.SIGKILL),
					StoppedAt: time.Now(),
				})
			}
			killed++
		} else {
			errors = append(errors, fmt.Errorf("failed to kill PID %d", proc.PID))
		}
	}

	return killed, errors
}

// killProcess sends a signal to a process and its group.
func (t *Tracker) killProcess(proc *TrackedProcess, sig syscall.Signal) error {
	// Try to kill the entire process group first
	if proc.PGID != 0 {
		err := syscall.Kill(-proc.PGID, sig)
		if err == nil {
			return nil
		}
	}

	// Fallback to killing just the process
	return syscall.Kill(proc.PID, sig)
}

// KillProcess terminates a specific process.
func (t *Tracker) KillProcess(ctx context.Context, sandboxID uuid.UUID, pid int) error {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}

	if !target.IsRunning() {
		return nil // Already dead
	}

	// SIGTERM first (ignore errors, try anyway)
	var errors []error
	if err := t.killProcess(target, syscall.SIGTERM); err != nil {
		errors = append(errors, err)
	}

	// Wait for graceful shutdown (configurable)
	time.Sleep(t.config.GracePeriod)
	if target.IsRunning() {
		if err := t.killProcess(target, syscall.SIGKILL); err != nil {
			errors = append(errors, err)
		}
		// Also try direct PID kill as last resort
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			errors = append(errors, err)
		}
	}

	// Give time for cleanup (configurable)
	time.Sleep(t.config.KillWait)

	if !target.IsRunning() {
		return nil
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to kill PID %d: %v", pid, errors)
	}
	return fmt.Errorf("failed to kill PID %d", pid)
}

// Cleanup removes tracking data for a sandbox.
// Should be called after KillAll when the sandbox is being deleted.
func (t *Tracker) Cleanup(sandboxID uuid.UUID) {
	t.mu.Lock()
	procs := t.processes[sandboxID]
	delete(t.processes, sandboxID)
	t.mu.Unlock()

	// Close any stdin pipes still attached.
	for _, proc := range procs {
		proc.stdinMu.Lock()
		if proc.stdin != nil {
			_ = proc.stdin.Close()
			proc.stdin = nil
		}
		proc.stdinMu.Unlock()
	}
}

// GetAllStats returns aggregate statistics across all sandboxes.
func (t *Tracker) GetAllStats() ProcessStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := ProcessStats{
		SandboxCounts: make(map[uuid.UUID]int),
	}

	for sandboxID, procs := range t.processes {
		for _, proc := range procs {
			stats.TotalTracked++
			if proc.IsRunning() {
				stats.TotalRunning++
				stats.SandboxCounts[sandboxID]++
			}
		}
	}

	stats.SandboxesWithProcesses = len(stats.SandboxCounts)
	return stats
}

// ProcessStats contains aggregate process statistics.
type ProcessStats struct {
	TotalTracked           int               `json:"totalTracked"`
	TotalRunning           int               `json:"totalRunning"`
	SandboxesWithProcesses int               `json:"sandboxesWithProcesses"`
	SandboxCounts          map[uuid.UUID]int `json:"sandboxCounts,omitempty"`
}

// Session represents a group of related processes within a sandbox.
type Session struct {
	ID        string            `json:"id"`
	SandboxID uuid.UUID         `json:"sandboxId"`
	StartedAt time.Time         `json:"startedAt"`
	EndedAt   *time.Time        `json:"endedAt,omitempty"`
	Processes []*TrackedProcess `json:"processes"`
}

// StartSession creates a new session for tracking related processes.
func (t *Tracker) StartSession(sandboxID uuid.UUID) *Session {
	return &Session{
		ID:        uuid.New().String(),
		SandboxID: sandboxID,
		StartedAt: time.Now(),
		Processes: []*TrackedProcess{},
	}
}

// TrackInSession adds a process to both the tracker and a session.
func (t *Tracker) TrackInSession(session *Session, pid int, command string) (*TrackedProcess, error) {
	proc, err := t.Track(session.SandboxID, pid, command, session.ID)
	if err != nil {
		return nil, err
	}
	session.Processes = append(session.Processes, proc)
	return proc, nil
}

// EndSession marks a session as ended and optionally kills its processes.
func (t *Tracker) EndSession(ctx context.Context, session *Session, killProcesses bool) error {
	now := time.Now()
	session.EndedAt = &now

	if killProcesses {
		var firstErr error
		for _, proc := range session.Processes {
			if proc.IsRunning() {
				if err := t.KillProcess(ctx, session.SandboxID, proc.PID); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		if firstErr != nil {
			return firstErr
		}
	}

	return nil
}

// WaitForProcess waits for a process to exit (driven by the driver's wait
// reaper recording ExitInfo via RecordExit) and returns the tracked process.
//
// Returns context.DeadlineExceeded or the timeout error if the process does
// not exit before the deadline.
func (t *Tracker) WaitForProcess(ctx context.Context, sandboxID uuid.UUID, pid int, timeout time.Duration) (*TrackedProcess, error) {
	target := t.findProcess(sandboxID, pid)
	if target == nil {
		return nil, fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}

	// Already exited?
	t.mu.RLock()
	if target.ExitCode != nil {
		t.mu.RUnlock()
		return target, nil
	}
	exitCh := target.exitCh
	t.mu.RUnlock()

	select {
	case <-exitCh:
		return target, nil
	case <-ctx.Done():
		return target, ctx.Err()
	case <-time.After(timeout):
		return target, fmt.Errorf("timeout waiting for process %d", pid)
	}
}
