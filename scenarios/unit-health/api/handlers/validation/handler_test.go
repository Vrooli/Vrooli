package validation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"unit-health/internal/discovery"
	internalvalidation "unit-health/internal/validation"
)

// commonStage aliases the generated stage proto so the stage-name helpers read
// cleanly.
type commonStage = commonv1.Stage

// fakeDiscoverer returns a canned inventory so handler tests never touch Code
// Facts. A single Go surface is enough to drive the validation through the
// planner and (when execution is requested) the executor.
type fakeDiscoverer struct {
	inv discovery.Inventory
}

func (f fakeDiscoverer) Discover(context.Context, string, string, bool) (discovery.Inventory, error) {
	return f.inv, nil
}

func goSurfaceInventory(t *testing.T) discovery.Inventory {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	return discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"}},
	}
}

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
			{Code: "TEST_NO_ASSERTION", Severity: "warning", Message: "m", Evidence: "role=ui policy_class=react_vite_ui", Expected: "coverage thresholds >= 85", Observed: "coverage thresholds below policy", Remediation: "Restore the template policy projection."},
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
	firstFinding := out.GetFindings()[0]
	if firstFinding.GetEvidence() != "role=ui policy_class=react_vite_ui" ||
		firstFinding.GetExpected() != "coverage thresholds >= 85" ||
		firstFinding.GetObserved() != "coverage thresholds below policy" ||
		firstFinding.GetRemediation() != "Restore the template policy projection." {
		t.Fatalf("rich finding fields not mapped: %+v", firstFinding)
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

// sharedHandlerWith builds a SharedHandler whose engine uses the fake
// discoverer, so ValidateScenario runs end-to-end without Code Facts.
func sharedHandlerWith(t *testing.T) *SharedHandler {
	t.Helper()
	svc := internalvalidation.New()
	svc.Spec = testSpec(t)
	svc.Discoverer = fakeDiscoverer{inv: goSurfaceInventory(t)}
	return NewSharedHandler(NewHandlerWithDeps(Deps{Service: svc, MaturitySpec: testSpec(t)}))
}

func stageNames(stages []*commonStage) []string {
	names := make([]string, 0, len(stages))
	for _, s := range stages {
		names = append(names, s.GetName())
	}
	return names
}

func hasStage(stages []*commonStage, name string) bool {
	for _, n := range stageNames(stages) {
		if n == name {
			return true
		}
	}
	return false
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := sharedHandlerWith(t)
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the shared validation response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}

func TestValidateScenarioPacksNativeDetail(t *testing.T) {
	h := sharedHandlerWith(t)
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	detail := resp.Msg.GetNativeDetail()
	if detail == nil {
		t.Fatal("native_detail must be packed in the shared response")
	}
	native := &validationv1.ValidateScenarioResponse{}
	if err := detail.UnmarshalTo(native); err != nil {
		t.Fatalf("native_detail unmarshal failed: %v", err)
	}
	if native.GetScenario() != "demo" {
		t.Fatalf("native scenario = %q, want demo", native.GetScenario())
	}
	if native.GetAssessment() == nil {
		t.Fatal("native assessment must remain available beside rich findings")
	}
	if len(native.GetFindings()) != 0 {
		t.Fatalf("expected clean fake scenario to have no native findings, got %d", len(native.GetFindings()))
	}
}

func TestPreviewFixReportsDeterministicCandidatesWithoutWriting(t *testing.T) {
	root := seedFixTarget(t)
	h := NewSharedHandler(NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)}))

	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		Path:     root,
	}))
	if err != nil {
		t.Fatalf("PreviewFix: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatal("preview must not mark response applied")
	}
	if len(resp.Msg.GetCandidates()) != 3 {
		t.Fatalf("candidate count = %d, want 3: %+v", len(resp.Msg.GetCandidates()), resp.Msg.GetCandidates())
	}
	vitePath := filepath.Join(root, "ui", "vite.config.ts")
	got, err := os.ReadFile(vitePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "setupFiles") || strings.Contains(string(got), "branches: 85") {
		t.Fatalf("preview wrote vite config:\n%s", got)
	}
}

func TestApplyFixWritesDeterministicCandidates(t *testing.T) {
	root := seedFixTarget(t)
	h := NewSharedHandler(NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)}))

	resp, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		Path:     root,
		RuleIds:  []string{"UNIT_POLICY_PROJECTION_DRIFT"},
	}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	if !resp.Msg.GetApplied() {
		t.Fatal("apply must mark response applied")
	}
	if len(resp.Msg.GetCandidates()) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(resp.Msg.GetCandidates()))
	}
	for _, c := range resp.Msg.GetCandidates() {
		if !c.GetApplied() {
			t.Fatalf("candidate not marked applied: %+v", c)
		}
	}
	vite, err := os.ReadFile(filepath.Join(root, "ui", "vite.config.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vite), "setupFiles: ['./src/test-setup.ts']") || !strings.Contains(string(vite), "branches: 85") {
		t.Fatalf("vite config not repaired:\n%s", vite)
	}
	pkg, err := os.ReadFile(filepath.Join(root, "ui", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkg), `"test": "vitest run"`) || !strings.Contains(string(pkg), `"test:coverage": "vitest run --coverage"`) {
		t.Fatalf("package scripts not repaired:\n%s", pkg)
	}
}

func seedFixTarget(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeHandlerFile(t, filepath.Join(repo, "scenarios", "test-genie", "schemas", "testing.schema.json"), "{}\n")
	root := filepath.Join(repo, "scenarios", "demo")
	writeHandlerFile(t, filepath.Join(root, ".vrooli", "testing.json"), `{
  "$schema": "../../../../scripts/scenarios/testing/schemas/testing.schema.json",
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [{"role": "ui", "policy_class": "react_vite_ui"}],
      "policy_classes": {"react_vite_ui": {"language": "typescript", "framework": "vitest"}},
      "customization": {"mode": "monotonic", "waivers": []}
    }
  }
}`)
	writeHandlerFile(t, filepath.Join(root, "ui", "package.json"), `{
  "scripts": {
    "test": "jest"
  },
  "devDependencies": {
    "vitest": "^3.0.0"
  }
}`)
	writeHandlerFile(t, filepath.Join(root, "ui", "vite.config.ts"), `export default {
  test: {
    environment: 'jsdom',
    coverage: {
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 70,
        statements: 85,
      },
    },
  },
};
`)
	return root
}

func writeHandlerFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestExecuteStageGatedByIncludeExecution proves the always-on stages
// (discover, static-analysis) appear regardless of the flag, while the execute
// stage appears ONLY when include_execution is set — a default validation's
// profile is never polluted by execute-path timing.
func TestExecuteStageGatedByIncludeExecution(t *testing.T) {
	t.Run("default omits execute", func(t *testing.T) {
		h := sharedHandlerWith(t)
		resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
		if err != nil {
			t.Fatalf("ValidateScenario: %v", err)
		}
		stages := resp.Msg.GetMetrics().GetStages()
		if !hasStage(stages, "discover") || !hasStage(stages, "static-analysis") {
			t.Fatalf("always-on stages missing: %v", stageNames(stages))
		}
		if hasStage(stages, "execute") {
			t.Fatalf("execute stage must be absent when include_execution=false: %v", stageNames(stages))
		}
	})

	t.Run("include_execution opens execute", func(t *testing.T) {
		h := sharedHandlerWith(t)
		resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", IncludeExecution: true}))
		if err != nil {
			t.Fatalf("ValidateScenario: %v", err)
		}
		stages := resp.Msg.GetMetrics().GetStages()
		if !hasStage(stages, "discover") || !hasStage(stages, "static-analysis") {
			t.Fatalf("always-on stages missing: %v", stageNames(stages))
		}
		if !hasStage(stages, "execute") {
			t.Fatalf("execute stage must be present when include_execution=true: %v", stageNames(stages))
		}
	})
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
