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
// - Test double exists (MockEntitlementProvider)
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

// MockEntitlementProvider is a test double for unit testing credit logic.
//
// # Usage Example
//
//	mock := &credits.MockEntitlementProvider{
//	    Entitlement: &entitlement.Entitlement{
//	        Tier:   entitlement.TierPro,
//	        Status: entitlement.StatusActive,
//	    },
//	    AICreditsLimit: 500,
//	    CanUseAI:       true,
//	}
//	svc := credits.NewService(credits.ServiceOptions{
//	    EntitlementProvider: mock,
//	    // ...
//	})
type MockEntitlementProvider struct {
	// Entitlement is returned by GetEntitlement. Set to nil to simulate no subscription.
	Entitlement *entitlement.Entitlement

	// GetEntitlementError is returned as the error from GetEntitlement.
	GetEntitlementError error

	// AICreditsLimit is returned by the test-only signed-limit seam. Use -1 for
	// unlimited, 0 for no access.
	AICreditsLimit int

	// CanUseAI is returned by CanUseAIWithEntitlement.
	CanUseAI bool

	// GetEntitlementCalls tracks how many times GetEntitlement was called.
	GetEntitlementCalls int

	// LastUserIdentity records the last userIdentity passed to GetEntitlement.
	LastUserIdentity string
}

// GetEntitlement returns the configured mock entitlement.
func (m *MockEntitlementProvider) GetEntitlement(_ context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	m.GetEntitlementCalls++
	m.LastUserIdentity = userIdentity
	return m.Entitlement, m.GetEntitlementError
}

// CanUseAIWithEntitlement returns the configured mock value.
func (m *MockEntitlementProvider) CanUseAIWithEntitlement(_ *entitlement.Entitlement) bool {
	return m.CanUseAI
}

func (m *MockEntitlementProvider) LimitForEntitlement(_ *entitlement.Entitlement) (int, bool) {
	return m.AICreditsLimit, true
}

// Compile-time check that MockEntitlementProvider implements EntitlementProvider.
var _ EntitlementProvider = (*MockEntitlementProvider)(nil)
