package supervision

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type effortQuotaStore struct {
	rows  []pricing.QuotaObservation
	reads int
}

func (s *effortQuotaStore) RecordQuotaObservation(context.Context, *pricing.QuotaObservation) error {
	return nil
}

func (s *effortQuotaStore) ListQuotaObservations(context.Context, string, string, string, int) ([]pricing.QuotaObservation, error) {
	s.reads++
	return append([]pricing.QuotaObservation(nil), s.rows...), nil
}

func TestEffortBoardReadsQuotaOnceAndJoinsCanonicalSourceRun(t *testing.T) {
	s, _, controller := effortFixture(t)
	runID := uuid.New()
	if _, err := s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{
		EffortRef: "effort:quota", TargetRevision: "accepted",
		Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: runID.String(), RunId: runID.String(), Role: "worker"}},
	}, IdempotencyKey: "enroll-quota"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	controller.runs[runID] = &domain.Run{ID: runID, Status: domain.RunStatusRunning}
	used := 42.5
	quota := &effortQuotaStore{rows: []pricing.QuotaObservation{{
		Provider: "openai", Pool: "primary", Window: "5h", ObservedAt: time.Unix(100, 0).UTC(),
		Standing: pricing.QuotaStandingAvailable, UsedPercent: &used, Provenance: "codex:event_msg.token_count.rate_limits",
		SourceRunID: runID.String(), Freshness: pricing.ObservationFresh,
	}}}
	s.SetQuotaObservationStore(quota)

	board, err := s.Board(context.Background(), &pb.GetEffortBoardRequest{EffortRef: "effort:quota", PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if quota.reads != 1 {
		t.Fatalf("expected one pricing owner read, got %d", quota.reads)
	}
	if len(board.GetQuotaObservations()) != 1 || len(board.GetRows()) != 1 || len(board.GetRows()[0].GetQuotaObservations()) != 1 {
		t.Fatalf("quota observation was not retained at board and matching row scope: %+v", board)
	}
	rowQuota := board.GetRows()[0].GetQuotaObservations()[0]
	if rowQuota.GetProvider() != "openai" || rowQuota.GetPool() != "primary" || rowQuota.GetUsedPercent() != used || rowQuota.GetSourceRunId() != runID.String() {
		t.Fatalf("provider observation metadata changed during board join: %+v", rowQuota)
	}
}
