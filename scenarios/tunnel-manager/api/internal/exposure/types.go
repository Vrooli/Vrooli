// Package exposure is the tiered exposure broker — the conceptual heart
// of Tunnel Manager. It turns "should this scenario be reachable, for how
// long?" into manifest + ingress + run state.
//
// CORE scenarios (the api-core/coreset closure) are always exposed; LEASED
// scenarios are exposed on demand with a TTL and reaped on expiry. This
// domain owns only the leases table and the tiering policy — it delegates
// the manifest to the routes domain, ingress to the config domain, and
// process lifecycle to a runner seam. All three are local interfaces
// (Manifest, Ingress, Runner, PortResolver) wired once at main.go, so this
// package never imports the config or lifecycle packages.
//
// Idempotency + failure-topography are first-class here (see service.go):
// Expose and Reconcile are safe to call repeatedly, a partial Expose leaves
// an orphan route that Reconcile reaps, and Revoke never retracts a route
// that is also CORE.
package exposure

import (
	"fmt"
	"time"
)

// DefaultTTL is the lease duration applied when a caller passes 0 (~1 week).
const DefaultTTL = 7 * 24 * time.Hour

// LeaseStatus is the lifecycle state of a lease. String-typed so it
// round-trips through SQLite; handlers translate to the proto enum.
type LeaseStatus string

const (
	LeaseActive  LeaseStatus = "active"
	LeaseExpired LeaseStatus = "expired"
	LeaseRevoked LeaseStatus = "revoked"
)

var leaseTransitions = map[LeaseStatus]map[LeaseStatus]struct{}{
	LeaseActive:  {LeaseActive: {}, LeaseExpired: {}, LeaseRevoked: {}},
	LeaseExpired: {LeaseExpired: {}, LeaseRevoked: {}},
	LeaseRevoked: {LeaseRevoked: {}},
}

// ErrInvalidLeaseTransition reports an illegal or unknown persisted lease
// status edge. Lease status is owned here so callers cannot revive a lease
// after reconciliation or operator revocation.
type ErrInvalidLeaseTransition struct {
	From LeaseStatus
	To   LeaseStatus
}

// ErrInvalidLeaseState reports corrupt or unsupported persisted status data.
type ErrInvalidLeaseState struct {
	Status LeaseStatus
}

func (e ErrInvalidLeaseState) Error() string {
	return fmt.Sprintf("invalid persisted lease status %q", e.Status)
}

func (e ErrInvalidLeaseTransition) Error() string {
	return fmt.Sprintf("invalid lease status transition %q -> %q", e.From, e.To)
}

// TransitionStatus advances a lease through its persisted lifecycle. Repeating
// the current state is allowed for metadata-only updates such as extension.
func (l *Lease) TransitionStatus(next LeaseStatus) error {
	if l == nil {
		from := LeaseStatus("")
		return ErrInvalidLeaseTransition{From: from, To: next}
	}
	allowed, known := leaseTransitions[l.Status]
	if !known || !validLeaseStatus(next) {
		return ErrInvalidLeaseTransition{From: l.Status, To: next}
	}
	if _, ok := allowed[next]; !ok {
		return ErrInvalidLeaseTransition{From: l.Status, To: next}
	}
	l.Status = next
	return nil
}

func validLeaseStatus(status LeaseStatus) bool {
	_, ok := leaseTransitions[status]
	return ok
}

// Tier values mirror the routes domain's tiers as plain strings for the
// reconciled exposure view.
const (
	TierCore   = "core"
	TierLeased = "leased"
)

// Lease records an on-demand exposure grant for a scenario.
type Lease struct {
	ID            string
	Scenario      string
	RequestedBy   string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	ExtendedCount int
	Status        LeaseStatus
}

// Exposure is the reconciled view of one scenario's exposure state.
type Exposure struct {
	Scenario  string
	Subdomain string
	PublicURL string
	LocalPort int
	Tier      string
	Enabled   bool
	// Lease is set when Tier == leased and an active lease exists.
	Lease *Lease
}

// ExposeInput is the explicit input to Service.Expose.
type ExposeInput struct {
	Scenario    string
	TTL         time.Duration
	RequestedBy string
}

// ErrInvalidExposure is the typed sentinel for bad input. Handlers map it
// to CodeInvalidArgument.
type ErrInvalidExposure struct {
	Field  string
	Reason string
}

func (e ErrInvalidExposure) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrLeaseNotFound is returned when no lease matches an id. Maps to
// CodeNotFound.
type ErrLeaseNotFound struct {
	ID string
}

func (e ErrLeaseNotFound) Error() string {
	return fmt.Sprintf("lease %q not found", e.ID)
}

// ErrPortUnresolved is returned when a scenario's fixed UI port cannot be
// determined (no service.json / ranged port). Maps to CodeFailedPrecondition.
type ErrPortUnresolved struct {
	Scenario string
	Reason   string
}

func (e ErrPortUnresolved) Error() string {
	return fmt.Sprintf("cannot resolve UI port for %q: %s", e.Scenario, e.Reason)
}
