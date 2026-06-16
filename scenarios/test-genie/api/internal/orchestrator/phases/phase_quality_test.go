package phases

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestRunQualityPhasePassesAndMapsWarningFindings(t *testing.T) {
	restore := overrideQualityValidate(func(context.Context, string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"demo",
			"status":"passed",
			"findings":[{
				"rule_id":"TS_DANGEROUS_PATTERNS",
				"severity":"warning",
				"message":"Dangerous TypeScript suppression patterns found",
				"file_path":"ui",
				"remediation":"Fix source code instead of suppressing checks."
			}],
			"counts":{"warnings":1,"surfaces":3,"contracts":4},
			"next_steps":["Run quality-health explain finding abc --scenario demo"]
		}`), 0, nil
	})
	defer restore()

	report := runQualityPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err != nil {
		t.Fatalf("quality phase should pass on warning-only report: %v", report.Err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected one mapped finding, got %d", len(report.Findings))
	}
	f := report.Findings[0]
	if f.GetSource() != architecturev1.FindingSource_FINDING_SOURCE_STANDARDS {
		t.Fatalf("finding source = %v, want STANDARDS", f.GetSource())
	}
	if f.GetCode() != "TS_DANGEROUS_PATTERNS" {
		t.Fatalf("finding code = %q", f.GetCode())
	}
	if f.GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Fatalf("finding severity = %v, want WARNING", f.GetSeverity())
	}
}

func TestRunQualityPhaseFailsOnErrorFindings(t *testing.T) {
	restore := overrideQualityValidate(func(context.Context, string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"demo",
			"status":"failed",
			"findings":[{
				"rule_id":"TS_CONFIG_STRICT",
				"severity":"error",
				"message":"tsconfig strict mode is disabled",
				"file_path":"ui/tsconfig.json"
			}],
			"counts":{"errors":1,"surfaces":1,"contracts":1}
		}`), 1, nil
	})
	defer restore()

	report := runQualityPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err == nil {
		t.Fatalf("quality phase should fail on error findings")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("classification = %q, want system", report.FailureClassification)
	}
	if !strings.Contains(report.Remediation, "quality-health audit run demo --json") {
		t.Fatalf("remediation should point at quality-health audit, got %q", report.Remediation)
	}
}

func TestRunQualityPhaseFailsWhenProviderMissing(t *testing.T) {
	restore := overrideQualityValidate(func(context.Context, string) ([]byte, int, error) {
		return nil, 0, errors.New("not found")
	})
	defer restore()

	report := runQualityPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err == nil {
		t.Fatalf("quality phase should fail when quality-health is unavailable")
	}
	if report.FailureClassification != "missing_dependency" {
		t.Fatalf("classification = %q, want missing_dependency", report.FailureClassification)
	}
}

func TestParseQualityOutputRejectsEmpty(t *testing.T) {
	if _, err := parseQualityOutput([]byte("   ")); err == nil {
		t.Fatalf("expected empty quality-health output to fail parsing")
	}
}

func overrideQualityValidate(fn func(context.Context, string) ([]byte, int, error)) func() {
	prev := runQualityValidate
	runQualityValidate = fn
	return func() { runQualityValidate = prev }
}
