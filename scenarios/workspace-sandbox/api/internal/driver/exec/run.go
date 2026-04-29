package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"strings"
	"syscall"
	"time"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// IsolationMode aliases driver.IsolationMode for convenience inside the
// exec package. Each driver declares its required mode via
// MountDriver.RequiresBwrap.
type IsolationMode = driver.IsolationMode

// Re-exported constants so exec callers don't have to import the driver
// package just to spell ModeNone / ModeBwrap*.
const (
	ModeNone           = driver.ModeNone
	ModeBwrapPreferred = driver.ModeBwrapPreferred
	ModeBwrapRequired  = driver.ModeBwrapRequired
)

// ExecResult contains the result of executing a command in the sandbox.
type ExecResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	PID      int
	Error    error
}

// Exec runs cmd inside the sandbox synchronously and returns the result.
// Wall-clock timeout from cfg.ResourceLimits.TimeoutSec is enforced via
// context. prlimit wrapping applies only when isolation mode uses bwrap.
func Exec(ctx context.Context, s *types.Sandbox, mode IsolationMode, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
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

	execCmd, err := buildCmd(execCtx, s, mode, cfg, cmd, args...)
	if err != nil {
		return nil, err
	}

	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if mode == ModeNone {
		// Direct execution inherits parent env so the agent sees PATH etc.
		// Override entries from cfg.Env take precedence.
		base := os.Environ()
		execCmd.Env = append(base, execCmd.Env...)
	}

	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()

	result := &ExecResult{
		Stdout: []byte(stdout.String()),
		Stderr: []byte(stderr.String()),
	}
	if execCmd.Process != nil {
		result.PID = execCmd.Process.Pid
	}

	if runErr != nil {
		// Timeout takes precedence: the runtime kills the process via
		// signal, which surfaces as ExitError, but we want to surface 124
		// (the standard timeout exit code) so callers can distinguish.
		if execCtx.Err() == context.DeadlineExceeded {
			result.ExitCode = 124
			result.Error = fmt.Errorf("process timed out after %d seconds", cfg.ResourceLimits.TimeoutSec)
		} else if exitErr, ok := runErr.(*osexec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			result.Error = runErr
		}
	}
	return result, nil
}

// StartProcess launches cmd asynchronously and returns its PID. cfg.OnExit
// fires exactly once after Wait() returns. Stdout/Stderr/Stdin are wired
// from cfg via wireIO.
func StartProcess(ctx context.Context, s *types.Sandbox, mode IsolationMode, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	if s.MergedDir == "" {
		return 0, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	// Background processes must not inherit a wall-clock timeout.
	ctxNoTimeout := context.Background()
	_ = ctx // request context governs the calling handler; the spawned process is detached

	execCmd, err := buildCmd(ctxNoTimeout, s, mode, cfg, cmd, args...)
	if err != nil {
		return 0, err
	}

	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if mode == ModeNone {
		base := os.Environ()
		execCmd.Env = append(base, execCmd.Env...)
	}

	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	wireIO(execCmd, cfg)

	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	spawnExitReaper(execCmd, cfg.OnExit)
	return execCmd.Process.Pid, nil
}

// buildCmd returns an *osexec.Cmd configured for the chosen isolation mode.
// ModeNone: direct exec in s.MergedDir.
// ModeBwrapPreferred: bwrap when present, else direct fallback.
// ModeBwrapRequired: bwrap or hard error.
func buildCmd(ctx context.Context, s *types.Sandbox, mode IsolationMode, cfg BwrapConfig, cmd string, args ...string) (*osexec.Cmd, error) {
	switch mode {
	case ModeNone:
		c := osexec.CommandContext(ctx, cmd, args...)
		c.Dir = s.MergedDir
		return c, nil
	case ModeBwrapPreferred:
		if _, err := osexec.LookPath("bwrap"); err == nil {
			return buildBwrapCmd(ctx, s, cfg, cmd, args...)
		}
		c := osexec.CommandContext(ctx, cmd, args...)
		c.Dir = s.MergedDir
		return c, nil
	case ModeBwrapRequired:
		if _, err := osexec.LookPath("bwrap"); err != nil {
			return nil, fmt.Errorf("bubblewrap (bwrap) not found: %w. Install with: apt-get install bubblewrap", err)
		}
		return buildBwrapCmd(ctx, s, cfg, cmd, args...)
	}
	return nil, fmt.Errorf("unknown isolation mode: %d", mode)
}

func buildBwrapCmd(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*osexec.Cmd, error) {
	executable, execArgs := BuildExecCommand(s, cfg, cmd, args...)
	execPath, err := osexec.LookPath(executable)
	if err != nil {
		if executable == "prlimit" {
			return nil, fmt.Errorf("prlimit not found: %w. Resource limits require prlimit (part of util-linux)", err)
		}
		return nil, fmt.Errorf("bubblewrap (bwrap) not found: %w", err)
	}
	return osexec.CommandContext(ctx, execPath, execArgs...), nil
}

// wireIO wires cmd.Stdout / cmd.Stderr / cmd.Stdin from cfg. Background
// processes always need a writer for each stream so output is captured;
// pass io.Discard if the caller does not want to retain it.
func wireIO(execCmd *osexec.Cmd, cfg BwrapConfig) {
	if cfg.StdoutWriter != nil {
		execCmd.Stdout = cfg.StdoutWriter
	} else {
		execCmd.Stdout = io.Discard
	}
	if cfg.StderrWriter != nil {
		execCmd.Stderr = cfg.StderrWriter
	} else {
		execCmd.Stderr = io.Discard
	}
	if cfg.StdinReader != nil {
		execCmd.Stdin = cfg.StdinReader
	}
}

// spawnExitReaper waits for the process in a goroutine and dispatches
// ExitInfo to onExit (when non-nil) exactly once. Wait is always invoked
// to prevent zombies, even when onExit is nil.
func spawnExitReaper(execCmd *osexec.Cmd, onExit func(int, int, bool)) {
	if onExit == nil {
		go func() { _ = execCmd.Wait() }()
		return
	}
	go func() {
		waitErr := execCmd.Wait()
		exitCode, signal, oom := ExitInfoFromState(execCmd.ProcessState, waitErr)
		onExit(exitCode, signal, oom)
	}()
}

// ExitInfoFromState extracts (exitCode, signal, oomKilled) from a
// *os.ProcessState plus the wait error. Exported so tests share the
// canonical extraction logic.
func ExitInfoFromState(state *os.ProcessState, waitErr error) (exitCode, signal int, oomKilled bool) {
	if state == nil {
		if waitErr != nil {
			return -1, 0, false
		}
		return 0, 0, false
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return -1, int(status.Signal()), false
		}
		if status.Exited() {
			return status.ExitStatus(), 0, false
		}
	}
	return state.ExitCode(), 0, false
}

// KillProcessGroup kills a process and all its children by process group ID.
func KillProcessGroup(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// IsProcessRunning checks if a process with the given PID is still running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
