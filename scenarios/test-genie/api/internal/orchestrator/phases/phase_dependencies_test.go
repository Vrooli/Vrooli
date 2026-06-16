package phases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func swapDependencyHealthSeam(t *testing.T, fn func(ctx context.Context, scenario string) ([]byte, int, error)) {
	t.Helper()
	prev := runDependencyHealth
	runDependencyHealth = fn
	t.Cleanup(func() { runDependencyHealth = prev })
}

func dependencyEnv(t *testing.T) workspace.Environment {
	t.Helper()
	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
		AppRoot:      filepath.Dir(dir),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestDependencyHealthArchFindings_MapsSourceAndStableID(t *testing.T) {
	report, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": false,
		"findings": [
			{
				"id": "graph.proto-health",
				"severity": "WARNING",
				"sourceDomain": "graph",
				"title": "undeclared scenario dependency",
				"description": "demo imports proto-health but does not declare it",
				"remediation": "Declare the scenario dependency.",
				"filePath": "scenarios/demo/.vrooli/service.json",
				"ruleId": "dependency.graph.undeclared"
			}
		],
		` + testProtoProviderAssessmentJSON("demo", "scenario-dependency-analyzer", "dependencies", "L4", "L5") + `
	}`))
	if err != nil {
		t.Fatalf("parseDependencyHealthOutput: %v", err)
	}
	got := dependencyHealthArchFindings("demo", report)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY {
		t.Errorf("source = %v, want FINDING_SOURCE_DEPENDENCY", got[0].GetSource())
	}
	if got[0].GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("severity = %v, want WARNING", got[0].GetSeverity())
	}
	if got[0].GetCode() != "dependency.graph.undeclared" {
		t.Errorf("code = %q, want dependency.graph.undeclared", got[0].GetCode())
	}
	if got[0].GetStableId() == "" {
		t.Error("stable id must be stamped")
	}
}

func TestTranslateDependencyHealthReport_ErrorSeverityFailsPhase(t *testing.T) {
	report, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": false,
		"summary": {"findings": 1, "errors": 1},
		"findings": [
			{
				"id": "release-age.ui.minimum-too-low",
				"severity": "ERROR",
				"sourceDomain": "release-age",
				"title": "pnpm release-age minimum is too low",
				"ruleId": "dependency.release_age.minimum_value"
			}
		]
		,
		` + testProtoProviderAssessmentJSON("demo", "scenario-dependency-analyzer", "dependencies", "L2", "L3") + `
	}`))
	if err != nil {
		t.Fatal(err)
	}
	out := translateDependencyHealthReport(report, 1)
	if out.Success {
		t.Fatalf("expected Success=false on ERROR finding")
	}
	if out.FailureClass == "" {
		t.Fatalf("expected failure class set")
	}
	if out.Summary.Errors != 1 {
		t.Fatalf("summary error count not propagated: %+v", out.Summary)
	}
}

func TestTranslateDependencyHealthReport_WarningOnlySucceeds(t *testing.T) {
	report, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": true,
		"summary": {"findings": 1, "warnings": 1},
		"findings": [
			{
				"id": "governance.npm.react.needs-review",
				"severity": "WARNING",
				"sourceDomain": "governance",
				"title": "Dependency needs governance review",
				"ruleId": "dependency.governance.needs_review"
			}
		]
		,
		` + testProtoProviderAssessmentJSON("demo", "scenario-dependency-analyzer", "dependencies", "L4", "L5") + `
	}`))
	if err != nil {
		t.Fatal(err)
	}
	out := translateDependencyHealthReport(report, 0)
	if !out.Success {
		t.Fatalf("expected Success=true on warnings-only")
	}
	if out.Summary.Warnings != 1 {
		t.Fatalf("summary warning count not propagated: %+v", out.Summary)
	}
}

func TestParseDependencyHealthOutput_RejectsMalformedPresentAssessment(t *testing.T) {
	_, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": true,
		"assessment": {
			"provider": "scenario-dependency-analyzer",
			"phase": "dependencies",
			"local": {}
		}
	}`))
	if err == nil {
		t.Fatal("expected malformed assessment to fail")
	}
	if !errors.Is(err, errProviderMaturityContract) {
		t.Fatalf("error = %v, want maturity contract error", err)
	}
}

func TestParseDependencyHealthOutput_RejectsMissingAssessment(t *testing.T) {
	_, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": true,
		"summary": {"findings": 0}
	}`))
	if err == nil {
		t.Fatal("expected missing assessment to fail")
	}
	if !errors.Is(err, errProviderMaturityContract) {
		t.Fatalf("error = %v, want maturity contract error", err)
	}
}

func TestDependencySummaryPreservesLocalMaturity(t *testing.T) {
	report, err := parseDependencyHealthOutput([]byte(`{
		"scenario": "demo",
		"passed": true,
		"summary": {"findings": 0},
		"assessment": {
			"scenario": "demo",
			"provider": "scenario-dependency-analyzer",
			"phase": "dependencies",
			"version": "1.0.0",
			"local": {
				"currentLevel": "L4",
				"nextLevel": "L5"
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	summary := dependencySummary(report)
	if summary.LocalCurrentLevel != "L4" || summary.LocalNextLevel != "L5" {
		t.Fatalf("summary local maturity = %s/%s, want L4/L5", summary.LocalCurrentLevel, summary.LocalNextLevel)
	}
}

func TestRunDependenciesPhase_HealthUnavailableFailsAsProducerDependency(t *testing.T) {
	swapDependencyHealthSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return nil, 0, errors.New("locate scenario-dependency-analyzer CLI: not found")
	})

	var buf bytes.Buffer
	report := runDependenciesPhase(context.Background(), dependencyEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatalf("unavailable dependency producer must fail the required dependency phase")
	}
	if len(report.Findings) != 1 || report.Findings[0].GetCode() != "dependency.producer_unavailable" {
		t.Fatalf("expected producer unavailable finding, got %+v", report.Findings)
	}
}

func TestRunDependenciesPhase_MapsSDAHealthFindings(t *testing.T) {
	swapDependencyHealthSeam(t, func(_ context.Context, scenario string) ([]byte, int, error) {
		return []byte(`{
			"scenario": "` + scenario + `",
			"passed": false,
			"summary": {"sections": 6, "findings": 4, "errors": 1, "warnings": 3, "degradedDependencies": 1},
			"sections": [
				{"id": "release-age", "title": "Package release-age policy", "status": "fail"},
				{"id": "security", "title": "Security Health dependency index", "status": "degraded"},
				{"id": "graph", "title": "Dependency graph drift", "status": "warn"},
				{"id": "governance", "title": "Approved dependency governance", "status": "warn"}
			],
			"findings": [
				{
					"id": "release-age.ui.minimum-too-low",
					"severity": "ERROR",
					"sourceDomain": "release-age",
					"title": "pnpm release-age minimum is too low",
					"ruleId": "dependency.release_age.minimum_value",
					"filePath": "scenarios/demo/ui/pnpm-workspace.yaml"
				},
				{
					"id": "graph.code-facts.undeclared",
					"severity": "WARNING",
					"sourceDomain": "graph",
					"title": "Undeclared scenario dependency",
					"ruleId": "dependency.graph.undeclared",
					"filePath": "scenarios/demo/.vrooli/service.json"
				},
				{
					"id": "governance.npm.react.needs-review",
					"severity": "WARNING",
					"sourceDomain": "governance",
					"title": "Dependency needs governance review",
					"ruleId": "dependency.governance.needs_review",
					"filePath": "scenarios/demo/ui/package.json"
				},
				{
					"id": "security.degraded",
					"severity": "WARNING",
					"sourceDomain": "security",
					"title": "Security Health dependency index unavailable",
					"ruleId": "dependency.security.degraded"
				}
			],
			"degradedDependencies": [
				{"id": "security-health-deps-status", "dependency": "security-health dependency index", "domain": "security", "reason": "index unavailable"}
			],
			` + testProtoProviderAssessmentJSON(scenario, "scenario-dependency-analyzer", "dependencies", "L2", "L3") + `
		}`), 1, nil
	})

	var buf bytes.Buffer
	report := runDependenciesPhase(context.Background(), dependencyEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatal("expected release-age ERROR finding to fail the dependency phase")
	}
	if len(report.Findings) != 4 {
		t.Fatalf("expected 4 dependency findings, got %+v", report.Findings)
	}
	wantCodes := map[string]bool{
		"dependency.release_age.minimum_value": false,
		"dependency.graph.undeclared":          false,
		"dependency.governance.needs_review":   false,
		"dependency.security.degraded":         false,
	}
	for _, finding := range report.Findings {
		if finding.GetSource() != architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY {
			t.Fatalf("expected dependency source for all findings, got %+v", finding)
		}
		if _, ok := wantCodes[finding.GetCode()]; ok {
			wantCodes[finding.GetCode()] = true
		}
	}
	for code, seen := range wantCodes {
		if !seen {
			t.Fatalf("missing mapped SDA health finding %s in %+v", code, report.Findings)
		}
	}
}
