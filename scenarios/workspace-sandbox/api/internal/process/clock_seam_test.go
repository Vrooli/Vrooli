// Round 4 Phase 2 — deterministic-time tests for the process layer.
//
// Tracker.KillAll's grace-period sleeps and Logger.openPendingStream's
// header timestamps both flow through the injected Clock, so tests can
// assert wording and call sequencing without any real wall-clock sleep
// in the test body.
//
// The shared scheduletest fake is safe here because it depends only on the
// api-core schedule contract; it does not import this scenario or its test
// helpers, so the process package remains cycle-free.

package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vrooli/api-core/scheduletest"
)

func newFakeClock(start time.Time) *scheduletest.FakeClock { return scheduletest.New(start) }

// TestTracker_RecordExit_FillsStoppedAtFromClock pins the contract
// that callers can leave ExitInfo.StoppedAt
// zero and the tracker stamps it from the injected schedule. Without
// this, every caller would need its own clock, fragmenting the time
// source.
func TestTracker_RecordExit_FillsStoppedAtFromClock(t *testing.T) {
	pinned := time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)
	clk := newFakeClock(pinned)
	tracker := NewTracker(clk)

	sandboxID := uuid.New()
	pid := os.Getpid()
	if _, err := tracker.Track(sandboxID, pid, "self", ""); err != nil {
		t.Fatalf("Track: %v", err)
	}

	tracker.RecordExit(sandboxID, pid, ExitInfo{ExitCode: 0})

	got := tracker.GetExitInfo(sandboxID, pid)
	if got == nil {
		t.Fatal("expected ExitInfo")
	}
	if !got.StoppedAt.Equal(pinned) {
		t.Errorf("StoppedAt = %s, want %s (FakeClock pin)", got.StoppedAt, pinned)
	}
}

// TestLogger_HeaderUsesClockTimestamp pins that the per-stream log
// header records the time the pending pair was created, sourced from
// the injected schedule. Forensic readers rely on these timestamps to
// reconstruct run timelines.
func TestLogger_HeaderUsesClockTimestamp(t *testing.T) {
	pinned := time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)
	clk := newFakeClock(pinned)
	logger := NewLogger(LogConfig{BaseDir: t.TempDir()}, clk)

	sandboxID := uuid.New()
	pending, err := logger.CreatePendingLogPair(sandboxID)
	if err != nil {
		t.Fatalf("CreatePendingLogPair: %v", err)
	}
	if pending == nil {
		t.Fatal("expected non-nil pending pair")
	}

	// Read the stdout pending file and assert the header carries the
	// pinned timestamp.
	data, err := os.ReadFile(pending.Stdout.path)
	if err != nil {
		t.Fatalf("read stdout pending: %v", err)
	}
	if !strings.Contains(string(data), pinned.Format(time.RFC3339)) {
		t.Errorf("expected header to contain %s, got: %s", pinned.Format(time.RFC3339), data)
	}
	_ = filepath.Base(pending.Stdout.path)

	// Tear-down: AbortPair removes the temp files and closes the
	// writers. Without this the test leaks open file descriptors.
	if err := logger.AbortPair(pending); err != nil {
		t.Fatalf("AbortPair: %v", err)
	}
}
