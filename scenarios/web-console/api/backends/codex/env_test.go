package codex

import (
	"os"
	"path/filepath"
	"strings"
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

// A home created before packages was shared holds a private daemon install;
// preparing it again replaces that install with the shared link instead of
// leaving the copy to grow with every Codex update.
func TestPrepareSessionHomeReplacesLegacyPrivatePackages(t *testing.T) {
	shared := t.TempDir()
	home := filepath.Join(t.TempDir(), "session")
	legacyBinary := filepath.Join(home, "packages", "app-server-daemon", "releases", "0.170.0", "bin", "codex")
	if err := os.MkdirAll(filepath.Dir(legacyBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyBinary, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	rollout := filepath.Join(home, "sessions", "rollout.jsonl")
	if err := os.WriteFile(rollout, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	PrepareSessionHome(home, shared)
	if target, err := os.Readlink(filepath.Join(home, "packages")); err != nil || target != filepath.Join(shared, "packages") {
		t.Fatalf("packages link = %q, %v; want shared", target, err)
	}
	if _, err := os.Stat(rollout); err != nil {
		t.Fatalf("rollout tree must be untouched: %v", err)
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "legacy") {
			t.Fatalf("legacy staging left behind: %s", entry.Name())
		}
	}
}
