// Package delivery owns entitlement-gating policy for paid delivery assets.
package delivery

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAppKeyRequired             = errors.New("download app key is required")
	ErrPlatformRequired           = errors.New("platform is required")
	ErrIdentityRequired           = errors.New("user identity required for gated downloads")
	ErrRequiresActiveSubscription = errors.New("active subscription required for downloads")
	ErrEntitlementsUnavailable    = errors.New("entitlements unavailable")
)

// EntitlementLookup resolves the current subscription status for a caller.
//
// seam: EntitlementLookup
type EntitlementLookup interface {
	GetStatus(userIdentity string) (string, error)
}

// Request is the normalized delivery-access decision input.
type Request struct {
	AppKey              string
	Platform            string
	UserIdentity        string
	RequiresEntitlement bool
}

// [REQ:DOWNLOAD-GATE] Authorize validates a delivery request and, when needed, requires an active
// or trialing subscription. Asset retrieval intentionally remains at the
// delivery composition boundary so this policy has no database dependency.
func Authorize(request Request, entitlements EntitlementLookup) error {
	if strings.TrimSpace(request.AppKey) == "" {
		return ErrAppKeyRequired
	}
	if strings.TrimSpace(request.Platform) == "" {
		return ErrPlatformRequired
	}
	if !request.RequiresEntitlement {
		return nil
	}
	identity := strings.TrimSpace(request.UserIdentity)
	if identity == "" {
		return ErrIdentityRequired
	}
	if entitlements == nil {
		return fmt.Errorf("retrieve entitlements: %w", ErrEntitlementsUnavailable)
	}
	status, err := entitlements.GetStatus(identity)
	if err != nil {
		return fmt.Errorf("retrieve entitlements: %w", err)
	}
	if status != "active" && status != "trialing" {
		return ErrRequiresActiveSubscription
	}
	return nil
}
