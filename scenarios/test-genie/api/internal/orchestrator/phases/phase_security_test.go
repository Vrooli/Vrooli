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

func swapSecuritySeam(t *testing.T, fn func(ctx context.Context, scenario string) ([]byte, int, error)) func() {
	t.Helper()
	prev := runSecurityValidate
	runSecurityValidate = fn
	return func() { runSecurityValidate = prev }
}

func TestTranslateSecurityReport_Passing(t *testing.T) {
	rep := &securityReport{Scenario: "demo", Passed: true}
	out := translateSecurityReport(rep, 0)
	if !out.Success {
		t.Fatalf("expected Success=true")
	}
	if out.Summary.Scenario != "demo" {
		t.Fatalf("summary not translated: %+v", out.Summary)
	}
}

func TestTranslateSecurityReport_ErrorSeverityFailsPhase(t *testing.T) {
	rep := &securityReport{
		Scenario: "demo",
		Passed:   false,
		Findings: []securityFinding{
			{Severity: "SEVERITY_ERROR", RuleID: "gitleaks.aws-access-token", Title: "secret", FilePath: "leak.go:2", Scanner: "gitleaks"},
		},
	}
	rep.Summary.Errors = 1
	out := translateSecurityReport(rep, 1)
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

func TestTranslateSecurityReport_PreservesLocalMaturity(t *testing.T) {
	rep := &securityReport{
		Scenario:   "demo",
		Passed:     true,
		Assessment: testProviderAssessment("demo", "security-health", "security", "L2", "L3"),
	}

	out := translateSecurityReport(rep, 0)
	if out.Summary.LocalCurrentLevel != "L2" || out.Summary.LocalNextLevel != "L3" {
		t.Fatalf("local maturity not preserved: %+v", out.Summary)
	}
}

func TestTranslateSecurityReport_WarningOnlySucceeds(t *testing.T) {
	rep := &securityReport{
		Scenario: "demo",
		Passed:   true,
		Findings: []securityFinding{
			{Severity: "SEVERITY_WARNING", RuleID: "pnpm-audit.GHSA-x", Title: "moderate dep", Scanner: "pnpm-audit"},
		},
	}
	rep.Summary.Warnings = 1
	out := translateSecurityReport(rep, 0)
	if !out.Success {
		t.Fatalf("expected Success=true on warnings-only")
	}
}

func TestSecurityArchFindings_MapsSourceAndStableID(t *testing.T) {
	rep := &securityReport{
		Scenario: "demo",
		Findings: []securityFinding{
			{Severity: "SEVERITY_ERROR", RuleID: "gitleaks.aws", Title: "secret", FilePath: "leak.go:2"},
		},
	}
	got := securityArchFindings("demo", rep)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_SECURITY {
		t.Errorf("source = %v, want FINDING_SOURCE_SECURITY", got[0].GetSource())
	}
	if got[0].GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Errorf("severity = %v, want ERROR", got[0].GetSeverity())
	}
	if got[0].GetStableId() == "" {
		t.Error("stable id must be stamped")
	}
}

func TestParseSecurityOutput_Empty(t *testing.T) {
	if _, err := parseSecurityOutput([]byte("  ")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestParseSecurityOutput_MalformedAssessmentFails(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{},"assessment":{"provider":"security-health","phase":"security","local":{}}}`)
	if _, err := parseSecurityOutput(raw); err == nil {
		t.Fatal("expected malformed assessment error")
	}
}

func TestParseSecurityOutput_MissingAssessmentFails(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{}}`)
	if _, err := parseSecurityOutput(raw); err == nil {
		t.Fatal("expected missing assessment error")
	}
}

func TestRunSecurityPhase_CLIMissingSkipsGracefully(t *testing.T) {
	// The security phase is optional; an absent producer must NOT fail the
	// suite — it degrades to a non-failing skip.
	restore := swapSecuritySeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return nil, 0, errors.New("locate security-health CLI: not found")
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runSecurityPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("absent producer must skip, not fail; got %v", report.Err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("a skipped phase emits no findings, got %d", len(report.Findings))
	}
}

func TestRunSecurityPhase_EmptyOutputSkipsGracefully(t *testing.T) {
	// CLI present but the API was unreachable (empty stdout) — also a skip.
	restore := swapSecuritySeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return []byte("  "), 1, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runSecurityPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("unreachable producer must skip, not fail; got %v", report.Err)
	}
}

func TestRunSecurityPhase_SkipEnv(t *testing.T) {
	t.Setenv("TEST_GENIE_SKIP_SECURITY", "1")
	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runSecurityPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("skip path should not error, got %v", report.Err)
	}
}

func TestRunSecurityPhase_PlantedSecretFailsPhase(t *testing.T) {
	restore := swapSecuritySeam(t, func(_ context.Context, scenario string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"` + scenario + `",
			"findings":[
				{"rule_id":"gitleaks.aws-access-token","severity":"SEVERITY_ERROR","title":"AWS","remediation":"rotate","file_path":"leak.go:2","scanner":"gitleaks"}
			],
			"summary":{"errors":1},
			` + testProviderAssessmentJSON(scenario, "security-health", "security", "L1", "L2") + `
		}`), 1, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "planted", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	report := runSecurityPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatal("expected ERROR finding to fail the phase")
	}
	if len(report.Findings) != 1 || report.Findings[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_SECURITY {
		t.Fatalf("expected 1 SECURITY finding, got %+v", report.Findings)
	}
}
