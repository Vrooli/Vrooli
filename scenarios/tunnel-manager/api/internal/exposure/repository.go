package exposure

import "context"

// ErrLeaseWriteConflict means the durable lease changed after the caller read
// the snapshot it is trying to save.
type ErrLeaseWriteConflict struct {
	ID             string
	ExpectedStatus LeaseStatus
	CurrentStatus  LeaseStatus
}

func (e ErrLeaseWriteConflict) Error() string {
	return "lease " + e.ID + " changed while saving (expected status " + string(e.ExpectedStatus) + ", current status " + string(e.CurrentStatus) + ")"
}

func leaseStatusTransitionAllowed(from, to LeaseStatus) bool {
	lease := Lease{Status: from}
	return lease.TransitionStatus(to) == nil
}

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

	// Save persists lease metadata and the status edge from the previously read
	// snapshot. The store compares all mutable fields so concurrent extensions
	// cannot overwrite each other's expiry even when status remains active.
	Save(ctx context.Context, l Lease, expected Lease) (Lease, error)

	// List returns leases ordered by created_at desc, optionally filtered
	// by status (empty status = all).
	List(ctx context.Context, status LeaseStatus) ([]Lease, error)
}
