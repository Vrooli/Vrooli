package validation

import (
	"context"
	"errors"
	"log"
	"os"
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
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
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
		manifestvalidation.CodeCLICommandUndeclared,
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
	if len(spec.Capabilities) != 5 {
		t.Fatalf("capabilities = %d, want 5", len(spec.Capabilities))
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
