package deps

import "context"

// Repository is the persistence seam the deps service depends on.
// Production wires sqlite.go; tests wire mocks.FakeRepository.
type Repository interface {
	// SyncForComponent replaces the declaration set for a component
	// atomically. An empty Declarations slice clears all rows for that
	// component (covers "header dropped @deps" without a separate verb).
	SyncForComponent(ctx context.Context, in SyncInput) error

	// ListForComponent returns the declarations for one component,
	// ordered by dep_name. Empty slice + nil error when none exist.
	ListForComponent(ctx context.Context, componentID string) ([]Declaration, error)

	// DeleteForComponent removes every declaration for a component.
	// Used when the indexer drops a component from the registry.
	DeleteForComponent(ctx context.Context, componentID string) error
}
