package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty/v2"
)

// DOC: docs/internal/SEAMS.md#pty-factory-seam-api
// PTY represents a pseudo-terminal process with read/write, resize, and
// lifecycle control. The default implementation wraps creack/pty; tests can
// substitute a pipe-based fake via PTYFactory.
type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	SetSize(cols, rows uint16) error
	Close() error
	Kill() error
	// ExitCode waits for the process to finish and returns its exit code.
	// Call only after Read returns an error (process exited). Returns -1 if
	// the exit code cannot be determined.
	ExitCode() int
}

// PTYFactory creates a PTY-backed process for the given shell and terminal size.
// Inject a custom factory into SessionManager for testing without real processes.
type PTYFactory func(shell string, cols, rows uint16) (PTY, error)

// realPTY wraps a creack/pty process.
type realPTY struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func (p *realPTY) Read(buf []byte) (int, error)  { return p.ptmx.Read(buf) }
func (p *realPTY) Write(buf []byte) (int, error) { return p.ptmx.Write(buf) }
func (p *realPTY) Close() error                  { return p.ptmx.Close() }

func (p *realPTY) SetSize(cols, rows uint16) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (p *realPTY) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *realPTY) ExitCode() int {
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

// defaultPTYFactory starts a real shell process with a PTY.
func defaultPTYFactory(shell string, cols, rows uint16) (PTY, error) {
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Dir = resolveWorkingDir()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	return &realPTY{ptmx: ptmx, cmd: cmd}, nil
}
