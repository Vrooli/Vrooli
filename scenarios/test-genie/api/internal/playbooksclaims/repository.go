package playbooksclaims

import (
	"context"
	"time"
)

// Repository is the persistence seam for playbooks claims.
//
// seam: Repository
//
// Atomic-acquire semantics: TryAcquire inserts a new row if absent, or
// steals an expired one in a single statement. On conflict with a live
// claim it returns *ErrBusy carrying the active holder.
type Repository interface {
	// TryAcquire attempts to claim the scenario for the caller. ttl is added
	// to now to set expires_at. Returns the granted claim on success.
	TryAcquire(ctx context.Context, in AcquireInput, now time.Time, ttl time.Duration) (Claim, error)

	// Heartbeat extends expires_at on a claim the caller still owns.
	// Returns ErrLeaseMismatch if scenarioName is held by a different runID,
	// ErrNotFound if the row is already gone.
	Heartbeat(ctx context.Context, scenarioName, runID string, now time.Time, ttl time.Duration) (Claim, error)

	// Release deletes the claim iff runID matches.
	// ErrLeaseMismatch on owner mismatch, ErrNotFound if absent.
	Release(ctx context.Context, scenarioName, runID string) error

	// Get fetches the current claim for the scenario, or ErrNotFound.
	Get(ctx context.Context, scenarioName string) (Claim, error)

	// List returns all active rows (no TTL filtering — caller decides).
	List(ctx context.Context) ([]Claim, error)

	// ForceBreak deletes the claim regardless of owner. Returns the broken
	// claim on success (or ErrNotFound).
	ForceBreak(ctx context.Context, scenarioName string) (Claim, error)
}
