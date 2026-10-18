package domain

import (
	"context"
	"time"
)

// Pending authorization request states. A request is outstanding only while
// pending; approving or rejecting it is terminal. Expiry is derived from
// ExpiresAt, never a stored "expired" state.
const (
	PendingAuthorizationPending  = "pending"
	PendingAuthorizationApproved = "approved"
	PendingAuthorizationRejected = "rejected"
)

// PendingAuthorizationKind distinguishes what a resolution should mint on
// approval: a durable subject-keyed standing grant (recurring, e.g. a heartbeat
// subject) or a short-TTL one-time grant (an ad-hoc in-session request).
const (
	PendingAuthorizationKindStanding = "standing"
	PendingAuthorizationKindOneTime  = "one-time"
)

// PendingAuthorizationRequest is the durable record of a raised "ask, don't
// block" authorization. It exists so (1) a repeated raise for the same
// (subject, operation, target) is idempotent — one operator notification, not a
// flood on every heartbeat retry; (2) the number of distinct outstanding asks
// per subject can be capped against a buggy or hostile caller; and (3) when the
// operator's notification-hub resolution arrives, the resolver knows exactly
// what to mint (kind, subject, bounds, expiry) and attributes it to the
// approving human. It holds no authority itself — only the description of the
// authority an approval would create.
type PendingAuthorizationRequest struct {
	// ID is the stable correlation key (a hash of subject+operation+target).
	// Reusing it as the primary key is what makes the raise idempotent.
	ID string `json:"id"`
	// Subject is the verified caller the ask is about (e.g. a service principal).
	Subject   string `json:"subject"`
	Operation string `json:"operation"`
	Target    string `json:"target"`
	// Kind selects what an approval mints (standing vs one-time).
	Kind string `json:"kind"`
	// SessionID scopes a one-time grant to a session when present.
	SessionID string `json:"session_id,omitempty"`
	// GrantTTL is the lifetime the issued grant will carry on approval.
	GrantTTL time.Duration `json:"grant_ttl"`
	// MaxConcurrentRuns is the concurrency bound the issued grant will carry.
	MaxConcurrentRuns int       `json:"max_concurrent_runs,omitempty"`
	State             string    `json:"state"`
	RaisedAt          time.Time `json:"raised_at"`
	// ExpiresAt bounds how long this ask stays outstanding before it ages out of
	// the anti-flood count and can be re-raised.
	ExpiresAt time.Time `json:"expires_at"`
	// EventID binds a hub answer to the exact request generation that was shown.
	EventID string `json:"event_id,omitempty"`
	// PublishedAt is set only after the request event is accepted by the publisher.
	PublishedAt *time.Time `json:"published_at,omitempty"`
	// ResolvedAt / ResolvedBy / GrantID record the terminal decision.
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
	GrantID    string     `json:"grant_id,omitempty"`
}

// Outstanding reports whether the request is still an un-decided, un-expired
// ask (so it counts toward the per-subject anti-flood cap and dedupes a raise).
func (p *PendingAuthorizationRequest) Outstanding(now time.Time) bool {
	if p == nil || p.State != PendingAuthorizationPending {
		return false
	}
	return now.Before(p.ExpiresAt)
}

// PendingAuthorizationRepository is durable storage for raised authorization
// asks, keyed by the correlation id.
type PendingAuthorizationRepository interface {
	// Get returns the request for a correlation id, or nil when none exists.
	Get(ctx context.Context, id string) (*PendingAuthorizationRequest, error)
	// Put inserts or replaces a request by id (the raise and the resolution both
	// write through here).
	Put(ctx context.Context, req *PendingAuthorizationRequest) error
	// CountOutstandingForSubject returns how many un-decided, un-expired asks the
	// subject currently has, for the anti-flood cap.
	CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error)
}
