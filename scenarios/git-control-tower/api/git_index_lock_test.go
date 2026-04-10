package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestCleanStaleLock_NoLockFile(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if removed {
		t.Error("expected false when no lock file exists")
	}
}

func TestCleanStaleLock_FreshLock(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if removed {
		t.Error("expected false for a fresh lock file (age < staleLockAge)")
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Error("lock file should still exist")
	}
}

func TestCleanStaleLock_StaleEmptyLock(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-2 * staleLockAge)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if !removed {
		t.Error("expected stale empty lock to be removed")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should have been removed")
	}
}

func TestCleanStaleLock_StaleWithInvalidPID(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-2 * staleLockAge)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if !removed {
		t.Error("expected stale lock with invalid PID to be removed")
	}
}

func TestCleanStaleLock_StaleWithDeadPID(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, []byte("999999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-2 * staleLockAge)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if !removed {
		t.Error("expected stale lock with dead PID to be removed")
	}
}

func TestCleanStaleLock_StaleWithNonGitLivePID(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	// Use our own PID — alive but not a git process.
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-2 * staleLockAge)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	removed := cleanStaleLock(dir)
	if !removed {
		t.Error("expected stale lock with non-git live PID to be removed")
	}
}

func TestReadLockPID_Empty(t *testing.T) {
	f := filepath.Join(t.TempDir(), "lock")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := readLockPID(f); pid != 0 {
		t.Errorf("expected 0 for empty file, got %d", pid)
	}
}

func TestReadLockPID_ValidPID(t *testing.T) {
	f := filepath.Join(t.TempDir(), "lock")
	if err := os.WriteFile(f, []byte("12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := readLockPID(f); pid != 12345 {
		t.Errorf("expected 12345, got %d", pid)
	}
}

func TestReadLockPID_NonExistentFile(t *testing.T) {
	if pid := readLockPID("/nonexistent/path"); pid != 0 {
		t.Errorf("expected 0 for missing file, got %d", pid)
	}
}

func TestIsGitProcess_DeadPID(t *testing.T) {
	if isGitProcess(999999999) {
		t.Error("expected false for non-existent PID")
	}
}

func TestIsGitProcess_OwnProcess(t *testing.T) {
	if isGitProcess(os.Getpid()) {
		t.Error("expected false for non-git process")
	}
}
