package targets

import "context"

// Repository is the persistence seam the targets service depends on.
// Production wires the sqlite-backed implementation from sqlite.go; service
// unit tests wire mocks.FakeRepository (from internal/targets/mocks). The
// surface is narrow — the service composes idempotent registration from these
// primitives; the repository owns no policy.
//
// seam: Repository persists Target rows keyed by (owner, name). Production
// wires SqliteRepository (sqlite.go); tests wire FakeRepository (mocks/).
type Repository interface {
	// Create persists a new target. The implementation populates ID,
	// CreatedAt, and UpdatedAt. Returns the persisted Target.
	Create(ctx context.Context, t Target) (Target, error)

	// Update overwrites the spec (source kind, locator) of an existing target
	// identified by t.ID and bumps UpdatedAt. Returns the persisted Target.
	Update(ctx context.Context, t Target) (Target, error)

	// GetByOwnerName returns the target keyed by (owner, name) or
	// ErrTargetNotFound when none matches.
	GetByOwnerName(ctx context.Context, owner, name string) (Target, error)

	// GetByID returns the target with the given id or ErrTargetNotFound.
	GetByID(ctx context.Context, id string) (Target, error)

	// List returns up to `limit` targets ordered by (owner, name). When owner
	// is non-empty, only that owner's targets are returned. limit <= 0 returns
	// no rows.
	List(ctx context.Context, owner string, limit int) ([]Target, error)

	// DeleteByOwnerName removes the target keyed by (owner, name). Returns true
	// when a row was removed, false when none matched.
	DeleteByOwnerName(ctx context.Context, owner, name string) (bool, error)
}
