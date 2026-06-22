package validation

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	phassessment "performance-health/internal/assessment"
	"performance-health/internal/autofix"
	"performance-health/internal/benchmark"
	"performance-health/internal/lighthouse"
	"performance-health/internal/readiness"
	"performance-health/internal/startup"
	"performance-health/internal/trend"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeBenchmark struct {
	res    benchmark.Result
	err    error
	called bool
}

func (f *fakeBenchmark) Benchmark(_ context.Context, scenario, _ string) (benchmark.Result, error) {
	f.called = true
	f.res.Scenario = scenario
	return f.res, f.err
}

type fakeStartup struct {
	m      startup.Measurement
	err    error
	called bool
}

func (f *fakeStartup) Benchmark(_ context.Context, _ string, _ time.Duration) (startup.Measurement, error) {
	f.called = true
	return f.m, f.err
}

type fakeLighthouse struct {
	res    lighthouse.Result
	err    error
	called bool
}

func (f *fakeLighthouse) Score(_ context.Context, _, _ string) (lighthouse.Result, error) {
	f.called = true
	return f.res, f.err
}

type fakeSampleWriter struct {
	samples []trend.Sample
	err     error
}

func (f *fakeSampleWriter) Insert(_ context.Context, sample trend.Sample) error {
	f.samples = append(f.samples, sample)
	return f.err
}

// TestExecutionOrchestratorPersistsThenGates is the core contract: producers run
// and ONE combined sample is persisted so the budgets domain — the sole emitter
// of PERF_BUDGET_BREACH_* — can gate the freshly persisted values. The
// orchestrator itself contributes only the Lighthouse and broken-build findings;
// build-budget breaches are NOT emitted here (Contract Decision 8.1).
func TestExecutionOrchestratorPersistsThenGates(t *testing.T) {
	bench := &fakeBenchmark{res: benchmark.Result{
		Outcome: benchmark.OutcomeMeasured,
		Timings: []benchmark.BuildTiming{
			{Surface: "go", DurationMs: 120000, BudgetMs: 90000, OverBudget: true},
			{Surface: "ui", DurationMs: 4000, BudgetMs: 180000, OverBudget: false},
		},
		BundleBytes: 4242,
	}}
	lh := &fakeLighthouse{res: lighthouse.Result{
		Outcome: lighthouse.OutcomeScored,
		Pages: []lighthouse.PageScore{{
			URL:             "http://localhost:3000/",
			ErrorViolations: []string{"performance 0.50 < error 0.75"},
			Violations:      []string{"performance 0.50 < error 0.75"},
		}},
	}}
	writer := &fakeSampleWriter{}
	o := NewExecutionOrchestrator(ExecutionDeps{Benchmark: bench, Lighthouse: lh, Trend: writer})

	findings, _ := o.Run(context.Background(), "demo", "")

	if !bench.called || !lh.called {
		t.Fatalf("expected benchmark and lighthouse to run; bench=%v lh=%v", bench.called, lh.called)
	}
	if len(writer.samples) != 1 {
		t.Fatalf("expected exactly one persisted sample, got %d", len(writer.samples))
	}
	got := writer.samples[0]
	if got.GoBuildMs != 120000 || got.UIBuildMs != 4000 || got.BundleBytes != 4242 {
		t.Fatalf("combined sample wrong: %+v", got)
	}
	// The orchestrator no longer emits build-budget breaches — the budgets domain
	// is the sole emitter, gating the persisted sample above.
	if hasCode(findings, "PERF_BUDGET_BREACH_GO_BUILD") || hasCode(findings, "PERF_BUDGET_BREACH_UI_BUILD") {
		t.Errorf("execution must not emit budget-breach findings (budgets domain owns those): %v", findingCodeList(findings))
	}
	if !hasCode(findings, "PERF_LIGHTHOUSE_BELOW_ERROR_THRESHOLD") {
		t.Errorf("expected lighthouse error finding, got %v", findingCodeList(findings))
	}
}

// TestExecutionOrchestratorSkipNotFailOnProducerError: a producer error degrades
// to skip-not-fail (no finding, no panic) and nothing is persisted.
func TestExecutionOrchestratorSkipNotFailOnProducerError(t *testing.T) {
	bench := &fakeBenchmark{err: errors.New("go toolchain missing")}
	lh := &fakeLighthouse{err: errors.New("no chrome")}
	writer := &fakeSampleWriter{}
	o := NewExecutionOrchestrator(ExecutionDeps{Benchmark: bench, Lighthouse: lh, Trend: writer})

	findings, _ := o.Run(context.Background(), "demo", "")
	if len(findings) != 0 {
		t.Errorf("producer errors must not produce findings, got %v", findingCodeList(findings))
	}
	if len(writer.samples) != 0 {
		t.Errorf("nothing measured => nothing persisted, got %d samples", len(writer.samples))
	}
}

// TestExecutionOrchestratorBrokenBuild maps a failed build to an ERROR finding.
func TestExecutionOrchestratorBrokenBuild(t *testing.T) {
	bench := &fakeBenchmark{res: benchmark.Result{
		Outcome: benchmark.OutcomeFailed,
		Timings: []benchmark.BuildTiming{{Surface: "go", DurationMs: 1000, BudgetMs: 90000}},
		Reason:  "go build failed: syntax error",
	}}
	o := NewExecutionOrchestrator(ExecutionDeps{Benchmark: bench, Trend: &fakeSampleWriter{}})
	findings, _ := o.Run(context.Background(), "demo", "")
	if !hasCode(findings, "PERF_BUILD_FAILED") {
		t.Errorf("expected PERF_BUILD_FAILED, got %v", findingCodeList(findings))
	}
}

// TestExecutionOrchestratorLighthouseSkippedNoFinding: a clean Lighthouse skip
// (no UI) yields no finding (skip-not-fail).
func TestExecutionOrchestratorLighthouseSkippedNoFinding(t *testing.T) {
	lh := &fakeLighthouse{res: lighthouse.Result{Outcome: lighthouse.OutcomeSkipped, Reason: "no UI"}}
	o := NewExecutionOrchestrator(ExecutionDeps{Lighthouse: lh, Trend: &fakeSampleWriter{}})
	findings, _ := o.Run(context.Background(), "demo", "")
	if len(findings) != 0 {
		t.Errorf("skipped lighthouse must not fail, got %v", findingCodeList(findings))
	}
}

// TestExecutionOrchestratorStartupFolded: when a startup measurer is wired, its
// time-to-healthy lands in the combined sample.
func TestExecutionOrchestratorStartupFolded(t *testing.T) {
	su := &fakeStartup{m: startup.Measurement{Scenario: "demo", TimeToHealthyMs: 2500, Healthy: true}}
	writer := &fakeSampleWriter{}
	o := NewExecutionOrchestrator(ExecutionDeps{Startup: su, Trend: writer})
	o.Run(context.Background(), "demo", "")
	if !su.called {
		t.Fatal("startup measurer should be called when wired")
	}
	if len(writer.samples) != 1 || writer.samples[0].StartupMs != 2500 {
		t.Fatalf("startup_ms not folded into sample: %+v", writer.samples)
	}
}

type recordingExecution struct {
	calls    int
	findings []phassessment.Finding
	measured bool
}

func (r *recordingExecution) Run(_ context.Context, _, _ string) ([]phassessment.Finding, bool) {
	r.calls++
	return r.findings, r.measured
}

// TestIncludeExecutionDivergence proves the flag is honored end-to-end:
// include_execution=false never runs the producers; true runs them and folds
// their ERROR findings into the status (FAILED).
func TestIncludeExecutionDivergence(t *testing.T) {
	spec, err := assessment.ParseSpec([]byte(budgetGateSpec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	newHandler := func(exec ExecutionRunner) *SharedHandler {
		return NewSharedHandler(NewHandlerWithDeps(Deps{
			Readiness:    readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
			Autofix:      autofix.NewService(),
			MaturitySpec: spec,
			Budgets:      fakeBudgetChecker{passed: true},
			Execution:    exec,
		}))
	}

	// include_execution=false: producers must NOT run.
	off := &recordingExecution{}
	if _, err := newHandler(off).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", IncludeExecution: false})); err != nil {
		t.Fatalf("ValidateScenario(false): %v", err)
	}
	if off.calls != 0 {
		t.Fatalf("include_execution=false must not run producers, got %d calls", off.calls)
	}

	// include_execution=true: producers run and their ERROR finding fails the gate.
	on := &recordingExecution{findings: []phassessment.Finding{{
		Code: "PERF_BUDGET_BREACH_GO_BUILD", Severity: "error", Title: "go build over threshold", Message: "120000ms > 90000ms",
	}}}
	resp, err := newHandler(on).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario(true): %v", err)
	}
	if on.calls != 1 {
		t.Fatalf("include_execution=true must run producers once, got %d calls", on.calls)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("an execution ERROR finding must fail validation, got %s", resp.Msg.GetStatus())
	}
}

// TestSkippedWhenNothingMeasured proves the skip-honesty contract: when
// include_execution=true but every producer cleanly skipped (measured=false)
// and nothing failed, the gate reports SKIPPED — never a false PASS.
func TestSkippedWhenNothingMeasured(t *testing.T) {
	spec, err := assessment.ParseSpec([]byte(budgetGateSpec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	handler := NewSharedHandler(NewHandlerWithDeps(Deps{
		Readiness:    readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
		Autofix:      autofix.NewService(),
		MaturitySpec: spec,
		Budgets:      fakeBudgetChecker{passed: true},
		Execution:    &recordingExecution{measured: false},
	}))
	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED {
		t.Fatalf("measuring nothing must report SKIPPED, got %s", resp.Msg.GetStatus())
	}
}

func hasCode(findings []phassessment.Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func findingCodeList(findings []phassessment.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Code)
	}
	return out
}
