package notes

import "context"

// Repository is the persistence seam the notes service depends on.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire mocks.FakeRepository (from internal/notes/mocks).
// New methods
// land here when the service proves it needs them — keep the surface
// narrow.
type Repository interface {
	// Create persists n. The implementation populates ID, CreatedAt,
	// and UpdatedAt; callers leaving those zero-valued is the canonical
	// shape. Returns the persisted Note (with the populated fields).
	Create(ctx context.Context, n Note) (Note, error)

	// Get returns the note with the given ID or ErrNoteNotFound{ID}
	// when no row matches. Wrapped error types other than
	// ErrNoteNotFound mean an underlying storage failure.
	Get(ctx context.Context, id string) (Note, error)

	// List returns up to `limit` notes ordered newest-first by
	// CreatedAt. limit <= 0 returns no rows; callers requesting "all"
	// pass an explicit upper bound.
	List(ctx context.Context, limit int) ([]Note, error)
}
