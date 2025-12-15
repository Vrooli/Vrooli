package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHelpCommand(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"help"}); err != nil {
			t.Fatalf("help command failed: %v", err)
		}
	})
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected help output to contain Usage, got: %s", output)
	}
	if !strings.Contains(output, "Commands:") {
		t.Fatalf("expected help output to list commands, got: %s", output)
	}
}

func TestVersionCommand(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"version"}); err != nil {
			t.Fatalf("version command failed: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(output), "version") {
		t.Fatalf("expected version output, got: %s", output)
	}
}

func TestConfigureCommand(t *testing.T) {
	app := newTestApp(t)
	apiBase := "http://test.example.com"

	if err := app.Run([]string{"configure", "api_base", apiBase}); err != nil {
		t.Fatalf("configure set failed: %v", err)
	}

	output := captureStdout(t, func() {
		if err := app.Run([]string{"configure"}); err != nil {
			t.Fatalf("configure get failed: %v", err)
		}
	})
	if !strings.Contains(output, apiBase) {
		t.Fatalf("expected configured api_base to be printed, got: %s", output)
	}
}

func TestUnknownCommand(t *testing.T) {
	app := newTestApp(t)
	err := app.Run([]string{"invalid_command"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "Unknown command") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestStatusCallsHealthEndpoint(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"status"}); err != nil {
			t.Fatalf("status failed: %v", err)
		}
	})
	if !strings.Contains(output, "healthy") {
		t.Fatalf("expected status output, got: %s", output)
	}
}

func TestManifestValidatePostsToValidateEndpoint(t *testing.T) {
	app := newTestApp(t)

	// [REQ:STC-P0-001] manifest validation should be callable via CLI (integration layer)
	manifestPath := writeTempFile(t, "cloud-manifest.json", `{
  "version": "1.0.0",
  "target": { "type": "vps", "vps": { "host": "203.0.113.10" } },
  "scenario": { "id": "landing-page-business-suite" },
  "dependencies": {
    "scenarios": ["landing-page-business-suite"],
    "resources": [],
    "analyzer": { "tool": "scenario-dependency-analyzer" }
  },
  "bundle": {
    "include_packages": true,
    "include_autoheal": true,
    "scenarios": ["landing-page-business-suite", "vrooli-autoheal"]
  },
  "ports": { "ui": 3000, "api": 3001, "ws": 3002 },
  "edge": { "domain": "example.com", "caddy": { "enabled": true, "email": "ops@example.com" } }
}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/manifest/validate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"valid":true,"issues":[],"manifest":{"version":"1.0.0"},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"manifest-validate", manifestPath}); err != nil {
			t.Fatalf("manifest validate failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"valid\": true") {
		t.Fatalf("expected validate output, got: %s", output)
	}
}

func TestPlanPostsToPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-007] plan generation should be callable via CLI (integration layer)
	app := newTestApp(t)

	manifestPath := writeTempFile(t, "cloud-manifest.json", `{
  "version": "1.0.0",
  "target": { "type": "vps", "vps": { "host": "203.0.113.10" } },
  "scenario": { "id": "landing-page-business-suite" },
  "dependencies": {
    "scenarios": ["landing-page-business-suite"],
    "resources": [],
    "analyzer": { "tool": "scenario-dependency-analyzer" }
  },
  "bundle": {
    "include_packages": true,
    "include_autoheal": true,
    "scenarios": ["landing-page-business-suite", "vrooli-autoheal"]
  },
  "ports": { "ui": 3000, "api": 3001, "ws": 3002 },
  "edge": { "domain": "example.com", "caddy": { "enabled": true, "email": "ops@example.com" } }
}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/plan" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"plan":[{"id":"preflight","title":"VPS Preflight","description":"..."}],"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"plan", manifestPath}); err != nil {
			t.Fatalf("plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"plan\"") {
		t.Fatalf("expected plan output, got: %s", output)
	}
}

func TestBundleBuildPostsToBundleBuildEndpoint(t *testing.T) {
	// [REQ:STC-P0-002] bundle build should be callable via CLI (integration layer)
	app := newTestApp(t)

	manifestPath := writeTempFile(t, "cloud-manifest.json", `{
  "version": "1.0.0",
  "target": { "type": "vps", "vps": { "host": "203.0.113.10" } },
  "scenario": { "id": "landing-page-business-suite" },
  "dependencies": {
    "scenarios": ["landing-page-business-suite"],
    "resources": [],
    "analyzer": { "tool": "scenario-dependency-analyzer" }
  },
  "bundle": {
    "include_packages": true,
    "include_autoheal": true,
    "scenarios": ["landing-page-business-suite", "vrooli-autoheal"]
  },
  "ports": { "ui": 3000, "api": 3001, "ws": 3002 },
  "edge": { "domain": "example.com", "caddy": { "enabled": true, "email": "ops@example.com" } }
}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/bundle/build" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"artifact":{"path":"/tmp/mini.tar.gz","sha256":"abc","size_bytes":123},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"bundle-build", manifestPath}); err != nil {
			t.Fatalf("bundle-build failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"artifact\"") {
		t.Fatalf("expected bundle-build output, got: %s", output)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("SCENARIO_TO_CLOUD_API_TOKEN", "test-token")
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func writeTempFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := t.TempDir() + "/" + name
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()
	_ = w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
