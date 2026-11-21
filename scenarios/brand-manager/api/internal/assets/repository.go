package assets

import "context"

// Repository is the persistence seam for Asset catalog rows. Production wires
// the sqlite-backed implementation from sqlite.go; service unit tests wire
// mocks.FakeRepository. Keep the surface narrow — new methods land here when
// the service proves it needs them.
type Repository interface {
	// Upsert persists a by its natural key (brand_id, filename): a row with the
	// same brand and filename is replaced in place, preserving the original ID
	// and CreatedAt. The implementation populates ID and CreatedAt on first
	// insert. Returns the persisted Asset.
	Upsert(ctx context.Context, a Asset) (Asset, error)

	// Get returns the asset with the given ID or ErrAssetNotFound{ID} when no
	// row matches.
	Get(ctx context.Context, id string) (Asset, error)

	// ListByBrand returns assets for brandID ordered newest-uploaded first. An
	// empty brandID returns every asset across brands.
	ListByBrand(ctx context.Context, brandID string) ([]Asset, error)

	// Delete removes the asset with the given ID. Returns ErrAssetNotFound{ID}
	// when no row matched (the service translates that into an idempotent
	// success at the application layer).
	Delete(ctx context.Context, id string) error
}

// BrandResolver is the cross-domain seam Service uses to confirm the referenced
// brand exists before an upload. Implemented at the composition root
// (handlers/assets/module.go) over the brands repository, so the two internal
// domains never import each other.
type BrandResolver interface {
	// BrandExists reports whether a brand with brandID exists. A storage failure
	// surfaces as a non-nil error (never silently false).
	BrandExists(ctx context.Context, brandID string) (bool, error)
}
