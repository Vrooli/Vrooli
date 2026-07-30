package delivery

import (
	"context"
	"errors"
	"testing"
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
