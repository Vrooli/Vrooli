package runner

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// DefaultStreamIdleTimeout is how long the scanner waits for output before
// killing the process. This catches runner processes that finish their work
// but fail to exit (e.g., Node.js event loop kept alive by unclosed handles).
const DefaultStreamIdleTimeout = 5 * time.Minute

// idleScanner wraps a bufio.Scanner with an idle timeout. If no lines are
// read for the configured duration, the associated process group is killed
// so the pipe closes and the scanner unblocks.
//
// Usage:
//
//	is := newIdleScanner(stdout, cmd, 5*time.Minute)
//	defer is.Stop()
//	for is.Scan() {
//	    line := is.Text()
//	    // ... process line
//	}
//	// is.Err() returns scanner error (nil on clean EOF)
//	// is.TimedOut() returns true if the idle timeout fired
type idleScanner struct {
	scanner  *bufio.Scanner
	cmd      *exec.Cmd
	timeout  time.Duration
	timer    *time.Timer
	mu       sync.Mutex
	timedOut bool
}

// newIdleScanner creates a scanner that kills the process group after
// idleTimeout without output. Pass 0 to disable the idle timeout.
func newIdleScanner(r io.Reader, cmd *exec.Cmd, idleTimeout time.Duration) *idleScanner {
	s := &idleScanner{
		scanner: bufio.NewScanner(r),
		cmd:     cmd,
		timeout: idleTimeout,
	}
	if idleTimeout > 0 {
		s.timer = time.AfterFunc(idleTimeout, s.kill)
	}
	return s
}

// WithBuffer sets the scanner buffer size (same as bufio.Scanner.Buffer).
func (s *idleScanner) WithBuffer(buf []byte, max int) *idleScanner {
	s.scanner.Buffer(buf, max)
	return s
}

// Scan advances the scanner. Resets the idle timer on each successful read.
func (s *idleScanner) Scan() bool {
	ok := s.scanner.Scan()
	if ok && s.timer != nil {
		s.timer.Reset(s.timeout)
	}
	return ok
}

// Text returns the current token (same as bufio.Scanner.Text).
func (s *idleScanner) Text() string {
	return s.scanner.Text()
}

// Err returns the first non-EOF error from the underlying scanner.
func (s *idleScanner) Err() error {
	return s.scanner.Err()
}

// TimedOut returns true if the idle timeout fired and killed the process.
func (s *idleScanner) TimedOut() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timedOut
}

// Stop cancels the idle timer. Always call via defer after newIdleScanner.
func (s *idleScanner) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
}

// kill sends SIGKILL to the process group, causing the pipe to close
// and the scanner to return EOF.
func (s *idleScanner) kill() {
	s.mu.Lock()
	s.timedOut = true
	s.mu.Unlock()

	killProcessGroup(s.cmd)
}

// killProcessGroup sends SIGKILL to the process group of cmd.
// Safe to call if the process already exited (returns ESRCH, ignored).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// reapProcess kills the process group and waits for exit.
// Call this after the scanner loop exits to ensure the process is cleaned up
// promptly — many runner CLIs (Node.js-based) keep the event loop alive after
// finishing their work (inotify watches, timers, etc.), so we cannot rely on
// them to exit on their own.
//
// Returns nil if we killed the process ourselves (the stream data is
// authoritative). Returns the real exit error if the process had already
// exited before our kill signal.
func reapProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Try to kill the process group. If the process already exited,
	// Kill returns ESRCH and we fall through to Wait for the real status.
	killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	waitErr := cmd.Wait()

	if killErr != nil {
		// Process was already dead — return the real exit status.
		return waitErr
	}
	// We killed it ourselves — the stream data is authoritative, not the
	// exit status (which would be "signal: killed").
	return nil
}
