package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"web-console/internal/continuity"
)

type retentionPrunerStub struct {
	removed int64
	calls   int
	err     error
}

func (p *retentionPrunerStub) PruneEvents(context.Context, time.Time, int) (int64, error) {
	p.calls++
	return p.removed, p.err
}

type retentionLedgerStub struct {
	put       int
	completed int
	err       error
}

func (l *retentionLedgerStub) Put(_ context.Context, _ continuity.Receipt) (continuity.Receipt, error) {
	l.put++
	if l.err != nil {
		return continuity.Receipt{}, l.err
	}
	return continuity.Receipt{Status: "pending"}, nil
}

func (l *retentionLedgerStub) Complete(_ context.Context, _, _, _ string, _ time.Time) (continuity.Receipt, error) {
	l.completed++
	if l.err != nil {
		return continuity.Receipt{}, l.err
	}
	return continuity.Receipt{Status: "succeeded"}, nil
}

func (*retentionLedgerStub) Get(context.Context, string) (continuity.Receipt, error) {
	return continuity.Receipt{}, errors.New("not implemented")
}

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
	found, err := repo.SearchArchived(ctx, ArchivedConversationSearchFilter{ConversationSearchQuery: ConversationSearchQuery{Query: "event", Limit: 20}})
	if err != nil {
		t.Fatalf("search after prune: %v", err)
	}
	if len(found.Matches) != 2 || found.Total != 2 {
		t.Fatalf("FTS/index mismatch after prune: matches=%d total=%d", len(found.Matches), found.Total)
	}
}

func TestConversationRetentionSweeperSkipsWhenUnbounded(t *testing.T) {
	repo := NewInMemoryConversationRepository()
	sweeper := newConversationRetentionSweeper(repo, func() int { return 0 }, func() int { return 0 })
	if got := sweeper.sweep(); got != 0 {
		t.Fatalf("unbounded sweep removed %d events", got)
	}
}

func TestConversationRetentionSweeperRequiresDurableReceipt(t *testing.T) {
	pruner := &retentionPrunerStub{removed: 7}
	withoutLedger := newConversationRetentionSweeper(pruner, func() int { return 30 }, func() int { return 0 })
	if got := withoutLedger.sweep(); got != 0 || pruner.calls != 0 {
		t.Fatalf("unreceipted sweep removed %d events", got)
	}

	ledger := &retentionLedgerStub{}
	withLedger := newConversationRetentionSweeper(pruner, func() int { return 30 }, func() int { return 0 }, ledger)
	if got := withLedger.sweep(); got != 7 {
		t.Fatalf("receipted sweep removed %d events, want 7", got)
	}
	if pruner.calls != 1 {
		t.Fatalf("receipted sweep pruner calls = %d, want 1", pruner.calls)
	}
	if ledger.put != 1 || ledger.completed != 1 {
		t.Fatalf("retention receipt calls = put:%d complete:%d, want 1/1", ledger.put, ledger.completed)
	}
}

func TestConversationRetentionSweeperDoesNotPruneWhenReceiptCreationFails(t *testing.T) {
	pruner := &retentionPrunerStub{removed: 7}
	ledger := &retentionLedgerStub{err: errors.New("ledger unavailable")}
	sweeper := newConversationRetentionSweeper(pruner, func() int { return 30 }, func() int { return 0 }, ledger)
	if got := sweeper.sweep(); got != 0 {
		t.Fatalf("failed-receipt sweep removed %d events", got)
	}
	if pruner.calls != 0 {
		t.Fatalf("failed-receipt sweep called pruner %d times", pruner.calls)
	}
	if ledger.put != 1 || ledger.completed != 0 {
		t.Fatalf("failed receipt calls = put:%d complete:%d, want 1/0", ledger.put, ledger.completed)
	}
}
