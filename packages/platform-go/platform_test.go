package platform

import (
	"context"
	"errors"
	"os"
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

func TestFileLockAdmittedHandleSurvivesPathReplacementWithoutLockingNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	file, err := OpenLockFile(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := os.Rename(path, path+".retired"); err != nil {
		t.Fatal(err)
	}
	replacement, err := AcquireFileLockContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer replacement()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	held, err := LockFileContext(ctx, file)
	if err != nil {
		t.Fatalf("handle wait reopened and contended with replacement: %v", err)
	}
	defer held()
	if err := VerifyLockFile(file, path); err == nil {
		t.Fatal("detached old inode accepted as the current lock path")
	}
	held()
	replacement()
	if _, err := file.Stat(); err != nil {
		t.Fatal("handle lock release closed caller-owned file")
	}
	if err := VerifyLockFile(file, path+".retired"); err != nil {
		t.Fatal(err)
	}
}

func TestFileLockHandleCancellationDoesNotCloseOrSteal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	first, err := AcquireFileLockContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	file, err := OpenLockFile(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := LockFileContext(ctx, file); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait did not cancel: %v", err)
	}
	if _, err := file.Stat(); err != nil {
		t.Fatal("canceled waiter closed the caller's admitted handle")
	}
	first()
	release, err := LockFileContext(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}
	release()
	release()
}
