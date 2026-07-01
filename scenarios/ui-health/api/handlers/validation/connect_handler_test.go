package validation

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	"ui-health/internal/services/manifestvalidation"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestBuildMaturityAssessmentMapsUIFindings(t *testing.T) {
	spec := testMaturitySpec()
	report := manifestvalidation.Report{
		Scenario: "demo",
		Findings: []manifestvalidation.Finding{{
			Severity:   manifestvalidation.SeverityError,
			Code:       "overlay_unknown_slot",
			Location:   "scenarios/demo/.vrooli/ui-manifest.json",
			Message:    "unknown slot",
			Suggestion: "remove slot",
		}},
	}
	got, err := buildMaturityAssessment(report, spec)
	if err != nil {
		t.Fatalf("buildMaturityAssessment returned error: %v", err)
	}
	if got.GetProvider() != "ui-health" || got.GetPhase() != "ui-health" {
		t.Fatalf("assessment identity = %s/%s, want ui-health/ui-health", got.GetProvider(), got.GetPhase())
	}
	if got.GetLocal().GetCurrentLevel() != "L2" || got.GetLocal().GetNextLevel() != "L3" {
		t.Fatalf("local maturity = current %q next %q, want L2/L3", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetFindings()[0].GetMaturity().GetGlobalImpact() != commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP {
		t.Fatalf("global impact = %v, want EVOLVABILITY_GAP", got.GetFindings()[0].GetMaturity().GetGlobalImpact())
	}
}

func TestMaturitySpecCoversUIHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"no_ui_surface",
		"ui_predates_template_layout",
		"service_json_missing",
		"service_json_invalid",
		"template_id_missing",
		"template_unknown",
		"template_manifest_invalid",
		"contract_kind_mismatch",
		"contract_schema_mismatch",
		"slots_empty",
		"overlay_read_failed",
		"overlay_invalid",
		"overlay_unknown_slot",
		"slot_dir_empty",
		"slot_dir_missing",
		"slot_dir_stat_failed",
		"slot_dir_not_directory",
		"slot_parent_dir_missing",
		"slot_instances_empty",
		"path_pattern_unknown_token",
		"slot_dir_overlap_equal",
		"runtime_render_ok",
		"runtime_handshake_failed",
		"runtime_network_failure",
		"runtime_render_broken",
		"runtime_page_error",
		"runtime_load_failed",
		"runtime_render_failed",
		"runtime_console_errors",
		"runtime_skipped_ui_unavailable",
		"runtime_skipped_bas_unavailable",
		"standard_pwa_manifest",
		"pwa_manifest_install_fields",
		"pwa_launch_scope",
		"pwa_service_worker_offline",
		"pwa_optional_platform_fields",
	} {
		if _, ok := spec.Findings[code]; !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
	}
}

func TestMaturitySpecEmitsCapabilityAssessment(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	report := manifestvalidation.Report{
		Scenario: "demo",
		Findings: []manifestvalidation.Finding{
			{
				Severity: manifestvalidation.SeverityError,
				Code:     "overlay_unknown_slot",
				Message:  "unknown slot",
			},
			{
				Severity: manifestvalidation.SeverityWarning,
				Code:     "standard_pwa_manifest",
				Message:  "missing PWA manifest metadata",
			},
		},
	}
	got, err := buildMaturityAssessment(report, spec)
	if err != nil {
		t.Fatalf("buildMaturityAssessment returned error: %v", err)
	}
	if len(got.GetCapabilities()) != 6 {
		t.Fatalf("capabilities = %d, want 6", len(got.GetCapabilities()))
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() != "manifest_contract" {
		t.Fatalf("highest priority = %#v, want manifest_contract", got.GetHighestPriorityCapability())
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "manifest_contract" {
		t.Fatalf("overlay capability = %q, want manifest_contract", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetFindings()[1].GetMaturity().GetCapabilityId() != "pwa_native_readiness" {
		t.Fatalf("pwa capability = %q, want pwa_native_readiness", got.GetFindings()[1].GetMaturity().GetCapabilityId())
	}
}

type stubValidator struct{}

func (s *stubValidator) ValidateScenario(_ context.Context, scenario string) (manifestvalidation.Report, error) {
	if scenario == "" {
		return manifestvalidation.Report{}, errors.New("scenario is required")
	}
	return manifestvalidation.Report{Scenario: scenario, Passed: true}, nil
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger:       log.New(log.Writer(), "", 0),
		Validator:    &stubValidator{},
		MaturitySpec: testMaturitySpec(),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "ui-health"}))
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
		Provider: "ui-health",
		Phase:    "ui-health",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No UI"},
			{ID: "L1", Name: "Template readable"},
			{ID: "L2", Name: "Contract valid"},
			{ID: "L3", Name: "Overlay compatible"},
		},
		Findings: map[string]assessment.FindingMapping{
			"overlay_unknown_slot": {
				LocalLevelImpact:    "L3",
				GlobalImpact:        assessment.ImpactEvolvabilityGap,
				Dimension:           "ui",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"ui-health"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L3",
			GlobalImpact:     assessment.ImpactEvolvabilityGap,
			Dimension:        "ui",
			SeverityDefault:  "SEVERITY_ERROR",
		},
	}
	if err := assessment.ValidateSpec(*spec); err != nil {
		panic(err)
	}
	return spec
}
