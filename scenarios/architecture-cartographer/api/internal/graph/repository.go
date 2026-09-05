package graph

import (
	"context"
	"time"
)

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

	// LatestSnapshotMeta returns the latest snapshot metadata for a scenario
	// without decoding the payload.
	LatestSnapshotMeta(ctx context.Context, scenario string) (GraphSnapshotMeta, error)

	// FindByHash returns the snapshot for (scenario, contentHash) or
	// ErrSnapshotNotFound when none is cached.
	FindByHash(ctx context.Context, scenario, contentHash string) (GraphSnapshot, error)

	// FindBySourceFingerprint returns the latest snapshot extracted from
	// the same source tree/options fingerprint.
	FindBySourceFingerprint(ctx context.Context, scenario, sourceFingerprint string) (GraphSnapshot, error)

	// ListSnapshots paginates snapshots; scenario filter when set.
	ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error)

	// ClearSnapshots removes cached snapshots for a scenario. Returns
	// the count deleted.
	ClearSnapshots(ctx context.Context, scenario string) (int, error)

	// PruneSnapshots enforces a retention policy, keeping the newest N
	// snapshots per scenario and returning freed pages to the filesystem.
	PruneSnapshots(ctx context.Context, policy RetentionPolicy) (RetentionResult, error)

	// SnapshotPayloadBytes reports the total live payload across all snapshots.
	// This is the measurement a declared storage budget is judged against.
	SnapshotPayloadBytes(ctx context.Context) (int64, error)

	// ReclaimableSnapshotBytes reports the payload bytes and row count beyond
	// the retention floor without deleting anything.
	ReclaimableSnapshotBytes(ctx context.Context, policy RetentionPolicy) (int64, int, error)

	// SnapshotCounts reports how many snapshots each scenario holds.
	SnapshotCounts(ctx context.Context) (map[string]int, error)
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

// GraphSnapshotMeta is the payload-free metadata view of a snapshot.
type GraphSnapshotMeta struct {
	ID                string
	Scenario          string
	ContentHash       string
	SourceFingerprint string
	ExtractedAt       time.Time
	ExtractionMS      int64
	PayloadBytes      int64
}
