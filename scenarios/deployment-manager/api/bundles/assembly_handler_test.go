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
	analyzerSkeleton := Manifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App: ManifestApp{
			Name:        "Demo App",
			Version:     "1.0.0",
			Description: "demo",
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
				Description: "Demo API",
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
	analyzerPort := strings.Split(analyzer.Listener.Addr().String(), ":")[1]
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT", analyzerPort)

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

	handler := NewHandler(secrets.NewClient(), func(msg string, fields map[string]interface{}) {})

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
