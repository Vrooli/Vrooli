package account

import (
	"encoding/json"
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestEntitlementPayloadSerializesStableCustomerFields(t *testing.T) {
	payload := EntitlementPayload{Status: "active", PlanTier: "pro", PriceID: "price_1", Features: []string{"download"}, Credits: &shared.CreditsBalance{BalanceCredits: 42}}
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
