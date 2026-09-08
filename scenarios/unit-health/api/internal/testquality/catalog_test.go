package testquality

import (
	"strings"
	"testing"
)

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
