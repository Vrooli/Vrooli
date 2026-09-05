package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"web-console/internal/config"
	"web-console/internal/pty"
)

// TestBuildTmuxNewSessionArgs_InjectsSessionEnv locks in the invariant
// that attribution env vars from pty.LaunchSpec.Env are rendered as
// `-e KEY=VAL` flags on `tmux new-session`. Without this, panes in
// later sessions on a long-lived tmux server inherit the server's
// frozen env (first session's WC_WEB_CONSOLE_SESSION_ID) and mid-session
// `claude`/`codex` invocations get mis-attributed.
func TestBuildTmuxNewSessionArgs_InjectsSessionEnv(t *testing.T) {
	spec := pty.LaunchSpec{
		SessionID: "sess-b",
		Shell:     "/bin/sh",
		Cols:      120,
		Rows:      40,
		Env: map[string]string{
			"WC_WEB_CONSOLE_SESSION_ID": "sess-b",
			"CODEX_HOME":                "/tmp/codex-b",
			"WC_CODEX_SESSIONS_DIR":     "/tmp/codex-b/sessions",
		},
	}
	got := buildTmuxNewSessionArgs("wc-sess-b", "/workdir", spec)
	want := []string{
		"new-session", "-d",
		"-s", "wc-sess-b",
		"-c", "/workdir",
		"-x", "120",
		"-y", "40",
		"-e", "CODEX_HOME=/tmp/codex-b",
		"-e", "WC_CODEX_SESSIONS_DIR=/tmp/codex-b/sessions",
		"-e", "WC_WEB_CONSOLE_SESSION_ID=sess-b",
		"/bin/sh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestBuildTmuxNewSessionArgs_NoEnvIsUnchanged(t *testing.T) {
	spec := pty.LaunchSpec{
		SessionID: "sess-empty",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	got := buildTmuxNewSessionArgs("wc-sess-empty", "/workdir", spec)
	want := []string{
		"new-session", "-d",
		"-s", "wc-sess-empty",
		"-c", "/workdir",
		"-x", "80",
		"-y", "24",
		"/bin/sh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestTmuxPTYFactory_UsesLaunchSpecWorkingDir(t *testing.T) {
	spec := pty.LaunchSpec{
		SessionID:  "cwd-override",
		Shell:      "/bin/sh",
		Cols:       80,
		Rows:       24,
		WorkingDir: "/recovered/project",
	}
	got := buildTmuxNewSessionArgs("wc-cwd-override", spec.WorkingDir, spec)
	if got[5] != "/recovered/project" {
		t.Fatalf("tmux working dir = %q, want recovered cwd", got[5])
	}
}

func TestTmuxPTYTerminalEchoStateTracksPaneEcho(t *testing.T) {
	requireTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "echo-state",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	opened, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	p := opened.(*tmuxPTY)
	defer func() {
		_ = p.WriteInput([]byte("stty echo\n"), pty.KindKeystroke)
		_ = p.Kill()
	}()

	state, err := p.TerminalEchoState()
	if err != nil || !state.Known || !state.EchoEnabled {
		t.Fatalf("initial echo state = %+v, err=%v; want known/enabled", state, err)
	}
	if err := p.WriteInput([]byte("stty -echo\n"), pty.KindKeystroke); err != nil {
		t.Fatalf("disable echo: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		state, err = p.TerminalEchoState()
		if err == nil && state.Known && !state.EchoEnabled {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("echo state did not become disabled: %+v, err=%v", state, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// TestTmuxPTYFactory_PropagatesSessionEnvIntoPane is the end-to-end
// guarantee for mid-session attribution inside tmux: a pane running in
// this session must see WC_WEB_CONSOLE_SESSION_ID set to THIS session's
// id, even when the tmux server was created by a different session
// beforehand (which is the common case — the server is long-lived and
// shared). This catches regressions if someone drops `-e` from the
// new-session args.
func TestTmuxPTYFactory_PropagatesSessionEnvIntoPane(t *testing.T) {
	requireTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-env-propagation",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
		Env: map[string]string{
			"WC_WEB_CONSOLE_SESSION_ID": "test-env-propagation",
			"CODEX_HOME":                "/tmp/codex-test-env-propagation",
			"WC_CODEX_SESSIONS_DIR":     "/tmp/codex-test-env-propagation/sessions",
		},
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	for _, want := range []string{
		"WC_WEB_CONSOLE_SESSION_ID=test-env-propagation",
		"CODEX_HOME=/tmp/codex-test-env-propagation",
		"WC_CODEX_SESSIONS_DIR=/tmp/codex-test-env-propagation/sessions",
	} {
		key := strings.SplitN(want, "=", 2)[0]
		out, err := tmuxCmd("show-environment", "-t", sessionName, key).Output()
		if err != nil {
			t.Fatalf("tmux show-environment %s: %v", key, err)
		}
		got := strings.TrimSpace(string(out))
		if got != want {
			t.Errorf("session env %s: got %q, want %q", key, got, want)
		}
	}
}

// TestTmuxPTYFactory_DisablesMouseMode verifies that persistent panes default
// to local xterm scrolling. Mouse mode is an explicit per-session choice;
// enabling it is deliberately not an implicit side effect of persistence.
func TestTmuxPTYFactory_DisablesMouseMode(t *testing.T) {
	requireTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-mouse-mode",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Query the tmux mouse option for this session
	out, err := tmuxCmd("show-options", "-t", sessionName, "mouse").Output()
	if err != nil {
		t.Fatalf("tmux show-options failed: %v", err)
	}

	got := strings.TrimSpace(string(out))
	if got != "mouse off" {
		t.Errorf("expected tmux mouse option to be 'mouse off', got %q", got)
	}
}

func TestTmuxPTYFactory_UsesRequestedMouseMode(t *testing.T) {
	requireTmux(t)

	p, err := tmuxPTYFactory(pty.LaunchSpec{
		SessionID:     "test-mouse-mode-on",
		Shell:         "/bin/sh",
		Cols:          80,
		Rows:          24,
		TmuxMouseMode: true,
	})
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()
	sessionName := tmuxSessionPrefix + "test-mouse-mode-on"
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	out, err := tmuxCmd("show-options", "-t", sessionName, "mouse").Output()
	if err != nil {
		t.Fatalf("tmux show-options failed: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "mouse on" {
		t.Fatalf("mouse mode = %q, want mouse on", got)
	}

	controller, ok := p.(interface{ SetMouseMode(bool) error })
	if !ok {
		t.Fatal("tmux PTY does not expose per-pane mouse control")
	}
	if err := controller.SetMouseMode(false); err != nil {
		t.Fatalf("disable mouse mode: %v", err)
	}
	out, err = tmuxCmd("show-options", "-t", sessionName, "mouse").Output()
	if err != nil {
		t.Fatalf("tmux show-options after toggle failed: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "mouse off" {
		t.Fatalf("mouse mode after toggle = %q, want mouse off", got)
	}
}

// TestTmuxPTYFactory_SetsHistoryLimit verifies that tmuxPTYFactory configures
// a generous scrollback buffer so users can scroll through substantial output.
func TestTmuxPTYFactory_SetsHistoryLimit(t *testing.T) {
	requireTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-history-limit",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Query the tmux history-limit for this session
	out, err := tmuxCmd("show-options", "-t", sessionName, "history-limit").Output()
	if err != nil {
		t.Fatalf("tmux show-options failed: %v", err)
	}

	got := strings.TrimSpace(string(out))
	want := fmt.Sprintf("history-limit %d", config.DefaultTerminalScrollbackLines)
	if got != want {
		t.Errorf("expected tmux history-limit to be %q, got %q", want, got)
	}
}

func TestTmuxPTYFactory_UsesResolvedWorkingDir(t *testing.T) {
	requireTmux(t)

	workingDir := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", workingDir)
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("SCENARIO_DIR", "")

	spec := pty.LaunchSpec{
		SessionID: "test-working-dir",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	out, err := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_current_path}").Output()
	if err != nil {
		t.Fatalf("tmux display-message failed: %v", err)
	}

	got := strings.TrimSpace(string(out))
	if got != workingDir {
		t.Errorf("expected tmux pane path %q, got %q", workingDir, got)
	}
}

// ProbeReady must return nil within the caller's timeout on a freshly
// attached tmux session — the attach process is already wired through,
// so list-clients reports our attach as present.
func TestTmuxPTY_ProbeReady_HappyPath(t *testing.T) {
	requireTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-probe-ready",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()
	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := p.ProbeReady(ctx); err != nil {
		t.Fatalf("ProbeReady on healthy tmux session failed: %v", err)
	}
}

// When the context deadline expires before an attach pipeline completes,
// ProbeReady must surface ctx.Err() so the WS handler can emit
// session_not_ready rather than hanging the connection forever.
func TestTmuxPTY_ProbeReady_TimeoutSurfacesCtxErr(t *testing.T) {
	requireTmux(t)

	// Construct a tmuxPTY referencing a session name that does not exist —
	// list-clients will always return empty output, so ProbeReady must loop
	// until ctx expires.
	p := &tmuxPTY{sessionName: "wc-does-not-exist-" + t.Name()}
	// poll interval is package-level; we don't reduce it — a short context
	// deadline (~100 ms) means the test still runs fast.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := p.ProbeReady(ctx)
	if err == nil {
		t.Fatal("ProbeReady on unreachable session must not succeed")
	}
	if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("expected deadline-exceeded-class error, got %v", err)
	}
}

// writeTmuxPaneScript creates an executable script for a pane to run. tmux
// takes a single command argument, so a multi-step pane program has to live in
// a file rather than a quoted shell string.
func writeTmuxPaneScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pane.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write pane script: %v", err)
	}
	return path
}

// tmuxPaneField reads one tmux format field for a session's active pane.
func tmuxPaneField(t *testing.T, sessionName, format string) string {
	t.Helper()
	out, err := tmuxCmd("display-message", "-t", sessionName, "-p", format).Output()
	if err != nil {
		t.Fatalf("tmux display-message %s: %v", format, err)
	}
	return strings.TrimSpace(string(out))
}

// startScrollablePane brings up a tmux session whose pane has already emitted
// far more output than fits on screen, so copy-mode has real history to move
// through.
func startScrollablePane(t *testing.T, sessionID string) (pty.PTY, string) {
	t.Helper()
	script := writeTmuxPaneScript(t, "i=1\nwhile [ $i -le 200 ]; do echo \"line-$i\"; i=$((i+1)); done\nsleep 300\n")
	p, err := tmuxPTYFactory(pty.LaunchSpec{SessionID: sessionID, Shell: script, Cols: 80, Rows: 10})
	if err != nil {
		t.Fatalf("tmuxPTYFactory failed: %v", err)
	}
	sessionName := tmuxSessionPrefix + sessionID
	t.Cleanup(func() {
		_ = p.Kill()
		p.Close()
		_ = tmuxCmd("kill-session", "-t", sessionName).Run()
	})
	// Let the pane produce its output before anything tries to scroll it.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if tmuxPaneField(t, sessionName, "#{history_size}") != "0" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return p, sessionName
}

func requireScroller(t *testing.T, p pty.PTY) interface{ Scroll(int) error } {
	t.Helper()
	scroller, ok := p.(interface{ Scroll(int) error })
	if !ok {
		t.Fatal("tmux PTY does not expose history scrolling")
	}
	return scroller
}

// TestTmuxPTY_Scroll_MovesPaneHistory is the server half of the "wheel does not
// scroll a Codex pane" fix. A tmux client parks the browser terminal in the
// alternate screen buffer for the whole session, so the browser has no
// scrollback to move; the pane's own tmux history is the only history there
// is, and the server has to walk it on the client's behalf.
func TestTmuxPTY_Scroll_MovesPaneHistory(t *testing.T) {
	requireTmux(t)
	p, sessionName := startScrollablePane(t, "test-scroll-history")
	scroller := requireScroller(t, p)

	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "0" {
		t.Fatalf("precondition: pane already in a mode: %q", got)
	}

	// Scrolling back enters copy-mode on demand.
	if err := scroller.Scroll(-5); err != nil {
		t.Fatalf("Scroll(-5): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "1" {
		t.Fatalf("pane_in_mode after scroll back = %q, want 1", got)
	}
	if got := tmuxPaneField(t, sessionName, "#{scroll_position}"); got != "5" {
		t.Fatalf("scroll_position after Scroll(-5) = %q, want 5", got)
	}

	// A second scroll accumulates rather than re-entering copy-mode at the
	// bottom, which is what makes a continuous gesture work.
	if err := scroller.Scroll(-5); err != nil {
		t.Fatalf("second Scroll(-5): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{scroll_position}"); got != "10" {
		t.Fatalf("scroll_position after two scrolls = %q, want 10", got)
	}

	// Scrolling forward walks back toward live output.
	if err := scroller.Scroll(3); err != nil {
		t.Fatalf("Scroll(3): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{scroll_position}"); got != "7" {
		t.Fatalf("scroll_position after Scroll(3) = %q, want 7", got)
	}
}

// TestTmuxPTY_Scroll_ForwardToBottomReturnsPaneToLive asserts the pane does not
// get stranded in copy-mode once the operator scrolls back to the bottom.
// tmux clamps scroll-down at the live edge and stays in the mode; leaving it
// there would make the operator's next keystroke spend itself cancelling.
func TestTmuxPTY_Scroll_ForwardToBottomReturnsPaneToLive(t *testing.T) {
	requireTmux(t)
	p, sessionName := startScrollablePane(t, "test-scroll-to-live")
	scroller := requireScroller(t, p)

	if err := scroller.Scroll(-4); err != nil {
		t.Fatalf("Scroll(-4): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "1" {
		t.Fatalf("precondition: pane not in copy-mode: %q", got)
	}

	if err := scroller.Scroll(50); err != nil {
		t.Fatalf("Scroll(50): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "0" {
		t.Fatalf("pane_in_mode after scrolling to the bottom = %q, want 0", got)
	}
}

// TestTmuxPTY_Scroll_ForwardWhenLiveIsNoop asserts forward scrolling never
// drags a live pane into copy-mode. Entering a mode the operator did not ask
// for would swallow their next keystroke to cancel it.
func TestTmuxPTY_Scroll_ForwardWhenLiveIsNoop(t *testing.T) {
	requireTmux(t)
	p, sessionName := startScrollablePane(t, "test-scroll-live-noop")
	scroller := requireScroller(t, p)

	if err := scroller.Scroll(5); err != nil {
		t.Fatalf("Scroll(5) on a live pane: %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "0" {
		t.Fatalf("pane_in_mode after forward scroll on a live pane = %q, want 0", got)
	}
}

func TestTmuxPTY_Scroll_ZeroIsNoop(t *testing.T) {
	requireTmux(t)
	p, sessionName := startScrollablePane(t, "test-scroll-zero")
	scroller := requireScroller(t, p)

	if err := scroller.Scroll(0); err != nil {
		t.Fatalf("Scroll(0): %v", err)
	}
	if got := tmuxPaneField(t, sessionName, "#{pane_in_mode}"); got != "0" {
		t.Fatalf("pane_in_mode after Scroll(0) = %q, want 0", got)
	}
}

// TestTmuxPTY_PaneInAltScreen_ReportsPaneNotClient locks in the distinction
// that the terminal emulator cannot make. tmux emits `\x1b[?1049h` to its
// client on attach, so an emulator reading the attach stream reports
// "alternate buffer" for every persistent pane regardless of what the pane
// runs. Only the pane's own alternate_on separates a full-screen program from
// a shell, and predictive echo is gated on that distinction.
func TestTmuxPTY_PaneInAltScreen_ReportsPaneNotClient(t *testing.T) {
	requireTmux(t)

	cases := []struct {
		name      string
		sessionID string
		body      string
		want      bool
	}{
		{
			name:      "plain program stays on the normal screen",
			sessionID: "test-alt-screen-normal",
			body:      "sleep 300\n",
			want:      false,
		},
		{
			name:      "full-screen program is reported as alternate",
			sessionID: "test-alt-screen-alternate",
			body:      "printf '\\033[?1049h'\nsleep 300\n",
			want:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := writeTmuxPaneScript(t, tc.body)
			p, err := tmuxPTYFactory(pty.LaunchSpec{SessionID: tc.sessionID, Shell: script, Cols: 80, Rows: 24})
			if err != nil {
				t.Fatalf("tmuxPTYFactory failed: %v", err)
			}
			sessionName := tmuxSessionPrefix + tc.sessionID
			defer func() {
				_ = p.Kill()
				p.Close()
				_ = tmuxCmd("kill-session", "-t", sessionName).Run()
			}()

			reader, ok := p.(interface{ PaneInAltScreen() (bool, error) })
			if !ok {
				t.Fatal("tmux PTY does not expose the pane alternate-screen state")
			}

			// The pane program needs a moment to emit its mode switch.
			var got bool
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				got, err = reader.PaneInAltScreen()
				if err != nil {
					t.Fatalf("PaneInAltScreen: %v", err)
				}
				if got == tc.want {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if got != tc.want {
				t.Fatalf("PaneInAltScreen() = %v, want %v", got, tc.want)
			}
		})
	}
}
