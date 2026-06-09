package findings

import (
	"context"
	"time"
)

// Repository is the persistence seam. Every mutating method appends a
// finding_audit row in the same call so the trail can never drift from the
// data. actor identifies who performed the mutation (e.g. "operator", "agent").
type Repository interface {
	// Add inserts a finding plus its citations and a "create" audit row.
	Add(ctx context.Context, in NewFinding, actor string) (Finding, error)
	// Get returns the finding (with citations) or ErrFindingNotFound{id}.
	Get(ctx context.Context, id string) (Finding, error)
	// GetMany returns the findings (with citations) for the given ids. Missing
	// ids are silently skipped; the result is keyed by id.
	GetMany(ctx context.Context, ids []string) (map[string]Finding, error)
	// List returns findings matching the filter, newest-first.
	List(ctx context.Context, f ListFilter) ([]Finding, error)
	// Edit overwrites claim/confidence and appends an "edit" audit row.
	Edit(ctx context.Context, id string, in EditInput, actor string) (Finding, error)
	// Supersede soft-retires id (status=superseded, superseded_by=replacement)
	// and appends a "supersede" audit row. Never deletes.
	Supersede(ctx context.Context, id, replacement, reason, actor string) (Finding, error)
	// Flag moves id to DISPUTED with a dispute note and appends a "flag" row.
	Flag(ctx context.Context, id, reason, actor string) (Finding, error)
	// Resolve returns a DISPUTED finding to ACTIVE, clears its dispute note, and
	// appends a "resolve" audit row. The "keep" dispute resolution.
	Resolve(ctx context.Context, id, reason, actor string) (Finding, error)
	// Prune is the one path that may hard-delete: it removes superseded
	// findings (citations cascade) and appends a "prune" audit row per id. When
	// dryRun is set it reports the ids without deleting.
	Prune(ctx context.Context, dryRun bool, actor string) ([]string, error)
	// Count returns findings created in the half-open range [from, to).
	Count(ctx context.Context, from, to time.Time) (int, error)
	// LoadIndexable returns the findings eligible for the semantic index
	// (active + disputed; superseded are excluded so they drop out of qdrant
	// on the next reconcile).
	LoadIndexable(ctx context.Context) ([]Finding, error)
	// SearchArchivedLike is the SQL fallback used to surface superseded
	// findings when an include-archived search is requested (they are not in
	// the semantic index).
	SearchArchivedLike(ctx context.Context, query string, limit int) ([]Finding, error)
}
