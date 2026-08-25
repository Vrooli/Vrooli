package main

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryTranscriptAndCodexCheckpoints(t *testing.T) {
	ctx := context.Background()
	transcript := NewInMemoryAgentTranscriptCheckpointStore()
	if _, ok, err := transcript.Get(ctx, "grok", "one"); err != nil || ok {
		t.Fatalf("empty transcript checkpoint = %v, %v", ok, err)
	}
	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "grok", SourceKey: "one", SessionID: "s1", Cursor: "42", UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := transcript.Get(ctx, "grok", "one"); err != nil || !ok || cp.Cursor != "42" {
		t.Fatalf("saved transcript checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "grok", SourceKey: "two", SessionID: "s2", Cursor: "7"}); err != nil {
		t.Fatal(err)
	}
	if err := transcript.DeleteSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := transcript.Get(ctx, "grok", "one"); ok {
		t.Fatal("deleted transcript checkpoint remains")
	}

	codex := NewInMemoryCodexCheckpointStore()
	if err := codex.Save(ctx, CodexRolloutCheckpoint{Path: "/tmp/a", SessionID: "s1", Offset: 10}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := codex.Get(ctx, "/tmp/a"); err != nil || !ok || cp.Offset != 10 {
		t.Fatalf("saved codex checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := codex.DeleteSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := codex.Get(ctx, "/tmp/a"); ok {
		t.Fatal("deleted codex checkpoint remains")
	}
}

func TestSQLTranscriptAndCodexCheckpoints(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	transcript := NewSQLAgentTranscriptCheckpointStore(db)
	when := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "opencode", SourceKey: "project", SessionID: "sql-session", Cursor: "{\"sequence\":4}", UpdatedAt: when}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := transcript.Get(ctx, "opencode", "project"); err != nil || !ok || cp.SessionID != "sql-session" || cp.Cursor != "{\"sequence\":4}" {
		t.Fatalf("saved transcript checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "opencode", SourceKey: "project", SessionID: "sql-session", Cursor: "{\"sequence\":5}", UpdatedAt: when.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := transcript.Get(ctx, "opencode", "project"); err != nil || !ok || cp.Cursor != "{\"sequence\":5}" {
		t.Fatalf("updated transcript checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := transcript.DeleteSession(ctx, "sql-session"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := transcript.Get(ctx, "opencode", "project"); err != nil || ok {
		t.Fatalf("deleted transcript checkpoint = ok=%v err=%v", ok, err)
	}

	codex := NewSQLCodexCheckpointStore(db)
	if err := codex.Save(ctx, CodexRolloutCheckpoint{Path: "/tmp/rollout.jsonl", SessionID: "sql-session", Offset: 128, UpdatedAt: when}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := codex.Get(ctx, "/tmp/rollout.jsonl"); err != nil || !ok || cp.Offset != 128 || cp.SessionID != "sql-session" {
		t.Fatalf("saved codex checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := codex.DeleteSession(ctx, "sql-session"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := codex.Get(ctx, "/tmp/rollout.jsonl"); err != nil || ok {
		t.Fatalf("deleted codex checkpoint = ok=%v err=%v", ok, err)
	}
}
