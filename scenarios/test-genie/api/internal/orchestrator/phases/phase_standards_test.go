package phases

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestParseAuditorStandardsSummaryParsesValidJSON(t *testing.T) {
	raw := `{
  "security": null,
  "standards": {
    "summary": {
      "total": 2,
      "by_severity": {"HIGH": 1, "low": 1},
      "by_rule": [{"rule_id":"prd_structure","count":2,"severity":"high","title":"PRD structure"}],
      "highest_severity": "HIGH",
      "top_violations": [{"severity":"high","rule_id":"prd_structure","file_path":"PRD.md","line_number":1,"title":"Bad PRD"}],
      "artifact": {"path":"logs/scenario-auditor/standards/demo.json"},
      "recommended_steps": ["Fix PRD.md"]
    }
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

func TestRunStandardsPhaseFailsOnHighWhenFailOnHigh(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
		if name != "scenario-auditor" {
			return "", nil
		}
		return `{"security":null,"standards":{"summary":{"total":1,"by_severity":{"high":1},"highest_severity":"high","top_violations":[{"severity":"high","rule_id":"prd_structure","file_path":"PRD.md","line_number":1,"title":"Bad PRD"}]}}}`, nil
	})

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

func TestRunStandardsPhaseHandlesMissingBinary(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		if name == "scenario-auditor" {
			return "", &commandNotFoundError{name}
		}
		return "/tmp/" + name, nil
	})

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

func TestRunStandardsPhaseClassifiesTimeout(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
		return `{"security":null,"standards":{"summary":{"total":0,"by_severity":{},"highest_severity":"","top_violations":[]}}}`, context.DeadlineExceeded
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
	}
	report := runStandardsPhase(context.Background(), env, io.Discard)
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

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
		return `{"security":null,"standards":{"summary":{"total":2,"by_severity":{"high":1,"low":1},"highest_severity":"high","top_violations":[{"severity":"low","rule_id":"x","file_path":"a","line_number":1,"title":"low"},{"severity":"high","rule_id":"y","file_path":"b","line_number":2,"title":"high"}]}}}`, nil
	})

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
