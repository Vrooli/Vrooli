package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	scenariocmd "scenario-to-cloud/cli/scenario"
)

// testManifestJSON returns a valid CloudManifest JSON for testing.
// This eliminates duplication across all CLI tests that need a manifest file.
const testManifestJSON = `{
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
}`

// writeTestManifest creates a temporary manifest file with valid test content.
func writeTestManifest(t *testing.T) string {
	t.Helper()
	return writeTempFile(t, "cloud-manifest.json", testManifestJSON)
}

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

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
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
	// [REQ:STC-P0-001] manifest validation should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"manifest", "validate", manifestPath}); err != nil {
			t.Fatalf("manifest validate failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"valid\": true") {
		t.Fatalf("expected validate output, got: %s", output)
	}
}

func TestManifestSchemaGetsSchemaEndpoint(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/manifest/schema" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"schema":{"type":"object"},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"manifest", "schema"}); err != nil {
			t.Fatalf("manifest schema failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"schema\"") {
		t.Fatalf("expected schema output, got: %s", output)
	}
}

func TestManifestInitPostsToInitEndpoint(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/manifest/init" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"manifest":{"version":"1.0.0"},"issues":[],"source":"template","timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"manifest", "init", "--scenario", "landing-page-business-suite"}); err != nil {
			t.Fatalf("manifest init failed: %v", err)
		}
	})
	if !strings.Contains(output, "Initialized manifest") {
		t.Fatalf("expected init output, got: %s", output)
	}
}

func TestScenarioDepsPrintsResourcesFromCurrentAPIShape(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/scenarios/landing-page-business-suite/dependencies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
		  "scenario_id": "landing-page-business-suite",
		  "resources": ["postgres"],
		  "scenarios": null,
		  "analyzer_available": true,
		  "source": "analyzer",
		  "timestamp": "2026-02-08T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"scenario", "deps", "landing-page-business-suite"}); err != nil {
			t.Fatalf("scenario deps failed: %v", err)
		}
	})
	if strings.Contains(output, "No dependencies.") {
		t.Fatalf("expected dependencies to be shown, got: %s", output)
	}
	if !strings.Contains(output, "resource") || !strings.Contains(output, "postgres") {
		t.Fatalf("expected postgres resource in output, got: %s", output)
	}
}

func TestScenarioDepsImpactShowsSummaryAndPerDependencyRows(t *testing.T) {
	app := newTestApp(t)

	apiServer := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/scenarios/landing-page-business-suite/dependencies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
		  "scenario_id": "landing-page-business-suite",
		  "resources": ["postgres"],
		  "scenarios": ["auth-service"],
		  "analyzer_available": true,
		  "source": "analyzer",
		  "timestamp": "2026-02-08T00:00:00Z"
		}`)
	}))
	defer apiServer.Close()

	analyzerServer := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/scenarios/landing-page-business-suite/deployment") {
			t.Fatalf("unexpected analyzer path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
		  "scenario": "landing-page-business-suite",
		  "dependencies": [
		    {
		      "name": "postgres",
		      "type": "resource",
		      "required": true,
		      "requirements": {"ram_mb": 768, "disk_mb": 4096, "cpu_cores": 1}
		    },
		    {
		      "name": "auth-service",
		      "type": "scenario",
		      "required": true,
		      "requirements": {"ram_mb": 256, "disk_mb": 512, "cpu_cores": 0.5},
		      "children": [
		        {
		          "name": "redis",
		          "type": "resource",
		          "required": true,
		          "requirements": {"ram_mb": 256, "disk_mb": 256, "cpu_cores": 0.5}
		        }
		      ]
		    }
		  ],
		  "aggregates": {
		    "tier-4-saas": {"estimated_requirements": {"ram_mb": 1280, "disk_mb": 4864, "cpu_cores": 2}}
		  },
		  "metadata_gaps": {"total_gaps": 0}
		}`)
	}))
	defer analyzerServer.Close()

	origResolve := scenariocmd.ResolveAnalyzerBaseURLForTest()
	scenariocmd.SetResolveAnalyzerBaseURLForTest(func(ctx context.Context) (string, error) {
		_ = ctx
		return analyzerServer.URL, nil
	})
	t.Cleanup(func() {
		scenariocmd.SetResolveAnalyzerBaseURLForTest(origResolve)
	})

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", apiServer.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"scenario", "deps", "landing-page-business-suite", "--impact", "--verbose"}); err != nil {
			t.Fatalf("scenario deps --impact failed: %v", err)
		}
	})
	if !strings.Contains(output, "Impact Summary (tier-4-saas): RAM 1280 MB") {
		t.Fatalf("expected impact summary in output, got: %s", output)
	}
	if !strings.Contains(output, "Coverage: 3/3 dependencies") {
		t.Fatalf("expected coverage summary in output, got: %s", output)
	}
	if !strings.Contains(output, "transitive") || !strings.Contains(output, "redis") {
		t.Fatalf("expected transitive redis row in output, got: %s", output)
	}
}

func TestBundleBuildPostsToBundleBuildEndpoint(t *testing.T) {
	// [REQ:STC-P0-002] bundle build should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"bundle", "build", manifestPath}); err != nil {
			t.Fatalf("bundle build failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"artifact\"") {
		t.Fatalf("expected bundle build output, got: %s", output)
	}
}

func TestPreflightPostsToPreflightEndpoint(t *testing.T) {
	// [REQ:STC-P0-003] preflight should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/preflight" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"checks":[],"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"preflight", "run", manifestPath}); err != nil {
			t.Fatalf("preflight failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected preflight output, got: %s", output)
	}
}

func TestVPSInspectPlanPostsToInspectPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-006] inspect plan should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/inspect/plan" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"plan":{"commands":[{"id":"scenario_status"}]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"inspect", "plan", manifestPath}); err != nil {
			t.Fatalf("inspect plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"scenario_status\"") {
		t.Fatalf("expected inspect plan output, got: %s", output)
	}
}

func TestVPSInspectApplyPostsToInspectApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-006] inspect apply should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/inspect/apply" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":{"ok":true,"steps":[]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"inspect", "status", manifestPath}); err != nil {
			t.Fatalf("inspect status failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected inspect apply output, got: %s", output)
	}
}

func TestInspectMetricsGetsMetricsDebugEndpoint(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/dep-123/metrics-debug" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"deployment_id": "dep-123",
			"result": {
				"ok": true,
				"collector": "linux",
				"os_id": "ubuntu",
				"os_version": "24.04",
				"commands": [{"id":"meminfo","command":"cat /proc/meminfo","exit_code":0,"duration_ms":3}],
				"system": {
					"cpu": {"cores":4,"usage_percent":12.5,"load_average":[0.1,0.2,0.3]},
					"memory": {"total_mb":1000,"used_mb":400,"free_mb":600,"usage_percent":40},
					"disk": {"total_gb":100,"used_gb":40,"free_gb":60,"usage_percent":40},
					"swap": {"total_mb":0,"used_mb":0,"usage_percent":0},
					"ssh": {"connected":true,"latency_ms":10,"key_in_auth":true,"key_path":"~/.ssh/id_ed25519"},
					"uptime_seconds": 100
				},
				"timestamp":"2026-02-07T00:00:00Z"
			},
			"timestamp":"2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"inspect", "metrics", "dep-123", "--json"}); err != nil {
			t.Fatalf("inspect metrics failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"deployment_id\": \"dep-123\"") {
		t.Fatalf("expected metrics JSON output, got: %s", output)
	}
}

func TestVPSSetupPlanPostsToSetupPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-004] setup plan should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)
	bundlePath := writeTempFile(t, "mini-vrooli.tar.gz", "not-a-real-tarball")

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/setup/plan" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), "\"bundle_path\"") || !strings.Contains(string(bodyBytes), bundlePath) {
			t.Fatalf("expected bundle_path in request body, got: %s", string(bodyBytes))
		}
		if !strings.Contains(string(bodyBytes), "\"manifest\"") || !strings.Contains(string(bodyBytes), "\"version\"") {
			t.Fatalf("expected manifest in request body, got: %s", string(bodyBytes))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"plan":{"remote_tar_path":"/root/Vrooli/.vrooli/cloud/bundles/mini-vrooli.tar.gz","commands":[{"id":"mkdir"}]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps", "setup", "plan", manifestPath, bundlePath}); err != nil {
			t.Fatalf("vps setup plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"remote_tar_path\"") {
		t.Fatalf("expected setup plan output, got: %s", output)
	}
}

func TestVPSSetupApplyPostsToSetupApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-004] setup apply should be callable via CLI (integration layer);
	// it compiles the plan and submits the plan_digest it was shown.
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)
	bundlePath := writeTempFile(t, "mini-vrooli.tar.gz", "not-a-real-tarball")

	var applied string
	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/vps/setup/plan":
			fmt.Fprint(w, `{"plan":{"remote_tar_path":"/tmp/b.tar.gz","commands":[]},"plan_digest":"sha256:setup","timestamp":"2025-01-01T00:00:00Z"}`)
		case "/api/v1/vps/setup/apply":
			applied = string(bodyBytes)
			if !strings.Contains(applied, "\"bundle_path\"") || !strings.Contains(applied, bundlePath) {
				t.Fatalf("expected bundle_path in request body, got: %s", applied)
			}
			fmt.Fprint(w, `{"result":{"ok":true,"steps":[]},"timestamp":"2025-01-01T00:00:00Z"}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps", "setup", "apply", manifestPath, bundlePath}); err != nil {
			t.Fatalf("vps setup apply failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected setup apply output, got: %s", output)
	}
	if !strings.Contains(applied, "\"plan_digest\":\"sha256:setup\"") {
		t.Fatalf("expected the compiled plan_digest in the apply body, got: %s", applied)
	}
}

func TestVPSDeployPlanPostsToDeployPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-005] deploy plan should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/deploy/plan" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), "\"manifest\"") {
			t.Fatalf("expected manifest wrapper in request body, got: %s", string(bodyBytes))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"plan":{"commands":[{"id":"caddy_install"}]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps", "deploy", "plan", manifestPath}); err != nil {
			t.Fatalf("vps deploy plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"caddy_install\"") {
		t.Fatalf("expected deploy plan output, got: %s", output)
	}
}

func TestVPSDeployApplyPostsToDeployApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-005] deploy apply should be callable via CLI (integration layer);
	// --plan-digest applies a previously reviewed plan without recompiling.
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	var applied string
	server := httptest.NewServer(withHealth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/deploy/apply" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		applied = string(bodyBytes)
		if !strings.Contains(applied, "\"manifest\"") {
			t.Fatalf("expected manifest wrapper in request body, got: %s", applied)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":{"ok":true,"steps":[]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps", "deploy", "apply", manifestPath, "--plan-digest", "sha256:reviewed"}); err != nil {
			t.Fatalf("vps deploy apply failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected deploy apply output, got: %s", output)
	}
	if !strings.Contains(applied, "\"plan_digest\":\"sha256:reviewed\"") {
		t.Fatalf("expected the reviewed plan_digest in the apply body, got: %s", applied)
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

// withHealth wraps a test handler so that GET /health responds with a
// healthy payload. The CLI probes /health before every API command, so
// every fake server must answer that probe in addition to its own routes.
func withHealth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		next(w, r)
	}
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
