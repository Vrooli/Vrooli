package scoring

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"

	"scenario-completeness-scoring/internal/freshness"
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

func TestGetScoreUnknownScenarioIsNotFound(t *testing.T) {
	stub := &stubScorer{err: internalscoring.ErrUnknownScenario}
	h := NewConnectHandler(Deps{Scorer: stub})

	_, err := h.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}
