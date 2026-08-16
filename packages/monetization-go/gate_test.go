package monetization

import (
	"context"
	"errors"
	"testing"
	"time"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

func TestStatusDecision(t *testing.T) {
	for _, test := range []struct {
		status  string
		allowed bool
		warning bool
		reason  string
	}{
		{status: "active", allowed: true, reason: ReasonAllowed},
		{status: "trialing", allowed: true, reason: ReasonAllowed},
		{status: "past_due", allowed: true, warning: true, reason: ReasonPastDue},
		{status: "canceled", reason: ReasonSubscriptionInactive},
		{status: "inactive", reason: ReasonSubscriptionInactive},
		{status: "unknown", reason: ReasonSubscriptionInactive},
	} {
		t.Run(test.status, func(t *testing.T) {
			got := StatusDecision(entitlementclient.Payload{Status: test.status}, "/upgrade")
			if got.Allowed != test.allowed || got.Warning != test.warning || got.Reason != test.reason || got.UpgradePath != "/upgrade" {
				t.Fatalf("decision = %+v, want allowed=%v warning=%v reason=%q", got, test.allowed, test.warning, test.reason)
			}
		})
	}
}

func TestLeaseValuesDriveRankFeaturesAndLimits(t *testing.T) {
	payload := entitlementclient.Payload{
		PlanRank: 3,
		Features: []string{"ai"},
		Limits:   []entitlementclient.Limit{{Key: "workflow_executions", Value: 12, BundleKey: "business_suite"}},
	}
	if !HasFeature(payload, "ai") || HasFeature(payload, "recording") {
		t.Fatalf("feature lookup did not use lease payload")
	}
	for rank, want := range map[int32]bool{2: true, 3: true, 4: false} {
		if got := AtLeastRank(payload, rank); got != want {
			t.Errorf("rank %d = %v, want %v", rank, got, want)
		}
	}
	if value, ok := Limit(payload, "workflow_executions", "business_suite"); !ok || value != 12 {
		t.Fatalf("lease limit = %d,%v, want 12,true", value, ok)
	}
	if _, ok := Limit(payload, "workflow_executions", "other_bundle"); ok {
		t.Fatal("limit from another bundle was accepted")
	}
}

func TestGateFallsBackToFreeTierWhenLeaseUnavailableOrExpired(t *testing.T) {
	for _, err := range []error{entitlementclient.ErrLeaseUnavailable, entitlementclient.ErrLeaseExpired} {
		decision := fallbackDecision(err, "/upgrade")
		if decision.Allowed || decision.UpgradePath != "/upgrade" {
			t.Fatalf("fallback decision = %+v", decision)
		}
		if errors.Is(err, entitlementclient.ErrLeaseExpired) && decision.Reason != ReasonLeaseExpired {
			t.Fatalf("expired fallback reason = %q", decision.Reason)
		}
	}
}

func TestGateWithoutClientDoesNotGrantAccess(t *testing.T) {
	gate := NewGate(nil, nil, "business_suite")
	decision := gate.Feature(context.Background(), "user@example.com", "ai", 1)
	if decision.Allowed || decision.Reason != ReasonLeaseUnavailable {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestAtLeastRankDoesNotDependOnTimeOrTierNames(t *testing.T) {
	if !AtLeastRank(entitlementclient.Payload{PlanRank: 1, NotAfter: time.Now().Add(time.Hour)}, 1) {
		t.Fatal("rank boundary denied")
	}
}
