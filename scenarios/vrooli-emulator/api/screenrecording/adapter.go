package screenrecording

import (
	"context"
	"time"
)

// ProcessExecutorResult is the interface that smoketest.ProcessExecutor.ExecuteWithResult returns.
// We define a narrower adapter here to avoid importing smoketest directly.
type ProcessExecutorResult interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
}

// ProcessExecutorAdapter wraps any executor that returns a result with Stdout/Stderr/ExitCode
// fields (like smoketest.DefaultProcessExecutor) to satisfy CommandExecutor.
type ProcessExecutorAdapter struct {
	// ExecFn executes a command and returns stdout, stderr, exit code, and error.
	ExecFn func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (stdout, stderr string, exitCode int, err error)
}

// ExecuteWithResult implements CommandExecutor by calling the wrapped function.
func (a *ProcessExecutorAdapter) ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*ExecutionResult, error) {
	stdout, stderr, exitCode, err := a.ExecFn(ctx, workDir, command, args, env, timeout)
	if err != nil {
		return &ExecutionResult{
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
		}, err
	}
	return &ExecutionResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}
