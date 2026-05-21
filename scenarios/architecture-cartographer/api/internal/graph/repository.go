package graph

import "context"

// Repository is the persistence seam for graph snapshots. Production
// wires the sqlite-backed implementation; service unit tests wire
// mocks.FakeRepository.
type Repository interface {
	// SaveSnapshot persists s. The implementation populates ID and
	// ExtractedAt when zero-valued.
	SaveSnapshot(ctx context.Context, s GraphSnapshot) (GraphSnapshot, error)

	// GetSnapshot returns the snapshot with the given ID or
	// ErrSnapshotNotFound{ID} when no row matches.
	GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error)

	// FindByHash returns the snapshot for (scenario, contentHash) or
	// ErrSnapshotNotFound when none is cached.
	FindByHash(ctx context.Context, scenario, contentHash string) (GraphSnapshot, error)

	// ListSnapshots paginates snapshots; scenario filter when set.
	ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error)

	// ClearSnapshots removes cached snapshots for a scenario. Returns
	// the count deleted.
	ClearSnapshots(ctx context.Context, scenario string) (int, error)
}

// ListSnapshotsFilter scopes ListSnapshots.
type ListSnapshotsFilter struct {
	Scenario  string
	PageSize  int
	PageToken string
}

// SnapshotPage is the paginated result.
type SnapshotPage struct {
	Snapshots     []GraphSnapshot
	NextPageToken string
}
