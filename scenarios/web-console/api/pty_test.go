package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// --- ensureTermEnv tests ---

func TestEnsureTermEnv_ReplacesExisting(t *testing.T) {
	env := []string{"HOME=/home/user", "TERM=dumb", "PATH=/usr/bin"}
	got := ensureTermEnv(env)

	found := false
	for _, v := range got {
		if v == "TERM=xterm-256color" {
			found = true
		}
		if v == "TERM=dumb" {
			t.Error("old TERM=dumb should be replaced")
		}
	}
	if !found {
		t.Error("TERM=xterm-256color should be present")
	}
}

func TestEnsureTermEnv_AddsWhenMissing(t *testing.T) {
	env := []string{"HOME=/home/user", "PATH=/usr/bin"}
	got := ensureTermEnv(env)

	found := false
	for _, v := range got {
		if v == "TERM=xterm-256color" {
			found = true
		}
	}
	if !found {
		t.Error("TERM=xterm-256color should be appended when TERM is missing")
	}
	if len(got) != 3 {
		t.Errorf("expected 3 env vars, got %d", len(got))
	}
}

func TestEnsureTermEnv_PreservesOtherVars(t *testing.T) {
	env := []string{"HOME=/home/user", "TERM=linux", "TERMINAL_EMULATOR=foo"}
	got := ensureTermEnv(env)

	// TERMINAL_EMULATOR should not be affected (it has TERM prefix but not TERM=)
	foundEmulator := false
	for _, v := range got {
		if v == "TERMINAL_EMULATOR=foo" {
			foundEmulator = true
		}
	}
	if !foundEmulator {
		t.Error("TERMINAL_EMULATOR should be preserved")
	}
}

// --- filterClaudeEnv tests ---

func TestFilterClaudeEnv_RemovesCLAUDECODE(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"CLAUDECODE=1",
		"PATH=/usr/bin",
	}
	got := filterClaudeEnv(env)

	for _, v := range got {
		if v == "CLAUDECODE=1" {
			t.Error("CLAUDECODE should be filtered out")
		}
	}
	if len(got) != 2 {
		t.Errorf("expected 2 env vars after filtering, got %d", len(got))
	}
}

func TestFilterClaudeEnv_RemovesAllClaudeVars(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"CLAUDECODE=1",
		"CLAUDE_SESSION_ID=abc123",
		"CLAUDE_CONFIG_DIR=/home/user/.claude",
		"CLAUDE_NON_INTERACTIVE=true",
		"CLAUDE_CODE_ENTRYPOINT=sdk-cli",
		"PATH=/usr/bin",
	}
	got := filterClaudeEnv(env)

	for _, v := range got {
		if strings.HasPrefix(v, "CLAUDE") {
			t.Errorf("Claude env var should be filtered out: %s", v)
		}
	}
	// Only HOME and PATH should remain
	if len(got) != 2 {
		t.Errorf("expected 2 env vars after filtering, got %d: %v", len(got), got)
	}
}

func TestFilterClaudeEnv_PreservesNonClaudeVars(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"CLAUDECODE=1",
		"TERM=xterm-256color",
		"SHELL=/bin/bash",
		"LANGUAGE=en_US",
	}
	got := filterClaudeEnv(env)

	expected := map[string]bool{
		"HOME=/home/user":     true,
		"TERM=xterm-256color": true,
		"SHELL=/bin/bash":     true,
		"LANGUAGE=en_US":      true,
	}

	for _, v := range got {
		if !expected[v] {
			t.Errorf("unexpected var in output: %s", v)
		}
		delete(expected, v)
	}
	if len(expected) > 0 {
		t.Errorf("missing expected vars: %v", expected)
	}
}

func TestFilterClaudeEnv_HandlesEmptyEnv(t *testing.T) {
	env := []string{}
	got := filterClaudeEnv(env)

	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestFilterClaudeEnv_RemovesBashFuncClaudeCode(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"BASH_FUNC_claude_code::run%%=() { echo test; }",
		"BASH_FUNC_claude_code::session%%=() { echo test; }",
		"BASH_FUNC_normal_func%%=() { echo test; }",
		"PATH=/usr/bin",
	}
	got := filterClaudeEnv(env)

	for _, v := range got {
		if strings.Contains(v, "claude_code::") {
			t.Errorf("BASH_FUNC claude_code:: should be filtered out: %s", v)
		}
	}
	// HOME, PATH, and normal_func should remain
	if len(got) != 3 {
		t.Errorf("expected 3 env vars after filtering, got %d: %v", len(got), got)
	}
}

// --- filterServiceEnv tests ---

func TestFilterServiceEnv_RemovesServiceVars(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"API_PORT=36232",
		"API_BASE_URL=http://localhost:36232",
		"UI_PORT=36240",
		"WS_PORT=25000",
		"VITE_API_BASE_URL=http://localhost:36232",
		"API_BASE=http://localhost:36232",
		"PATH=/usr/bin",
		"SHELL=/bin/bash",
	}
	got := filterServiceEnv(env)

	blocked := map[string]bool{
		"API_PORT": true, "API_BASE_URL": true, "UI_PORT": true,
		"WS_PORT": true, "VITE_API_BASE_URL": true, "API_BASE": true,
	}
	for _, v := range got {
		name, _, _ := strings.Cut(v, "=")
		if blocked[name] {
			t.Errorf("service env var should be filtered out: %s", v)
		}
	}
	if len(got) != 3 {
		t.Errorf("expected 3 env vars (HOME, PATH, SHELL), got %d: %v", len(got), got)
	}
}

func TestFilterServiceEnv_PreservesScenarioSpecificVars(t *testing.T) {
	env := []string{
		"TUNNEL_MANAGER_API_PORT=15001",
		"TUNNEL_MANAGER_API_BASE=http://localhost:15001",
		"API_PORT=36232",
		"HOME=/home/user",
	}
	got := filterServiceEnv(env)

	foundTM := 0
	for _, v := range got {
		if strings.HasPrefix(v, "TUNNEL_MANAGER_") {
			foundTM++
		}
		if strings.HasPrefix(v, "API_PORT=") {
			t.Error("generic API_PORT should be filtered out")
		}
	}
	if foundTM != 2 {
		t.Errorf("expected 2 TUNNEL_MANAGER_* vars preserved, got %d", foundTM)
	}
}

func TestFilterServiceEnv_HandlesEmptyEnv(t *testing.T) {
	got := filterServiceEnv([]string{})
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestFilterServiceEnv_RemovesHostTerminalVars(t *testing.T) {
	// REGRESSION: web-console-api typically runs inside the user's own
	// terminal (often tmux). Without this filter, every standard-backend
	// shell inherits TMUX/TMUX_PANE/TERM_PROGRAM pointing at a tmux session
	// that the shell is *not* actually inside. Programs like Claude Code
	// then think they're in tmux, emit tmux DCS passthrough escapes, and
	// hang silently before rendering any UI.
	env := []string{
		"HOME=/home/user",
		"TMUX=/tmp/tmux-1000/wc,1564421,0",
		"TMUX_PANE=%0",
		"TERM_PROGRAM=tmux",
		"TERM_PROGRAM_VERSION=3.4",
		"PATH=/usr/bin",
	}
	got := filterServiceEnv(env)

	for _, v := range got {
		name, _, _ := strings.Cut(v, "=")
		switch name {
		case "TMUX", "TMUX_PANE", "TERM_PROGRAM", "TERM_PROGRAM_VERSION":
			t.Errorf("host-terminal var should be filtered out: %s", v)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected 2 env vars (HOME, PATH), got %d: %v", len(got), got)
	}
}

func TestFilterServiceEnv_RemovesLifecycleVars(t *testing.T) {
	// REGRESSION: The tmux server inherited VROOLI_LIFECYCLE_MANAGED from the
	// API process. The autoheal orphan checker then detected the tmux server as
	// a Vrooli process and killed it as an "orphan", destroying all persistent
	// sessions. These lifecycle vars must be stripped.
	env := []string{
		"HOME=/home/user",
		"VROOLI_LIFECYCLE_MANAGED=true",
		"VROOLI_SCENARIO=web-console",
		"VROOLI_STEP=start-api",
		"VROOLI_PHASE=develop",
		"PATH=/usr/bin",
	}
	got := filterServiceEnv(env)

	for _, v := range got {
		if strings.HasPrefix(v, "VROOLI_") {
			t.Errorf("Vrooli lifecycle var should be filtered out: %s", v)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected 2 env vars (HOME, PATH), got %d: %v", len(got), got)
	}
}

// TestRealPTY_ProbeReady_Synchronous exercises the standard PTY's
// ProbeReady, which must return immediately so the WebSocket input loop
// can emit session_ready without an extra round-trip.
func TestRealPTY_ProbeReady_Synchronous(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PTY tests unsupported on Windows")
	}
	p := &realPTY{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := p.ProbeReady(ctx); err != nil {
		t.Fatalf("standard PTY ProbeReady must be a no-op: %v", err)
	}
}

func TestPrepareCodexSessionHome_SharesAuthAndConfig(t *testing.T) {
	sharedHome := filepath.Join(t.TempDir(), ".codex-shared")
	if err := os.MkdirAll(sharedHome, 0o755); err != nil {
		t.Fatalf("mkdir shared home: %v", err)
	}
	for _, entry := range []string{"auth.json", "config.toml", "settings.json"} {
		if err := os.WriteFile(filepath.Join(sharedHome, entry), []byte(entry), 0o600); err != nil {
			t.Fatalf("write shared entry %s: %v", entry, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(sharedHome, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir shared skills: %v", err)
	}

	sessionHome := filepath.Join(t.TempDir(), "session-home")
	got := prepareCodexSessionHome(sessionHome, sharedHome)
	if got != sessionHome {
		t.Fatalf("expected session home %q, got %q", sessionHome, got)
	}

	for _, entry := range []string{"auth.json", "config.toml", "settings.json", "skills"} {
		linkPath := filepath.Join(sessionHome, entry)
		info, err := os.Lstat(linkPath)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", entry, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected %s to be a symlink", entry)
		}
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("readlink %s: %v", entry, err)
		}
		if target != filepath.Join(sharedHome, entry) {
			t.Fatalf("expected %s -> %s, got %s", entry, filepath.Join(sharedHome, entry), target)
		}
	}

	for _, dir := range []string{"sessions", "log", "logs", "outputs", "tmp"} {
		info, err := os.Stat(filepath.Join(sessionHome, dir))
		if err != nil {
			t.Fatalf("expected private dir %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func TestSessionManagerCreate_UsesSharedAuthAndSessionOwnedRoutingDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stateRoot := useIsolatedSessionState(t)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir shared codex dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "auth.json"), []byte("auth"), 0o600); err != nil {
		t.Fatalf("write shared codex auth: %v", err)
	}

	var captured pty.LaunchSpec
	sm := NewSessionManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		captured = spec
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, ok := captured.Env["CLAUDE_CONFIG_DIR"]; ok {
		t.Fatalf("did not expect CLAUDE_CONFIG_DIR override")
	}
	if _, ok := captured.Env["CLAUDE_SESSIONS_DIR"]; ok {
		t.Fatalf("did not expect CLAUDE_SESSIONS_DIR override")
	}
	codexHome := captured.Env["CODEX_HOME"]
	if !strings.Contains(codexHome, sess.ID) {
		t.Fatalf("expected session CODEX_HOME to contain %q, got %q", sess.ID, codexHome)
	}
	wantPrefix := filepath.Join(stateRoot, "codex", sess.ID)
	if codexHome != wantPrefix {
		t.Fatalf("expected session CODEX_HOME %q, got %q", wantPrefix, codexHome)
	}
	info, err := os.Lstat(filepath.Join(codexHome, "auth.json"))
	if err != nil {
		t.Fatalf("expected shared codex auth symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected session CODEX_HOME auth.json to be a symlink")
	}
}

// TestTmuxAttach_SetsTermEnv verifies that tmuxAttach sets TERM=xterm-256color
// on the attach command, preventing "terminal does not support clear" failures
// when the server process has TERM=dumb (common for non-interactive lifecycle).
func TestTmuxAttach_SetsTermEnv(t *testing.T) {
	requireIsolatedTmux(t)

	sessionName := tmuxSessionPrefix + "test-term-env"
	// Create a detached tmux session on the dedicated wc socket
	createCmd := tmuxCmd("new-session", "-d",
		"-s", sessionName,
		"-x", "80", "-y", "24",
		"/bin/sh",
	)
	if err := createCmd.Run(); err != nil {
		t.Fatalf("tmux new-session failed: %v", err)
	}
	defer func() { _ = tmuxCmd("kill-session", "-t", sessionName).Run() }()

	// Temporarily set TERM=dumb to simulate lifecycle-launched server
	origTerm := os.Getenv("TERM")
	os.Setenv("TERM", "dumb")
	defer os.Setenv("TERM", origTerm)

	// tmuxAttach should succeed despite TERM=dumb because it overrides TERM
	p, err := tmuxAttach(sessionName)
	if err != nil {
		t.Fatalf("tmuxAttach failed with TERM=dumb: %v", err)
	}
	defer func() { _ = p.Kill() }()
	defer p.Close()

	// Verify the session is readable (not immediately dead)
	buf := make([]byte, 1024)
	done := make(chan error, 1)
	go func() {
		_, readErr := p.Read(buf)
		done <- readErr
	}()

	select {
	case err := <-done:
		// A successful read (even 0 bytes) or a timeout means the attach
		// is alive. An immediate error means it died.
		if err != nil {
			t.Fatalf("tmux attach died immediately (TERM=dumb not overridden?): %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// Timeout = attach is alive and waiting for output. Success.
	}
}

// [REQ:P0-002a] PTY Session Backend - fast session tests via fake PTY seam
func TestFakePTY_CreateAndGet(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()

	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))
	sess, err := sm.Create("/fake/shell", 100, 50, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sess.Shell != "/fake/shell" {
		t.Errorf("expected shell=/fake/shell, got %s", sess.Shell)
	}
	if sess.Cols != 100 {
		t.Errorf("expected cols=100, got %d", sess.Cols)
	}
	if sess.Rows != 50 {
		t.Errorf("expected rows=50, got %d", sess.Rows)
	}

	got, ok := sm.Get(sess.ID)
	if !ok {
		t.Fatal("Get should find the session")
	}
	if got.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, got.ID)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - broadcast via fake PTY
func TestFakePTY_SubscribeAndBroadcast(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	// Write output from fake PTY
	testData := []byte("hello from fake")
	go func() {
		_, _ = fake.OutW.Write(testData)
	}()

	select {
	case data := <-sub.OutputCh:
		if string(data) != "hello from fake" {
			t.Errorf("expected 'hello from fake', got %q", string(data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast data")
	}

	// Cleanup
	fake.Close()
	<-sess.Done()
}

// [REQ:P0-003b] Reconnect State Restoration — snapshot replay via fake PTY.
func TestFakePTY_OfflineSnapshot(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := fake.OutW.Write([]byte("offline data")); err != nil {
		t.Fatalf("write fake pty: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)
	if !bytes.Contains(sub.Snapshot, []byte("offline data")) {
		t.Fatalf("snapshot missing offline data; got=%q", sub.Snapshot)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] Terminal Resize Handling - resize delegates to PTY interface
func TestFakePTY_Resize(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()

	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)

	sess.Resize(200, 60)

	got, _ := sm.Get(sess.ID)
	if got.Cols != 200 {
		t.Errorf("expected cols=200, got %d", got.Cols)
	}
	if got.Rows != 60 {
		t.Errorf("expected rows=60, got %d", got.Rows)
	}

	// Verify the PTY seam received the resize
	fake.Mu.Lock()
	if fake.Cols != 200 || fake.Rows != 60 {
		t.Errorf("fake PTY should have received resize: cols=%d rows=%d", fake.Cols, fake.Rows)
	}
	fake.Mu.Unlock()
}

// [REQ:P0-002a] PTY Session Backend - delete calls Kill + Close on PTY
func TestFakePTY_DeleteCleanup(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := sm.Delete(sess.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	fake.Mu.Lock()
	if !fake.Killed {
		t.Error("Delete should call Kill on PTY")
	}
	fake.Mu.Unlock()

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should not exist after Delete")
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - exit code forwarding via fake PTY
func TestFakePTY_ExitCodeForwarding(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set a non-zero exit code and close to simulate process exit
	fake.SetExitCode(42)
	fake.OutW.Close()

	select {
	case <-sess.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exit signal")
	}

	if code := sess.ExitCode(); code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

// [REQ:P0-002a] PTY Session Backend - exit signal via fake PTY
func TestFakePTY_ExitSignal(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close the output pipe to simulate process exit
	fake.OutW.Close()

	select {
	case <-sess.Done():
		// Expected: session signals exit
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exit signal")
	}

	if !sess.IsDead() {
		t.Error("session should be dead after PTY close")
	}

	// SessionManager should auto-remove the session
	time.Sleep(50 * time.Millisecond)
	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should be auto-removed after exit")
	}
}
