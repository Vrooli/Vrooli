package credits

import (
	"context"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

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
