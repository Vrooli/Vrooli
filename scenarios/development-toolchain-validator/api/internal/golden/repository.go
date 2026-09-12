package golden

import "context"

// Repository is the persistence seam the golden service depends on.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire FakeRepository.
type Repository interface {
	// Create inserts g. The implementation populates ID, CreatedAt, and
	// LastRegeneratedAt when they are zero-valued. Returns
	// ErrGoldenAlreadyExists when slug is already taken.
	Create(ctx context.Context, g Golden) (Golden, error)

	// Get returns the golden whose Slug matches. Returns
	// ErrGoldenNotFound{Slug} when no row matches.
	Get(ctx context.Context, slug string) (Golden, error)

	// List returns all goldens ordered by slug ascending.
	List(ctx context.Context) ([]Golden, error)

	// Update applies in. Fields with their zero value (empty string)
	// mean "leave unchanged". Always bumps LastRegeneratedAt iff one of
	// the materially-tracked fields (path, version) changed; the service
	// layer owns that decision and passes a populated LastRegeneratedAt
	// in the persisted Golden record. Returns ErrGoldenNotFound when
	// slug does not exist.
	Update(ctx context.Context, g Golden) (Golden, error)

	// Delete removes the row with the given slug. Returns
	// ErrGoldenNotFound when slug does not exist.
	Delete(ctx context.Context, slug string) error
}
