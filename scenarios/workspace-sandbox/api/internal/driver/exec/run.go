package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// ExecResult contains the result of executing a command in the sandbox.
type ExecResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	PID      int
	Error    error

	// Backend is the id of the containment backend that actually ran this
	// command ("bwrap" when contained, "none" for the direct path). Stamped
	// from the buildStartOpts dispatch so per-launch provenance is truth, not
	// inference.
	Backend string
}

// Exec runs cmd inside the sandbox synchronously and returns the result.
// Wall-clock timeout from cfg.ResourceLimits.TimeoutSec is enforced via
// context. prlimit wrapping applies only when the chosen containment
// backend is bwrap.
//
// All process invocation routes through process.Starter (Round 4 Phase
// 7). The starter is required; passing nil panics with a structured
// message to fail loud at wiring.
func Exec(ctx context.Context, starter process.Starter, s *types.Sandbox, level driver.ContainmentLevel, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	if starter == nil {
		panic("exec.Exec: starter is required")
	}
	if s.MergedDir == "" {
		return nil, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if cfg.ResourceLimits.TimeoutSec > 0 {
		timeout := time.Duration(cfg.ResourceLimits.TimeoutSec) * time.Second
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	opts, backendID, err := buildStartOpts(starter, s, level, cfg, cmd, args...)
	if err != nil {
		return nil, err
	}

	for k, v := range cfg.Env {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if level == driver.ContainmentNone {
		// Direct execution inherits parent env so the agent sees PATH etc.
		// Override entries from cfg.Env take precedence.
		base := os.Environ()
		opts.Env = append(base, opts.Env...)
	}

	res, runErr := process.Run(execCtx, starter, opts)

	result := &ExecResult{
		PID:      res.PID,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: res.Exit.ExitCode,
		Backend:  backendID,
	}

	if runErr != nil {
		// Timeout takes precedence: the runtime kills the process via
		// signal, which surfaces as ExitError, but we want to surface 124
		// (the standard timeout exit code) so callers can distinguish.
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			result.ExitCode = 124
			result.Error = fmt.Errorf("process timed out after %d seconds", cfg.ResourceLimits.TimeoutSec)
		} else {
			if result.ExitCode == 0 {
				result.ExitCode = -1
			}
			result.Error = runErr
		}
		return result, nil
	}
	if res.Exit.ExitCode != 0 && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		result.ExitCode = 124
		result.Error = fmt.Errorf("process timed out after %d seconds", cfg.ResourceLimits.TimeoutSec)
	}
	return result, nil
}

// StartProcess launches cmd asynchronously and returns its PID plus the id
// of the containment backend that actually ran it ("bwrap" or "none").
// cfg.OnExit fires exactly once after the spawned process exits.
// Stdout/Stderr/Stdin are wired from cfg.
func StartProcess(ctx context.Context, starter process.Starter, s *types.Sandbox, level driver.ContainmentLevel, cfg BwrapConfig, cmd string, args ...string) (int, string, error) {
	if starter == nil {
		panic("exec.StartProcess: starter is required")
	}
	if s.MergedDir == "" {
		return 0, "", fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	// Background processes must not inherit a wall-clock timeout.
	ctxNoTimeout := context.Background()
	_ = ctx // request context governs the calling handler; the spawned process is detached

	opts, backendID, err := buildStartOpts(starter, s, level, cfg, cmd, args...)
	if err != nil {
		return 0, "", err
	}

	for k, v := range cfg.Env {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if level == driver.ContainmentNone {
		base := os.Environ()
		opts.Env = append(base, opts.Env...)
	}

	opts.SysProcAttr = process.NewProcessGroupSysProcAttr()
	wireStartIO(&opts, cfg)

	handle, err := starter.Start(ctxNoTimeout, opts)
	if err != nil {
		return 0, "", fmt.Errorf("failed to start process: %w", err)
	}

	spawnExitReaper(handle, cfg.OnExit)
	return handle.PID(), backendID, nil
}

// wireStartIO wires StartOpts.Stdout / .Stderr / .Stdin from cfg.
// Background processes always need a writer for each stream so output
// is captured; pass io.Discard if the caller does not want to retain it.
func wireStartIO(opts *process.StartOpts, cfg BwrapConfig) {
	if cfg.StdoutWriter != nil {
		opts.Stdout = cfg.StdoutWriter
	} else {
		opts.Stdout = io.Discard
	}
	if cfg.StderrWriter != nil {
		opts.Stderr = cfg.StderrWriter
	} else {
		opts.Stderr = io.Discard
	}
	if cfg.StdinReader != nil {
		opts.Stdin = cfg.StdinReader
	}
}

// spawnExitReaper waits for the process in a goroutine and dispatches
// to onExit (when non-nil) exactly once. Wait is always invoked to
// prevent zombies even when onExit is nil.
func spawnExitReaper(handle process.Handle, onExit func(int, int, bool)) {
	if onExit == nil {
		go func() { _, _ = handle.Wait(context.Background()) }()
		return
	}
	go func() {
		exit, _ := handle.Wait(context.Background())
		onExit(exit.ExitCode, exit.Signal, exit.OOMKilled)
	}()
}

// ExitInfoFromState extracts (exitCode, signal, oomKilled) from a
// *os.ProcessState plus the wait error. Exported so tests share the
// canonical extraction logic. Delegates to process.ExitFromState so the
// translation lives in exactly one place.
func ExitInfoFromState(state *os.ProcessState, waitErr error) (exitCode, signal int, oomKilled bool) {
	exit := process.ExitFromState(state, waitErr)
	return exit.ExitCode, exit.Signal, exit.OOMKilled
}

// KillProcessGroup kills a process and all its children by process group ID.
// Delegates to process.KillProcessGroupByPID so the syscall path stays
// in the canonical seam.
func KillProcessGroup(pid int) error {
	return process.KillProcessGroupByPID(pid)
}

// IsProcessRunning checks if a process with the given PID is still running.
func IsProcessRunning(pid int) bool {
	return process.IsProcessRunning(pid)
}
