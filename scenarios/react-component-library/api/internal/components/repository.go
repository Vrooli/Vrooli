package components

import "context"

// Repository is the persistence seam the components service depends
// on. Production wires sqlite.go; tests wire mocks.FakeRepository.
type Repository interface {
	// Upsert inserts or updates by LibraryID. Populates ID (if new),
	// IndexedAt, and UpdatedAt. Returns the persisted Component with
	// those fields filled in.
	Upsert(ctx context.Context, in UpsertInput) (Component, error)

	// UpsertManifest inserts or updates a component manifest and
	// replaces that component's indexed version rows with the provided
	// validated versions.
	UpsertManifest(ctx context.Context, in IndexManifestInput) (Component, error)

	// Get fetches a component by primary ID. Returns
	// ErrComponentNotFound when no row matches.
	Get(ctx context.Context, id string) (Component, error)

	// GetByLibraryID fetches by the disk-declared @libraryId. Returns
	// ErrComponentNotFound when no row matches.
	GetByLibraryID(ctx context.Context, libraryID string) (Component, error)

	// List returns components matching q, ordered newest-indexed
	// first. q.Limit <= 0 uses the service default.
	List(ctx context.Context, q SearchQuery) ([]Component, error)

	ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error)

	GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error)
	SetVersionPresence(ctx context.Context, componentID, version, presence string) error

	ListStories(ctx context.Context, q StoryQuery) ([]ComponentStory, error)

	// RestoreEvictedStories rebuilds missing typed story projections from the
	// durable story.json mirrors. It is safe to run before a full reindex and
	// returns the number of projections restored.
	RestoreEvictedStories(ctx context.Context) (int, error)

	// DeleteMissing removes registry rows whose LibraryID is not in
	// keep, cascading to that component's child rows (versions, files,
	// parity, stories, headers, design affinities) since the soft-FK
	// model has no ON DELETE CASCADE. Used by the indexer at the end of
	// a full walk so deleted files leave the registry without stranding
	// child rows. Returns the number of registry rows deleted.
	DeleteMissing(ctx context.Context, keep []string) (int, error)

	// SweepOrphans deletes every component-scoped row whose component_id
	// has no owning row in the components registry (soft-FK cruft left
	// by a re-slug or a withdrawn component before DeleteMissing began
	// cascading). Returns the component_versions rows it removed so the
	// indexer can emit a registry-orphan conformance finding per row.
	// The index is rebuildable from Git, so the sweep is safe to run on
	// every reindex.
	SweepOrphans(ctx context.Context) ([]OrphanVersion, error)
}
