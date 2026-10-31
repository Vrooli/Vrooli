package trials

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"
)

// fakeGen / fakeRunner / fakeRepo are in-memory seams.
type fakeGen struct {
	tasks []TrialTask
	err   error
}

func (f *fakeGen) Generate(_ context.Context, suite string) ([]TrialTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	if suite == "" {
		return f.tasks, nil
	}
	var out []TrialTask
	for _, t := range f.tasks {
		if t.Suite == suite {
			out = append(out, t)
		}
	}
	return out, nil
}

type fakeRunner struct {
	result RunResult
	calls  int
}

func (f *fakeRunner) RunTask(_ context.Context, _ TrialTask, _ Fixture) RunResult {
	f.calls++
	return f.result
}

// fakeFixtures returns a fixed fixture for every task (or a per-task map). ok is
// false for tasks listed in `missing`.
type fakeFixtures struct {
	fixture Fixture
	byTask  map[string]Fixture
	missing map[string]bool
	err     error
}

func (f *fakeFixtures) Resolve(_ context.Context, task TrialTask) (Fixture, bool, error) {
	if f.err != nil {
		return Fixture{}, false, f.err
	}
	if f.missing[task.ID] {
		return Fixture{}, false, nil
	}
	if fx, ok := f.byTask[task.ID]; ok {
		return fx, true, nil
	}
	fx := f.fixture
	fx.Negative = task.Negative || task.Suite == SuiteNegative
	return fx, true, nil
}

// fakeEvaluator returns a configured verdict, or echoes the evidence verdict
// when passthrough is set (so negative/abstention logic can be exercised via a
// real evaluator elsewhere).
type fakeEvaluator struct {
	verdict Verdict
	calls   int
	lastRes RunResult
}

func (e *fakeEvaluator) Judge(_ context.Context, _ TrialTask, _ Fixture, res RunResult) Verdict {
	e.calls++
	e.lastRes = res
	return e.verdict
}

type fakeRepo struct {
	runs      []TrialRun
	gated     int
	recordErr error
}

func (r *fakeRepo) LatestRun(_ context.Context, taskID, model, fixtureRev string) (TrialRun, bool, error) {
	var latest TrialRun
	found := false
	for _, run := range r.runs {
		if run.TaskID == taskID && run.Model == model && run.FixtureRev == fixtureRev {
			if !found || run.At.After(latest.At) {
				latest = run
				found = true
			}
		}
	}
	return latest, found, nil
}

func (r *fakeRepo) RecordRun(_ context.Context, run TrialRun) error {
	if r.recordErr != nil {
		return r.recordErr
	}
	r.runs = append(r.runs, run)
	return nil
}

func (r *fakeRepo) GetRun(_ context.Context, id string) (TrialRun, bool, error) {
	for _, run := range r.runs {
		if run.ID == id {
			return run, true, nil
		}
	}
	return TrialRun{}, false, nil
}

func (r *fakeRepo) Runs(_ context.Context, filter RunFilter, limit int, desc bool) ([]TrialRun, error) {
	var out []TrialRun
	for _, run := range r.runs {
		if filter.Suite != "" && run.Suite != filter.Suite {
			continue
		}
		if filter.TaskID != "" && run.TaskID != filter.TaskID {
			continue
		}
		out = append(out, run)
	}
	if desc {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *fakeRepo) GatedGuideTaskCount(_ context.Context) (int, error) { return r.gated, nil }

func tasksFixture() []TrialTask {
	return []TrialTask{
		{ID: "trial/g1", Suite: SuiteAddFeature, GuideTaskID: "g1", Description: "add a domain"},
		{ID: "trial/g2", Suite: SuiteComprehend, GuideTaskID: "g2", Description: "explain a flow"},
		{ID: "trial/negative/add-feature", Suite: SuiteNegative, GuideTaskID: "negative/add-feature", Negative: true},
	}
}

// newSvc wires a service with passthrough fixtures (a fixture for every task,
// rev "rev1") and an evaluator that always passes — the default for tests that
// exercise orchestration rather than evaluation/idempotency.
func newSvc(gen TaskGenerator, runner Runner, repo Repository) Service {
	return newSvcFull(gen, runner, repo,
		&fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1", TargetDir: "/t"}},
		&fakeEvaluator{verdict: VerdictPass},
		scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)))
}

func newSvcFull(gen TaskGenerator, runner Runner, repo Repository, fx FixtureResolver, ev Evaluator, clk schedule.Clock) Service {
	return NewService(Deps{Tasks: gen, Fixtures: fx, Runner: runner, Evaluator: ev, Repo: repo, Clock: clk})
}

func TestListTrialTasksFilters(t *testing.T) {
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, &fakeRunner{}, &fakeRepo{})
	all, err := svc.ListTrialTasks(context.Background(), "")
	if err != nil || len(all) != 3 {
		t.Fatalf("list all: len=%d err=%v", len(all), err)
	}
	one, err := svc.ListTrialTasks(context.Background(), SuiteComprehend)
	if err != nil || len(one) != 1 || one[0].ID != "trial/g2" {
		t.Fatalf("suite filter failed: %+v err=%v", one, err)
	}
}

func TestRunTrialsDispatchesAndRecords(t *testing.T) {
	gen := &fakeGen{tasks: tasksFixture()}
	runner := &fakeRunner{result: RunResult{Verdict: VerdictPass, Tokens: 1000, DurationMs: 5000, SandboxDiffRef: "sbx-1"}}
	repo := &fakeRepo{}
	svc := newSvc(gen, runner, repo)

	// Run a single task.
	runs, err := svc.RunTrials(context.Background(), "", "trial/g1")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(runs) != 1 || runs[0].Verdict != VerdictPass || runs[0].GuideTaskID != "g1" {
		t.Fatalf("run result wrong: %+v", runs)
	}
	if runs[0].ID == "" || runs[0].At.IsZero() {
		t.Fatalf("run not stamped: %+v", runs[0])
	}
	if len(repo.runs) != 1 {
		t.Fatalf("expected 1 recorded run, got %d", len(repo.runs))
	}

	// Run a whole suite.
	runner.calls = 0
	runs, err = svc.RunTrials(context.Background(), SuiteAddFeature, "")
	if err != nil || len(runs) != 1 || runner.calls != 0 {
		t.Fatalf("suite run should reuse the just-recorded matching task: runs=%d calls=%d err=%v", len(runs), runner.calls, err)
	}
}

func TestRunTrialsUnknownTaskErrors(t *testing.T) {
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, &fakeRunner{}, &fakeRepo{})
	if _, err := svc.RunTrials(context.Background(), "", "trial/nope"); err == nil {
		t.Fatalf("expected error for unknown task")
	}
}

func TestRunTrialsAllTasksWhenUnscoped(t *testing.T) {
	runner := &fakeRunner{result: RunResult{Verdict: VerdictPass}}
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, runner, &fakeRepo{})
	runs, err := svc.RunTrials(context.Background(), "", "")
	if err != nil || len(runs) != 3 {
		t.Fatalf("expected 3 runs (whole suite), got %d err=%v", len(runs), err)
	}
}

func TestGetTrialRunNotFound(t *testing.T) {
	svc := newSvc(&fakeGen{}, &fakeRunner{}, &fakeRepo{})
	if _, err := svc.GetTrialRun(context.Background(), "nope"); err == nil {
		t.Fatalf("expected not-found error")
	}
}

func TestGateCoverage(t *testing.T) {
	// 2 non-negative guide tasks (g1, g2); 1 gated.
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, &fakeRunner{}, &fakeRepo{gated: 1})
	gc, err := svc.GetGateCoverage(context.Background())
	if err != nil {
		t.Fatalf("gate coverage: %v", err)
	}
	if gc.GuideTasksTotal != 2 || gc.GuideTasksWithGate != 1 {
		t.Fatalf("gate coverage counts: %+v", gc)
	}
	if gc.Ratio != 0.5 {
		t.Fatalf("ratio = %v want 0.5", gc.Ratio)
	}
}

func TestHistoryAggregation(t *testing.T) {
	repo := &fakeRepo{runs: []TrialRun{
		{ID: "1", Suite: SuiteAddFeature, Verdict: VerdictPass, Tokens: 100, DurationMs: 1000, At: time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)},
		{ID: "2", Suite: SuiteAddFeature, Verdict: VerdictFail, Tokens: 300, DurationMs: 3000, At: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)},
		{ID: "3", Suite: SuiteAddFeature, Verdict: VerdictPass, Tokens: 200, DurationMs: 2000, At: time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)},
	}}
	svc := newSvc(&fakeGen{}, &fakeRunner{}, repo)
	hist, err := svc.GetTrialHistory(context.Background(), "", "")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist.Points) != 2 {
		t.Fatalf("want 2 day points, got %d", len(hist.Points))
	}
	// Day 1: 1 pass / 2 runs = 0.5; median tokens of [100,300] = 100 (lower-middle).
	if hist.Points[0].SuccessRate != 0.5 || hist.Points[0].RunCount != 2 || hist.Points[0].MedianTokens != 100 {
		t.Fatalf("day1 aggregation wrong: %+v", hist.Points[0])
	}
	// Day 2: 1 pass / 1 = 1.0.
	if hist.Points[1].SuccessRate != 1.0 {
		t.Fatalf("day2 success rate wrong: %+v", hist.Points[1])
	}
	if len(hist.RecentRuns) != 3 {
		t.Fatalf("recent runs = %d", len(hist.RecentRuns))
	}
}

func TestRunTrialsEvaluatorDecidesVerdict(t *testing.T) {
	// The runner returns EVIDENCE (no verdict); the evaluator decides.
	runner := &fakeRunner{result: RunResult{Verdict: VerdictUnspecified, Tokens: 900, DurationMs: 3000, SandboxDiffRef: "sbx"}}
	ev := &fakeEvaluator{verdict: VerdictFail}
	repo := &fakeRepo{}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, repo,
		&fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1"}}, ev,
		scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)))

	runs, err := svc.RunTrials(context.Background(), "", "trial/g1")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(runs) != 1 || runs[0].Verdict != VerdictFail {
		t.Fatalf("evaluator verdict not applied: %+v", runs)
	}
	if ev.calls != 1 || ev.lastRes.Tokens != 900 {
		t.Fatalf("evaluator not called with evidence: calls=%d res=%+v", ev.calls, ev.lastRes)
	}
	if runs[0].FixtureRev != "rev1" {
		t.Fatalf("fixture rev not recorded: %+v", runs[0])
	}
}

func TestRunTrialsRunnerErrorSkipsEvaluator(t *testing.T) {
	runner := &fakeRunner{result: RunResult{Verdict: VerdictError, Detail: "spawn failed"}}
	ev := &fakeEvaluator{verdict: VerdictPass}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, &fakeRepo{},
		&fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1"}}, ev,
		scheduletest.New(time.Now()))
	runs, err := svc.RunTrials(context.Background(), "", "trial/g1")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if runs[0].Verdict != VerdictError {
		t.Fatalf("runner error must NOT be overridden to a pass: %+v", runs[0])
	}
	if ev.calls != 0 {
		t.Fatalf("evaluator must be skipped on a runner error, calls=%d", ev.calls)
	}
}

func TestRunTrialsMissingFixtureRecordsError(t *testing.T) {
	runner := &fakeRunner{result: RunResult{Verdict: VerdictUnspecified}}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, &fakeRepo{},
		&fakeFixtures{missing: map[string]bool{"trial/g1": true}}, &fakeEvaluator{verdict: VerdictPass},
		scheduletest.New(time.Now()))
	runs, err := svc.RunTrials(context.Background(), "", "trial/g1")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if runs[0].Verdict != VerdictError {
		t.Fatalf("missing fixture must record VerdictError, got %+v", runs[0])
	}
	if runner.calls != 0 {
		t.Fatalf("runner must not dispatch when no fixture exists, calls=%d", runner.calls)
	}
}

func TestRunTrialsIdempotencyReuse(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	runner := &fakeRunner{result: RunResult{Verdict: VerdictUnspecified, Tokens: 100}}
	ev := &fakeEvaluator{verdict: VerdictPass}
	repo := &fakeRepo{}
	fx := &fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1"}}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, repo, fx, ev, clk)

	// First run dispatches and records.
	if _, err := svc.RunTrials(context.Background(), "", "trial/g1"); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || len(repo.runs) != 1 {
		t.Fatalf("first run should dispatch+record: calls=%d recorded=%d", runner.calls, len(repo.runs))
	}

	// Immediate identical re-run (same task and fixture revision) REUSES — no new
	// dispatch, no new record.
	runs, err := svc.RunTrials(context.Background(), "", "trial/g1")
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 {
		t.Fatalf("identical re-run must not dispatch again, calls=%d", runner.calls)
	}
	if len(repo.runs) != 1 || runs[0].ID != repo.runs[0].ID {
		t.Fatalf("re-run should reuse the prior run, got %+v", runs[0])
	}

	// A changed fixture revision is a different key → dispatches.
	fx.fixture.Rev = "rev2"
	if _, err := svc.RunTrials(context.Background(), "", "trial/g1"); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 2 {
		t.Fatalf("changed fixture revision must dispatch, calls=%d", runner.calls)
	}
}

func TestRunTrialsIdempotencyWindowExpiry(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	runner := &fakeRunner{result: RunResult{Verdict: VerdictUnspecified}}
	// Seed a prior run OLDER than the idempotency window.
	repo := &fakeRepo{runs: []TrialRun{{
		ID: "old", TaskID: "trial/g1", Model: "ollama/x", FixtureRev: "rev1",
		Verdict: VerdictPass, At: clk.Now().Add(-idempotencyWindow - time.Minute),
	}}}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, repo,
		&fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1"}}, &fakeEvaluator{verdict: VerdictPass}, clk)

	if _, err := svc.RunTrials(context.Background(), "", "trial/g1"); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 {
		t.Fatalf("a stale prior run must NOT be reused, calls=%d", runner.calls)
	}
}

func TestRunTrialsIdempotencyIgnoresPriorError(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	runner := &fakeRunner{result: RunResult{Verdict: VerdictUnspecified}}
	// A recent prior run that ERRORED must not be reused (retry is reasonable).
	repo := &fakeRepo{runs: []TrialRun{{
		ID: "errd", TaskID: "trial/g1", Model: "ollama/x", FixtureRev: "rev1",
		Verdict: VerdictError, At: clk.Now(),
	}}}
	svc := newSvcFull(&fakeGen{tasks: tasksFixture()}, runner, repo,
		&fakeFixtures{fixture: Fixture{Family: "f", Rev: "rev1"}}, &fakeEvaluator{verdict: VerdictPass}, clk)

	if _, err := svc.RunTrials(context.Background(), "", "trial/g1"); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 {
		t.Fatalf("a prior errored run must not block a retry, calls=%d", runner.calls)
	}
}

func TestRunResultErrorVerdictOnNilRunner(t *testing.T) {
	r := NewRunnerWithCommand(nil)
	res := r.RunTask(context.Background(), TrialTask{ID: "x"}, Fixture{Family: "f"})
	if res.Verdict != VerdictError {
		t.Fatalf("nil runner should yield VerdictError, got %v", res.Verdict)
	}
}
