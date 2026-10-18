package domain

import (
	"context"
	"time"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
)

// Standing grant states. A grant is authority only while approved; revoking
// flips the state so the very next lookup denies it. Expiry is derived from the
// grant's own ExpiresAt and the record's ExpiresAt, never a stored "expired"
// state.
const (
	StandingGrantApproved = "approved"
	StandingGrantRevoked  = "revoked"
)

// StandingGrant is an operator-approved, durable authority for one verified
// subject to perform a costly operation within declared bounds. It is the
// subject-keyed, non-session record behind the standing-grant model: a service
// principal (Phase 2) or a human session (Phase 5, via SessionID + a short
// ExpiresAt) presents identity only, and this record — not a token the caller
// holds — carries the authority. One OperationGrant shape and one evaluator
// (authn.AuthorizeStandingGrant) serve both.
type StandingGrant struct {
	ID string `json:"id"`
	// Subject is the verified principal subject this grant is keyed to (a service
	// principal name, or a human owner subject for a session grant).
	Subject string `json:"subject"`
	// SessionID scopes a grant to one session (Phase 5). Empty for a standing
	// service grant that is not session-bound.
	SessionID string `json:"session_id,omitempty"`
	// Grant is the authority itself, evaluated by OperationGrant.Allows. Its
	// Approver is the accountable operator and the owner created runs attribute to.
	Grant *requestidentity.OperationGrant `json:"grant"`
	// Limits bound how much the grant may be exercised. They live here, not in a
	// proto message or token claim, so an operator tightening a bound is a record
	// edit with no contract change.
	Limits authn.StandingGrantLimits `json:"limits"`
	State  string                    `json:"state"`
	// ApprovedBy is the operator identity that wrote (approved) the record.
	ApprovedBy string     `json:"approved_by"`
	ApprovedAt time.Time  `json:"approved_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// Active reports whether the record currently confers authority: approved, not
// revoked, and not past its record-level expiry at the given time. The grant's
// own ExpiresAt is additionally enforced by OperationGrant.Allows at evaluation,
// so an expired grant is refused even if this record has a later ExpiresAt.
func (g *StandingGrant) Active(now time.Time) bool {
	if g == nil || g.State != StandingGrantApproved || g.RevokedAt != nil {
		return false
	}
	return now.Before(g.ExpiresAt)
}

// StandingGrantRepository is durable, subject-keyed storage for standing grants.
// ActiveForSubject must return only records that are currently authority
// (approved, not revoked, not expired); callers never re-filter state.
type StandingGrantRepository interface {
	// Put inserts or replaces a grant by ID (approving is a Put; rotation
	// overwrites the prior record for the same ID).
	Put(ctx context.Context, g *StandingGrant) error
	// ActiveForSubject returns every currently-authoritative grant keyed to the
	// subject, newest first. Expired and revoked records are excluded.
	ActiveForSubject(ctx context.Context, subject string, now time.Time) ([]*StandingGrant, error)
	// Revoke flips a grant to revoked; it denies on the next lookup. Revoking an
	// unknown or already-revoked grant is a no-op that reports whether a row
	// transitioned.
	Revoke(ctx context.Context, id string, now time.Time) (bool, error)
	// List returns recorded grants newest first for the operator admin surface. It
	// includes revoked and expired records (each carries its own state/expiry) so
	// the operator can see the full standing-grant history, bounded by limit.
	List(ctx context.Context, limit int) ([]*StandingGrant, error)
	// ListForApprover returns recorded grants attributed to one verified approver,
	// newest first. The owner predicate is applied before the bounded limit.
	ListForApprover(ctx context.Context, approver string, limit int) ([]*StandingGrant, error)
}
