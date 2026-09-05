package platform

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveHomePath(t *testing.T) {
	home := t.TempDir()
	got, err := ResolveHomePath(home, "state/runtime.db")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(home, "state", "runtime.db") {
		t.Fatalf("path = %q", got)
	}
	if _, err := ResolveHomePath(home, "../escape"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestConfigureCommandDetached(t *testing.T) {
	cmd := exec.Command("echo", "ok")
	if err := ConfigureCommand(cmd, ProcessOptions{Detached: true}); err != nil {
		t.Fatal(err)
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("expected native detached attributes")
	}
}

func TestAcquireFileLock(t *testing.T) {
	release, err := AcquireFileLock(filepath.Join(t.TempDir(), "lock"))
	if err != nil {
		t.Fatal(err)
	}
	release()
}
