package main

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryTranscriptCheckpoints(t *testing.T) {
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

	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "codex_rollout", SourceKey: "/tmp/a", SessionID: "s1", Cursor: "10"}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := transcript.Get(ctx, "codex_rollout", "/tmp/a"); err != nil || !ok || cp.Cursor != "10" {
		t.Fatalf("saved codex checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := transcript.DeleteSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := transcript.Get(ctx, "codex_rollout", "/tmp/a"); ok {
		t.Fatal("deleted codex checkpoint remains")
	}
}

func TestSQLTranscriptCheckpoints(t *testing.T) {
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

	if err := transcript.Save(ctx, AgentTranscriptCheckpoint{Source: "codex_rollout", SourceKey: "/tmp/rollout.jsonl", SessionID: "sql-session", Cursor: "128", UpdatedAt: when}); err != nil {
		t.Fatal(err)
	}
	if cp, ok, err := transcript.Get(ctx, "codex_rollout", "/tmp/rollout.jsonl"); err != nil || !ok || cp.Cursor != "128" || cp.SessionID != "sql-session" {
		t.Fatalf("saved codex checkpoint = %#v, %v, %v", cp, ok, err)
	}
	if err := transcript.DeleteSession(ctx, "sql-session"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := transcript.Get(ctx, "codex_rollout", "/tmp/rollout.jsonl"); err != nil || ok {
		t.Fatalf("deleted codex checkpoint = ok=%v err=%v", ok, err)
	}
}
