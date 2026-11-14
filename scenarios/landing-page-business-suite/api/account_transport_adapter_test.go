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
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{subscription: subscription, credits: credits, entitlements: entitlements}, resolveD8TestAccount)

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
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{err: want}, resolveD8TestAccount)
	if _, err := reader.GetCreditsContext(context.Background(), "customer@example.test"); !errors.Is(err, want) {
		t.Fatalf("credits error=%v, want %v", err, want)
	}
	if _, err := reader.GetEntitlementsContext(context.Background(), "customer@example.test"); !errors.Is(err, want) {
		t.Fatalf("entitlements error=%v, want %v", err, want)
	}
}

func TestAccountTransportReaderRejectsNilCommercePayloads(t *testing.T) {
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{}, resolveD8TestAccount)
	if _, err := reader.GetCreditsContext(context.Background(), "customer@example.test"); err == nil {
		t.Fatal("expected nil credits payload to fail closed")
	}
	if _, err := reader.GetEntitlementsContext(context.Background(), "customer@example.test"); err == nil {
		t.Fatal("expected nil entitlements payload to fail closed")
	}
	if _, err := reader.GetCommercialContext(context.Background(), "customer@example.test", "integrations", "github"); err == nil {
		t.Fatal("expected nil commercial payload source to fail closed")
	}
}

func TestAccountTransportReaderScopesCommercialContentToEligibleContext(t *testing.T) {
	reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{
		entitlements: &commerce.EntitlementPayload{Status: "active", PlanTier: "free"},
	}, resolveD8TestAccount)
	got, err := reader.GetCommercialContext(context.Background(), "customer@example.test", "integrations", "github")
	if err != nil || len(got.Content) != 1 || got.Content[0].ContentId != "integrations-capability-account-github" {
		t.Fatalf("context=%#v err=%v", got, err)
	}
	other, err := reader.GetCommercialContext(context.Background(), "customer@example.test", "settings", "github")
	if err != nil || len(other.Content) != 0 {
		t.Fatalf("unexpected non-integrations content=%#v err=%v", other, err)
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

func (f fakeAccountTransportSource) GetCreditsForBusinessAccountContext(_ context.Context, _ string, account string) (*commerce.CreditsEnvelope, error) {
	if account != "acct:test" {
		return nil, errors.New("unverified account reached credits reader")
	}
	return f.credits, f.err
}

func (f fakeAccountTransportSource) GetEntitlementsForBusinessAccountContext(_ context.Context, _ string, account string) (*commerce.EntitlementPayload, error) {
	if account != "acct:test" {
		return nil, errors.New("unverified account reached entitlements reader")
	}
	return f.entitlements, f.err
}

func resolveD8TestAccount(context.Context, string) (string, error) { return "acct:test", nil }

func TestAccountTransportReaderRequiresVerifiedAccount(t *testing.T) {
	for _, resolve := range []func(context.Context, string) (string, error){nil, func(context.Context, string) (string, error) { return "", nil }, func(context.Context, string) (string, error) { return "", errors.New("not a member") }} {
		reader := accounthttp.NewCommerceReader(fakeAccountTransportSource{credits: &commerce.CreditsEnvelope{}}, resolve)
		if _, err := reader.GetCreditsContext(context.Background(), "customer@example.test"); err == nil {
			t.Fatal("credits must fail without a verified account")
		}
	}
}
