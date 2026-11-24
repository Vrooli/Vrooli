package catalog

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
)

func TestGateReportGroupsFindingsAndLabelsCalibration(t *testing.T) {
	msg := &catalogv1.RunGateResponse{
		Gate: "content", InspectedFiles: 2, EvidenceRowsWritten: 1,
		SurfaceVerdictCounts: map[string]int32{"passed": 2, "failed": 1},
		Findings: []*catalogv1.GateFinding{
			{Code: "missing-token", File: "Button.tsx", Line: 9, SeverityClass: catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING, Scope: catalogv1.FindingScope_FINDING_SCOPE_ASSET, Owner: "design", Remediation: "use token", DocsRef: "docs/tokens", Message: "color is literal", Blocking: true},
			{Code: "missing-token", File: "Card.tsx", Severity: "warning", Remediation: "use token", Message: "border is literal"},
		},
		Calibration: []*catalogv1.CalibrationResult{{Status: "failed", Message: "fixture did not fail", RequiredFailureCode: "expected", ObservedFailureCode: "none"}},
	}
	report := (&handlers{}).gateReport(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}), msg)
	body := strings.Join(append(report.Summary, report.Results...), "\n")
	for _, want := range []string{"inspected 2 file(s), 2 finding(s)", "persisted 1 evidence row(s)", "failed=1", "[missing-token] error — 2 location(s)", "Button.tsx:9: color is literal", "Card.tsx: border is literal", "asset, owner design", "fix: use token", "docs: docs/tokens", "calibration.failed", "expected, observed none"} {
		if !strings.Contains(body, want) {
			t.Errorf("report missing %q:\n%s", want, body)
		}
	}
}

func TestGateReportWarnsOnEmptyAndQuarantinedResults(t *testing.T) {
	report := (&handlers{}).gateReport(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}), &catalogv1.RunGateResponse{Gate: "empty", NonDiscriminating: true})
	body := strings.Join(report.Summary, "\n")
	if !strings.Contains(body, "inspected zero files") || !strings.Contains(body, "QUARANTINED") {
		t.Fatalf("missing safety warnings: %s", body)
	}
}

func TestFindingLabelsAndOwnerSuffix(t *testing.T) {
	for _, tc := range []struct {
		finding         *catalogv1.GateFinding
		severity, scope string
	}{
		{&catalogv1.GateFinding{SeverityClass: catalogv1.FindingSeverity_FINDING_SEVERITY_INFO, Scope: catalogv1.FindingScope_FINDING_SCOPE_CORPUS}, "info", "corpus"},
		{&catalogv1.GateFinding{SeverityClass: catalogv1.FindingSeverity_FINDING_SEVERITY_WARNING}, "warning", ""},
		{&catalogv1.GateFinding{Severity: "notice"}, "notice", ""},
		{&catalogv1.GateFinding{Severity: "error"}, "error", ""},
	} {
		if got := findingSeverityLabel(tc.finding); got != tc.severity {
			t.Errorf("severity=%q want %q", got, tc.severity)
		}
		if got := findingScopeLabel(tc.finding); got != tc.scope {
			t.Errorf("scope=%q want %q", got, tc.scope)
		}
	}
	if ownerSuffix("") != "" || ownerSuffix("team") != ", owner team" {
		t.Fatal("owner suffix formatting changed")
	}
}
