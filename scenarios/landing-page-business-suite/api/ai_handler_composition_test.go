package main

import (
	"context"
	"testing"

	"landing-page-business-suite-api/internal/commerce"
)

func TestNewMeteredInferenceDependencies_ComposesUsageAndSubscriptionSeams(t *testing.T) {
	db := setupTestDB(t)
	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB:            db,
		LimitsService: NewLimitsService(db, "postgres"),
		Dialect:       "postgres",
	})
	account := newAccountServiceWithTestPlanStore(t, db)

	deps := newMeteredInferenceDependencies(nil, usage, account)
	if deps.Service != nil {
		t.Fatal("expected the supplied nil gateway to remain nil")
	}
	if deps.UserRateLimiter == nil || deps.IPRateLimiter == nil {
		t.Fatal("expected independent user and IP rate limiters")
	}
	if deps.IPKeyFunc == nil || deps.UserIdentity == nil || deps.WriteJSONError == nil || deps.Log == nil || deps.LogError == nil {
		t.Fatal("expected every HTTP composition dependency to be installed")
	}

	summary, err := deps.Usage(context.Background(), "Buyer@Example.com", "")
	if err != nil {
		t.Fatalf("usage seam returned an error: %v", err)
	}
	if summary.BillingPeriod == "" || summary.ResetDate.IsZero() {
		t.Fatalf("expected usage summary timing metadata, got %+v", summary)
	}

	tier, err := deps.SubscriptionTier(context.Background(), "nosubscription@example.com")
	if err != nil {
		t.Fatalf("subscription seam returned an error for a user without a subscription: %v", err)
	}
	if tier != "" {
		t.Fatalf("expected no tier for a user without a subscription, got %q", tier)
	}
}
