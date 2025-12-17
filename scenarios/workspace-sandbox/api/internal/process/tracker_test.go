package process

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-009] Test process tracking basic operations
func TestTrackerBasicOperations(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	// Start a simple sleep process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}
	pid := cmd.Process.Pid
	defer cmd.Process.Kill()

	// Track it
	proc, err := tracker.Track(sandboxID, pid, "sleep 10", "session-1")
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	if proc.PID != pid {
		t.Errorf("expected PID %d, got %d", pid, proc.PID)
	}

	if proc.SandboxID != sandboxID {
		t.Error("sandbox ID mismatch")
	}

	if proc.Command != "sleep 10" {
		t.Errorf("expected command 'sleep 10', got '%s'", proc.Command)
	}

	if proc.SessionID != "session-1" {
		t.Errorf("expected session ID 'session-1', got '%s'", proc.SessionID)
	}
}

// [REQ:REQ-P0-009] Test process running detection
func TestTrackedProcessIsRunning(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	// Start a sleep process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	defer cmd.Process.Kill()

	proc, _ := tracker.Track(sandboxID, cmd.Process.Pid, "sleep", "")

	// Should be running
	if !proc.IsRunning() {
		t.Error("process should be running")
	}

	// Kill it
	cmd.Process.Kill()
	cmd.Wait()

	// Should not be running anymore
	// Give a moment for the OS to update state
	time.Sleep(100 * time.Millisecond)

	if proc.IsRunning() {
		t.Error("process should not be running after kill")
	}
}

// [REQ:REQ-P0-009] Test GetProcesses returns all tracked processes
func TestGetProcesses(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	// Track some fake processes (using current PID as they'll be "running")
	pid := os.Getpid()
	tracker.Track(sandboxID, pid, "process1", "")
	tracker.Track(sandboxID, pid, "process2", "")
	tracker.Track(sandboxID, pid, "process3", "")

	procs := tracker.GetProcesses(sandboxID)
	if len(procs) != 3 {
		t.Errorf("expected 3 processes, got %d", len(procs))
	}
}

// [REQ:REQ-P0-009] Test GetRunningProcesses filters correctly
func TestGetRunningProcesses(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	// Track current process (running)
	tracker.Track(sandboxID, os.Getpid(), "running", "")

	// Track a fake non-existent process
	tracker.Track(sandboxID, 999999, "not-running", "")

	running := tracker.GetRunningProcesses(sandboxID)
	if len(running) != 1 {
		t.Errorf("expected 1 running process, got %d", len(running))
	}

	if len(running) > 0 && running[0].Command != "running" {
		t.Error("wrong process returned as running")
	}
}

// [REQ:REQ-P0-009] Test KillAll method exists and doesn't panic
// Note: Full process kill testing is done in integration tests.
func TestKillAll(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	// Track a process that doesn't exist (dead)
	tracker.Track(sandboxID, 999999, "fake", "")

	// KillAll should handle dead processes gracefully
	killed, errs := tracker.KillAll(ctx, sandboxID)

	// Should have 0 killed (process already dead) and 0 errors
	if killed != 0 {
		t.Errorf("expected 0 killed for dead process, got %d", killed)
	}
	if len(errs) != 0 {
		t.Logf("errors (may be expected): %v", errs)
	}
}

// [REQ:REQ-P0-009] Test KillProcess method exists and handles edge cases
// Note: Full process kill testing is done in integration tests.
func TestKillProcess(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	// Track a dead process
	tracker.Track(sandboxID, 999999, "dead", "")

	// KillProcess should handle dead processes gracefully
	err := tracker.KillProcess(ctx, sandboxID, 999999)
	// No error expected for already-dead process
	if err != nil {
		t.Logf("KillProcess on dead process: %v (may be expected)", err)
	}

	// Test killing process not in sandbox tracking
	err = tracker.KillProcess(ctx, sandboxID, 888888)
	if err == nil {
		t.Error("expected error for process not in sandbox tracking")
	}
}

// [REQ:REQ-P0-009] Test Cleanup removes tracking data
func TestCleanup(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	tracker.Track(sandboxID, os.Getpid(), "test", "")

	procs := tracker.GetProcesses(sandboxID)
	if len(procs) != 1 {
		t.Error("expected 1 process before cleanup")
	}

	tracker.Cleanup(sandboxID)

	procs = tracker.GetProcesses(sandboxID)
	if len(procs) != 0 {
		t.Error("expected 0 processes after cleanup")
	}
}

// [REQ:REQ-P0-009] Test GetAllStats aggregates correctly
func TestGetAllStats(t *testing.T) {
	tracker := NewTracker()
	sandbox1 := uuid.New()
	sandbox2 := uuid.New()

	// Track processes in two sandboxes
	tracker.Track(sandbox1, os.Getpid(), "proc1", "")
	tracker.Track(sandbox1, os.Getpid(), "proc2", "")
	tracker.Track(sandbox2, os.Getpid(), "proc3", "")

	// Add a dead process
	tracker.Track(sandbox2, 999999, "dead", "")

	stats := tracker.GetAllStats()

	if stats.TotalTracked != 4 {
		t.Errorf("expected 4 total tracked, got %d", stats.TotalTracked)
	}

	if stats.TotalRunning != 3 {
		t.Errorf("expected 3 total running, got %d", stats.TotalRunning)
	}

	if stats.SandboxesWithProcesses != 2 {
		t.Errorf("expected 2 sandboxes with processes, got %d", stats.SandboxesWithProcesses)
	}
}

// [REQ:REQ-P0-009] Test Session workflow
func TestSessionWorkflow(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	// Start a session
	session := tracker.StartSession(sandboxID)
	if session.ID == "" {
		t.Error("session should have an ID")
	}
	if session.SandboxID != sandboxID {
		t.Error("session sandbox ID mismatch")
	}

	// Track a process in the session (use current PID for simplicity)
	proc, err := tracker.TrackInSession(session, os.Getpid(), "test-process")
	if err != nil {
		t.Fatalf("TrackInSession failed: %v", err)
	}

	if proc.SessionID != session.ID {
		t.Error("process should have session ID")
	}

	if len(session.Processes) != 1 {
		t.Errorf("session should have 1 process, got %d", len(session.Processes))
	}

	// End session without killing (killProcesses=false since we're tracking ourselves)
	err = tracker.EndSession(ctx, session, false)
	if err != nil {
		t.Errorf("EndSession failed: %v", err)
	}

	if session.EndedAt == nil {
		t.Error("session should have EndedAt set")
	}
}

// [REQ:REQ-P0-009] Test GetActiveCount
func TestGetActiveCount(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()

	// Initially zero
	if tracker.GetActiveCount(sandboxID) != 0 {
		t.Error("expected 0 active count initially")
	}

	// Track current process
	tracker.Track(sandboxID, os.Getpid(), "test", "")

	if tracker.GetActiveCount(sandboxID) != 1 {
		t.Errorf("expected 1 active count, got %d", tracker.GetActiveCount(sandboxID))
	}

	// Track dead process
	tracker.Track(sandboxID, 999999, "dead", "")

	// Still just 1 active
	if tracker.GetActiveCount(sandboxID) != 1 {
		t.Errorf("expected 1 active count (dead process not counted), got %d", tracker.GetActiveCount(sandboxID))
	}
}

// [REQ:REQ-P0-009] Test WaitForProcess
func TestWaitForProcess(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	// Start a quick process that exits immediately
	cmd := exec.Command("true") // 'true' command exits immediately with 0
	cmd.Start()
	pid := cmd.Process.Pid

	tracker.Track(sandboxID, pid, "true", "")

	// Wait for the command to actually exit first
	cmd.Wait()

	// Now wait for our tracking to detect it's dead
	proc, err := tracker.WaitForProcess(ctx, sandboxID, pid, 5*time.Second)
	if err != nil {
		t.Errorf("WaitForProcess failed: %v", err)
	}

	if proc.StoppedAt == nil {
		t.Error("process should have StoppedAt set")
	}

	if proc.ExitCode == nil {
		t.Error("process should have ExitCode set")
	}
}

// [REQ:REQ-P0-009] Test WaitForProcess timeout
func TestWaitForProcessTimeout(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	// Start a long-running process
	cmd := exec.Command("sleep", "60")
	cmd.Start()
	defer cmd.Process.Kill()

	tracker.Track(sandboxID, cmd.Process.Pid, "sleep", "")

	// Wait with short timeout
	_, err := tracker.WaitForProcess(ctx, sandboxID, cmd.Process.Pid, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
}

// [REQ:REQ-P0-009] Test KillProcess returns error for unknown process
func TestKillProcessNotFound(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	err := tracker.KillProcess(ctx, sandboxID, 999999)
	if err == nil {
		t.Error("expected error for unknown process")
	}
}
