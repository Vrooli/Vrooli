package main

import (
	"bytes"
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

	// Start with a canonical Makefile and inject custom targets.
	existing := generateCanonicalMakefile(scenarioName)
	existing = strings.Replace(existing, "# Development shortcuts",
		`deploy: ## Deploy to production
	@echo "Deploying..."

migrate: ## Run database migrations
	@echo "Migrating..."

# Development shortcuts`, 1)

	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte(existing), 0o644); err != nil {
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
