// Package process — PTY-attached process spawning.
//
// PTYStart is the canonical entry point for interactive (pseudo-
// terminal) command execution. It exists separately from Starter
// because the underlying primitive — github.com/creack/pty's
// StartWithSize — requires a *os/exec.Cmd directly: PTY allocation
// happens before fork, so the kernel can't be tricked into the same
// shape via the abstract Starter contract.
//
// We confine the os/exec dependency to this single function so the
// Round 4 Phase 7 syscall-seam invariant ("every external command
// invocation routes through a canonical seam") still holds: the seam
// for PTY-attached commands is process.PTYStart; everything else
// routes through Starter.
package process

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"syscall"

	"github.com/creack/pty/v2"
)

// PTYOpts mirrors StartOpts for the PTY case. Stdout/Stderr/Stdin
// fields are intentionally omitted — the PTY itself is the bidirectional
// stream; reads and writes go through the returned *os.File.
type PTYOpts struct {
	Path string
	Args []string
	Dir  string
	Env  []string

	// Rows / Cols set the PTY window size at allocation. Subsequent
	// resizes happen via SetPTYSize.
	Rows uint16
	Cols uint16

	// SysProcAttr forwards process attributes (notably Setpgid).
	SysProcAttr *syscall.SysProcAttr
}

// PTYHandle wraps the started PTY-attached process. The caller reads
// from / writes to PTY (an *os.File-like) for input and output, calls
// Wait to reap, and uses Kill / Resize to manage the lifecycle.
type PTYHandle struct {
	cmd *osexec.Cmd
	pty io.ReadWriteCloser
	pid int
}

// PID returns the spawned process ID.
func (h *PTYHandle) PID() int { return h.pid }

// PTY returns the PTY's read/write channel. Callers stream input and
// output through this until Wait returns.
func (h *PTYHandle) PTY() io.ReadWriteCloser { return h.pty }

// Wait blocks until the process exits. Mirrors *exec.Cmd.Wait so the
// returned error carries WaitStatus information for callers that need
// it (the interactive handler reads exit status from the error).
func (h *PTYHandle) Wait() error { return h.cmd.Wait() }

// ExitInfo returns the canonical ProcessExit translation post-Wait.
// Only meaningful after Wait has returned.
func (h *PTYHandle) ExitInfo(waitErr error) ProcessExit {
	return ExitFromState(h.cmd.ProcessState, waitErr)
}

// Kill sends SIGKILL to the process. Idempotent.
func (h *PTYHandle) Kill() error {
	if h.cmd == nil || h.cmd.Process == nil {
		return nil
	}
	return h.cmd.Process.Kill()
}

// SetPTYSize updates the PTY window size on a running session.
func (h *PTYHandle) SetPTYSize(rows, cols uint16) error {
	if f, ok := h.pty.(*os.File); ok {
		return pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols})
	}
	return fmt.Errorf("pty handle does not expose a *os.File")
}

// Close tears down the PTY. The underlying process is not killed; call
// Kill first when needed.
func (h *PTYHandle) Close() error {
	if h.pty == nil {
		return nil
	}
	return h.pty.Close()
}

// PTYStart spawns a process attached to a fresh PTY at the configured
// size. Returns a handle the caller uses to drive I/O and reap the
// process. Errors from os/exec or pty bubble up unwrapped enough that
// callers can translate them to user-facing messages.
func PTYStart(opts PTYOpts) (*PTYHandle, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("process.PTYStart: opts.Path is required")
	}
	cmd := osexec.Command(opts.Path, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if opts.Env != nil {
		cmd.Env = opts.Env
	}
	if opts.SysProcAttr != nil {
		cmd.SysProcAttr = opts.SysProcAttr
	}
	rows := opts.Rows
	cols := opts.Cols
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, fmt.Errorf("pty start %s: %w", opts.Path, err)
	}
	return &PTYHandle{
		cmd: cmd,
		pty: ptmx,
		pid: cmd.Process.Pid,
	}, nil
}
