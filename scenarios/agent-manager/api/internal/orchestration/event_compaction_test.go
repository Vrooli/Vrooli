package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/config"
)

type compactionStoreStub struct {
	retentionStoreStub
	cutoff          time.Time
	minBytes, limit int
	calls           int
}

func (s *compactionStoreStub) CompactImportedToolPayloads(_ context.Context, cutoff time.Time, minBytes, limit int) (int, error) {
	s.calls++
	s.cutoff, s.minBytes, s.limit = cutoff, minBytes, limit
	return 5, nil
}

func TestReconcilerImportedCompactionUsesLeversAndBoundedBatch(t *testing.T) {
	store := &compactionStoreStub{}
	levers := config.DefaultLevers()
	levers.Storage.ImportedToolCompactionDays = 12
	levers.Storage.ImportedToolCompactionMinBytes = 8192
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	reconciler := NewReconciler(nil, nil, WithReconcilerLevers(levers), WithReconcilerEventRetention(store))
	reconciler.clock = func() time.Time { return now }

	compacted, err := reconciler.compactImportedEventPayloads(context.Background())

	if err != nil || compacted != 5 {
		t.Fatalf("compactImportedEventPayloads = %d, %v", compacted, err)
	}
	if !store.cutoff.Equal(now.Add(-12*24*time.Hour)) || store.minBytes != 8192 || store.limit != eventRetentionBatchSize {
		t.Fatalf("compaction input = cutoff %s, min %d, limit %d", store.cutoff, store.minBytes, store.limit)
	}
}

func TestReconcilerImportedCompactionDisabledByLever(t *testing.T) {
	store := &compactionStoreStub{}
	levers := config.DefaultLevers()
	levers.Storage.ImportedToolCompactionDays = 0
	reconciler := NewReconciler(nil, nil, WithReconcilerLevers(levers), WithReconcilerEventRetention(store))

	if compacted, err := reconciler.compactImportedEventPayloads(context.Background()); err != nil || compacted != 0 || store.calls != 0 {
		t.Fatalf("disabled compaction ran: %d, %v, calls=%d", compacted, err, store.calls)
	}
}
