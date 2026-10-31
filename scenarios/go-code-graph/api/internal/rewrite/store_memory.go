package rewrite

import (
	"context"
	"sync"
)

// MemoryStore is the v1 in-process PlanStore. Plans live until the
// process exits; the follow-up to persist them (REQ-P1-002) is tracked
// in docs/internal/PROBLEMS.md.
type MemoryStore struct {
	m sync.Map // map[PlanID]Plan
}

// NewMemoryStore returns a fresh, empty MemoryStore typed as PlanStore
// so the production wire-up (api/main.go) stores the interface and
// tests cannot accidentally reach for unexported fields.
func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

// Save persists plan keyed by plan.ID. Re-saving an existing plan is a
// no-op overwrite — Plan derivation is deterministic so identical
// inputs intentionally collide.
func (s *MemoryStore) Save(_ context.Context, plan Plan) error {
	s.m.Store(plan.ID, plan)
	return nil
}

// Load returns the stored plan or RewriteError{Kind: PlanNotFound}.
func (s *MemoryStore) Load(_ context.Context, id PlanID) (Plan, error) {
	v, ok := s.m.Load(id)
	if !ok {
		return Plan{}, RewriteError{
			Kind:    RewriteErrorPlanNotFound,
			Message: "unknown plan_id " + string(id),
		}
	}
	return v.(Plan), nil
}

// Compile-time assertion.
var _ PlanStore = (*MemoryStore)(nil)
