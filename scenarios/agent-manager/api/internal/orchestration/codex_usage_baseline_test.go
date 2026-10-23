package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type codexBaselineStore struct {
	event.Store
	summary *event.Summary
	rows    []*domain.RunEvent
	gotOpts event.GetOptions
}

func (s *codexBaselineStore) Summarize(context.Context, uuid.UUID, event.GetOptions) (*event.Summary, error) {
	return s.summary, nil
}

func (s *codexBaselineStore) Get(_ context.Context, _ uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	s.gotOpts = opts
	return s.rows, nil
}

func codexUsageReceipt(snapshot *domain.CodexUsageSnapshot) *domain.RunEvent {
	return &domain.RunEvent{EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{
		RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true,
		CodexUsageSnapshot: snapshot,
	}}
}

func TestContinuationCodexUsageBaselineUsesBoundedLatestCompatibleReceipt(t *testing.T) {
	store := &codexBaselineStore{
		summary: &event.Summary{Counts: map[string]int64{string(domain.EventTypeMetric): 900}},
		rows: []*domain.RunEvent{
			codexUsageReceipt(&domain.CodexUsageSnapshot{SessionID: "old", InputTokens: 10}),
			codexUsageReceipt(&domain.CodexUsageSnapshot{SessionID: "current", InputTokens: 90, CachedInputTokens: 20, OutputTokens: 30}),
		},
	}
	o := &Orchestrator{events: store}
	runID := uuid.New()
	baseline, err := o.continuationCodexUsageBaseline(context.Background(), runID, "current")
	if err != nil {
		t.Fatal(err)
	}
	if baseline.SessionID != "current" || baseline.Snapshot == nil || baseline.Snapshot.InputTokens != 90 || baseline.Snapshot.CachedInputTokens != 20 || baseline.Snapshot.OutputTokens != 30 {
		t.Fatalf("baseline=%+v", baseline)
	}
	if store.gotOpts.Limit != codexUsageBaselineTail || store.gotOpts.Offset != 900-codexUsageBaselineTail || len(store.gotOpts.EventTypes) != 1 || store.gotOpts.EventTypes[0] != domain.EventTypeMetric {
		t.Fatalf("baseline lookup was not bounded to the metric tail: %+v", store.gotOpts)
	}
}

func TestContinuationCodexUsageBaselineDoesNotFallBackPastLatestLegacyReceipt(t *testing.T) {
	store := &codexBaselineStore{
		summary: &event.Summary{Counts: map[string]int64{string(domain.EventTypeMetric): 2}},
		rows: []*domain.RunEvent{
			codexUsageReceipt(&domain.CodexUsageSnapshot{SessionID: "current", InputTokens: 10}),
			codexUsageReceipt(nil), // A newer legacy authoritative receipt has no raw counter.
		},
	}
	o := &Orchestrator{events: store}
	baseline, err := o.continuationCodexUsageBaseline(context.Background(), uuid.New(), "current")
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Snapshot != nil {
		t.Fatalf("stale older snapshot was reused: %+v", baseline.Snapshot)
	}
}
