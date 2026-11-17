package monetization

import (
	"testing"
	"time"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

func TestSnapshotFromLeasePreservesVerifiedClaimsAndNormalizesDefaults(t *testing.T) {
	fetchedAt := time.Now().UTC()
	expiresAt := fetchedAt.Add(time.Hour)
	snapshot := SnapshotFromLease(entitlementclient.Payload{
		UserIdentity:      "member@example.com",
		Status:            "ACTIVE",
		PlanTier:          "PRO",
		PlanRank:          3,
		PriceID:           "price_pro",
		Features:          []string{FeatureAI},
		Limits:            []entitlementclient.Limit{{Key: "ai_credits", Value: 80}},
		Credits:           map[string]any{"balance_credits": float64(41)},
		BillingCycleStart: 15,
		NotAfter:          expiresAt,
	}, fetchedAt)
	if snapshot.UserIdentity != "member@example.com" || snapshot.Status != StatusActive || snapshot.Tier != "pro" || snapshot.PlanRank != 3 || snapshot.Credits != 41 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if !snapshot.IsActive() || len(snapshot.Features) != 1 || snapshot.Features[0] != FeatureAI {
		t.Fatalf("snapshot methods do not match verified claims: %#v", snapshot)
	}
	if len(snapshot.Limits) != 1 || snapshot.Limits[0].Key != "ai_credits" || snapshot.Limits[0].Value != 80 {
		t.Fatalf("limits = %#v", snapshot.Limits)
	}
	start, end := snapshot.GetBillingPeriod(time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC))
	if start.Day() != 15 || end.Day() != 14 {
		t.Fatalf("billing period = %s - %s", start, end)
	}
	defaults := SnapshotFromLease(entitlementclient.Payload{}, fetchedAt)
	if defaults.Tier != string(PlanFree) || defaults.Status != StatusInactive {
		t.Fatalf("default snapshot = %#v", defaults)
	}
}

func TestSnapshotIsExpiredUsesSignedLeaseBoundary(t *testing.T) {
	if !(&Snapshot{}).IsExpired() {
		t.Fatal("snapshot without an expiry must be expired")
	}
	if (&Snapshot{ExpiresAt: time.Now().Add(time.Hour)}).IsExpired() {
		t.Fatal("unexpired signed lease snapshot reported expired")
	}
	if !(&Snapshot{ExpiresAt: time.Now().Add(-time.Second)}).IsExpired() {
		t.Fatal("expired signed lease snapshot reported active")
	}
	var missing *Snapshot
	if !missing.IsExpired() {
		t.Fatal("missing snapshot must fail closed as expired")
	}
}

func TestSnapshotRetainsAppScopeAndDoesNotExposeGlobalFallback(t *testing.T) {
	snapshot := SnapshotFromLease(entitlementclient.Payload{
		Features: []string{"audio:export", "web:admin"},
		Limits:   []entitlementclient.Limit{{Key: "ai_credits", Value: 999}},
		AppEntitlements: []entitlementclient.AppEntitlement{
			{AppKey: "audio-tools", Status: "active", Features: []string{"audio:export"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 10}}},
			{AppKey: "web-console", Status: "active", Features: []string{"web:admin"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 40}}},
		},
	}, time.Now())
	if !snapshot.HasFeatureForApp("audio-tools", "audio:export") || snapshot.HasFeatureForApp("audio-tools", "web:admin") || snapshot.HasFeatureForApp("", "audio:export") {
		t.Fatal("snapshot did not preserve exact app feature scope")
	}
	if _, ok := LimitForApp(entitlementclient.Payload{AppEntitlements: snapshot.AppEntitlements}, "", "ai_credits", "suite"); ok {
		t.Fatal("unscoped snapshot meter read exposed app-scoped limit")
	}
	if value, ok := snapshot.LimitValueForApp("web-console", "ai_credits", "suite"); !ok || value != 40 {
		t.Fatalf("web console limit = %d,%v, want 40,true", value, ok)
	}
}
