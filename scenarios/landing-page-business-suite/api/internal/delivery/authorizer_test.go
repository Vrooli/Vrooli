package delivery

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/api-core/database"
)

type entitlementStub struct {
	status string
	err    error
	user   string
}

func (s *entitlementStub) GetStatus(userIdentity string) (string, error) {
	s.user = userIdentity
	return s.status, s.err
}

// [REQ:DOWNLOAD-GATE] Paid assets require an active or trialing entitlement.
func TestAuthorizeGatedAsset(t *testing.T) {
	lookup := &entitlementStub{status: "trialing"}
	err := Authorize(Request{AppKey: "desktop", Platform: "linux", UserIdentity: " user@example.com ", RequiresEntitlement: true}, lookup)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if lookup.user != "user@example.com" {
		t.Fatalf("lookup identity = %q, want trimmed identity", lookup.user)
	}

	lookup.status = "canceled"
	err = Authorize(Request{AppKey: "desktop", Platform: "linux", UserIdentity: "user@example.com", RequiresEntitlement: true}, lookup)
	if !errors.Is(err, ErrRequiresActiveSubscription) {
		t.Fatalf("Authorize() error = %v, want ErrRequiresActiveSubscription", err)
	}
}

func TestAuthorizeUngatedAssetSkipsEntitlements(t *testing.T) {
	lookup := &entitlementStub{err: errors.New("must not be called")}
	if err := Authorize(Request{AppKey: "desktop", Platform: "mac", RequiresEntitlement: false}, lookup); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if lookup.user != "" {
		t.Fatal("unexpected entitlement lookup for ungated asset")
	}
}

type assetLookupStub struct{ asset *Asset }

func (s assetLookupStub) GetAsset(_, _, _ string) (*Asset, error) { return s.asset, nil }

type selectedAssetLookupStub struct {
	asset    *Asset
	ctx      context.Context
	bundle   string
	app      string
	platform string
	id       int64
}

type contextAssetLookupStub struct {
	ctx   context.Context
	asset *Asset
}

func (s *contextAssetLookupStub) GetAsset(_, _, _ string) (*Asset, error) {
	return &Asset{ID: 1, ArtifactURL: "https://wrong-primary.example/asset"}, nil
}

func (s *contextAssetLookupStub) GetAssetContext(ctx context.Context, _, _, _ string) (*Asset, error) {
	s.ctx = ctx
	return s.asset, nil
}

func (s *selectedAssetLookupStub) GetAsset(_, _, _ string) (*Asset, error) {
	return &Asset{ID: 999, Platform: "wrong-row"}, nil
}

func (s *selectedAssetLookupStub) GetAssetByIDContext(ctx context.Context, bundle, app, platform string, id int64) (*Asset, error) {
	s.ctx, s.bundle, s.app, s.platform, s.id = ctx, bundle, app, platform, id
	return s.asset, nil
}

type entitlementStatusStub struct {
	ctx    context.Context
	user   string
	status string
}

type requestIDContextKey struct{}

func (s *entitlementStatusStub) GetEntitlementStatus(ctx context.Context, user string) (string, error) {
	s.ctx = ctx
	s.user = user
	return s.status, nil
}

func TestDownloadAuthorizerPassesContextAndTrimmedIdentity(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "delivery-test")
	entitlements := &entitlementStatusStub{status: "active"}
	authorizer := NewDownloadAuthorizer(assetLookupStub{asset: &Asset{RequiresEntitlement: true}}, entitlements, "bundle")

	asset, err := authorizer.Authorize(ctx, " app ", " windows ", " user@example.com ")
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if asset == nil {
		t.Fatal("Authorize() returned nil asset")
	}
	if entitlements.ctx != ctx || entitlements.user != "user@example.com" {
		t.Fatalf("entitlement call = context %v user %q, want original context and trimmed identity", entitlements.ctx, entitlements.user)
	}
}

func TestDownloadAuthorizerOmittedSelectorUsesContextAwareLookup(t *testing.T) {
	lookup := &contextAssetLookupStub{asset: &Asset{ID: 2, ArtifactURL: "https://leased.example/asset"}}
	authorizer := NewDownloadAuthorizer(lookup, &entitlementStatusStub{status: "active"}, "bundle")
	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "leased-request")
	asset, err := authorizer.Authorize(ctx, "app", "linux", "user@example.com")
	if err != nil || asset == nil || asset.ID != 2 || asset.ArtifactURL != "https://leased.example/asset" {
		t.Fatalf("context-aware authorization = %+v, error = %v", asset, err)
	}
	if lookup.ctx != ctx {
		t.Fatalf("lookup context = %v, want request context", lookup.ctx)
	}
}

func TestDownloadAuthorizerOmittedSelectorRefusesLegacyLookupInTestMode(t *testing.T) {
	authorizer := NewDownloadAuthorizer(assetLookupStub{asset: &Asset{ID: 1}}, &entitlementStatusStub{status: "active"}, "bundle")
	if _, err := authorizer.Authorize(database.WithTestMode(context.Background()), "app", "linux", "user@example.com"); !errors.Is(err, ErrRequestStorageLeaseUnavailable) {
		t.Fatalf("legacy test-mode lookup error = %v, want ErrRequestStorageLeaseUnavailable", err)
	}
}

func TestDownloadAuthorizerSelectedUsesExactContextAwareCatalogRow(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "exact-row")
	lookup := &selectedAssetLookupStub{asset: &Asset{ID: 42, BundleKey: "bundle", AppKey: "desktop", Platform: "linux", RequiresEntitlement: true}}
	entitlements := &entitlementStatusStub{status: "active"}
	authorizer := NewDownloadAuthorizer(lookup, entitlements, "bundle")

	asset, err := authorizer.AuthorizeSelected(ctx, " desktop ", " linux ", " user@example.com ", 42)
	if err != nil {
		t.Fatalf("AuthorizeSelected() error = %v", err)
	}
	if asset == nil || asset.ID != 42 || lookup.ctx != ctx || lookup.bundle != "bundle" || lookup.app != "desktop" || lookup.platform != "linux" || lookup.id != 42 {
		t.Fatalf("selected lookup = asset:%+v ctx:%v bundle:%q app:%q platform:%q id:%d", asset, lookup.ctx, lookup.bundle, lookup.app, lookup.platform, lookup.id)
	}
	if entitlements.ctx != ctx || entitlements.user != "user@example.com" {
		t.Fatalf("entitlement lookup = ctx:%v user:%q", entitlements.ctx, entitlements.user)
	}
}

func TestDownloadAuthorizerSelectedNeverFallsBackWhenExactSelectorUnavailable(t *testing.T) {
	entitlements := &entitlementStatusStub{status: "active"}
	authorizer := NewDownloadAuthorizer(assetLookupStub{asset: &Asset{ID: 7, Platform: "linux"}}, entitlements, "bundle")
	if _, err := authorizer.AuthorizeSelected(context.Background(), "app", "linux", "user@example.com", 7); !errors.Is(err, ErrAssetSelectorUnavailable) {
		t.Fatalf("AuthorizeSelected() error = %v, want ErrAssetSelectorUnavailable", err)
	}
	if entitlements.ctx != nil {
		t.Fatal("exact selector failure reached entitlement authorization")
	}
}
