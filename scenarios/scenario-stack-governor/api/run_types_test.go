package main

import (
	"context"
	"strings"
	"testing"
)

func TestComputeCounts(t *testing.T) {
	tests := []struct {
		name                  string
		findings              []Finding
		initialErrors         int
		initialWarnings       int
		wantErrors, wantWarns int
	}{
		{
			name: "mixed levels",
			findings: []Finding{
				{Level: "error", Message: "a"}, {Level: "warn", Message: "b"},
				{Level: "error", Message: "c"}, {Level: "info", Message: "d"}, {Level: "warn", Message: "e"},
			},
			wantErrors: 2, wantWarns: 2,
		},
		{name: "all errors", findings: []Finding{{Level: "error", Message: "a"}, {Level: "error", Message: "b"}}, wantErrors: 2},
		{name: "all warnings", findings: []Finding{{Level: "warn", Message: "a"}, {Level: "warn", Message: "b"}, {Level: "warn", Message: "c"}}, wantWarns: 3},
		{name: "no findings"},
		{name: "info only", findings: []Finding{{Level: "info", Message: "a"}, {Level: "info", Message: "b"}}},
		{name: "recomputes instead of accumulating", initialErrors: 99, initialWarnings: 99, findings: []Finding{{Level: "error", Message: "a"}}, wantErrors: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RuleResult{ErrorCount: tt.initialErrors, WarnCount: tt.initialWarnings, Findings: tt.findings}
			r.ComputeCounts()
			if r.ErrorCount != tt.wantErrors || r.WarnCount != tt.wantWarns {
				t.Errorf("counts = errors:%d warnings:%d, want errors:%d warnings:%d", r.ErrorCount, r.WarnCount, tt.wantErrors, tt.wantWarns)
			}
		})
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

func TestHasActionableFindings(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     bool
	}{
		{name: "nil", findings: nil},
		{name: "empty", findings: []Finding{}},
		{name: "info only", findings: []Finding{{Level: "info", Message: "note"}}},
		{name: "error", findings: []Finding{{Level: "info", Message: "note"}, {Level: "error", Message: "problem"}}, want: true},
		{name: "warning", findings: []Finding{{Level: "warn", Message: "caution"}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasActionableFindings(tt.findings); got != tt.want {
				t.Errorf("hasActionableFindings() = %v, want %v", got, tt.want)
			}
		})
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
