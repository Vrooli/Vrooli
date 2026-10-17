package smoketest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
)

// DefaultMaxOutputBytes is the default maximum output size (10MB).
const DefaultMaxOutputBytes = 10 * 1024 * 1024

// DefaultKillGracePeriod is how long to wait for a process to exit after being killed.
const DefaultKillGracePeriod = 2 * time.Second

// ProcessExecutorOption configures a DefaultProcessExecutor.
type ProcessExecutorOption func(*DefaultProcessExecutor)

// WithClock sets the clock for time-based operations.
func WithClock(clock Clock) ProcessExecutorOption {
	return func(e *DefaultProcessExecutor) {
		e.clock = clock
	}
}

// WithOutputLimit sets the maximum output size.
func WithOutputLimit(maxBytes int) ProcessExecutorOption {
	return func(e *DefaultProcessExecutor) {
		if maxBytes > 0 {
			e.maxOutputBytes = maxBytes
		}
	}
}

// WithKillGracePeriod sets how long to wait for process exit after kill.
func WithKillGracePeriod(d time.Duration) ProcessExecutorOption {
	return func(e *DefaultProcessExecutor) {
		if d > 0 {
			e.killGracePeriod = d
		}
	}
}

// WithOnProcessStarted sets a callback invoked when the process starts.
// This is primarily for testing - allows tests to know when to cancel.
func WithOnProcessStarted(fn func()) ProcessExecutorOption {
	return func(e *DefaultProcessExecutor) {
		e.onProcessStarted = fn
	}
}

// WithOnProcessStartedPID sets a callback invoked with the PID after the process starts.
// This runs after cmd.Start() succeeds and before cmd.Wait() blocks,
// enabling monitoring of the process while it runs.
func WithOnProcessStartedPID(fn func(pid int)) ProcessExecutorOption {
	return func(e *DefaultProcessExecutor) {
		e.onProcessStartedPID = fn
	}
}

// DefaultProcessExecutor implements ProcessExecutor using real process execution.
type DefaultProcessExecutor struct {
	logger              Logger
	maxOutputBytes      int
	clock               Clock
	killGracePeriod     time.Duration
	onProcessStarted    func()    // testing hook: called when process starts
	onProcessStartedPID func(int) // monitoring hook: called with PID after process starts
}

// NewProcessExecutor creates a new process executor.
func NewProcessExecutor(logger Logger, opts ...ProcessExecutorOption) *DefaultProcessExecutor {
	e := &DefaultProcessExecutor{
		logger:          logger,
		maxOutputBytes:  DefaultMaxOutputBytes,
		clock:           RealClock{},
		killGracePeriod: DefaultKillGracePeriod,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// NewProcessExecutorWithLimit creates a new process executor with a custom output limit.
// Deprecated: Use NewProcessExecutor with WithOutputLimit option instead.
func NewProcessExecutorWithLimit(logger Logger, maxOutputBytes int) *DefaultProcessExecutor {
	return NewProcessExecutor(logger, WithOutputLimit(maxOutputBytes))
}

// Execute runs a command and returns combined stdout/stderr output.
// Deprecated: Use ExecuteWithResult for access to separated stdout/stderr.
func (e *DefaultProcessExecutor) Execute(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (string, error) {
	result, err := e.ExecuteWithResult(ctx, workDir, command, args, env, timeout)
	if result != nil {
		return result.Combined, err
	}
	return "", err
}

// ExecuteWithResult runs a command and returns detailed execution result with separated stdout/stderr.
func (e *DefaultProcessExecutor) ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*ExecutionResult, error) {
	timeout = normalizedTimeout(timeout)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd, capture := e.executionCommand(workDir, command, args, env)
	startTime := e.clock.Now()
	result := func(waitCompleted bool) *ExecutionResult {
		return capture.result(e.clock.Now().Sub(startTime), cmd, waitCompleted)
	}
	if err := commandContextError(execCtx.Err(), timeout); err != nil {
		return result(false), err
	}
	if err := e.startProcess(cmd); err != nil {
		return result(false), err
	}
	return e.waitForProcess(execCtx, cmd, timeout, result)
}

type executionCapture struct{ stdout, stderr, combined *limitedWriter }

func normalizedTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 30 * time.Second
	}
	return timeout
}

func (e *DefaultProcessExecutor) executionCommand(workDir, command string, args, env []string) (*exec.Cmd, executionCapture) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	configureProcessGroup(cmd)
	if len(env) > 0 {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(env))
	}
	capture := executionCapture{newLimitedWriter(e.maxOutputBytes), newLimitedWriter(e.maxOutputBytes), newLimitedWriter(e.maxOutputBytes)}
	cmd.Stdout = io.MultiWriter(capture.stdout, capture.combined)
	cmd.Stderr = io.MultiWriter(capture.stderr, capture.combined)
	return cmd, capture
}

func (c executionCapture) result(duration time.Duration, cmd *exec.Cmd, waitCompleted bool) *ExecutionResult {
	truncated := c.stdout.truncatedBytes + c.stderr.truncatedBytes
	result := &ExecutionResult{Stdout: c.stdout.String(), Stderr: c.stderr.String(), Combined: c.combined.String(), ExitCode: -1, Duration: duration, Truncated: truncated > 0, TruncatedBytes: truncated}
	if waitCompleted && cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	return result
}

func commandContextError(err error, timeout time.Duration) error {
	if err == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		return fmt.Errorf("command cancelled")
	}
	return nil
}

func (e *DefaultProcessExecutor) startProcess(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	if e.onProcessStarted != nil {
		e.onProcessStarted()
	}
	if e.onProcessStartedPID != nil && cmd.Process != nil {
		e.onProcessStartedPID(cmd.Process.Pid)
	}
	return nil
}

func (e *DefaultProcessExecutor) waitForProcess(ctx context.Context, cmd *exec.Cmd, timeout time.Duration, result func(bool) *ExecutionResult) (*ExecutionResult, error) {
	completed := make(chan error, 1)
	go func() { completed <- cmd.Wait() }()
	select {
	case err := <-completed:
		// The launcher can exit successfully while a descendant (notably an
		// Electron-launched runtime) survives in its dedicated process group.
		// Success is not a cleanup exemption: the smoke test owns that group.
		e.cleanupExitedProcessGroup(cmd.Process.Pid)
		return result(true), err
	case <-ctx.Done():
		e.terminateProcess(cmd)
	}
	select {
	case <-completed:
		return result(true), commandContextError(ctx.Err(), timeout)
	case <-e.clock.After(e.killGracePeriod):
		return result(false), commandContextError(ctx.Err(), timeout)
	}
}

func (e *DefaultProcessExecutor) cleanupExitedProcessGroup(pid int) {
	cleanupProcessGroupAfterLeaderExit(pid, e.killGracePeriod)
}

// LookPath searches for an executable in the system PATH.
func (e *DefaultProcessExecutor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// terminateProcess kills a process and its children.
func (e *DefaultProcessExecutor) terminateProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	terminateProcessTree(cmd)
}

// limitedWriter wraps a buffer and limits the amount of data written.
// It tracks how many bytes were truncated.
type limitedWriter struct {
	buf            bytes.Buffer
	limit          int
	written        int
	truncatedBytes int
	mu             sync.Mutex
}

// newLimitedWriter creates a new limitedWriter with the specified limit.
func newLimitedWriter(limit int) *limitedWriter {
	return &limitedWriter{limit: limit}
}

// Write implements io.Writer with size limiting.
func (w *limitedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n = len(p)
	if w.written >= w.limit {
		// Already at limit, just count truncated bytes
		w.truncatedBytes += n
		return n, nil
	}

	remaining := w.limit - w.written
	if len(p) > remaining {
		// Write what we can, track the rest as truncated
		w.buf.Write(p[:remaining])
		w.written += remaining
		w.truncatedBytes += len(p) - remaining
	} else {
		w.buf.Write(p)
		w.written += len(p)
	}

	return n, nil
}

// String returns the buffered content.
func (w *limitedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}
