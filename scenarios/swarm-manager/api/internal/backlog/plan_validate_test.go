package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePlanCompleteness_CompletePlan(t *testing.T) {
	plan := `## Purpose
Some purpose.

## Required Reading
` + "```bash\nprompt-manager skill read implementation-plan-authoring\n```" + `

## Greenfield Declaration
This is greenfield work.

## Problem Statement
The problem.

## Scope
In scope.

## Current Technical Context
Context here.

## Target End State
End state.

## Implementation Strategy
Strategy.

## Contract Decisions
Decisions.

## Testing Plan
Tests.

## Rollout/Validation Checklist
1. vrooli scenario restart swarm-manager

## Risks + Mitigations
Risks.

## Non-goals/Prohibited Patterns
Non-goals.

## Definition of Done
Done.
`
	result := ValidatePlanCompleteness(plan, KindExecute)
	if !result.Passed {
		t.Errorf("expected Passed=true, got false. Missing: %v, Warnings: %v", result.SectionsMissing, result.Warnings)
	}
	if len(result.SectionsMissing) != 0 {
		t.Errorf("expected no missing sections, got %v", result.SectionsMissing)
	}
	if len(result.SectionsPresent) != 13 {
		t.Errorf("expected 13 sections present, got %d: %v", len(result.SectionsPresent), result.SectionsPresent)
	}
}

func TestValidatePlanCompleteness_MissingSections(t *testing.T) {
	plan := `## Purpose
Some purpose.

## Required Reading
` + "```bash\nprompt-manager skill read foo\n```" + `

## Greenfield Declaration
This is greenfield work.

## Problem Statement
The problem.

## Scope
In scope.

## Current Technical Context
Context here.

## Target End State
End state.

## Implementation Strategy
Strategy.

## Contract Decisions
Decisions.

## Rollout/Validation Checklist
1. vrooli scenario restart swarm-manager
`
	// Missing: Testing Plan, Risks + Mitigations, Non-goals/Prohibited Patterns, Definition of Done
	result := ValidatePlanCompleteness(plan, KindExecute)
	if result.Passed {
		t.Error("expected Passed=false for plan missing sections")
	}
	if len(result.SectionsMissing) != 4 {
		t.Errorf("expected 4 missing sections, got %d: %v", len(result.SectionsMissing), result.SectionsMissing)
	}
	expected := map[string]bool{
		"Testing Plan":                  true,
		"Risks + Mitigations":           true,
		"Non-goals/Prohibited Patterns": true,
		"Definition of Done":            true,
	}
	for _, s := range result.SectionsMissing {
		if !expected[s] {
			t.Errorf("unexpected missing section: %s", s)
		}
	}
}

func TestValidatePlanCompleteness_MissingPromptManager(t *testing.T) {
	plan := completePlanWithout("prompt-manager skill read")
	result := ValidatePlanCompleteness(plan, KindFix)
	if result.Passed {
		t.Error("expected Passed=false when prompt-manager is missing")
	}
	found := false
	for _, w := range result.Warnings {
		if contains(w, "prompt-manager") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about prompt-manager, got %v", result.Warnings)
	}
}

func TestValidatePlanCompleteness_MissingGreenfield(t *testing.T) {
	plan := strings.ReplaceAll(completePlanWithout("greenfield"), "Greenfield", "REDACTED")
	result := ValidatePlanCompleteness(plan, KindFix)
	if result.Passed {
		t.Error("expected Passed=false when greenfield is missing")
	}
	found := false
	for _, w := range result.Warnings {
		if contains(w, "greenfield") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about greenfield, got %v", result.Warnings)
	}
}

func TestValidatePlanCompleteness_MissingRestart(t *testing.T) {
	plan := completePlanWithout("vrooli scenario restart")
	result := ValidatePlanCompleteness(plan, KindFix)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "vrooli scenario restart") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about vrooli scenario restart, got %v", result.Warnings)
	}
}

func TestValidatePlanCompleteness_IdeaMissingTemplate(t *testing.T) {
	plan := completePlanBase()
	result := ValidatePlanCompleteness(plan, KindIdea)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "scenario-generation") || contains(w, "vrooli scenario create") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about scenario template for idea kind, got %v", result.Warnings)
	}
}

func TestValidatePlanCompleteness_IdeaMissingUI(t *testing.T) {
	// Plan without "ui" or "template" mentions
	plan := completePlanBase()
	result := ValidatePlanCompleteness(plan, KindIdea)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "UI") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about UI for idea kind, got %v", result.Warnings)
	}
}

func TestValidatePlanCompleteness_EmptyPlan(t *testing.T) {
	result := ValidatePlanCompleteness("", KindExecute)
	if result.Passed {
		t.Error("expected Passed=false for empty plan")
	}
	if len(result.SectionsMissing) != 13 {
		t.Errorf("expected 13 missing sections, got %d", len(result.SectionsMissing))
	}
}

func TestValidatePlanCompleteness_ResearchKindSkipped(t *testing.T) {
	result := ValidatePlanCompleteness("", KindResearch)
	if !result.Passed {
		t.Error("expected Passed=true for research kind (skipped)")
	}
}

func TestValidatePlanCompleteness_FuzzyMatching(t *testing.T) {
	tests := []struct {
		header  string
		section string
		match   bool
	}{
		{"## Required Reading", "Required Reading", true},
		{"## 2. Required Reading", "Required Reading", true},
		{"## required reading", "Required Reading", true},
		{"### Required Reading", "Required Reading", false}, // h3 should NOT match
		{"## Required-Reading", "Required Reading", true},
		{"## Risks + Mitigations", "Risks + Mitigations", true},
		{"## Risks and Mitigations", "Risks + Mitigations", true},
		{"## 11. Risks + Mitigations", "Risks + Mitigations", true},
	}

	for _, tc := range tests {
		t.Run(tc.header, func(t *testing.T) {
			// Build a plan with only this header plus all required elements
			plan := tc.header + "\nContent.\n"
			result := ValidatePlanCompleteness(plan, KindExecute)
			found := false
			for _, s := range result.SectionsPresent {
				if s == tc.section {
					found = true
					break
				}
			}
			if found != tc.match {
				t.Errorf("header %q: expected match=%v for section %q, got %v. Present: %v",
					tc.header, tc.match, tc.section, found, result.SectionsPresent)
			}
		})
	}
}

func TestWriteAndLoadValidationReport(t *testing.T) {
	dir := t.TempDir()
	report := PlanValidationResult{
		SectionsPresent: []string{"Purpose", "Scope"},
		SectionsMissing: []string{"Testing Plan"},
		Warnings:        []string{"No greenfield"},
		Passed:          false,
		ValidatedAt:     "2026-04-07T00:00:00Z",
	}
	if err := WriteValidationReport(dir, report); err != nil {
		t.Fatalf("WriteValidationReport: %v", err)
	}

	loaded, err := LoadValidationReport(dir)
	if err != nil {
		t.Fatalf("LoadValidationReport: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil report")
	}
	if loaded.Passed != false {
		t.Error("expected Passed=false")
	}
	if len(loaded.SectionsMissing) != 1 || loaded.SectionsMissing[0] != "Testing Plan" {
		t.Errorf("unexpected SectionsMissing: %v", loaded.SectionsMissing)
	}
}

func TestLoadValidationReport_NotExist(t *testing.T) {
	dir := t.TempDir()
	loaded, err := LoadValidationReport(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for non-existent report")
	}
}

func TestLoadOrRefreshValidationReport_StaleReport(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.md")
	reportPath := filepath.Join(dir, "validation-report.json")

	// Write initial plan and report
	plan := completePlanBase()
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	staleReport := PlanValidationResult{Passed: true, ValidatedAt: "2026-01-01T00:00:00Z"}
	data, _ := json.MarshalIndent(staleReport, "", "  ")
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Make plan newer than report (touch plan)
	// The report was written first, so plan is already newer or same. Force staleness:
	// Backdate the report file.
	oldTime := staleReport.ValidatedAt
	_ = oldTime
	// On most systems writing plan after report is enough, but let's be explicit:
	if err := os.WriteFile(planPath, []byte(plan+"\n## Extra"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadOrRefreshValidationReport(dir, KindExecute)
	if err != nil {
		t.Fatalf("LoadOrRefreshValidationReport: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// The refreshed result should reflect current plan content (missing sections from base plan)
	if result.ValidatedAt == "2026-01-01T00:00:00Z" {
		t.Error("expected report to be refreshed with new timestamp")
	}
}

func TestFormatGapReport(t *testing.T) {
	result := PlanValidationResult{
		SectionsMissing: []string{"Testing Plan", "Definition of Done"},
		Warnings:        []string{"No `prompt-manager skill read` command found in Required Reading"},
		Passed:          false,
	}
	report := FormatGapReport(result)
	if !contains(report, "Missing section: Testing Plan") {
		t.Error("expected gap report to mention Testing Plan")
	}
	if !contains(report, "Missing section: Definition of Done") {
		t.Error("expected gap report to mention Definition of Done")
	}
	if !contains(report, "prompt-manager") {
		t.Error("expected gap report to mention prompt-manager warning")
	}
}

func TestFormatGapReport_Passed(t *testing.T) {
	result := PlanValidationResult{Passed: true}
	report := FormatGapReport(result)
	if report != "" {
		t.Errorf("expected empty gap report for passed validation, got: %s", report)
	}
}

func TestValidateSuggestedSkills(t *testing.T) {
	tests := []struct {
		name            string
		planContent     string
		suggestedSkills []string
		wantCount       int
	}{
		{
			name:            "nil skills returns no warnings",
			planContent:     "some plan content",
			suggestedSkills: nil,
			wantCount:       0,
		},
		{
			name:            "empty skills returns no warnings",
			planContent:     "some plan content",
			suggestedSkills: []string{},
			wantCount:       0,
		},
		{
			name:            "skill present in plan",
			planContent:     "## Required Reading\nprompt-manager skill read scientific-debugging\n",
			suggestedSkills: []string{"scientific-debugging"},
			wantCount:       0,
		},
		{
			name:            "skill missing from plan",
			planContent:     "## Required Reading\nprompt-manager skill read other-skill\n",
			suggestedSkills: []string{"scientific-debugging"},
			wantCount:       1,
		},
		{
			name:            "multiple skills all present",
			planContent:     "prompt-manager skill read refactor screaming-architecture-audit\n",
			suggestedSkills: []string{"refactor", "screaming-architecture-audit"},
			wantCount:       0,
		},
		{
			name:            "one present one missing",
			planContent:     "prompt-manager skill read refactor\n",
			suggestedSkills: []string{"refactor", "screaming-architecture-audit"},
			wantCount:       1,
		},
		{
			name:            "case insensitive match",
			planContent:     "prompt-manager skill read Scientific-Debugging\n",
			suggestedSkills: []string{"scientific-debugging"},
			wantCount:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateSuggestedSkills(tt.planContent, tt.suggestedSkills)
			if len(warnings) != tt.wantCount {
				t.Errorf("ValidateSuggestedSkills() returned %d warnings, want %d: %v", len(warnings), tt.wantCount, warnings)
			}
		})
	}
}

func TestLoadOrRefreshValidationReport_SuggestedSkills(t *testing.T) {
	dir := t.TempDir()

	// Write plan.md that references "refactor" but not "screaming-architecture-audit"
	plan := completePlanBase()
	plan = strings.ReplaceAll(plan, "prompt-manager skill read implementation-plan-authoring",
		"prompt-manager skill read implementation-plan-authoring refactor")
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write spec.json with suggested_skills
	spec := map[string]any{
		"name":             "test-item",
		"title":            "Test",
		"description":      "Test",
		"status":           "ready",
		"priority":         5,
		"tags":             []string{},
		"created":          "2026-04-07T00:00:00Z",
		"updated":          "2026-04-07T00:00:00Z",
		"kind":             "execute",
		"suggested_skills": []string{"refactor", "screaming-architecture-audit"},
	}
	specData, _ := json.MarshalIndent(spec, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), specData, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadOrRefreshValidationReport(dir, KindExecute)
	if err != nil {
		t.Fatalf("LoadOrRefreshValidationReport: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Should have a warning about screaming-architecture-audit being missing
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "screaming-architecture-audit") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing screaming-architecture-audit, got warnings: %v", result.Warnings)
	}
}

// --- helpers ---

// completePlanBase returns a valid plan with all 13 sections and required elements.
func completePlanBase() string {
	return `## Purpose
Some purpose.

## Required Reading
` + "```bash\nprompt-manager skill read implementation-plan-authoring\n```" + `

## Greenfield Declaration
This is greenfield work.

## Problem Statement
The problem.

## Scope
In scope.

## Current Technical Context
Context here.

## Target End State
End state.

## Implementation Strategy
Strategy.

## Contract Decisions
Decisions.

## Testing Plan
Tests.

## Rollout/Validation Checklist
1. vrooli scenario restart swarm-manager

## Risks + Mitigations
Risks.

## Non-goals/Prohibited Patterns
Non-goals.

## Definition of Done
Done.
`
}

// completePlanWithout returns a complete plan with a specific string removed.
func completePlanWithout(remove string) string {
	plan := completePlanBase()
	return strings.ReplaceAll(plan, remove, "REDACTED_ELEMENT")
}
