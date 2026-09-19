package gate

import "context"

// Repository is the persistence seam the gate service depends on. Production
// wires the sqlite-backed implementation (sqlite.go); service unit tests wire
// mocks.FakeRepository. It persists the durable Gate record and its per-OS
// result ledger.
type Repository interface {
	// Create persists gate + its per-OS results as one record. The
	// implementation populates ID (when empty) and CreatedAt. Returns the
	// persisted gate.
	Create(ctx context.Context, g Gate, results []OSResult) (Gate, error)

	// Get returns the gate with the given id or ErrGateNotFound{id}.
	Get(ctx context.Context, id string) (Gate, error)

	// Results returns a gate's per-OS ledger in stable (os) order.
	Results(ctx context.Context, gateID string) ([]OSResult, error)

	// List returns gates newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Gate, error)
}
