package mocks

import (
	"context"
	"sort"
	"sync"

	"data-backup-manager/internal/runs"
)

// FakeRepository is an in-memory runs.Repository for handler tests and any
// service test that doesn't need real SQL. The catalog/last-success rollup
// (TargetStatuses) mirrors the sqlite logic so the fake stays behaviorally
// faithful.
type FakeRepository struct {
	mu   sync.Mutex
	runs []runs.Run
	seq  int

	CreateErr error
	SaveErr   error
	GetErr    error
	ListErr   error
}

func NewFakeRepository() *FakeRepository { return &FakeRepository{} }

func (f *FakeRepository) CreateRun(_ context.Context, r runs.Run) (runs.Run, error) {
	if f.CreateErr != nil {
		return runs.Run{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.ID == "" {
		f.seq++
		r.ID = "run-" + itoa(f.seq)
	}
	f.runs = append(f.runs, r)
	return r, nil
}

func (f *FakeRepository) SaveRun(_ context.Context, r runs.Run) (runs.Run, error) {
	if f.SaveErr != nil {
		return runs.Run{}, f.SaveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.runs {
		if f.runs[i].ID == r.ID {
			f.runs[i] = r
			return r, nil
		}
	}
	f.runs = append(f.runs, r)
	return r, nil
}

func (f *FakeRepository) GetRun(_ context.Context, id string) (runs.Run, error) {
	if f.GetErr != nil {
		return runs.Run{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.runs {
		if r.ID == id {
			return r, nil
		}
	}
	return runs.Run{}, runs.ErrRunNotFound{ID: id}
}

func (f *FakeRepository) ListRuns(_ context.Context, planID string, limit int) ([]runs.Run, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	if limit <= 0 {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]runs.Run, 0, len(f.runs))
	for _, r := range f.runs {
		if planID != "" && r.PlanID != planID {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) TargetStatuses(_ context.Context, targetIDs []string) ([]runs.TargetStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := map[string]bool{}
	for _, id := range targetIDs {
		want[id] = true
	}
	// Latest run per target + last success.
	ordered := append([]runs.Run(nil), f.runs...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].StartedAt.After(ordered[j].StartedAt) })

	byTarget := map[string]*runs.TargetStatus{}
	order := []string{}
	for _, r := range ordered {
		for _, o := range r.Outcomes {
			if len(want) > 0 && !want[o.TargetID] {
				continue
			}
			ts, ok := byTarget[o.TargetID]
			if !ok {
				ts = &runs.TargetStatus{TargetID: o.TargetID, LastRunStatus: r.Status, LastRunAt: r.StartedAt}
				byTarget[o.TargetID] = ts
				order = append(order, o.TargetID)
			}
			if o.Status == runs.OutcomeSucceeded && o.FinishedAt.After(ts.LastSuccessAt) {
				ts.LastSuccessAt = o.FinishedAt
			}
		}
	}
	out := make([]runs.TargetStatus, 0, len(order))
	for _, id := range order {
		out = append(out, *byTarget[id])
	}
	return out, nil
}

var _ runs.Repository = (*FakeRepository)(nil)

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
