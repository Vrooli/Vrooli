package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	internalvalidation "unit-health/internal/validation"
)

func testSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("read maturity.json: %v", err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return spec
}

func TestResponseToProtoMapsFields(t *testing.T) {
	spec := testSpec(t)
	in := internalvalidation.Response{
		RunID:      "uh-123",
		Status:     "passed",
		Scenario:   "demo",
		TargetKind: "scenario",
		TargetPath: "/x",
		Summary:    "ok",
		Surfaces:   []internalvalidation.Surface{{ID: "api", Kind: "api", Language: "go"}},
		Workspaces: []internalvalidation.Workspace{{ID: "api", Language: "go"}},
		Coverage:   []internalvalidation.CoverageTarget{{ID: "api:a.go", FilePath: "a.go", CoveragePercent: 80}},
		Findings: []internalvalidation.Finding{
			{Code: "TEST_NO_ASSERTION", Severity: "warning", Message: "m"},
			{Code: "LOW_COVERAGE", Severity: "warning", Message: "c"},
		},
		Maturity: internalvalidation.Maturity{Rung: 5, Label: "L5"},
		Artifacts: []internalvalidation.Artifact{
			{Label: "Validation run", Kind: "run", Reference: "uh-123"},
			{Label: "Coverage (api)", Kind: "coverage", Reference: "/x/api"},
		},
	}
	out, err := responseToProto(in, spec)
	if err != nil {
		t.Fatalf("responseToProto: %v", err)
	}
	if out.GetRunId() != "uh-123" || out.GetStatus() != "passed" || out.GetScenario() != "demo" {
		t.Errorf("scalar fields not mapped: %+v", out)
	}
	if out.GetCounts().GetWarnings() != 2 || out.GetCounts().GetSurfaces() != 1 || out.GetCounts().GetCoverageTargets() != 1 {
		t.Errorf("counts wrong: %+v", out.GetCounts())
	}
	if len(out.GetFindings()) != 2 || len(out.GetSurfaces()) != 1 || len(out.GetWorkspaces()) != 1 {
		t.Errorf("collections not mapped: findings=%d surfaces=%d", len(out.GetFindings()), len(out.GetSurfaces()))
	}
	if out.GetAssessment() == nil {
		t.Error("maturity assessment must be built")
	}
	if out.GetMaturity().GetRung() != 5 {
		t.Errorf("maturity rung = %d, want 5", out.GetMaturity().GetRung())
	}
	if len(out.GetArtifacts()) != 2 {
		t.Fatalf("artifacts not mapped: got %d, want 2", len(out.GetArtifacts()))
	}
	if got := out.GetArtifacts()[0]; got.GetKind() != "run" || got.GetReference() != "uh-123" {
		t.Errorf("first artifact = %+v, want run/uh-123", got)
	}
	if got := out.GetArtifacts()[1]; got.GetLabel() != "Coverage (api)" || got.GetReference() != "/x/api" {
		t.Errorf("second artifact = %+v, want Coverage (api)//x/api", got)
	}
}

func TestResponseToProtoRequiresSpec(t *testing.T) {
	_, err := responseToProto(internalvalidation.Response{Scenario: "demo"}, nil)
	if err == nil {
		t.Error("responseToProto must error when the maturity spec is nil")
	}
}

func TestValidateScenarioRejectsEmptyTarget(t *testing.T) {
	h := NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)})
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{}))
	if err == nil {
		t.Fatal("expected an error for empty scenario+path")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestFindingSourceFromDimension(t *testing.T) {
	spec := testSpec(t)
	if got := findingSource("LOW_COVERAGE", spec); got.String() != "FINDING_SOURCE_COVERAGE" {
		t.Errorf("LOW_COVERAGE source = %v, want coverage", got)
	}
	if got := findingSource("TEST_NO_ASSERTION", spec); got.String() != "FINDING_SOURCE_STANDARDS" {
		t.Errorf("TEST_NO_ASSERTION source = %v, want standards", got)
	}
}

func TestSeverityToAssessment(t *testing.T) {
	cases := map[string]string{
		"error":   "FINDING_SEVERITY_ERROR",
		"warning": "FINDING_SEVERITY_WARNING",
		"info":    "FINDING_SEVERITY_INFO",
	}
	for in, want := range cases {
		if got := severityToAssessment(in); got != want {
			t.Errorf("severityToAssessment(%q) = %q, want %q", in, got, want)
		}
	}
}
