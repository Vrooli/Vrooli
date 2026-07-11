package trials

import (
	"context"
	"fmt"
	"time"

	"meta-optimization-manager/internal/clock"

	"github.com/google/uuid"
)

// recentRunsLimit bounds the recent-runs list returned alongside the trend.
const recentRunsLimit = 20

// idempotencyWindow is how recently an identical (task, fixture-rev) run
// must have completed for a re-run to reuse it instead of dispatching again.
// Trials are expensive and operator-invoked; this stops an accidental immediate
// re-run from double-spending while still letting the trend grow over time.
const idempotencyWindow = 15 * time.Minute

// Service is the trials application surface.
type Service interface {
	ListTrialTasks(ctx context.Context, suite string) ([]TrialTask, error)
	RunTrials(ctx context.Context, suite, taskID string) ([]TrialRun, error)
	GetTrialHistory(ctx context.Context, taskID, suite string) (History, error)
	GetTrialRun(ctx context.Context, id string) (TrialRun, error)
	GetGateCoverage(ctx context.Context) (GateCoverage, error)
}

type service struct {
	tasks     TaskGenerator
	fixtures  FixtureResolver
	runner    Runner
	evaluator Evaluator
	repo      Repository
	clock     clock.Clock
}

// Deps wires the trials Service. Repo is optional (nil disables persistence);
// Fixtures/Runner/Evaluator default to their production implementations.
type Deps struct {
	Tasks     TaskGenerator
	Fixtures  FixtureResolver
	Runner    Runner
	Evaluator Evaluator
	Repo      Repository
	Clock     clock.Clock
}

// NewService constructs the trials Service.
func NewService(d Deps) Service {
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	if d.Runner == nil {
		d.Runner = NewRunner()
	}
	if d.Fixtures == nil {
		d.Fixtures = NewFixtureResolver()
	}
	if d.Evaluator == nil {
		d.Evaluator = NewEvaluator(nil)
	}
	return &service{
		tasks:     d.Tasks,
		fixtures:  d.Fixtures,
		runner:    d.Runner,
		evaluator: d.Evaluator,
		repo:      d.Repo,
		clock:     d.Clock,
	}
}

var _ Service = (*service)(nil)

// ListTrialTasks returns the generated suite, optionally filtered.
func (s *service) ListTrialTasks(ctx context.Context, suite string) ([]TrialTask, error) {
	if s.tasks == nil {
		return nil, nil
	}
	return s.tasks.Generate(ctx, suite)
}

// RunTrials runs a task or suite end to end and records each run. EXPLICIT
// INVOCATION ONLY — this is the one trials action; it is never on a hot path and
// always sandboxed. A single task (task_id) takes precedence over a suite
// filter; with neither, the whole generated suite runs.
//
// Per task the flow is: resolve the fixture (defines success + supplies the
// idempotency rev) → reuse a recent identical run if one exists (no double
// spend) → Runner collects evidence via agent-manager's sandboxed primitive →
// Evaluator decides the verdict → record. Every failure mode degrades that one
// run to an honest VerdictError and is still recorded; the suite never blocks.
func (s *service) RunTrials(ctx context.Context, suite, taskID string) ([]TrialRun, error) {
	tasks, err := s.resolveTasks(ctx, suite, taskID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("trials: no matching tasks to run")
	}
	runs := make([]TrialRun, 0, len(tasks))
	for _, t := range tasks {
		runs = append(runs, s.runOne(ctx, t))
	}
	return runs, nil
}

// runOne resolves the fixture, applies idempotency, dispatches, evaluates, and
// records one task. It never returns an error — failures become a recorded
// VerdictError so a partial suite still reports.
func (s *service) runOne(ctx context.Context, t TrialTask) TrialRun {
	// A missing fixture means no deterministic substrate exists for this family —
	// an honest VerdictError (recorded for history), never a fabricated pass.
	fixture, ok, ferr := s.fixtures.Resolve(ctx, t)
	if ferr != nil || !ok {
		return s.record(ctx, TrialRun{
			ID:          uuid.NewString(),
			TaskID:      t.ID,
			Suite:       t.Suite,
			GuideTaskID: t.GuideTaskID,
			Verdict:     VerdictError,
			At:          s.clock.Now().UTC(),
		})
	}

	// Idempotency: reuse a recent, identical, non-error run rather than spend
	// another local-model dispatch on it.
	if reused, found := s.reusableRun(ctx, t.ID, fixture.Rev); found {
		return reused
	}

	res := s.runner.RunTask(ctx, t, fixture)
	verdict := res.Verdict
	if verdict != VerdictError {
		verdict = s.evaluator.Judge(ctx, t, fixture, res)
	}
	return s.record(ctx, TrialRun{
		ID:             uuid.NewString(),
		TaskID:         t.ID,
		Suite:          t.Suite,
		GuideTaskID:    t.GuideTaskID,
		FixtureRev:     fixture.Rev,
		Verdict:        verdict,
		Tokens:         res.Tokens,
		DurationMs:     res.DurationMs,
		SandboxDiffRef: res.SandboxDiffRef,
		At:             s.clock.Now().UTC(),
	})
}

// record persists a run (when a repo is wired) and returns it. A record failure
// must not lose the dispatched result: it downgrades the verdict to error but
// keeps going so a partial suite still reports.
func (s *service) record(ctx context.Context, run TrialRun) TrialRun {
	if s.repo != nil {
		if err := s.repo.RecordRun(ctx, run); err != nil {
			run.Verdict = VerdictError
		}
	}
	return run
}

// reusableRun returns a recent identical (task, fixture-rev) run if one
// exists within the idempotency window and did not itself error.
func (s *service) reusableRun(ctx context.Context, taskID, fixtureRev string) (TrialRun, bool) {
	if s.repo == nil || fixtureRev == "" {
		return TrialRun{}, false
	}
	prev, ok, err := s.repo.LatestRun(ctx, taskID, "", fixtureRev)
	if err != nil || !ok {
		return TrialRun{}, false
	}
	if prev.Verdict == VerdictError {
		return TrialRun{}, false
	}
	if s.clock.Now().UTC().Sub(prev.At) > idempotencyWindow {
		return TrialRun{}, false
	}
	return prev, true
}

// resolveTasks picks the tasks to run: one by id, else a suite, else all.
func (s *service) resolveTasks(ctx context.Context, suite, taskID string) ([]TrialTask, error) {
	if s.tasks == nil {
		return nil, fmt.Errorf("trials: task generator unavailable")
	}
	all, err := s.tasks.Generate(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("trials: generate tasks: %w", err)
	}
	if taskID != "" {
		for _, t := range all {
			if t.ID == taskID {
				return []TrialTask{t}, nil
			}
		}
		return nil, fmt.Errorf("trials: task %q not found", taskID)
	}
	if suite != "" {
		var out []TrialTask
		for _, t := range all {
			if t.Suite == suite {
				out = append(out, t)
			}
		}
		return out, nil
	}
	return all, nil
}

// GetTrialHistory returns the aggregated trend plus the most recent runs.
func (s *service) GetTrialHistory(ctx context.Context, taskID, suite string) (History, error) {
	if s.repo == nil {
		return History{}, nil
	}
	filter := RunFilter{TaskID: taskID, Suite: suite}
	all, err := s.repo.Runs(ctx, filter, 0, false) // oldest-first for aggregation
	if err != nil {
		return History{}, err
	}
	recent, err := s.repo.Runs(ctx, filter, recentRunsLimit, true) // newest-first
	if err != nil {
		return History{}, err
	}
	return History{Points: aggregate(all), RecentRuns: recent}, nil
}

// GetTrialRun returns one run by id.
func (s *service) GetTrialRun(ctx context.Context, id string) (TrialRun, error) {
	if s.repo == nil {
		return TrialRun{}, fmt.Errorf("trials: history unavailable")
	}
	run, ok, err := s.repo.GetRun(ctx, id)
	if err != nil {
		return TrialRun{}, err
	}
	if !ok {
		return TrialRun{}, fmt.Errorf("trials: run %q not found", id)
	}
	return run, nil
}

// GetGateCoverage returns the recursive Guide-gate-coverage metric: how many
// Guide-space tasks have at least one live empirical gate.
func (s *service) GetGateCoverage(ctx context.Context) (GateCoverage, error) {
	gc := GateCoverage{}
	if s.tasks != nil {
		all, err := s.tasks.Generate(ctx, "")
		if err == nil {
			// Count distinct non-negative Guide tasks (the denominator is the Guide
			// space, not the synthetic negative cases).
			seen := map[string]bool{}
			for _, t := range all {
				if t.Negative || t.GuideTaskID == "" {
					continue
				}
				seen[t.GuideTaskID] = true
			}
			gc.GuideTasksTotal = len(seen)
		}
	}
	if s.repo != nil {
		gated, err := s.repo.GatedGuideTaskCount(ctx)
		if err == nil {
			gc.GuideTasksWithGate = gated
		}
	}
	if gc.GuideTasksTotal > 0 {
		gc.Ratio = float64(gc.GuideTasksWithGate) / float64(gc.GuideTasksTotal)
	}
	return gc, nil
}
