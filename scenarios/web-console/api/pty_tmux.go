package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
)

// errPTYClosed is returned when I/O is attempted on a closed tmuxPTY.
var errPTYClosed = errors.New("pty is closed")

// tmuxSessionPrefix distinguishes web console sessions from user tmux sessions.
const tmuxSessionPrefix = "wc-"

// defaultTmuxSocket is the dedicated tmux socket name for web-console sessions.
// Using a separate socket ensures the tmux server is distinct from the user's
// default tmux server, giving us full lifecycle control.
const defaultTmuxSocket = "wc"

// tmuxScopeName is the systemd scope unit name used to isolate the tmux server
// from the parent service's cgroup. Without this, the tmux server inherits
// the API's cgroup (e.g., vrooli-autoheal.service), and gets killed when that
// service restarts.
const defaultTmuxScopeName = "wc-tmux-server"

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

func (p *tmuxPTY) Read(buf []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, io.EOF
	}
	p.mu.Unlock()
	return p.ptmx.Read(buf)
}

func (p *tmuxPTY) Write(buf []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, errPTYClosed
	}
	p.mu.Unlock()
	return p.ptmx.Write(buf)
}

// tmuxProbeReadyPollInterval is how often ProbeReady re-queries tmux for
// attached clients. Kept short because the typical handshake completes in
// 50–500 ms — we want to return quickly once tmux is ready.
var tmuxProbeReadyPollInterval = 25 * time.Millisecond

// ProbeReady waits until `tmux list-clients -t <session>` reports at least
// one attached client (our attach-session process), which guarantees that
// subsequent writes to the PTY master will be relayed into the tmux pane
// rather than silently buffered in the unattached attach process.
//
// Returns context.DeadlineExceeded if the handshake does not complete
// within the caller's deadline; returns errPTYClosed if Close races with
// the probe.
func (p *tmuxPTY) ProbeReady(ctx context.Context) error {
	for {
		p.mu.Lock()
		closed := p.closed
		sessionName := p.sessionName
		p.mu.Unlock()
		if closed {
			return errPTYClosed
		}

		out, err := tmuxCmdContext(ctx, "list-clients", "-t", sessionName, "-F", "#{client_tty}").Output()
		if err == nil && len(bytes.TrimSpace(out)) > 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(tmuxProbeReadyPollInterval):
		}
	}
}

func (p *tmuxPTY) SetSize(cols, rows uint16) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errPTYClosed
	}
	p.mu.Unlock()
	// Resize the tmux window directly (not the local PTY)
	if err := tmuxCmd("resize-window", "-t", p.sessionName,
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
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	// Kill the tmux session, which kills the shell inside it
	if err := tmuxCmd("kill-session", "-t", p.sessionName).Run(); err != nil {
		log.Printf("tmux: kill-session %s failed: %v", p.sessionName, err)
	}
	// Also kill the attach process (only if ptmx wasn't already closed)
	if !closed && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return nil
}

func (p *tmuxPTY) ExitCode() int {
	// Check pane_dead first — pane_dead_status is only meaningful when the pane
	// has exited. Without this guard, a running pane returns "0" for
	// pane_dead_status which is indistinguishable from "exited with code 0".
	out, err := tmuxCmd("display-message", "-t", p.sessionName, "-p", "#{pane_dead}:#{pane_dead_status}").Output()
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(out)), ":", 2)
		if len(parts) == 2 && parts[0] == "1" {
			if code, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
				return code
			}
		}
	}
	// Fall back to the attach process. ProcessState is set after Wait() returns,
	// so check it first to avoid calling Wait() twice (which panics).
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
	out, err := tmuxCmd("display-message", "-t", p.sessionName, "-p", "#{pane_pid}").Output()
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

// buildTmuxNewSessionArgs constructs the argv suffix (everything after
// `tmux -L <socket>`) for creating a detached tmux session. spec.Env keys
// are passed as `-e KEY=VAL` so they scope to this specific tmux session;
// shells spawned later inside the session inherit them. This is the only
// reliable way to get per-session env on a long-lived tmux server — the
// server's own environment is frozen at first-session creation time.
func buildTmuxNewSessionArgs(sessionName, workingDir string, spec SessionLaunchSpec) []string {
	args := []string{
		"new-session", "-d",
		"-s", sessionName,
		"-c", workingDir,
		"-x", strconv.Itoa(int(spec.Cols)),
		"-y", strconv.Itoa(int(spec.Rows)),
	}
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", k+"="+spec.Env[k])
	}
	args = append(args, spec.Shell)
	return args
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

// resolveTmuxSocket returns the tmux socket name used for web-console
// persistent sessions. Tests override this to isolate themselves from the live
// app's persistent-session server; production keeps the stable default.
func resolveTmuxSocket() string {
	if socket := strings.TrimSpace(os.Getenv("WC_TMUX_SOCKET")); socket != "" {
		return socket
	}
	return defaultTmuxSocket
}

// resolveTmuxScopeName returns the systemd scope name used when booting the
// tmux server. Tests may override this to avoid colliding with the live app's
// scope unit while still exercising the real tmux path.
func resolveTmuxScopeName() string {
	if scope := strings.TrimSpace(os.Getenv("WC_TMUX_SCOPE_NAME")); scope != "" {
		return scope
	}
	return defaultTmuxScopeName
}

// tmuxCmd builds a tmux command that uses the dedicated web-console socket.
// All tmux operations MUST go through this helper to ensure they target the
// correct server (isolated from the parent service's cgroup and test sockets).
func tmuxCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"-L", resolveTmuxSocket()}, args...)
	return exec.Command("tmux", fullArgs...)
}

// tmuxCmdContext is like tmuxCmd but accepts a context for timeout/cancellation.
func tmuxCmdContext(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-L", resolveTmuxSocket()}, args...)
	return exec.CommandContext(ctx, "tmux", fullArgs...)
}

// tmuxPTYFactory creates a tmux-backed PTY for persistent sessions.
func tmuxPTYFactory(spec SessionLaunchSpec) (PTY, error) {
	sessionName := tmuxSessionPrefix + spec.SessionID
	workingDir := resolveWorkingDir()

	// 1. Create detached tmux session with the target shell.
	// We use systemd-run --scope to launch the tmux new-session command in
	// its own systemd scope. On the FIRST invocation (when no tmux server
	// exists), this causes the forked tmux server to live in the scope
	// instead of inheriting the parent's cgroup (e.g., vrooli-autoheal.service).
	// This is critical: without it, a service restart kills the tmux server.
	//
	// On subsequent calls the tmux server is already running (in its own
	// scope); the new-session client just asks it to create a session, and
	// the scope only wraps the short-lived client — which is harmless.
	//
	// Setpgid isolates the child from the API's process group so that
	// lifecycle SIGTERM (kill -TERM -$pgid) doesn't kill tmux processes.
	//
	// Session-scoped env vars are injected via `tmux new-session -e KEY=VAL`
	// so panes opened inside THIS session (even when the tmux server was
	// created by a previous session and thus has frozen server env) see the
	// correct WC_WEB_CONSOLE_SESSION_ID / CODEX_HOME. Without this, the
	// second and later sessions on the same tmux server would inherit the
	// first session's attribution vars, breaking conversation tracking.
	sessionArgs := buildTmuxNewSessionArgs(sessionName, workingDir, spec)
	socketName := resolveTmuxSocket()
	createCmd := exec.Command("systemd-run", append([]string{
		"--user", "--scope", "--unit=" + resolveTmuxScopeName(),
		"tmux", "-L", socketName,
	}, sessionArgs...)...)
	createCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	createCmd.Env = buildSessionEnv(spec)
	if err := createCmd.Run(); err != nil {
		// Fallback: if systemd-run fails (e.g., no systemd user session),
		// create directly. The server will inherit the parent cgroup, but
		// that's better than failing entirely.
		log.Printf("tmux: systemd-run scope creation failed, falling back to direct: %v", err)
		fallbackCmd := tmuxCmd(sessionArgs...)
		fallbackCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		fallbackCmd.Env = buildSessionEnv(spec)
		if err := fallbackCmd.Run(); err != nil {
			return nil, fmt.Errorf("tmux new-session: %w", err)
		}
	}

	// 2. Configure session options
	applyTmuxOptions(sessionName)

	// 3. Attach to tmux session via a PTY for I/O streaming
	p, err := tmuxAttach(sessionName)
	if err != nil {
		_ = tmuxCmd("kill-session", "-t", sessionName).Run()
		return nil, err
	}
	return p, nil
}

// applyTmuxOptions configures session options (mouse mode, scrollback, titles).
// Errors are logged but not fatal — a session with missing options is still
// usable, and callers need to know something went wrong for debugging.
func applyTmuxOptions(sessionName string) {
	opts := [][2]string{
		{"remain-on-exit", "on"},
		{"mouse", "on"},
		{"history-limit", "50000"},
		{"set-titles", "on"},
		{"set-titles-string", "#{pane_title}"},
	}
	for _, opt := range opts {
		if err := tmuxCmd("set-option", "-t", sessionName, opt[0], opt[1]).Run(); err != nil {
			log.Printf("tmux: set-option %s=%s on %s failed: %v", opt[0], opt[1], sessionName, err)
		}
	}
}

// tmuxAttachTimeout bounds how long we wait for `tmux attach-session` to start.
// If the tmux server is hung or saturated, this prevents recovery goroutines
// from blocking indefinitely.
const tmuxAttachTimeout = 10 * time.Second

// tmuxAttach connects to an existing tmux session for I/O.
// Used by both tmuxPTYFactory (new sessions) and recovery (existing sessions).
func tmuxAttach(sessionName string) (*tmuxPTY, error) {
	// Verify the session exists and tmux responds before committing to attach.
	// This fails fast if the tmux server is hung or the session was killed.
	ctx, cancel := context.WithTimeout(context.Background(), tmuxAttachTimeout)
	defer cancel()
	hasCmd := tmuxCmdContext(ctx, "has-session", "-t", sessionName)
	hasCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := hasCmd.Run(); err != nil {
		return nil, fmt.Errorf("tmux has-session %s: %w", sessionName, err)
	}

	attachCmd := tmuxCmd("attach-session", "-t", sessionName)
	// NOTE: No Setpgid here — pty.Start() below sets Setsid=true internally,
	// which already puts the attach process in its own session+process group,
	// isolating it from lifecycle SIGTERM (kill -TERM -$pgid). Setting both
	// Setpgid and Setsid causes EPERM because a session leader cannot setpgid.
	//
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

// tmuxAttachAsPTY wraps tmuxAttach to return the PTY interface, matching
// TmuxAttachFunc's signature so production code and tests use the same type.
func tmuxAttachAsPTY(sessionName string) (PTY, error) {
	return tmuxAttach(sessionName)
}

// DiscoverTmuxSessions finds surviving web console tmux sessions.
// Returns session IDs (without the "wc-" prefix).
func DiscoverTmuxSessions() ([]string, error) {
	out, err := tmuxCmd("list-sessions", "-F", "#{session_name}").Output()
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
