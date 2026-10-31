// Package credits provides unified credit management for all billable operations.
package credits

import (
	"context"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// EntitlementProvider abstracts entitlement lookups for testability.
//
// This interface serves as a testing seam, allowing the credit service to be
// tested without depending on the concrete entitlement.Service implementation.
// In production, the DefaultEntitlementProvider wraps the real entitlement service.
// In tests, a mock can be injected to control entitlement behavior.
//
// # Design Rationale
//
// The entitlement service has its own complex dependencies (HTTP client, cache,
// configuration). By abstracting it behind this interface, credit service tests
// can focus on credit logic without setting up entitlement infrastructure.
//
// # Testing Seam
//
// This is a "Strong" seam per SEAMS.md criteria:
// - Interface defined (this file)
// - Compile-time enforcement via var _ check
// - Injectable via ServiceOptions.EntitlementProvider
type EntitlementProvider interface {
	// GetEntitlement retrieves the entitlement for a user.
	// Returns nil if no entitlement service is configured.
	GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error)

	// CanUseAIWithEntitlement checks if the signed lease allows AI operations.
	CanUseAIWithEntitlement(ent *entitlement.Entitlement) bool
}

// EntitlementLimitProvider is an optional test and integration seam for the
// server-signed limit carried by an entitlement snapshot. It has no tier
// fallback: false means the authority did not publish that limit.
type EntitlementLimitProvider interface {
	LimitForEntitlement(*entitlement.Entitlement) (int, bool)
}

// DefaultEntitlementProvider wraps the real entitlement.Service.
// This is the production implementation used when no mock is injected.
type DefaultEntitlementProvider struct {
	svc *entitlement.Service
}

// NewDefaultEntitlementProvider creates a provider wrapping the real service.
func NewDefaultEntitlementProvider(svc *entitlement.Service) *DefaultEntitlementProvider {
	return &DefaultEntitlementProvider{svc: svc}
}

// GetEntitlement retrieves entitlement from the real service.
func (p *DefaultEntitlementProvider) GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	if p.svc == nil {
		return nil, nil
	}
	return p.svc.GetEntitlement(ctx, userIdentity)
}

// CanUseAIWithEntitlement checks AI access via the real service.
func (p *DefaultEntitlementProvider) CanUseAIWithEntitlement(ent *entitlement.Entitlement) bool {
	if p.svc == nil {
		return true // Allow if no service configured
	}
	return p.svc.CanUseAIWithEntitlement(ent)
}

func (p *DefaultEntitlementProvider) LimitForEntitlement(ent *entitlement.Entitlement) (int, bool) {
	if ent == nil {
		return 0, false
	}
	value, ok := ent.LimitValue("ai_credits")
	return int(value), ok
}

// Compile-time check that DefaultEntitlementProvider implements EntitlementProvider.
var _ EntitlementProvider = (*DefaultEntitlementProvider)(nil)
