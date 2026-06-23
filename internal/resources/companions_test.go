package resources

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

// TestStartCompanionsDormant proves the supervisor is a no-op (byte-identical)
// for a resource that declares no companions.
func TestStartCompanionsDormant(t *testing.T) {
	called := false
	defer withCompanionDir(t, func(string) (string, error) {
		called = true
		return "", nil
	})()
	startCompanions("plain-resource", nil, io.Discard)
	stopCompanions("plain-resource", nil, io.Discard)
	if called {
		t.Error("no companions declared must not touch the companion dir")
	}
}

// TestCompanionLifecycle proves start launches + tracks a detached process,
// start is idempotent while alive, and stop signals it and clears the pidfile.
func TestCompanionLifecycle(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	c := ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}
	if err := startCompanion("whisper", c); err != nil {
		t.Fatalf("start companion: %v", err)
	}
	pidPath := filepath.Join(dir, "edge.pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok || !process.IsPIDRunning(pid) {
		t.Fatalf("companion should be running; pidfile ok=%v pid=%d", ok, pid)
	}

	// Idempotent: a second start while alive reuses the same process.
	if err := startCompanion("whisper", c); err != nil {
		t.Fatalf("idempotent start: %v", err)
	}
	if pid2, _ := readCompanionPID(pidPath); pid2 != pid {
		t.Fatalf("idempotent start spawned a new process: %d -> %d", pid, pid2)
	}

	if err := stopCompanion("whisper", c); err != nil {
		t.Fatalf("stop companion: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Errorf("pidfile should be removed after stop, stat err = %v", err)
	}
	// Give the signal a moment to take effect.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && process.IsPIDRunning(pid) {
		time.Sleep(20 * time.Millisecond)
	}
	if process.IsPIDRunning(pid) {
		t.Errorf("companion pid %d still running after stop", pid)
	}
}

// TestStopCompanionNoPidfile is a clean no-op when nothing is tracked.
func TestStopCompanionNoPidfile(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()
	if err := stopCompanion("whisper", ResourceCompanion{Name: "edge", Command: "sleep"}); err != nil {
		t.Errorf("stop with no pidfile should be a no-op, got %v", err)
	}
}

func withCompanionDir(t *testing.T, fn func(string) (string, error)) func() {
	t.Helper()
	prev := resolveCompanionDir
	resolveCompanionDir = fn
	return func() { resolveCompanionDir = prev }
}
