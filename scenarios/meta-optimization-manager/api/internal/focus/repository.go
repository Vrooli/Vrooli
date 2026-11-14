package focus

import "context"

// Repository is the owned-state seam for the gaps registry: the persistent side
// of focus (notes/approaches the team appends, plus registry-only global gaps).
// Production wires the SQLite implementation; tests use a fake or in-memory map.
// A nil Repository disables persistence — the service then surfaces only the
// live-derived gaps and AddGapNote returns an error (nothing to persist into).
type Repository interface {
	// List returns every registry row (derived gaps are joined in by the
	// service, not stored here).
	List(ctx context.Context) ([]Gap, error)
	// Get returns one registry row by id; ok=false when absent.
	Get(ctx context.Context, id string) (Gap, bool, error)
	// Upsert inserts or replaces a registry row keyed by id.
	Upsert(ctx context.Context, g Gap) error
}
