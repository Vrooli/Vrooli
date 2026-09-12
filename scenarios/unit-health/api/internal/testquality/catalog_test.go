package testquality

import (
	"strings"
	"testing"
)

// [REQ:UH-ANALYZE-007]
func TestCatalogContractAndIndependentCopies(t *testing.T) {
	c, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != 7 {
		t.Fatalf("expected seven declared automated rule contracts, got %d", len(c.Rules))
	}
	for _, r := range c.Rules {
		if r.DefaultEnforcement != Advisory {
			t.Fatalf("uncalibrated blocking rule: %s", r.ID)
		}
	}
	c.Rules[0].SupportProfiles[0] = "mutated"
	again, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if again.Rules[0].SupportProfiles[0] == "mutated" {
		t.Fatal("catalog shares mutable policy")
	}
}

func TestCatalogRejectsDuplicateAndUnjustifiedBlocking(t *testing.T) {
	c, _ := LoadCatalog()
	c.Rules = append(c.Rules, c.Rules[0])
	if err := c.Validate(); err == nil {
		t.Fatal("accepted duplicate rule")
	}
	c, _ = LoadCatalog()
	c.Rules[0].DefaultEnforcement = Blocking
	if err := c.Validate(); err == nil {
		t.Fatal("accepted implicit promotion")
	}
	c, _ = LoadCatalog()
	c.Rules[0].PromotionPrerequisites = nil
	if err := c.Validate(); err == nil {
		t.Fatal("accepted absent promotion prerequisites")
	}
}

func TestEveryPromotionPrerequisiteIsRequired(t *testing.T) {
	for _, missing := range []string{"reviewed-calibration", "reviewed-holdout", "native-profile-conformance", "false-positive-budget", "owner-promotion-decision"} {
		c, _ := LoadCatalog()
		kept := []string{}
		for _, prerequisite := range c.Rules[0].PromotionPrerequisites {
			if prerequisite != missing {
				kept = append(kept, prerequisite)
			}
		}
		c.Rules[0].PromotionPrerequisites = kept
		if err := c.Validate(); err == nil {
			t.Fatalf("accepted promotion without %s", missing)
		}
	}
}

func TestCatalogRejectsInvalidPromotionDecisionStatus(t *testing.T) {
	c, _ := LoadCatalog()
	c.Rules[0].PromotionDecisions[0].Status = "maybe"
	if err := c.Validate(); err == nil {
		t.Fatal("accepted invalid promotion decision status")
	}
}

func TestCatalogCarriesHoldoutBudgetAndRequestedDecision(t *testing.T) {
	c, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	rule := c.Rules[0]
	if rule.FalsePositiveBudget != 0.05 || len(rule.PromotionDecisions) != 1 || rule.PromotionDecisions[0].Status != "requested" {
		t.Fatalf("promotion gate = %+v", rule)
	}
}

func TestCatalogProjectsOnlyApprovedBlockingEnforcement(t *testing.T) {
	c, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	c.Rules[0].DefaultEnforcement = Blocking
	c.Rules[0].PromotionDecisions[0].Status = "approved"
	results := []Result{{RuleID: c.Rules[0].ID, SupportProfile: c.Rules[0].SupportProfiles[0], Enforcement: Advisory}, {RuleID: "other", SupportProfile: c.Rules[0].SupportProfiles[0], Enforcement: Advisory}}
	projected := c.ApplyCatalogEnforcement(results)
	if projected[0].Enforcement != Blocking || projected[1].Enforcement != Advisory {
		t.Fatalf("projected enforcement = %+v", projected)
	}
	if !c.Rules[0].hasApprovedPromotion() {
		t.Fatal("approved promotion was not recognized")
	}
}

func TestEnforcementReferenceIsDeterministicAndLabelsPlannedScope(t *testing.T) {
	c, _ := LoadCatalog()
	first, second := c.EnforcementReference(), c.EnforcementReference()
	if first != second {
		t.Fatal("unstable generated reference")
	}
	for _, r := range c.Rules {
		if !strings.Contains(first, `id="`+r.DocumentationAnchor+`"`) || !strings.Contains(first, r.Intent) {
			t.Fatalf("missing documented rule %s", r.ID)
		}
	}
	if !strings.Contains(first, "Implementation: planned") || !strings.Contains(first, "not certification") {
		t.Fatal("reference overstates implemented scope")
	}
}
