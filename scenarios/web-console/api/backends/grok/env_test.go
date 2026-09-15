package grok

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareSessionHomeIsolatesRuntimeState(t *testing.T) {
	shared := t.TempDir()
	for _, name := range []string{"config.json", "sessions", "active_sessions.json", "active_sessions.lock", "other.lock"} {
		path := filepath.Join(shared, name)
		if name == "sessions" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := PrepareSessionHome(filepath.Join(t.TempDir(), "session"), shared)
	if _, err := os.Stat(filepath.Join(session, "sessions")); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(session, "config.json")); err != nil || target != filepath.Join(shared, "config.json") {
		t.Fatalf("config link = %q, %v", target, err)
	}
	for _, name := range []string{"active_sessions.json", "active_sessions.lock", "other.lock"} {
		if _, err := os.Lstat(filepath.Join(session, name)); !os.IsNotExist(err) {
			t.Fatalf("isolated entry %s exists: %v", name, err)
		}
	}
	if got := PrepareSessionHome(filepath.Join(t.TempDir(), "empty"), ""); got == "" {
		t.Fatal("empty shared home returned empty session home")
	}
}

func TestSessionHomeAndSessionsDir(t *testing.T) {
	root := t.TempDir()
	home := SessionHome(root, "session-1")
	if home == "" || SessionsDir(root, "session-1") != filepath.Join(home, "sessions") {
		t.Fatalf("home layout = %q", home)
	}
}
