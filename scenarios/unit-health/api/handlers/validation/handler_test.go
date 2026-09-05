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

	"unit-health/internal/discovery"
	internalvalidation "unit-health/internal/validation"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
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

func (f fakeDiscoverer) Discover(context.Context, string, string, string, bool) (discovery.Inventory, error) {
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
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("load descriptor maturity: %v", err)
	}
	return spec
}

func TestResponseToProtoMapsFields(t *testing.T) {
	spec := testSpec(t)
	out, err := responseToProto(protoMappingFixture(), spec)
	if err != nil {
		t.Fatalf("responseToProto: %v", err)
	}

	assertResponseScalars(t, out)
	assertResponseCollections(t, out)
	assertResponseProjectionChecks(t, out)
	assertResponseFindingDetails(t, out)
	assertResponseArtifacts(t, out)
}

func protoMappingFixture() internalvalidation.Response {
	return internalvalidation.Response{
		RunID:      "uh-123",
		Status:     "passed",
		Scenario:   "demo",
		TargetKind: "scenario",
		TargetPath: "/x",
		Summary:    "ok",
		Surfaces:   []internalvalidation.Surface{{ID: "api", Kind: "api", Language: "go"}},
		Workspaces: []internalvalidation.Workspace{{ID: "api", Language: "go"}},
		Coverage:   []internalvalidation.CoverageTarget{{ID: "api:a.go", FilePath: "a.go", CoveragePercent: 80}},
		ProjectionChecks: []internalvalidation.ProjectionCheck{{
			ID:          "api:testutil.production_import_ban",
			WorkspaceID: "api",
			SurfaceID:   "api",
			Key:         "testutil.production_import_ban",
			Owner:       "go test native guard",
			FilePath:    "api/internal/testutil/no_prod_import_test.go",
			PolicyValue: "true",
			NativeValue: "false",
			Status:      "missing",
			Remediation: "Add the import-ban meta-test.",
			FindingCode: "UNIT_POLICY_PROJECTION_DRIFT",
		}},
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
}

func assertResponseScalars(t *testing.T, out *validationv1.ValidateScenarioResponse) {
	t.Helper()
	if out.GetRunId() != "uh-123" || out.GetStatus() != "passed" || out.GetScenario() != "demo" {
		t.Errorf("scalar fields not mapped: %+v", out)
	}
	if out.GetCounts().GetWarnings() != 2 || out.GetCounts().GetSurfaces() != 1 || out.GetCounts().GetCoverageTargets() != 1 {
		t.Errorf("counts wrong: %+v", out.GetCounts())
	}
	if out.GetAssessment() == nil {
		t.Error("maturity assessment must be built")
	}
	if out.GetMaturity().GetRung() != 5 {
		t.Errorf("maturity rung = %d, want 5", out.GetMaturity().GetRung())
	}
}

func assertResponseCollections(t *testing.T, out *validationv1.ValidateScenarioResponse) {
	t.Helper()
	if len(out.GetFindings()) != 2 || len(out.GetSurfaces()) != 1 || len(out.GetWorkspaces()) != 1 {
		t.Errorf("collections not mapped: findings=%d surfaces=%d", len(out.GetFindings()), len(out.GetSurfaces()))
	}
}

func assertResponseProjectionChecks(t *testing.T, out *validationv1.ValidateScenarioResponse) {
	t.Helper()
	if len(out.GetProjectionChecks()) != 1 {
		t.Fatalf("projection checks not mapped: got %d, want 1", len(out.GetProjectionChecks()))
	}
	got := out.GetProjectionChecks()[0]
	if got.GetPolicyValue() != "true" || got.GetNativeValue() != "false" || got.GetStatus() != "missing" {
		t.Fatalf("projection check fields not mapped: %+v", got)
	}
}

func assertResponseFindingDetails(t *testing.T, out *validationv1.ValidateScenarioResponse) {
	t.Helper()
	firstFinding := out.GetFindings()[0]
	if firstFinding.GetEvidence() != "role=ui policy_class=react_vite_ui" ||
		firstFinding.GetExpected() != "coverage thresholds >= 85" ||
		firstFinding.GetObserved() != "coverage thresholds below policy" ||
		firstFinding.GetRemediation() != "Restore the template policy projection." {
		t.Fatalf("rich finding fields not mapped: %+v", firstFinding)
	}
}

func assertResponseArtifacts(t *testing.T, out *validationv1.ValidateScenarioResponse) {
	t.Helper()
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

func TestValidateTargetPreservesPackageKind(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/package\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	svc := internalvalidation.New()
	svc.Spec = testSpec(t)
	svc.Discoverer = fakeDiscoverer{inv: discovery.Inventory{Scenario: "api-core", TargetKind: "package", RootPath: root, Surfaces: []discovery.Surface{{ID: "package", Language: "go", RootPath: root, Status: "known"}}}}
	h := NewSharedHandler(NewHandlerWithDeps(Deps{Service: svc, MaturitySpec: svc.Spec}))
	resp, err := h.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{Target: &commonv1.ValidationTarget{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE, Id: "api-core"}, Path: root}))
	if err != nil {
		t.Fatalf("ValidateTarget: %v", err)
	}
	if resp.Msg.GetTarget().GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE {
		t.Fatalf("returned target = %v, want package", resp.Msg.GetTarget().GetKind())
	}
	if resp.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		t.Fatal("ValidateTarget returned unspecified status")
	}
	if resp.Msg.GetAssessment() == nil {
		t.Fatal("ValidateTarget must return an assessment")
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
	if len(resp.Msg.GetCandidates()) != 6 {
		t.Fatalf("candidate count = %d, want 6: %+v", len(resp.Msg.GetCandidates()), resp.Msg.GetCandidates())
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
	if len(resp.Msg.GetCandidates()) != 5 {
		t.Fatalf("candidate count = %d, want 5", len(resp.Msg.GetCandidates()))
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
	if !strings.Contains(string(vite), "setupFiles: ['./src/test-setup.ts']") ||
		!strings.Contains(string(vite), "branches: 85") ||
		!strings.Contains(string(vite), "reportOnFailure: true") ||
		!strings.Contains(string(vite), "include: ['src/**/*.{ts,tsx}']") ||
		!strings.Contains(string(vite), "'src/test-utils/**'") {
		t.Fatalf("vite config not repaired:\n%s", vite)
	}
	pkg, err := os.ReadFile(filepath.Join(root, "ui", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkg), `"test": "vitest run"`) || !strings.Contains(string(pkg), `"test:coverage": "vitest run --coverage"`) {
		t.Fatalf("package scripts not repaired:\n%s", pkg)
	}
	renderHelper, err := os.ReadFile(filepath.Join(root, "ui", "src", "test-utils", "renderWithProviders.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(renderHelper), "renderWithProviders") {
		t.Fatalf("render helper not repaired:\n%s", renderHelper)
	}
	eslint, err := os.ReadFile(filepath.Join(root, "ui", "eslint.config.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(eslint), "no-restricted-imports") || !strings.Contains(string(eslint), "@/features/*/mocks/*") {
		t.Fatalf("eslint import ban not repaired:\n%s", eslint)
	}
}

func TestApplyFixRaisesCoverageThresholdsToPolicyFloor(t *testing.T) {
	root := seedFixTarget(t)
	writeHandlerFile(t, filepath.Join(root, ".vrooli", "testing.json"), `{
  "$schema": "../../../../scenarios/test-genie/schemas/testing.schema.json",
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [{"role": "ui", "policy_class": "react_vite_ui"}],
      "policy_classes": {
        "react_vite_ui": {
          "language": "typescript",
          "framework": "vitest",
          "coverage": {"minimum_percent": 90}
        }
      },
      "customization": {"mode": "monotonic", "waivers": []}
    }
  }
}`)
	h := NewSharedHandler(NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)}))

	_, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		Path:     root,
		RuleIds:  []string{"UNIT_POLICY_PROJECTION_DRIFT"},
	}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	vite, err := os.ReadFile(filepath.Join(root, "ui", "vite.config.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vite), "branches: 90") {
		t.Fatalf("vite config threshold not raised to policy floor:\n%s", vite)
	}
}

func TestApplyFixRepairsVitestProjectionFieldsFromPolicy(t *testing.T) {
	root := seedFixTarget(t)
	writeHandlerFile(t, filepath.Join(root, ".vrooli", "testing.json"), `{
  "$schema": "../../../../scenarios/test-genie/schemas/testing.schema.json",
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [{"role": "ui", "policy_class": "react_vite_ui"}],
      "policy_classes": {
        "react_vite_ui": {
          "language": "typescript",
          "framework": "vitest",
          "coverage": {
            "minimum_percent": 88,
            "provider": "v8",
            "reporters": ["text", "json-summary", "json"]
          },
          "projection": {
            "settings": {
              "environment": "jsdom",
              "setup_files": ["./src/test-setup.ts"]
            }
          }
        }
      },
      "customization": {"mode": "monotonic", "waivers": []}
    }
  }
}`)
	writeHandlerFile(t, filepath.Join(root, "ui", "package.json"), `{
  "scripts": {
    "test": "vitest run",
    "test:coverage": "jest --coverage"
  },
  "devDependencies": {
    "vitest": "^3.0.0"
  }
}`)
	writeHandlerFile(t, filepath.Join(root, "ui", "vite.config.ts"), `export default {
  test: {
    environment: 'node',
    setupFiles: ['./src/legacy-setup.ts'],
    coverage: {
      provider: 'istanbul',
      reporter: ['text'],
      thresholds: {
        lines: 80,
        functions: 80,
        branches: 80,
        statements: 80,
      },
    },
  },
};
`)
	h := NewSharedHandler(NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)}))

	_, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		Path:     root,
		RuleIds:  []string{"UNIT_POLICY_PROJECTION_DRIFT"},
	}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}

	pkg, err := os.ReadFile(filepath.Join(root, "ui", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkg), `"test:coverage": "vitest run --coverage"`) {
		t.Fatalf("package coverage script was not repaired:\n%s", pkg)
	}
	vite, err := os.ReadFile(filepath.Join(root, "ui", "vite.config.ts"))
	if err != nil {
		t.Fatal(err)
	}
	viteText := string(vite)
	for _, want := range []string{
		"environment: 'jsdom'",
		"setupFiles: ['./src/test-setup.ts']",
		"provider: 'v8'",
		"reporter: ['text', 'json-summary', 'json']",
		"reportOnFailure: true",
		"include: ['src/**/*.{ts,tsx}']",
		"'src/test-utils/**'",
		"lines: 88",
		"functions: 88",
		"branches: 88",
		"statements: 88",
	} {
		if !strings.Contains(viteText, want) {
			t.Fatalf("vite config missing %q after repair:\n%s", want, viteText)
		}
	}
}

func TestApplyFixDoesNotTreatCommentedImportBanAsRepaired(t *testing.T) {
	root := seedFixTarget(t)
	writeHandlerFile(t, filepath.Join(root, "ui", "eslint.config.js"), `export default [{
  rules: {
    // "no-restricted-imports" "@/test-utils/*" "@/features/*/mocks/*"
  },
}];
`)
	h := NewSharedHandler(NewHandlerWithDeps(Deps{MaturitySpec: testSpec(t)}))

	_, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		Path:     root,
		RuleIds:  []string{"UNIT_POLICY_PROJECTION_DRIFT"},
	}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	eslint, err := os.ReadFile(filepath.Join(root, "ui", "eslint.config.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(eslint), `"no-restricted-imports": ["error"`) {
		t.Fatalf("commented import ban was not repaired:\n%s", eslint)
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
