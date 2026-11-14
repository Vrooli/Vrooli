package versions

import "context"

// Repository is the persistence seam for versions. Production wires
// sqlite.go; tests wire mocks.FakeRepository.
type Repository interface {
	// Insert appends a new version row, generating ID + RecordedAt.
	Insert(ctx context.Context, v Version) (Version, error)

	// Latest returns the most recently recorded row for componentID,
	// or (Version{}, nil) when none exist. Callers distinguish empty
	// from error rather than via a sentinel.
	Latest(ctx context.Context, componentID string) (Version, error)

	// List returns rows newest-first, capped at q.Limit.
	List(ctx context.Context, q ListQuery) ([]Version, error)

	// Get returns the row for (componentID, version). Returns
	// ErrVersionNotFound when no such row exists.
	Get(ctx context.Context, componentID, version string) (Version, error)
}
