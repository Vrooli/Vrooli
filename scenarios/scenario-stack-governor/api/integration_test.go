package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// postJSON sends a POST with JSON body to the test server and returns the response.
func postJSON(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// decodeJSON reads and decodes the response body into v.
func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// TestIntegration_RunReturnsMAKEFILERules verifies that running rules against a
// scenario with a valid canonical Makefile produces 3 MAKEFILE_* results, all passed.
func TestIntegration_RunReturnsMAKEFILERules(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "valid-makefile"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a valid canonical Makefile.
	canonical := generateCanonicalMakefile(scenarioName)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	if len(runResp.Results) != 3 {
		t.Fatalf("expected 3 MAKEFILE_* results, got %d", len(runResp.Results))
	}

	expectedRules := map[string]bool{
		"MAKEFILE_STRUCTURE": false,
		"MAKEFILE_LIFECYCLE": false,
		"MAKEFILE_QUALITY":   false,
	}
	for _, r := range runResp.Results {
		if _, ok := expectedRules[r.RuleID]; !ok {
			t.Errorf("unexpected rule %s in results", r.RuleID)
			continue
		}
		expectedRules[r.RuleID] = true
		if len(r.Findings) > 0 {
			t.Errorf("rule %s should have passed but had %d findings:", r.RuleID, len(r.Findings))
			for _, f := range r.Findings {
				t.Errorf("  [%s] %s", f.Level, f.Message)
			}
		}
	}
	for id, seen := range expectedRules {
		if !seen {
			t.Errorf("expected rule %s in results but not found", id)
		}
	}
}

func TestIntegration_ListRulesIncludesPackageGovernanceRule(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules")
	if err != nil {
		t.Fatalf("GET /api/v1/rules: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	}
	decodeJSON(t, resp, &payload)

	for _, rule := range payload.Rules {
		if rule.ID == packageGovernanceRuleID {
			return
		}
	}
	t.Fatalf("expected %s in rule catalog", packageGovernanceRuleID)
}

// TestIntegration_RunDetectsViolations verifies that a broken Makefile produces findings.
func TestIntegration_RunDetectsViolations(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "broken-makefile"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a broken Makefile: wrong header, missing targets.
	broken := "# Wrong Header\n\nhelp:\n\t@echo help\n"
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	// At least one rule should have findings (failures).
	hasFindings := false
	for _, r := range runResp.Results {
		if len(r.Findings) > 0 {
			hasFindings = true
			break
		}
	}
	if !hasFindings {
		t.Error("expected at least one rule to report findings for a broken Makefile")
	}
}

// TestIntegration_FixDryRun verifies that dry_run:true reports fixed:true but
// does not actually write a Makefile to disk.
func TestIntegration_FixDryRun(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "dryrun-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"},
		DryRun:        true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	if len(fixResp.Results) == 0 {
		t.Fatal("expected fix results")
	}

	for _, r := range fixResp.Results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true in dry-run", r.RuleID)
		}
		if r.Diff == nil {
			t.Errorf("rule %s: expected Diff to be populated in dry-run", r.RuleID)
		}
	}

	// Makefile must NOT exist on disk.
	makefilePath := filepath.Join(scenarioDir, "Makefile")
	if _, err := os.Stat(makefilePath); err == nil {
		t.Error("expected Makefile to NOT be written in dry-run mode")
	}
}

// TestIntegration_FixApplies verifies that fix without dry_run creates a Makefile
// on disk and that its content passes all 3 MAKEFILE_* check rules.
func TestIntegration_FixApplies(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "fix-apply"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	for _, r := range fixResp.Results {
		if !r.Fixed {
			t.Errorf("rule %s: expected fixed=true", r.RuleID)
		}
		if r.Error != "" {
			t.Errorf("rule %s: unexpected error: %s", r.RuleID, r.Error)
		}
	}

	// Verify Makefile was created.
	makefilePath := filepath.Join(scenarioDir, "Makefile")
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("expected Makefile to exist: %v", err)
	}

	// Verify it passes all 3 check rules.
	if violations, _ := CheckMakefileStructure(string(content), makefilePath); len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("structure violation: %s (line %d)", v.Message, v.Line)
		}
	}
	if violations, _ := CheckMakefileLifecycle(string(content), makefilePath); len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("lifecycle violation: %s (line %d)", v.Message, v.Line)
		}
	}
	if violations, _ := CheckMakefileQuality(string(content), makefilePath); len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("quality violation: %s (line %d)", v.Message, v.Line)
		}
	}
}

// TestIntegration_FixSweepMultipleScenarios verifies that fixing multiple
// scenarios in one request creates Makefiles for all of them.
func TestIntegration_FixSweepMultipleScenarios(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarios := []string{"sweep-alpha", "sweep-beta"}
	for _, name := range scenarios {
		mkdirAll(t, filepath.Join(root, "scenarios", name))
	}

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: scenarios,
		RuleIDs:       []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	// Should have 3 results per scenario (one per MAKEFILE_* rule).
	if len(fixResp.Results) != 6 {
		for i, r := range fixResp.Results {
			t.Logf("result[%d]: scenario=%s rule=%s fixed=%v", i, r.ScenarioName, r.RuleID, r.Fixed)
		}
		t.Fatalf("expected 6 results (3 per scenario), got %d", len(fixResp.Results))
	}

	// Verify both Makefiles exist on disk.
	for _, name := range scenarios {
		makefilePath := filepath.Join(root, "scenarios", name, "Makefile")
		if _, err := os.Stat(makefilePath); err != nil {
			t.Errorf("expected Makefile for %s to exist: %v", name, err)
		}
	}
}

// TestIntegration_FixPreservesCustomTargets verifies that a fix preserves custom
// targets that were in the original Makefile.
func TestIntegration_FixPreservesCustomTargets(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "custom-targets"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Start with a BROKEN Makefile that has custom targets — the fix should
	// regenerate canonical structure while preserving the custom targets.
	broken := `# Wrong Header

.PHONY: help deploy migrate

help:
	@echo "help"

deploy: ## Deploy to production
	@echo "Deploying..."

migrate: ## Run database migrations
	@echo "Migrating..."
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	// Read back the fixed Makefile.
	content, err := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	output := string(content)

	if !strings.Contains(output, "deploy:") {
		t.Error("expected custom target 'deploy' to be preserved")
	}
	if !strings.Contains(output, "migrate:") {
		t.Error("expected custom target 'migrate' to be preserved")
	}

	// Verify at least one result mentions preserved_custom changes.
	foundPreserved := false
	for _, r := range fixResp.Results {
		for _, c := range r.Changes {
			if c.Type == "preserved_custom" {
				foundPreserved = true
				break
			}
		}
	}
	if !foundPreserved {
		t.Error("expected preserved_custom changes in fix results")
	}
}

// TestIntegration_FixSkipsPassingMakefile verifies that a Makefile that already
// passes all checks is not modified by the fix endpoint.
func TestIntegration_FixSkipsPassingMakefile(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "passing-makefile"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a canonical Makefile that passes all checks.
	canonical := generateCanonicalMakefile(scenarioName)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	for _, r := range fixResp.Results {
		if r.Fixed {
			t.Errorf("rule %s: expected fixed=false for already-passing Makefile", r.RuleID)
		}
	}

	// File should be unchanged.
	content, _ := os.ReadFile(filepath.Join(scenarioDir, "Makefile"))
	if string(content) != canonical {
		t.Error("expected Makefile to be unchanged when already passing")
	}
}

// TestIntegration_FixApplyNoDiff verifies that applying fixes (non-dry-run)
// does not populate Diff fields.
func TestIntegration_FixApplyNoDiff(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "apply-no-diff"
	mkdirAll(t, filepath.Join(root, "scenarios", scenarioName))

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE"},
		DryRun:        false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	for _, r := range fixResp.Results {
		if r.Diff != nil {
			t.Errorf("rule %s: expected Diff to be nil on actual apply", r.RuleID)
		}
	}
}

// TestIntegration_RunPassedAndFinishedAtSet verifies that rules correctly set
// Passed=true and FinishedAt when there are no findings. This was broken when
// rule functions used unnamed return values with defer (Bug 1).
func TestIntegration_RunPassedAndFinishedAtSet(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "passed-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a valid canonical Makefile so all MAKEFILE_* rules pass.
	canonical := generateCanonicalMakefile(scenarioName)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	for _, r := range runResp.Results {
		if len(r.Findings) > 0 {
			continue // only check rules that should pass
		}
		if !r.Passed {
			t.Errorf("rule %s: Passed should be true when findings is empty (named return value bug)", r.RuleID)
		}
		if r.FinishedAt.IsZero() {
			t.Errorf("rule %s: FinishedAt should be set (named return value bug)", r.RuleID)
		}
	}
}

// TestIntegration_RunComputesCounts verifies that error_count and warn_count
// are correctly populated in rule results.
func TestIntegration_RunComputesCounts(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "counts-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a broken Makefile to trigger findings.
	broken := "# Wrong\nhelp:\n\t@echo help\n"
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	for _, r := range runResp.Results {
		// Manually count expected values.
		expectedErrors := 0
		expectedWarns := 0
		for _, f := range r.Findings {
			switch f.Level {
			case "error":
				expectedErrors++
			case "warn":
				expectedWarns++
			}
		}
		if r.ErrorCount != expectedErrors {
			t.Errorf("rule %s: error_count=%d, expected %d", r.RuleID, r.ErrorCount, expectedErrors)
		}
		if r.WarnCount != expectedWarns {
			t.Errorf("rule %s: warn_count=%d, expected %d", r.RuleID, r.WarnCount, expectedWarns)
		}
	}
}

// TestIntegration_RunCountsPassingRuleZero verifies that passing rules have
// error_count=0 and warn_count=0.
func TestIntegration_RunCountsPassingRuleZero(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "counts-pass"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	canonical := generateCanonicalMakefile(scenarioName)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	for _, r := range runResp.Results {
		if len(r.Findings) == 0 {
			if r.ErrorCount != 0 {
				t.Errorf("rule %s: expected error_count=0 for passing rule, got %d", r.RuleID, r.ErrorCount)
			}
			if r.WarnCount != 0 {
				t.Errorf("rule %s: expected warn_count=0 for passing rule, got %d", r.RuleID, r.WarnCount)
			}
		}
	}
}

// TestIntegration_FixPerRuleDryRun verifies that dry-run fix returns per-rule
// Diff only for rules that have violations.
func TestIntegration_FixPerRuleDryRun(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "per-rule-dryrun"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a canonical Makefile but break only the lifecycle (wrong start command).
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing,
		`@vrooli scenario start $(SCENARIO_NAME)`,
		`@vrooli scenario run $(SCENARIO_NAME)`, 1)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/fix", FixRequest{
		ScenarioNames: []string{scenarioName},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"},
		DryRun:        true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var fixResp FixResponse
	decodeJSON(t, resp, &fixResp)

	for _, r := range fixResp.Results {
		switch r.RuleID {
		case "MAKEFILE_LIFECYCLE":
			if !r.Fixed {
				t.Error("LIFECYCLE: expected fixed=true")
			}
			if r.Diff == nil {
				t.Error("LIFECYCLE: expected Diff populated in dry-run")
			}
		case "MAKEFILE_STRUCTURE":
			// Structure may or may not be violated depending on whether
			// the start command change affects structure checks.
			if !r.Fixed && r.Diff != nil {
				t.Error("STRUCTURE: if fixed=false, Diff should be nil")
			}
		case "MAKEFILE_QUALITY":
			if r.Fixed {
				t.Error("QUALITY: expected fixed=false (quality targets are correct)")
			}
			if r.Diff != nil {
				t.Error("QUALITY: expected nil Diff")
			}
		}
	}
}

// TestIntegration_RunFailedPassedIsFalse verifies that rules with findings
// correctly set Passed=false.
func TestIntegration_RunFailedPassedIsFalse(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "failed-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	// Write a broken Makefile to trigger findings.
	broken := "# Wrong\nhelp:\n\t@echo help\n"
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(broken), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	for _, r := range runResp.Results {
		if len(r.Findings) == 0 {
			continue
		}
		if r.Passed {
			t.Errorf("rule %s: Passed should be false when there are %d findings", r.RuleID, len(r.Findings))
		}
		if r.FinishedAt.IsZero() {
			t.Errorf("rule %s: FinishedAt should be set even on failure", r.RuleID)
		}
	}
}

// TestIntegration_RunRejectsInvalidJSON verifies that POST /run with invalid
// JSON returns a 400 error instead of silently using defaults.
func TestIntegration_RunRejectsInvalidJSON(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/run", "application/json", strings.NewReader("{invalid"))
	if err != nil {
		t.Fatalf("POST /run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TestIntegration_FixPanicRecovery verifies that a panicking fixer doesn't crash
// the server — instead it returns an error in the FixResult.
func TestIntegration_FixPanicRecovery(t *testing.T) {
	entry := RuleEntry{
		Definition: RuleDefinition{ID: "PANIC_TEST", Fixable: true},
		Fixer: func(ctx context.Context, repoRoot, scenario string, dryRun bool) []FixResult {
			panic("intentional test panic")
		},
	}

	results := callFixerSafe(t.Context(), entry, "/tmp", "test-scenario", false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Fixed {
		t.Error("expected fixed=false after panic")
	}
	if r.Error == "" {
		t.Error("expected error message after panic")
	}
	if !strings.Contains(r.Error, "fixer panicked") {
		t.Errorf("expected 'fixer panicked' in error, got: %s", r.Error)
	}
	if r.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got: %s", r.ScenarioName)
	}
}

// TestIntegration_RunTimedOutField verifies that RunResponse includes TimedOut field.
func TestIntegration_RunTimedOutField(t *testing.T) {
	srv, root := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	scenarioName := "timeout-test"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, scenarioDir)

	canonical := generateCanonicalMakefile(scenarioName)
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{scenarioName}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)

	// Under normal conditions, TimedOut should be false.
	if runResp.TimedOut {
		t.Error("expected TimedOut=false for normal run")
	}
}

// TestRegistryConsistency verifies that every rule in AllRules has a non-nil
// Runner and that Fixable rules have a non-nil Fixer. This prevents the old
// bug where adding a rule to the definitions but forgetting to add a runner
// caused silent skipping.
func TestRegistryConsistency(t *testing.T) {
	for _, entry := range AllRules() {
		if entry.Runner == nil {
			t.Errorf("rule %s has nil Runner", entry.Definition.ID)
		}
		if entry.Definition.Fixable && entry.Fixer == nil {
			t.Errorf("rule %s is marked Fixable but has nil Fixer", entry.Definition.ID)
		}
		if !entry.Definition.Fixable && entry.Fixer != nil {
			t.Errorf("rule %s is not Fixable but has a Fixer set", entry.Definition.ID)
		}
		if entry.Definition.ID == "" {
			t.Error("found rule with empty ID")
		}
	}

	// Verify no duplicate IDs.
	seen := map[string]bool{}
	for _, entry := range AllRules() {
		if seen[entry.Definition.ID] {
			t.Errorf("duplicate rule ID: %s", entry.Definition.ID)
		}
		seen[entry.Definition.ID] = true
	}
}
