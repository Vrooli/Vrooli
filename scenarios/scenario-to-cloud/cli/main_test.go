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
	// [REQ:STC-P0-001] manifest validation should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	analyzerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestPlanPostsToPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-007] plan generation should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

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
	// Check for pretty-printed output format
	if !strings.Contains(output, "Deployment Plan") || !strings.Contains(output, "VPS Preflight") {
		t.Fatalf("expected plan output with steps, got: %s", output)
	}
}

func TestBundleBuildPostsToBundleBuildEndpoint(t *testing.T) {
	// [REQ:STC-P0-002] bundle build should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

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

func TestPreflightPostsToPreflightEndpoint(t *testing.T) {
	// [REQ:STC-P0-003] preflight should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"preflight", manifestPath}); err != nil {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"vps-inspect-plan", manifestPath}); err != nil {
			t.Fatalf("vps-inspect-plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"scenario_status\"") {
		t.Fatalf("expected inspect plan output, got: %s", output)
	}
}

func TestDeploymentHealthAcceptsJSONFlagBeforeOrAfterID(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/dep-123/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"ok": true,
			"health": "degraded",
			"deployment_id": "dep-123",
			"deployment_name": "demo",
			"scenario_id": "landing-page-business-suite",
			"summary": "1 passed  |  0 warning  |  0 failed",
			"sections": [],
			"duration_ms": 1234,
			"timestamp": "2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	outputBefore := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "health", "--json", "dep-123"}); err != nil {
			t.Fatalf("deployment health with --json before id failed: %v", err)
		}
	})
	if !strings.Contains(outputBefore, `"deployment_id": "dep-123"`) {
		t.Fatalf("expected deployment JSON output, got: %s", outputBefore)
	}

	outputAfter := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "health", "dep-123", "--json"}); err != nil {
			t.Fatalf("deployment health with --json after id failed: %v", err)
		}
	})
	if !strings.Contains(outputAfter, `"deployment_id": "dep-123"`) {
		t.Fatalf("expected deployment JSON output, got: %s", outputAfter)
	}
}

func TestDeploymentHealthPrintsFreshnessNotes(t *testing.T) {
	app := newTestApp(t)
	note := "Scenario version not detected from service.json or ui/package.json; falling back to bundle SHA comparison"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments/dep-456/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"ok": true,
			"health": "degraded",
			"deployment_id": "dep-456",
			"deployment_name": "demo",
			"scenario_id": "landing-page-business-suite",
			"summary": "1 passed  |  1 warning  |  0 failed",
			"sections": [],
			"freshness": {
				"status": "outdated",
				"summary": "Deployment is healthy but outdated relative to local scenario state",
				"version_status": "unknown",
				"fingerprint_status": "outdated",
				"version_source": "default",
				"notes": [%q]
			},
			"duration_ms": 1234,
			"timestamp": "2026-02-07T00:00:00Z"
		}`, note)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "health", "dep-456"}); err != nil {
			t.Fatalf("deployment health failed: %v", err)
		}
	})
	if !strings.Contains(output, note) {
		t.Fatalf("expected freshness note in output, got: %s", output)
	}
}

func TestDeploymentResolveByHostSelector(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("scenario_id"); got != "" {
			t.Fatalf("expected no scenario_id filter, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"deployments": [
				{
					"id": "dep-old",
					"name": "old",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "app-old.example.com",
					"host": "203.0.113.10",
					"progress_percent": 100,
					"created_at": "2026-02-01T00:00:00Z"
				},
				{
					"id": "dep-new",
					"name": "new",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "app.example.com",
					"host": "203.0.113.10",
					"progress_percent": 100,
					"created_at": "2026-02-07T00:00:00Z"
				}
			],
			"timestamp": "2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "resolve", "--host", "203.0.113.10"}); err != nil {
			t.Fatalf("deployment resolve failed: %v", err)
		}
	})

	if !strings.Contains(output, "Resolved deployment: dep-new") {
		t.Fatalf("expected latest deployment id in output, got: %s", output)
	}
}

func TestDeploymentResolveByDomainSelectorWithoutHost(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"deployments": [
				{
					"id": "dep-1",
					"name": "by-domain",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "app.example.com",
					"host": "203.0.113.10",
					"progress_percent": 100,
					"created_at": "2026-02-07T00:00:00Z"
				}
			],
			"timestamp": "2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "resolve", "--domain", "app.example.com"}); err != nil {
			t.Fatalf("deployment resolve failed: %v", err)
		}
	})

	if !strings.Contains(output, "Resolved deployment: dep-1") {
		t.Fatalf("expected deployment id in output, got: %s", output)
	}
}

func TestDeploymentResolveByTargetPrefersDomainMatchBeforeHost(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"deployments": [
				{
					"id": "dep-host",
					"name": "host-match",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "other.example.com",
					"host": "app.example.com",
					"progress_percent": 100,
					"created_at": "2026-02-07T00:00:00Z"
				},
				{
					"id": "dep-domain",
					"name": "domain-match",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "app.example.com",
					"host": "203.0.113.10",
					"progress_percent": 100,
					"created_at": "2026-02-06T00:00:00Z"
				}
			],
			"timestamp": "2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "resolve", "--target", "app.example.com"}); err != nil {
			t.Fatalf("deployment resolve failed: %v", err)
		}
	})

	if !strings.Contains(output, "Resolved deployment: dep-domain") {
		t.Fatalf("expected domain-priority match, got: %s", output)
	}
}

func TestDeploymentHealthResolvesByHostAndScenario(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			if got := r.URL.Query().Get("scenario_id"); got != "landing-page-business-suite" {
				t.Fatalf("expected scenario_id filter, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"deployments": [
					{
						"id": "dep-789",
						"name": "prod",
						"scenario_id": "landing-page-business-suite",
						"status": "deployed",
						"domain": "app.example.com",
						"host": "203.0.113.10",
						"progress_percent": 100,
						"created_at": "2026-02-07T00:00:00Z"
					}
				],
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case "/api/v1/deployments/dep-789/health":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"ok": true,
				"health": "healthy",
				"deployment_id": "dep-789",
				"deployment_name": "prod",
				"scenario_id": "landing-page-business-suite",
				"summary": "4 passed  |  0 warning  |  0 failed",
				"sections": [],
				"duration_ms": 200,
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "health", "--host", "203.0.113.10", "--scenario", "landing-page-business-suite"}); err != nil {
			t.Fatalf("deployment health failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deployment ID: dep-789") {
		t.Fatalf("expected full deployment ID in output, got: %s", output)
	}
}

func TestDeploymentHealthResolvesByTargetDomain(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"deployments": [
					{
						"id": "dep-900",
						"name": "prod",
						"scenario_id": "landing-page-business-suite",
						"status": "deployed",
						"domain": "vrooli.com",
						"host": "138.197.95.182",
						"progress_percent": 100,
						"created_at": "2026-02-07T00:00:00Z"
					}
				],
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case "/api/v1/deployments/dep-900/health":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"ok": true,
				"health": "healthy",
				"deployment_id": "dep-900",
				"deployment_name": "prod",
				"scenario_id": "landing-page-business-suite",
				"summary": "4 passed  |  0 warning  |  0 failed",
				"sections": [],
				"duration_ms": 200,
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "health", "--target", "vrooli.com"}); err != nil {
			t.Fatalf("deployment health failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deployment ID: dep-900") {
		t.Fatalf("expected target resolution to health check by id, got: %s", output)
	}
}

func TestDeploymentCreateAcceptsJSONFlagAfterManifestPath(t *testing.T) {
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"deployment": {
				"id": "dep-create-1",
				"name": "demo",
				"scenario_id": "landing-page-business-suite",
				"status": "pending",
				"created_at": "2026-02-07T00:00:00Z",
				"updated_at": "2026-02-07T00:00:00Z"
			},
			"updated": false,
			"timestamp": "2026-02-07T00:00:00Z"
		}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "create", manifestPath, "--json"}); err != nil {
			t.Fatalf("deployment create failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id": "dep-create-1"`) {
		t.Fatalf("expected JSON response output, got: %s", output)
	}
}

func TestRedeployAcceptsJSONFlagAfterManifestPath(t *testing.T) {
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		w.Header().Set("Content-Type", "application/json")
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments" {
				t.Fatalf("unexpected create request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{
				"deployment": {
					"id": "dep-redeploy-1",
					"name": "demo",
					"scenario_id": "landing-page-business-suite",
					"status": "pending",
					"created_at": "2026-02-07T00:00:00Z",
					"updated_at": "2026-02-07T00:00:00Z"
				},
				"updated": false,
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments/dep-redeploy-1/execute" {
				t.Fatalf("unexpected execute request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{
				"run_id": "run-1",
				"status": "started",
				"message": "ok",
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"redeploy", manifestPath, "--json"}); err != nil {
			t.Fatalf("redeploy failed: %v", err)
		}
	})

	if !strings.Contains(output, `"dep-redeploy-1"`) {
		t.Fatalf("expected deployment id in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"run_id": "run-1"`) {
		t.Fatalf("expected execute run id in JSON output, got: %s", output)
	}
}

func TestRedeploySelectorModeIfNeededExecutesExistingDeployment(t *testing.T) {
	app := newTestApp(t)

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments":
			if got := r.URL.Query().Get("scenario_id"); got != "landing-page-business-suite" {
				t.Fatalf("expected scenario_id query, got: %q", got)
			}
			fmt.Fprint(w, `{
				"deployments": [{
					"id": "dep-selector-1",
					"name": "demo",
					"scenario_id": "landing-page-business-suite",
					"status": "failed",
					"domain": "vrooli.com",
					"host": "138.197.95.182",
					"created_at": "2026-02-07T00:00:00Z"
				}],
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-selector-1/health":
			fmt.Fprint(w, `{
				"ok": true,
				"health": "failed",
				"deployment_id": "dep-selector-1",
				"deployment_name": "demo",
				"scenario_id": "landing-page-business-suite",
				"freshness": {"status":"outdated"},
				"summary": "failed",
				"sections": [],
				"duration_ms": 1,
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/deployments/dep-selector-1/execute":
			bodyBytes, _ := io.ReadAll(r.Body)
			body := string(bodyBytes)
			if !strings.Contains(body, `"run_preflight":true`) {
				t.Fatalf("expected run_preflight true, got: %s", body)
			}
			if !strings.Contains(body, `"force_bundle_build":true`) {
				t.Fatalf("expected force_bundle_build true for outdated deployment, got: %s", body)
			}
			fmt.Fprint(w, `{
				"run_id": "run-selector-1",
				"status": "started",
				"message": "ok",
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{
			"redeploy",
			"--domain", "vrooli.com",
			"--scenario", "landing-page-business-suite",
			"--if-needed",
			"--preflight",
		}); err != nil {
			t.Fatalf("selector redeploy failed: %v", err)
		}
	})

	if !strings.Contains(output, "Found existing deployment by selector: dep-selector-1") {
		t.Fatalf("expected selector match output, got: %s", output)
	}
	if !strings.Contains(output, "run-selector-1") {
		t.Fatalf("expected run id output, got: %s", output)
	}
}

func TestRedeploySelectorModeIfNeededWaitJSONOutputsStructuredResult(t *testing.T) {
	app := newTestApp(t)

	getCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments":
			if got := r.URL.Query().Get("scenario_id"); got != "landing-page-business-suite" {
				t.Fatalf("expected scenario_id query, got: %q", got)
			}
			fmt.Fprint(w, `{
				"deployments": [{
					"id": "dep-selector-json-1",
					"name": "demo",
					"scenario_id": "landing-page-business-suite",
					"status": "failed",
					"domain": "vrooli.com",
					"host": "138.197.95.182",
					"created_at": "2026-02-07T00:00:00Z"
				}],
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-selector-json-1/health":
			fmt.Fprint(w, `{
				"ok": true,
				"health": "failed",
				"deployment_id": "dep-selector-json-1",
				"deployment_name": "demo",
				"scenario_id": "landing-page-business-suite",
				"freshness": {"status":"outdated"},
				"summary": "failed",
				"sections": [],
				"duration_ms": 1,
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/deployments/dep-selector-json-1/execute":
			fmt.Fprint(w, `{
				"run_id": "run-selector-json-1",
				"status": "started",
				"message": "ok",
				"timestamp": "2026-02-07T00:00:00Z"
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-selector-json-1":
			getCount++
			if getCount == 1 {
				fmt.Fprint(w, `{
					"deployment": {
						"id": "dep-selector-json-1",
						"name": "demo",
						"scenario_id": "landing-page-business-suite",
						"status": "deploying",
						"progress_step": "bundle_build",
						"progress_percent": 42,
						"created_at": "2026-02-07T00:00:00Z",
						"updated_at": "2026-02-07T00:00:01Z"
					},
					"timestamp": "2026-02-07T00:00:01Z"
				}`)
				return
			}
			fmt.Fprint(w, `{
				"deployment": {
					"id": "dep-selector-json-1",
					"name": "demo",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"progress_step": "verify",
					"progress_percent": 100,
					"created_at": "2026-02-07T00:00:00Z",
					"updated_at": "2026-02-07T00:00:03Z"
				},
				"timestamp": "2026-02-07T00:00:03Z"
			}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{
			"redeploy",
			"--domain", "vrooli.com",
			"--scenario", "landing-page-business-suite",
			"--if-needed",
			"--preflight",
			"--wait",
			"--json",
		}); err != nil {
			t.Fatalf("selector redeploy failed: %v", err)
		}
	})

	if !strings.Contains(output, `"mode": "selector_if_needed"`) {
		t.Fatalf("expected selector_if_needed mode in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"run_id": "run-selector-json-1"`) {
		t.Fatalf("expected run id in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"final_status": "deployed"`) {
		t.Fatalf("expected wait summary final status in JSON output, got: %s", output)
	}
	if strings.Contains(output, "Waiting for deployment to complete") {
		t.Fatalf("expected JSON-only output, got: %s", output)
	}
}

func TestRedeploySelectorModeRequiresIfNeeded(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	err := app.Run([]string{
		"redeploy",
		"--domain", "vrooli.com",
		"--scenario", "landing-page-business-suite",
	})
	if err == nil {
		t.Fatalf("expected selector mode to require --if-needed")
	}
	if !strings.Contains(err.Error(), "selector mode requires --if-needed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRedeploySelectorModeMissingDeploymentReturnsManifestGuidance(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"deployments":[],"timestamp":"2026-02-07T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	err := app.Run([]string{
		"redeploy",
		"--domain", "vrooli.com",
		"--scenario", "landing-page-business-suite",
		"--if-needed",
	})
	if err == nil {
		t.Fatalf("expected no deployment error")
	}
	if !strings.Contains(err.Error(), "no deployment found for selector") {
		t.Fatalf("expected selector error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "manifest.prod.json") {
		t.Fatalf("expected manifest.prod.json guidance, got: %v", err)
	}
}

func TestVPSInspectApplyPostsToInspectApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-006] inspect apply should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"vps-inspect-apply", manifestPath}); err != nil {
			t.Fatalf("vps-inspect-apply failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected inspect apply output, got: %s", output)
	}
}

func TestInspectMetricsGetsMetricsDebugEndpoint(t *testing.T) {
	app := newTestApp(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"vps-setup-plan", manifestPath, bundlePath}); err != nil {
			t.Fatalf("vps-setup-plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"remote_tar_path\"") {
		t.Fatalf("expected setup plan output, got: %s", output)
	}
}

func TestVPSSetupApplyPostsToSetupApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-004] setup apply should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)
	bundlePath := writeTempFile(t, "mini-vrooli.tar.gz", "not-a-real-tarball")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/vps/setup/apply" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), "\"bundle_path\"") || !strings.Contains(string(bodyBytes), bundlePath) {
			t.Fatalf("expected bundle_path in request body, got: %s", string(bodyBytes))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":{"ok":true,"steps":[]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps-setup-apply", manifestPath, bundlePath}); err != nil {
			t.Fatalf("vps-setup-apply failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected setup apply output, got: %s", output)
	}
}

func TestVPSDeployPlanPostsToDeployPlanEndpoint(t *testing.T) {
	// [REQ:STC-P0-005] deploy plan should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := app.Run([]string{"vps-deploy-plan", manifestPath}); err != nil {
			t.Fatalf("vps-deploy-plan failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"caddy_install\"") {
		t.Fatalf("expected deploy plan output, got: %s", output)
	}
}

func TestVPSDeployApplyPostsToDeployApplyEndpoint(t *testing.T) {
	// [REQ:STC-P0-005] deploy apply should be callable via CLI (integration layer)
	app := newTestApp(t)
	manifestPath := writeTestManifest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if !strings.Contains(string(bodyBytes), "\"manifest\"") {
			t.Fatalf("expected manifest wrapper in request body, got: %s", string(bodyBytes))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":{"ok":true,"steps":[]},"timestamp":"2025-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", server.URL)

	output := captureStdout(t, func() {
		if err := app.Run([]string{"vps-deploy-apply", manifestPath}); err != nil {
			t.Fatalf("vps-deploy-apply failed: %v", err)
		}
	})
	if !strings.Contains(output, "\"ok\": true") {
		t.Fatalf("expected deploy apply output, got: %s", output)
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
