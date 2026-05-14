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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	creackpty "github.com/creack/pty/v2"

	"web-console/backends/claude"
	"web-console/internal/config"
	"web-console/internal/pty"
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

// WriteInput delivers bytes to the tmux pane using the kind-appropriate
// mode-safe path. Writing to the attach PTY master directly (as the old
// code did) is NOT safe for text input: tmux interprets those bytes as
// CLIENT input, meaning any payload sent while the client is in
// copy-mode / command-prompt / menu / prefix-pending is consumed as a
// tmux command and never reaches the pane's running program. This was
// the root cause of the long-standing "message is lost, Ctrl+C
// unblocks it" bug.
//
//   - pty.KindKeystroke: `tmux send-keys -t <session> -l -- <data>`.
//     The `-l` (literal) flag tells tmux to deliver the bytes to the
//     active pane's stdin verbatim, bypassing key-name lookup AND
//     client-mode interpretation. This path handles arbitrary byte
//     sequences including control characters.
//     EXCEPTION: mouse-tracking CSI sequences (wheel scroll, click,
//     drag) are intentionally routed through the attach PTY master so
//     that the tmux client can interpret them at the client level
//     (enter copy-mode on scroll, select text, etc.). Routing these
//     via send-keys would deliver them to the pane's shell, which
//     does not understand them, and mobile scroll would silently
//     break. See isMouseTrackingSequence.
//   - pty.KindPaste: `tmux load-buffer -b <buf> -` (piped stdin) then
//     `tmux paste-buffer -d -b <buf> -t <session>`. The `-d` flag
//     deletes the buffer after paste so our per-session buffers don't
//     accumulate. The buffer name is scoped per-session with a
//     per-call counter to guarantee no cross-call collision.
//
// Both non-mouse branches surface tmux's stderr in the returned error
// so the caller can forward it as stdin_ack.reason.
func (p *tmuxPTY) WriteInput(data []byte, kind pty.InputKind) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errPTYClosed
	}
	sessionName := p.sessionName
	ptmx := p.ptmx
	p.mu.Unlock()

	if len(data) == 0 {
		return nil
	}

	// Mouse-tracking CSI sequences (scroll wheel, click, drag) must
	// reach the tmux client — not the pane's shell — so tmux can
	// interpret them (enter copy-mode on wheel-up, select text on
	// drag, etc.). Bypass the mode-aware send-keys path entirely for
	// these. This preserves mobile scroll and desktop mouse-select in
	// tmux-backed sessions. Paste payloads never qualify because the
	// paste kind is only used for clipboard data, not xterm events.
	if kind == pty.KindKeystroke && isMouseTrackingSequence(data) {
		if _, err := ptmx.Write(data); err != nil {
			return fmt.Errorf("tmux mouse passthrough write: %w", err)
		}
		return nil
	}

	switch kind {
	case pty.KindPaste:
		return p.deliverPaste(sessionName, data)
	default:
		return p.deliverKeystroke(sessionName, data)
	}
}

// isMouseTrackingSequence returns true if data begins with an xterm
// mouse-tracking CSI introducer. These must be delivered to the tmux
// client (via the attach PTY master) rather than routed through
// `send-keys` into the pane, because the client is what interprets
// scroll / click / drag events. Two forms are recognized:
//
//   - X10 / VT200: `\x1b[M` followed by 3 encoding bytes.
//   - SGR (mode 1006): `\x1b[<Cb;Cx;Cy{M|m}`.
//
// Arrow keys (`\x1b[A..D`) and function keys (`\x1b[11~`, etc.) do
// NOT match — data[2] must be 'M' or '<'. URXVT mode is not emitted
// by xterm.js and is not handled here; if we ever need it we can add
// a digit-based introducer check.
func isMouseTrackingSequence(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	if data[0] != 0x1b || data[1] != '[' {
		return false
	}
	return data[2] == 'M' || data[2] == '<'
}

// exitModeIfAny ensures the pane is NOT in copy-mode / command-prompt /
// menu / any tmux mode before subsequent input is delivered. `send-keys
// -l` and `paste-buffer` both respect the current client mode: if the
// pane is in copy-mode, the bytes are interpreted as mode commands
// rather than delivered to the pane's running program. This was the
// root cause of the "Ctrl+C unblocks lost input" bug.
//
// We query `#{pane_in_mode}` first because `send-keys -X cancel`
// returns a non-zero exit status and prints "not in a mode" when the
// pane isn't in a mode, which would otherwise be surfaced as a
// spurious stdin_ack.ok=false. Skipping the cancel when not needed is
// also the hot path (no mode is the steady state).
func (p *tmuxPTY) exitModeIfAny(sessionName string) error {
	out, err := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_in_mode}").Output()
	if err != nil {
		// If display-message fails the session likely died; surface
		// it as a typed write failure via the caller.
		return fmt.Errorf("tmux display-message: %w", err)
	}
	if strings.TrimSpace(string(out)) != "1" {
		return nil
	}
	cancelCmd := tmuxCmd("send-keys", "-t", sessionName, "-X", "cancel")
	if cancelOut, cancelErr := cancelCmd.CombinedOutput(); cancelErr != nil {
		// Treat as fatal: if we can't exit the mode, subsequent input
		// will be eaten by the mode. Better to surface the failure.
		return fmt.Errorf("tmux send-keys -X cancel: %w (%s)",
			cancelErr, strings.TrimSpace(string(cancelOut)))
	}
	return nil
}

// deliverKeystroke sends data via `tmux send-keys -t <target> -l --`.
// Large keystroke payloads fall through to the paste-buffer path to
// avoid argv size limits. Before delivery, any tmux mode on the pane
// is cancelled so the bytes reach the running program rather than
// being interpreted as mode commands.
func (p *tmuxPTY) deliverKeystroke(sessionName string, data []byte) error {
	// Oversized keystrokes (unusual but possible — e.g. voice
	// transcription producing very long text) go through the paste
	// path, which is buffer-limited only by tmux's own buffer cap,
	// not by argv size. deliverPaste also cancels mode.
	const maxArgvChunk = 64 * 1024
	if len(data) > maxArgvChunk {
		return p.deliverPaste(sessionName, data)
	}
	if err := p.exitModeIfAny(sessionName); err != nil {
		return err
	}
	cmd := tmuxCmd("send-keys", "-t", sessionName, "-l", "--", string(data))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux send-keys failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// tmuxPasteBufferSeq increments per call to produce unique buffer
// names, avoiding collisions when paste calls overlap for the same
// session (rare, but possible under test harness or retry paths).
var tmuxPasteBufferSeq uint64

// deliverPaste pipes data into a dedicated tmux buffer and then pastes
// it into the session's pane. Before delivery, any tmux mode on the
// pane is cancelled so the payload reaches the running program. The
// `-d` flag on paste-buffer deletes the buffer after delivery so our
// per-call buffers don't accumulate.
func (p *tmuxPTY) deliverPaste(sessionName string, data []byte) error {
	if err := p.exitModeIfAny(sessionName); err != nil {
		return err
	}

	seq := atomic.AddUint64(&tmuxPasteBufferSeq, 1)
	buf := fmt.Sprintf("wc-paste-%s-%d", sessionName, seq)

	loadCmd := tmuxCmd("load-buffer", "-b", buf, "-")
	loadCmd.Stdin = bytes.NewReader(data)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux load-buffer failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	pasteCmd := tmuxCmd("paste-buffer", "-d", "-b", buf, "-t", sessionName)
	if out, err := pasteCmd.CombinedOutput(); err != nil {
		// Best-effort cleanup of the orphaned buffer; ignore errors.
		_ = tmuxCmd("delete-buffer", "-b", buf).Run()
		return fmt.Errorf("tmux paste-buffer failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
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

func (p *tmuxPTY) CurrentDir(_ context.Context) (string, error) {
	out, err := tmuxCmd("display-message", "-t", p.sessionName, "-p", "#{pane_current_path}").Output()
	if err != nil {
		return "", fmt.Errorf("tmux display-message cwd: %w", err)
	}
	cwd := strings.TrimSpace(string(out))
	if cwd == "" {
		return "", fmt.Errorf("tmux current path is empty")
	}
	return filepath.Clean(cwd), nil
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
	return creackpty.Setsize(p.ptmx, &creackpty.Winsize{Rows: rows, Cols: cols})
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
func buildTmuxNewSessionArgs(sessionName, workingDir string, spec pty.LaunchSpec) []string {
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
func buildSessionEnv(spec pty.LaunchSpec) []string {
	return applySessionEnv(
		ensureTermEnv(
			filterServiceEnv(
				claude.FilterEnv(os.Environ()),
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
func tmuxPTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	sessionName := tmuxSessionPrefix + spec.SessionID
	workingDir := config.ResolveWorkingDir()

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
	ptmx, err := creackpty.Start(attachCmd)
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
// session.TmuxAttachFunc's signature so production code and tests use the same type.
func tmuxAttachAsPTY(sessionName string) (pty.PTY, error) {
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
