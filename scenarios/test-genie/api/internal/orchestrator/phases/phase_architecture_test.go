package phases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	audit_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	conflicts_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"test-genie/internal/orchestrator/workspace"
)

func workspaceEnv(t *testing.T, dir string) workspace.Environment {
	t.Helper()
	testDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: testDir}
}

func swapArchitectureSeam(t *testing.T, fn func(ctx context.Context, scenario string) (*audit_v1.AuditRunResponse, error)) func() {
	t.Helper()
	prev := runArchitectureAudit
	runArchitectureAudit = fn
	return func() { runArchitectureAudit = prev }
}

func sampleAuditResponse() *audit_v1.AuditRunResponse {
	return &audit_v1.AuditRunResponse{
		Scenario:      "demo",
		Outcome:       audit_v1.AuditOutcome_AUDIT_OUTCOME_FINDINGS,
		TotalFindings: 2,
		BySeverity:    map[string]int32{"blocker": 1, "warn": 1},
		Assessment:    testProtoProviderAssessment("demo", "architecture-cartographer", "architecture", "L3", "L4"),
		Findings: []*audit_v1.ConflictSummary{
			{Type: "cycle", Subtype: "cross-domain", Severity: conflicts_v1.Severity_SEVERITY_BLOCKER, Locations: []string{"api/internal/a", "api/internal/b"}, Domains: []string{"a", "b"}, Headline: "import cycle a↔b"},
			{Type: "coupling_smell", Severity: conflicts_v1.Severity_SEVERITY_WARN, Locations: []string{"api/internal/c"}, Headline: "high fan-out"},
		},
		Domains: &audit_v1.DerivedDomainSummary{Confidence: "high"},
	}
}

func TestTranslateArchitectureResponse_FindingsAreAdvisory(t *testing.T) {
	out := translateArchitectureResponse(sampleAuditResponse())
	if !out.Success {
		t.Fatalf("findings outcome must NOT fail the advisory phase")
	}
	if out.Summary.Total != 2 || out.Summary.Blockers != 1 || out.Summary.Warnings != 1 {
		t.Errorf("summary mismatch: %+v", out.Summary)
	}
	if out.Summary.Outcome != "findings" {
		t.Errorf("outcome = %q, want findings", out.Summary.Outcome)
	}
	if out.Summary.Authority != "high" {
		t.Errorf("authority = %q, want high", out.Summary.Authority)
	}
	if out.Summary.LocalCurrentLevel != "L3" || out.Summary.LocalNextLevel != "L4" {
		t.Errorf("local maturity = %q/%q, want L3/L4", out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel)
	}
}

func TestTranslateArchitectureResponse_ToolErrorFails(t *testing.T) {
	resp := &audit_v1.AuditRunResponse{
		Scenario: "demo",
		Outcome:  audit_v1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR,
		Error:    "graph extraction failed",
	}
	out := translateArchitectureResponse(resp)
	if out.Success {
		t.Fatalf("TOOL_ERROR must fail the phase")
	}
	if out.FailureClass == "" {
		t.Errorf("expected failure class set")
	}
}

func TestArchitectureArchFindings(t *testing.T) {
	got := architectureArchFindings("demo", sampleAuditResponse())
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d", len(got))
	}
	if got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE {
		t.Errorf("source = %v, want ARCHITECTURE", got[0].Source)
	}
	if got[0].Code != "cycle/cross-domain" {
		t.Errorf("code = %q, want cycle/cross-domain", got[0].Code)
	}
	if got[0].Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER {
		t.Errorf("severity = %v, want BLOCKER", got[0].Severity)
	}
	if len(got[0].Domains) != 2 {
		t.Errorf("domains not carried: %v", got[0].Domains)
	}
	if got[0].StableId == "" {
		t.Errorf("missing stable id")
	}
	// coupling smell with no subtype keeps the bare type as code.
	if got[1].Code != "coupling_smell" {
		t.Errorf("code = %q, want coupling_smell", got[1].Code)
	}
}

func TestRunArchitecturePhase_PopulatesFindings(t *testing.T) {
	restore := swapArchitectureSeam(t, func(_ context.Context, _ string) (*audit_v1.AuditRunResponse, error) {
		return sampleAuditResponse(), nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspaceEnv(t, dir)
	var buf bytes.Buffer
	report := runArchitecturePhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("advisory phase should not error on findings: %v", report.Err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("want 2 findings on report, got %d", len(report.Findings))
	}
}

func TestRunArchitecturePhase_MissingAssessmentFailsMaturityContract(t *testing.T) {
	restore := swapArchitectureSeam(t, func(_ context.Context, _ string) (*audit_v1.AuditRunResponse, error) {
		resp := sampleAuditResponse()
		resp.Assessment = nil
		return resp, nil
	})
	defer restore()

	dir := t.TempDir()
	env := workspaceEnv(t, dir)
	report := runArchitecturePhase(context.Background(), env, io.Discard)
	if report.FailureClassification != FailureClassMaturityContract {
		t.Fatalf("expected maturity contract classification, got %s", report.FailureClassification)
	}
	if report.Err == nil {
		t.Fatalf("expected missing assessment error")
	}
}

// An unreachable cartographer must NOT gate the suite: the architecture axis is
// advisory, so it degrades to an advisory skip (no error, skip observation) so
// presets that include it (comprehensive) never hard-fail when the optional
// architecture-cartographer infrastructure is absent.
func TestRunArchitecturePhase_UnreachableDegradesToAdvisorySkip(t *testing.T) {
	restore := swapArchitectureSeam(t, func(_ context.Context, _ string) (*audit_v1.AuditRunResponse, error) {
		return nil, errors.New("connection refused")
	})
	defer restore()

	dir := t.TempDir()
	env := workspaceEnv(t, dir)
	var buf bytes.Buffer
	report := runArchitecturePhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("unreachable cartographer must NOT fail the advisory phase, got: %v", report.Err)
	}
	if report.FailureClassification != "" {
		t.Errorf("classification = %q, want empty (advisory skip)", report.FailureClassification)
	}
	foundSkip := false
	for _, obs := range report.Observations {
		if obs.Prefix == "SKIP" {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Errorf("expected an advisory skip observation, got %+v", report.Observations)
	}
}
