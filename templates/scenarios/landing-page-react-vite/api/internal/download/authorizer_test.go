package download_test

import (
	"errors"
	"landing-page-react-vite-api/internal/download"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeDownloads struct {
	assets map[string]*download.Asset
	err    error
}

func (f *fakeDownloads) GetAsset(bundleKey, appKey, platform string) (*download.Asset, error) {
	if f.err != nil {
		return nil, f.err
	}
	if asset, ok := f.assets[strings.Join([]string{bundleKey, appKey, platform}, ":")]; ok {
		return asset, nil
	}
	return nil, errors.New("asset missing")
}

type trackingEntitlements struct {
	payload *download.Entitlements
	calls   int
	err     error
}

func (t *trackingEntitlements) GetEntitlements(string) (*download.Entitlements, error) {
	t.calls++
	if t.err != nil {
		return nil, t.err
	}
	return t.payload, nil
}

func TestAuthorizeAllowsUngatedAssets(t *testing.T) {
	downloads := &fakeDownloads{assets: map[string]*download.Asset{"bundle:app:mac": {Platform: "mac", RequiresEntitlement: false}}}
	ent := &trackingEntitlements{payload: &download.Entitlements{Status: "inactive"}}
	asset, err := download.NewAuthorizer(downloads, ent, "bundle").Authorize("app", "mac", "")
	require.NoError(t, err)
	require.Equal(t, "mac", asset.Platform)
	require.Equal(t, 0, ent.calls, "entitlement lookup should be skipped for ungated asset")
}

func TestAuthorizeRequiresActiveSubscription(t *testing.T) {
	downloads := &fakeDownloads{assets: map[string]*download.Asset{"bundle:app:windows": {Platform: "windows", RequiresEntitlement: true}}}
	ent := &trackingEntitlements{payload: &download.Entitlements{Status: "trialing"}}
	authorizer := download.NewAuthorizer(downloads, ent, "bundle")

	_, err := authorizer.Authorize("app", "windows", "user@example.com")
	require.NoError(t, err)
	require.Equal(t, 1, ent.calls)

	ent.payload.Status = "inactive"
	_, err = authorizer.Authorize("app", "windows", "user@example.com")
	require.ErrorIs(t, err, download.ErrRequiresActive)
}

func TestAuthorizePropagatesLookupErrors(t *testing.T) {
	downloads := &fakeDownloads{err: download.ErrNotFound}
	ent := &trackingEntitlements{}
	_, err := download.NewAuthorizer(downloads, ent, "bundle").Authorize("app", "ios", "user@example.com")
	require.ErrorIs(t, err, download.ErrNotFound)
	require.Equal(t, 0, ent.calls)
}

func TestAuthorizeRequiresIdentityForGatedAssets(t *testing.T) {
	downloads := &fakeDownloads{assets: map[string]*download.Asset{"bundle:app:linux": {Platform: "linux", RequiresEntitlement: true}}}
	ent := &trackingEntitlements{payload: &download.Entitlements{Status: "active"}}
	_, err := download.NewAuthorizer(downloads, ent, "bundle").Authorize("app", "linux", "")
	require.ErrorIs(t, err, download.ErrIdentityRequired)
	require.Equal(t, 0, ent.calls)
}

func TestAuthorizeRejectsBlankPlatform(t *testing.T) {
	_, err := download.NewAuthorizer(&fakeDownloads{}, &trackingEntitlements{}, "bundle").Authorize("app", "   ", "user@example.com")
	require.ErrorIs(t, err, download.ErrPlatformRequired)
}

func TestAuthorizeErrorsOnNilEntitlements(t *testing.T) {
	downloads := &fakeDownloads{assets: map[string]*download.Asset{"bundle:app:mac": {Platform: "mac", RequiresEntitlement: true}}}
	_, err := download.NewAuthorizer(downloads, &trackingEntitlements{}, "bundle").Authorize("app", "mac", "user@example.com")
	require.Error(t, err)
}

func TestAuthorizeRequiresAppKey(t *testing.T) {
	_, err := download.NewAuthorizer(&fakeDownloads{}, &trackingEntitlements{}, "bundle").Authorize("   ", "windows", "user@example.com")
	require.ErrorIs(t, err, download.ErrAppNotFound)
}
