package fleet

import "context"

// Repository is the persistence seam the fleet service depends on. Production
// wires the sqlite-backed implementation (sqlite.go); service unit tests wire
// mocks.FakeRepository. It persists the durable Rollout record and its per-node
// result ledger.
type Repository interface {
	// Create persists rollout + its per-node results as one rollout. The
	// implementation populates ID (when empty) and CreatedAt. Returns the
	// persisted rollout.
	Create(ctx context.Context, rollout Rollout, results []NodeResult) (Rollout, error)

	// Get returns the rollout with the given id or ErrRolloutNotFound{id}.
	Get(ctx context.Context, id string) (Rollout, error)

	// Results returns a rollout's per-node ledger in stable (node id) order.
	Results(ctx context.Context, rolloutID string) ([]NodeResult, error)

	// List returns rollouts newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Rollout, error)
}
