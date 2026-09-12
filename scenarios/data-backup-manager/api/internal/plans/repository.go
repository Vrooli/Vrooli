package plans

import "context"

// Repository is the persistence seam the plans service depends on. Production
// wires the sqlite-backed implementation from sqlite.go; service unit tests
// wire mocks.FakeRepository (from internal/plans/mocks). The surface is narrow
// — the service composes CRUD from these primitives; the repository owns no
// policy.
//
// seam: Repository persists Plan rows with membership tables plan_targets and
// plan_destinations. Production wires SqliteRepository (sqlite.go); tests wire
// FakeRepository (mocks/).
type Repository interface {
	// Create persists a new plan including its membership lists. The
	// implementation populates ID, CreatedAt, and UpdatedAt. Returns the
	// persisted Plan (with TargetIDs and DestinationIDs populated).
	Create(ctx context.Context, p Plan) (Plan, error)

	// Update replaces the plan fields and membership lists identified by p.ID.
	// Bumps UpdatedAt. Returns the persisted Plan.
	Update(ctx context.Context, p Plan) (Plan, error)

	// GetByID returns the plan with the given id (including membership lists) or
	// ErrPlanNotFound when none matches.
	GetByID(ctx context.Context, id string) (Plan, error)

	// List returns up to limit plans ordered by name. limit <= 0 returns no
	// rows.
	List(ctx context.Context, limit int) ([]Plan, error)

	// Delete removes the plan and its membership rows by id. Returns true when a
	// row was removed, false when none matched.
	Delete(ctx context.Context, id string) (bool, error)
}
