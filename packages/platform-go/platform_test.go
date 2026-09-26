package platform

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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

func TestFileLockWaitIsCancellableAndDoesNotStealOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	release, err := AcquireFileLockContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := AcquireFileLockContext(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked lock: %v", err)
	}
	release()
	release()
	next, err := AcquireFileLockContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	next()
}
