// Package process provides process tracking and cleanup for sandboxes.
//
// This package implements OT-P0-008 (Process/Session Tracking), enabling:
//   - Tracking of all PIDs spawned within a sandbox context
//   - Clean termination of tracked processes on sandbox stop/delete
//   - Prevention of orphaned processes
//
// Design Notes:
//   - Process tracking is best-effort, not a security boundary
//   - Uses process groups for reliable cleanup of child processes
//   - Stores tracked PIDs in the sandbox metadata (not separate table)
//   - Supports multiple sessions (multiple exec calls per sandbox)
package process

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

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

	// ExitCode is the process exit code (if exited).
	ExitCode *int `json:"exitCode,omitempty"`

	// SessionID groups related processes.
	SessionID string `json:"sessionId,omitempty"`
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
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.processes[sandboxID] = append(t.processes[sandboxID], proc)
	return proc, nil
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
		t.killProcess(proc, syscall.SIGTERM)

		// Wait for graceful shutdown (configurable)
		time.Sleep(t.config.GracePeriod)

		// If still running, force kill (SIGKILL)
		if proc.IsRunning() {
			if err := t.killProcess(proc, syscall.SIGKILL); err != nil {
				// Still try direct PID kill as last resort
				syscall.Kill(proc.PID, syscall.SIGKILL)
			}
		}

		// Give time for the process to die (configurable)
		time.Sleep(t.config.KillWait)

		// Check if actually dead now
		if !proc.IsRunning() {
			now := time.Now()
			proc.StoppedAt = &now
			exitCode := -1
			proc.ExitCode = &exitCode
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
	t.mu.RLock()
	var target *TrackedProcess
	for _, proc := range t.processes[sandboxID] {
		if proc.PID == pid {
			target = proc
			break
		}
	}
	t.mu.RUnlock()

	if target == nil {
		return fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}

	if !target.IsRunning() {
		return nil // Already dead
	}

	// SIGTERM first (ignore errors, try anyway)
	t.killProcess(target, syscall.SIGTERM)

	// Wait for graceful shutdown (configurable)
	time.Sleep(t.config.GracePeriod)
	if target.IsRunning() {
		t.killProcess(target, syscall.SIGKILL)
		// Also try direct PID kill as last resort
		syscall.Kill(pid, syscall.SIGKILL)
	}

	// Give time for cleanup (configurable)
	time.Sleep(t.config.KillWait)

	if !target.IsRunning() {
		now := time.Now()
		target.StoppedAt = &now
	}
	return nil
}

// Cleanup removes tracking data for a sandbox.
// Should be called after KillAll when the sandbox is being deleted.
func (t *Tracker) Cleanup(sandboxID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.processes, sandboxID)
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
		for _, proc := range session.Processes {
			if proc.IsRunning() {
				t.KillProcess(ctx, session.SandboxID, proc.PID)
			}
		}
	}

	return nil
}

// WaitForProcess waits for a process to exit and updates its tracked state.
// Note: This uses polling since we can't call Wait() on processes we didn't spawn.
func (t *Tracker) WaitForProcess(ctx context.Context, sandboxID uuid.UUID, pid int, timeout time.Duration) (*TrackedProcess, error) {
	t.mu.RLock()
	var target *TrackedProcess
	for _, proc := range t.processes[sandboxID] {
		if proc.PID == pid {
			target = proc
			break
		}
	}
	t.mu.RUnlock()

	if target == nil {
		return nil, fmt.Errorf("process %d not found in sandbox %s", pid, sandboxID)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !target.IsRunning() {
			now := time.Now()
			target.StoppedAt = &now
			// We can't get the actual exit code without calling Wait() on a process we spawned
			// Set a placeholder value
			exitCode := 0
			target.ExitCode = &exitCode
			return target, nil
		}

		select {
		case <-ctx.Done():
			return target, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue polling
		}
	}

	return target, fmt.Errorf("timeout waiting for process %d", pid)
}
