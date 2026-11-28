// Package credits provides BAS account usage read models.
package credits

import (
	"context"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// EntitlementProvider is the narrow lookup seam for isolated credit-service
// tests. Production reads verified snapshots directly from entitlement.Service.
type EntitlementProvider interface {
	// GetEntitlement retrieves a shared snapshot for a test identity.
	GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error)
}

// EntitlementLimitProvider is an optional fixture seam for credit-limit cases.
type EntitlementLimitProvider interface {
	LimitForEntitlement(*entitlement.Entitlement) (int, bool)
}
