package brands

import "context"

// Repository is the persistence seam for Brand entities. Production wires the
// sqlite-backed implementation from sqlite.go; service unit tests wire
// mocks.FakeRepository. Keep the surface narrow — new methods land here when
// the service proves it needs them.
type Repository interface {
	// Create persists b. The implementation populates ID, Version (1),
	// CreatedAt, and UpdatedAt; callers leaving those zero-valued is the
	// canonical shape. Returns the persisted Brand.
	Create(ctx context.Context, b Brand) (Brand, error)

	// Get returns the brand with the given ID or ErrBrandNotFound{ID} when no
	// row matches.
	Get(ctx context.Context, id string) (Brand, error)

	// List returns brands matching filter, ordered newest-updated first.
	List(ctx context.Context, filter ListFilter) ([]Brand, error)

	// Update persists the mutated brand, incrementing Version and refreshing
	// UpdatedAt. Returns ErrBrandNotFound{ID} when no row matches. Returns the
	// persisted Brand (with the new Version + UpdatedAt).
	Update(ctx context.Context, b Brand) (Brand, error)

	// Delete removes the brand with the given ID. Returns ErrBrandNotFound{ID}
	// when no row matched (the service translates that into an idempotent
	// success at the application layer).
	Delete(ctx context.Context, id string) error
}

// VersionRepository is the persistence seam for immutable BrandVersion
// snapshots. Method names are distinct from Repository's so a single
// sqliteRepository struct can satisfy both seams without a name collision.
type VersionRepository interface {
	// CreateVersion persists a snapshot. The implementation populates ID and
	// CreatedAt when zero-valued.
	CreateVersion(ctx context.Context, v BrandVersion) (BrandVersion, error)

	// ListVersions returns every version for brandID, newest-first by version.
	ListVersions(ctx context.Context, brandID string) ([]BrandVersion, error)
}
