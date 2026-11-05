package mocks

import (
	"context"

	intrewrite "go-code-graph/internal/rewrite"
)

// FakeStore is the canned PlanStore for tests. Both SaveFunc and
// LoadFunc are optional; the zero value persists to an internal map.
type FakeStore struct {
	SaveFunc func(ctx context.Context, plan intrewrite.Plan) error
	LoadFunc func(ctx context.Context, id intrewrite.PlanID) (intrewrite.Plan, error)

	plans map[intrewrite.PlanID]intrewrite.Plan
}

// Save delegates to SaveFunc when set, otherwise stores in-memory.
func (f *FakeStore) Save(ctx context.Context, plan intrewrite.Plan) error {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, plan)
	}
	if f.plans == nil {
		f.plans = make(map[intrewrite.PlanID]intrewrite.Plan)
	}
	f.plans[plan.ID] = plan
	return nil
}

// Load delegates to LoadFunc when set, otherwise reads the in-memory
// map (returning PlanNotFound on miss).
func (f *FakeStore) Load(ctx context.Context, id intrewrite.PlanID) (intrewrite.Plan, error) {
	if f.LoadFunc != nil {
		return f.LoadFunc(ctx, id)
	}
	if p, ok := f.plans[id]; ok {
		return p, nil
	}
	return intrewrite.Plan{}, intrewrite.RewriteError{
		Kind:    intrewrite.RewriteErrorPlanNotFound,
		Message: "unknown plan_id " + string(id),
	}
}

// Compile-time assertion.
var _ intrewrite.PlanStore = (*FakeStore)(nil)
