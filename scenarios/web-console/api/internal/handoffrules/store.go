package handoffrules

import "context"

// Store abstracts capture-rule persistence. MemStore is in-memory (tests),
// SQLStore is SQLite-backed (production).
type Store interface {
	// List returns every rule ordered by sort_order, then id.
	List(ctx context.Context) ([]Rule, error)
	// Upsert creates or updates a rule. Returns ErrInvalidRule when the
	// request fails Validate.
	Upsert(ctx context.Context, req UpsertRequest) (Rule, error)
	// Delete removes a rule. Idempotent; returns true if a row went away.
	// Every rule is deletable, including a seeded example.
	Delete(ctx context.Context, id string) (bool, error)
}
