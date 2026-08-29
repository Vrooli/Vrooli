package resources

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/tuning"
)

// TestStartCompanionsDormant proves the supervisor is a no-op (byte-identical)
// for a resource that declares no companions.
func TestStartCompanionsDormant(t *testing.T) {
	called := false
	defer withCompanionDir(t, func(string) (string, error) {
		called = true
		return "", nil
	})()
	startCompanions("plain-resource", nil, 0, io.Discard)
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
	if err := startCompanion("whisper", c, 0); err != nil {
		t.Fatalf("start companion: %v", err)
	}
	pidPath := filepath.Join(dir, "edge.pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok || !process.IsPIDRunning(pid) {
		t.Fatalf("companion should be running; pidfile ok=%v pid=%d", ok, pid)
	}

	// Idempotent: a second start while alive reuses the same process.
	if err := startCompanion("whisper", c, 0); err != nil {
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

func TestCompanionCrashLoopCapWritesFailureMarker(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	c := ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}
	pidPath := filepath.Join(dir, "edge.pid")
	if err := os.WriteFile(pidPath, []byte("99999999"), 0o644); err != nil {
		t.Fatalf("write dead pidfile: %v", err)
	}

	if err := startCompanion("whisper", c, 1); err != nil {
		t.Fatalf("first respawn should be allowed: %v", err)
	}
	pid, ok := readCompanionPID(pidPath)
	if !ok || !process.IsPIDRunning(pid) {
		t.Fatalf("respawned companion should be running; ok=%v pid=%d", ok, pid)
	}
	defer func() { _ = terminateCompanion(pid) }()
	if err := os.WriteFile(pidPath, []byte("99999999"), 0o644); err != nil {
		t.Fatalf("restore stale pidfile: %v", err)
	}

	err := startCompanion("whisper", c, 1)
	if err == nil {
		t.Fatal("second respawn inside the crash window should be capped")
	}
	failed, failure := readCompanionFailure(dir, "edge")
	if !failed {
		t.Fatal("expected failed marker after crash-loop cap")
	}
	if !strings.Contains(failure, "crash-loop cap reached") {
		t.Fatalf("failure = %q, want crash-loop cap", failure)
	}
	statuses, err := companionStatuses("whisper", []ResourceCompanion{{Name: "edge", Port: 8090}})
	if err != nil {
		t.Fatalf("companionStatuses: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Failed || !strings.Contains(statuses[0].Failure, "crash-loop cap reached") {
		t.Fatalf("statuses = %#v", statuses)
	}
	if !strings.Contains(companionDownMessage("whisper", statuses), "crash-loop cap reached") {
		t.Fatalf("message missing crash-loop cap: %q", companionDownMessage("whisper", statuses))
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

func TestCompanionStatusesReportAliveAndDeadPidfiles(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	if err := os.WriteFile(filepath.Join(dir, "alive.pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write alive pidfile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dead.pid"), []byte("99999999"), 0o644); err != nil {
		t.Fatalf("write dead pidfile: %v", err)
	}

	statuses, err := companionStatuses("whisper", []ResourceCompanion{
		{Name: "alive", Port: 8090},
		{Name: "dead", Port: 8091},
	})
	if err != nil {
		t.Fatalf("companionStatuses: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if !statuses[0].Alive || statuses[0].PID != os.Getpid() {
		t.Fatalf("alive status = %#v", statuses[0])
	}
	if statuses[1].Alive || statuses[1].PID != 99999999 {
		t.Fatalf("dead status = %#v", statuses[1])
	}
}

func TestAppendCompanionLogBoundsActiveAndRetainedFiles(t *testing.T) {
	dir := t.TempDir()
	line := strings.Repeat("x", int(tuning.CompanionLogMaxBytes()/2)) + "\n"
	appendCompanionLog(dir, "edge", line)
	appendCompanionLog(dir, "edge", line)
	appendCompanionLog(dir, "edge", "latest-state\n")
	for _, suffix := range []string{"", ".1"} {
		path := filepath.Join(dir, "edge.log"+suffix)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() > tuning.CompanionLogMaxBytes() {
			t.Fatalf("%s size = %d, cap = %d", path, info.Size(), tuning.CompanionLogMaxBytes())
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "edge.log"))
	if err != nil || !strings.Contains(string(data), "latest-state") {
		t.Fatalf("active log = %q, err=%v; newest line was not retained", string(data), err)
	}
}

func withCompanionDir(t *testing.T, fn func(string) (string, error)) func() {
	t.Helper()
	prev := resolveCompanionDir
	resolveCompanionDir = fn
	return func() { resolveCompanionDir = prev }
}
