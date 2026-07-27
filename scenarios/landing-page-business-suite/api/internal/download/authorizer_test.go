package download

import (
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
		t.Fatalf("unexpected entitlement lookup for ungated asset")
	}
}
