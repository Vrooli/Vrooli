package main

import (
	"context"
	"strings"
	"testing"
)

func TestComputeCounts_Mixed(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "error", Message: "a"},
			{Level: "warn", Message: "b"},
			{Level: "error", Message: "c"},
			{Level: "info", Message: "d"},
			{Level: "warn", Message: "e"},
		},
	}
	r.ComputeCounts()

	if r.ErrorCount != 2 {
		t.Errorf("expected ErrorCount=2, got %d", r.ErrorCount)
	}
	if r.WarnCount != 2 {
		t.Errorf("expected WarnCount=2, got %d", r.WarnCount)
	}
}

func TestComputeCounts_AllErrors(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "error", Message: "a"},
			{Level: "error", Message: "b"},
		},
	}
	r.ComputeCounts()

	if r.ErrorCount != 2 {
		t.Errorf("expected ErrorCount=2, got %d", r.ErrorCount)
	}
	if r.WarnCount != 0 {
		t.Errorf("expected WarnCount=0, got %d", r.WarnCount)
	}
}

func TestComputeCounts_AllWarnings(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "warn", Message: "a"},
			{Level: "warn", Message: "b"},
			{Level: "warn", Message: "c"},
		},
	}
	r.ComputeCounts()

	if r.ErrorCount != 0 {
		t.Errorf("expected ErrorCount=0, got %d", r.ErrorCount)
	}
	if r.WarnCount != 3 {
		t.Errorf("expected WarnCount=3, got %d", r.WarnCount)
	}
}

func TestComputeCounts_NoFindings(t *testing.T) {
	r := RuleResult{}
	r.ComputeCounts()

	if r.ErrorCount != 0 {
		t.Errorf("expected ErrorCount=0, got %d", r.ErrorCount)
	}
	if r.WarnCount != 0 {
		t.Errorf("expected WarnCount=0, got %d", r.WarnCount)
	}
}

func TestComputeCounts_OnlyInfo(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "info", Message: "a"},
			{Level: "info", Message: "b"},
		},
	}
	r.ComputeCounts()

	if r.ErrorCount != 0 {
		t.Errorf("expected ErrorCount=0, got %d", r.ErrorCount)
	}
	if r.WarnCount != 0 {
		t.Errorf("expected WarnCount=0, got %d", r.WarnCount)
	}
}

func TestComputeCounts_Resets(t *testing.T) {
	r := RuleResult{
		ErrorCount: 99,
		WarnCount:  99,
		Findings: []Finding{
			{Level: "error", Message: "a"},
		},
	}
	r.ComputeCounts()

	if r.ErrorCount != 1 {
		t.Errorf("expected ErrorCount=1, got %d", r.ErrorCount)
	}
	if r.WarnCount != 0 {
		t.Errorf("expected WarnCount=0 (reset from 99), got %d", r.WarnCount)
	}
}

// --- Passed / info-level finding tests (Fix 3) ---

func TestComputeCounts_RecomputesPassed(t *testing.T) {
	r := RuleResult{
		Passed: true,
		Findings: []Finding{
			{Level: "error", Message: "fail"},
		},
	}
	r.ComputeCounts()
	if r.Passed {
		t.Error("expected Passed=false after ComputeCounts with error findings")
	}
}

func TestComputeCounts_InfoOnlyDoesNotFailRule(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "info", Message: "informational"},
			{Level: "info", Message: "also informational"},
		},
	}
	r.ComputeCounts()
	if !r.Passed {
		t.Error("expected Passed=true when only info-level findings exist")
	}
	if r.ErrorCount != 0 || r.WarnCount != 0 {
		t.Errorf("expected 0 errors and 0 warnings, got %d errors %d warnings", r.ErrorCount, r.WarnCount)
	}
}

func TestComputeCounts_MixedInfoAndErrorFailsRule(t *testing.T) {
	r := RuleResult{
		Findings: []Finding{
			{Level: "info", Message: "ok"},
			{Level: "error", Message: "bad"},
		},
	}
	r.ComputeCounts()
	if r.Passed {
		t.Error("expected Passed=false when error findings exist alongside info")
	}
}

func TestHasActionableFindings_Empty(t *testing.T) {
	if hasActionableFindings(nil) {
		t.Error("expected false for nil findings")
	}
	if hasActionableFindings([]Finding{}) {
		t.Error("expected false for empty findings")
	}
}

func TestHasActionableFindings_InfoOnly(t *testing.T) {
	findings := []Finding{{Level: "info", Message: "note"}}
	if hasActionableFindings(findings) {
		t.Error("expected false for info-only findings")
	}
}

func TestHasActionableFindings_WithError(t *testing.T) {
	findings := []Finding{
		{Level: "info", Message: "note"},
		{Level: "error", Message: "problem"},
	}
	if !hasActionableFindings(findings) {
		t.Error("expected true when error finding exists")
	}
}

func TestHasActionableFindings_WithWarn(t *testing.T) {
	findings := []Finding{{Level: "warn", Message: "caution"}}
	if !hasActionableFindings(findings) {
		t.Error("expected true when warn finding exists")
	}
}

// --- Scenario directory filtering tests (Fix 2) ---

func TestIsScenarioDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"normal scenario", "browser-automation-studio", true},
		{"underscore prefix", "_artifacts", false},
		{"dot prefix", ".git", false},
		{"dot vrooli", ".vrooli", false},
		{"empty string", "", false},
		{"single char", "x", true},
		{"hyphenated", "my-scenario", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isScenarioDir(tt.input)
			if got != tt.expected {
				t.Errorf("isScenarioDir(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// --- Registry validation tests ---

func TestValidateRuleRegistry_ProductionRegistryIsValid(t *testing.T) {
	// The real AllRules() registry must pass validation at all times.
	if err := ValidateRuleRegistry(); err != nil {
		t.Fatalf("production rule registry is invalid: %v", err)
	}
}

func TestValidateRuleRegistry_DuplicateID(t *testing.T) {
	// Save and restore the original AllRules function isn't possible since
	// it's a top-level function. Instead, test the validation logic directly
	// by checking that the production registry has unique IDs.
	seen := make(map[string]struct{})
	for _, entry := range AllRules() {
		if _, dup := seen[entry.Definition.ID]; dup {
			t.Errorf("duplicate rule ID in production registry: %s", entry.Definition.ID)
		}
		seen[entry.Definition.ID] = struct{}{}
	}
}

func TestValidateRuleRegistry_AllRunnersNonNil(t *testing.T) {
	for _, entry := range AllRules() {
		if entry.Runner == nil {
			t.Errorf("rule %s has nil Runner", entry.Definition.ID)
		}
	}
}

func TestValidateRuleRegistry_FixableConsistency(t *testing.T) {
	for _, entry := range AllRules() {
		if entry.Definition.Fixable && entry.Fixer == nil {
			t.Errorf("rule %s is Fixable but has nil Fixer", entry.Definition.ID)
		}
		if !entry.Definition.Fixable && entry.Fixer != nil {
			t.Errorf("rule %s has Fixer but is not Fixable", entry.Definition.ID)
		}
	}
}

func TestValidateRuleRegistry_FixerGroupConsistency(t *testing.T) {
	// Rules in the same FixerGroup must share the same Fixer function.
	groups := make(map[string]string) // group → first rule ID
	for _, entry := range AllRules() {
		if entry.FixerGroup == "" {
			continue
		}
		if _, ok := groups[entry.FixerGroup]; !ok {
			groups[entry.FixerGroup] = entry.Definition.ID
		}
	}

	// This is also validated by ValidateRuleRegistry(), but having explicit
	// test coverage makes regression debugging easier.
	if err := ValidateRuleRegistry(); err != nil {
		if strings.Contains(err.Error(), "FixerGroup") {
			t.Errorf("fixer group inconsistency: %v", err)
		}
	}
}

func TestValidateRuleRegistry_AllRunnersReturn(t *testing.T) {
	// Smoke test: each runner should not panic when given an empty repo root.
	// We don't check results — just that they don't crash.
	for _, entry := range AllRules() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("rule %s Runner panicked: %v", entry.Definition.ID, r)
				}
			}()
			entry.Runner(context.Background(), "/nonexistent-repo-root", "nonexistent-scenario")
		}()
	}
}
