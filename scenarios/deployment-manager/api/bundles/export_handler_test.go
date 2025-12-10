package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deployment-manager/secrets"
)

func TestHandleExportBundle(t *testing.T) {
	analyzerSkeleton := Manifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App: ManifestApp{
			Name:        "Export Test App",
			Version:     "1.0.0",
			Description: "testing export",
		},
		IPC: ManifestIPC{
			Mode:          "loopback-http",
			Host:          "127.0.0.1",
			Port:          48000,
			AuthTokenPath: "runtime/auth-token",
		},
		Telemetry: ManifestTelemetry{
			File: "telemetry/deployment-telemetry.jsonl",
		},
		Ports: &ManifestPorts{
			DefaultRange: PortRange{Min: 47000, Max: 47999},
			Reserved:     []int{},
		},
		Services: []ServiceEntry{
			{
				ID:          "api",
				Type:        "api-binary",
				Description: "Export Test API",
				Binaries: map[string]ServiceBinary{
					"linux-x64": {Path: "bin/api"},
				},
				LogDir: "logs/api",
				Ports: &ServicePorts{
					Requested: []RequestedPort{
						{Name: "http", Range: PortRange{Min: 48100, Max: 48110}},
					},
				},
				Health: HealthCheck{
					Type:     "tcp",
					PortName: "http",
					Interval: 1000,
					Timeout:  2000,
					Retries:  3,
				},
				Readiness: ReadinessCheck{
					Type:     "port_open",
					PortName: "http",
					Timeout:  5000,
				},
			},
		},
	}

	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "export-test-app",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerSkeleton,
			},
		})
	}))
	defer analyzer.Close()
	analyzerPort := strings.Split(analyzer.Listener.Addr().String(), ":")[1]
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", analyzerPort)

	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "SESSION_KEY",
					"class":    "per_install_generated",
					"required": true,
					"target": map[string]string{
						"type": "env",
						"name": "SESSION_KEY",
					},
				},
			},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader(`{"scenario":"export-test-app","tier":"tier-2-desktop"}`))
	rec := httptest.NewRecorder()

	handler.ExportBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ExportBundleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Verify response structure
	if resp.Status != "exported" {
		t.Errorf("expected status=exported, got %s", resp.Status)
	}
	if resp.Schema != "desktop.v0.1" {
		t.Errorf("expected schema=desktop.v0.1, got %s", resp.Schema)
	}
	if resp.Scenario != "export-test-app" {
		t.Errorf("expected scenario=export-test-app, got %s", resp.Scenario)
	}
	if resp.Tier != "tier-2-desktop" {
		t.Errorf("expected tier=tier-2-desktop, got %s", resp.Tier)
	}
	if resp.Checksum == "" {
		t.Error("expected checksum to be non-empty")
	}
	if resp.GeneratedAt == "" {
		t.Error("expected generated_at to be non-empty")
	}

	// Verify checksum is valid hex
	if len(resp.Checksum) != 64 {
		t.Errorf("expected 64-character SHA256 hex, got %d characters", len(resp.Checksum))
	}
	if _, err := hex.DecodeString(resp.Checksum); err != nil {
		t.Errorf("checksum is not valid hex: %v", err)
	}

	// Verify manifest content
	if resp.Manifest.App.Name != "Export Test App" {
		t.Errorf("expected app name=Export Test App, got %s", resp.Manifest.App.Name)
	}
	if len(resp.Manifest.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(resp.Manifest.Services))
	}
}

func TestHandleExportBundleChecksumConsistency(t *testing.T) {
	analyzerSkeleton := Manifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App:           ManifestApp{Name: "Checksum Test", Version: "1.0.0"},
		IPC:           ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 48000, AuthTokenPath: "runtime/auth-token"},
		Telemetry:     ManifestTelemetry{File: "telemetry.jsonl"},
		Ports:         &ManifestPorts{DefaultRange: PortRange{Min: 47000, Max: 47999}, Reserved: []int{}},
		Services: []ServiceEntry{
			{
				ID:        "api",
				Type:      "api-binary",
				Binaries:  map[string]ServiceBinary{"linux-x64": {Path: "bin/api"}},
				Health:    HealthCheck{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 1000, Retries: 1},
				Readiness: ReadinessCheck{Type: "port_open", PortName: "http", Timeout: 1000},
				Ports:     &ServicePorts{Requested: []RequestedPort{{Name: "http", Range: PortRange{Min: 48000, Max: 48010}}}},
			},
		},
	}

	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "checksum-test",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerSkeleton,
			},
		})
	}))
	defer analyzer.Close()
	analyzerPort := strings.Split(analyzer.Listener.Addr().String(), ":")[1]
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", analyzerPort)

	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader(`{"scenario":"checksum-test","include_secrets":false}`))
	rec := httptest.NewRecorder()

	handler.ExportBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ExportBundleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Verify checksum matches manifest content
	manifestBytes, err := json.Marshal(resp.Manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	expectedHash := sha256.Sum256(manifestBytes)
	expectedChecksum := hex.EncodeToString(expectedHash[:])

	if resp.Checksum != expectedChecksum {
		t.Errorf("checksum mismatch: got %s, expected %s", resp.Checksum, expectedChecksum)
	}
}

func TestHandleExportBundleInvalidJSON(t *testing.T) {
	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader("not valid json{"))
	rec := httptest.NewRecorder()

	handler.ExportBundle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid JSON") {
		t.Errorf("expected error about invalid JSON, got: %s", rec.Body.String())
	}
}

func TestHandleExportBundleMissingScenario(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader(tt.payload))
			rec := httptest.NewRecorder()

			handler.ExportBundle(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "scenario is required") {
				t.Errorf("expected error about scenario required, got: %s", rec.Body.String())
			}
		})
	}
}

func TestHandleExportBundleFiltersInfrastructureSecrets(t *testing.T) {
	analyzerSkeleton := Manifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App:           ManifestApp{Name: "Filter Test", Version: "1.0.0"},
		IPC:           ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 48000, AuthTokenPath: "runtime/auth-token"},
		Telemetry:     ManifestTelemetry{File: "telemetry.jsonl"},
		Ports:         &ManifestPorts{DefaultRange: PortRange{Min: 47000, Max: 47999}, Reserved: []int{}},
		Services: []ServiceEntry{
			{
				ID:        "api",
				Type:      "api-binary",
				Binaries:  map[string]ServiceBinary{"linux-x64": {Path: "bin/api"}},
				Health:    HealthCheck{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 1000, Retries: 1},
				Readiness: ReadinessCheck{Type: "port_open", PortName: "http", Timeout: 1000},
				Ports:     &ServicePorts{Requested: []RequestedPort{{Name: "http", Range: PortRange{Min: 48000, Max: 48010}}}},
			},
		},
	}

	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "filter-test",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": analyzerSkeleton,
			},
		})
	}))
	defer analyzer.Close()
	analyzerPort := strings.Split(analyzer.Listener.Addr().String(), ":")[1]
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", analyzerPort)

	// Return both infrastructure and bundle-safe secrets
	secretsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bundle_secrets": []map[string]interface{}{
				{
					"id":       "DB_ADMIN_PASSWORD",
					"class":    "infrastructure", // Should be FILTERED OUT
					"required": true,
					"target":   map[string]string{"type": "env", "name": "DB_ADMIN_PASSWORD"},
				},
				{
					"id":       "SESSION_SECRET",
					"class":    "per_install_generated", // Should be included
					"required": true,
					"target":   map[string]string{"type": "env", "name": "SESSION_SECRET"},
				},
			},
		})
	}))
	defer secretsServer.Close()
	t.Setenv("SECRETS_MANAGER_URL", secretsServer.URL)

	handler := NewHandler(secrets.NewClient(), nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader(`{"scenario":"filter-test"}`))
	rec := httptest.NewRecorder()

	handler.ExportBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ExportBundleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should only have 1 secret (infrastructure filtered out)
	if len(resp.Manifest.Secrets) != 1 {
		t.Fatalf("expected 1 secret (infrastructure filtered), got %d", len(resp.Manifest.Secrets))
	}

	// Verify the included secret is the bundle-safe one
	if resp.Manifest.Secrets[0].ID != "SESSION_SECRET" {
		t.Errorf("expected SESSION_SECRET, got %s", resp.Manifest.Secrets[0].ID)
	}

	// Verify no infrastructure secrets made it through
	for _, s := range resp.Manifest.Secrets {
		if s.Class == "infrastructure" {
			t.Error("infrastructure secret should not be in exported bundle")
		}
	}
}

func TestHandleExportBundleDefaultTier(t *testing.T) {
	analyzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"scenario":  "tier-test",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"manifest": map[string]interface{}{
				"skeleton": Manifest{
					SchemaVersion: "v0.1",
					Target:        "desktop",
					App:           ManifestApp{Name: "Tier Test", Version: "1.0.0"},
					IPC:           ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 48000, AuthTokenPath: "runtime/auth-token"},
					Telemetry:     ManifestTelemetry{File: "telemetry.jsonl"},
					Ports:         &ManifestPorts{DefaultRange: PortRange{Min: 47000, Max: 47999}, Reserved: []int{}},
					Services: []ServiceEntry{
						{
							ID:        "api",
							Type:      "api-binary",
							Binaries:  map[string]ServiceBinary{"linux-x64": {Path: "bin/api"}},
							Health:    HealthCheck{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 1000, Retries: 1},
							Readiness: ReadinessCheck{Type: "port_open", PortName: "http", Timeout: 1000},
							Ports:     &ServicePorts{Requested: []RequestedPort{{Name: "http", Range: PortRange{Min: 48000, Max: 48010}}}},
						},
					},
				},
			},
		})
	}))
	defer analyzer.Close()
	analyzerPort := strings.Split(analyzer.Listener.Addr().String(), ":")[1]
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", analyzerPort)

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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/export", strings.NewReader(`{"scenario":"tier-test"}`))
	rec := httptest.NewRecorder()

	handler.ExportBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ExportBundleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Tier != "tier-2-desktop" {
		t.Errorf("expected tier=tier-2-desktop, got %s", resp.Tier)
	}
}
