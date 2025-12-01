package main

import (
	"errors"
	"testing"
)

type fakeDownloads struct {
	assets map[string]*DownloadAsset
	err    error
}

func (f *fakeDownloads) GetAsset(bundleKey, platform string) (*DownloadAsset, error) {
	if f.err != nil {
		return nil, f.err
	}
	if asset, ok := f.assets[bundleKey+":"+platform]; ok {
		return asset, nil
	}
	return nil, errors.New("asset missing")
}

type trackingEntitlements struct {
	payload *EntitlementPayload
	calls   int
	err     error
}

func (t *trackingEntitlements) GetEntitlements(user string) (*EntitlementPayload, error) {
	t.calls++
	if t.err != nil {
		return nil, t.err
	}
	return t.payload, nil
}

func TestDownloadAuthorizerAuthorize_AllowsUngatedAssets(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:mac": {Platform: "mac", RequiresEntitlement: false},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "inactive"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	asset, err := authorizer.Authorize("mac", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if asset.Platform != "mac" {
		t.Fatalf("expected mac asset, got %s", asset.Platform)
	}
	if entitlements.calls != 0 {
		t.Fatalf("expected entitlement lookup to be skipped, called %d", entitlements.calls)
	}
}

// [REQ:DOWNLOAD-GATE] Download gating enforces entitlement state before granting assets.
func TestDownloadAuthorizerAuthorize_RequiresActiveSubscription(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:windows": {Platform: "windows", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "trialing"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("windows", "user@example.com"); err != nil {
		t.Fatalf("expected access for trialing status, got %v", err)
	}
	if entitlements.calls != 1 {
		t.Fatalf("expected entitlement lookup once, got %d", entitlements.calls)
	}

	entitlements.payload.Status = "inactive"
	if _, err := authorizer.Authorize("windows", "user@example.com"); !errors.Is(err, ErrDownloadRequiresActiveSubscription) {
		t.Fatalf("expected ErrDownloadRequiresActiveSubscription, got %v", err)
	}
}

func TestDownloadAuthorizerAuthorize_PropagatesLookupErrors(t *testing.T) {
	downloads := &fakeDownloads{
		err: ErrDownloadNotFound,
	}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("ios", "user@example.com"); !errors.Is(err, ErrDownloadNotFound) {
		t.Fatalf("expected ErrDownloadNotFound, got %v", err)
	}
	if entitlements.calls != 0 {
		t.Fatalf("expected entitlement lookup skipped on asset failure, got %d", entitlements.calls)
	}
}

func TestDownloadAuthorizerAuthorize_RequiresIdentityForGatedAssets(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:linux": {Platform: "linux", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "active"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("linux", ""); !errors.Is(err, ErrDownloadIdentityRequired) {
		t.Fatalf("expected ErrDownloadIdentityRequired, got %v", err)
	}
	if entitlements.calls != 0 {
		t.Fatalf("expected entitlements skipped until identity provided")
	}
}

func TestDownloadAuthorizerAuthorize_RejectsBlankPlatform(t *testing.T) {
	downloads := &fakeDownloads{}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("   ", "user@example.com"); !errors.Is(err, ErrDownloadPlatformRequired) {
		t.Fatalf("expected ErrDownloadPlatformRequired, got %v", err)
	}
}

func TestDownloadAuthorizerAuthorize_ErrorsOnNilEntitlements(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:android": {Platform: "android", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("android", "user@example.com"); err == nil {
		t.Fatalf("expected error when entitlement provider returns nil")
	}
}
