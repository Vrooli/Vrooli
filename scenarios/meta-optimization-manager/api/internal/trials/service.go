package trials

import (
	"context"
	"fmt"

	"meta-optimization-manager/internal/clock"

	"github.com/google/uuid"
)

// recentRunsLimit bounds the recent-runs list returned alongside the trend.
const recentRunsLimit = 20

// Service is the trials application surface.
type Service interface {
	ListTrialTasks(ctx context.Context, suite string) ([]TrialTask, error)
	RunTrials(ctx context.Context, suite, taskID, model string) ([]TrialRun, error)
	GetTrialHistory(ctx context.Context, taskID, suite string) (History, error)
	GetTrialRun(ctx context.Context, id string) (TrialRun, error)
	GetGateCoverage(ctx context.Context) (GateCoverage, error)
}

type service struct {
	tasks  TaskGenerator
	runner Runner
	repo   Repository
	clock  clock.Clock
}

// Deps wires the trials Service. Repo is optional (nil disables persistence).
type Deps struct {
	Tasks  TaskGenerator
	Runner Runner
	Repo   Repository
	Clock  clock.Clock
}

// NewService constructs the trials Service.
func NewService(d Deps) Service {
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	if d.Runner == nil {
		d.Runner = NewRunner()
	}
	return &service{tasks: d.Tasks, runner: d.Runner, repo: d.Repo, clock: d.Clock}
}

var _ Service = (*service)(nil)

// ListTrialTasks returns the generated suite, optionally filtered.
func (s *service) ListTrialTasks(ctx context.Context, suite string) ([]TrialTask, error) {
	if s.tasks == nil {
		return nil, nil
	}
	return s.tasks.Generate(ctx, suite)
}

// RunTrials dispatches a task or suite through the runner and records each run.
// EXPLICIT INVOCATION ONLY — this is the one trials action; it is never on a hot
// path and always sandboxed. A single task (task_id) takes precedence over a
// suite filter; with neither, the whole generated suite runs.
func (s *service) RunTrials(ctx context.Context, suite, taskID, model string) ([]TrialRun, error) {
	tasks, err := s.resolveTasks(ctx, suite, taskID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("trials: no matching tasks to run")
	}
	runs := make([]TrialRun, 0, len(tasks))
	for _, t := range tasks {
		res := s.runner.RunTask(ctx, t, model)
		run := TrialRun{
			ID:             uuid.NewString(),
			TaskID:         t.ID,
			Suite:          t.Suite,
			Model:          res.Model,
			GuideTaskID:    t.GuideTaskID,
			Verdict:        res.Verdict,
			Tokens:         res.Tokens,
			DurationMs:     res.DurationMs,
			SandboxDiffRef: res.SandboxDiffRef,
			At:             s.clock.Now().UTC(),
		}
		if s.repo != nil {
			if err := s.repo.RecordRun(ctx, run); err != nil {
				// A record failure must not lose the dispatched result; surface it
				// but keep going so a partial suite still reports.
				run.Verdict = VerdictError
			}
		}
		runs = append(runs, run)
	}
	return runs, nil
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
