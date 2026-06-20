package scenariocli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

func sampleFreshnessReport() lifecycle.FreshnessReport {
	return lifecycle.FreshnessReport{
		Scenario: "demo",
		Stale:    true,
		Checks: []lifecycle.FreshnessCheckResult{
			{CheckType: "binaries", Target: "api/demo-api", Stale: true, Cause: "content changed", File: "api/main.go"},
			{CheckType: "ui-bundle", Target: "ui/dist/index.html", Stale: false},
		},
		Dependencies: []lifecycle.FreshnessDependencyPolicy{
			{Name: "auth", Policy: "restart_when_stale"},
		},
	}
}

func TestParseFreshnessRequest(t *testing.T) {
	req, err := ParseFreshnessRequest(false, []string{"demo", "--explain", "--path", "/x"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Name != "demo" || req.Path != "/x" || !req.Explain || req.JSON {
		t.Fatalf("unexpected request: %+v", req)
	}
	if _, err := ParseFreshnessRequest(false, nil); err == nil {
		t.Fatal("expected error when no scenario name supplied")
	}
	req, err = ParseFreshnessRequest(true, []string{"demo"})
	if err != nil || !req.JSON {
		t.Fatalf("global --json should propagate: %+v err=%v", req, err)
	}
}

func TestWriteScenarioFreshnessJSONShape(t *testing.T) {
	var buf bytes.Buffer
	if err := writeScenarioFreshnessJSON(&buf, sampleFreshnessReport()); err != nil {
		t.Fatalf("write: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, buf.String())
	}
	// snake_case + EmitUnpopulated: top-level keys present.
	for _, key := range []string{"success", "scenario", "stale", "checks", "dependencies"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, buf.String())
		}
	}
	checks, ok := decoded["checks"].([]any)
	if !ok || len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %v", decoded["checks"])
	}
	first := checks[0].(map[string]any)
	if first["check_type"] != "binaries" || first["file"] != "api/main.go" {
		t.Fatalf("unexpected first check: %v", first)
	}
}

func TestRenderFreshnessResponseHuman(t *testing.T) {
	// Without --explain: overall verdict + stale checks only (fresh checks hidden).
	var summary bytes.Buffer
	if err := RenderFreshnessResponse(&summary, cliout.FormatHuman, FreshnessResponse{Report: sampleFreshnessReport()}); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := summary.String()
	if !strings.Contains(out, "demo is stale") {
		t.Fatalf("missing overall verdict: %q", out)
	}
	if !strings.Contains(out, "api/demo-api") || !strings.Contains(out, "content changed") {
		t.Fatalf("stale check should name cause+file: %q", out)
	}
	if strings.Contains(out, "ui/dist/index.html") {
		t.Fatalf("fresh check must be hidden without --explain: %q", out)
	}
	if strings.Contains(out, "auth") {
		t.Fatalf("dependency policies must be hidden without --explain: %q", out)
	}

	// With --explain: every check + dependency policies.
	var explain bytes.Buffer
	if err := RenderFreshnessResponse(&explain, cliout.FormatHuman, FreshnessResponse{Report: sampleFreshnessReport(), Explain: true}); err != nil {
		t.Fatalf("render explain: %v", err)
	}
	full := explain.String()
	if !strings.Contains(full, "ui/dist/index.html") || !strings.Contains(full, "auth: restart_when_stale") {
		t.Fatalf("--explain should show fresh checks + dep policies: %q", full)
	}
}
