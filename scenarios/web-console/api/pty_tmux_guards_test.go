package main

import (
	"io"
	"testing"
	"web-console/internal/pty"
)

// --- tmuxPTY closed-state guard tests ---
// These verify that Read/Write/SetSize return proper errors after Close()
// instead of panicking on a closed file descriptor.

func TestTmuxPTY_ReadAfterClose(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-read-after-close",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Close the PTY
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read should return EOF, not panic
	buf := make([]byte, 128)
	_, err = p.Read(buf)
	if err != io.EOF {
		t.Errorf("Read after Close: got err=%v, want io.EOF", err)
	}
}

func TestTmuxPTY_WriteAfterClose(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-write-after-close",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// WriteInput should return errPTYClosed, not panic
	err = p.WriteInput([]byte("hello"), pty.KindKeystroke)
	if err != errPTYClosed {
		t.Errorf("WriteInput after Close: got err=%v, want errPTYClosed", err)
	}
}

func TestTmuxPTY_SetSizeAfterClose(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-setsize-after-close",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// SetSize should return errPTYClosed, not spawn orphan tmux commands
	err = p.SetSize(120, 40)
	if err != errPTYClosed {
		t.Errorf("SetSize after Close: got err=%v, want errPTYClosed", err)
	}
}

func TestTmuxPTY_CloseIdempotent(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-close-idempotent",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}

	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Close twice should not panic
	if err := p.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("second Close: %v (expected nil)", err)
	}
}

func TestTmuxAttach_FailsForNonexistentSession(t *testing.T) {
	requireIsolatedTmux(t)

	_, err := tmuxAttach("wc-nonexistent-session-12345")
	if err == nil {
		t.Fatal("expected error attaching to nonexistent session")
	}
}

func TestApplyTmuxOptions_NonexistentSession(t *testing.T) {
	requireIsolatedTmux(t)

	// Should not panic, just log errors
	applyTmuxOptions("wc-nonexistent-session-12345")
}
