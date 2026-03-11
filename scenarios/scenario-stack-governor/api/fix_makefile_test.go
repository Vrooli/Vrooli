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

	// Write a Makefile with custom targets.
	existing := generateCanonicalMakefile(scenarioName)
	// Insert custom targets before the shortcuts section.
	existing = strings.Replace(existing, "# Development shortcuts",
		`export-variants: ## Export variant data
	@echo "Exporting variants..."

sync-variants: ## Sync variant data
	@echo "Syncing variants..."

# Development shortcuts`, 1)

	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
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
	vars := extractCustomVariables(content)

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
