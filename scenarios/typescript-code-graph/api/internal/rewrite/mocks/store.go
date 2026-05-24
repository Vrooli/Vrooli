// Package mocks holds test doubles for the rewrite seams.
//
// The production MemoryPlanStore is already an in-memory map and is
// usable directly in most unit tests; FakePlanStore exists so tests
// that need to force a Save or Get failure (e.g. exercising the
// service_error_mapping path) can do so without monkey-patching.
package mocks

import (
	"sync"

	"typescript-code-graph/internal/rewrite"
)

// FakePlanStore is a programmable PlanStore for tests that need to
// inject errors. By default it behaves identically to
// rewrite.MemoryPlanStore.
type FakePlanStore struct {
	mu     sync.Mutex
	plans  map[fakeKey]rewrite.Plan
	SaveFn func(plan rewrite.Plan) error
	GetFn  func(scenarioPath string, id rewrite.PlanID) (rewrite.Plan, error)

	SaveCalls int
	GetCalls  int
}

type fakeKey struct {
	scenarioPath string
	id           rewrite.PlanID
}

// NewFakePlanStore returns a ready-to-use fake.
func NewFakePlanStore() *FakePlanStore {
	return &FakePlanStore{plans: make(map[fakeKey]rewrite.Plan)}
}

// Save records the call and dispatches to SaveFn if set; otherwise
// stores normally.
func (f *FakePlanStore) Save(plan rewrite.Plan) error {
	f.mu.Lock()
	f.SaveCalls++
	fn := f.SaveFn
	f.mu.Unlock()
	if fn != nil {
		return fn(plan)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.plans[fakeKey{scenarioPath: plan.ScenarioPath, id: plan.ID}] = plan
	return nil
}

// Get records the call and dispatches to GetFn if set; otherwise
// returns from the in-memory map (or ErrPlanNotFound).
func (f *FakePlanStore) Get(scenarioPath string, id rewrite.PlanID) (rewrite.Plan, error) {
	f.mu.Lock()
	f.GetCalls++
	fn := f.GetFn
	f.mu.Unlock()
	if fn != nil {
		return fn(scenarioPath, id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.plans[fakeKey{scenarioPath: scenarioPath, id: id}]
	if !ok {
		return rewrite.Plan{}, rewrite.ErrPlanNotFound
	}
	return p, nil
}

// Compile-time guarantee.
var _ rewrite.PlanStore = (*FakePlanStore)(nil)
