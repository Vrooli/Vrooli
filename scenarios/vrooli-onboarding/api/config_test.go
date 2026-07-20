package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestConfigGenerate verifies POST /api/v1/config/generate with valid resources.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerate(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": ["postgres", "redis"]}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	config, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'config' object")
	}

	resources, ok := config["resources"].(map[string]any)
	if !ok {
		t.Fatal("config missing 'resources' map")
	}

	if len(resources) != 2 {
		t.Errorf("expected 2 resources in config, got %d", len(resources))
	}

	if _, ok := resp["warnings"]; ok {
		t.Errorf("expected no warnings, got %v", resp["warnings"])
	}
}

// TestConfigGenerateEmptyResources verifies 400 when resources list is empty.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateEmptyResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": []}`)
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigGenerateUnknownResources verifies warnings for unknown resources.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateUnknownResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": ["postgres", "nonexistent-thing"]}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	config, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'config' object")
	}
	resources, ok := config["resources"].(map[string]any)
	if !ok {
		t.Fatal("config missing 'resources' map")
	}
	if _, ok := resources["postgres"]; !ok {
		t.Error("expected postgres in config resources")
	}
	if _, ok := resources["nonexistent-thing"]; ok {
		t.Error("unknown resource should not be in config")
	}

	warningsRaw, ok := resp["warnings"]
	if !ok {
		t.Fatal("expected warnings for unknown resources")
	}
	warningsList, ok := warningsRaw.([]any)
	if !ok || len(warningsList) == 0 {
		t.Error("expected non-empty warnings list")
	}
	// Verify warning content references the unknown resource
	foundUnknownWarning := false
	for _, w := range warningsList {
		ws, _ := w.(string)
		if strings.Contains(ws, "nonexistent-thing") {
			foundUnknownWarning = true
		}
	}
	if !foundUnknownWarning {
		t.Errorf("expected warning mentioning 'nonexistent-thing', got %v", warningsList)
	}
}

// TestConfigGenerateInvalidJSON verifies 400 for malformed JSON.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateInvalidJSON(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", "{bad json")
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigValidate verifies POST /api/v1/config/validate with valid config.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidate(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"postgres": {"enabled": true, "name": "postgres"}, "redis": {"enabled": true, "name": "redis"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, ok := resp["valid"].(bool)
	if !ok {
		t.Fatal("response missing 'valid' field")
	}
	if !valid {
		t.Errorf("expected valid=true, got false; results: %v", resp["results"])
	}
}

// TestConfigValidateWithDependencyWarnings verifies dependency warnings (judge0 without redis).
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateWithDependencyWarnings(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResJudge0, testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"judge0": {"enabled": true, "name": "judge0"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, _ := resp["valid"].(bool)
	if !valid {
		t.Error("expected valid=true even with dependency warnings")
	}

	results, ok := resp["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatal("expected non-empty results")
	}

	foundWarning := false
	for _, r := range results {
		result, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if result["resource"] == "judge0" {
			if warnings, ok := result["warnings"].([]any); ok && len(warnings) > 0 {
				foundWarning = true
			}
		}
	}
	if !foundWarning {
		t.Error("expected dependency warning for judge0 without redis enabled")
	}
}

// TestConfigValidateEmptyConfig verifies 400 when config is empty.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateEmptyConfig(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate", `{"resources": {}}`)
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigValidateUnknownResource verifies validation flags unknown resources as invalid.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateUnknownResource(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"fake-resource": {"enabled": true, "name": "fake-resource"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, _ := resp["valid"].(bool)
	if valid {
		t.Error("expected valid=false when unknown resources are in config")
	}
}

// TestConfigGenerateDuplicateResources verifies duplicate names produce unique config entries.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateDuplicateResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": ["postgres", "postgres"]}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	config, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'config' object")
	}
	resources, ok := config["resources"].(map[string]any)
	if !ok {
		t.Fatal("config missing 'resources' map")
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 unique resource in config, got %d", len(resources))
	}
}

// TestConfigGenerateAllUnknown verifies config is empty when all resources are unknown.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateAllUnknown(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": ["totally-fake", "also-fake"]}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	config, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'config' object")
	}
	resources, ok := config["resources"].(map[string]any)
	if !ok {
		t.Fatal("config missing 'resources' map")
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources when all are unknown, got %d", len(resources))
	}
	warnings, ok := resp["warnings"].([]any)
	if !ok {
		t.Fatal("response missing 'warnings' array")
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for all-unknown resources")
	}
}

// TestConfigValidateMultipleDependencyWarnings verifies multiple dependency warnings.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateMultipleDependencyWarnings(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		testResPostgres, testResRedis,
		{
			"name": "judge0", "status": "running",
			"installed": "true", "last_updated": "2026-01-01T00:00:00Z",
		},
	})

	// judge0 depends on postgres and redis - enable judge0 without them
	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"judge0": {"enabled": true, "name": "judge0"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatal("response missing 'results' array")
	}
	found := false
	for _, r := range results {
		result, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if result["resource"] == "judge0" {
			warnings, ok := result["warnings"].([]any)
			if !ok || len(warnings) < 2 {
				t.Errorf("expected at least 2 dependency warnings for judge0, got %d", len(warnings))
			}
			found = true
		}
	}
	if !found {
		t.Error("expected judge0 in validation results")
	}
}

// TestConfigValidateMixedKnownUnknown verifies mixed known and unknown resources.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateMixedKnownUnknown(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"postgres": {"enabled": true, "name": "postgres"}, "fake": {"enabled": true, "name": "fake"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, _ := resp["valid"].(bool)
	if valid {
		t.Error("expected valid=false when mix includes unknown resources")
	}

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatal("response missing 'results' array")
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (one per resource), got %d", len(results))
	}
}

// TestConfigGenerateEmptyBody verifies 400 for empty JSON body.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateEmptyBody(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{}`)
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigValidateDisabledResourceSkipsDepsCheck verifies disabled resources don't trigger dependency warnings.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateDisabledResourceSkipsDepsCheck(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResJudge0})

	// judge0 is disabled, so its dependencies should NOT generate warnings.
	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"judge0": {"enabled": false, "name": "judge0"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	results, ok := resp["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatal("response missing 'results' array")
	}
	result, ok := results[0].(map[string]any)
	if !ok {
		t.Fatal("result entry is not a map")
	}

	// Should have a "disabled" warning but NOT a dependency warning
	warnings, ok := result["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Fatal("expected at least a disabled warning")
	}
	for _, w := range warnings {
		ws, ok := w.(string)
		if !ok {
			continue
		}
		if strings.Contains(ws, "dependency") {
			t.Error("disabled resource should not generate dependency warnings")
		}
	}
}

// TestConfigValidateEnabledWithAllDeps verifies no warnings when all deps are satisfied.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateEnabledWithAllDeps(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis, testResJudge0})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"judge0": {"enabled": true, "name": "judge0"}, "postgres": {"enabled": true, "name": "postgres"}, "redis": {"enabled": true, "name": "redis"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, _ := resp["valid"].(bool)
	if !valid {
		t.Error("expected valid=true when all deps are satisfied")
	}

	// judge0 should have no warnings when all dependencies are enabled.
	results := resp["results"].([]any)
	for _, r := range results {
		result := r.(map[string]any)
		if result["resource"] == "judge0" {
			if warnings, ok := result["warnings"].([]any); ok && len(warnings) > 0 {
				t.Errorf("judge0 should have no warnings when all dependencies are enabled, got %v", warnings)
			}
		}
	}
}

// TestConfigGenerateNullResources verifies 400 for null resources array.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateNullResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": null}`)
	requireStatus(t, w, http.StatusBadRequest)
}
