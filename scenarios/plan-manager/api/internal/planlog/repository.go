package planlog

import "context"

// Repository is the persistence seam for the log ledger store. Production wires
// the SQLite implementation (sqlite.go) over the ~/.vrooli home store; service
// unit tests substitute a fake or a real sqlite repo. Entries are queried across
// one another (by plan/execution/phase/type/triage/sync state), so the surface
// is per-record plus a filtered list.
type Repository interface {
	// SaveEntry upserts a ledger entry keyed by id.
	SaveEntry(ctx context.Context, e Entry) error
	// GetEntry returns the entry matching id; ok=false when absent.
	GetEntry(ctx context.Context, id string) (Entry, bool, error)
	// ListEntries returns entries matching the filter, oldest-first.
	ListEntries(ctx context.Context, f Filter) ([]Entry, error)
	// FindByIdempotencyKey returns the entry previously created with the given
	// idempotency key within the same plan scope; ok=false when none exists.
	FindByIdempotencyKey(ctx context.Context, planID, key string) (Entry, bool, error)
}
