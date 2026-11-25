package main

import (
	"context"
	"fmt"
	"testing"
)

func assertBulkResolveOrder(t *testing.T, repo ConversationRepository, count int) {
	t.Helper()
	ctx := context.Background()
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("bulk-%05d", i)
		if _, err := repo.AppendEvent(ctx, ConversationEvent{ID: id, SessionID: "bulk-session", Role: ConversationRoleUser, Text: id}); err != nil {
			t.Fatalf("append event %s: %v", id, err)
		}
	}

	ids := make([]string, 0, count+3)
	for i := count; i >= 1; i-- {
		ids = append(ids, fmt.Sprintf("bulk-%05d", i))
	}
	ids = append(ids, "bulk-00001", "bulk-missing", "bulk-00003")
	events, missing, err := NewConversationStoreWithRepository(repo).ResolveEvents(ctx, "bulk-session", ids)
	if err != nil {
		t.Fatalf("resolve %d events: %v", count, err)
	}
	if len(events) != count || len(missing) != 1 || missing[0] != "bulk-missing" {
		t.Fatalf("resolved=%d missing=%v; want %d and [bulk-missing]", len(events), missing, count)
	}
	for i, event := range events {
		want := fmt.Sprintf("bulk-%05d", count-i)
		if event.ID != want {
			t.Fatalf("event[%d]=%q; want %q", i, event.ID, want)
		}
	}
}

func TestMemoryResolveEventsBulkPreservesLargeStableIDSelection(t *testing.T) {
	assertBulkResolveOrder(t, NewInMemoryConversationRepository(), 6005)
}

func TestSQLResolveEventsBulkPreservesLargeStableIDSelection(t *testing.T) {
	assertBulkResolveOrder(t, NewSQLConversationRepository(setupTestDB(t)), 805)
}
