package process

import (
	"context"
	"io"
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
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("failed to kill process: %v", err)
		}
	}()

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
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("failed to kill process: %v", err)
		}
	}()

	proc, err := tracker.Track(sandboxID, cmd.Process.Pid, "sleep", "")
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	// Should be running
	if !proc.IsRunning() {
		t.Error("process should be running")
	}

	// Kill it
	if err := cmd.Process.Kill(); err != nil {
		t.Logf("failed to kill process: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("wait failed: %v", err)
		}
	}

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
	if _, err := tracker.Track(sandboxID, pid, "process1", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	if _, err := tracker.Track(sandboxID, pid, "process2", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	if _, err := tracker.Track(sandboxID, pid, "process3", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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
	if _, err := tracker.Track(sandboxID, os.Getpid(), "running", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	// Track a fake non-existent process
	if _, err := tracker.Track(sandboxID, 999999, "not-running", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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
	if _, err := tracker.Track(sandboxID, 999999, "fake", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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
	if _, err := tracker.Track(sandboxID, 999999, "dead", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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

	if _, err := tracker.Track(sandboxID, os.Getpid(), "test", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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
	if _, err := tracker.Track(sandbox1, os.Getpid(), "proc1", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	if _, err := tracker.Track(sandbox1, os.Getpid(), "proc2", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	if _, err := tracker.Track(sandbox2, os.Getpid(), "proc3", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	// Add a dead process
	if _, err := tracker.Track(sandbox2, 999999, "dead", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

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
	if _, err := tracker.Track(sandboxID, os.Getpid(), "test", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	if tracker.GetActiveCount(sandboxID) != 1 {
		t.Errorf("expected 1 active count, got %d", tracker.GetActiveCount(sandboxID))
	}

	// Track dead process
	if _, err := tracker.Track(sandboxID, 999999, "dead", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	// Still just 1 active
	if tracker.GetActiveCount(sandboxID) != 1 {
		t.Errorf("expected 1 active count (dead process not counted), got %d", tracker.GetActiveCount(sandboxID))
	}
}

// [REQ:REQ-P0-009] WaitForProcess returns the structured exit info
// recorded by the driver's wait reaper via RecordExit.
func TestWaitForProcess(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	pid := cmd.Process.Pid

	if _, err := tracker.Track(sandboxID, pid, "true", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	// Simulate the driver's wait reaper recording exit info.
	tracker.RecordExit(sandboxID, pid, ExitInfo{ExitCode: 0, StoppedAt: time.Now()})

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
	if info := tracker.GetExitInfo(sandboxID, pid); info == nil || info.ExitCode != 0 {
		t.Errorf("GetExitInfo: want exitCode=0, got %v", info)
	}
}

// WaitForProcess times out when the wait reaper never records exit info.
func TestWaitForProcessTimeout(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	ctx := context.Background()

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	if _, err := tracker.Track(sandboxID, cmd.Process.Pid, "sleep", ""); err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	_, err := tracker.WaitForProcess(ctx, sandboxID, cmd.Process.Pid, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error when no RecordExit happens")
	}
}

// RecordExit unblocks ExitChannel subscribers and is idempotent.
func TestRecordExit_IdempotentAndNotifies(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	pid := 12345
	if _, err := tracker.Track(sandboxID, pid, "fake", ""); err != nil {
		t.Fatalf("Track: %v", err)
	}

	exitCh := tracker.ExitChannel(sandboxID, pid)
	select {
	case <-exitCh:
		t.Fatal("ExitChannel should not be closed before RecordExit")
	default:
	}

	tracker.RecordExit(sandboxID, pid, ExitInfo{ExitCode: 7, Signal: 9, OOMKilled: true, StoppedAt: time.Now()})

	select {
	case <-exitCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ExitChannel did not close after RecordExit")
	}

	info := tracker.GetExitInfo(sandboxID, pid)
	if info == nil {
		t.Fatal("GetExitInfo returned nil")
	}
	if info.ExitCode != 7 || info.Signal != 9 || !info.OOMKilled {
		t.Errorf("ExitInfo = %+v; want code=7 sig=9 oom=true", *info)
	}

	// Idempotent — second call should not panic or change state.
	tracker.RecordExit(sandboxID, pid, ExitInfo{ExitCode: 99})
	if info2 := tracker.GetExitInfo(sandboxID, pid); info2.ExitCode != 7 {
		t.Errorf("second RecordExit changed exit code; got %d", info2.ExitCode)
	}
}

// SetStdin / WriteStdin / CloseStdin form a one-shot pipe contract.
func TestStdinPipeContract(t *testing.T) {
	tracker := NewTracker()
	sandboxID := uuid.New()
	pid := 22222
	if _, err := tracker.Track(sandboxID, pid, "fake", ""); err != nil {
		t.Fatalf("Track: %v", err)
	}

	// Without SetStdin, WriteStdin should fail.
	if _, err := tracker.WriteStdin(sandboxID, pid, []byte("oops")); err == nil {
		t.Error("WriteStdin without stdin pipe should fail")
	}

	r, w := io.Pipe()
	defer r.Close()
	if err := tracker.SetStdin(sandboxID, pid, w); err != nil {
		t.Fatalf("SetStdin: %v", err)
	}

	got := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(r)
		got <- buf
	}()

	n, err := tracker.WriteStdin(sandboxID, pid, []byte("hello stdin"))
	if err != nil {
		t.Fatalf("WriteStdin: %v", err)
	}
	if n != len("hello stdin") {
		t.Errorf("WriteStdin n = %d; want %d", n, len("hello stdin"))
	}
	if err := tracker.CloseStdin(sandboxID, pid); err != nil {
		t.Errorf("CloseStdin: %v", err)
	}

	select {
	case payload := <-got:
		if string(payload) != "hello stdin" {
			t.Errorf("read = %q; want hello stdin", string(payload))
		}
	case <-time.After(time.Second):
		t.Fatal("io.Pipe reader didn't see write before timeout")
	}

	// CloseStdin is idempotent.
	if err := tracker.CloseStdin(sandboxID, pid); err != nil {
		t.Errorf("second CloseStdin: %v", err)
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
