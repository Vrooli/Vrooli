package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

// AuthorizationGrant preserves consent separately from original run admission.
// ClaimsJSON and TokenHash never contain a plaintext bearer credential.
type AuthorizationGrant struct {
	ID      uuid.UUID `json:"id"`
	RunID   uuid.UUID `json:"run_id"`
	State   string    `json:"state"`
	Policy  string    `json:"policy"`
	Targets []string  `json:"targets"`
	// Rules holds each reviewed policy with its own targets when one consent
	// covers several. Empty means the single Policy/Targets pair.
	Rules           []AuthorizationRule    `json:"rules,omitempty"`
	DurationSeconds int64                  `json:"duration_seconds"`
	Binding         cliutil.ProcessBinding `json:"binding"`
	HarnessKind     string                 `json:"harness_kind"`
	HarnessSession  string                 `json:"harness_session"`
	// RequesterContext is private review material, never public grant metadata.
	RequesterContext       *AuthorizationRequesterContext  `json:"-"`
	CreatedAt              time.Time                       `json:"created_at"`
	RequestExpiresAt       time.Time                       `json:"request_expires_at"`
	Grant                  *requestidentity.OperationGrant `json:"grant,omitempty"`
	ApprovedAt             *time.Time                      `json:"approved_at,omitempty"`
	ApproverSource         requestidentity.AuthSource      `json:"approver_source,omitempty"`
	ApproverIssuer         string                          `json:"approver_issuer,omitempty"`
	ApproverAudience       string                          `json:"approver_audience,omitempty"`
	ApproverProofExpiresAt time.Time                       `json:"approver_proof_expires_at,omitempty"`
	RevokedAt              *time.Time                      `json:"revoked_at,omitempty"`
	RevokedBy              string                          `json:"revoked_by,omitempty"`
	SupersededBy           *uuid.UUID                      `json:"superseded_by,omitempty"`
	ClaimsJSON             string                          `json:"-"`
	TokenHash              string                          `json:"-"`
	// NotificationSnapshot freezes the operator FYI event for this pending
	// generation. It is persisted in the repository wrapper but omitted from
	// public grant receipts.
	NotificationSnapshot    *AuthorizationNotificationSnapshot `json:"-"`
	NotificationPublishedAt *time.Time                         `json:"-"`
	NotificationUnreached   string                             `json:"-"`
}

// AuthorizationRequesterContext is the optional requester-authored explanation
// shown only on the native consent page.
type AuthorizationRequesterContext struct {
	BigPicture       string `json:"big_picture,omitempty"`
	TechnicalDetails string `json:"technical_details,omitempty"`
}

// AuthorizationNotificationSnapshot is the immutable event payload and
// identity prepared for one pending consent request. It is delivery metadata,
// not authority; publication can only point a human at the native consent page.
type AuthorizationNotificationSnapshot struct {
	Source    string         `json:"source"`
	EventType string         `json:"event_type"`
	EventID   string         `json:"event_id"`
	Occurred  time.Time      `json:"occurred"`
	Payload   map[string]any `json:"payload"`
}

// AuthorizationRule is one reviewed policy and the targets it may reach.
type AuthorizationRule struct {
	Policy  string   `json:"policy"`
	Targets []string `json:"targets"`
}

// Authorization request states. Expired is derived, never stored: a pending
// request past its review deadline reads as expired.
const (
	AuthorizationPending    = "pending"
	AuthorizationApproving  = "approving"
	AuthorizationApproved   = "approved"
	AuthorizationRevoked    = "revoked"
	AuthorizationSuperseded = "superseded"
	AuthorizationExpired    = "expired"
)

// PolicyRules returns the grant's rules, or its single legacy policy pair.
func (g *AuthorizationGrant) PolicyRules() []AuthorizationRule {
	if len(g.Rules) > 0 {
		return g.Rules
	}
	return []AuthorizationRule{{Policy: g.Policy, Targets: g.Targets}}
}

// EffectiveState reports a pending request past its review deadline as
// expired, so readers never see an undecidable request as pending.
func (g *AuthorizationGrant) EffectiveState(now time.Time) string {
	if g.State == AuthorizationPending && !now.Before(g.RequestExpiresAt) {
		return AuthorizationExpired
	}
	// Approved consent lapses with its grant; the stored state stays approved.
	if g.State == AuthorizationApproved && g.Grant != nil && !now.Before(g.Grant.ExpiresAt) {
		return AuthorizationExpired
	}
	return g.State
}

type AuthorizationRepository interface {
	Create(context.Context, *AuthorizationGrant) error
	Get(context.Context, uuid.UUID) (*AuthorizationGrant, error)
	List(context.Context, int) ([]*AuthorizationGrant, error)
	Replace(context.Context, *AuthorizationGrant, string) (bool, error)
	UpdateNotification(context.Context, *AuthorizationGrant, *AuthorizationGrant) (bool, error)
}

// AuthorizationNotificationRetryRepository is the optional durable scan used
// by the background notification reconciler. Keeping it separate preserves the
// core authority repository contract for in-memory/test implementations.
type AuthorizationNotificationRetryRepository interface {
	ListNotificationRetryScanPage(context.Context, string, int) ([]*AuthorizationGrant, error)
}

// AuthorizationError carries bounded recovery metadata without credentials.
type AuthorizationError struct{ Code string }

func (e *AuthorizationError) Error() string { return "agent authorization " + e.Code }
