package adoptions

import "context"

// Repository is the persistence seam the adoptions service depends on.
// Production wires sqlite.go; tests wire mocks.FakeRepository.
type Repository interface {
	// Create inserts a new adoption row, generating ID + CreatedAt.
	// The returned Adoption has Status==StatusEmpty until Refresh runs.
	Create(ctx context.Context, in CreateInput) (Adoption, error)

	// Get fetches an adoption by primary ID. Returns
	// ErrAdoptionNotFound when no row matches.
	Get(ctx context.Context, id string) (Adoption, error)

	// List returns adoptions matching q, ordered newest-created first.
	// q.Limit <= 0 uses the service default.
	List(ctx context.Context, q ListQuery) ([]Adoption, error)

	// Delete removes the adoption by ID. Returns ErrAdoptionNotFound
	// when no row matches.
	Delete(ctx context.Context, id string) error

	// ApplyRefresh writes the per-row updates produced by Refresh.
	// Repository merges only the status / status_detail / refreshed_at
	// fields; everything else stays. Returns the count of rows
	// actually touched.
	ApplyRefresh(ctx context.Context, updates []RefreshUpdate) (int, error)
}
