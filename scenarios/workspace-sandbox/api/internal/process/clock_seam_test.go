// Round 4 Phase 2 — deterministic-time tests for the process layer.
//
// Tracker.KillAll's grace-period sleeps and Logger.openPendingStream's
// header timestamps both flow through the injected Clock, so tests can
// assert wording and call sequencing without any real wall-clock sleep
// in the test body.
//
// We use a locally-defined fakeClock instead of testutil/mocks.FakeClock
// to avoid a test-time import cycle: process is imported by
// internal/driver/deps.go (Round 4 Phase 7), and testutil/mocks imports
// driver via FakeDriver — so importing testutil/mocks from a process
// internal test creates a cycle.

package process

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
)

// fakeClock is the minimal Clock fake the tests in this file need. It
// matches the surface area of testutil/mocks.FakeClock for the methods
// these tests exercise (Now, Since, Sleep, NewTicker). Production code
// is unaffected — production wires clock.System{}.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock { return &fakeClock{now: start} }

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Since(t time.Time) time.Duration { return f.Now().Sub(t) }
func (f *fakeClock) Sleep(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f *fakeClock) NewTicker(d time.Duration) clock.Ticker {
	// Round 4 Phase 2 tests don't drive ticker behavior in the process
	// package; the production fake (testutil/mocks.FakeClock) does.
	// Returning a real ticker keeps the type contract honest without
	// pulling in extra plumbing.
	return realTicker{t: time.NewTicker(d)}
}

type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()               { r.t.Stop() }

// TestTracker_RecordExit_FillsStoppedAtFromClock pins the contract
// that callers (handlers, toolexecution) can leave ExitInfo.StoppedAt
// zero and the tracker stamps it from the injected clock. Without
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
// the injected clock. Forensic readers rely on these timestamps to
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
