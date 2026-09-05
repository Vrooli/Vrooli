package main

import (
	"context"
	"testing"
	"time"
)

func TestSQLConversationRepositoryPruneEventsHonorsAgeAndPerSessionCap(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLConversationRepository(db)
	ctx := context.Background()
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatalf("ensure FTS: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, archived_at) VALUES ('retention-session', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert archived session: %v", err)
	}
	old := time.Now().UTC().Add(-48 * time.Hour)
	for i, created := range []time.Time{old, old, time.Now().UTC()} {
		if _, err := repo.AppendEvent(ctx, ConversationEvent{
			ID: "retention-" + string(rune('a'+i)), SessionID: "retention-session",
			Source: "test", Role: ConversationRoleAssistant, Text: "event",
			CreatedAt: created, DeliveryState: ConversationDeliveryPending,
			TTSState: ConversationTTSIdle, ConsumptionState: ConversationConsumptionUnseen,
		}); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}
	removed, err := repo.PruneEvents(ctx, old.Add(time.Hour), 0)
	if err != nil || removed != 2 {
		t.Fatalf("age prune = %d, %v; want 2", removed, err)
	}

	for i := 3; i < 7; i++ {
		if _, err := repo.AppendEvent(ctx, ConversationEvent{
			ID: "retention-" + string(rune('a'+i)), SessionID: "retention-session",
			Source: "test", Role: ConversationRoleUser, Text: "event",
			CreatedAt: time.Now().UTC(), DeliveryState: ConversationDeliveryPending,
			TTSState: ConversationTTSIdle, ConsumptionState: ConversationConsumptionUnseen,
		}); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}
	removed, err = repo.PruneEvents(ctx, time.Time{}, 2)
	if err != nil || removed != 3 {
		t.Fatalf("count prune = %d, %v; want 3", removed, err)
	}
	state, err := repo.ListSession(ctx, "retention-session")
	if err != nil || len(state.Events) != 2 {
		t.Fatalf("remaining events = %d, %v; want 2", len(state.Events), err)
	}

	// The external-content FTS delete trigger must remove pruned text too.
	matches, _, total, _, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{Query: "event", Limit: 20})
	if err != nil {
		t.Fatalf("search after prune: %v", err)
	}
	if len(matches) != 2 || total != 2 {
		t.Fatalf("FTS/index mismatch after prune: matches=%d total=%d", len(matches), total)
	}
}

func TestConversationRetentionSweeperSkipsWhenUnbounded(t *testing.T) {
	repo := NewInMemoryConversationRepository()
	sweeper := newConversationRetentionSweeper(repo, func() int { return 0 }, func() int { return 0 })
	if got := sweeper.sweep(); got != 0 {
		t.Fatalf("unbounded sweep removed %d events", got)
	}
}
