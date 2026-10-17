// Transcript scheduler tests cover lifecycle ownership and stable session-id
// extraction without touching the operator's real harness transcript roots.
package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTranscriptImportSchedulerStartStopIsIdempotent(t *testing.T) {
	scheduler := NewTranscriptImportScheduler(nil, time.Millisecond)
	scheduler.Start(context.Background())
	scheduler.Start(context.Background())
	scheduler.Stop()
	scheduler.Stop()
}

// The first tick is a whole interval away and every restart restarts that
// interval, so without a startup sweep a frequently-restarted deployment would
// never import at all. The long interval here means only a startup sweep can
// satisfy this test.
func TestTranscriptImportSchedulerSweepsOnStart(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	scheduler := NewTranscriptImportScheduler(&Orchestrator{}, time.Hour)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for scheduler.Sweeps() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("scheduler did not sweep at startup")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A cancelled context must not start work the caller has already given up on.
func TestTranscriptImportSchedulerSkipsSweepWhenContextIsDone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scheduler := NewTranscriptImportScheduler(&Orchestrator{}, time.Hour)
	scheduler.Start(ctx)
	scheduler.Stop()
	if got := scheduler.Sweeps(); got != 0 {
		t.Fatalf("sweeps = %d, want 0 for an already-cancelled context", got)
	}
}

func TestTranscriptSessionIDFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session_meta","id":"session-123"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := transcriptSessionIDFromPath(path); got != "session-123" {
		t.Fatalf("session id=%q", got)
	}
}
