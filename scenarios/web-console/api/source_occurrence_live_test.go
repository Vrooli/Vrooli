package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClaudeLiveTailStoresSourceOccurrenceSeparately(t *testing.T) {
	srv, sessionID, path := newClaudeTailerTestServer(t)
	occurredAt := time.Date(2025, 10, 17, 14, 27, 0, 123456789, time.UTC)
	writeClaudeTranscript(t, path, `{"type":"assistant","timestamp":"2025-10-17T14:27:00.123456789Z","message":{"content":"historical Claude answer"}}`)

	NewClaudeTailer(srv).scan()
	state := srv.conversations.ListSession(context.Background(), sessionID)
	if len(state.Events) != 1 {
		t.Fatalf("live Claude events=%d, want 1", len(state.Events))
	}
	event := state.Events[0]
	if !event.OccurredAt.Equal(occurredAt) || event.CreatedAt.IsZero() || event.CreatedAt.Equal(event.OccurredAt) {
		t.Fatalf("Claude event ingest/source times=%s / %s", event.CreatedAt, event.OccurredAt)
	}
}

func TestCodexLiveTailStoresSourceOccurrenceSeparately(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-occurrence.jsonl")
	occurredAt := time.Date(2025, 10, 17, 14, 27, 0, 123456789, time.UTC)
	payload, err := json.Marshal(rolloutMessageFixture{
		Role:    "assistant",
		Content: []rolloutContentFixture{{Type: "output_text", Text: "historical Codex answer"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	line, err := json.Marshal(rolloutFixture{Timestamp: occurredAt.Format(time.RFC3339Nano), Type: "response_item", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rolloutPath, append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	tailer := NewCodexTailer(srv)
	tailer.scanForNewFiles()
	t.Cleanup(tailer.Stop)
	event := waitForFirstEvent(t, srv.conversations, sess.ID, 3*time.Second)
	if !event.OccurredAt.Equal(occurredAt) || event.CreatedAt.IsZero() || event.CreatedAt.Equal(event.OccurredAt) {
		t.Fatalf("Codex event ingest/source times=%s / %s", event.CreatedAt, event.OccurredAt)
	}
}
