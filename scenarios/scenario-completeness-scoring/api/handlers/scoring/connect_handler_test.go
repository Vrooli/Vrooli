package scoring

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"

	"scenario-completeness-scoring/internal/freshness"
	"scenario-completeness-scoring/internal/importance"
	internalscoring "scenario-completeness-scoring/internal/scoring"
	"scenario-completeness-scoring/internal/signals"
)

type stubScorer struct {
	result internalscoring.Result
	err    error
	got    string
}

func (s *stubScorer) GetScore(scenario string) (internalscoring.Result, error) {
	s.got = scenario
	return s.result, s.err
}

type stubSnapshots struct {
	previous internalscoring.Snapshot
	has      bool
	err      error
	page     internalscoring.ListResult
	upserts  []internalscoring.Snapshot

	gotScenario string
	gotDigest   string
	gotList     internalscoring.ListQuery
}

func (s *stubSnapshots) LatestDifferingDigest(ctx context.Context, scenario, digest string) (internalscoring.Snapshot, bool, error) {
	s.gotScenario = scenario
	s.gotDigest = digest
	return s.previous, s.has, s.err
}

func (s *stubSnapshots) SeriesFor(ctx context.Context, q internalscoring.TrendQuery) ([]internalscoring.Snapshot, error) {
	return nil, nil
}

func (s *stubSnapshots) ListPage(ctx context.Context, q internalscoring.ListQuery) (internalscoring.ListResult, error) {
	s.gotList = q
	return s.page, nil
}

func (s *stubSnapshots) UpsertSnapshot(ctx context.Context, snap internalscoring.Snapshot) (bool, error) {
	s.upserts = append(s.upserts, snap)
	return true, nil
}

func TestGetScoreConvertsDomainResult(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	stub := &stubScorer{result: internalscoring.Result{
		Scenario: "fixture",
		Category: "utility",
		Maturity: internalscoring.Maturity{
			WorkingRung:      "R1 Safe & standards-clean",
			SatisfiedThrough: "R0 Runnable & green",
			BuildPassing:     true,
			Dimensions: []internalscoring.DimensionCount{
				{Dimension: "standards", ErrorPlus: 2, Total: 5, Approximate: true},
			},
		},
		Composite: internalscoring.Composite{
			Score:               72,
			Classification:      "mostly_complete",
			ClassificationLabel: "Mostly complete, needs refinement and validation",
			Groups: []internalscoring.Group{
				{ID: "quality", Label: "Quality", Score: 40, Max: 50, Metrics: []internalscoring.Metric{
					{ID: "requirement_pass_rate", Label: "Requirements", Observed: "10/12", Points: 16, MaxPoints: 20},
				}},
			},
		},
		Freshness: freshness.Result{
			Digest:           "td:abc",
			SuggestedCommand: "test-genie execute fixture --preset quick",
			Phases: []freshness.PhaseStatus{
				{Phase: "unit", Verdict: "stale", LastRunID: "run-1", LastRunAt: now, LastDigest: "td:old", LastStatus: "passed"},
			},
		},
		Importance: &importance.Summary{
			Score:          0.82,
			SystemRequired: true,
			Components: importance.Components{
				Centrality:    0.7,
				CoreProximity: 0.5,
				Recency:       0.4,
			},
			Signals: importance.Signals{
				DirectReverseDependencyCount:     2,
				TransitiveReverseDependencyCount: 5,
				RequiredReverseDependencyCount:   3,
				RequiredEdgeWeightedScore:        8,
				DistanceToCoreSeed:               1,
				NearestCoreSeed:                  "test-genie",
				RecentActivityCount:              2,
			},
			Degraded: []string{"recency:not_configured"},
		},
		Recommends: []internalscoring.Recommendation{
			{Priority: "high", Description: "Fix failing test phases", Impact: 5},
		},
		ActionPlan: []internalscoring.ActionPhase{
			{Title: "Restore quality gates", Actions: []string{"Fix failing test phases"}, EstimatedPoints: 5},
		},
		Degradations: []signals.Degradation{
			{Collector: "ui", State: "failed", Reason: "no ui sources"},
		},
		CalculatedAt: now,
	}}

	h := NewConnectHandler(Deps{Scorer: stub})
	resp, err := h.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: "fixture"}))
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if stub.got != "fixture" {
		t.Fatalf("scorer called with %q", stub.got)
	}

	msg := resp.Msg
	if msg.GetScenario() != "fixture" || msg.GetCategory() != "utility" {
		t.Fatalf("identity fields wrong: %v", msg)
	}
	if msg.GetMaturity().GetWorkingRung() != "R1 Safe & standards-clean" || !msg.GetMaturity().GetBuildPassing() {
		t.Fatalf("maturity wrong: %v", msg.GetMaturity())
	}
	dims := msg.GetMaturity().GetDimensions()
	if len(dims) != 1 || dims[0].GetErrorPlus() != 2 || !dims[0].GetApproximate() {
		t.Fatalf("dimensions wrong: %v", dims)
	}
	if msg.GetComposite().GetScore() != 72 || len(msg.GetComposite().GetGroups()) != 1 {
		t.Fatalf("composite wrong: %v", msg.GetComposite())
	}
	if msg.GetFreshness().GetCurrentDigest() != "td:abc" || len(msg.GetFreshness().GetPhases()) != 1 {
		t.Fatalf("freshness wrong: %v", msg.GetFreshness())
	}
	if msg.GetImportance().GetScore() != 0.82 || !msg.GetImportance().GetSystemRequired() {
		t.Fatalf("importance wrong: %v", msg.GetImportance())
	}
	if msg.GetImportance().GetSignals().GetTransitiveReverseDependencyCount() != 5 {
		t.Fatalf("importance signals wrong: %v", msg.GetImportance().GetSignals())
	}
	pf := msg.GetFreshness().GetPhases()[0]
	if pf.GetVerdict() != "stale" || pf.GetLastDigest() != "td:old" || pf.GetLastRunAt().AsTime() != now {
		t.Fatalf("phase freshness wrong: %v", pf)
	}
	if len(msg.GetRecommendations()) != 1 || msg.GetRecommendations()[0].GetImpactPoints() != 5 {
		t.Fatalf("recommendations wrong: %v", msg.GetRecommendations())
	}
	if len(msg.GetActionPlan()) != 1 || len(msg.GetDegradations()) != 1 {
		t.Fatalf("plan/degradations wrong: %v / %v", msg.GetActionPlan(), msg.GetDegradations())
	}
	if msg.GetCalculatedAt().AsTime() != now {
		t.Fatalf("calculated_at wrong: %v", msg.GetCalculatedAt())
	}
}

func TestGetScoreAttachesTrendFromLatestDifferingDigest(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	previousAt := now.Add(-48 * time.Hour)
	stub := &stubScorer{result: internalscoring.Result{
		Scenario: "fixture",
		Composite: internalscoring.Composite{
			Score:          72,
			Classification: "mostly_complete",
		},
		Freshness:    freshness.Result{Digest: "td:new"},
		CalculatedAt: now,
	}}
	snapshots := &stubSnapshots{
		previous: internalscoring.Snapshot{
			Scenario:  "fixture",
			Digest:    "td:old",
			Composite: 65,
			CreatedAt: previousAt,
		},
		has: true,
	}

	h := NewConnectHandler(Deps{Scorer: stub, Snapshots: snapshots})
	resp, err := h.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: "fixture"}))
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if snapshots.gotScenario != "fixture" || snapshots.gotDigest != "td:new" {
		t.Fatalf("trend lookup = (%q, %q), want fixture/td:new", snapshots.gotScenario, snapshots.gotDigest)
	}
	trend := resp.Msg.GetTrend()
	if trend == nil {
		t.Fatalf("trend omitted")
	}
	if trend.GetPreviousScore() != 65 || trend.GetDelta() != 7 {
		t.Fatalf("trend score math wrong: %v", trend)
	}
	if trend.GetPreviousCalculatedAt().AsTime() != previousAt {
		t.Fatalf("trend timestamp wrong: %v", trend.GetPreviousCalculatedAt())
	}
}

func TestGetScoreOmitsTrendWhenNoDifferingSnapshot(t *testing.T) {
	stub := &stubScorer{result: internalscoring.Result{
		Scenario:     "fixture",
		Composite:    internalscoring.Composite{Score: 72},
		Freshness:    freshness.Result{Digest: "td:new"},
		CalculatedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}}
	h := NewConnectHandler(Deps{Scorer: stub, Snapshots: &stubSnapshots{}})

	resp, err := h.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: "fixture"}))
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if resp.Msg.GetTrend() != nil {
		t.Fatalf("unexpected trend: %v", resp.Msg.GetTrend())
	}
}

func TestGetScoreUnknownScenarioIsNotFound(t *testing.T) {
	stub := &stubScorer{err: internalscoring.ErrUnknownScenario}
	h := NewConnectHandler(Deps{Scorer: stub})

	_, err := h.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestListScoresRecomputeIsBoundedToReturnedPage(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	snapshots := &stubSnapshots{
		page: internalscoring.ListResult{
			Snapshots: []internalscoring.Snapshot{
				{
					Scenario:       "alpha",
					Category:       "utility",
					Digest:         "td:old",
					Composite:      40,
					Classification: "foundation_laid",
					WorkingRung:    "R1",
					CreatedAt:      now.Add(-time.Hour),
				},
			},
			HasNext:    true,
			NextOffset: 1,
		},
	}
	stub := &stubScorer{result: internalscoring.Result{
		Scenario: "alpha",
		Category: "utility",
		Composite: internalscoring.Composite{
			Score:          90,
			Classification: "nearly_ready",
		},
		Maturity:     internalscoring.Maturity{WorkingRung: "R3"},
		Freshness:    freshness.Result{Digest: "td:new"},
		CalculatedAt: now,
	}}
	h := NewConnectHandler(Deps{Scorer: stub, Snapshots: snapshots})

	resp, err := h.ListScores(context.Background(), connect.NewRequest(&scoringv1.ListScoresRequest{
		PageSize:  500,
		Recompute: true,
	}))
	if err != nil {
		t.Fatalf("ListScores: %v", err)
	}
	if snapshots.gotList.Limit != maxRecomputePageSize {
		t.Fatalf("ListPage limit = %d, want recompute cap %d", snapshots.gotList.Limit, maxRecomputePageSize)
	}
	if stub.got != "alpha" {
		t.Fatalf("scorer called with %q, want alpha", stub.got)
	}
	if len(snapshots.upserts) != 1 || snapshots.upserts[0].Digest != "td:new" || snapshots.upserts[0].Source != "recompute" {
		t.Fatalf("upserts = %+v, want recomputed snapshot persisted", snapshots.upserts)
	}
	rows := resp.Msg.GetScores()
	if len(rows) != 1 || rows[0].GetScore() != 90 || rows[0].GetDigest() != "td:new" {
		t.Fatalf("rows = %+v, want recomputed row", rows)
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Fatalf("next page token omitted")
	}
}
