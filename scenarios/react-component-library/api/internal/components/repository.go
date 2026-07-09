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

	ListExamples(ctx context.Context, q ExampleQuery) ([]ComponentExample, error)

	// DeleteMissing removes registry rows whose LibraryID is not in
	// keep. Used by the indexer at the end of a full walk so deleted
	// files leave the registry. Returns the number of rows deleted.
	DeleteMissing(ctx context.Context, keep []string) (int, error)
}
