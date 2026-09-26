// Package mocks provides in-memory fakes for the assignments domain seams so
// service unit tests run without the sqlite round-trip. Mirrors
// internal/brands/mocks.
package mocks

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"brand-manager/internal/assignments"

	"github.com/google/uuid"
)

// FakeRepository satisfies assignments.Repository with an in-memory map keyed by
// scenario_name (the natural key). Per-method error knobs drive failure paths;
// atomic counters keep `go test -race` quiet under fan-out.
type FakeRepository struct {
	mu    sync.Mutex
	byScn map[string]assignments.Assignment

	UpsertErr error
	GetErr    error
	ListErr   error
	DeleteErr error

	UpsertCalls atomic.Int64
	DeleteCalls atomic.Int64
}

func (f *FakeRepository) Upsert(_ context.Context, a assignments.Assignment) (assignments.Assignment, error) {
	f.UpsertCalls.Add(1)
	if f.UpsertErr != nil {
		return assignments.Assignment{}, f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byScn == nil {
		f.byScn = map[string]assignments.Assignment{}
	}
	if existing, ok := f.byScn[a.ScenarioName]; ok && a.ID == "" {
		a.ID = existing.ID
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.AppliedAt.IsZero() {
		a.AppliedAt = time.Now().UTC()
	}
	f.byScn[a.ScenarioName] = a
	return a, nil
}

func (f *FakeRepository) GetByScenario(_ context.Context, scenarioName string) (assignments.Assignment, error) {
	if f.GetErr != nil {
		return assignments.Assignment{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.byScn[scenarioName]
	if !ok {
		return assignments.Assignment{}, assignments.ErrAssignmentNotFound{Scenario: scenarioName}
	}
	return a, nil
}

func (f *FakeRepository) ListByBrand(_ context.Context, brandID string) ([]assignments.Assignment, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]assignments.Assignment, 0, len(f.byScn))
	for _, a := range f.byScn {
		if brandID == "" || a.BrandID == brandID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].AppliedAt.Equal(out[j].AppliedAt) {
			return out[i].AppliedAt.After(out[j].AppliedAt)
		}
		return out[i].ScenarioName < out[j].ScenarioName
	})
	return out, nil
}

func (f *FakeRepository) DeleteByScenario(_ context.Context, scenarioName string) error {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byScn[scenarioName]; !ok {
		return assignments.ErrAssignmentNotFound{Scenario: scenarioName}
	}
	delete(f.byScn, scenarioName)
	return nil
}

// Seed inserts a directly, bypassing Upsert, for arranging test state.
func (f *FakeRepository) Seed(a assignments.Assignment) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byScn == nil {
		f.byScn = map[string]assignments.Assignment{}
	}
	f.byScn[a.ScenarioName] = a
}

var _ assignments.Repository = (*FakeRepository)(nil)

// FakeBrandResolver satisfies assignments.BrandResolver from a fixed
// version-by-brand map. Brands absent from the map resolve ok=false; set Err to
// drive the lookup-failure path.
type FakeBrandResolver struct {
	Versions map[string]int
	Err      error
}

func (f FakeBrandResolver) BrandVersion(_ context.Context, brandID string) (int, bool, error) {
	if f.Err != nil {
		return 0, false, f.Err
	}
	v, ok := f.Versions[brandID]
	return v, ok, nil
}

var _ assignments.BrandResolver = (*FakeBrandResolver)(nil)
