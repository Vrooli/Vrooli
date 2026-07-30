package main

import (
	"context"
	"errors"
	"testing"

	accounthttp "landing-page-business-suite-api/handlers/account"
	"landing-page-business-suite-api/internal/commerce"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestAccountTransportReaderMapsCommerceDTOs(t *testing.T) {
	subscription := &shared.SubscriptionStatus{UserIdentity: "customer@example.test"}
	credits := &commerce.CreditsEnvelope{DisplayCreditsLabel: "tokens", DisplayCreditsMultiplier: 2}
	entitlements := &commerce.EntitlementPayload{Status: "active", PlanTier: "pro", PriceID: "price_123", Features: []string{"downloads"}, BillingCycleStart: 42, Subscription: subscription}
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{subscription: subscription, credits: credits, entitlements: entitlements})

	gotSubscription, err := reader.GetSubscriptionContext(context.Background(), "customer@example.test")
	if err != nil || gotSubscription != subscription {
		t.Fatalf("subscription=%#v err=%v", gotSubscription, err)
	}
	gotCredits, err := reader.GetCreditsContext(context.Background(), "customer@example.test")
	if err != nil || gotCredits.DisplayCreditsLabel != "tokens" || gotCredits.DisplayCreditsMultiplier != 2 {
		t.Fatalf("credits=%#v err=%v", gotCredits, err)
	}
	gotEntitlements, err := reader.GetEntitlementsContext(context.Background(), "customer@example.test")
	if err != nil || gotEntitlements.PlanTier != "pro" || gotEntitlements.BillingCycleStart != 42 || gotEntitlements.Subscription != subscription {
		t.Fatalf("entitlements=%#v err=%v", gotEntitlements, err)
	}
}

func TestAccountTransportReaderPropagatesSourceErrors(t *testing.T) {
	want := errors.New("store unavailable")
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{err: want})
	if _, err := reader.GetCreditsContext(context.Background(), "customer@example.test"); !errors.Is(err, want) {
		t.Fatalf("credits error=%v, want %v", err, want)
	}
	if _, err := reader.GetEntitlementsContext(context.Background(), "customer@example.test"); !errors.Is(err, want) {
		t.Fatalf("entitlements error=%v, want %v", err, want)
	}
}

type fakeAccountTransportSource struct {
	subscription *shared.SubscriptionStatus
	credits      *commerce.CreditsEnvelope
	entitlements *commerce.EntitlementPayload
	err          error
}

func (f fakeAccountTransportSource) GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error) {
	return f.subscription, f.err
}

func (f fakeAccountTransportSource) GetCreditsContext(context.Context, string) (*commerce.CreditsEnvelope, error) {
	return f.credits, f.err
}

func (f fakeAccountTransportSource) GetEntitlementsContext(context.Context, string) (*commerce.EntitlementPayload, error) {
	return f.entitlements, f.err
}
