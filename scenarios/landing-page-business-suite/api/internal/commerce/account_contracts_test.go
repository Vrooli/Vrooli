package commerce

import (
	"encoding/json"
	"reflect"
	"testing"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestEntitlementPayloadSerializesStableCustomerFields(t *testing.T) {
	payload := EntitlementPayload{Status: "active", PlanTier: "pro", PriceID: "price_1", Features: []string{"download"}, AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "audio-tools", Status: "active", Features: []string{"audio:export"}}}, Credits: &shared.CreditsBalance{BalanceCredits: 42}}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "active" || body["plan_tier"] != "pro" || body["price_id"] != "price_1" {
		t.Fatalf("body=%#v", body)
	}
	if _, ok := body["features"]; !ok {
		t.Fatalf("features missing from %#v", body)
	}
	if _, ok := body["app_entitlements"]; !ok {
		t.Fatalf("app-scoped grants missing from %#v", body)
	}
}

func TestEntitlementPayloadLeaseClaimsPreserveAppScopedEffectiveGrants(t *testing.T) {
	grants := []entitlementclient.AppEntitlement{{AppKey: "audio-tools", BundleKey: "business_suite", Status: "active", PlanRank: 2, Features: []string{"audio:export"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 100, BundleKey: "business_suite"}}}}
	claims := (&EntitlementPayload{
		Status: "active", PlanTier: "solo", PlanRank: 2, Features: []string{"audio:export"},
		Limits: grants[0].Limits, AppEntitlements: grants,
	}).LeaseClaims("buyer@example.test")
	if claims.UserIdentity != "buyer@example.test" || !reflect.DeepEqual(claims.AppEntitlements, grants) {
		t.Fatalf("lease claims lost identity or app scope: %#v", claims)
	}
}

func TestCreditsEnvelopeRetainsDisplayConfiguration(t *testing.T) {
	payload := CreditsEnvelope{Balance: &shared.CreditsBalance{BalanceCredits: 7}, DisplayCreditsLabel: "tokens", DisplayCreditsMultiplier: 1.5}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	if body["display_credits_label"] != "tokens" || body["display_credits_multiplier"] != 1.5 {
		t.Fatalf("body=%#v", body)
	}
}
