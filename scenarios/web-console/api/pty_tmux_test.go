package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestTmuxPTYFactory_EnablesMouseMode verifies that tmuxPTYFactory enables
// the tmux "mouse" option so that mouse wheel scrolling works in xterm.js.
// Without mouse mode, tmux manages its own viewport and xterm.js has no
// scrollback buffer — mouse wheel events are silently discarded.
func TestTmuxPTYFactory_EnablesMouseMode(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	spec := SessionLaunchSpec{
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
	if got != "mouse on" {
		t.Errorf("expected tmux mouse option to be 'mouse on', got %q", got)
	}
}

// TestTmuxPTYFactory_SetsHistoryLimit verifies that tmuxPTYFactory configures
// a generous scrollback buffer so users can scroll through substantial output.
func TestTmuxPTYFactory_SetsHistoryLimit(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	spec := SessionLaunchSpec{
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
	if got != "history-limit 50000" {
		t.Errorf("expected tmux history-limit to be 'history-limit 50000', got %q", got)
	}
}

func TestTmuxPTYFactory_UsesResolvedWorkingDir(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	workingDir := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", workingDir)
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("SCENARIO_DIR", "")

	spec := SessionLaunchSpec{
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
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	spec := SessionLaunchSpec{
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
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

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
