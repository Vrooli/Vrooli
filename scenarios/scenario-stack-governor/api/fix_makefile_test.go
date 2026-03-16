package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFixTestDir creates a minimal repo structure for fix tests.
func setupFixTestDir(t *testing.T) (repoRoot string) {
	t.Helper()
	root := t.TempDir()

	// Create minimal repo structure that FindRepoRoot recognizes.
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources"), 0o755); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestFixMakefile_GeneratesCanonical(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	// Should have 3 results, one per rule.
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true", r.RuleID)
		}
		if r.Error != "" {
			t.Errorf("rule %s: unexpected error: %s", r.RuleID, r.Error)
		}
		if r.Diff != nil {
			t.Errorf("rule %s: expected Diff to be nil on non-dry-run", r.RuleID)
		}
	}

	// Verify the file was written.
	content, err := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if err != nil {
		t.Fatalf("expected Makefile to be written: %v", err)
	}

	// Verify the generated content passes all 3 check rules.
	structureViolations, _ := CheckMakefileStructure(string(content), "Makefile")
	if len(structureViolations) != 0 {
		for _, v := range structureViolations {
			t.Errorf("structure violation: %s (line %d)", v.Message, v.Line)
		}
	}

	lifecycleViolations, _ := CheckMakefileLifecycle(string(content), "Makefile")
	if len(lifecycleViolations) != 0 {
		for _, v := range lifecycleViolations {
			t.Errorf("lifecycle violation: %s (line %d)", v.Message, v.Line)
		}
	}

	qualityViolations, _ := CheckMakefileQuality(string(content), "Makefile")
	if len(qualityViolations) != 0 {
		for _, v := range qualityViolations {
			t.Errorf("quality violation: %s (line %d)", v.Message, v.Line)
		}
	}
}

func TestFixMakefile_Idempotent(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "idempotent-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// First fix.
	FixMakefileAll(t.Context(), root, scenarioName, false)
	first, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	// Second fix.
	results := FixMakefileAll(t.Context(), root, scenarioName, false)
	second, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	if string(first) != string(second) {
		t.Error("expected idempotent output, but second fix produced different content")
	}

	// Second run should report no changes.
	for _, r := range results {
		if r.Fixed {
			t.Errorf("rule %s: expected fixed=false on second run", r.RuleID)
		}
	}
}

func TestFixMakefile_PreservesCustomTargets(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "custom-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a broken Makefile with custom targets that the fix must preserve.
	broken := `# Wrong Header

.PHONY: help export-variants sync-variants

help:
	@echo "help"

export-variants: ## Export variant data
	@echo "Exporting variants..."

sync-variants: ## Sync variant data
	@echo "Syncing variants..."
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	output := string(content)

	if !strings.Contains(output, "export-variants") {
		t.Error("expected custom target 'export-variants' to be preserved")
	}
	if !strings.Contains(output, "sync-variants") {
		t.Error("expected custom target 'sync-variants' to be preserved")
	}

	// Check that at least one result has preserved_custom changes.
	foundPreserved := false
	for _, r := range results {
		for _, c := range r.Changes {
			if c.Type == "preserved_custom" {
				foundPreserved = true
				break
			}
		}
	}
	if !foundPreserved {
		t.Error("expected preserved_custom changes in results")
	}
}

func TestFixMakefile_SkipsAlreadyPassing(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "already-passing"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a canonical Makefile with custom targets — it should pass all checks.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "# Development shortcuts",
		`deploy: ## Deploy to production
	@echo "Deploying..."

# Development shortcuts`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	// Since the Makefile already passes all checks, fix should report fixed=false.
	for _, r := range results {
		if r.Fixed {
			t.Errorf("rule %s: expected fixed=false for already-passing Makefile", r.RuleID)
		}
	}

	// File should be unchanged.
	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if string(content) != existing {
		t.Error("expected Makefile to be unchanged when already passing")
	}
}

func TestFixMakefile_DryRunDoesNotWrite(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "dryrun-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	makefilePath := filepath.Join(scenarioDir, "Makefile")

	results := FixMakefileAll(t.Context(), root, scenarioName, true)

	// Should report fixed=true (would fix).
	for _, r := range results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true in dry-run", r.RuleID)
		}
	}

	// File should NOT exist.
	if _, err := os.Stat(makefilePath); err == nil {
		t.Error("expected Makefile to NOT be written in dry-run mode")
	}

	// Diff should be populated.
	if results[0].Diff == nil {
		t.Fatal("expected Diff to be populated in dry-run")
	}
	if results[0].Diff.Before != "" {
		t.Errorf("expected Diff.Before to be empty (no existing Makefile), got %q", results[0].Diff.Before)
	}
	if !strings.Contains(results[0].Diff.After, "Scenario Makefile") {
		t.Error("expected Diff.After to contain the canonical header")
	}
}

func TestFixMakefile_NoMakefile(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "no-makefile"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	for _, r := range results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true for missing Makefile", r.RuleID)
		}
		if r.Error != "" {
			t.Errorf("rule %s: unexpected error: %s", r.RuleID, r.Error)
		}
	}

	// Verify file was created.
	content, err := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if err != nil {
		t.Fatalf("expected Makefile to be created: %v", err)
	}

	if !strings.Contains(string(content), "No Makefile Scenario Makefile") {
		t.Error("expected header with scenario title")
	}
}

func TestFixMakefile_CustomTargetsPassChecks(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "custom-check"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile with custom targets.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "# Development shortcuts",
		`deploy: ## Deploy to production
	@echo "Deploying..."

# Development shortcuts`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	// Verify the fixed output still passes all check rules.
	structureViolations, _ := CheckMakefileStructure(string(content), "Makefile")
	if len(structureViolations) != 0 {
		for _, v := range structureViolations {
			t.Errorf("structure violation: %s (line %d)", v.Message, v.Line)
		}
	}

	lifecycleViolations, _ := CheckMakefileLifecycle(string(content), "Makefile")
	if len(lifecycleViolations) != 0 {
		for _, v := range lifecycleViolations {
			t.Errorf("lifecycle violation: %s (line %d)", v.Message, v.Line)
		}
	}

	qualityViolations, _ := CheckMakefileQuality(string(content), "Makefile")
	if len(qualityViolations) != 0 {
		for _, v := range qualityViolations {
			t.Errorf("quality violation: %s (line %d)", v.Message, v.Line)
		}
	}
}

func TestGenerateCanonicalMakefile_TitleConversion(t *testing.T) {
	tests := []struct {
		slug     string
		expected string
	}{
		{"my-cool-scenario", "My Cool Scenario"},
		{"simple", "Simple"},
		{"agent-inbox", "Agent Inbox"},
	}

	for _, tt := range tests {
		output := generateCanonicalMakefile(tt.slug)
		expectedHeader := "# " + tt.expected + " Scenario Makefile"
		if !strings.HasPrefix(output, expectedHeader) {
			firstLine := strings.SplitN(output, "\n", 2)[0]
			t.Errorf("slug %q: expected header %q, got %q", tt.slug, expectedHeader, firstLine)
		}
	}
}

func TestExtractCustomTargets(t *testing.T) {
	content := `help:
	@echo "help"

start:
	@echo "start"

export-variants: ## Export variants
	@echo "Exporting..."

sync-data:
	@echo "Syncing..."

dev: start
`
	customs := extractCustomTargets(content)

	names := map[string]bool{}
	for _, c := range customs {
		names[c.name] = true
	}

	if !names["export-variants"] {
		t.Error("expected export-variants to be extracted")
	}
	if !names["sync-data"] {
		t.Error("expected sync-data to be extracted")
	}
	if names["help"] || names["start"] || names["dev"] {
		t.Error("canonical/shortcut targets should not be extracted")
	}
}

func TestFixMakefile_PreservesCustomVariables(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "custom-vars"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile with a custom variable and a custom target that references it.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "RESET := \\033[0m",
		"RESET := \\033[0m\n\nAPI_URL ?= http://localhost:3000", 1)
	existing = strings.Replace(existing, "# Development shortcuts",
		`export-variants: ## Export variant data
	@echo "Exporting to $(API_URL)..."

# Development shortcuts`, 1)

	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	output := string(content)

	if !strings.Contains(output, "API_URL ?= http://localhost:3000") {
		t.Error("expected custom variable 'API_URL' to be preserved")
	}
	if !strings.Contains(output, "export-variants") {
		t.Error("expected custom target 'export-variants' to be preserved")
	}
}

func TestFixMakefile_PreservesCustomVariablesWithCustomTargets(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "vars-and-targets"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile with both custom variables and custom targets.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "RESET := \\033[0m",
		"RESET := \\033[0m\n\nAPI_URL ?= http://localhost:3000\nCUSTOM_FLAG := true", 1)
	existing = strings.Replace(existing, "# Development shortcuts",
		`export-variants: ## Export variant data
	@echo "Exporting to $(API_URL) with flag $(CUSTOM_FLAG)..."

# Development shortcuts`, 1)

	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	output := string(content)

	if !strings.Contains(output, "API_URL ?= http://localhost:3000") {
		t.Error("expected custom variable 'API_URL' to be preserved")
	}
	if !strings.Contains(output, "CUSTOM_FLAG := true") {
		t.Error("expected custom variable 'CUSTOM_FLAG' to be preserved")
	}
	if !strings.Contains(output, "export-variants") {
		t.Error("expected custom target 'export-variants' to be preserved")
	}

	// Variables should come before the custom targets.
	apiURLIdx := strings.Index(output, "API_URL ?=")
	exportIdx := strings.Index(output, "export-variants:")
	if apiURLIdx > exportIdx {
		t.Error("expected custom variables to appear before custom targets")
	}
}

func TestExtractCustomTargets_BackslashContinuation(t *testing.T) {
	content := `help:
	@echo "help"

start:
	@echo "start"

export-data: \
	build \
	test ## Export data after build and test
	@echo "Exporting..."

deploy: build
	@echo "Deploying..."

dev: start
`
	customs := extractCustomTargets(content)

	names := map[string]bool{}
	for _, c := range customs {
		names[c.name] = true
	}

	if !names["export-data"] {
		t.Error("expected export-data to be extracted")
	}
	if !names["deploy"] {
		t.Error("expected deploy to be extracted")
	}

	// Verify export-data has its prerequisites from the continuation line.
	for _, c := range customs {
		if c.name == "export-data" {
			if len(c.prerequisites) < 2 {
				t.Errorf("expected export-data to have at least 2 prerequisites (build, test), got %v", c.prerequisites)
			}
			break
		}
	}
}

func TestExtractCustomVariables_BackslashContinuation(t *testing.T) {
	content := `SCENARIO_NAME := my-scenario
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

MULTI_LINE := part1 \
	part2 \
	part3

SIMPLE_VAR := value

help: ## Help
	@echo "help"
`
	vars, _ := extractCustomVariables(content)

	names := map[string]bool{}
	for _, v := range vars {
		names[v.name] = true
	}

	if !names["MULTI_LINE"] {
		t.Error("expected MULTI_LINE to be extracted (multi-line variable)")
	}
	if !names["SIMPLE_VAR"] {
		t.Error("expected SIMPLE_VAR to be extracted")
	}
	if names["SCENARIO_NAME"] || names["GREEN"] {
		t.Error("canonical variables should not be extracted")
	}

	// Verify the multi-line definition includes the continuation content.
	for _, v := range vars {
		if v.name == "MULTI_LINE" {
			if !strings.Contains(v.definition, "part2") || !strings.Contains(v.definition, "part3") {
				t.Errorf("expected MULTI_LINE definition to include continuation parts, got %q", v.definition)
			}
			break
		}
	}
}

func TestJoinContinuationLines(t *testing.T) {
	lines := []string{
		"TARGET := value \\",
		"  continued \\",
		"  final",
		"\trecipe line 1 \\",
		"\trecipe line 2",
		"SIMPLE := value",
	}

	result := joinContinuationLines(lines)

	// The first three lines should be joined into one.
	if len(result) != 4 {
		t.Fatalf("expected 4 logical lines, got %d: %v", len(result), result)
	}

	// First logical line should contain all three parts.
	if !strings.Contains(result[0], "value") || !strings.Contains(result[0], "continued") || !strings.Contains(result[0], "final") {
		t.Errorf("expected joined continuation, got %q", result[0])
	}

	// Recipe lines should not be joined (even with backslash).
	if !strings.HasPrefix(result[1], "\t") {
		t.Errorf("expected recipe line preserved, got %q", result[1])
	}
}

func TestExtractCustomVariables(t *testing.T) {
	content := `SCENARIO_NAME := my-scenario
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

API_URL ?= http://localhost:3000
CUSTOM_FLAG := true

.DEFAULT_GOAL := help

help: ## Help
	@echo "help"

deploy: ## Deploy
	@echo "Deploying to $(API_URL)"
`
	vars, _ := extractCustomVariables(content)

	names := map[string]bool{}
	for _, v := range vars {
		names[v.name] = true
	}

	if !names["API_URL"] {
		t.Error("expected API_URL to be extracted")
	}
	if !names["CUSTOM_FLAG"] {
		t.Error("expected CUSTOM_FLAG to be extracted")
	}
	if names["SCENARIO_NAME"] {
		t.Error("SCENARIO_NAME should not be extracted (canonical)")
	}
	if names["GREEN"] {
		t.Error("GREEN should not be extracted (canonical)")
	}
	if names["RESET"] {
		t.Error("RESET should not be extracted (canonical)")
	}

	// Verify full definition is captured.
	for _, v := range vars {
		if v.name == "API_URL" && v.definition != "API_URL ?= http://localhost:3000" {
			t.Errorf("expected full definition, got %q", v.definition)
		}
	}
}

func TestFixMakefile_CustomVarsIdempotent(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "vars-idempotent"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile with custom variables.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "RESET := \\033[0m",
		"RESET := \\033[0m\n\nAPI_URL ?= http://localhost:3000", 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	// First fix.
	FixMakefileAll(t.Context(), root, scenarioName, false)
	first, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	// Second fix.
	FixMakefileAll(t.Context(), root, scenarioName, false)
	second, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	if string(first) != string(second) {
		t.Error("expected idempotent output with custom variables, but second fix produced different content")
	}
}

// TestFixMakefile_PhonyNoDuplicates verifies that the .PHONY line does not
// contain duplicated target names, even when custom target names are substrings
// of canonical targets (e.g. "lint" vs "lint-go").
func TestFixMakefile_PhonyNoDuplicates(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "phony-dedup"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a broken Makefile with custom targets whose names are substrings
	// of canonical targets.
	broken := `# Wrong Header
.PHONY: help lint-all check-all

help:
	@echo "help"

lint-all: ## Run all linters
	@echo "linting everything..."

check-all: ## Check everything
	@echo "checking..."
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	output := string(content)

	// Find the .PHONY line.
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, ".PHONY:") {
			continue
		}
		targets := strings.Fields(line)
		seen := map[string]int{}
		for _, t2 := range targets {
			seen[t2]++
		}
		for target, count := range seen {
			if count > 1 {
				t.Errorf(".PHONY contains duplicate target %q (%d times)", target, count)
			}
		}
		// Verify custom targets are present.
		if !strings.Contains(line, "lint-all") {
			t.Error("expected lint-all in .PHONY")
		}
		if !strings.Contains(line, "check-all") {
			t.Error("expected check-all in .PHONY")
		}
		break
	}
}

// TestFixMakefile_PhonySubstringTargets verifies that a custom target named
// "b" doesn't get skipped because "b" is a substring of "build".
func TestFixMakefile_PhonySubstringTargets(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "phony-substr"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	broken := `# Wrong
.PHONY: help deploy-staging

help:
	@echo "help"

deploy-staging: ## Deploy to staging
	@echo "deploying..."
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	// .PHONY should contain deploy-staging exactly once and as a distinct word.
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(line, ".PHONY:") {
			continue
		}
		targets := strings.Fields(line)
		count := 0
		for _, t2 := range targets {
			if t2 == "deploy-staging" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected deploy-staging exactly once in .PHONY, found %d times in: %s", count, line)
		}
		break
	}
}

// TestFixMakefile_PerRuleChanges verifies that each MAKEFILE_* rule gets only
// changes relevant to its specific violations.
func TestFixMakefile_PerRuleChanges(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "per-rule-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile that passes STRUCTURE and QUALITY but fails LIFECYCLE
	// (wrong echo message in start target).
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing,
		`@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"`,
		`@echo "Starting..."`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, true)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		switch r.RuleID {
		case "MAKEFILE_LIFECYCLE":
			if !r.Fixed {
				t.Error("MAKEFILE_LIFECYCLE: expected fixed=true (lifecycle has violations)")
			}
			if r.Diff == nil {
				t.Error("MAKEFILE_LIFECYCLE: expected Diff to be populated")
			}
			if len(r.Changes) == 0 {
				t.Error("MAKEFILE_LIFECYCLE: expected non-empty changes")
			}
		case "MAKEFILE_STRUCTURE":
			// Structure also detects the echo change as a help-target violation
			// or other structural issue, so it may or may not be fixed.
			// The key test is that each rule gets independent status.
		case "MAKEFILE_QUALITY":
			// Quality checks fmt/lint/check targets which are canonical and correct.
			// It should not need fixing if the quality targets are fine.
			if r.Fixed && r.Diff == nil {
				t.Error("MAKEFILE_QUALITY: if fixed=true, Diff should be populated in dry-run")
			}
			if !r.Fixed && r.Diff != nil {
				t.Error("MAKEFILE_QUALITY: if fixed=false, Diff should be nil")
			}
			if !r.Fixed && len(r.Changes) > 0 {
				t.Error("MAKEFILE_QUALITY: if fixed=false, changes should be empty")
			}
		}
	}
}

// TestFixMakefile_PerRuleOnlyLifecycleFails verifies that when only lifecycle
// violations exist, only the LIFECYCLE result has fixed=true.
func TestFixMakefile_PerRuleOnlyLifecycleFails(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "only-lifecycle"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Start with canonical, then break only the start command (lifecycle violation).
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing,
		`@vrooli scenario start $(SCENARIO_NAME)`,
		`@vrooli scenario run $(SCENARIO_NAME)`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify precondition: structure and quality pass, lifecycle fails.
	structV, _ := CheckMakefileStructure(existing, "Makefile")
	lifecycleV, _ := CheckMakefileLifecycle(existing, "Makefile")
	qualityV, _ := CheckMakefileQuality(existing, "Makefile")
	if len(structV) != 0 {
		t.Fatalf("precondition: expected no structure violations, got %d", len(structV))
	}
	if len(lifecycleV) == 0 {
		t.Fatal("precondition: expected lifecycle violations")
	}
	if len(qualityV) != 0 {
		t.Fatalf("precondition: expected no quality violations, got %d", len(qualityV))
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, true)

	for _, r := range results {
		switch r.RuleID {
		case "MAKEFILE_LIFECYCLE":
			if !r.Fixed {
				t.Error("LIFECYCLE should be fixed=true")
			}
		case "MAKEFILE_STRUCTURE":
			if r.Fixed {
				t.Error("STRUCTURE should be fixed=false (no violations)")
			}
			if r.Diff != nil {
				t.Error("STRUCTURE should have nil Diff")
			}
			if len(r.Changes) > 0 {
				t.Errorf("STRUCTURE should have no changes, got %d", len(r.Changes))
			}
		case "MAKEFILE_QUALITY":
			if r.Fixed {
				t.Error("QUALITY should be fixed=false (no violations)")
			}
			if r.Diff != nil {
				t.Error("QUALITY should have nil Diff")
			}
			if len(r.Changes) > 0 {
				t.Errorf("QUALITY should have no changes, got %d", len(r.Changes))
			}
		}
	}
}

// TestFixMakefile_NewFileAllRulesFixed verifies that when no Makefile exists,
// all 3 rules report fixed=true (all need the new canonical file).
func TestFixMakefile_NewFileAllRulesFixed(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "new-file-all"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, true)

	for _, r := range results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true for new file", r.RuleID)
		}
		if r.Diff == nil {
			t.Errorf("rule %s: expected Diff populated for new file dry-run", r.RuleID)
		}
		// All should have "generated" change type.
		hasGenerated := false
		for _, c := range r.Changes {
			if c.Type == "generated" {
				hasGenerated = true
			}
		}
		if !hasGenerated {
			t.Errorf("rule %s: expected 'generated' change type", r.RuleID)
		}
	}
}

// TestFixMakefile_PerRuleWritesFileOnce verifies that when only one rule has
// violations, the file is still written (since all rules share the Makefile).
func TestFixMakefile_PerRuleWritesFileOnce(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "write-once"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Break only lifecycle.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing,
		`@vrooli scenario start $(SCENARIO_NAME)`,
		`@vrooli scenario run $(SCENARIO_NAME)`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	// At least lifecycle should be fixed.
	lifecycleFixed := false
	for _, r := range results {
		if r.RuleID == "MAKEFILE_LIFECYCLE" && r.Fixed {
			lifecycleFixed = true
		}
	}
	if !lifecycleFixed {
		t.Error("expected LIFECYCLE to be fixed")
	}

	// File should be written and pass all checks.
	content, err := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if err != nil {
		t.Fatalf("expected Makefile to exist: %v", err)
	}

	lifecycleV, _ := CheckMakefileLifecycle(string(content), "Makefile")
	if len(lifecycleV) != 0 {
		for _, v := range lifecycleV {
			t.Errorf("lifecycle violation after fix: %s", v.Message)
		}
	}
}

func TestFixMakefile_CustomVarsPassChecks(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "vars-checks"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Makefile with custom variables and targets.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "RESET := \\033[0m",
		"RESET := \\033[0m\n\nAPI_URL ?= http://localhost:3000", 1)
	existing = strings.Replace(existing, "# Development shortcuts",
		`deploy: ## Deploy to production
	@echo "Deploying to $(API_URL)..."

# Development shortcuts`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	FixMakefileAll(t.Context(), root, scenarioName, false)

	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))

	structureViolations, _ := CheckMakefileStructure(string(content), "Makefile")
	if len(structureViolations) != 0 {
		for _, v := range structureViolations {
			t.Errorf("structure violation: %s (line %d)", v.Message, v.Line)
		}
	}

	lifecycleViolations, _ := CheckMakefileLifecycle(string(content), "Makefile")
	if len(lifecycleViolations) != 0 {
		for _, v := range lifecycleViolations {
			t.Errorf("lifecycle violation: %s (line %d)", v.Message, v.Line)
		}
	}

	qualityViolations, _ := CheckMakefileQuality(string(content), "Makefile")
	if len(qualityViolations) != 0 {
		for _, v := range qualityViolations {
			t.Errorf("quality violation: %s (line %d)", v.Message, v.Line)
		}
	}
}

// TestExtractCustomTargets_SkipsPatternRules verifies that Make pattern rules
// (targets containing %) are not extracted as custom targets.
func TestExtractCustomTargets_SkipsPatternRules(t *testing.T) {
	content := `help:
	@echo "help"

%.o: %.c
	$(CC) -c $< -o $@

deploy: ## Deploy to production
	@echo "deploying"

%.pb.go: %.proto
	protoc --go_out=. $<

dev: start
`
	customs := extractCustomTargets(content)

	names := map[string]bool{}
	for _, c := range customs {
		names[c.name] = true
	}

	if !names["deploy"] {
		t.Error("expected deploy to be extracted as custom target")
	}
	for name := range names {
		if strings.Contains(name, "%") {
			t.Errorf("pattern rule %q should not be extracted as custom target", name)
		}
	}
}

// TestExtractCustomVariables_ReportsCollisions verifies that variables colliding
// with canonical names are reported in the second return value.
func TestExtractCustomVariables_ReportsCollisions(t *testing.T) {
	content := `SCENARIO_NAME := custom-override
GREEN := custom-green
API_URL ?= http://localhost:3000
`
	vars, collisions := extractCustomVariables(content)

	// SCENARIO_NAME and GREEN are canonical — should be reported as collisions.
	collisionSet := map[string]bool{}
	for _, c := range collisions {
		collisionSet[c] = true
	}
	if !collisionSet["SCENARIO_NAME"] {
		t.Error("expected SCENARIO_NAME in collisions")
	}
	if !collisionSet["GREEN"] {
		t.Error("expected GREEN in collisions")
	}

	// API_URL is not canonical — should be in vars, not collisions.
	varNames := map[string]bool{}
	for _, v := range vars {
		varNames[v.name] = true
	}
	if !varNames["API_URL"] {
		t.Error("expected API_URL in extracted custom variables")
	}
	if varNames["SCENARIO_NAME"] || varNames["GREEN"] {
		t.Error("canonical variables should not be in extracted custom variables")
	}
}

// TestFixMakefile_ReportsVariableCollisions verifies that when an existing
// Makefile redefines canonical variables, the fixer reports skipped_variable changes.
func TestFixMakefile_ReportsVariableCollisions(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "var-collision"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a broken Makefile that redefines SCENARIO_NAME.
	broken := `# Wrong Header
SCENARIO_NAME := hardcoded-name

help:
	@echo "help"
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	results := FixMakefileAll(t.Context(), root, scenarioName, false)

	foundSkipped := false
	for _, r := range results {
		for _, c := range r.Changes {
			if c.Type == "skipped_variable" && strings.Contains(c.Detail, "SCENARIO_NAME") {
				foundSkipped = true
			}
		}
	}
	if !foundSkipped {
		t.Error("expected skipped_variable change for SCENARIO_NAME collision")
	}
}
