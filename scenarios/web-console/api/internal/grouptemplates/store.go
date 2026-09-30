package grouptemplates

import "context"

// Store abstracts group-template persistence. MemStore is in-memory (tests),
// SQLStore is SQLite-backed (production).
type Store interface {
	// List returns every template ordered by name, then id.
	List(ctx context.Context) ([]Template, error)
	// Upsert creates or updates a template. Returns ErrInvalidTemplate when
	// the request fails Validate.
	Upsert(ctx context.Context, req UpsertRequest) (Template, error)
	// Delete removes a template. Idempotent; returns true if a row went away.
	// Every template is deletable, including a seeded example.
	Delete(ctx context.Context, id string) (bool, error)
}
