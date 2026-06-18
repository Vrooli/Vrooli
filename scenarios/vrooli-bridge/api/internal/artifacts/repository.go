package artifacts

import "context"

// Repository is the persistence seam the artifacts service depends on.
// Production wires the sqlite-backed implementation (sqlite.go); service unit
// tests wire mocks.FakeRepository. It persists the durable Distribution record
// (reference + metadata only — never the bytes).
type Repository interface {
	// Create persists a new distribution. The implementation populates ID (when
	// empty), CreatedAt, and UpdatedAt. Returns the persisted distribution.
	Create(ctx context.Context, d Distribution) (Distribution, error)

	// Get returns the distribution with the given id or ErrDistributionNotFound{id}.
	Get(ctx context.Context, id string) (Distribution, error)

	// List returns distributions newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Distribution, error)

	// UpdateStatus persists the mutable lifecycle columns (status, delivery_ref,
	// detail, updated_at) matched by id and returns the stored distribution.
	UpdateStatus(ctx context.Context, id string, status DeliveryStatus, deliveryRef, detail string) (Distribution, error)
}
