package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"web-console/backends/codex"
)

// TestCodexTailer_SurvivesPastStaleTimeoutWhileFileRecentlyModified verifies
// P1 fix: the watcher no longer exits on the hard stale timeout if the file
// has been written to within the stale window. Previously, an agent that
// went quiet for 1h would cause the watcher to exit permanently and any
// subsequent writes would never be parsed.
func TestCodexTailer_SurvivesPastStaleTimeoutWhileFileRecentlyModified(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-stale.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ct := NewCodexTailer(srv)
	// 200ms stale timeout so the test doesn't sleep for minutes.
	ct.staleTimeout = 200 * time.Millisecond
	ct.scanForNewFiles()
	t.Cleanup(ct.Stop)

	// Wait well past the stale timeout; the watcher should self-heal because
	// the file's mtime is still recent (touch it mid-wait).
	time.Sleep(150 * time.Millisecond)
	if err := os.Chtimes(rolloutPath, now, time.Now()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	// The watcher must still be alive — writing a new line should route to
	// the owning session.
	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "late reply"}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	event := waitForFirstEvent(t, srv.conversations, sess.ID, 3*time.Second)
	if event.Text != "late reply" {
		t.Fatalf("expected tailer to still be watching; got %q", event.Text)
	}
}
