package validation

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"cli-health/internal/services/manifestvalidation"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type stubValidator struct {
	called bool
}

func (s *stubValidator) ValidateScenario(_ context.Context, scenario string) (manifestvalidation.Report, error) {
	s.called = true
	if scenario == "" {
		return manifestvalidation.Report{}, errors.New("scenario is required")
	}
	return manifestvalidation.Report{Scenario: scenario, Passed: true}, nil
}

func TestValidate_ReservedNameRejected(t *testing.T) {
	v := &stubValidator{}
	h := NewConnectHandler(Deps{
		Logger:        log.New(log.Writer(), "", 0),
		Validator:     v,
		ReservedNames: []string{"vrooli"},
		MaturitySpec:  testMaturitySpec(),
	})
	req := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "vrooli"})
	_, err := h.ValidateScenario(context.Background(), req)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", ce.Code())
	}
	if !strings.Contains(ce.Message(), "not a scenario") {
		t.Errorf("message = %q, want substring %q", ce.Message(), "not a scenario")
	}
	if v.called {
		t.Error("validator should not be called for reserved names")
	}
}

func TestValidate_ScenarioPassesThrough(t *testing.T) {
	v := &stubValidator{}
	h := NewConnectHandler(Deps{
		Logger:        log.New(log.Writer(), "", 0),
		Validator:     v,
		ReservedNames: []string{"vrooli"},
		MaturitySpec:  testMaturitySpec(),
	})
	req := connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "cli-health"})
	resp, err := h.ValidateScenario(context.Background(), req)
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !v.called {
		t.Error("validator was not called")
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Errorf("status=%v; want PASSED", resp.Msg.GetStatus())
	}
}

func TestValidateTarget_ProjectBypassesReservedScenarioGuard(t *testing.T) {
	v := &stubValidator{}
	h := NewConnectHandler(Deps{
		Logger:        log.New(log.Writer(), "", 0),
		Validator:     v,
		ReservedNames: []string{"vrooli", "repo"},
		MaturitySpec:  testMaturitySpec(),
	})
	resp, err := h.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT,
			Id:   manifestvalidation.ProjectTargetID,
		},
	}))
	if err != nil {
		t.Fatalf("ValidateTarget project: %v", err)
	}
	if !v.called || resp.Msg.GetTarget().GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT {
		t.Fatalf("project target was not delegated: called=%v target=%v", v.called, resp.Msg.GetTarget())
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status=%v; want PASSED", resp.Msg.GetStatus())
	}
}

func TestBuildMaturityAssessmentMapsCLIFindings(t *testing.T) {
	spec := testMaturitySpec()
	report := manifestvalidation.Report{
		Scenario: "demo",
		Findings: []manifestvalidation.Finding{{
			Severity:   manifestvalidation.SeverityError,
			Code:       manifestvalidation.CodeBindingUnknownMethod,
			Location:   "scenarios/demo/cli/manifest.json",
			Message:    "bad binding",
			Suggestion: "fix method",
		}},
	}
	got, err := buildMaturityAssessment(report, spec)
	if err != nil {
		t.Fatalf("buildMaturityAssessment returned error: %v", err)
	}
	if got.GetProvider() != "cli-health" || got.GetPhase() != "contracts" {
		t.Fatalf("assessment identity = %s/%s, want cli-health/contracts", got.GetProvider(), got.GetPhase())
	}
	if got.GetLocal().GetCurrentLevel() != "L1" || got.GetLocal().GetNextLevel() != "L2" {
		t.Fatalf("local maturity = current %q next %q, want L1/L2", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetFindings()[0].GetMaturity().GetGlobalImpact() != commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP {
		t.Fatalf("global impact = %v, want EVOLVABILITY_GAP", got.GetFindings()[0].GetMaturity().GetGlobalImpact())
	}
}

func TestMaturitySpecCoversCLIHealthFindings(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		manifestvalidation.CodeManifestMissing,
		manifestvalidation.CodeManifestRequired,
		manifestvalidation.CodeManifestParseError,
		manifestvalidation.CodeManifestSchemaError,
		manifestvalidation.CodeProtoBuildFailed,
		manifestvalidation.CodeBindingUnknownSvc,
		manifestvalidation.CodeBindingUnknownMethod,
		manifestvalidation.CodeBindingDuplicate,
		manifestvalidation.CodeProtoOrphanMethod,
		manifestvalidation.CodeOmissionOrphan,
		manifestvalidation.CodeMeasureInvalid,
		manifestvalidation.CodeMeasureUnknownType,
		manifestvalidation.CodeMeasureSchemaUnread,
		manifestvalidation.CodeMeasureTier,
		manifestvalidation.CodeCLIBinaryUnrunnable,
		manifestvalidation.CodeCLIHelpFailed,
		manifestvalidation.CodeCLICommandMissing,
		manifestvalidation.CodeCLICommandUndeclared,
		manifestvalidation.CodeCLIMainUnreadable,
		manifestvalidation.CodeCLIMainHeavy,
		manifestvalidation.CodeArchUnclassifiable,
		manifestvalidation.CodeArchPrimitiveUndecl,
		manifestvalidation.CodeArchPrimitiveUnverif,
		manifestvalidation.CodeProjectCLIEmpty,
		manifestvalidation.CodeArchPrimitiveMismatch,
		manifestvalidation.CodeArchMetadataInvalid,
		manifestvalidation.CodeArchClaimedViolation,
	} {
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", code)
		}
		if mapping.CleanRequirement == "" {
			t.Fatalf("maturity spec finding %q must declare clean_requirement", code)
		}
	}
	if len(spec.Capabilities) != 7 {
		t.Fatalf("capabilities = %d, want 7", len(spec.Capabilities))
	}
	if len(spec.Levels) == 0 {
		t.Fatal("maturity spec must declare local levels")
	}
	if spec.Fallback.CapabilityID == "" {
		t.Fatal("maturity spec fallback must declare capability_id")
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger:       log.New(log.Writer(), "", 0),
		Validator:    &stubValidator{},
		MaturitySpec: testMaturitySpec(),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "cli-health"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
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

func testMaturitySpec() *assessment.Spec {
	spec := &assessment.Spec{
		Provider: "cli-health",
		Phase:    "contracts",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No manifest"},
			{ID: "L1", Name: "Schema valid"},
			{ID: "L2", Name: "Bindings valid"},
		},
		Findings: map[string]assessment.FindingMapping{
			manifestvalidation.CodeBindingUnknownMethod: {
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactEvolvabilityGap,
				Dimension:           "contracts",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"cli-steer"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L2",
			GlobalImpact:     assessment.ImpactEvolvabilityGap,
			Dimension:        "contracts",
			SeverityDefault:  "SEVERITY_ERROR",
		},
	}
	if err := assessment.ValidateSpec(*spec); err != nil {
		panic(err)
	}
	return spec
}

// cliHealthScenarioDir resolves scenarios/cli-health from this test file's
// location so tests can load the real provider descriptor.
func cliHealthScenarioDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// TestRealDescriptor_CommandArchitectureCapability loads the real
// .vrooli/test-genie.json descriptor (LoadSpecFromScenario validates the whole
// maturity spec) and asserts the new command_architecture capability and its
// finding mappings are present and well-formed.
func TestRealDescriptor_CommandArchitectureCapability(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(cliHealthScenarioDir(t))
	if err != nil {
		t.Fatalf("load + validate real cli-health descriptor: %v", err)
	}
	var capSpec *assessment.CapabilitySpec
	for i := range spec.Capabilities {
		if spec.Capabilities[i].ID == "command_architecture" {
			capSpec = &spec.Capabilities[i]
		}
	}
	if capSpec == nil {
		t.Fatal("real descriptor is missing the command_architecture capability")
	}
	wantLevels := []string{"L0", "L1", "L2", "L3", "L4"}
	if len(capSpec.Levels) != len(wantLevels) {
		t.Fatalf("command_architecture has %d levels, want %d", len(capSpec.Levels), len(wantLevels))
	}
	for i, l := range capSpec.Levels {
		if l.ID != wantLevels[i] {
			t.Errorf("level[%d] id = %q, want %q", i, l.ID, wantLevels[i])
		}
	}
	for _, code := range []string{
		manifestvalidation.CodeArchUnclassifiable,
		manifestvalidation.CodeArchPrimitiveUndecl,
		manifestvalidation.CodeArchPrimitiveUnverif,
		manifestvalidation.CodeArchPrimitiveMismatch,
		manifestvalidation.CodeArchMetadataInvalid,
		manifestvalidation.CodeArchClaimedViolation,
	} {
		m, ok := spec.Findings[code]
		if !ok {
			t.Errorf("real descriptor does not map arch finding %q", code)
			continue
		}
		if m.CapabilityID != "command_architecture" {
			t.Errorf("finding %q maps to capability %q, want command_architecture", code, m.CapabilityID)
		}
	}
}

// TestRealDescriptor_PrimitiveUndeclaredCapsAtL3 proves end-to-end scoring
// through the real descriptor: a single advisory arch.primitive_undeclared
// finding caps the command_architecture capability at L3 (renderer separation
// not yet confirmed) without failing the phase — the honest-debt contract.
func TestRealDescriptor_PrimitiveUndeclaredCapsAtL3(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(cliHealthScenarioDir(t))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	rep := manifestvalidation.Report{
		Scenario: "target",
		Findings: []manifestvalidation.Finding{{
			Severity: manifestvalidation.SeverityWarning,
			Code:     manifestvalidation.CodeArchPrimitiveUndecl,
			Message:  "command declares no primitive",
		}},
	}
	a, err := buildMaturityAssessment(rep, spec)
	if err != nil {
		t.Fatalf("build assessment: %v", err)
	}
	got := ""
	for _, c := range a.GetCapabilities() {
		if c.GetId() == "command_architecture" {
			got = c.GetCurrentLevel()
		}
	}
	if got != "L3" {
		t.Fatalf("command_architecture current level = %q, want L3", got)
	}
}
