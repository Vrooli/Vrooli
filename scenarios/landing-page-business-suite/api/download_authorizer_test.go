package main

import (
	"errors"
	"strings"
	"testing"
)

type fakeDownloads struct {
	assets map[string]*DownloadAsset
	err    error
}

func (f *fakeDownloads) GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := strings.Join([]string{bundleKey, appKey, platform}, ":")
	if asset, ok := f.assets[key]; ok {
		return asset, nil
	}
	return nil, errors.New("asset missing")
}

type trackingEntitlements struct {
	payload  *EntitlementPayload
	calls    int
	lastUser string
	err      error
}

func (t *trackingEntitlements) GetEntitlements(user string) (*EntitlementPayload, error) {
	t.calls++
	t.lastUser = user
	if t.err != nil {
		return nil, t.err
	}
	return t.payload, nil
}

func TestDownloadAuthorizerAuthorize_AllowsUngatedAssets(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:mac": {Platform: "mac", RequiresEntitlement: false},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "inactive"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	asset, err := authorizer.Authorize("app", "mac", "")
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
			"bundle:app:windows": {Platform: "windows", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "trialing"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("app", "windows", "user@example.com"); err != nil {
		t.Fatalf("expected access for trialing status, got %v", err)
	}
	if entitlements.calls != 1 {
		t.Fatalf("expected entitlement lookup once, got %d", entitlements.calls)
	}

	entitlements.payload.Status = "inactive"
	if _, err := authorizer.Authorize("app", "windows", "user@example.com"); !errors.Is(err, ErrDownloadRequiresActiveSubscription) {
		t.Fatalf("expected ErrDownloadRequiresActiveSubscription, got %v", err)
	}
}

func TestDownloadAuthorizerAuthorize_PropagatesLookupErrors(t *testing.T) {
	downloads := &fakeDownloads{
		err: ErrDownloadNotFound,
	}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("app", "ios", "user@example.com"); !errors.Is(err, ErrDownloadNotFound) {
		t.Fatalf("expected ErrDownloadNotFound, got %v", err)
	}
	if entitlements.calls != 0 {
		t.Fatalf("expected entitlement lookup skipped on asset failure, got %d", entitlements.calls)
	}
}

func TestDownloadAuthorizerAuthorize_RequiresIdentityForGatedAssets(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:linux": {Platform: "linux", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "active"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("app", "linux", ""); !errors.Is(err, ErrDownloadIdentityRequired) {
		t.Fatalf("expected ErrDownloadIdentityRequired, got %v", err)
	}
	if entitlements.calls != 0 {
		t.Fatalf("expected entitlements skipped until identity provided")
	}
}

func TestDownloadAuthorizerAuthorize_PropagatesEntitlementErrors(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:ios": {Platform: "ios", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		err: errors.New("entitlements offline"),
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	_, err := authorizer.Authorize("app", "ios", "user@example.com")
	if err == nil || !strings.Contains(err.Error(), "entitlements offline") {
		t.Fatalf("expected entitlement error to propagate, got %v", err)
	}
	if entitlements.calls != 1 {
		t.Fatalf("expected entitlement lookup once, got %d", entitlements.calls)
	}
}

func TestDownloadAuthorizerAuthorize_RejectsBlankPlatform(t *testing.T) {
	downloads := &fakeDownloads{}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("app", "   ", "user@example.com"); !errors.Is(err, ErrDownloadPlatformRequired) {
		t.Fatalf("expected ErrDownloadPlatformRequired, got %v", err)
	}
}

func TestDownloadAuthorizerAuthorize_ErrorsOnNilEntitlements(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:android": {Platform: "android", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("app", "android", "user@example.com"); err == nil {
		t.Fatalf("expected error when entitlement provider returns nil")
	}
}

func TestDownloadAuthorizerAuthorize_RequiresAppKey(t *testing.T) {
	downloads := &fakeDownloads{}
	entitlements := &trackingEntitlements{}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	if _, err := authorizer.Authorize("   ", "windows", "user@example.com"); !errors.Is(err, ErrDownloadAppNotFound) {
		t.Fatalf("expected ErrDownloadAppNotFound on blank app key, got %v", err)
	}
}

func TestDownloadAuthorizerAuthorize_TrimsInputsBeforeLookup(t *testing.T) {
	downloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:android": {Platform: "android", RequiresEntitlement: true},
		},
	}
	entitlements := &trackingEntitlements{
		payload: &EntitlementPayload{Status: "active"},
	}

	authorizer := NewDownloadAuthorizer(downloads, entitlements, "bundle")
	asset, err := authorizer.Authorize("  app  ", " android ", "  user@example.com ")
	if err != nil {
		t.Fatalf("expected trimmed inputs to authorize, got %v", err)
	}
	if asset.Platform != "android" {
		t.Fatalf("expected android asset returned, got %s", asset.Platform)
	}
	if entitlements.lastUser != "user@example.com" {
		t.Fatalf("expected trimmed identity passed to entitlements, got %q", entitlements.lastUser)
	}
}
