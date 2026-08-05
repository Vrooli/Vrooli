// Package ptyfake provides a pipe-based PTY substitute for fast, deterministic
// tests across packages. It satisfies the pty.PTY interface without spawning
// real shell processes.
package ptyfake

import (
	"context"
	"io"
	"path/filepath"
	"sync"

	"web-console/internal/config"
	"web-console/internal/pty"
)

// FakePTY is a pipe-based PTY substitute. Use FakePTYWithOutput when you need
// to simulate PTY stdout from tests.
type FakePTY struct {
	StdoutReader  *io.PipeReader
	StdinWriter   *io.PipeWriter
	Mu            sync.Mutex
	Cols          uint16
	Rows          uint16
	CurrentDirVal string
	Killed        bool
	Closed        bool
	ExitCodeVal   int
	SetSizeCalls  int
	// WriteInputErr, when non-nil, is returned by WriteInput instead of
	// the bytes being written. Lets tests drive backend write failures —
	// and in particular the difference between a dead PTY and a backend
	// that merely rejected one payload — without a real backend.
	WriteInputErr error
}

func (f *FakePTY) Read(p []byte) (int, error) { return f.StdoutReader.Read(p) }

func (f *FakePTY) WriteInput(data []byte, _ pty.InputKind) error {
	f.Mu.Lock()
	forced := f.WriteInputErr
	f.Mu.Unlock()
	if forced != nil {
		return forced
	}
	_, err := f.StdinWriter.Write(data)
	return err
}

// SetWriteInputErr makes every subsequent WriteInput fail with err (nil
// restores normal delivery).
func (f *FakePTY) SetWriteInputErr(err error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.WriteInputErr = err
}

func (f *FakePTY) SetSize(cols, rows uint16) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Cols = cols
	f.Rows = rows
	f.SetSizeCalls++
	return nil
}

func (f *FakePTY) Close() error {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.Closed {
		return nil
	}
	f.Closed = true
	f.StdoutReader.Close()
	f.StdinWriter.Close()
	return nil
}

func (f *FakePTY) Kill() error {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Killed = true
	return nil
}

func (f *FakePTY) ExitCode() int {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	return f.ExitCodeVal
}

func (f *FakePTY) HasChildProcess() bool { return false }

// ProbeReady is a no-op on the fake PTY.
func (f *FakePTY) ProbeReady(_ context.Context) error { return nil }

func (f *FakePTY) CurrentDir(_ context.Context) (string, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CurrentDirVal != "" {
		return f.CurrentDirVal, nil
	}
	cwd, err := filepath.Abs(config.ResolveWorkingDir())
	if err != nil {
		return config.ResolveWorkingDir(), nil
	}
	return cwd, nil
}

func (f *FakePTY) SetExitCode(code int) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.ExitCodeVal = code
}

// FakePTYWithOutput extends FakePTY with a writable stdout pipe so tests
// can inject output that the session's readLoop will broadcast to subscribers.
type FakePTYWithOutput struct {
	FakePTY
	OutW *io.PipeWriter
}

// NewFakePTYWithOutput constructs a FakePTYWithOutput with the stdin pipe
// drained in a background goroutine so callers writing to it don't block.
func NewFakePTYWithOutput() *FakePTYWithOutput {
	stdoutR, stdoutW := io.Pipe()
	stdinR, stdinW := io.Pipe()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := stdinR.Read(buf); err != nil {
				return
			}
		}
	}()
	return &FakePTYWithOutput{
		FakePTY: FakePTY{
			StdoutReader: stdoutR,
			StdinWriter:  stdinW,
		},
		OutW: stdoutW,
	}
}

func (f *FakePTYWithOutput) Close() error {
	f.OutW.Close()
	f.StdinWriter.Close()
	return nil
}

// Factory returns a pty.Factory that always returns the supplied PTY instance.
// Use when a test needs to inspect or control the exact PTY a session uses.
func Factory(p pty.PTY) pty.Factory {
	return func(spec pty.LaunchSpec) (pty.PTY, error) {
		return p, nil
	}
}

// NewFactory returns a pty.Factory that creates a fresh FakePTYWithOutput
// for each session. Useful for tests that create multiple sessions.
func NewFactory() pty.Factory {
	return func(spec pty.LaunchSpec) (pty.PTY, error) {
		return NewFakePTYWithOutput(), nil
	}
}
