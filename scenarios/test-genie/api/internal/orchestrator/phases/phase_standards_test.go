package phases

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"test-genie/internal/orchestrator/workspace"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestParseAuditorStandardsSummaryParsesValidJSON(t *testing.T) {
	raw := `{
  "summary": {
    "total": 2,
    "by_severity": {"HIGH": 1, "low": 1},
    "by_rule": [{"rule_id":"prd_structure","count":2,"severity":"high","title":"PRD structure"}],
    "highest_severity": "HIGH",
    "top_violations": [{"severity":"high","rule_id":"prd_structure","file_path":"PRD.md","line_number":1,"title":"Bad PRD"}],
    "artifact": {"path":"logs/scenario-auditor/standards/demo.json"},
    "recommended_steps": ["Fix PRD.md"]
  }
}`

	summary, err := parseAuditorStandardsSummary(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("expected total=2, got %d", summary.Total)
	}
	if summary.HighestSeverity != "high" {
		t.Fatalf("expected highest=high, got %q", summary.HighestSeverity)
	}
	if summary.BySeverity["high"] != 1 || summary.BySeverity["low"] != 1 {
		t.Fatalf("unexpected by_severity: %#v", summary.BySeverity)
	}
	if summary.Artifact == nil || summary.Artifact.Path == "" {
		t.Fatalf("expected artifact path to be present")
	}
}

func overrideScenarioAuditorBaseURL(url string, err error) func() {
	prev := resolveScenarioAuditorBaseURL
	resolveScenarioAuditorBaseURL = func(context.Context) (string, error) {
		return url, err
	}
	return func() { resolveScenarioAuditorBaseURL = prev }
}

func TestRunStandardsPhaseFailsOnHighWhenFailOnHigh(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/standards/check/demo":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"job_id":"job-123","status":{"id":"job-123","status":"running"}}`))
		case "/api/v1/standards/check/jobs/job-123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"job-123","status":"completed"}`))
		case "/api/v1/standards/check/jobs/job-123/summary":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"summary":{"total":1,"by_severity":{"high":1},"highest_severity":"high","top_violations":[{"severity":"high","rule_id":"prd_structure","file_path":"PRD.md","line_number":1,"title":"Bad PRD"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	restoreResolver := overrideScenarioAuditorBaseURL(server.URL, nil)
	defer restoreResolver()

	t.Setenv("TEST_GENIE_STANDARDS_FAIL_ON", "high")

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
	}
	report := runStandardsPhase(context.Background(), env, io.Discard)
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
	if report.Err == nil || !strings.Contains(report.Err.Error(), "fail_on") {
		t.Fatalf("expected threshold error, got %v", report.Err)
	}
}

func TestRunStandardsPhaseHandlesUnavailableScenarioAuditorAPI(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	restoreResolver := overrideScenarioAuditorBaseURL("", &discovery.Error{
		Kind:     discovery.ErrScenarioNotRunning,
		Scenario: "scenario-auditor",
		PortKey:  "API_PORT",
	})
	defer restoreResolver()

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
	}
	report := runStandardsPhase(context.Background(), env, io.Discard)
	if report.FailureClassification != FailureClassMissingDependency {
		t.Fatalf("expected missing dependency classification, got %s", report.FailureClassification)
	}
}

func TestStartAuditorStandardsScanSendsLogicalMapping(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/standards/check/demo" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"job_id":"job-123"}`))
	}))
	defer server.Close()

	mapping := workspace.Mapping{
		PhysicalScenarioDir:    filepath.Join(t.TempDir(), "scenarios", "demo"),
		PhysicalAppRoot:        t.TempDir(),
		LogicalRepoRoot:        t.TempDir(),
		LogicalScenarioRelPath: "scenarios/demo",
	}
	jobID, err := startAuditorStandardsScan(context.Background(), server.URL, "demo", mapping)
	if err != nil {
		t.Fatalf("startAuditorStandardsScan() error = %v", err)
	}
	if jobID != "job-123" {
		t.Fatalf("jobID = %q", jobID)
	}
	if payload["scenario_path"] != mapping.PhysicalScenarioDir ||
		payload["logical_repo_root"] != mapping.LogicalRepoRoot ||
		payload["logical_scenario_relpath"] != mapping.LogicalScenarioRelPath {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestRunStandardsPhaseClassifiesTimeout(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/standards/check/demo":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"job_id":"job-timeout","status":{"id":"job-timeout","status":"running"}}`))
		case "/api/v1/standards/check/jobs/job-timeout":
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"job-timeout","status":"running","message":"still running"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	restoreResolver := overrideScenarioAuditorBaseURL(server.URL, nil)
	defer restoreResolver()

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	report := runStandardsPhase(ctx, env, io.Discard)
	if report.FailureClassification != FailureClassTimeout {
		t.Fatalf("expected timeout classification, got %s", report.FailureClassification)
	}
	if report.Err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestRunStandardsPhaseHonorsMinSeverityForDisplay(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/standards/check/demo":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"job_id":"job-456","status":{"id":"job-456","status":"running"}}`))
		case "/api/v1/standards/check/jobs/job-456":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"job-456","status":"completed"}`))
		case "/api/v1/standards/check/jobs/job-456/summary":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"summary":{"total":2,"by_severity":{"high":1,"low":1},"highest_severity":"high","top_violations":[{"severity":"low","rule_id":"x","file_path":"a","line_number":1,"title":"low"},{"severity":"high","rule_id":"y","file_path":"b","line_number":2,"title":"high"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	restoreResolver := overrideScenarioAuditorBaseURL(server.URL, nil)
	defer restoreResolver()

	t.Setenv("TEST_GENIE_STANDARDS_MIN_SEVERITY", "high")
	t.Setenv("TEST_GENIE_STANDARDS_FAIL_ON", "critical")

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
	}
	report := runStandardsPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected pass when fail_on=critical, got err: %v", report.Err)
	}
	if report.FailureClassification != "" {
		t.Fatalf("expected no failure classification, got %s", report.FailureClassification)
	}

	joined := strings.Join(ObservationsToStrings(report.Observations), "\n")
	if strings.Contains(joined, "[LOW]") {
		t.Fatalf("expected low-severity violation to be omitted from observations, got:\n%s", joined)
	}
	if !strings.Contains(joined, "[HIGH]") {
		t.Fatalf("expected high-severity violation to be present, got:\n%s", joined)
	}
}
