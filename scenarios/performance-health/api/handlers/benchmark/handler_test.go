package benchmark

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalbench "performance-health/internal/benchmark"
	"performance-health/internal/perfsample"

	benchmarkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark"
)

// fakeRunner drives the benchmark Service's lowest seam.
type fakeRunner struct {
	res    internalbench.Result
	err    error
	called bool
}

func (f *fakeRunner) Run(_ context.Context, scenario, _ string) (internalbench.Result, error) {
	f.called = true
	f.res.Scenario = scenario
	return f.res, f.err
}

// fakeSampleWriter records the trend sample(s) the handler persists.
type fakeSampleWriter struct {
	samples []perfsample.Sample
	err     error
}

func (f *fakeSampleWriter) Insert(_ context.Context, s perfsample.Sample) error {
	f.samples = append(f.samples, s)
	return f.err
}

// TestRunBenchmarkMapsResultToProto builds the REAL benchmark service over a
// fake runner, calls the RunBenchmark RPC, and asserts the proto response is
// mapped correctly — including the newly-added bundle_bytes field and the
// per-surface timings. It also proves a measured run persists one trend sample.
func TestRunBenchmarkMapsResultToProto(t *testing.T) {
	runner := &fakeRunner{res: internalbench.Result{
		Outcome:     internalbench.OutcomeMeasured,
		BundleBytes: 524288,
		Timings: []internalbench.BuildTiming{
			{Surface: "go", DurationMs: 12000, BudgetMs: 90000, OverBudget: false},
			{Surface: "ui", DurationMs: 200000, BudgetMs: 180000, OverBudget: true},
		},
	}}
	writer := &fakeSampleWriter{}
	h := NewHandler(internalbench.NewService(runner), writer, nil)

	resp, err := h.RunBenchmark(context.Background(), connect.NewRequest(&benchmarkv1.RunBenchmarkRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunBenchmark: %v", err)
	}
	if !runner.called {
		t.Fatal("runner seam was not exercised")
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q, want demo", msg.GetScenario())
	}
	if msg.GetOutcome() != benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_MEASURED {
		t.Errorf("outcome = %v, want MEASURED", msg.GetOutcome())
	}
	// bundle_bytes is a newly-added proto field — assert it is populated from the
	// Result's BundleBytes.
	if msg.GetBundleBytes() != 524288 {
		t.Errorf("bundle_bytes = %d, want 524288", msg.GetBundleBytes())
	}
	if len(msg.GetTimings()) != 2 {
		t.Fatalf("timings len = %d, want 2", len(msg.GetTimings()))
	}
	ui := msg.GetTimings()[1]
	if ui.GetSurface() != "ui" || ui.GetDurationMs() != 200000 || ui.GetBudgetMs() != 180000 || !ui.GetOverBudget() {
		t.Errorf("ui timing mapped wrong: %+v", ui)
	}
	// A measured run persists exactly one combined trend sample.
	if len(writer.samples) != 1 {
		t.Fatalf("expected 1 persisted sample, got %d", len(writer.samples))
	}
	got := writer.samples[0]
	if got.Scenario != "demo" || got.GoBuildMs != 12000 || got.UIBuildMs != 200000 || got.BundleBytes != 524288 {
		t.Errorf("persisted sample wrong: %+v", got)
	}
}

// TestRunBenchmarkSkippedDoesNotPersist proves a non-measured outcome maps to
// the SKIPPED enum and does NOT write a trend sample.
func TestRunBenchmarkSkippedDoesNotPersist(t *testing.T) {
	runner := &fakeRunner{res: internalbench.Result{Outcome: internalbench.OutcomeSkipped, Reason: "no buildable surfaces"}}
	writer := &fakeSampleWriter{}
	h := NewHandler(internalbench.NewService(runner), writer, nil)

	resp, err := h.RunBenchmark(context.Background(), connect.NewRequest(&benchmarkv1.RunBenchmarkRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunBenchmark: %v", err)
	}
	if resp.Msg.GetOutcome() != benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_SKIPPED {
		t.Errorf("outcome = %v, want SKIPPED", resp.Msg.GetOutcome())
	}
	if resp.Msg.GetReason() != "no buildable surfaces" {
		t.Errorf("reason = %q", resp.Msg.GetReason())
	}
	if len(writer.samples) != 0 {
		t.Errorf("skipped run must not persist, got %d samples", len(writer.samples))
	}
}

// TestRunBenchmarkRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestRunBenchmarkRequiresScenario(t *testing.T) {
	h := NewHandler(internalbench.NewService(&fakeRunner{}), nil, nil)
	_, err := h.RunBenchmark(context.Background(), connect.NewRequest(&benchmarkv1.RunBenchmarkRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
