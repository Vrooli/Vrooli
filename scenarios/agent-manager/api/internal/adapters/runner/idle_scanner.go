package runner

import (
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

// DefaultStreamIdleTimeout is a safety-net timeout that fires only when the
// runner process has produced no output for an extended period. It is NOT the
// primary completion mechanism — process exit is. This timeout exists solely
// for catastrophic cases where the process hangs without producing any output
// at all (dead API, frozen process, etc.).
const DefaultStreamIdleTimeout = 5 * time.Minute

// managedProcess wraps an exec.Cmd with a manual stdout pipe and a background
// goroutine that calls cmd.Wait(). When the main process exits, the goroutine
// kills the process group (terminating grandchildren), which closes all
// write-ends of the stdout pipe and lets the scanner get EOF.
//
// This solves the deadlock where cmd.StdoutPipe() + cmd.Wait() after the
// scanner loop causes the scanner to block forever when grandchild processes
// inherit the pipe's write-end.
type managedProcess struct {
	cmd      *exec.Cmd
	stdout   *os.File // read end of manual pipe (NOT in Go's parentIOPipes)
	timer    *time.Timer
	timeout  time.Duration
	waitCh   chan struct{} // closed when cmd.Wait() returns
	waitErr  error         // result of cmd.Wait()
	timedOut atomic.Bool
	waited   atomic.Bool // ensures Wait() logic runs only once
}

// startManagedProcess creates a manual stdout pipe, starts the command, and
// launches a background goroutine to wait for process exit and clean up.
//
// The caller MUST defer mp.Wait() after a successful return.
func startManagedProcess(cmd *exec.Cmd, timeout time.Duration) (*managedProcess, error) {
	// Create stdout pipe manually. Because we assign an *os.File directly
	// to cmd.Stdout (not via cmd.StdoutPipe), Go does NOT add it to
	// parentIOPipes, so cmd.Wait() won't try to close/wait on it.
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = stdoutW

	mp := &managedProcess{
		cmd:     cmd,
		stdout:  stdoutR,
		timeout: timeout,
		waitCh:  make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}

	// Close the parent's copy of the write-end. Only the child process
	// (and any grandchildren that inherit it) hold write-ends now.
	stdoutW.Close()

	// Start safety-net timer
	if timeout > 0 {
		mp.timer = time.AfterFunc(timeout, mp.kill)
	}

	// Background goroutine: wait for process exit, then kill grandchildren.
	go func() {
		mp.waitErr = cmd.Wait()
		if mp.timer != nil {
			mp.timer.Stop()
		}
		// Kill the process group to terminate grandchildren. This closes
		// their inherited pipe write-ends, causing the scanner to get EOF.
		killProcessGroup(cmd)
		close(mp.waitCh)
	}()

	return mp, nil
}

// Stdout returns the read end of the stdout pipe for use with bufio.Scanner.
func (mp *managedProcess) Stdout() *os.File {
	return mp.stdout
}

// ResetTimer resets the safety-net timer. Call this from the scanner loop
// on each successful read to prevent the timeout from firing during normal
// operation.
func (mp *managedProcess) ResetTimer() {
	if mp.timer != nil {
		mp.timer.Reset(mp.timeout)
	}
}

// TimedOut returns true if the safety-net timer fired and killed the process.
func (mp *managedProcess) TimedOut() bool {
	return mp.timedOut.Load()
}

// Kill explicitly kills the process group. Use this for early exit scenarios
// (e.g., OpenCode's stepFinished).
func (mp *managedProcess) Kill() {
	killProcessGroup(mp.cmd)
}

// Wait blocks until the background goroutine finishes (process exited and
// grandchildren killed), then closes the stdout pipe read-end. Returns the
// process exit error.
//
// Safe to call multiple times — only the first call does cleanup.
// Must be called at least once (typically via defer) to avoid leaking the
// stdout file descriptor.
func (mp *managedProcess) Wait() error {
	<-mp.waitCh
	if mp.waited.CompareAndSwap(false, true) {
		mp.stdout.Close()
	}
	return mp.waitErr
}

// kill sends SIGKILL to the process group. Called by the safety-net timer.
func (mp *managedProcess) kill() {
	mp.timedOut.Store(true)
	killProcessGroup(mp.cmd)
}

// killProcessGroup sends SIGKILL to the process group of cmd.
// Safe to call if the process already exited (returns ESRCH, ignored).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
