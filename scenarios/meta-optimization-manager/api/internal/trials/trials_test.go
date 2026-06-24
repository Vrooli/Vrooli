package trials

import (
	"context"
	"testing"
	"time"

	"meta-optimization-manager/internal/testutil/mocks"
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

func (f *fakeRunner) RunTask(_ context.Context, _ TrialTask, model string) RunResult {
	f.calls++
	r := f.result
	if r.Model == "" {
		r.Model = model
	}
	return r
}

type fakeRepo struct {
	runs      []TrialRun
	gated     int
	recordErr error
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

func newSvc(gen TaskGenerator, runner Runner, repo Repository) Service {
	return NewService(Deps{Tasks: gen, Runner: runner, Repo: repo, Clock: mocks.NewFakeClock(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))})
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
	runs, err := svc.RunTrials(context.Background(), "", "trial/g1", "ollama/x")
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
	runs, err = svc.RunTrials(context.Background(), SuiteAddFeature, "", "")
	if err != nil || len(runs) != 1 || runner.calls != 1 {
		t.Fatalf("suite run failed: runs=%d calls=%d err=%v", len(runs), runner.calls, err)
	}
}

func TestRunTrialsUnknownTaskErrors(t *testing.T) {
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, &fakeRunner{}, &fakeRepo{})
	if _, err := svc.RunTrials(context.Background(), "", "trial/nope", ""); err == nil {
		t.Fatalf("expected error for unknown task")
	}
}

func TestRunTrialsAllTasksWhenUnscoped(t *testing.T) {
	runner := &fakeRunner{result: RunResult{Verdict: VerdictPass}}
	svc := newSvc(&fakeGen{tasks: tasksFixture()}, runner, &fakeRepo{})
	runs, err := svc.RunTrials(context.Background(), "", "", "")
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

func TestRunResultErrorVerdictOnNilRunner(t *testing.T) {
	r := NewRunnerWithCommand(nil)
	res := r.RunTask(context.Background(), TrialTask{ID: "x"}, "")
	if res.Verdict != VerdictError {
		t.Fatalf("nil runner should yield VerdictError, got %v", res.Verdict)
	}
}

func TestParseDispatch(t *testing.T) {
	res := parseDispatch([]byte(`{"verdict":"pass","tokens":1234,"duration_ms":4567,"sandbox_diff_ref":"sbx-9","model":"ollama/z"}`), "fallback")
	if res.Verdict != VerdictPass || res.Tokens != 1234 || res.SandboxDiffRef != "sbx-9" || res.Model != "ollama/z" {
		t.Fatalf("parse dispatch wrong: %+v", res)
	}
	bad := parseDispatch([]byte("not json"), "fallback")
	if bad.Verdict != VerdictError || bad.Model != "fallback" {
		t.Fatalf("bad parse should error: %+v", bad)
	}
}
