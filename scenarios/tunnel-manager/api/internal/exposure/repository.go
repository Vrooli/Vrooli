package exposure

import "context"

// Repository is the persistence seam for the leases table. Production
// wires the sqlite-backed implementation; service unit tests wire a fake.
type Repository interface {
	// Create persists a new lease, populating ID/CreatedAt when zero.
	Create(ctx context.Context, l Lease) (Lease, error)

	// Get returns the lease with the given ID or ErrLeaseNotFound.
	Get(ctx context.Context, id string) (Lease, error)

	// ActiveForScenario returns the active (non-expired, non-revoked)
	// lease for a scenario, or ErrLeaseNotFound when none exists.
	ActiveForScenario(ctx context.Context, scenario string) (Lease, error)

	// Update persists the supplied lease by ID.
	Update(ctx context.Context, l Lease) (Lease, error)

	// List returns leases ordered by created_at desc, optionally filtered
	// by status (empty status = all).
	List(ctx context.Context, status LeaseStatus) ([]Lease, error)
}
