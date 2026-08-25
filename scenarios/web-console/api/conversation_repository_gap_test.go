package main

import (
	"context"
	"testing"
	"time"

	conversationH "web-console/handlers/conversation"
)

func TestSQLConversationRepositoryLifecycleAndCursorState(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLConversationRepository(setupTestDB(t))
	created := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	for i, text := range []string{"first message", "second message", "third message"} {
		if _, err := repo.AppendEvent(ctx, ConversationEvent{
			ID: "gap-event-" + string(rune('1'+i)), SessionID: "gap-session", Source: "test",
			Role: ConversationRoleAssistant, Text: text, SpeechParagraphs: []string{text}, CreatedAt: created.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if count, err := repo.CountSessionEvents(ctx, "gap-session"); err != nil || count != 3 {
		t.Fatalf("count = %d, err=%v", count, err)
	}
	if size, err := repo.SessionStorageBytes(ctx, "gap-session"); err != nil || size <= 0 {
		t.Fatalf("storage size = %d, err=%v", size, err)
	}
	if _, found, err := repo.GetEvent(ctx, "gap-session", "missing"); err != nil || found {
		t.Fatalf("missing event found=%v err=%v", found, err)
	}

	rangeEvents, err := repo.ListSessionRange(ctx, "gap-session", 2, 3)
	if err != nil || len(rangeEvents) != 2 || rangeEvents[0].Sequence != 2 || rangeEvents[1].Sequence != 3 {
		t.Fatalf("range=%+v err=%v", rangeEvents, err)
	}

	if err := repo.UpdateSpeechParagraphs(ctx, "gap-session", "gap-event-1", []string{"rephrased"}); err != nil {
		t.Fatal(err)
	}
	updated, found, err := repo.GetEvent(ctx, "gap-session", "gap-event-1")
	if err != nil || !found || !updated.Summarized || len(updated.SpeechParagraphs) != 1 || updated.SpeechParagraphs[0] != "rephrased" || updated.OriginalSpeechParagraphs[0] != "first message" {
		t.Fatalf("updated event=%+v found=%v err=%v", updated, found, err)
	}

	seen, listened := int64(3), int64(2)
	cursor, err := repo.UpdateCursor(ctx, "gap-session", conversationCursorPatch{seenSequence: &seen, listenedSequence: &listened})
	if err != nil || cursor.LastSeenSequence != 3 || cursor.LastListenedSequence != 2 {
		t.Fatalf("cursor=%+v err=%v", cursor, err)
	}
	lower := int64(1)
	cursor, err = repo.UpdateCursor(ctx, "gap-session", conversationCursorPatch{seenSequence: &lower})
	if err != nil || cursor.LastSeenSequence != 3 {
		t.Fatalf("cursor regressed to %+v err=%v", cursor, err)
	}

	if err := repo.RecordPlaybackStage(ctx, "gap-session", "gap-event-3", "playback_started"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordPlaybackStage(ctx, "gap-session", "gap-event-3", "playback_succeeded"); err != nil {
		t.Fatal(err)
	}
	played, _, err := repo.GetEvent(ctx, "gap-session", "gap-event-3")
	if err != nil || played.TTSState != ConversationTTSPlayed || played.ConsumptionState != ConversationConsumptionListened || played.DeliveryState != ConversationDeliverySeen {
		t.Fatalf("played event=%+v err=%v", played, err)
	}

	if err := repo.DeleteSession(ctx, "gap-session"); err != nil {
		t.Fatal(err)
	}
	if count, err := repo.CountSessionEvents(ctx, "gap-session"); err != nil || count != 0 {
		t.Fatalf("post-delete count = %d, err=%v", count, err)
	}
}

func TestInMemoryConversationRepositoryLifecycleMethods(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryConversationRepository()
	event, err := repo.AppendEvent(ctx, ConversationEvent{ID: "memory-event", SessionID: "memory-session", Role: ConversationRoleAssistant, Text: "memory needle", SpeechParagraphs: []string{"memory needle"}})
	if err != nil {
		t.Fatal(err)
	}
	if count, err := repo.CountSessionEvents(ctx, "memory-session"); err != nil || count != 1 {
		t.Fatalf("count = %d, err=%v", count, err)
	}
	if size, err := repo.SessionStorageBytes(ctx, "memory-session"); err != nil || size <= 0 {
		t.Fatalf("storage size = %d, err=%v", size, err)
	}
	if matches, _, total, err := repo.SearchSession(ctx, "memory-session", "needle", 10); err != nil || len(matches) != 1 || total != 1 {
		t.Fatalf("search=%+v total=%d err=%v", matches, total, err)
	}
	if events, err := repo.ListSessionRange(ctx, "memory-session", event.Sequence, event.Sequence); err != nil || len(events) != 1 {
		t.Fatalf("range=%+v err=%v", events, err)
	}
	if err := repo.UpdateSpeechParagraphs(ctx, "memory-session", event.ID, []string{"updated"}); err != nil {
		t.Fatal(err)
	}
	seen := int64(1)
	if _, err := repo.UpdateCursor(ctx, "memory-session", conversationCursorPatch{seenSequence: &seen}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordPlaybackStage(ctx, "memory-session", event.ID, "playback_succeeded"); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteSession(ctx, "memory-session"); err != nil {
		t.Fatal(err)
	}
}

func TestConversationStoreDelegatesRepositoryOperations(t *testing.T) {
	ctx := context.Background()
	store := NewConversationStore()
	event, result := store.AppendAssistantEvent(ctx, "store-session", "test", "store needle")
	if !result.Appended {
		t.Fatalf("append result=%+v", result)
	}
	if got, ok := store.GetEvent(ctx, "store-session", event.ID); !ok || got.ID != event.ID {
		t.Fatalf("store event=%+v ok=%v", got, ok)
	}
	if store.CountSessionEvents(ctx, "store-session") != 1 || store.SessionStorageBytes(ctx, "store-session") <= 0 {
		t.Fatal("store count or storage size not delegated")
	}
	if matches, _, total, err := store.SearchSession(ctx, "store-session", "needle", 10); err != nil || len(matches) != 1 || total != 1 {
		t.Fatalf("store search=%+v total=%d err=%v", matches, total, err)
	}
	if events, err := store.ListSessionRange(ctx, "store-session", 1, 1); err != nil || len(events) != 1 {
		t.Fatalf("store range=%+v err=%v", events, err)
	}
	if !store.HasConversationAfter(ctx, "store-session", event.CreatedAt.Add(-time.Second)) {
		t.Fatal("store did not report a recent conversation")
	}
	seen := int64(1)
	if cursor := store.UpdateCursor(ctx, "store-session", conversationCursorPatch{seenSequence: &seen}); cursor.LastSeenSequence != 1 {
		t.Fatalf("store cursor=%+v", cursor)
	}
	store.RecordPlaybackStage(ctx, "store-session", event.ID, "seen")
	store.UpdateSpeechParagraphs(ctx, "store-session", event.ID, []string{"summarized store needle"})
	store.DeleteSession(ctx, "store-session")
	if store.CountSessionEvents(ctx, "store-session") != 0 {
		t.Fatal("store delete did not remove session")
	}
}

func TestConversationAdapterProjectsStoreOperations(t *testing.T) {
	ctx := context.Background()
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(ctx, "", 80, 24, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(ctx, sess.ID) })
	if _, result := srv.conversations.AppendAssistantEvent(ctx, sess.ID, "test", "adapter needle"); !result.Appended {
		t.Fatalf("append result=%+v", result)
	}
	adapter := newConversationAdapter(srv)

	state, err := adapter.Get(sess.ID, 0, 10, 0)
	if err != nil || len(state.Events) != 1 || state.TotalCount != 1 || state.OldestSequence != 1 || state.NewestSequence != 1 {
		t.Fatalf("get state=%+v err=%v", state, err)
	}
	if matches, _, total, err := adapter.Search(sess.ID, "needle", 10); err != nil || len(matches) != 1 || total != 1 {
		t.Fatalf("search=%+v total=%d err=%v", matches, total, err)
	}
	if ranged, err := adapter.GetRange(sess.ID, 1, 1); err != nil || len(ranged.Events) != 1 {
		t.Fatalf("range=%+v err=%v", ranged, err)
	}
	cursor, err := adapter.UpdateCursor(sess.ID, conversationH.CursorPatch{HasLastSeenSequence: true, LastSeenSequence: 1})
	if err != nil || cursor.LastSeenSequence != 1 {
		t.Fatalf("cursor=%+v err=%v", cursor, err)
	}
	if summary, err := adapter.SummarizeEvent(ctx, sess.ID, state.Events[0].ID); err != nil || summary.Error == "" {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	if _, err := adapter.SearchArchived(ctx, conversationH.ArchivedSearchFilter{CreatedAfter: "not-rfc3339"}); err == nil {
		t.Fatal("invalid archive timestamp was accepted")
	}
	if _, err := adapter.SearchArchived(ctx, conversationH.ArchivedSearchFilter{Query: "needle"}); err != nil {
		t.Fatal(err)
	}
}
