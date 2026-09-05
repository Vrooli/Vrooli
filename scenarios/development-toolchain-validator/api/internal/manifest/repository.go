package manifest

import (
	"context"
	"time"
)

// Repository is the persistence seam for manifest storage. Production
// wires the sqlite-backed implementation from sqlite.go; service unit
// tests substitute fakes from mocks/.
//
// seam: Repository
type Repository interface {
	// Upsert stores or replaces the manifest for (skill_id, golden_slug).
	// Returns the post-write row (with UpdatedAt stamped).
	Upsert(ctx context.Context, m Manifest) (Manifest, error)

	// Get returns the manifest for (skill_id, golden_slug). Returns
	// ErrManifestNotFound when no row matches.
	Get(ctx context.Context, skillID, goldenSlug string) (Manifest, error)

	// List returns every manifest ordered by (skill_id, golden_slug).
	List(ctx context.Context) ([]Manifest, error)

	// ClearStaleOverride records (or refreshes) the manual-clear
	// timestamp for (skill_id, golden_slug). Used by the staleness
	// derivation to suppress drift reports until the next upsert.
	ClearStaleOverride(ctx context.Context, skillID, goldenSlug string, at time.Time) error

	// GetStaleOverride returns the recorded cleared_at timestamp for
	// (skill_id, golden_slug) or the zero time if none exists.
	GetStaleOverride(ctx context.Context, skillID, goldenSlug string) (time.Time, error)
}
