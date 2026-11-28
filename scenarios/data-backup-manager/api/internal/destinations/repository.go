package destinations

import "context"

// Repository is the persistence seam the destinations service depends on.
// Production wires the sqlite-backed implementation from sqlite.go; service
// unit tests wire mocks.FakeRepository (from internal/destinations/mocks). The
// surface is narrow — the service composes CRUD from these primitives; the
// repository owns no policy.
//
// seam: Repository persists Destination rows keyed by name. Production wires
// SqliteRepository (sqlite.go); tests wire FakeRepository (mocks/).
type Repository interface {
	// Create persists a new destination. The implementation populates ID,
	// CreatedAt, and UpdatedAt. Returns the persisted Destination.
	Create(ctx context.Context, d Destination) (Destination, error)

	// Update overwrites mutable destination metadata, policy and recovery
	// identity for the existing row identified by d.ID, and bumps UpdatedAt.
	// Callers load the row first so immutable repository fields are preserved.
	Update(ctx context.Context, d Destination) (Destination, error)

	// GetByID returns the destination with the given id or ErrDestinationNotFound.
	GetByID(ctx context.Context, id string) (Destination, error)

	// GetByName returns the destination keyed by name or ErrDestinationNotFound.
	GetByName(ctx context.Context, name string) (Destination, error)

	// List returns up to limit destinations ordered by name. limit <= 0 returns
	// no rows.
	List(ctx context.Context, limit int) ([]Destination, error)

	// Delete removes the destination by id. Returns true when a row was removed,
	// false when none matched.
	Delete(ctx context.Context, id string) (bool, error)
}
