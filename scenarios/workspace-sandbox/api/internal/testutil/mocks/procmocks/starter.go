// Package procmocks — FakeStarter / FakeHandle for process.Starter.
//
// Lives in a subpackage of testutil/mocks (rather than the top-level
// mocks package) because process imports clock, and the top-level
// mocks package imports both clock and process — placing FakeStarter
// next to FakeClock would create a cycle (the process package's own
// tests use FakeClock from testutil/mocks).
//
// Tests that need both FakeClock and FakeStarter import them from
// their respective packages. See docs/SEAMS.md "Process Starter Seam
// (Round 4 Phase 7)".
package procmocks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"

	"workspace-sandbox/internal/process"
)

// CommandBehavior describes how FakeStarter should respond to a
// matching Start invocation. Zero value = exit 0 with empty output.
type CommandBehavior struct {
	// StartErr, when non-nil, is returned from Start (no Handle is
	// created). Use this to simulate "binary not found" or "fork
	// failed" failures.
	StartErr error

	// WaitErr, when non-nil, is returned from Handle.Wait alongside the
	// scripted Exit. Use this for "process kill returned EPERM" or
	// context-cancellation simulation.
	WaitErr error

	// Exit is the ProcessExit returned from Wait. Zero value = exit 0.
	Exit process.ProcessExit

	// Stdout / Stderr are written to the Stdout / Stderr writers passed
	// in StartOpts (when non-nil) at Wait time. This mirrors the real
	// behavior where output streams flush as the process runs.
	Stdout []byte
	Stderr []byte

	// WaitDelay, when non-zero, makes Wait block for the duration before
	// returning. Useful for testing context-cancellation / hung-process
	// behavior.
	WaitDelay time.Duration

	// Hold, when true, blocks Wait until Release() is called on the
	// FakeHandle. Use this to model long-running processes that must
	// be killed explicitly.
	Hold bool
}

// FakeStarter is the test double for process.Starter.
type FakeStarter struct {
	mu sync.Mutex

	commands        map[string]CommandBehavior
	defaultBehavior *CommandBehavior
	lookPath        map[string]string
	startErr        error
	failOnUnmatched bool
	pidCounter      int

	// Calls records every Start invocation in order.
	Calls []StartCall
}

// StartCall records a single Start invocation.
type StartCall struct {
	Path        string
	Args        []string
	Dir         string
	Env         []string
	HasStdin    bool
	HasStdout   bool
	HasStderr   bool
	SysProcAttr *syscall.SysProcAttr
	FullCommand string
}

// NewFakeStarter constructs a FakeStarter with paranoid defaults
// (unmatched commands fail explicitly).
func NewFakeStarter() *FakeStarter {
	return &FakeStarter{
		commands:        map[string]CommandBehavior{},
		lookPath:        map[string]string{},
		failOnUnmatched: true,
	}
}

// SetDefault configures the fallback behavior for unmatched commands.
// Calling SetDefault disables fail-on-unmatched.
func (f *FakeStarter) SetDefault(b CommandBehavior) {
	f.mu.Lock()
	defer f.mu.Unlock()
	bb := b
	f.defaultBehavior = &bb
	f.failOnUnmatched = false
}

// SetFailOnUnmatched toggles paranoid mode.
func (f *FakeStarter) SetFailOnUnmatched(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failOnUnmatched = v
}

// AddCommand registers a behavior for commands matching pattern.
func (f *FakeStarter) AddCommand(pattern string, b CommandBehavior) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands[pattern] = b
}

// SetLookPath configures the LookPath table.
func (f *FakeStarter) SetLookPath(name, path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookPath[name] = path
}

// SetStartErr makes every subsequent Start call fail with err.
func (f *FakeStarter) SetStartErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startErr = err
}

// Reset clears recorded calls (but keeps configured behaviors).
func (f *FakeStarter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = nil
}

// LookPath implements process.Starter.
func (f *FakeStarter) LookPath(name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if path, ok := f.lookPath[name]; ok {
		return path, nil
	}
	return "", fmt.Errorf("%w: %s", process.ErrBinaryNotFound, name)
}

// Start implements process.Starter.
func (f *FakeStarter) Start(ctx context.Context, opts process.StartOpts) (process.Handle, error) {
	f.mu.Lock()
	full := strings.TrimSpace(opts.Path + " " + strings.Join(opts.Args, " "))
	call := StartCall{
		Path:        opts.Path,
		Args:        append([]string(nil), opts.Args...),
		Dir:         opts.Dir,
		Env:         append([]string(nil), opts.Env...),
		HasStdin:    opts.Stdin != nil,
		HasStdout:   opts.Stdout != nil,
		HasStderr:   opts.Stderr != nil,
		SysProcAttr: opts.SysProcAttr,
		FullCommand: full,
	}
	f.Calls = append(f.Calls, call)

	if f.startErr != nil {
		err := f.startErr
		f.mu.Unlock()
		return nil, err
	}

	behavior, matched := f.matchLocked(full)
	failOnUnmatched := f.failOnUnmatched
	pid := f.nextPIDLocked()
	f.mu.Unlock()

	if !matched && failOnUnmatched {
		return nil, fmt.Errorf("FakeStarter: no behavior configured for command %q", full)
	}
	if behavior.StartErr != nil {
		return nil, behavior.StartErr
	}

	return &FakeHandle{
		pid:      pid,
		behavior: behavior,
		stdout:   opts.Stdout,
		stderr:   opts.Stderr,
		release:  make(chan struct{}),
	}, nil
}

func (f *FakeStarter) matchLocked(full string) (CommandBehavior, bool) {
	if behavior, ok := f.commands[full]; ok {
		return behavior, true
	}
	var bestPattern string
	var bestBehavior CommandBehavior
	for pattern, behavior := range f.commands {
		if strings.HasPrefix(full, pattern) && len(pattern) > len(bestPattern) {
			bestPattern = pattern
			bestBehavior = behavior
		}
	}
	if bestPattern != "" {
		return bestBehavior, true
	}
	if f.defaultBehavior != nil {
		return *f.defaultBehavior, true
	}
	return CommandBehavior{}, false
}

func (f *FakeStarter) nextPIDLocked() int {
	f.pidCounter++
	return 90000 + f.pidCounter
}

// MatchedCalls returns the recorded calls whose FullCommand starts with
// pattern. Useful for "fuse-overlayfs was invoked exactly twice".
func (f *FakeStarter) MatchedCalls(pattern string) []StartCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []StartCall
	for _, c := range f.Calls {
		if strings.HasPrefix(c.FullCommand, pattern) {
			out = append(out, c)
		}
	}
	return out
}

// FakeHandle implements process.Handle for tests.
type FakeHandle struct {
	mu       sync.Mutex
	pid      int
	behavior CommandBehavior
	stdout   io.Writer
	stderr   io.Writer
	release  chan struct{}
	released bool
	waited   bool
	exit     process.ProcessExit
	waitErr  error
	killed   bool
}

// PID implements process.Handle.
func (h *FakeHandle) PID() int { return h.pid }

// Wait implements process.Handle.
func (h *FakeHandle) Wait(ctx context.Context) (process.ProcessExit, error) {
	h.mu.Lock()
	if h.waited {
		exit, err := h.exit, h.waitErr
		h.mu.Unlock()
		return exit, err
	}
	hold := h.behavior.Hold
	delay := h.behavior.WaitDelay
	stdout, stderr := h.stdout, h.stderr
	stdoutData, stderrData := h.behavior.Stdout, h.behavior.Stderr
	scriptedExit := h.behavior.Exit
	scriptedErr := h.behavior.WaitErr
	h.mu.Unlock()

	if stdout != nil && len(stdoutData) > 0 {
		_, _ = io.Copy(stdout, bytes.NewReader(stdoutData))
	}
	if stderr != nil && len(stderrData) > 0 {
		_, _ = io.Copy(stderr, bytes.NewReader(stderrData))
	}

	if hold {
		select {
		case <-h.release:
		case <-ctx.Done():
			h.mu.Lock()
			h.waited = true
			h.exit = process.ProcessExit{ExitCode: -1, Signal: int(syscall.SIGKILL)}
			h.waitErr = ctx.Err()
			exit, err := h.exit, h.waitErr
			h.mu.Unlock()
			return exit, err
		}
	} else if delay > 0 {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			h.mu.Lock()
			h.waited = true
			h.exit = process.ProcessExit{ExitCode: -1, Signal: int(syscall.SIGKILL)}
			h.waitErr = ctx.Err()
			exit, err := h.exit, h.waitErr
			h.mu.Unlock()
			return exit, err
		}
	}

	h.mu.Lock()
	h.waited = true
	h.exit = scriptedExit
	h.waitErr = scriptedErr
	exit, err := h.exit, h.waitErr
	h.mu.Unlock()
	return exit, err
}

// Kill implements process.Handle. Idempotent.
func (h *FakeHandle) Kill() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.killed {
		return nil
	}
	h.killed = true
	if !h.released {
		close(h.release)
		h.released = true
	}
	return nil
}

// KillProcessGroup implements process.Handle.
func (h *FakeHandle) KillProcessGroup() error { return h.Kill() }

// Killed reports whether Kill has been called.
func (h *FakeHandle) Killed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.killed
}

// Release unblocks a Hold-ing Wait.
func (h *FakeHandle) Release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.released {
		close(h.release)
		h.released = true
	}
}

// Compile-time interface assertions.
var (
	_ process.Starter = (*FakeStarter)(nil)
	_ process.Handle  = (*FakeHandle)(nil)
)
