// Responsibility: authenticate parent run tokens and mint attenuated child
// credentials only for an existing, explicitly linked child run.
package orchestration

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
)

// A model qualification receipt is not permission to change execution policy.
// Dependent children may retain or narrow the parent's restrictions. Widening or
// unprovable changes need independent owner admission, not child overrides. This
// compares declared policy only; runtime containment still needs qualification.
func dependentExecutionPermissionsError(parent, child *domain.RunConfig) error {
	field := ""
	switch {
	case !delegatedNetworkWithin(parent.NetworkAccess, child.NetworkAccess):
		field = "networkAccess"
	case child.SkipPermissionPrompt && !parent.SkipPermissionPrompt:
		field = "skipPermissionPrompt"
	case !parent.ToolRestrictionPolicy.IsValid() || !child.ToolRestrictionPolicy.IsValid() ||
		(parent.ToolRestrictionPolicy.Effective() == domain.ToolRestrictionPolicyEnforced && child.ToolRestrictionPolicy.Effective() != domain.ToolRestrictionPolicyEnforced) ||
		!delegatedAllowListWithin(parent.AllowedTools, child.AllowedTools) || !delegatedSetContains(child.DeniedTools, parent.DeniedTools):
		field = "tools"
	case !delegatedAllowListWithin(parent.AllowedPaths, child.AllowedPaths) || !delegatedSetContains(child.DeniedPaths, parent.DeniedPaths):
		field = "paths"
	case (parent.RequireEffectContainment && !child.RequireEffectContainment) || !delegatedSetContains(parent.AllowedEffects, child.AllowedEffects):
		field = "allowedEffects"
	case child.Features.EnableBrowser && !parent.Features.EnableBrowser:
		field = "features"
	case !maps.EqualFunc(parent.ExtraFlags, child.ExtraFlags, slices.Equal[[]string]):
		field = "extraFlags"
	case !delegatedSandboxWithin(parent.SandboxConfig, child.SandboxConfig):
		field = "sandboxConfig"
	}
	if field != "" {
		return domain.NewValidationErrorWithCode(field,
			"dependent child must retain or narrow parent execution restrictions; unproven policy changes require independent owner admission",
			domain.ErrCodePolicyScope)
	}
	return nil
}

func delegatedNetworkWithin(parent, child domain.NetworkAccess) bool {
	if !parent.IsValid() || !child.IsValid() {
		return false
	}
	return parent.Effective() == child.Effective() || child.Effective() == domain.NetworkAccessNone || parent.Effective() == domain.NetworkAccessFull
}

// Compare declared entries, not inferred glob-language containment. Empty
// allowlists mean unrestricted; empty effect grants and denylists mean no entries.
func delegatedAllowListWithin(parent, child []string) bool {
	return len(parent) == 0 || (len(child) > 0 && delegatedSetContains(parent, child))
}

func delegatedSetContains(superset, subset []string) bool {
	for _, value := range subset {
		if !slices.Contains(superset, value) {
			return false
		}
	}
	return true
}

func delegatedSandboxWithin(parent, child *domain.SandboxConfig) bool {
	if parent == nil || child == nil {
		return parent == child
	}
	return parent.Mode.IsValid() && child.Mode.IsValid() && child.Mode.AtLeast(parent.Mode) &&
		delegatedNetworkWithin(parent.NetworkMode, child.NetworkMode) &&
		(!parent.ManualReview || child.ManualReview) &&
		(parent.GetAutoApply() || !child.GetAutoApply()) &&
		(parent.GetApplyOnFailure() || !child.GetApplyOnFailure()) &&
		(parent.NoLock || !child.NoLock) &&
		(parent.WritePolicy == nil || (child.WritePolicy != nil && delegatedSetContains(parent.WritePolicy.Paths, child.WritePolicy.Paths))) &&
		// Wire round-trips may turn nil lists into empty lists. Compare every
		// control by value; representation differences are not wider authority.
		parent.Acceptance.Mode == child.Acceptance.Mode &&
		parent.Acceptance.IgnoreBinary == child.Acceptance.IgnoreBinary &&
		slices.Equal(parent.Acceptance.Allow.PathGlobs, child.Acceptance.Allow.PathGlobs) &&
		slices.Equal(parent.Acceptance.Allow.Extensions, child.Acceptance.Allow.Extensions) &&
		slices.Equal(parent.Acceptance.Deny.PathGlobs, child.Acceptance.Deny.PathGlobs) &&
		slices.Equal(parent.Acceptance.Deny.Extensions, child.Acceptance.Deny.Extensions) &&
		parent.Lifecycle.TTL == child.Lifecycle.TTL &&
		parent.Lifecycle.IdleTimeout == child.Lifecycle.IdleTimeout &&
		slices.Equal(parent.Lifecycle.CheckpointOn, child.Lifecycle.CheckpointOn) &&
		slices.Equal(parent.Lifecycle.StopOn, child.Lifecycle.StopOn) &&
		slices.Equal(parent.Lifecycle.DeleteOn, child.Lifecycle.DeleteOn)
}

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
	if !child.Status.IsActive() || child.IdentityTokenRevokedAt != nil {
		return nil, domain.NewValidationErrorWithCode("childRunId", "child identity is not active; delegation cannot resume or undo revocation", domain.ErrCodePolicyScope)
	}
	if child.OwnerSubject != verified.Claims.Subject {
		return nil, domain.NewValidationErrorWithCode("childRunId", "child and parent must have the same admitted owner", domain.ErrCodePolicyScope)
	}
	var profile *domain.AgentProfile
	if child.AgentProfileID != nil {
		if o.profiles == nil {
			return nil, domain.NewConfigMissingError("profiles", "child profile authority is unavailable", nil)
		}
		profile, err = o.profiles.Get(ctx, *child.AgentProfileID)
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, domain.NewNotFoundError("AgentProfile", *child.AgentProfileID)
		}
	}
	maxExpiry := time.Unix(verified.Claims.ExpiresAt, 0)
	if child.OwnerExpiresAt != nil && child.OwnerExpiresAt.Before(maxExpiry) {
		maxExpiry = *child.OwnerExpiresAt
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = maxExpiry
	} else if req.ExpiresAt.After(maxExpiry) {
		return nil, domain.NewValidationErrorWithCode("expiresAt", "delegated expiry exceeds the child or parent authority", domain.ErrCodePolicyScope)
	}

	now := o.now()
	claims, err := identity.Attenuate(verified.Claims, child.ID, child.TaskID, req.RequestedScopes, req.ExpiresAt, now)
	if err != nil {
		return nil, domain.NewValidationErrorWithCode("delegation", err.Error(), domain.ErrCodePolicyScope)
	}
	// Parent authority is one ceiling, not a substitute for the child's own
	// admitted account, profile and request ceilings. Filter concrete parent
	// scopes so wildcard filters never become new bearer capabilities.
	ownerScopes := append([]string{}, child.OwnerScopes...)
	claims.Scopes = identity.IntersectScopes(ownerScopes, profile.IdentityScopeCeiling(), claims.Scopes)
	claims.Scopes = identity.IntersectScopes(claims.Scopes, child.RequestedScopes, claims.Scopes)
	claims.ProfileKey = ""
	if profile != nil {
		claims.ProfileKey = profile.ProfileKey
	}
	// Workflow role belongs to the child attempt, not to its delegating parent.
	claims.Meta = workflowIdentityMeta(child.CustomEnv)
	claims.Meta["credential_generation"] = uuid.NewString()
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
