package skill_catalog

import "context"

// Repository is the persistence seam for the skill_catalog mirror.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests substitute fakes from mocks/.
//
// seam: Repository
type Repository interface {
	// Upsert inserts the skill if the id is new, or updates version /
	// content_hash / synced_at if it already exists. Returns true if a
	// row was inserted, false if an existing row was updated unchanged
	// or in place.
	Upsert(ctx context.Context, s Skill) (inserted bool, changed bool, err error)

	// Get returns the skill with the given id. Returns ErrSkillNotFound
	// when no row matches.
	Get(ctx context.Context, id string) (Skill, error)

	// List returns every skill in the mirror ordered by id ascending.
	List(ctx context.Context) ([]Skill, error)

	// DeleteMissing removes every row whose id is NOT in keep. Returns
	// the number of rows removed. Used by Sync to reconcile deletions
	// upstream.
	DeleteMissing(ctx context.Context, keep []string) (int, error)
}
