package commerce

import (
	"context"
	"reflect"
	"testing"
)

func TestReceiptValidatorsNormalizeEveryPurchaseRailToOneShape(t *testing.T) {
	canonical := NormalizedSubscription{
		SubscriptionID: "sub-canonical", ExternalSubscription: "external-canonical",
		UserIdentity: "buyer@example.com", Status: "active", PlanTier: "pro",
		PriceID: "price-pro", BundleKey: "business_suite",
	}
	validators := ReceiptValidators{
		"stripe": StripeReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
		"apple":  AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
		"google": GoogleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
	}
	for _, source := range []string{"stripe", "apple", "google"} {
		got, err := validators.Validate(context.Background(), Receipt{Source: source, Token: "verified", UserIdentity: canonical.UserIdentity})
		if err != nil {
			t.Fatalf("%s: %v", source, err)
		}
		if !reflect.DeepEqual(got, canonical) {
			t.Fatalf("%s normalized = %+v, want %+v", source, got, canonical)
		}
	}
}

func TestReceiptValidatorsRejectWrongSourceAndUnknownRail(t *testing.T) {
	validator := AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) {
		return NormalizedSubscription{}, nil
	}}
	if _, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "x"}); err != ErrReceiptInvalid {
		t.Fatalf("wrong source error = %v", err)
	}
	if _, err := (ReceiptValidators{"apple": validator}).Validate(context.Background(), Receipt{Source: "play", Token: "x"}); err == nil {
		t.Fatal("unknown source unexpectedly accepted")
	}
}
