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

func swapProtoSeam(t *testing.T, fn func(ctx context.Context, scenario string) ([]byte, int, error)) {
	t.Helper()
	prev := runProtoValidate
	runProtoValidate = fn
	t.Cleanup(func() { runProtoValidate = prev })
}

func protoEnv(t *testing.T) workspace.Environment {
	t.Helper()
	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestTranslateProtoReport_ErrorSeverityFailsPhase(t *testing.T) {
	rep := &protoReport{
		Scenario: "demo",
		Passed:   false,
		Findings: []protoFinding{
			{Severity: "SEVERITY_ERROR", Code: "proto.cycle", Message: "cycle", Location: "packages/proto/schemas/demo/v1/a/a.proto"},
		},
	}
	rep.Summary.Errors = 1
	out := translateProtoReport(rep, 1)
	if out.Success {
		t.Fatal("expected Success=false on ERROR finding")
	}
	if out.FailureClass == "" {
		t.Fatal("expected failure class set")
	}
}

func TestTranslateProtoReport_WarningOnlySucceeds(t *testing.T) {
	rep := &protoReport{
		Scenario: "demo",
		Passed:   true,
		Findings: []protoFinding{
			{Severity: "SEVERITY_WARNING", Code: "proto.cross_domain_import", Message: "domain import"},
		},
	}
	rep.Summary.Warnings = 1
	out := translateProtoReport(rep, 0)
	if !out.Success {
		t.Fatal("expected Success=true on warnings-only")
	}
}

func TestProtoArchFindings_MapsSourceAndStableID(t *testing.T) {
	rep := &protoReport{
		Scenario: "demo",
		Findings: []protoFinding{
			{Severity: "SEVERITY_WARNING", Code: "proto.hand_rolled_transport", Message: "hand rolled", Location: "api/server.go"},
		},
	}
	got := protoArchFindings("demo", rep)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_PROTO {
		t.Errorf("source = %v, want FINDING_SOURCE_PROTO", got[0].GetSource())
	}
	if got[0].GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("severity = %v, want WARNING", got[0].GetSeverity())
	}
	if got[0].GetStableId() == "" {
		t.Error("stable id must be stamped")
	}
}

func TestParseProtoOutput_Empty(t *testing.T) {
	if _, err := parseProtoOutput([]byte("  ")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestRunProtoPhase_CLIMissingSkipsGracefully(t *testing.T) {
	swapProtoSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return nil, 0, errors.New("locate proto-health CLI: not found")
	})

	var buf bytes.Buffer
	report := runProtoPhase(context.Background(), protoEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("absent producer must skip, not fail; got %v", report.Err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("a skipped phase emits no findings, got %d", len(report.Findings))
	}
}

func TestRunProtoPhase_ErrorFindingFailsPhase(t *testing.T) {
	swapProtoSeam(t, func(_ context.Context, scenario string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"` + scenario + `",
			"findings":[
				{"code":"proto.gen_out_of_sync","severity":"SEVERITY_ERROR","message":"generated artifacts are stale","suggestion":"run make generate","location":"packages/proto/gen"}
			],
			"summary":{"errors":1}
		}`), 1, nil
	})

	var buf bytes.Buffer
	report := runProtoPhase(context.Background(), protoEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatal("expected ERROR finding to fail the phase")
	}
	if len(report.Findings) != 1 || report.Findings[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_PROTO {
		t.Fatalf("expected 1 PROTO finding, got %+v", report.Findings)
	}
}
