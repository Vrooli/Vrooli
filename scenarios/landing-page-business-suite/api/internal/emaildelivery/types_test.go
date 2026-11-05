package emaildelivery

import (
	"context"
	"testing"
)

type boolCheck bool

func (b boolCheck) Verify(context.Context, Provider) (bool, string) { return bool(b), "test" }

type quotaCheck bool

func (b quotaCheck) Available(context.Context, Provider) (bool, string) { return bool(b), "test" }

func TestRouterChoosesCheapestEligibleProviderAndExplainsSkips(t *testing.T) {
	route := Router{Credentials: boolCheck(true), DNS: boolCheck(true), Quota: quotaCheck(true)}.Select(context.Background(), PurposeSignIn, []Provider{
		{ID: "expensive", Enabled: true, CostRank: 20, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}},
		{ID: "blocked", Enabled: false, CostRank: 1, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}},
		{ID: "cheap", Enabled: true, CostRank: 5, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}},
	})
	if route.Chosen == nil || route.Chosen.ID != "cheap" {
		t.Fatalf("chosen = %#v", route.Chosen)
	}
	for _, candidate := range route.Candidates {
		if candidate.Provider.ID != "cheap" && candidate.SkipReason == "" {
			t.Fatalf("candidate explanation missing = %#v", route.Candidates)
		}
	}
}

func TestRouterOrderingChangesWithRegistryData(t *testing.T) {
	providers := []Provider{
		{ID: "mailgun", Enabled: true, CostRank: 20, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}},
		{ID: "resend", Enabled: true, CostRank: 10, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}},
	}
	route := (Router{}).Select(context.Background(), PurposeSignIn, providers)
	if route.Chosen == nil || route.Chosen.ID != "resend" {
		t.Fatalf("registry ordering chose %#v", route.Chosen)
	}
	for i := range providers {
		if providers[i].ID == "mailgun" {
			providers[i].CostRank = 5
		}
	}
	route = (Router{}).Select(context.Background(), PurposeSignIn, providers)
	if route.Chosen == nil || route.Chosen.ID != "mailgun" {
		t.Fatalf("updated registry ordering chose %#v", route.Chosen)
	}
}

func TestRouterExplainsOpenCircuit(t *testing.T) {
	route := Router{Circuit: quotaCheck(false)}.Select(context.Background(), PurposeSignIn, []Provider{{ID: "mailgun", Enabled: true, CostRank: 1, SupportedPurposes: map[Purpose]bool{PurposeSignIn: true}}})
	if route.Chosen != nil || len(route.Candidates) != 1 || route.Candidates[0].SkipReason != "provider circuit breaker is open" {
		t.Fatalf("circuit explanation = %#v", route)
	}
}
