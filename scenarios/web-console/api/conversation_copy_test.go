package main

import (
	"testing"
)

// appendCopyTestEvent inserts one event marked fully seen/played/listened so
// the copy test can assert per-event state is carried over verbatim.
func appendCopyTestEvent(t *testing.T, repo ConversationRepository, sessionID, id, text string, role ConversationRole) {
	t.Helper()
	if _, err := repo.AppendEvent(ConversationEvent{
		ID:               id,
		SessionID:        sessionID,
		Source:           "test",
		Role:             role,
		Text:             text,
		SpeechParagraphs: []string{text},
		DeliveryState:    ConversationDeliverySeen,
		TTSState:         ConversationTTSPlayed,
		ConsumptionState: ConversationConsumptionListened,
	}); err != nil {
		t.Fatalf("append event %q: %v", id, err)
	}
}

func TestSQLCopySession_PreservesHistoryStateAndSequence(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLConversationRepository(db)
	const oldID, newID = "old-session", "new-session"

	appendCopyTestEvent(t, repo, oldID, "evt-1", "first turn", ConversationRoleUser)
	appendCopyTestEvent(t, repo, oldID, "evt-2", "second turn", ConversationRoleAssistant)
	appendCopyTestEvent(t, repo, oldID, "evt-3", "third turn", ConversationRoleAssistant)

	seen, listened := int64(2), int64(2)
	if _, err := repo.UpdateCursor(oldID, conversationCursorPatch{seenSequence: &seen, listenedSequence: &listened}); err != nil {
		t.Fatalf("update cursor: %v", err)
	}

	if err := repo.CopySession(oldID, newID); err != nil {
		t.Fatalf("copy session: %v", err)
	}

	got, err := repo.ListSession(newID)
	if err != nil {
		t.Fatalf("list new session: %v", err)
	}
	if len(got.Events) != 3 {
		t.Fatalf("want 3 copied events, got %d", len(got.Events))
	}

	wants := []struct {
		seq    int64
		text   string
		origID string
	}{{1, "first turn", "evt-1"}, {2, "second turn", "evt-2"}, {3, "third turn", "evt-3"}}
	for i, w := range wants {
		ev := got.Events[i]
		if ev.Sequence != w.seq || ev.Text != w.text {
			t.Errorf("event %d: got seq=%d text=%q, want seq=%d text=%q", i, ev.Sequence, ev.Text, w.seq, w.text)
		}
		if ev.SessionID != newID {
			t.Errorf("event %d: session id not rekeyed to %q: %q", i, newID, ev.SessionID)
		}
		if ev.ID == w.origID {
			t.Errorf("event %d: id should be regenerated, still %q", i, ev.ID)
		}
		if ev.DeliveryState != ConversationDeliverySeen || ev.TTSState != ConversationTTSPlayed || ev.ConsumptionState != ConversationConsumptionListened {
			t.Errorf("event %d: per-event state not preserved: delivery=%q tts=%q consumption=%q", i, ev.DeliveryState, ev.TTSState, ev.ConsumptionState)
		}
	}

	if got.Cursor.LastSeenSequence != 2 || got.Cursor.LastListenedSequence != 2 {
		t.Errorf("cursor not preserved: %+v", got.Cursor)
	}

	// A freshly appended event must continue numbering after the copied tail,
	// not collide with sequence 1..3.
	next, err := repo.AppendEvent(ConversationEvent{ID: "evt-new", SessionID: newID, Role: ConversationRoleAssistant, Text: "fourth turn", SpeechParagraphs: []string{"fourth turn"}})
	if err != nil {
		t.Fatalf("append after copy: %v", err)
	}
	if next.Sequence != 4 {
		t.Errorf("next sequence after copy: got %d, want 4", next.Sequence)
	}

	// The source session is untouched by the copy.
	src, err := repo.ListSession(oldID)
	if err != nil {
		t.Fatalf("list old session: %v", err)
	}
	if len(src.Events) != 3 {
		t.Errorf("source session mutated by copy: got %d events, want 3", len(src.Events))
	}
}

func TestSQLCopySession_MissingSourceIsNoop(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLConversationRepository(db)
	if err := repo.CopySession("missing", "dest"); err != nil {
		t.Fatalf("copy with missing source should be a no-op, got %v", err)
	}
	got, err := repo.ListSession("dest")
	if err != nil {
		t.Fatalf("list dest: %v", err)
	}
	if len(got.Events) != 0 {
		t.Errorf("want 0 events for dest with no source, got %d", len(got.Events))
	}
}

func TestInMemoryCopySession_PreservesHistoryAndSequence(t *testing.T) {
	repo := NewInMemoryConversationRepository()
	const oldID, newID = "old", "new"
	appendCopyTestEvent(t, repo, oldID, "e1", "alpha", ConversationRoleUser)
	appendCopyTestEvent(t, repo, oldID, "e2", "beta", ConversationRoleAssistant)

	if err := repo.CopySession(oldID, newID); err != nil {
		t.Fatalf("copy: %v", err)
	}
	got, _ := repo.ListSession(newID)
	if len(got.Events) != 2 {
		t.Fatalf("want 2 copied events, got %d", len(got.Events))
	}
	if got.Events[0].SessionID != newID || got.Events[0].ID == "e1" {
		t.Errorf("event not rekeyed/regenerated: id=%q session=%q", got.Events[0].ID, got.Events[0].SessionID)
	}

	next, err := repo.AppendEvent(ConversationEvent{ID: "e3", SessionID: newID, Role: ConversationRoleAssistant, Text: "gamma"})
	if err != nil {
		t.Fatalf("append after copy: %v", err)
	}
	if next.Sequence != 3 {
		t.Errorf("next sequence after copy: got %d, want 3", next.Sequence)
	}
}
