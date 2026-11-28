package main

import (
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestWithMonthlyActiveUsersPreservesPeriodScopedFacts(t *testing.T) {
	ctx := policy.RightsContext{MonthlyActiveUsers: map[string]string{"2024-06": "above_threshold"}}
	updated, err := withMonthlyActiveUsers(ctx, "2024-08=at_or_below_threshold")
	if err != nil {
		t.Fatal(err)
	}
	if updated.MonthlyActiveUsers["2024-06"] != "above_threshold" || updated.MonthlyActiveUsers["2024-08"] != "at_or_below_threshold" {
		t.Fatalf("period facts were not preserved: %+v", updated.MonthlyActiveUsers)
	}
	if ctx.MonthlyActiveUsers["2024-08"] != "" {
		t.Fatal("hypothetical context update mutated the source context")
	}
}

func TestWithMonthlyActiveUsersRejectsInvalidFacts(t *testing.T) {
	for _, input := range []string{"2024-13=above_threshold", "2024-08=unknown", "not-a-period=below_threshold", "2024-08", "2024-08=-1", "2024-08=1.5", "2024-08="} {
		if _, err := withMonthlyActiveUsers(policy.RightsContext{}, input); err == nil {
			t.Errorf("accepted invalid hypothetical MAU fact %q", input)
		}
	}
}

func TestWithMonthlyActiveUsersPreservesExactCountsAndMarksThemHypothetical(t *testing.T) {
	ctx := policy.RightsContext{MonthlyActiveUsers: map[string]string{"2024-06": "0"}, MonthlyActiveUsersEvidence: map[string]string{"2024-06": "operator source"}}
	updated, err := withMonthlyActiveUsers(ctx, "2024-08=0")
	if err != nil {
		t.Fatal(err)
	}
	if updated.MonthlyActiveUsers["2024-08"] != "0" || updated.MonthlyActiveUsersEvidence["2024-08"] != "hypothetical rights-impact command-line input" {
		t.Fatalf("exact count or hypothetical evidence was lost: %+v", updated)
	}
	if ctx.MonthlyActiveUsers["2024-08"] != "" || ctx.MonthlyActiveUsersEvidence["2024-08"] != "" {
		t.Fatal("hypothetical count mutated its source context")
	}
}

func TestWithAnnualRevenuePreservesOtherContextAndRejectsInvalidAmounts(t *testing.T) {
	ctx := policy.RightsContext{RevenueStatus: "pre_revenue", Source: "fixture"}
	updated, err := withAnnualRevenue(ctx, "1000000")
	if err != nil {
		t.Fatal(err)
	}
	if updated.AnnualRevenueUSD == nil || *updated.AnnualRevenueUSD != 1000000 || updated.RevenueStatus != "exact_value" {
		t.Fatalf("hypothetical exact amount did not set its explicit category: %+v", updated)
	}
	if ctx.AnnualRevenueUSD != nil {
		t.Fatal("hypothetical exact amount mutated the recorded context")
	}
	for _, raw := range []string{"-1", "NaN", "+Inf", "not-a-number"} {
		if _, err := withAnnualRevenue(ctx, raw); err == nil {
			t.Errorf("accepted invalid annual revenue %q", raw)
		}
	}
}
