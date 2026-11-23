package bundles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deployment-manager/secrets"
)

func TestHandleAssembleBundle(t *testing.T) {
	analyzerSkeleton := analyzerTestSkeleton("Demo App", "1.0.0", "demo", "Demo API")
	analyzerSkeleton.Services[0].Ports.Requested[0].Range = PortRange{Min: 48100, Max: 48110}

	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/bundle/manifest") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "demo-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerSkeleton,
			},
		})
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "http://"+analyzer.Listener.Addr().String())

	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "USER_API_KEY",
					"class":    "user_prompt",
					"required": true,
					"target": map[string]string{
						"type": "env",
						"name": "USER_API_KEY",
					},
					"prompt": map[string]string{
						"label":       "API key",
						"description": "Provide your API key",
					},
				},
			},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"demo-app","tier":"tier-2-desktop"}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status   string   `json:"status"`
		Schema   string   `json:"schema"`
		Manifest Manifest `json:"manifest"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "assembled" || resp.Schema != "desktop.v0.1" {
		t.Fatalf("unexpected status/schema: %+v", resp)
	}
	if resp.Manifest.App.Name != "Demo App" {
		t.Fatalf("expected manifest to include app data from analyzer skeleton")
	}
	if len(resp.Manifest.Secrets) != 1 || resp.Manifest.Secrets[0].ID != "USER_API_KEY" {
		t.Fatalf("expected secrets merged from secrets-manager, got %+v", resp.Manifest.Secrets)
	}
	if len(resp.Manifest.Services) != 1 || resp.Manifest.Services[0].ID != "api" {
		t.Fatalf("expected services from skeleton to survive merge, got %+v", resp.Manifest.Services)
	}
}

func TestHandleAssembleBundleInvalidJSON(t *testing.T) {
	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader("not valid json{"))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid JSON") {
		t.Errorf("expected error about invalid JSON, got: %s", rec.Body.String())
	}
}

func TestHandleAssembleBundleMissingScenario(t *testing.T) {
	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	tests := []struct {
		name    string
		payload string
	}{
		{"empty scenario", `{"scenario":"","tier":"tier-2-desktop"}`},
		{"whitespace scenario", `{"scenario":"   ","tier":"tier-2-desktop"}`},
		{"missing scenario field", `{"tier":"tier-2-desktop"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(tt.payload))
			rec := httptest.NewRecorder()

			handler.AssembleBundle(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "scenario is required") {
				t.Errorf("expected error about scenario required, got: %s", rec.Body.String())
			}
		})
	}
}

func TestHandleAssembleBundleAnalyzerUnavailable(t *testing.T) {
	// Point to a port where nothing is listening
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", "59999")

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"test-app"}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAssembleBundleSecretsManagerUnavailable(t *testing.T) {
	// Set up a working analyzer
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "test-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerTestSkeleton("Test", "1.0.0", "", ""),
			},
		})
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "http://"+analyzer.Listener.Addr().String())

	// Point to unavailable secrets manager
	t.Setenv("SECRETS_MANAGER_URL", "http://localhost:59998")

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"test-app"}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("expected error about secrets, got: %s", rec.Body.String())
	}
}

func TestHandleAssembleBundleWithoutSecrets(t *testing.T) {
	// Set up analyzer with valid skeleton
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "test-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerTestSkeleton("Test", "1.0.0", "", ""),
			},
		})
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "http://"+analyzer.Listener.Addr().String())

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	// Request with include_secrets=false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"test-app","include_secrets":false}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Status   string   `json:"status"`
		Manifest Manifest `json:"manifest"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "assembled" {
		t.Errorf("expected status=assembled, got %s", resp.Status)
	}
	// Secrets should be empty since we skipped them
	if len(resp.Manifest.Secrets) != 0 {
		t.Errorf("expected no secrets with include_secrets=false, got %d", len(resp.Manifest.Secrets))
	}
}

func TestHandleAssembleBundleDefaultTier(t *testing.T) {
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "test-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerTestSkeleton("Test", "1.0.0", "", ""),
			},
		})
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "http://"+analyzer.Listener.Addr().String())

	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify default tier is used
		if !strings.Contains(r.URL.String(), "tier-2-desktop") {
			t.Errorf("expected tier-2-desktop in request, got: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	// Request without tier - should default to tier-2-desktop
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"test-app"}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAssembleBundleResponseStructure(t *testing.T) {
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "test-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerTestSkeleton("Test App", "2.0.0", "A test application", ""),
			},
		})
	}))
	defer analyzer.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "http://"+analyzer.Listener.Addr().String())

	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/assemble", strings.NewReader(`{"scenario":"test-app"}`))
	rec := httptest.NewRecorder()

	handler.AssembleBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Verify required response fields
	if resp["status"] != "assembled" {
		t.Errorf("expected status=assembled, got %v", resp["status"])
	}
	if resp["schema"] != "desktop.v0.1" {
		t.Errorf("expected schema=desktop.v0.1, got %v", resp["schema"])
	}
	if resp["manifest"] == nil {
		t.Error("expected manifest in response")
	}

	// Verify manifest structure
	manifest := resp["manifest"].(map[string]interface{})
	app := manifest["app"].(map[string]interface{})
	if app["name"] != "Test App" {
		t.Errorf("expected app.name=Test App, got %v", app["name"])
	}
	if app["version"] != "2.0.0" {
		t.Errorf("expected app.version=2.0.0, got %v", app["version"])
	}
}
