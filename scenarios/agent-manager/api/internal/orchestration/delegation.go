// Responsibility: authenticate parent run tokens and mint attenuated child
// credentials only for an existing, explicitly linked child run.
package orchestration

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
)

// MintDelegatedIdentityRequest describes a child run that already exists. The
// parent token is supplied by the authenticated request header, never by the
// request body. The child must point back to the parent through ParentRunID.
type MintDelegatedIdentityRequest struct {
	ParentToken     string
	ChildRunID      uuid.UUID
	RequestedScopes []string
	ExpiresAt       time.Time
}

// MintDelegatedIdentityResponse is intentionally small: the bearer credential
// is returned only to the caller that presented the active parent token.
type MintDelegatedIdentityResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MintDelegatedIdentity verifies the active parent run, confirms the child
// lineage, and issues a one-way attenuated credential for that child.
func (o *Orchestrator) MintDelegatedIdentity(ctx context.Context, req MintDelegatedIdentityRequest) (*MintDelegatedIdentityResponse, error) {
	parentToken := strings.TrimSpace(req.ParentToken)
	verified, err := o.VerifyIdentityToken(ctx, parentToken)
	if err != nil {
		return nil, err
	}
	if verified == nil || !verified.Valid || verified.Claims == nil {
		return nil, domain.NewValidationErrorWithCode("identity", "parent identity is not active", domain.ErrCodePolicyScope)
	}
	if req.ChildRunID == uuid.Nil {
		return nil, domain.NewValidationError("childRunId", "field is required")
	}
	child, err := o.runs.Get(ctx, req.ChildRunID)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, domain.NewNotFoundError("Run", req.ChildRunID)
	}
	if child.ParentRunID == nil || *child.ParentRunID != verified.Claims.RunID {
		return nil, domain.NewValidationErrorWithCode("childRunId", "child run is not linked to the authenticated parent", domain.ErrCodePolicyScope)
	}

	now := o.now()
	claims, err := identity.Attenuate(verified.Claims, child.ID, child.TaskID, req.RequestedScopes, req.ExpiresAt, now)
	if err != nil {
		return nil, domain.NewValidationErrorWithCode("delegation", err.Error(), domain.ErrCodePolicyScope)
	}
	token, err := identity.GenerateToken(claims, o.identitySecret)
	if err != nil {
		return nil, err
	}
	child.IdentityTokenHash = identity.HashToken(token)
	child.IdentityTokenRevokedAt = nil
	if err := o.runs.Update(ctx, child); err != nil {
		return nil, err
	}
	return &MintDelegatedIdentityResponse{Token: token, ExpiresAt: time.Unix(claims.ExpiresAt, 0)}, nil
}
