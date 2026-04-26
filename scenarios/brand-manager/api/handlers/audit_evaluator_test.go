package handlers

import (
	"testing"

	"brand-manager/domain"
)

// Unit tests for audit rule evaluation decision logic.
// [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-PROVIDER]

func TestEvaluateRules_NilBrand(t *testing.T) {
	results := evaluateRules(ruleEvaluators, nil, "no brand assigned")
	if len(results) != len(ruleEvaluators) {
		t.Fatalf("expected %d results, got %d", len(ruleEvaluators), len(results))
	}
	for _, r := range results {
		if r.Pass {
			t.Errorf("rule %s should fail for nil brand", r.RuleID)
		}
		if r.Message != "no brand assigned" {
			t.Errorf("rule %s message = %q, want fallback", r.RuleID, r.Message)
		}
	}
}

func TestEvaluateRules_CompleteBrand(t *testing.T) {
	brand := &domain.Brand{
		Identity: &domain.Identity{
			DisplayName: "Test",
			LogoPath:    "/logo.png",
			FaviconPath: "/favicon.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#ff0000",
			Background: "#ffffff",
			Surface:    "#f5f5f5",
			Text:       "#333333",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Open Sans",
		},
	}

	results := evaluateRules(ruleEvaluators, brand, "")
	for _, r := range results {
		if !r.Pass {
			t.Errorf("rule %s should pass for complete brand, msg: %s", r.RuleID, r.Message)
		}
	}
}

func TestEvaluateRules_PartialBrand_MissingLogo(t *testing.T) {
	brand := &domain.Brand{
		Identity: &domain.Identity{
			DisplayName: "Test",
			FaviconPath: "/favicon.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#ff0000",
			Background: "#ffffff",
			Surface:    "#f5f5f5",
			Text:       "#333333",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Open Sans",
		},
	}

	results := evaluateRules(ruleEvaluators, brand, "")
	for _, r := range results {
		if r.RuleID == "has-logo" {
			if r.Pass {
				t.Error("has-logo should fail without LogoPath")
			}
			return
		}
	}
	t.Error("has-logo rule not found in results")
}

func TestEvaluateRules_MinimalBrand_NoIdentity(t *testing.T) {
	brand := &domain.Brand{Name: "Bare"}

	results := evaluateRules(ruleEvaluators, brand, "")
	passing := 0
	for _, r := range results {
		if r.Pass {
			passing++
		}
	}
	if passing != 0 {
		t.Errorf("expected 0 passing rules for bare brand, got %d", passing)
	}
}

func TestRuleEvaluators_MatchStandardRules(t *testing.T) {
	if len(ruleEvaluators) != len(standardRules) {
		t.Fatalf("ruleEvaluators (%d) must match standardRules (%d)", len(ruleEvaluators), len(standardRules))
	}
	for i, re := range ruleEvaluators {
		if re.rule.ID != standardRules[i].ID {
			t.Errorf("evaluator[%d] rule ID = %q, want %q", i, re.rule.ID, standardRules[i].ID)
		}
	}
}
