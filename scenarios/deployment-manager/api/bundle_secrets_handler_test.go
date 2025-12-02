package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
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

	server := &Server{
		config: &Config{Port: "0"},
		router: mux.NewRouter(),
	}
	server.setupRoutes()

	manifest := desktopBundleManifest{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App:           manifestApp{Name: "demo", Version: "1.0.0"},
		IPC:           manifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 47710, AuthTokenPath: "runtime/auth-token"},
		Telemetry:     manifestTelemetry{File: "telemetry.jsonl"},
		Ports: &manifestPorts{
			DefaultRange: manifestPortRange{Min: 47000, Max: 47999},
			Reserved:     []int{},
		},
		Services: []manifestServiceEntry{
			{
				ID:       "api",
				Type:     "api-binary",
				Binaries: map[string]manifestServiceBinary{"linux-x64": {Path: "bin/api"}},
				Health:   manifestHealth{Type: "tcp", PortName: "http", Interval: 1000, Timeout: 1000, Retries: 1},
				Readiness: manifestReadiness{
					Type:     "port_open",
					PortName: "http",
					Timeout:  1000,
				},
				Ports: &manifestServicePorts{
					Requested: []manifestRequestedPort{
						{Name: "http", Range: manifestPortRange{Min: 48000, Max: 48010}},
					},
				},
			},
		},
	}

	payload := map[string]interface{}{
		"scenario": "picker-wheel",
		"tier":     "tier-2-desktop",
		"manifest": manifest,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/merge-secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var merged desktopBundleManifest
	if err := json.NewDecoder(rec.Body).Decode(&merged); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(merged.Secrets) != 1 || merged.Secrets[0].ID != "API_KEY" {
		t.Fatalf("expected merged secrets from secrets-manager, got %+v", merged.Secrets)
	}
	_ = os.Unsetenv("SECRETS_MANAGER_URL")
}
