package bundles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"deployment-manager/secrets"
)

func TestHandleMergeBundleSecrets(t *testing.T) {
	sm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "API_KEY",
					"class":    "user_prompt",
					"required": true,
					"target": map[string]string{
						"type": "env",
						"name": "API_KEY",
					},
					"prompt": map[string]string{
						"label":       "API key",
						"description": "Provide API key",
					},
				},
			},
		})
	}))
	defer sm.Close()
	t.Setenv("SECRETS_MANAGER_URL", sm.URL)

	manifest := validBundleTestManifest()
	manifest.IPC.Port = 47710

	payload := map[string]interface{}{
		"scenario": "picker-wheel",
		"tier":     "tier-2-desktop",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var merged Manifest
	if err := json.NewDecoder(rec.Body).Decode(&merged); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(merged.Secrets) != 1 || merged.Secrets[0].ID != "API_KEY" {
		t.Fatalf("expected merged secrets from secrets-manager, got %+v", merged.Secrets)
	}
	_ = os.Unsetenv("SECRETS_MANAGER_URL")
}

func TestHandleMergeBundleSecretsInvalidJSON(t *testing.T) {
	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", strings.NewReader("not valid json{"))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid JSON") {
		t.Errorf("expected error about invalid JSON, got: %s", rec.Body.String())
	}
}

func TestHandleMergeBundleSecretsMissingScenario(t *testing.T) {
	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()

	payload := map[string]interface{}{
		"scenario": "",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "scenario is required") {
		t.Errorf("expected error about scenario required, got: %s", rec.Body.String())
	}
}

func TestHandleMergeBundleSecretsSecretsManagerUnavailable(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_URL", "http://localhost:59997")

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()

	payload := map[string]interface{}{
		"scenario": "test-app",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMergeBundleSecretsDefaultTier(t *testing.T) {
	sm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{},
		})
	}))
	defer sm.Close()
	t.Setenv("SECRETS_MANAGER_URL", sm.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()

	// Request without tier - should default to tier-2-desktop
	payload := map[string]interface{}{
		"scenario": "test-app",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMergeBundleSecretsMultipleSecrets(t *testing.T) {
	sm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "API_KEY",
					"class":    "user_prompt",
					"required": true,
					"target":   map[string]string{"type": "env", "name": "API_KEY"},
				},
				{
					"id":       "DB_PASSWORD",
					"class":    "per_install_generated",
					"required": true,
					"target":   map[string]string{"type": "env", "name": "DB_PASSWORD"},
				},
				{
					"id":       "OPTIONAL_TOKEN",
					"class":    "user_prompt",
					"required": false,
					"target":   map[string]string{"type": "env", "name": "OPTIONAL_TOKEN"},
				},
			},
		})
	}))
	defer sm.Close()
	t.Setenv("SECRETS_MANAGER_URL", sm.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()

	payload := map[string]interface{}{
		"scenario": "test-app",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var merged Manifest
	if err := json.NewDecoder(rec.Body).Decode(&merged); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(merged.Secrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(merged.Secrets))
	}

	// Verify all secrets are present
	ids := make(map[string]bool)
	for _, s := range merged.Secrets {
		ids[s.ID] = true
	}
	if !ids["API_KEY"] || !ids["DB_PASSWORD"] || !ids["OPTIONAL_TOKEN"] {
		t.Errorf("missing expected secrets: %+v", ids)
	}
}

func TestHandleMergeBundleSecretsPreservesManifestFields(t *testing.T) {
	sm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "SECRET",
					"class":    "user_prompt",
					"required": true,
					"target":   map[string]string{"type": "env", "name": "SECRET"},
				},
			},
		})
	}))
	defer sm.Close()
	t.Setenv("SECRETS_MANAGER_URL", sm.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()
	manifest.App = ManifestApp{Name: "My App", Version: "2.5.0", Description: "Test application"}
	manifest.IPC.Port = 48888
	manifest.Telemetry.File = "custom-telemetry.jsonl"
	manifest.Ports = &ManifestPorts{DefaultRange: PortRange{Min: 50000, Max: 50999}, Reserved: []int{50500}}
	manifest.Services[0].Description = "Main API service"
	manifest.Services[0].Health = HealthCheck{Type: "http", Path: "/health", PortName: "http", Interval: 5000, Timeout: 3000, Retries: 5}
	manifest.Services[0].Readiness = ReadinessCheck{Type: "health_success", PortName: "http", Timeout: 10000}

	payload := map[string]interface{}{
		"scenario": "test-app",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var merged Manifest
	if err := json.NewDecoder(rec.Body).Decode(&merged); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Verify original fields are preserved
	if merged.App.Name != "My App" {
		t.Errorf("App.Name changed: got %s", merged.App.Name)
	}
	if merged.App.Version != "2.5.0" {
		t.Errorf("App.Version changed: got %s", merged.App.Version)
	}
	if merged.IPC.Port != 48888 {
		t.Errorf("IPC.Port changed: got %d", merged.IPC.Port)
	}
	if merged.Telemetry.File != "custom-telemetry.jsonl" {
		t.Errorf("Telemetry.File changed: got %s", merged.Telemetry.File)
	}
	if len(merged.Services) != 1 || merged.Services[0].ID != "api" {
		t.Error("Services changed unexpectedly")
	}
	if merged.Services[0].Health.Interval != 5000 {
		t.Errorf("Service health interval changed: got %d", merged.Services[0].Health.Interval)
	}
}

func TestHandleMergeBundleSecretsEmptySecrets(t *testing.T) {
	sm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{},
		})
	}))
	defer sm.Close()
	t.Setenv("SECRETS_MANAGER_URL", sm.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	manifest := validBundleTestManifest()

	payload := map[string]interface{}{
		"scenario": "no-secrets-app",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.MergeBundleSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var merged Manifest
	if err := json.NewDecoder(rec.Body).Decode(&merged); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should have no secrets but still be valid
	if len(merged.Secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(merged.Secrets))
	}
}
