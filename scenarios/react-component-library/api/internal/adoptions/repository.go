package adoptions

import "context"

// Repository is the persistence seam the adoptions service depends on.
// Production wires sqlite.go; tests wire mocks.FakeRepository.
type Repository interface {
	// Create inserts a new adoption row. The repository generates an
	// ID only when the caller does not provide one.
	Create(ctx context.Context, in CreateInput) (Adoption, error)

	// UpdateAppliedSnapshot persists the library version/hash snapshot
	// written by a reapply without changing ownership metadata.
	UpdateAppliedSnapshot(ctx context.Context, in AppliedSnapshotUpdate) (Adoption, error)
	UpdateAppliedUnit(ctx context.Context, in AppliedUnitUpdate) (Adoption, error)

	// Rebaseline overwrites only the recorded pristine snapshot hashes (parent +
	// per-file) of an existing row, leaving version / status / applied_at /
	// drift_backlog_ref untouched. It is the heal seam for poisoned snapshots;
	// callers recompute drift status separately via ApplyRefresh.
	Rebaseline(ctx context.Context, in RebaselineInput) (Adoption, error)

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
