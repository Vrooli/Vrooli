package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/pricing"

	"github.com/google/uuid"
)

type recordingQuotaObservationStore struct {
	rows []pricing.QuotaObservation
}

func (s *recordingQuotaObservationStore) RecordQuotaObservation(_ context.Context, observation *pricing.QuotaObservation) error {
	s.rows = append(s.rows, *observation)
	return nil
}

func (s *recordingQuotaObservationStore) ListQuotaObservations(_ context.Context, provider, pool, window string, limit int) ([]pricing.QuotaObservation, error) {
	result := make([]pricing.QuotaObservation, 0, len(s.rows))
	for i := len(s.rows) - 1; i >= 0 && len(result) < limit; i-- {
		row := s.rows[i]
		if (provider != "" && row.Provider != provider) || (pool != "" && row.Pool != pool) || (window != "" && row.Window != window) {
			continue
		}
		result = append(result, row)
	}
	return result, nil
}

func TestRunEventSinkPersistsProviderQuotaObservationAfterEventEvidence(t *testing.T) {
	runID := uuid.New()
	observedAt := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	store := mocks.NewFakeEventStore()
	quota := &recordingQuotaObservationStore{}
	o := &Orchestrator{
		events:            store,
		quotaObservations: quota,
		clock:             func() time.Time { return observedAt.Add(time.Minute) },
	}
	percent := 42.5
	event := domain.NewRateLimitEvent(runID, "5h", "native quota", nil, 0)
	event.Timestamp = observedAt
	data := event.Data.(*domain.RateLimitEventData)
	data.Provider = "openai"
	data.Pool = "primary"
	data.UsedPercent = &percent
	data.WindowMinutes = 300
	data.Provenance = "codex:event_msg.token_count.rate_limits"

	if err := o.runEventSink(runID).Emit(event); err != nil {
		t.Fatal(err)
	}
	rows, err := quota.ListQuotaObservations(context.Background(), "openai", "primary", "300m", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("quota rows=%+v err=%v", rows, err)
	}
	row := rows[0]
	if row.SourceRunID != runID.String() || row.EvidenceRef == "" || row.Freshness != pricing.ObservationFresh {
		t.Fatalf("observation lineage/freshness=%+v", row)
	}
	if row.Used != nil || row.Limit != nil || row.UsedPercent == nil || *row.UsedPercent != percent {
		t.Fatalf("observation attributed unreported amount=%+v", row)
	}
}
