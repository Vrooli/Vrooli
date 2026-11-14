package smoketest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

type executionCapture struct {
	stdout, stderr, combined *limitedWriter
	processSessionID         string
	processScopeRoot         string
}

func newProcessSessionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

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
	sessionID := newProcessSessionID()
	env = append(env, "S2D_PROCESS_SESSION="+sessionID)
	if len(env) > 0 {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(env))
	}
	capture := executionCapture{stdout: newLimitedWriter(e.maxOutputBytes), stderr: newLimitedWriter(e.maxOutputBytes), combined: newLimitedWriter(e.maxOutputBytes), processSessionID: sessionID, processScopeRoot: smokeProcessRoot(env)}
	cmd.Stdout = io.MultiWriter(capture.stdout, capture.combined)
	cmd.Stderr = io.MultiWriter(capture.stderr, capture.combined)
	return cmd, capture
}

// smokeProcessRoot accepts only the owner's paired run identity and temporary
// directory. An ambient TMPDIR or arbitrary working directory is not a scope.
func smokeProcessRoot(env []string) string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	id := values["SMOKE_TEST_RUN_ID"]
	if id == "" || id == "." || id == ".." || filepath.Base(id) != id {
		return ""
	}
	root := smokeUserDataDirPath(id, "", nil)
	if values["TMPDIR"] != root || !filepath.IsAbs(root) {
		return ""
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ""
	}
	return canonical
}

func (c executionCapture) result(duration time.Duration, cmd *exec.Cmd, waitCompleted bool) *ExecutionResult {
	truncated := c.combined.truncatedBytes
	result := &ExecutionResult{Stdout: c.stdout.String(), Stderr: c.stderr.String(), Combined: c.combined.String(), ExitCode: -1, Duration: duration, Truncated: truncated > 0, TruncatedBytes: truncated, ProcessSessionID: c.processSessionID, processScopeRoot: c.processScopeRoot}
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
		checked, leakErr := e.assertNoSessionDescendants(result(true))
		if leakErr != nil {
			return checked, leakErr
		}
		return checked, err
	case <-ctx.Done():
		e.terminateProcess(cmd)
	}
	select {
	case <-completed:
		checked, leakErr := e.assertNoSessionDescendants(result(true))
		if leakErr != nil {
			return checked, leakErr
		}
		return checked, commandContextError(ctx.Err(), timeout)
	case <-e.clock.After(e.killGracePeriod):
		checked, leakErr := e.assertNoSessionDescendants(result(false))
		if leakErr != nil {
			return checked, leakErr
		}
		return checked, commandContextError(ctx.Err(), timeout)
	}
}

func (e *DefaultProcessExecutor) assertNoSessionDescendants(result *ExecutionResult) (*ExecutionResult, error) {
	if result == nil || result.ProcessSessionID == "" {
		return result, fmt.Errorf("post-launch descendant assertion unavailable: launch session marker missing")
	}
	remaining, err := cleanupProcessSession(result.ProcessSessionID, result.processScopeRoot, e.killGracePeriod)
	if err != nil {
		result.RemainingDescendantPIDs = []int{-1}
		result.Stderr += fmt.Sprintf("\npost-launch descendant assertion unavailable: %v", err)
		return result, fmt.Errorf("post-launch descendant assertion unavailable: %w", err)
	}
	result.RemainingDescendantPIDs = remaining
	if len(remaining) > 0 {
		return result, fmt.Errorf("post-launch descendant assertion failed: launch session %s still has processes %v", result.ProcessSessionID, remaining)
	}
	return result, nil
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
