package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestConfigGenerate verifies POST /api/v1/config/generate with valid resources.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerate(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "redis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": ["postgres", "redis"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

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

	// Should have no warnings
	if _, ok := resp["warnings"]; ok {
		t.Errorf("expected no warnings, got %v", resp["warnings"])
	}
}

// TestConfigGenerateEmptyResources verifies 400 when resources list is empty.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateEmptyResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestConfigGenerateUnknownResources verifies warnings for unknown resources.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateUnknownResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": ["postgres", "nonexistent-thing"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should include postgres in config
	config := resp["config"].(map[string]any)
	resources := config["resources"].(map[string]any)
	if _, ok := resources["postgres"]; !ok {
		t.Error("expected postgres in config resources")
	}
	if _, ok := resources["nonexistent-thing"]; ok {
		t.Error("unknown resource should not be in config")
	}

	// Should have warnings
	warnings, ok := resp["warnings"]
	if !ok {
		t.Fatal("expected warnings for unknown resources")
	}
	warningsList, ok := warnings.([]any)
	if !ok || len(warningsList) == 0 {
		t.Error("expected non-empty warnings list")
	}
}

// TestConfigGenerateInvalidJSON verifies 400 for malformed JSON.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateInvalidJSON(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/generate", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestConfigValidate verifies POST /api/v1/config/validate with valid config.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidate(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "redis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}, "redis": {"enabled": true, "name": "redis"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	valid, ok := resp["valid"].(bool)
	if !ok {
		t.Fatal("response missing 'valid' field")
	}
	if !valid {
		t.Errorf("expected valid=true, got false; results: %v", resp["results"])
	}
}

// TestConfigValidateWithDependencyWarnings verifies dependency warnings (postgis without postgres).
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateWithDependencyWarnings(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	// Enable postgis but NOT postgres - should trigger dependency warning
	body := `{"resources": {"postgis": {"enabled": true, "name": "postgis"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should still be valid (warnings, not errors)
	valid, _ := resp["valid"].(bool)
	if !valid {
		t.Error("expected valid=true even with dependency warnings")
	}

	// Check that results contain a warning about postgres dependency
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
		if result["resource"] == "postgis" {
			if warnings, ok := result["warnings"].([]any); ok && len(warnings) > 0 {
				foundWarning = true
			}
		}
	}
	if !foundWarning {
		t.Error("expected dependency warning for postgis without postgres enabled")
	}
}

// TestConfigValidateEmptyConfig verifies 400 when config is empty.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateEmptyConfig(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestConfigValidateUnknownResource verifies validation flags unknown resources as invalid.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateUnknownResource(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": {"fake-resource": {"enabled": true, "name": "fake-resource"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	valid, _ := resp["valid"].(bool)
	if valid {
		t.Error("expected valid=false when unknown resources are in config")
	}
}
