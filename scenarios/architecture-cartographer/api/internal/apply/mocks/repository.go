// Package mocks holds in-memory fakes for the apply domain.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/conflicts"

	"github.com/google/uuid"
)

// FakeRepository satisfies apply.Repository.
type FakeRepository struct {
	mu sync.Mutex

	Plans []apply.Plan
	Runs  []apply.ApplyRun
	Baseline apply.BuildBaseline

	SaveErr     error
	GetErr      error
	ListErr     error
	BaselineErr error

	SaveCalls     atomic.Int64
	GetCalls      atomic.Int64
	ListCalls     atomic.Int64
	BaselineCalls atomic.Int64
}

func (f *FakeRepository) SavePlan(_ context.Context, p apply.Plan) (apply.Plan, error) {
	f.SaveCalls.Add(1)
	if f.SaveErr != nil {
		return apply.Plan{}, f.SaveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.PlannedAt.IsZero() {
		p.PlannedAt = time.Now().UTC()
	}
	f.Plans = append(f.Plans, p)
	return p, nil
}

func (f *FakeRepository) GetPlan(_ context.Context, id string) (apply.Plan, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return apply.Plan{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.Plans {
		if p.ID == id {
			return p, nil
		}
	}
	return apply.Plan{}, apply.ErrInvalidPlanRequest{Field: "id", Reason: "not found"}
}

func (f *FakeRepository) ListRuns(_ context.Context, filter apply.ListRunsFilter) (apply.RunPage, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return apply.RunPage{}, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []apply.ApplyRun
	for _, r := range f.Runs {
		if filter.Scenario != "" && r.Scenario != filter.Scenario {
			continue
		}
		if filter.Domain != "" && r.Domain != filter.Domain {
			continue
		}
		out = append(out, r)
	}
	return apply.RunPage{Runs: out}, nil
}

func (f *FakeRepository) GetBaseline(_ context.Context, scenario string) (apply.BuildBaseline, error) {
	f.BaselineCalls.Add(1)
	if f.BaselineErr != nil {
		return apply.BuildBaseline{}, f.BaselineErr
	}
	if f.Baseline.Scenario == scenario {
		return f.Baseline, nil
	}
	return apply.BuildBaseline{Scenario: scenario}, nil
}

var _ apply.Repository = (*FakeRepository)(nil)

// FakeConflictLister satisfies apply.ConflictLister.
type FakeConflictLister struct {
	Conflicts []conflicts.Conflict
	Err       error
	Calls     atomic.Int64
}

func (f *FakeConflictLister) ListConflicts(_ context.Context, filter conflicts.ListConflictsFilter) (conflicts.ConflictPage, error) {
	f.Calls.Add(1)
	if f.Err != nil {
		return conflicts.ConflictPage{}, f.Err
	}
	var out []conflicts.Conflict
	for _, c := range f.Conflicts {
		if filter.Scenario != "" && c.Scenario != filter.Scenario {
			continue
		}
		if len(filter.Statuses) > 0 {
			matched := false
			for _, s := range filter.Statuses {
				if s == c.Status {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, c)
	}
	return conflicts.ConflictPage{Conflicts: out}, nil
}

var _ apply.ConflictLister = (*FakeConflictLister)(nil)

// FakeBuildGuard implements apply.BuildGuard.
type FakeBuildGuard struct {
	NameValue       string
	BaselineResult  apply.BuildBaseline
	BaselineErr     error
	VerifyResult    apply.BuildBaseline
	VerifyErr       error
	BaselineCalls   atomic.Int64
	VerifyCalls     atomic.Int64
}

func (f *FakeBuildGuard) Name() string { return f.NameValue }
func (f *FakeBuildGuard) Baseline(_ context.Context, _ string) (apply.BuildBaseline, error) {
	f.BaselineCalls.Add(1)
	return f.BaselineResult, f.BaselineErr
}
func (f *FakeBuildGuard) Verify(_ context.Context, _ string) (apply.BuildBaseline, error) {
	f.VerifyCalls.Add(1)
	return f.VerifyResult, f.VerifyErr
}

var _ apply.BuildGuard = (*FakeBuildGuard)(nil)

// FakeService satisfies apply.Service for handler tests.
type FakeService struct {
	Plan     apply.Plan
	Baseline apply.BuildBaseline
	Page     apply.RunPage
	NextErr  error

	PlanCalls     atomic.Int64
	RunCalls      atomic.Int64
	HistoryCalls  atomic.Int64
	BaselineCalls atomic.Int64
}

func (f *FakeService) PlanApply(_ context.Context, _ apply.PlanInput) (apply.Plan, bool, error) {
	f.PlanCalls.Add(1)
	if f.NextErr != nil {
		return apply.Plan{}, false, f.NextErr
	}
	return f.Plan, false, nil
}

func (f *FakeService) RunApply(_ context.Context, _ string, _ bool) (apply.ApplyRun, error) {
	f.RunCalls.Add(1)
	if f.NextErr != nil {
		return apply.ApplyRun{}, f.NextErr
	}
	return apply.ApplyRun{}, apply.ErrApplyUnimplemented{NextPlan: "architecture-cartographer-apply-execution"}
}

func (f *FakeService) ListApplyHistory(_ context.Context, _ apply.ListRunsFilter) (apply.RunPage, error) {
	f.HistoryCalls.Add(1)
	if f.NextErr != nil {
		return apply.RunPage{}, f.NextErr
	}
	return f.Page, nil
}

func (f *FakeService) GetBuildBaseline(_ context.Context, _ string) (apply.BuildBaseline, error) {
	f.BaselineCalls.Add(1)
	if f.NextErr != nil {
		return apply.BuildBaseline{}, f.NextErr
	}
	return f.Baseline, nil
}

var _ apply.Service = (*FakeService)(nil)
