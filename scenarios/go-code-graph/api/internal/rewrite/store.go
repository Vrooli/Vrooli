package rewrite

import "context"

// PlanStore persists plans between Plan and Apply. Production wires
// MemoryStore; tests wire FakeStore from mocks/store.go.
//
// Load MUST return a typed RewriteError{Kind: RewriteErrorPlanNotFound}
// when the id is unknown so the Service can map it to the right
// Connect code without sniffing strings.
type PlanStore interface {
	Save(ctx context.Context, plan Plan) error
	Load(ctx context.Context, id PlanID) (Plan, error)
}
