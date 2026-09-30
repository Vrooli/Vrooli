package sweep

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"performance-health/internal/analysis"
	"performance-health/internal/budgets"
	"performance-health/internal/capture"
	"performance-health/internal/perfsample"
	"performance-health/internal/readiness"

	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
)

type fakeAuditor struct {
	byFlow map[string]capture.Result
}

func (f fakeAuditor) Orchestrate(_ context.Context, _, workflow string, _ readiness.Tier) (capture.Result, error) {
	return f.byFlow[workflow], nil
}

type fakeAnalyzer struct {
	res analysis.Result
}

func (f fakeAnalyzer) Analyze(context.Context, string, string) (analysis.Result, error) {
	return f.res, nil
}

type recordingTrend struct {
	samples []perfsample.Sample
}

func (r *recordingTrend) Insert(_ context.Context, s perfsample.Sample) error {
	r.samples = append(r.samples, s)
	return nil
}

type fakeGate struct {
	budget     budgets.Budget
	declared   bool
	flowResult map[string]struct {
		passed     bool
		violations []budgets.Violation
	}
}

func (f fakeGate) Get(context.Context, string) (budgets.Budget, bool, error) {
	return f.budget, f.declared, nil
}

func (f fakeGate) CheckFlow(_ context.Context, _, flow string) (bool, []budgets.Violation, error) {
	r := f.flowResult[flow]
	return r.passed, r.violations, nil
}

// [REQ:PH-SWEEP-001] A budgeted flow that captures gets a FLOW-TAGGED sample
// persisted and its per-flow budget verdict reported.
func TestRunSweepPersistsFlowTaggedSampleAndReportsVerdict(t *testing.T) {
	trend := &recordingTrend{}
	h := NewHandler(
		fakeAuditor{byFlow: map[string]capture.Result{
			"scroll-list": {Outcome: capture.OutcomeCaptured, TraceArtifact: "/runs/scroll.json"},
		}},
		fakeAnalyzer{res: analysis.Result{LCPMs: 3200, Components: []analysis.ComponentTiming{{Component: "List", AvgMs: 12, MaxMs: 40}}}},
		trend,
		fakeGate{
			declared: true,
			budget:   budgets.Budget{Scenario: "demo", Flows: map[string]budgets.FlowBudget{"scroll-list": {LCPMaxMs: 2000}}},
			flowResult: map[string]struct {
				passed     bool
				violations []budgets.Violation
			}{"scroll-list": {passed: false, violations: []budgets.Violation{{Axis: "lcp", Measured: 3200, Budget: 2000, Unit: "ms"}}}},
		},
		nil,
		nil,
	)

	resp, err := h.RunSweep(context.Background(), connect.NewRequest(&sweepv1.RunSweepRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunSweep: %v", err)
	}
	if len(resp.Msg.GetResults()) != 1 {
		t.Fatalf("expected 1 flow result, got %d", len(resp.Msg.GetResults()))
	}
	r := resp.Msg.GetResults()[0]
	if r.GetFlow() != "scroll-list" || r.GetOutcome() != "captured" {
		t.Fatalf("unexpected result: %#v", r)
	}
	if r.GetWithinBudget() {
		t.Fatal("expected over-budget verdict")
	}
	if len(r.GetViolations()) != 1 {
		t.Fatalf("expected one violation line, got %v", r.GetViolations())
	}
	// The persisted sample MUST be tagged with the flow slug.
	if len(trend.samples) != 1 {
		t.Fatalf("expected one persisted sample, got %d", len(trend.samples))
	}
	s := trend.samples[0]
	if s.Flow != "scroll-list" || s.LCPMs != 3200 || s.SlowestComponent != "List" {
		t.Fatalf("flow sample not tagged/populated: %#v", s)
	}
}

// [REQ:PH-SWEEP-001] A scenario with no per-flow budgets sweeps nothing.
func TestRunSweepNoBudgetedFlows(t *testing.T) {
	trend := &recordingTrend{}
	h := NewHandler(fakeAuditor{}, fakeAnalyzer{}, trend, fakeGate{declared: true, budget: budgets.Budget{Scenario: "demo"}}, nil, nil)
	resp, err := h.RunSweep(context.Background(), connect.NewRequest(&sweepv1.RunSweepRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunSweep: %v", err)
	}
	if len(resp.Msg.GetResults()) != 0 || len(trend.samples) != 0 {
		t.Fatalf("expected no sweep work, got results=%d samples=%d", len(resp.Msg.GetResults()), len(trend.samples))
	}
}

// [REQ:PH-SWEEP-001] An unavailable capture persists no sample and reports the
// outcome verbatim (never read as a pass).
func TestRunSweepUnavailableCapturePersistsNothing(t *testing.T) {
	trend := &recordingTrend{}
	h := NewHandler(
		fakeAuditor{byFlow: map[string]capture.Result{
			"scroll-list": {Outcome: capture.OutcomeUnavailable, Reason: "no browser"},
		}},
		fakeAnalyzer{},
		trend,
		fakeGate{declared: true, budget: budgets.Budget{Scenario: "demo", Flows: map[string]budgets.FlowBudget{"scroll-list": {LCPMaxMs: 2000}}}},
		nil,
		nil,
	)
	resp, err := h.RunSweep(context.Background(), connect.NewRequest(&sweepv1.RunSweepRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunSweep: %v", err)
	}
	r := resp.Msg.GetResults()[0]
	if r.GetOutcome() != "unavailable" {
		t.Fatalf("expected unavailable, got %q", r.GetOutcome())
	}
	if len(trend.samples) != 0 {
		t.Fatal("an unavailable capture must persist no sample")
	}
}
