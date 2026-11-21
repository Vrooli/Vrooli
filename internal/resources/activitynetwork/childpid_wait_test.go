package activitynetwork

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForChildPIDReturnsPIDPublishedAfterStart(t *testing.T) {
	dir := t.TempDir()
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(dir, "child_pid"), []byte("4242\n"), 0o444)
	}()
	pid, err := WaitForChildPID(dir, 2*time.Second, 5*time.Millisecond)
	if err != nil || pid != 4242 {
		t.Fatalf("WaitForChildPID = %d, %v; want 4242, nil", pid, err)
	}
}

func TestWaitForChildPIDTimesOutWhenNeverPublished(t *testing.T) {
	_, err := WaitForChildPID(t.TempDir(), 30*time.Millisecond, 5*time.Millisecond)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("WaitForChildPID error = %v; want not-exist after timeout", err)
	}
}

func TestWaitForChildPIDDoesNotRetryMalformedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "child_pid"), []byte("nope"), 0o444); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := WaitForChildPID(dir, 2*time.Second, 5*time.Millisecond); err == nil {
		t.Fatal("WaitForChildPID accepted a malformed PID")
	}
	if time.Since(start) > time.Second {
		t.Fatal("WaitForChildPID retried a malformed PID file")
	}
}
