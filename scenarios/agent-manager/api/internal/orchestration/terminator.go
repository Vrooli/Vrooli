// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the Terminator which handles robust process termination
// with retry logic, process group killing, and graceful degradation.
//
// TERMINATION STRATEGY:
//   1. SIGTERM to main process (graceful)
//   2. Wait for grace period
//   3. SIGKILL to main process (force)
//   4. SIGKILL to process group (nuclear)
//   5. Verify termination
//
// RETRY LOGIC:
//   - Exponential backoff between attempts
//   - Configurable max retries
//   - Different strategies for different failure modes

package orchestration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// TerminatorConfig holds configuration for the termination service.
type TerminatorConfig struct {
	// GracePeriod is how long to wait after SIGTERM before SIGKILL
	GracePeriod time.Duration

	// MaxRetries is the maximum number of termination attempts
	MaxRetries int

	// BaseBackoff is the initial backoff duration between retries
	BaseBackoff time.Duration

	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration

	// VerifyTimeout is how long to wait when verifying termination
	VerifyTimeout time.Duration

	// KillProcessGroup determines whether to kill the entire process group
	KillProcessGroup bool
}

// DefaultTerminatorConfig returns sensible defaults.
func DefaultTerminatorConfig() TerminatorConfig {
	return TerminatorConfig{
		GracePeriod:      5 * time.Second,
		MaxRetries:       3,
		BaseBackoff:      500 * time.Millisecond,
		MaxBackoff:       5 * time.Second,
		VerifyTimeout:    2 * time.Second,
		KillProcessGroup: true,
	}
}

// Terminator handles robust process termination with retry logic.
type Terminator struct {
	runs    repository.RunRepository
	runners runner.Registry
	config  TerminatorConfig
}

// NewTerminator creates a new terminator with the given dependencies.
func NewTerminator(
	runs repository.RunRepository,
	runners runner.Registry,
	config TerminatorConfig,
) *Terminator {
	return &Terminator{
		runs:    runs,
		runners: runners,
		config:  config,
	}
}

// TerminateResult contains the outcome of a termination attempt.
type TerminateResult struct {
	Success        bool
	Attempts       int
	FinalMethod    string // "sigterm", "sigkill", "pgkill", "cli"
	Duration       time.Duration
	Error          error
	ProcessWasGone bool
}

// Terminate attempts to stop a run with full retry and escalation logic.
func (t *Terminator) Terminate(ctx context.Context, runID uuid.UUID) (*TerminateResult, error) {
	start := time.Now()
	result := &TerminateResult{}

	// Get the run
	run, err := t.runs.Get(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", runID)
	}

	// Check if run is in a stoppable state
	if run.Status != domain.RunStatusRunning && run.Status != domain.RunStatusStarting {
		return nil, domain.NewStateError("Run", string(run.Status), "stop", "can only stop running or starting runs")
	}

	tag := run.GetTag()

	// Try termination with retries
	for attempt := 1; attempt <= t.config.MaxRetries; attempt++ {
		result.Attempts = attempt

		// Method 1: Try via runner interface (cleanest)
		if t.runners != nil && run.ResolvedConfig != nil {
			if r, err := t.runners.Get(run.ResolvedConfig.RunnerType); err == nil {
				if err := r.Stop(ctx, runID); err == nil {
					// Verify it's actually stopped
					if t.verifyTerminated(tag) {
						result.Success = true
						result.FinalMethod = "runner"
						break
					}
				}
			}
		}

		// Method 2: CLI stop command
		if t.tryCliStop(ctx, run, tag) {
			if t.verifyTerminated(tag) {
				result.Success = true
				result.FinalMethod = "cli"
				break
			}
		}

		// Method 3: Find PID and signal directly
		pid := t.findProcessPID(tag)
		if pid == 0 {
			// Process not found - might have already terminated
			result.Success = true
			result.ProcessWasGone = true
			result.FinalMethod = "not_found"
			break
		}

		// Method 3a: SIGTERM (graceful)
		if t.trySIGTERM(pid) {
			time.Sleep(t.config.GracePeriod)
			if t.verifyTerminated(tag) {
				result.Success = true
				result.FinalMethod = "sigterm"
				break
			}
		}

		// Method 3b: SIGKILL (force)
		if t.trySIGKILL(pid) {
			time.Sleep(500 * time.Millisecond)
			if t.verifyTerminated(tag) {
				result.Success = true
				result.FinalMethod = "sigkill"
				break
			}
		}

		// Method 3c: Process group kill (nuclear)
		if t.config.KillProcessGroup {
			pgid := t.getProcessGroupID(pid)
			if pgid > 0 && t.tryKillProcessGroup(pgid) {
				time.Sleep(500 * time.Millisecond)
				if t.verifyTerminated(tag) {
					result.Success = true
					result.FinalMethod = "pgkill"
					break
				}
			}
		}

		// Backoff before retry
		if attempt < t.config.MaxRetries {
			backoff := t.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				break
			case <-time.After(backoff):
			}
		}
	}

	result.Duration = time.Since(start)

	// Update run status if successful
	if result.Success {
		now := time.Now()
		run.Status = domain.RunStatusCancelled
		run.EndedAt = &now
		run.UpdatedAt = now
		if err := t.runs.Update(ctx, run); err != nil {
			// Log but don't fail - the process is dead
			result.Error = fmt.Errorf("process terminated but failed to update DB: %w", err)
		}
	} else {
		result.Error = fmt.Errorf("failed to terminate run after %d attempts", result.Attempts)
	}

	return result, nil
}

// TerminateByTag attempts to stop a process by its tag (for orphan cleanup).
func (t *Terminator) TerminateByTag(ctx context.Context, tag string) (*TerminateResult, error) {
	start := time.Now()
	result := &TerminateResult{}

	for attempt := 1; attempt <= t.config.MaxRetries; attempt++ {
		result.Attempts = attempt

		pid := t.findProcessPID(tag)
		if pid == 0 {
			result.Success = true
			result.ProcessWasGone = true
			result.FinalMethod = "not_found"
			break
		}

		// SIGTERM
		if t.trySIGTERM(pid) {
			time.Sleep(t.config.GracePeriod)
			if t.verifyTerminated(tag) {
				result.Success = true
				result.FinalMethod = "sigterm"
				break
			}
		}

		// SIGKILL
		if t.trySIGKILL(pid) {
			time.Sleep(500 * time.Millisecond)
			if t.verifyTerminated(tag) {
				result.Success = true
				result.FinalMethod = "sigkill"
				break
			}
		}

		// Process group kill
		if t.config.KillProcessGroup {
			pgid := t.getProcessGroupID(pid)
			if pgid > 0 && t.tryKillProcessGroup(pgid) {
				time.Sleep(500 * time.Millisecond)
				if t.verifyTerminated(tag) {
					result.Success = true
					result.FinalMethod = "pgkill"
					break
				}
			}
		}

		// Backoff
		if attempt < t.config.MaxRetries {
			backoff := t.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				break
			case <-time.After(backoff):
			}
		}
	}

	result.Duration = time.Since(start)
	if !result.Success {
		result.Error = fmt.Errorf("failed to terminate process with tag %s after %d attempts", tag, result.Attempts)
	}

	return result, nil
}

// tryCliStop attempts to stop via the resource CLI.
func (t *Terminator) tryCliStop(ctx context.Context, run *domain.Run, tag string) bool {
	if run.ResolvedConfig == nil {
		return false
	}

	var cliCmd string
	switch run.ResolvedConfig.RunnerType {
	case domain.RunnerTypeClaudeCode:
		cliCmd = "resource-claude-code"
	case domain.RunnerTypeCodex:
		cliCmd = "resource-codex"
	case domain.RunnerTypeOpenCode:
		cliCmd = "resource-opencode"
	default:
		return false
	}

	// Try: resource-X agents stop <tag>
	cmd := exec.CommandContext(ctx, cliCmd, "agents", "stop", tag)
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// findProcessPID finds the PID of a process by its tag.
func (t *Terminator) findProcessPID(tag string) int {
	cmd := exec.Command("pgrep", "-f", tag)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Get first PID if multiple
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return 0
	}
	return pid
}

// trySIGTERM sends SIGTERM to a process.
func (t *Terminator) trySIGTERM(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.SIGTERM) == nil
}

// trySIGKILL sends SIGKILL to a process.
func (t *Terminator) trySIGKILL(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Kill() == nil
}

// getProcessGroupID gets the process group ID for a PID.
func (t *Terminator) getProcessGroupID(pid int) int {
	// Read from /proc/[pid]/stat
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0
	}

	// PGID is field 5 (0-indexed: 4)
	fields := strings.Fields(string(data))
	if len(fields) < 5 {
		return 0
	}

	pgid, err := strconv.Atoi(fields[4])
	if err != nil {
		return 0
	}
	return pgid
}

// tryKillProcessGroup kills all processes in a process group.
func (t *Terminator) tryKillProcessGroup(pgid int) bool {
	// Use negative PID to signal the entire process group
	return syscall.Kill(-pgid, syscall.SIGKILL) == nil
}

// verifyTerminated checks if a process is truly dead.
func (t *Terminator) verifyTerminated(tag string) bool {
	// Wait a moment for process table to update
	time.Sleep(100 * time.Millisecond)

	// Check if any process with this tag exists
	cmd := exec.Command("pgrep", "-f", tag)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output)) == ""
}

// calculateBackoff calculates the backoff duration for a retry attempt.
func (t *Terminator) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * 2^attempt
	backoff := t.config.BaseBackoff * time.Duration(1<<uint(attempt-1))
	if backoff > t.config.MaxBackoff {
		backoff = t.config.MaxBackoff
	}
	return backoff
}

// =============================================================================
// ENHANCED STOP RUN FOR SERVICE
// =============================================================================

// StopRunWithRetry is an enhanced StopRun implementation that uses the terminator.
// This should replace the simple StopRun in service.go
func (t *Terminator) StopRunWithRetry(ctx context.Context, runID uuid.UUID) error {
	result, err := t.Terminate(ctx, runID)
	if err != nil {
		return err
	}

	if !result.Success {
		return result.Error
	}

	return nil
}
