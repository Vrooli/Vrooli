// Package authz contains the deployment-manager mutation boundaries shared by
// typed transports and domain handlers. Authentication attaches a verified
// principal; this package applies the scenario capability policy.
package authz

import (
	"context"
	"errors"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const (
	ReadCapability        = "deployment-manager:read"
	WriteCapability       = "deployment-manager:write"
	DestructiveCapability = "deployment-manager:destructive"
	// The scenario authentication catalog intentionally exposes only the three
	// coarse effects. These operation-specific names remain source-level aliases
	// so owner routes can keep distinct checks and error messages without
	// declaring capabilities that the service schema rejects.
	ClientReceiptCapability      = WriteCapability
	ReleasePreparationCapability = WriteCapability
	ReadinessEvidenceCapability  = WriteCapability
)

// RequireRead admits any verified principal with the read capability. It is
// used for release evidence and review retrieval; mutation helpers below keep
// the stronger actor-specific rules.
func RequireRead(ctx context.Context) error {
	if _, ok := identity.PrincipalFromContext(ctx); !ok {
		return errors.New("verified deployment-manager principal required")
	}
	if _, err := authn.RequireCapability(ctx, ReadCapability); err != nil {
		return errors.New("deployment-manager read capability required")
	}
	return nil
}

// RequireWrite admits a verified human operator with the deployment-manager
// write capability. Domain handlers still apply their exact identity and
// approval checks after this boundary.
func RequireWrite(ctx context.Context) error {
	if _, err := authn.RequireHuman(ctx); err != nil {
		return errors.New("verified deployment-manager operator required")
	}
	if _, err := authn.RequireCapability(ctx, WriteCapability); err != nil {
		return errors.New("deployment-manager write capability required")
	}
	return nil
}

// RequireReleasePreparation admits the service that has produced an
// immutable candidate or destination revision. The registration boundary
// still canonicalizes and derives every identity before persistence.
func RequireReleasePreparation(ctx context.Context) error {
	if _, err := authn.RequireService(ctx); err != nil {
		return errors.New("verified deployment-manager preparation service required")
	}
	if _, err := authn.RequireCapability(ctx, ReleasePreparationCapability); err != nil {
		return errors.New("deployment-manager release preparation capability required")
	}
	return nil
}

// RequireClientUpdateReceipt admits an authenticated owner service. Client
// update receipts are producer evidence, so a human operator must not be able
// to submit one through the owner route.
func RequireClientUpdateReceipt(ctx context.Context) error {
	if _, err := authn.RequireService(ctx); err != nil {
		return errors.New("verified deployment-manager owner service required")
	}
	if _, err := authn.RequireCapability(ctx, ClientReceiptCapability); err != nil {
		return errors.New("deployment-manager client receipt capability required")
	}
	return nil
}

// RequireReadinessEvidence admits an authenticated owner service that reports
// candidate-bound readiness observations. Readiness evidence changes release
// standing, so producer labels in the request are never sufficient authority.
func RequireReadinessEvidence(ctx context.Context) error {
	if _, err := authn.RequireService(ctx); err != nil {
		return errors.New("verified deployment-manager readiness owner service required")
	}
	if _, err := authn.RequireCapability(ctx, ReadinessEvidenceCapability); err != nil {
		return errors.New("deployment-manager readiness evidence capability required")
	}
	return nil
}

// RequireReadinessEvidenceForBinding additionally binds the authenticated
// owner service to the policy binding it is reporting. A capability grants
// access to the report route; it does not let one owner speak for another.
func RequireReadinessEvidenceForBinding(ctx context.Context, binding string) error {
	principal, err := RequireReadinessEvidencePrincipal(ctx)
	if err != nil {
		return err
	}
	binding = strings.TrimSpace(binding)
	owner, _, ok := strings.Cut(binding, ".")
	if !ok || owner == "" || principal.Subject != owner {
		return errors.New("readiness evidence service is not the declared producer owner")
	}
	return nil
}

// RequireEvidenceForRamp binds target evidence to the authenticated producer
// subject. The ramp value is a routing identity, not an authority claim.
func RequireEvidenceForRamp(ctx context.Context, ramp string) error {
	ramp = strings.TrimSpace(ramp)
	if ramp == "" {
		return errors.New("evidence target ramp is required")
	}
	return RequireReadinessEvidenceForBinding(ctx, ramp+".evidence")
}

func RequireReadinessEvidencePrincipal(ctx context.Context) (identity.Principal, error) {
	principal, err := authn.RequireService(ctx)
	if err != nil {
		return identity.Principal{}, errors.New("verified deployment-manager readiness owner service required")
	}
	if _, err := authn.RequireCapability(ctx, ReadinessEvidenceCapability); err != nil {
		return identity.Principal{}, errors.New("deployment-manager readiness evidence capability required")
	}
	return principal, nil
}

// RequireDestructive admits a verified human operator with the stronger
// recovery capability. Recovery handlers still validate the exact release,
// review, candidate, destination, and owner receipt binding.
func RequireDestructive(ctx context.Context) error {
	if _, err := authn.RequireHuman(ctx); err != nil {
		return errors.New("verified deployment-manager operator required")
	}
	if _, err := authn.RequireCapability(ctx, DestructiveCapability); err != nil {
		return errors.New("deployment-manager destructive capability required")
	}
	return nil
}
