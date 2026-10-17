package operatordecision

import (
	"context"
	"time"
)

// Pending ask states. An ask is outstanding only while pending; approving or
// rejecting it is terminal. Expiry is derived from ExpiresAt, never a stored
// "expired" state. They match agent-manager's domain.PendingAuthorization*
// states so a scenario's SQL rows port without rewriting values.
const (
	PendingStatePending  = "pending"
	PendingStateApproved = "approved"
	PendingStateRejected = "rejected"
)

// PendingAsk is the domain-neutral durable record of a raised decision. It
// generalizes domain.PendingAuthorizationRequest: the fields a scenario needs to
// make a raise idempotent (ID), cap it per subject (Subject + State + ExpiresAt),
// and resolve it later (ResolvedAt/By). Scenario-specific payload the resolver
// needs on approval (grant kind, bounds, TTL, …) lives in Extra, so the neutral
// record need not grow a field per scenario.
//
// The SQL implementation stays per-scenario (behind PendingStore); this package
// never touches a database.
type PendingAsk struct {
	// ID is the stable correlation key (see CorrelationID). Reusing it as the
	// primary key is what makes Raise idempotent.
	ID string
	// Subject is the entity the ask is about; the per-subject flood cap counts
	// outstanding asks for it.
	Subject string
	State   string
	// RaisedAt is the deterministic raise time (from the Raiser's Clock).
	RaisedAt time.Time
	// ExpiresAt bounds how long the ask stays outstanding before it ages out of the
	// anti-flood count and may be re-raised.
	ExpiresAt time.Time
	// ResolvedAt / ResolvedBy record the terminal decision; a scenario's resolver
	// sets them when it applies a DecisionResolution.
	ResolvedAt *time.Time
	ResolvedBy string
	// Extra carries scenario-specific fields (grant kind, operation, target, TTL,
	// bounds, session) the resolver reads on approval. Opaque to this package.
	Extra map[string]string
}

// Outstanding reports whether the ask is still an un-decided, un-expired ask (so
// it counts toward the per-subject flood cap and dedupes a raise). It mirrors
// domain.PendingAuthorizationRequest.Outstanding.
func (p *PendingAsk) Outstanding(now time.Time) bool {
	if p == nil || p.State != PendingStatePending {
		return false
	}
	return now.Before(p.ExpiresAt)
}

// PendingStore is durable storage for raised asks, keyed by correlation id. A
// scenario implements it over its own table (the Raiser needs nothing more). It
// generalizes domain.PendingAuthorizationRepository.
type PendingStore interface {
	// Get returns the ask for a correlation id, or nil when none exists.
	Get(ctx context.Context, id string) (*PendingAsk, error)
	// Put inserts or replaces an ask by id (the raise and the resolution both write
	// through here).
	Put(ctx context.Context, ask *PendingAsk) error
	// CountOutstandingForSubject returns how many un-decided, un-expired asks the
	// subject currently has, for the anti-flood cap.
	CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error)
}
