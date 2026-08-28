//go:build !windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"web-console/backends/claude"
	"web-console/internal/config"
	"web-console/internal/pty"
	"web-console/session"

	creackpty "github.com/creack/pty/v2"
	platform "github.com/vrooli/platform-go"
)

// errPTYClosed is returned when I/O is attempted on a closed tmuxPTY.
var errPTYClosed = session.ErrPTYClosed

var systemdRunProbe = struct {
	sync.Once
	available bool
}{}

func systemdRunUsable() bool {
	systemdRunProbe.Do(func() {
		if runtime.GOOS == "linux" {
			_, err := exec.LookPath("systemd-run")
			systemdRunProbe.available = err == nil
		}
		log.Printf("tmux capability: systemd-run=%t (goos=%s)", systemdRunProbe.available, runtime.GOOS)
	})
	return systemdRunProbe.available
}

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
	control     *tmuxControl
	mu          sync.Mutex
	closed      bool
}

func (p *tmuxPTY) TerminalEchoState() (session.EchoState, error) {
	// The attach PTY has its own termios. Query the pane's slave tty instead;
	// that is the terminal whose ECHO bit the shell/application changes for a
	// password prompt. The tty path is supplied by tmux and is validated before
	// it reaches stty so this remains a data query, never a shell command.
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	paneOut, err := p.tmuxOutput(ctx, "display-message", "-t", p.sessionName, "-p", "#{pane_tty}")
	if err != nil {
		return session.EchoState{}, fmt.Errorf("tmux pane tty: %w", err)
	}
	tty := strings.TrimSpace(string(paneOut))
	if !strings.HasPrefix(tty, "/dev/") || strings.ContainsAny(tty, " \t\r\n") {
		return session.EchoState{}, fmt.Errorf("tmux returned invalid pane tty %q", tty)
	}
	// The control channel supplies the pane tty path. Read its termios
	// directly so echo sampling remains a channel request plus ioctl: no
	// per-sample stty process is spawned.
	return readPTYEchoPath(tty)
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
//   - pty.KindControl: raw bytes to the attach PTY master. These are
//     best-effort client controls, with no mode
//     cancellation or replay semantics.
//   - pty.KindKeystroke: `tmux send-keys -t <session> -l -- <data>`.
//     The `-l` (literal) flag tells tmux to deliver the bytes to the
//     active pane's stdin verbatim, bypassing key-name lookup AND
//     client-mode interpretation. This path handles arbitrary byte
//     sequences including control characters.
//   - pty.KindPaste: `tmux load-buffer -b <buf> -` (piped stdin) then
//     `tmux paste-buffer -d -p -b <buf> -t <session>`. The `-d` flag
//     deletes the buffer after paste so our per-session buffers don't
//     accumulate; `-p` lets tmux add bracketed-paste markers when the
//     pane's application asked for them. The buffer name is scoped
//     per-session with a per-call counter to guarantee no cross-call
//     collision.
//
// Size, not kind, chooses the transport: keystroke payloads above
// maxKeystrokeArgvBytes cannot fit in a tmux command and fall back to
// the same buffer mechanism, minus the bracketing. Keep those two
// decisions separate — conflating them is what made a single wrong
// constant silently drop every paste between 16 KiB and 64 KiB.
//
// Both input branches surface tmux's stderr in the returned error
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

	if kind == pty.KindControl {
		if _, err := ptmx.Write(data); err != nil {
			return fmt.Errorf("tmux control passthrough write: %w", err)
		}
		return nil
	}

	switch kind {
	case pty.KindPaste:
		return p.deliverBulkText(sessionName, data)
	default:
		return p.deliverTyping(sessionName, data)
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	out, err := p.tmuxOutput(ctx, "display-message", "-t", sessionName, "-p", "#{pane_in_mode}")
	if err != nil {
		// If display-message fails the session likely died; surface
		// it as a typed write failure via the caller.
		return fmt.Errorf("tmux display-message: %w", err)
	}
	if strings.TrimSpace(string(out)) != "1" {
		return nil
	}
	cancelOut, cancelErr := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "-X", "cancel")
	if cancelErr != nil {
		// Treat as fatal: if we can't exit the mode, subsequent input
		// will be eaten by the mode. Better to surface the failure.
		return fmt.Errorf("tmux send-keys -X cancel: %w (%s)", cancelErr, strings.TrimSpace(cancelOut))
	}
	return nil
}

// maxKeystrokeArgvBytes is the largest payload we hand to `send-keys`
// as a command argument.
//
// The binding constraint is NOT exec(2)'s argv limit — `tmux send-keys`
// does not exec anything. The tmux client packs the whole command
// (name, flags, and every argument) into a single imsg, which is capped
// at MAX_IMSGSIZE = 16 KiB. Past that tmux rejects the command outright
// with "command too long"; no quoting or splitting of the argument
// works around it. Measured ceiling against tmux 3.4 with a realistic
// `wc-<uuid>` target is 16,304 bytes.
//
// 8 KiB is deliberately conservative: it leaves headroom for the
// command name, flags, and session names longer than a UUID (tests use
// descriptive ones) without recomputing a budget on the typing hot
// path. Anything larger takes the buffer path, which streams via stdin
// and is not subject to the command ceiling at all.
//
// TestTmuxSendKeysCeiling_IsAboveOurThreshold measures what the
// installed tmux actually accepts and fails if this constant is not
// safely below it — so a tmux change that tightens the limit surfaces
// as a red test rather than as silently dropped user input.
const maxKeystrokeArgvBytes = 8 * 1024

const tmuxCommandTimeout = 5 * time.Second

func tmuxCmdWithTimeout(args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	return tmuxCmdContext(ctx, args...), cancel
}

// deliverTyping sends data via `tmux send-keys -t <target> -l --`.
// Payloads too large for a single tmux command fall through to the
// buffer path. Before delivery, any tmux mode on the pane is cancelled
// so the bytes reach the running program rather than being interpreted
// as mode commands.
func (p *tmuxPTY) deliverTyping(sessionName string, data []byte) error {
	// Oversized keystroke payloads (a large paste arriving through
	// xterm's onData, or voice transcription producing very long
	// text) go through the buffer path.
	//
	// This is a *transport* fallback and nothing more: the bytes still
	// carry keystroke semantics, so they must not be bracketed. Any
	// bracketed-paste markers the client wanted are already embedded
	// in data — see deliverBuffer's contract.
	if len(data) > maxKeystrokeArgvBytes {
		return p.deliverBuffer(sessionName, data, false)
	}
	if err := p.exitModeIfAny(sessionName); err != nil {
		return err
	}
	return p.deliverControlText(sessionName, data)
}

// deliverControlText keeps control-mode commands line-oriented without
// changing the byte stream delivered to the pane. tmux's Enter key is the
// protocol-safe representation of a newline; all other chunks remain literal
// send-keys payloads.
func (p *tmuxPTY) deliverControlText(sessionName string, data []byte) error {
	chunks := bytes.Split(data, []byte{'\n'})
	for index, chunk := range chunks {
		if len(chunk) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
			out, err := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "-l", "--", string(chunk))
			cancel()
			if err != nil {
				return fmt.Errorf("tmux send-keys failed: %w (%s)", err, strings.TrimSpace(out))
			}
		}
		if index < len(chunks)-1 {
			ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
			out, err := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "Enter")
			cancel()
			if err != nil {
				return fmt.Errorf("tmux send-keys Enter failed: %w (%s)", err, strings.TrimSpace(out))
			}
		}
	}
	return nil
}

// tmuxPasteBufferSeq increments per call to produce unique buffer
// names, avoiding collisions when paste calls overlap for the same
// session (rare, but possible under test harness or retry paths).
var tmuxPasteBufferSeq uint64

// deliverBulkText delivers bulk text that arrived as raw text — the
// context-menu paste path reads navigator.clipboard directly, so the
// payload carries no bracketed-paste markers of its own. tmux supplies
// them, and only when the pane's application has actually requested
// bracketed paste.
func (p *tmuxPTY) deliverBulkText(sessionName string, data []byte) error {
	return p.deliverBuffer(sessionName, data, true)
}

// deliverBuffer pipes data into a dedicated tmux buffer and then pastes
// it into the session's pane. Before delivery, any tmux mode on the
// pane is cancelled so the payload reaches the running program. The
// `-d` flag on paste-buffer deletes the buffer after delivery so our
// per-call buffers don't accumulate.
//
// bracketed selects `paste-buffer -p`, which wraps the payload in
// \e[200~ / \e[201~ if — and only if — the pane's application has
// enabled bracketed paste (DECSET 2004). Callers must set it from the
// *semantics* of the input, never from which transport was chosen:
//
//   - true  for pty.KindPaste: raw clipboard text with no markers.
//     Without this, every newline in a multi-line paste lands on a TUI
//     as a separate Enter press, so an agent submits the first line and
//     treats the rest as follow-up prompts.
//   - false for oversized pty.KindKeystroke payloads. xterm.js already
//     bracketed those itself before they reached the wire (its
//     bracketTextForPaste runs on the browser side), so adding -p here
//     would double-wrap them and leak a literal \e[200~ into the
//     application's input.
func (p *tmuxPTY) deliverBuffer(sessionName string, data []byte, bracketed bool) error {
	if err := p.exitModeIfAny(sessionName); err != nil {
		return err
	}

	seq := atomic.AddUint64(&tmuxPasteBufferSeq, 1)
	buf := fmt.Sprintf("wc-paste-%s-%d", sessionName, seq)

	loadCmd, cancelLoad := tmuxCmdWithTimeout("load-buffer", "-b", buf, "-")
	defer cancelLoad()
	loadCmd.Stdin = bytes.NewReader(data)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux load-buffer failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	pasteArgs := []string{"paste-buffer", "-d"}
	if bracketed {
		pasteArgs = append(pasteArgs, "-p")
	}
	pasteArgs = append(pasteArgs, "-b", buf, "-t", sessionName)
	pasteCmd, cancelPaste := tmuxCmdWithTimeout(pasteArgs...)
	defer cancelPaste()
	if out, err := pasteCmd.CombinedOutput(); err != nil {
		// Best-effort cleanup of the orphaned buffer; ignore errors.
		cleanupCmd, cancelCleanup := tmuxCmdWithTimeout("delete-buffer", "-b", buf)
		_ = cleanupCmd.Run()
		cancelCleanup()
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

		out, err := p.tmuxOutput(ctx, "list-clients", "-t", sessionName, "-F", "#{client_tty}")
		if err == nil && len(strings.TrimSpace(out)) > 0 {
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
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	out, err := p.tmuxOutput(ctx, "display-message", "-t", p.sessionName, "-p", "#{pane_current_path}")
	if err != nil {
		return "", fmt.Errorf("tmux display-message cwd: %w", err)
	}
	cwd := strings.TrimSpace(out)
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
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	if _, err := p.tmuxOutput(ctx, "resize-window", "-t", p.sessionName,
		"-x", strconv.Itoa(int(cols)),
		"-y", strconv.Itoa(int(rows))); err != nil {
		return fmt.Errorf("tmux resize: %w", err)
	}
	// Also resize the local PTY so the attach process knows the new size
	return creackpty.Setsize(p.ptmx, &creackpty.Winsize{Rows: rows, Cols: cols})
}

func (p *tmuxPTY) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	control := p.control
	ptmx := p.ptmx
	p.mu.Unlock()
	if control != nil {
		_ = control.Close()
	}
	return ptmx.Close()
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
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	out, err := p.tmuxOutput(ctx, "display-message", "-t", p.sessionName, "-p", "#{pane_dead}:#{pane_dead_status}")
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(out), ":", 2)
		if len(parts) == 2 && parts[0] == "1" {
			if code, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
				return code
			}
		}
	}
	// Fall back to the attach process. ProcessState is set after Wait() returns,
	// so check it first to avoid calling Wait() twice (which panics).
	return exitCodeOnce(p.cmd)
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
		absolute := socket
		if !filepath.IsAbs(absolute) {
			// tmux sockets are subject to a short UNIX-domain path limit. Test
			// and operator-provided socket labels therefore live under /tmp
			// rather than gaining the scenario working directory prefix.
			absolute = filepath.Join(os.TempDir(), absolute+".sock")
		}
		var err error
		absolute, err = filepath.Abs(absolute)
		if err == nil {
			_ = os.MkdirAll(filepath.Dir(absolute), 0o755)
			return absolute
		}
	}
	path := filepath.Join(resolveSessionStateRoot(), "tmux", defaultTmuxSocket+".sock")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return path
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
	fullArgs := append([]string{"-S", resolveTmuxSocket()}, args...)
	return exec.Command("tmux", fullArgs...)
}

// tmuxCmdContext is like tmuxCmd but accepts a context for timeout/cancellation.
func tmuxCmdContext(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-S", resolveTmuxSocket()}, args...)
	return exec.CommandContext(ctx, "tmux", fullArgs...)
}

// tmuxOutput uses the persistent control client for session operations. The
// nil-control path is intentionally retained for degraded hosts and for
// construction-time operations that happen before an attach client exists.
func (p *tmuxPTY) tmuxOutput(ctx context.Context, args ...string) (string, error) {
	p.mu.Lock()
	control := p.control
	p.mu.Unlock()
	if control != nil {
		return control.Exec(ctx, args...)
	}
	out, err := tmuxCmdContext(ctx, args...).Output()
	return string(out), err
}

// tmuxPTYFactory creates a tmux-backed PTY for persistent sessions.
func tmuxPTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	sessionName := tmuxSessionPrefix + spec.SessionID
	workingDir := resolveLaunchDir(spec)

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
	// correct WC_WEB_CONSOLE_SESSION_ID / WC_SESSION_STATE_ROOT. The native
	// agent launcher derives the per-agent home from those values only when an
	// agent starts. Without the per-session identity, later panes on the same
	// tmux server would inherit the first session's attribution and break
	// conversation tracking.
	sessionArgs := buildTmuxNewSessionArgs(sessionName, workingDir, spec)
	socketName := resolveTmuxSocket()
	if systemdRunUsable() {
		createCmd := exec.Command("systemd-run", append([]string{
			"--user", "--scope", "--unit=" + resolveTmuxScopeName(),
			"tmux", "-S", socketName,
		}, sessionArgs...)...)
		_ = platform.ConfigureCommand(createCmd, platform.ProcessOptions{Detached: true})
		createCmd.Env = buildSessionEnv(spec)
		if output, err := createCmd.CombinedOutput(); err == nil {
			goto configureTmuxSession
		} else if strings.TrimSpace(string(output)) != "" {
			log.Printf("tmux systemd-run create failed: %s", strings.TrimSpace(string(output)))
		}
	}
	{
		fallbackCmd := tmuxCmd(sessionArgs...)
		_ = platform.ConfigureCommand(fallbackCmd, platform.ProcessOptions{Detached: true})
		fallbackCmd.Env = buildSessionEnv(spec)
		if output, err := fallbackCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("tmux new-session: %w (%s)", err, strings.TrimSpace(string(output)))
		}
	}

configureTmuxSession:
	// 2. Configure session options
	applyTmuxOptions(sessionName, spec.TmuxMouseMode)

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
func applyTmuxOptions(sessionName string, mouseMode bool) {
	opts := [][2]string{
		{"remain-on-exit", "on"},
		{"mouse", tmuxMouseModeValue(mouseMode)},
		{"history-limit", strconv.Itoa(config.DefaultTerminalScrollbackLines)},
		{"set-titles", "on"},
		{"set-titles-string", "#{pane_title}"},
	}
	for _, opt := range opts {
		if err := tmuxCmd("set-option", "-t", sessionName, opt[0], opt[1]).Run(); err != nil {
			log.Printf("tmux: set-option %s=%s on %s failed: %v", opt[0], opt[1], sessionName, err)
		}
	}
}

// refreshTmuxOptions reapplies options that are safe to normalize during
// recovery. Mouse mode is deliberately omitted so existing persistent
// sessions retain the choice made when they were created.
func refreshTmuxOptions(sessionName string) {
	for _, opt := range [][2]string{
		{"remain-on-exit", "on"},
		{"history-limit", strconv.Itoa(config.DefaultTerminalScrollbackLines)},
		{"set-titles", "on"},
		{"set-titles-string", "#{pane_title}"},
	} {
		if err := tmuxCmd("set-option", "-t", sessionName, opt[0], opt[1]).Run(); err != nil {
			log.Printf("tmux: set-option %s=%s on %s failed: %v", opt[0], opt[1], sessionName, err)
		}
	}
}

// SetMouseMode changes capture for one running persistent pane.
func (p *tmuxPTY) SetMouseMode(enabled bool) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errPTYClosed
	}
	sessionName := p.sessionName
	p.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	if _, err := p.tmuxOutput(ctx, "set-option", "-t", sessionName, "mouse", tmuxMouseModeValue(enabled)); err != nil {
		return fmt.Errorf("tmux set mouse mode: %w", err)
	}
	return nil
}

// paneInMode reports whether the pane is in copy-mode (or any other tmux
// mode). `send-keys -X` fails with "not in a mode" outside one, so the
// forward-scroll path has to ask before it acts.
func (p *tmuxPTY) paneInMode(ctx context.Context, sessionName string) (bool, error) {
	out, err := p.tmuxOutput(ctx, "display-message", "-t", sessionName, "-p", "#{pane_in_mode}")
	if err != nil {
		return false, fmt.Errorf("tmux display-message pane_in_mode: %w", err)
	}
	return strings.TrimSpace(out) == "1", nil
}

// Scroll moves the pane's copy-mode view through its own history.
//
// Negative lines scroll back toward older output, positive scroll forward
// toward live output, matching the browser's wheel and touch sign convention.
//
// This is the only real scrollback a persistent pane has. A tmux client puts
// the browser terminal into the alternate screen buffer the moment it
// attaches, so the browser's own scrollback can never hold pane history; the
// pane's `history-limit` buffer holds all of it.
func (p *tmuxPTY) Scroll(lines int) error {
	if lines == 0 {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errPTYClosed
	}
	sessionName := p.sessionName
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()

	if lines < 0 {
		// Entering copy-mode when the pane is already in it is a no-op that
		// preserves the current scroll position, so this needs no probe.
		if _, err := p.tmuxOutput(ctx, "copy-mode", "-t", sessionName); err != nil {
			return fmt.Errorf("tmux copy-mode: %w", err)
		}
		if _, err := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "-X",
			"-N", strconv.Itoa(-lines), "scroll-up"); err != nil {
			return fmt.Errorf("tmux scroll-up: %w", err)
		}
		return nil
	}

	// Forward scrolling only means something while scrolled back. Entering
	// copy-mode to scroll down would strand the pane in a mode it was never
	// in, and would swallow the next keystroke to cancel it.
	inMode, err := p.paneInMode(ctx, sessionName)
	if err != nil {
		return err
	}
	if !inMode {
		return nil
	}
	if _, err := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "-X",
		"-N", strconv.Itoa(lines), "scroll-down"); err != nil {
		return fmt.Errorf("tmux scroll-down: %w", err)
	}
	// tmux clamps scroll-down at the live edge and stays in copy-mode. Leaving
	// the pane there would make the next keystroke spend itself cancelling the
	// mode, so returning to the bottom returns the pane to live.
	out, err := p.tmuxOutput(ctx, "display-message", "-t", sessionName, "-p", "#{scroll_position}")
	if err != nil {
		return fmt.Errorf("tmux display-message scroll_position: %w", err)
	}
	if strings.TrimSpace(out) == "0" {
		if _, err := p.tmuxOutput(ctx, "send-keys", "-t", sessionName, "-X", "cancel"); err != nil {
			return fmt.Errorf("tmux cancel copy-mode: %w", err)
		}
	}
	return nil
}

// PaneInAltScreen reports whether the program running *inside* the pane is on
// the alternate screen.
//
// This is deliberately not the emulator's view of the attach stream. tmux
// emits its own `\x1b[?1049h` on attach, so an emulator reading that stream
// reports "alternate buffer" for every tmux session no matter what the pane
// runs. Only the pane's own `alternate_on` distinguishes a full-screen program
// from a shell.
func (p *tmuxPTY) PaneInAltScreen() (bool, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return false, errPTYClosed
	}
	sessionName := p.sessionName
	p.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	out, err := p.tmuxOutput(ctx, "display-message", "-t", sessionName, "-p", "#{alternate_on}")
	if err != nil {
		return false, fmt.Errorf("tmux display-message alternate_on: %w", err)
	}
	return strings.TrimSpace(out) == "1", nil
}

func tmuxMouseModeValue(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func (p *tmuxPTY) MouseMode() (bool, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return false, errPTYClosed
	}
	sessionName := p.sessionName
	p.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), tmuxCommandTimeout)
	defer cancel()
	out, err := p.tmuxOutput(ctx, "show-options", "-t", sessionName, "-v", "mouse")
	if err != nil {
		return false, fmt.Errorf("tmux show mouse mode: %w", err)
	}
	return strings.TrimSpace(out) == "on", nil
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
	_ = platform.ConfigureCommand(hasCmd, platform.ProcessOptions{Detached: true})
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
	p := &tmuxPTY{
		sessionName: sessionName,
		ptmx:        ptmx,
		cmd:         attachCmd,
	}
	control, controlErr := newTmuxControl(sessionName)
	if controlErr != nil {
		// The fork path remains a deliberate degraded-mode fallback. It keeps
		// the terminal usable when tmux control mode is unavailable, while all
		// normal hosts use one persistent command channel.
		log.Printf("tmux control channel unavailable for %s; using fork fallback: %v", sessionName, controlErr)
	} else {
		p.control = control
	}
	return p, nil
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
