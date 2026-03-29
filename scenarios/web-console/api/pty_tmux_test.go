package main

import (
	"os/exec"
	"strings"
	"testing"
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
	defer p.Kill()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Query the tmux mouse option for this session
	out, err := exec.Command("tmux", "show-options", "-t", sessionName, "mouse").Output()
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
	defer p.Kill()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Query the tmux history-limit for this session
	out, err := exec.Command("tmux", "show-options", "-t", sessionName, "history-limit").Output()
	if err != nil {
		t.Fatalf("tmux show-options failed: %v", err)
	}

	got := strings.TrimSpace(string(out))
	if got != "history-limit 50000" {
		t.Errorf("expected tmux history-limit to be 'history-limit 50000', got %q", got)
	}
}
