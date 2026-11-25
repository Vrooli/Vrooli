package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

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

func TestAppendNativeEventDeduplicatesByProviderIdentity(t *testing.T) {
	store := NewConversationStore()
	provenance := &NativeProvenance{Provider: "codex", SessionID: "thread-1", TurnID: "turn-1", MessageID: "item-1", BoundaryID: "item-1"}
	first, result := store.AppendNativeEvent(context.Background(), "session-1", "codex_app_server", "assistant", "first text", provenance)
	if !result.Appended || result.Duplicate {
		t.Fatalf("first append = %+v", result)
	}
	second, result := store.AppendNativeEvent(context.Background(), "session-1", "codex_app_server", "assistant", "replayed text", provenance)
	if !result.Duplicate || second.ID != first.ID || second.Text != "first text" {
		t.Fatalf("replay = event %+v result %+v", second, result)
	}
}

func TestSQLNativeEventDedupUsesIndexedPointLookupWithoutSessionCache(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	provenance := &NativeProvenance{Provider: "claude", SessionID: "native-session", TurnID: "turn-1", MessageID: "message-1"}
	first, result := store.AppendNativeEvent(ctx, "native-session", "claude_archive", "assistant", "first message", provenance)
	if !result.Appended || result.Duplicate {
		t.Fatalf("first append = %+v", result)
	}
	duplicate, result := store.AppendNativeEvent(ctx, "native-session", "claude_archive", "assistant", "replayed message", provenance)
	if !result.Duplicate || duplicate.ID != first.ID || duplicate.Text != first.Text {
		t.Fatalf("duplicate lookup = event %+v result %+v", duplicate, result)
	}
	var indexed bool
	var planDetails []string
	rows, err := db.QueryContext(ctx, `EXPLAIN QUERY PLAN `+findNativeEventByIdentityQuery, "native-session", nativeProvenanceJSON(provenance))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "idx_conversation_events_native_identity") {
			indexed = true
		}
		planDetails = append(planDetails, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !indexed {
		t.Fatalf("native identity point lookup does not use idx_conversation_events_native_identity: %v", planDetails)
	}
}

func TestAppendNativeEventDedupTracksTruncateAndDelete(t *testing.T) {
	ctx := context.Background()
	store := NewConversationStore()
	provenance := &NativeProvenance{Provider: "codex", SessionID: "thread-cache", TurnID: "turn-cache", MessageID: "item-cache", BoundaryID: "item-cache"}
	first, result := store.AppendNativeEvent(ctx, "session-cache", "codex_archive", "assistant", "first", provenance)
	if !result.Appended || result.Duplicate {
		t.Fatalf("first append = %+v", result)
	}
	if err := store.TruncateSessionAfter(ctx, "session-cache", 0); err != nil {
		t.Fatal(err)
	}
	second, result := store.AppendNativeEvent(ctx, "session-cache", "codex_archive", "assistant", "after truncate", provenance)
	if !result.Appended || result.Duplicate || second.ID == first.ID {
		t.Fatalf("append after truncate = event %+v result %+v", second, result)
	}
	if err := store.DeleteSession(ctx, "session-cache"); err != nil {
		t.Fatal(err)
	}
	third, result := store.AppendNativeEvent(ctx, "session-cache", "codex_archive", "assistant", "after delete", provenance)
	if !result.Appended || result.Duplicate || third.ID == second.ID {
		t.Fatalf("append after delete = event %+v result %+v", third, result)
	}
}

func TestTruncateSessionAfterPreservesSelectedProjectionPrefix(t *testing.T) {
	store := NewConversationStore()
	first, _ := store.AppendAssistantEvent(context.Background(), "session-1", "test", "first")
	_, _ = store.AppendAssistantEvent(context.Background(), "session-1", "test", "second")
	if err := store.TruncateSessionAfter(context.Background(), "session-1", first.Sequence); err != nil {
		t.Fatal(err)
	}
	state := store.ListSession(context.Background(), "session-1")
	if len(state.Events) != 1 || state.Events[0].ID != first.ID {
		t.Fatalf("projection after truncate = %+v", state.Events)
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

func TestAppendEventAtKeepsSourceAndIngestTimeSeparate(t *testing.T) {
	ctx := context.Background()
	store := NewConversationStore()
	occurredAt := time.Date(2025, 10, 17, 14, 27, 0, 123456789, time.UTC)
	assistant, assistantResult := store.AppendAssistantEventAt(ctx, "assistant-session", "claude-tail", "older answer", occurredAt)
	user, userResult := store.AppendUserEventAt(ctx, "user-session", "codex-tail", "older question", occurredAt)
	for name, result := range map[string]ConversationAppendResult{"assistant": assistantResult, "user": userResult} {
		if !result.Appended {
			t.Fatalf("%s append failed: %+v", name, result)
		}
	}
	for name, event := range map[string]ConversationEvent{"assistant": assistant, "user": user} {
		if event.CreatedAt.IsZero() || !event.OccurredAt.Equal(occurredAt) || event.CreatedAt.Equal(event.OccurredAt) {
			t.Errorf("%s event ingest/source times = %s / %s, want distinct ingest and %s source time", name, event.CreatedAt, event.OccurredAt, occurredAt)
		}
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

	archived, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "archive needle", Limit: 20}})
	if err != nil {
		t.Fatalf("search archived: %v", err)
	}
	if archived.Truncated || archived.Total != 2 || archived.DistinctSessions != 2 || len(archived.Matches) != 2 {
		t.Fatalf("search result matches=%d total=%d distinct=%d truncated=%v", len(archived.Matches), archived.Total, archived.DistinctSessions, archived.Truncated)
	}
	for _, match := range archived.Matches {
		if match.SessionID == "live" {
			t.Fatalf("live session leaked into archive search: %+v", match)
		}
	}

	perSession, err := repo.SearchSession(ctx, "live", ConversationSearchQuery{Query: "archive needle", Limit: 20})
	if err != nil || len(perSession.Matches) != 1 || perSession.Total != 1 {
		t.Fatalf("existing per-session search changed: %+v err=%v", perSession, err)
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

func TestSQLSearchArchivedHarnessFilterIncludesNativeArchiveSources(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLConversationRepository(db)
	for _, row := range []struct{ sessionID, source string }{
		{"native-claude", "claude_archive"},
		{"native-codex", "codex_archive"},
	} {
		if _, err := repo.AppendEvent(ctx, ConversationEvent{
			ID: row.sessionID + "-event", SessionID: row.sessionID, Source: row.source,
			Role: ConversationRoleAssistant, Text: "archive-source-filter probe", CreatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	result, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{
		ConversationSearchQuery: ConversationSearchQuery{Query: "archive-source-filter probe", Limit: 10},
		AgentType:               "claude",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Matches) != 1 || result.Matches[0].SessionID != "native-claude" {
		t.Fatalf("claude-filtered native results = %+v; want only the native claude archive", result)
	}
}

func TestSQLSearchArchivedUsesSourceOccurrenceTimeAndPreservesIngestTime(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := ensureConversationOccurrenceIndex(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	cutoff := time.Date(2026, 11, 1, 0, 0, 0, 500_000_000, time.UTC)
	plans, err := db.QueryContext(ctx, `EXPLAIN QUERY PLAN SELECT id FROM conversation_events WHERE source = ? AND occurred_at >= ?`, "gemini-cli_archive", formatOccurredAt(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	usesOccurrenceIndex := false
	for plans.Next() {
		var id, parent, unused int
		var detail string
		if err := plans.Scan(&id, &parent, &unused, &detail); err != nil {
			plans.Close()
			t.Fatal(err)
		}
		usesOccurrenceIndex = usesOccurrenceIndex || strings.Contains(detail, "idx_conversation_events_source_occurred_at")
	}
	if err := plans.Close(); err != nil {
		t.Fatal(err)
	}
	if !usesOccurrenceIndex {
		t.Fatal("native occurrence-time range did not use idx_conversation_events_source_occurred_at")
	}
	appendNative := func(messageID string, occurredAt time.Time) ConversationEvent {
		provenance := &NativeProvenance{Provider: "gemini-cli", SessionID: "native-session", TurnID: "turn-" + messageID, MessageID: messageID}
		event, result := store.AppendNativeEventAt(ctx, "native-session", "gemini-cli_archive", "assistant", "timestamp filter marker", provenance, occurredAt)
		if !result.Appended {
			t.Fatalf("append native message %s: %+v", messageID, result)
		}
		return event
	}
	old := appendNative("old", cutoff.Add(-time.Hour))
	current := appendNative("current", cutoff.Add(time.Hour))
	unknown := appendNative("unknown", time.Time{})
	boundary := appendNative("boundary", cutoff.Truncate(time.Second))
	ordinaryBefore, ordinaryBeforeResult := store.AppendAssistantEvent(ctx, "ordinary-before", "test", "timestamp filter marker")
	ordinaryAfter, ordinaryAfterResult := store.AppendAssistantEvent(ctx, "ordinary-after", "test", "timestamp filter marker")
	if !ordinaryBeforeResult.Appended || !ordinaryAfterResult.Appended {
		t.Fatalf("append ordinary rows: before=%+v after=%+v", ordinaryBeforeResult, ordinaryAfterResult)
	}
	if _, err := db.ExecContext(ctx, `UPDATE conversation_events SET created_at = ? WHERE id = ?`, formatTime(cutoff.Truncate(time.Second)), ordinaryBefore.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE conversation_events SET created_at = ? WHERE id = ?`, formatTime(cutoff.Add(time.Second)), ordinaryAfter.ID); err != nil {
		t.Fatal(err)
	}
	unknownProvenance := &NativeProvenance{Provider: "gemini-cli", SessionID: "native-session", TurnID: "turn-unknown", MessageID: "unknown"}
	backfilledOccurrence := cutoff.Add(-2 * time.Hour)
	replayed, duplicate := store.AppendNativeEventAt(ctx, "native-session", "gemini-cli_archive", "assistant", "timestamp filter marker", unknownProvenance, backfilledOccurrence)
	if !duplicate.Duplicate || replayed.ID != unknown.ID || replayed.Sequence != unknown.Sequence || !replayed.CreatedAt.Equal(unknown.CreatedAt) || !replayed.OccurredAt.Equal(backfilledOccurrence) {
		t.Fatalf("timestamp backfill replay changed identity/ingest time or failed to fill occurrence time: event=%+v result=%+v", replayed, duplicate)
	}
	if old.CreatedAt.IsZero() || current.CreatedAt.IsZero() || old.CreatedAt.Equal(old.OccurredAt) {
		t.Fatalf("ingestion and source occurrence timestamps were conflated: old=%+v current=%+v", old, current)
	}

	result, err := store.SearchArchived(ctx, ArchivedConversationSearchFilter{
		ConversationSearchQuery: ConversationSearchQuery{Query: "timestamp filter marker", Limit: 10},
		CreatedAfter:            cutoff,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 2 {
		t.Fatalf("date filtered matches = %+v, want after-cutoff native and ordinary events", result.Matches)
	}
	matchedIDs := map[string]bool{}
	for _, match := range result.Matches {
		matchedIDs[match.EventID] = true
	}
	if !matchedIDs[current.ID] || !matchedIDs[ordinaryAfter.ID] || matchedIDs[boundary.ID] || matchedIDs[ordinaryBefore.ID] || matchedIDs[unknown.ID] {
		t.Fatalf("date filter violated native/ordinary time precision or unknown exclusion: %+v", result.Matches)
	}
	for _, match := range result.Matches {
		if match.EventID == current.ID && !match.OccurredAt.Equal(current.OccurredAt) {
			t.Fatalf("search source time = %s, want %s", match.OccurredAt, current.OccurredAt)
		}
	}
	if !unknown.OccurredAt.IsZero() {
		t.Fatalf("missing source timestamp was inferred: %s", unknown.OccurredAt)
	}
}

func TestNativeOccurrenceReplayResetClearsOnlyCursorOnceAndKeepsEvents(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS web_console_migrations (name TEXT PRIMARY KEY, last_rowid INTEGER NOT NULL DEFAULT 0, completed_at TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	checkpoints := NewSQLAgentTranscriptCheckpointStore(db)
	provenance := &NativeProvenance{Provider: "claude", SessionID: "native-session", TurnID: "turn-1", MessageID: "message-1"}
	event, result := store.AppendNativeEvent(ctx, "native-session", "claude_archive", "assistant", "replay migration marker", provenance)
	if !result.Appended {
		t.Fatalf("append prior native projection: %+v", result)
	}
	priorCheckpoint := AgentTranscriptCheckpoint{Source: "native-archive-index", SourceKey: "archive-key", SessionID: "native-session", Cursor: "42"}
	if err := checkpoints.Save(ctx, priorCheckpoint); err != nil {
		t.Fatal(err)
	}
	if err := checkpoints.ResetNativeArchiveOccurrenceCursors(ctx, "native-archive-index", nativeArchiveOccurrenceTimeReplayMigration); err != nil {
		t.Fatal(err)
	}
	if _, found, err := checkpoints.Get(ctx, priorCheckpoint.Source, priorCheckpoint.SourceKey); err != nil || found {
		t.Fatalf("cursor after first migration reset found=%v err=%v", found, err)
	}
	if preserved, found := store.GetEvent(ctx, "native-session", event.ID); !found || preserved.ID != event.ID || preserved.Sequence != event.Sequence {
		t.Fatalf("projection was not preserved across cursor reset: %+v found=%v", preserved, found)
	}
	if err := checkpoints.Save(ctx, priorCheckpoint); err != nil {
		t.Fatal(err)
	}
	if err := checkpoints.ResetNativeArchiveOccurrenceCursors(ctx, "native-archive-index", nativeArchiveOccurrenceTimeReplayMigration); err != nil {
		t.Fatal(err)
	}
	if resumed, found, err := checkpoints.Get(ctx, priorCheckpoint.Source, priorCheckpoint.SourceKey); err != nil || !found || resumed.Cursor != priorCheckpoint.Cursor {
		t.Fatalf("one-time migration cleared a subsequent cursor: %+v found=%v err=%v", resumed, found, err)
	}
	occurrence := time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC)
	replayed, duplicate := store.AppendNativeEventAt(ctx, "native-session", "claude_archive", "assistant", "replayed text", provenance, occurrence)
	if !duplicate.Duplicate || replayed.ID != event.ID || replayed.Sequence != event.Sequence || !replayed.CreatedAt.Equal(event.CreatedAt) || !replayed.OccurredAt.Equal(occurrence) {
		t.Fatalf("idempotent occurrence backfill = %+v result=%+v, prior=%+v", replayed, duplicate, event)
	}
}

func TestSQLSearchArchivedFindsTranscriptWhenSessionMetadataIsMissing(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("ensure FTS: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions(id, archived_at) VALUES ('orphan', '2026-09-04T17:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLConversationRepository(db)
	if _, err := repo.AppendEvent(ctx, ConversationEvent{ID: "orphan-event", SessionID: "orphan", Source: "codex_tailer", Role: ConversationRoleAssistant, Text: "plan-manager continuity evidence", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM sessions WHERE id = 'orphan'`); err != nil {
		t.Fatal(err)
	}
	found, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "plan-manager continuity"}})
	if err != nil {
		t.Fatal(err)
	}
	if found.Total != 1 || found.DistinctSessions != 1 || len(found.Matches) != 1 || found.Matches[0].SessionID != "orphan" {
		t.Fatalf("orphan search = %+v", found)
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
		found, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: query}})
		if err != nil || found.Total != 1 || len(found.Matches) != 1 {
			t.Fatalf("query %q: %+v err=%v", query, found, err)
		}
	}
}

func TestVoiceHandoffReceiptDoesNotCollapseIntoMatchingTypedPrompt(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	_, typed := store.AppendUserEvent(ctx, "session", "codex_tailer", "Run tests")
	event, handoff := store.AppendUserEvent(ctx, "session", codexVoiceHandoffSource, "Run tests")
	if !typed.Appended || !handoff.Appended || handoff.Duplicate || event.Source != codexVoiceHandoffSource {
		t.Fatalf("typed=%+v handoff=%+v event=%+v", typed, handoff, event)
	}
	_, replay := store.AppendUserEvent(ctx, "session", codexVoiceHandoffSource, "Run tests")
	if !replay.Duplicate {
		t.Fatalf("handoff replay was not deduplicated: %+v", replay)
	}
	state := store.ListSession(ctx, "session")
	if len(state.Events) != 2 {
		t.Fatalf("events=%+v", state.Events)
	}
}
