package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	// Filter Claude Code env vars first, then ensure TERM is set.
	// This prevents nested session detection when users run `claude` in
	// web-console terminals, even if the server was started from Claude Code.
	cmd.Env = ensureTermEnv(filterClaudeEnv(os.Environ()))
	cmd.Dir = resolveWorkingDir()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	return &realPTY{ptmx: ptmx, cmd: cmd}, nil
}

// ensureTermEnv guarantees TERM=xterm-256color in the environment slice.
// The server's own TERM (which may be "dumb" or unset when launched via
// systemd/lifecycle) is irrelevant — the browser-side xterm.js emulator
// supports xterm-256color, and shell programs need this to emit colors.
func ensureTermEnv(env []string) []string {
	const want = "TERM=xterm-256color"
	for i, v := range env {
		if strings.HasPrefix(v, "TERM=") {
			env[i] = want
			return env
		}
	}
	return append(env, want)
}

// filterClaudeEnv removes Claude Code-specific environment variables from
// the environment slice. This prevents nested session detection when users
// run `claude` in a web-console terminal, even if the web-console server
// was started from within a Claude Code session.
//
// Filtered patterns:
//   - CLAUDECODE (nested session detection marker)
//   - CLAUDE_* (session IDs, config paths, internal state)
//   - BASH_FUNC_claude_code::* (exported shell functions)
func filterClaudeEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, v := range env {
		// Filter CLAUDECODE exactly
		if strings.HasPrefix(v, "CLAUDECODE=") {
			continue
		}
		// Filter all CLAUDE_* variables
		if strings.HasPrefix(v, "CLAUDE_") {
			continue
		}
		// Filter exported bash functions for claude_code::
		if strings.HasPrefix(v, "BASH_FUNC_claude_code::") {
			continue
		}
		result = append(result, v)
	}
	return result
}
