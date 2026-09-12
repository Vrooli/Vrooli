package assignments

import "context"

// Repository is the persistence seam for Assignment entities. Production wires
// the sqlite-backed implementation from sqlite.go; service unit tests wire
// mocks.FakeRepository. A scenario carries at most one assignment, so the store
// is keyed by scenario_name and Upsert replaces in place.
type Repository interface {
	// Upsert persists a (re-)assignment keyed by scenario_name. The
	// implementation populates ID (when zero) and AppliedAt. Returns the
	// persisted Assignment.
	Upsert(ctx context.Context, a Assignment) (Assignment, error)

	// GetByScenario returns the assignment for scenarioName or
	// ErrAssignmentNotFound{Scenario} when none exists.
	GetByScenario(ctx context.Context, scenarioName string) (Assignment, error)

	// ListByBrand returns assignments ordered newest-applied first. An empty
	// brandID returns every assignment.
	ListByBrand(ctx context.Context, brandID string) ([]Assignment, error)

	// DeleteByScenario removes a scenario's assignment. Returns
	// ErrAssignmentNotFound{Scenario} when no row matched (the service
	// translates that into an idempotent success at the application layer).
	DeleteByScenario(ctx context.Context, scenarioName string) error
}

// BrandResolver is the narrow seam the assignment service uses to pin a brand's
// current version at assignment time. Defined at the consumer (seam-discovery)
// so internal/assignments never imports internal/brands; the composition root
// (handlers/assignments/module.go) wires the adapter.
type BrandResolver interface {
	// BrandVersion returns the current version of the brand and ok=true, or
	// ok=false when no such brand exists. A non-nil error signals a lookup
	// failure (not "absent") and is surfaced as an internal error.
	BrandVersion(ctx context.Context, brandID string) (version int, ok bool, err error)
}
