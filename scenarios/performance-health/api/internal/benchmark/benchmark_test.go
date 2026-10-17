package benchmark

import (
	"context"
	"testing"
)

type fakeRunner struct {
	res Result
	err error
}

func (f fakeRunner) Run(context.Context, string, string) (Result, error) { return f.res, f.err }

// [REQ:PH-BENCH-001] Benchmark wraps the runner and surfaces per-surface build
// timings.
func TestBenchmarkReturnsTimings(t *testing.T) {
	svc := NewService(fakeRunner{res: Result{Outcome: OutcomeMeasured, Timings: []BuildTiming{
		{Surface: "go", DurationMs: 12000, BudgetMs: 90000},
	}}})
	res, err := svc.Benchmark(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Benchmark: %v", err)
	}
	if res.Outcome != OutcomeMeasured || len(res.Timings) != 1 {
		t.Fatalf("unexpected result: %#v", res)
	}
}

// [REQ:PH-BENCH-002] MarkOverBudget flags only surfaces over a positive budget.
func TestMarkOverBudget(t *testing.T) {
	timings := MarkOverBudget([]BuildTiming{
		{Surface: "go", DurationMs: 120000, BudgetMs: 90000},
		{Surface: "ui", DurationMs: 60000, BudgetMs: 180000},
		{Surface: "none", DurationMs: 999, BudgetMs: 0},
	})
	if !timings[0].OverBudget {
		t.Fatal("go should be over budget")
	}
	if timings[1].OverBudget || timings[2].OverBudget {
		t.Fatal("ui and no-budget surfaces should not be over budget")
	}
}

func TestBenchmarkSkipsWhenRunnerAbsent(t *testing.T) {
	svc := NewService(nil)
	res, err := svc.Benchmark(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Benchmark: %v", err)
	}
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED, got %#v", res)
	}
}
