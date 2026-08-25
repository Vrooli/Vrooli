package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"web-console/internal/pty"
)

// TestTmuxPTY_InCopyMode_InputReachesPane is the regression
// test for the long-standing "persistent-mode input is swallowed, need
// to press Ctrl+C to unblock" bug. Before the fix, tmuxPTY.Write wrote
// directly to the attach-session PTY master, meaning any input sent
// while the client was in copy-mode was interpreted as a copy-mode
// motion command instead of being delivered to the pane's program.
// Pressing Ctrl+C exited copy-mode, at which point queued bytes
// resumed reaching the pane (producing the "Ctrl+C unblocks it"
// behavior the user observed).
//
// WriteInput(..., pty.KindKeystroke) routes through
// `tmux send-keys -t <session> -l --`. The `-l` (literal) flag tells
// tmux to deliver the bytes to the pane's stdin regardless of client
// mode. This test asserts that: after entering copy-mode, both a
// keystroke and a paste payload reach the pane's shell.
func TestTmuxPTY_InCopyMode_InputReachesPane(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-copy-mode-input",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Give the shell a moment to finish its prompt rendering before
	// we start asserting on captured pane content.
	time.Sleep(300 * time.Millisecond)

	// Force tmux into copy-mode. In the OLD code path, any subsequent
	// bytes written to the attach PTY master would be consumed as
	// copy-mode commands and never reach the shell.
	if err := tmuxCmd("copy-mode", "-t", sessionName).Run(); err != nil {
		t.Fatalf("enter copy-mode: %v", err)
	}
	// Verify we're actually in copy-mode before proceeding.
	modeOut, err := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_in_mode}").Output()
	if err != nil {
		t.Fatalf("display-message pane_in_mode: %v", err)
	}
	if strings.TrimSpace(string(modeOut)) != "1" {
		t.Fatalf("tmux did not report pane_in_mode=1 after copy-mode command, got %q",
			strings.TrimSpace(string(modeOut)))
	}

	// Send a sentinel command as a keystroke. The shell must execute
	// it (produce its echo in pane output) even though tmux is in
	// copy-mode. Use a distinctive marker and terminate with \n so the
	// shell runs it.
	const marker = "COPY_MODE_MARKER_ae3f71"
	cmd := []byte("echo " + marker + "\n")
	if err := p.WriteInput(cmd, pty.KindKeystroke); err != nil {
		t.Fatalf("WriteInput keystroke in copy-mode: %v", err)
	}

	// `send-keys -l` also exits copy-mode as a side effect on most
	// tmux versions because the pane becomes the input target. Allow
	// a short grace period for the shell to process and emit output.
	if !waitForPaneContent(t, sessionName, marker, 3*time.Second) {
		t.Fatalf("keystroke sent in copy-mode did not reach pane within 3s")
	}

	// Put tmux back into copy-mode and repeat for a PASTE payload.
	// The paste path uses tmux load-buffer + paste-buffer -d which
	// auto-cancels copy-mode by design.
	if err := tmuxCmd("copy-mode", "-t", sessionName).Run(); err != nil {
		t.Fatalf("re-enter copy-mode: %v", err)
	}
	const pasteMarker = "PASTE_MARKER_ae3f71"
	paste := []byte("echo " + pasteMarker + "\n")
	if err := p.WriteInput(paste, pty.KindPaste); err != nil {
		t.Fatalf("WriteInput paste in copy-mode: %v", err)
	}
	if !waitForPaneContent(t, sessionName, pasteMarker, 3*time.Second) {
		t.Fatalf("paste sent in copy-mode did not reach pane within 3s")
	}
}

// waitForPaneContent polls `tmux capture-pane` until the content
// contains the given substring or the deadline elapses. Returns true
// on success. Intended for end-to-end tmux tests where observable
// pane content is the ground truth for "did this byte reach the
// shell".
func waitForPaneContent(t *testing.T, sessionName, want string, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		out, err := tmuxCmd("capture-pane", "-t", sessionName, "-p", "-J").Output()
		if err == nil && bytes.Contains(out, []byte(want)) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// Large-keystroke coverage lives in pty_tmux_payload_size_test.go,
// which asserts byte-exact delivery across both transports instead of
// only that the oversized branch returns no error.

// TestTmuxPTY_MouseWheelPreservesCopyMode is the direct regression
// test for the "mobile scroll broken" bug. The fix for Bug A
// (exitModeIfAny before keystroke delivery) accidentally stripped
// out tmux copy-mode as soon as the user started scrolling, because
// xterm.js emits mouse-wheel events as CSI byte sequences that look
// like stdin to the server. WriteInput must detect those and route
// them to the tmux client directly, preserving whatever mode the
// client has entered.
func TestTmuxPTY_MouseWheelPreservesCopyMode(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-mouse-copy-mode",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	time.Sleep(300 * time.Millisecond)

	// Force the pane into copy-mode to simulate the user having
	// scrolled up.
	if err := tmuxCmd("copy-mode", "-t", sessionName).Run(); err != nil {
		t.Fatalf("enter copy-mode: %v", err)
	}
	before, err := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_in_mode}").Output()
	if err != nil {
		t.Fatalf("display-message (before): %v", err)
	}
	if strings.TrimSpace(string(before)) != "1" {
		t.Fatalf("precondition: pane not in mode; got %q", strings.TrimSpace(string(before)))
	}

	// SGR-encoded mouse wheel-up event. In the regression, this
	// is a control-lane write, so it flows directly to the tmux client
	// via the attach PTY master and copy-mode is preserved.
	wheelUp := []byte{0x1b, '[', '<', '6', '4', ';', '1', ';', '1', 'M'}
	if err := p.WriteInput(wheelUp, pty.KindControl); err != nil {
		t.Fatalf("WriteInput wheel-up: %v", err)
	}

	after, err := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_in_mode}").Output()
	if err != nil {
		t.Fatalf("display-message (after): %v", err)
	}
	if strings.TrimSpace(string(after)) != "1" {
		t.Errorf("mouse-wheel event cancelled copy-mode: pane_in_mode=%q want=1",
			strings.TrimSpace(string(after)))
	}
}

// TestTmuxPTY_Paste_CleansUpBuffer asserts that the paste
// path's `-d` flag on paste-buffer successfully removes the tmux
// buffer after delivery so our per-call buffers don't accumulate
// across many pastes.
func TestTmuxPTY_Paste_CleansUpBuffer(t *testing.T) {
	requireIsolatedTmux(t)

	spec := pty.LaunchSpec{
		SessionID: "test-paste-buffer-cleanup",
		Shell:     "/bin/sh",
		Cols:      80,
		Rows:      24,
	}
	p, err := tmuxPTYFactory(spec)
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	sessionName := tmuxSessionPrefix + spec.SessionID
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Send three pastes in a row.
	for i := 0; i < 3; i++ {
		if err := p.WriteInput([]byte("echo ok\n"), pty.KindPaste); err != nil {
			t.Fatalf("paste %d: %v", i, err)
		}
	}

	// list-buffers should show no `wc-paste-` buffers remaining. The
	// socket is per-test (via requireIsolatedTmux) so any leftover
	// buffer is from THIS test.
	out, err := tmuxCmd("list-buffers", "-F", "#{buffer_name}").Output()
	if err != nil {
		// `list-buffers` returns non-zero when there are zero buffers,
		// which is the desired state.
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "wc-paste-") {
			t.Errorf("paste buffer %q was not cleaned up after paste-buffer -d", line)
		}
	}
}
