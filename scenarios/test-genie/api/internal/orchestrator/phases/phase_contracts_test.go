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
)

func swapContractsSeam(t *testing.T, fn func(ctx context.Context, scenario string) ([]byte, int, error)) func() {
	t.Helper()
	prev := runCLIHealthValidate
	runCLIHealthValidate = fn
	return func() { runCLIHealthValidate = prev }
}

func TestTranslateContractsReport_Passing(t *testing.T) {
	rep := &cliHealthReport{Scenario: "demo", Passed: true}
	out := translateContractsReport(rep, 0)
	if !out.Success {
		t.Fatalf("expected Success=true")
	}
	if out.Summary.Scenario != "demo" || out.Summary.Errors != 0 {
		t.Fatalf("summary not translated: %+v", out.Summary)
	}
}

func TestTranslateContractsReport_ErrorSeverityFailsPhase(t *testing.T) {
	rep := &cliHealthReport{
		Scenario: "demo",
		Passed:   false,
		Findings: []cliHealthFinding{
			{Severity: "SEVERITY_ERROR", Code: "proto.orphan_method", Message: "method not bound", Location: "manifest.json"},
		},
	}
	rep.Summary.Errors = 1
	out := translateContractsReport(rep, 1)
	if out.Success {
		t.Fatalf("expected Success=false on ERROR finding")
	}
	if out.FailureClass == "" {
		t.Fatalf("expected failure class set")
	}
	if len(out.Observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(out.Observations))
	}
	if out.Summary.Errors != 1 {
		t.Fatalf("summary error count not propagated: %+v", out.Summary)
	}
}

func TestTranslateContractsReport_WarningOnlySucceeds(t *testing.T) {
	rep := &cliHealthReport{
		Scenario: "demo",
		Passed:   true,
		Findings: []cliHealthFinding{
			{Severity: "SEVERITY_WARNING", Code: "manifest.missing", Message: "no manifest"},
		},
	}
	rep.Summary.Warnings = 1
	out := translateContractsReport(rep, 0)
	if !out.Success {
		t.Fatalf("expected Success=true on warnings-only")
	}
	if out.Summary.Warnings != 1 {
		t.Fatalf("warning count not propagated")
	}
}

func TestTranslateContractsReport_PreservesLocalMaturitySummary(t *testing.T) {
	rep := &cliHealthReport{
		Scenario:   "demo",
		Passed:     true,
		Assessment: testProviderAssessment("demo", "cli-health", "contracts", "L2", "L3"),
	}
	out := translateContractsReport(rep, 0)
	if out.Summary.LocalCurrentLevel != "L2" || out.Summary.LocalNextLevel != "L3" {
		t.Fatalf("local summary = current %q next %q, want L2/L3", out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel)
	}
	if got := out.Summary.String(); got != "demo passed=true errors=0 warnings=0 infos=0 local=L2 next=L3" {
		t.Fatalf("summary string = %q", got)
	}
}

func TestTranslateContractsReport_NonZeroExitWithoutErrorsFailsPhase(t *testing.T) {
	rep := &cliHealthReport{Scenario: "demo", Passed: true}
	out := translateContractsReport(rep, 2)
	if out.Success {
		t.Fatalf("expected Success=false when exit != 0")
	}
	if out.FailureClass == "" {
		t.Fatalf("expected failure class set")
	}
}

func TestParseCLIHealthOutput_Empty(t *testing.T) {
	if _, err := parseCLIHealthOutput([]byte("   ")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestParseCLIHealthOutput_InvalidJSON(t *testing.T) {
	if _, err := parseCLIHealthOutput([]byte("not json")); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseCLIHealthOutput_RejectsMalformedAssessment(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{},"assessment":{"provider":"cli-health","phase":"contracts","local":{}}}`)
	if _, err := parseCLIHealthOutput(raw); err == nil {
		t.Fatal("expected malformed assessment error")
	} else if got := classifyProviderParseFailure(err); got != "maturity_contract" {
		t.Fatalf("classification = %q, want maturity_contract", got)
	}
}

func TestParseCLIHealthOutput_RejectsMissingAssessment(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{}}`)
	if _, err := parseCLIHealthOutput(raw); err == nil {
		t.Fatal("expected missing assessment error")
	} else if got := classifyProviderParseFailure(err); got != "maturity_contract" {
		t.Fatalf("classification = %q, want maturity_contract", got)
	}
}

func TestRunContractsPhase_HappyPath(t *testing.T) {
	restore := swapContractsSeam(t, func(_ context.Context, scenario string) ([]byte, int, error) {
		return []byte(`{"scenario":"` + scenario + `","passed":true,"summary":{},` + testProviderAssessmentJSON(scenario, "cli-health", "contracts", "L5", "") + `}`), 0, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runContractsPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("expected no error, got %v", report.Err)
	}
}

func TestRunContractsPhase_CLIMissingIsMissingDependency(t *testing.T) {
	restore := swapContractsSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return nil, 0, errors.New("locate cli-health CLI: not found")
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runContractsPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatal("expected error when CLI cannot be invoked")
	}
	if string(report.FailureClassification) != "missing_dependency" {
		t.Fatalf("FailureClassification = %q, want missing_dependency", report.FailureClassification)
	}
}

func TestRunContractsPhase_FindingsFailPhase(t *testing.T) {
	restore := swapContractsSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"broken",
			"passed":false,
			"findings":[
				{"severity":"SEVERITY_ERROR","code":"proto.orphan_method","location":"x","message":"y"}
			],
			"summary":{"errors":1},
			` + testProviderAssessmentJSON("broken", "cli-health", "contracts", "L1", "L2") + `
		}`), 1, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "broken",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runContractsPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatal("expected phase to fail when ERROR findings are present")
	}
}

func TestRunContractsPhase_SkipEnvVar(t *testing.T) {
	t.Setenv("TEST_GENIE_SKIP_CONTRACTS", "1")
	restore := swapContractsSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		t.Fatal("CLI should not be invoked when skip env is set")
		return nil, 0, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runContractsPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("skip path should not error: %v", report.Err)
	}
}
