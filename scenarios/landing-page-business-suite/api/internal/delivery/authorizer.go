// Package delivery owns entitlement-gating policy for paid delivery assets.
package delivery

import (
	"context"
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

// AssetLookup retrieves a catalog asset by its stable delivery identity.
//
// seam: AssetLookup
type AssetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*Asset, error)
}

// EntitlementStatusProvider resolves subscription status while preserving the
// request context at the commerce boundary.
//
// seam: EntitlementStatusProvider
type EntitlementStatusProvider interface {
	GetEntitlementStatus(context.Context, string) (string, error)
}

// DownloadAuthorizer coordinates catalog lookup with entitlement-gating
// policy. It belongs to delivery because it returns delivery assets and does
// not expose any commerce payloads to transport code.
type DownloadAuthorizer struct {
	downloads    AssetLookup
	entitlements EntitlementStatusProvider
	bundleKey    string
}

// NewDownloadAuthorizer wires the dependencies required for gated delivery.
func NewDownloadAuthorizer(downloads AssetLookup, entitlements EntitlementStatusProvider, bundleKey string) *DownloadAuthorizer {
	return &DownloadAuthorizer{downloads: downloads, entitlements: entitlements, bundleKey: bundleKey}
}

// Authorize validates a request and returns its resolved delivery asset.
func (a *DownloadAuthorizer) Authorize(ctx context.Context, appKey, platform, userIdentity string) (*Asset, error) {
	trimmedApp := strings.TrimSpace(appKey)
	if trimmedApp == "" {
		// Preserve the established catalog-facing contract: callers see a
		// missing application rather than a lower-level validation detail.
		return nil, ErrAppNotFound
	}
	trimmedPlatform := strings.TrimSpace(platform)
	if trimmedPlatform == "" {
		return nil, ErrPlatformRequired
	}
	asset, err := a.downloads.GetAsset(a.bundleKey, trimmedApp, trimmedPlatform)
	if err != nil {
		return nil, err
	}
	if err := Authorize(Request{
		AppKey: trimmedApp, Platform: trimmedPlatform, UserIdentity: userIdentity, RequiresEntitlement: asset.RequiresEntitlement,
	}, entitlementStatusLookup{ctx: ctx, provider: a.entitlements}); err != nil {
		return nil, err
	}
	return asset, nil
}

type entitlementStatusLookup struct {
	ctx      context.Context
	provider EntitlementStatusProvider
}

func (l entitlementStatusLookup) GetStatus(userIdentity string) (string, error) {
	if l.provider == nil {
		return "", ErrEntitlementsUnavailable
	}
	return l.provider.GetEntitlementStatus(l.ctx, userIdentity)
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
