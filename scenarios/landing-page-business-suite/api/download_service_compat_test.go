package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
)

type DownloadAuthorizer = delivery.DownloadAuthorizer

var (
	ErrDownloadRequiresActiveSubscription = delivery.ErrRequiresActiveSubscription
	ErrDownloadIdentityRequired           = delivery.ErrIdentityRequired
	ErrDownloadPlatformRequired           = delivery.ErrPlatformRequired
	ErrDownloadEntitlementsUnavailable    = delivery.ErrEntitlementsUnavailable
)

func NewDownloadService(db delivery.CatalogStore) *delivery.CatalogService {
	return delivery.NewCatalogService(db)
}

type downloadAssetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*delivery.Asset, error)
}

type entitlementProvider interface {
	GetEntitlementsContext(context.Context, string) (*commerce.EntitlementPayload, error)
}

type legacyEntitlementStatusProvider struct{ provider entitlementProvider }

func (p legacyEntitlementStatusProvider) GetEntitlementStatus(ctx context.Context, userIdentity string) (string, error) {
	entitlements, err := p.provider.GetEntitlementsContext(ctx, userIdentity)
	if err != nil || entitlements == nil {
		return "", err
	}
	return entitlements.Status, nil
}

func NewDownloadAuthorizer(downloads downloadAssetLookup, entitlements entitlementProvider, bundleKey string) *delivery.DownloadAuthorizer {
	return delivery.NewDownloadAuthorizer(downloads, legacyEntitlementStatusProvider{provider: entitlements}, bundleKey)
}
