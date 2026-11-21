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

// The app-server daemon install is ~430 MB per release. A session home must
// reuse the shared install rather than download its own copy, while its rollout
// tree stays private to the session.
func TestPrepareSessionHomeSharesDaemonInstallButNotRollouts(t *testing.T) {
	shared := t.TempDir()
	session := PrepareSessionHome(filepath.Join(t.TempDir(), "session"), shared)
	for _, dir := range []string{"packages", ".tmp", "plugins", "cache"} {
		if target, err := os.Readlink(filepath.Join(session, dir)); err != nil || target != filepath.Join(shared, dir) {
			t.Fatalf("%s link = %q, %v; want shared %s", dir, target, err, filepath.Join(shared, dir))
		}
	}
	if info, err := os.Lstat(filepath.Join(session, "sessions")); err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		t.Fatalf("rollout tree must be a private directory: %v", err)
	}
}

func TestSessionHomeAndSessionsDir(t *testing.T) {
	root := t.TempDir()
	home := SessionHome(root, "session-1")
	if home == "" || SessionsDir(root, "session-1") != filepath.Join(home, "sessions") {
		t.Fatalf("home layout = %q", home)
	}
}
