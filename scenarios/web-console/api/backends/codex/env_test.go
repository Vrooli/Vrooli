package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareSessionHomeCreatesLayoutAndSharedLinks(t *testing.T) {
	shared := t.TempDir()
	for _, name := range []string{"auth.json", "config.toml", "skills", "ignored.lock"} {
		path := filepath.Join(shared, name)
		if name == "skills" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := PrepareSessionHome(filepath.Join(t.TempDir(), "session"), shared)
	for _, dir := range []string{"sessions", "log", "logs", "outputs", "tmp"} {
		if info, err := os.Stat(filepath.Join(session, dir)); err != nil || !info.IsDir() {
			t.Fatalf("missing dir %s: %v", dir, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(session, "auth.json")); err != nil || target != filepath.Join(shared, "auth.json") {
		t.Fatalf("auth link = %q, %v", target, err)
	}
	if got := PrepareSessionHome("", shared); got != "" {
		t.Fatalf("empty session home = %q", got)
	}
}

func TestSessionHomeAndSessionsDir(t *testing.T) {
	root := t.TempDir()
	home := SessionHome(root, "session-1")
	if home == "" || SessionsDir(root, "session-1") != filepath.Join(home, "sessions") {
		t.Fatalf("home layout = %q", home)
	}
}
