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

func TestTranscriptSessionIDFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session_meta","id":"session-123"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := transcriptSessionIDFromPath(path); got != "session-123" {
		t.Fatalf("session id=%q", got)
	}
}
