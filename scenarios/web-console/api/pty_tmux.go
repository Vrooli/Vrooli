package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/creack/pty/v2"
)

// tmuxSessionPrefix distinguishes web console sessions from user tmux sessions.
const tmuxSessionPrefix = "wc-"

// tmuxPTY implements the PTY interface using a tmux session as the backing process.
// The shell runs inside a detached tmux session; I/O is streamed via
// `tmux attach-session` wrapped in a local PTY.
type tmuxPTY struct {
	sessionName string   // tmux session name: "wc-{id}"
	ptmx        *os.File // PTY master connected to tmux attach process
	cmd         *exec.Cmd
	mu          sync.Mutex
	closed      bool
}

func (p *tmuxPTY) Read(buf []byte) (int, error)  { return p.ptmx.Read(buf) }
func (p *tmuxPTY) Write(buf []byte) (int, error) { return p.ptmx.Write(buf) }

func (p *tmuxPTY) SetSize(cols, rows uint16) error {
	// Resize the tmux window directly (not the local PTY)
	if err := exec.Command("tmux", "resize-window", "-t", p.sessionName,
		"-x", strconv.Itoa(int(cols)),
		"-y", strconv.Itoa(int(rows))).Run(); err != nil {
		return fmt.Errorf("tmux resize: %w", err)
	}
	// Also resize the local PTY so the attach process knows the new size
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (p *tmuxPTY) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.ptmx.Close()
}

func (p *tmuxPTY) Kill() error {
	// Kill the tmux session, which kills the shell inside it
	_ = exec.Command("tmux", "kill-session", "-t", p.sessionName).Run()
	// Also kill the attach process
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return nil
}

func (p *tmuxPTY) ExitCode() int {
	// Try to read the exit code from tmux's pane_dead_status
	out, err := exec.Command("tmux", "display-message", "-t", p.sessionName, "-p", "#{pane_dead_status}").Output()
	if err == nil {
		status := strings.TrimSpace(string(out))
		if code, parseErr := strconv.Atoi(status); parseErr == nil {
			return code
		}
	}
	// Fall back to waiting on the attach process
	if p.cmd.ProcessState != nil {
		return p.cmd.ProcessState.ExitCode()
	}
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

func (p *tmuxPTY) HasChildProcess() bool {
	// Get the pane PID from tmux
	out, err := exec.Command("tmux", "display-message", "-t", p.sessionName, "-p", "#{pane_pid}").Output()
	if err != nil {
		return false
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return false
	}
	childrenPath := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
	data, readErr := os.ReadFile(childrenPath)
	if readErr != nil {
		return false
	}
	return len(bytes.TrimSpace(data)) > 0
}

// buildSessionEnv constructs the filtered environment for a new session.
// Used by both defaultPTYFactory and tmuxPTYFactory.
func buildSessionEnv(spec SessionLaunchSpec) []string {
	return applySessionEnv(
		ensureTermEnv(
			filterServiceEnv(
				filterClaudeEnv(os.Environ()),
			),
		),
		spec.Env,
	)
}

// tmuxPTYFactory creates a tmux-backed PTY for persistent sessions.
func tmuxPTYFactory(spec SessionLaunchSpec) (PTY, error) {
	sessionName := tmuxSessionPrefix + spec.SessionID

	// 1. Create detached tmux session with the target shell
	createCmd := exec.Command("tmux", "new-session", "-d",
		"-s", sessionName,
		"-x", strconv.Itoa(int(spec.Cols)),
		"-y", strconv.Itoa(int(spec.Rows)),
		spec.Shell,
	)
	createCmd.Env = buildSessionEnv(spec)
	if err := createCmd.Run(); err != nil {
		return nil, fmt.Errorf("tmux new-session: %w", err)
	}

	// 2. Configure session options
	_ = exec.Command("tmux", "set-option", "-t", sessionName, "remain-on-exit", "on").Run()
	// Enable mouse mode so xterm.js scroll wheel events are forwarded to tmux,
	// which manages its own scrollback buffer (xterm.js has no scrollback when
	// tmux controls the viewport).
	_ = exec.Command("tmux", "set-option", "-t", sessionName, "mouse", "on").Run()
	// Set a generous scrollback limit (default 2000 is often insufficient)
	_ = exec.Command("tmux", "set-option", "-t", sessionName, "history-limit", "50000").Run()
	// Propagate pane title changes (from OSC escape sequences) to the parent
	// terminal so xterm.js onTitleChange fires and tab names update.
	_ = exec.Command("tmux", "set-option", "-t", sessionName, "set-titles", "on").Run()
	_ = exec.Command("tmux", "set-option", "-t", sessionName, "set-titles-string", "#{pane_title}").Run()

	// 3. Attach to tmux session via a PTY for I/O streaming
	p, err := tmuxAttach(sessionName)
	if err != nil {
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		return nil, err
	}
	return p, nil
}

// tmuxAttach connects to an existing tmux session for I/O.
// Used by both tmuxPTYFactory (new sessions) and recovery (existing sessions).
func tmuxAttach(sessionName string) (*tmuxPTY, error) {
	attachCmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	// The server process may have TERM=dumb (common when started by a
	// non-interactive lifecycle manager). tmux attach requires a terminal
	// type that supports basic operations like clear, so we ensure
	// TERM=xterm-256color — matching what ensureTermEnv sets for shells.
	attachCmd.Env = ensureTermEnv(os.Environ())
	ptmx, err := pty.Start(attachCmd)
	if err != nil {
		return nil, fmt.Errorf("tmux attach %s: %w", sessionName, err)
	}
	return &tmuxPTY{
		sessionName: sessionName,
		ptmx:        ptmx,
		cmd:         attachCmd,
	}, nil
}

// DiscoverTmuxSessions finds surviving web console tmux sessions.
// Returns session IDs (without the "wc-" prefix).
func DiscoverTmuxSessions() ([]string, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// tmux not running or no sessions — not an error
		return nil, nil
	}
	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, tmuxSessionPrefix) {
			sessions = append(sessions, strings.TrimPrefix(line, tmuxSessionPrefix))
		}
	}
	return sessions, nil
}
