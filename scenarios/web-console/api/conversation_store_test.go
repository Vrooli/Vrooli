package main

import (
	"context"
	"strings"
	"time"
)
import "testing"

func TestAppendAssistantEvent_Basic(t *testing.T) {
	store := NewConversationStore()
	event, result := store.AppendAssistantEvent(context.Background(), "sess1", "test", "Hello world")
	if !result.Appended {
		t.Fatalf("expected appended, got code=%s reason=%s", result.Code, result.Reason)
	}
	if event.Role != ConversationRoleAssistant {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", event.Text)
	}
	if len(event.SpeechParagraphs) == 0 {
		t.Error("expected speechParagraphs to be populated for assistant events")
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
}

func TestAppendAssistantEvent_EmptySession(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendAssistantEvent(context.Background(), "", "test", "text")
	if result.Appended {
		t.Error("expected failure for empty session ID")
	}
	if result.Code != "conversation_target_missing" {
		t.Errorf("expected code conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendAssistantEvent_EmptyText(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendAssistantEvent(context.Background(), "sess1", "test", "")
	if result.Appended {
		t.Error("expected failure for empty text")
	}
	if result.Code != "conversation_input_required" {
		t.Errorf("expected code conversation_input_required, got %s", result.Code)
	}
}

func TestAppendAssistantEvent_Dedup(t *testing.T) {
	store := NewConversationStore()
	_, r1 := store.AppendAssistantEvent(context.Background(), "sess1", "test", "Hello")
	if !r1.Appended || r1.Duplicate {
		t.Fatal("first append should succeed and not be duplicate")
	}
	_, r2 := store.AppendAssistantEvent(context.Background(), "sess1", "test", "Hello")
	if !r2.Appended || !r2.Duplicate {
		t.Fatal("second append of same text should be marked duplicate")
	}
}

func TestAppendUserEvent_Basic(t *testing.T) {
	store := NewConversationStore()
	event, result := store.AppendUserEvent(context.Background(), "sess1", "test", "What is 2+2?")
	if !result.Appended {
		t.Fatalf("expected appended, got code=%s reason=%s", result.Code, result.Reason)
	}
	if event.Role != ConversationRoleUser {
		t.Errorf("expected role user, got %s", event.Role)
	}
	if event.Text != "What is 2+2?" {
		t.Errorf("expected text 'What is 2+2?', got %q", event.Text)
	}
	if event.SpeechParagraphs != nil {
		t.Error("expected nil speechParagraphs for user events")
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
}

func TestAppendUserEvent_EmptySession(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendUserEvent(context.Background(), "", "test", "text")
	if result.Appended {
		t.Error("expected failure for empty session ID")
	}
}

func TestAppendUserEvent_EmptyText(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendUserEvent(context.Background(), "sess1", "test", "   ")
	if result.Appended {
		t.Error("expected failure for whitespace-only text")
	}
}

func TestAppendUserEvent_Dedup(t *testing.T) {
	store := NewConversationStore()
	_, r1 := store.AppendUserEvent(context.Background(), "sess1", "test", "Hello")
	if !r1.Appended || r1.Duplicate {
		t.Fatal("first append should succeed")
	}
	_, r2 := store.AppendUserEvent(context.Background(), "sess1", "test", "Hello")
	if !r2.Appended || !r2.Duplicate {
		t.Fatal("second append should be duplicate")
	}
}

func TestAppendUserEvent_SequenceIncrements(t *testing.T) {
	store := NewConversationStore()
	e1, _ := store.AppendAssistantEvent(context.Background(), "sess1", "test", "assistant msg")
	e2, _ := store.AppendUserEvent(context.Background(), "sess1", "test", "user msg")
	if e2.Sequence != e1.Sequence+1 {
		t.Errorf("expected sequence %d, got %d", e1.Sequence+1, e2.Sequence)
	}
}

func TestUpdateSpeechParagraphs(t *testing.T) {
	store := NewConversationStore()
	event, _ := store.AppendAssistantEvent(context.Background(), "sess1", "test", "Hello world")

	newParagraphs := []string{"summarized text"}
	store.UpdateSpeechParagraphs(context.Background(), "sess1", event.ID, newParagraphs)

	state := store.ListSession(context.Background(), "sess1")
	if len(state.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(state.Events))
	}
	updated := state.Events[0]
	if len(updated.SpeechParagraphs) != 1 || updated.SpeechParagraphs[0] != "summarized text" {
		t.Errorf("expected summarized paragraphs, got %v", updated.SpeechParagraphs)
	}
	if !updated.Summarized {
		t.Error("expected Summarized to be true after UpdateSpeechParagraphs")
	}
	if len(updated.OriginalSpeechParagraphs) == 0 {
		t.Error("expected OriginalSpeechParagraphs to be preserved")
	}
	if updated.OriginalSpeechParagraphs[0] != event.SpeechParagraphs[0] {
		t.Errorf("expected OriginalSpeechParagraphs to match original, got %v", updated.OriginalSpeechParagraphs)
	}
}

func TestUpdateSpeechParagraphs_NonExistent(t *testing.T) {
	store := NewConversationStore()
	store.AppendAssistantEvent(context.Background(), "sess1", "test", "Hello")
	// Should be a no-op, no panic
	store.UpdateSpeechParagraphs(context.Background(), "sess1", "nonexistent-id", []string{"text"})
	store.UpdateSpeechParagraphs(context.Background(), "nonexistent-session", "id", []string{"text"})
}

func TestListSession_Empty(t *testing.T) {
	store := NewConversationStore()
	state := store.ListSession(context.Background(), "nonexistent")
	if len(state.Events) != 0 {
		t.Errorf("expected empty events, got %d", len(state.Events))
	}
}

func TestListSession_MixedRoles(t *testing.T) {
	store := NewConversationStore()
	store.AppendUserEvent(context.Background(), "sess1", "test", "user question")
	store.AppendAssistantEvent(context.Background(), "sess1", "test", "assistant answer")

	state := store.ListSession(context.Background(), "sess1")
	if len(state.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(state.Events))
	}
	if state.Events[0].Role != ConversationRoleUser {
		t.Errorf("expected first event user, got %s", state.Events[0].Role)
	}
	if state.Events[1].Role != ConversationRoleAssistant {
		t.Errorf("expected second event assistant, got %s", state.Events[1].Role)
	}
}

func TestSQLSearchArchivedUsesFTSAndExcludesLiveSessions(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	db := setupTestDB(t)
	var fts5 int
	if err := db.QueryRow(`SELECT sqlite_compileoption_used('ENABLE_FTS5')`).Scan(&fts5); err != nil || fts5 != 1 {
		t.Fatalf("modernc sqlite FTS5 unavailable: value=%d err=%v", fts5, err)
	}
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("ensure FTS: %v", err)
	}
	for _, row := range []struct{ id, archivedAt string }{
		{"archive-a", "2026-08-18T18:00:00Z"},
		{"archive-b", "2026-08-18T18:01:00Z"},
		{"archive-c", "2026-08-18T18:02:00Z"},
		{"live", ""},
	} {
		if _, err := db.Exec(`INSERT INTO sessions(id, archived_at) VALUES (?, ?)`, row.id, row.archivedAt); err != nil {
			t.Fatalf("insert session %s: %v", row.id, err)
		}
	}
	// archive-a is an older copy of archive-b. Deep search follows the same
	// collapsed-lineage contract as ListArchived, so duplicate transcript
	// copies never crowd out independent sessions in a bounded result page.
	if _, err := db.Exec(`UPDATE sessions SET recovered_into = 'archive-b' WHERE id = 'archive-a'`); err != nil {
		t.Fatalf("link archived lineage: %v", err)
	}
	repo := NewSQLConversationRepository(db)
	for index, id := range []string{"archive-a", "archive-b", "archive-c", "live"} {
		if _, err := repo.AppendEvent(ctx, ConversationEvent{
			ID: "event-" + id, SessionID: id, Source: "test", Role: ConversationRoleAssistant,
			Text: "shared archive needle", CreatedAt: time.Date(2026, 8, 18, 18, index, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("append %s: %v", id, err)
		}
	}

	matches, truncated, total, distinct, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{Query: "archive needle", Limit: 20})
	if err != nil {
		t.Fatalf("search archived: %v", err)
	}
	if truncated || total != 2 || distinct != 2 || len(matches) != 2 {
		t.Fatalf("search result matches=%d total=%d distinct=%d truncated=%v", len(matches), total, distinct, truncated)
	}
	for _, match := range matches {
		if match.SessionID == "live" {
			t.Fatalf("live session leaked into archive search: %+v", match)
		}
	}

	perSession, _, perTotal, err := repo.SearchSession(ctx, "live", "archive needle", 20)
	if err != nil || len(perSession) != 1 || perTotal != 1 {
		t.Fatalf("existing per-session search changed: matches=%+v total=%d err=%v", perSession, perTotal, err)
	}

	rows, err := db.Query(`EXPLAIN QUERY PLAN SELECT e.id FROM conversation_events_fts
		JOIN conversation_events e ON e.rowid=conversation_events_fts.rowid
		JOIN sessions s ON s.id=e.session_id
		WHERE conversation_events_fts.text MATCH '"archive" AND "needle"' AND s.archived_at <> ''`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(detail)
		plan.WriteByte('\n')
	}
	if got := strings.ToUpper(plan.String()); !strings.Contains(got, "VIRTUAL TABLE INDEX") {
		t.Fatalf("query plan does not use FTS index:\n%s", plan.String())
	}
}

func TestEnsureConversationFTSBackfillsOnceAndIndexesNewAppends(t *testing.T) { // [REQ:REQ-P0-003a]
	ctx := context.Background()
	db := setupTestDB(t)
	if _, err := db.Exec(`INSERT INTO sessions(id, archived_at) VALUES ('archived', '2026-08-18T18:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLConversationRepository(db)
	if _, err := repo.AppendEvent(ctx, ConversationEvent{ID: "before", SessionID: "archived", Role: ConversationRoleUser, Text: "backfill sentinel"}); err != nil {
		t.Fatal(err)
	}
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("idempotent ensure: %v", err)
	}
	if _, err := repo.AppendEvent(ctx, ConversationEvent{ID: "after", SessionID: "archived", Role: ConversationRoleUser, Text: "fresh sentinel"}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"backfill sentinel", "fresh sentinel"} {
		matches, _, total, _, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{Query: query})
		if err != nil || total != 1 || len(matches) != 1 {
			t.Fatalf("query %q: matches=%+v total=%d err=%v", query, matches, total, err)
		}
	}
}
