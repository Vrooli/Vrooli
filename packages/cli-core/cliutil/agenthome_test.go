package cliutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchCodingAgentMaterializesWebConsoleCodexHome(t *testing.T) {
	home := t.TempDir()
	stateRoot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	if err := os.WriteFile(filepath.Join(mustMkdir(t, filepath.Join(home, ".codex")), "auth.json"), []byte("auth"), 0o600); err != nil {
		t.Fatalf("write shared auth: %v", err)
	}

	var childEnvironment []string
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent:   "codex",
		APIBase: "http://127.0.0.1:1",
		Environment: []string{
			"WC_WEB_CONSOLE_SESSION_ID=session-1",
			"WC_SESSION_STATE_ROOT=" + stateRoot,
		},
		LookPath: func(string) (string, error) { return "/safe/codex", nil },
		RunChild: func(_ context.Context, _ string, _ []string, environment []string, _ io.Reader, _, _ io.Writer) error {
			childEnvironment = environment
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}

	gotHome := environmentValue(childEnvironment, webConsoleCodexHomeEnv)
	wantHome := filepath.Join(stateRoot, "codex", "session-1")
	if gotHome != wantHome {
		t.Fatalf("CODEX_HOME = %q, want %q", gotHome, wantHome)
	}
	if _, err := os.Lstat(gotHome); err != nil {
		t.Fatalf("launch did not materialize CODEX_HOME: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(gotHome, "auth.json")); err != nil || target != filepath.Join(home, ".codex", "auth.json") {
		t.Fatalf("auth link = %q, %v", target, err)
	}
	if info, err := os.Lstat(filepath.Join(gotHome, "sessions")); err != nil || !info.IsDir() {
		t.Fatalf("sessions is not a real directory: %v", err)
	}
}

func TestPrepareWebConsoleAgentHomeLeavesPlainEnvironmentUntouched(t *testing.T) {
	stateRoot := t.TempDir()
	environment := []string{
		"WC_WEB_CONSOLE_SESSION_ID=session-plain",
		"WC_SESSION_STATE_ROOT=" + stateRoot,
	}
	got := PrepareWebConsoleAgentHome("shell", environment)
	if len(got) != len(environment) || got[0] != environment[0] || got[1] != environment[1] {
		t.Fatalf("plain environment changed: got %v", got)
	}
	if _, err := os.Stat(filepath.Join(stateRoot, "codex", "session-plain")); !os.IsNotExist(err) {
		t.Fatalf("plain environment materialized agent home: %v", err)
	}
}

func mustMkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	return path
}
