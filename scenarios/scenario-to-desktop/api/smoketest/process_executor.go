package smoketest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// DefaultMaxOutputBytes is the default maximum output size (10MB).
const DefaultMaxOutputBytes = 10 * 1024 * 1024

// DefaultProcessExecutor implements ProcessExecutor using real process execution.
type DefaultProcessExecutor struct {
	logger         Logger
	maxOutputBytes int
}

// NewProcessExecutor creates a new process executor.
func NewProcessExecutor(logger Logger) *DefaultProcessExecutor {
	return &DefaultProcessExecutor{
		logger:         logger,
		maxOutputBytes: DefaultMaxOutputBytes,
	}
}

// NewProcessExecutorWithLimit creates a new process executor with a custom output limit.
func NewProcessExecutorWithLimit(logger Logger, maxOutputBytes int) *DefaultProcessExecutor {
	if maxOutputBytes <= 0 {
		maxOutputBytes = DefaultMaxOutputBytes
	}
	return &DefaultProcessExecutor{
		logger:         logger,
		maxOutputBytes: maxOutputBytes,
	}
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
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir

	// Set process group on Unix systems for clean termination
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	// Create limited writers for stdout and stderr
	stdoutWriter := newLimitedWriter(e.maxOutputBytes)
	stderrWriter := newLimitedWriter(e.maxOutputBytes)
	combinedWriter := newLimitedWriter(e.maxOutputBytes)

	// Use MultiWriter to capture both separate and combined output
	cmd.Stdout = io.MultiWriter(stdoutWriter, combinedWriter)
	cmd.Stderr = io.MultiWriter(stderrWriter, combinedWriter)

	startTime := time.Now()

	type execResult struct {
		err error
	}
	resultCh := make(chan execResult, 1)

	go func() {
		err := cmd.Run()
		resultCh <- execResult{err: err}
	}()

	buildResult := func() *ExecutionResult {
		duration := time.Since(startTime)
		truncatedBytes := stdoutWriter.truncatedBytes + stderrWriter.truncatedBytes
		result := &ExecutionResult{
			Stdout:         stdoutWriter.String(),
			Stderr:         stderrWriter.String(),
			Combined:       combinedWriter.String(),
			ExitCode:       -1,
			Duration:       duration,
			Truncated:      truncatedBytes > 0,
			TruncatedBytes: truncatedBytes,
		}
		if cmd.ProcessState != nil {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
		return result
	}

	select {
	case res := <-resultCh:
		return buildResult(), res.err

	case <-execCtx.Done():
		e.terminateProcess(cmd)

		// Wait briefly for process to exit and collect output
		select {
		case <-resultCh:
			result := buildResult()
			if execCtx.Err() == context.DeadlineExceeded {
				return result, fmt.Errorf("command timed out after %s", timeout)
			}
			return result, fmt.Errorf("command cancelled")
		case <-time.After(2 * time.Second):
			result := buildResult()
			if execCtx.Err() == context.DeadlineExceeded {
				return result, fmt.Errorf("command timed out after %s", timeout)
			}
			return result, fmt.Errorf("command cancelled")
		}
	}
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

	if runtime.GOOS == "windows" {
		// Use taskkill with /T flag to kill the entire process tree
		// /F = force, /T = tree (all child processes), /PID = process ID
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
		if err := killCmd.Run(); err != nil {
			// Fallback to direct process kill if taskkill fails
			_ = cmd.Process.Kill()
		}
		return
	}

	// On Unix, kill the process group
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return
	}

	// Fallback to killing just the process
	_ = cmd.Process.Kill()
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
