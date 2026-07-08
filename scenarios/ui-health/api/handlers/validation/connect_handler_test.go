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

func TestReportGroupsPreserveCompositionOrder(t *testing.T) {
	h := NewConnectHandler(Deps{Logger: log.New(log.Writer(), "", 0)})
	groups := h.reportGroups(context.Background(), "demo", "/tmp/demo", true)
	want := []string{"static-interop", "static-freshness", "runtime-render"}
	if len(groups) != len(want) {
		t.Fatalf("groups = %d, want %d", len(groups), len(want))
	}
	for i, group := range groups {
		if group.name != want[i] {
			t.Fatalf("group[%d] = %q, want %q", i, group.name, want[i])
		}
		if group.run == nil {
			t.Fatalf("group[%d] has nil run function", i)
		}
	}
}

func TestMaturitySpecCoversUIHealthFindings(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
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
		"standard_no_raw_hex",
		"standard_unused_custom_component",
		"standard_raw_primitive_overuse",
		"standard_component_location",
		"standard_component_version_staleness",
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
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
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

func TestRunInteropFindingsIncludesComponentCanonFromDesignIntent(t *testing.T) {
	root := t.TempDir()
	writeValidationFixture(t, root, ".vrooli/service.json", `{"generation":{"design":{"adapter":"react-vite-tailwind"}}}`)
	writeValidationFixture(t, root, "ui/package.json", `{"dependencies":{"react":"^18.3.1"}}`)
	writeValidationFixture(t, root, "ui/src/pages/Fleet.tsx", `export function Fleet() {
  return <table><tbody><tr><td>x</td></tr></tbody></table>;
}`)

	findings := runInteropFindings(root, "demo")
	for _, finding := range findings {
		if finding.Code == "standard_raw_primitive_overuse" {
			if finding.Severity != manifestvalidation.SeverityWarning {
				t.Fatalf("canon severity = %v, want warning", finding.Severity)
			}
			if finding.Location != filepath.Join(root, "ui", "src", "pages", "Fleet.tsx") {
				t.Fatalf("canon location = %q", finding.Location)
			}
			return
		}
	}
	t.Fatalf("standard_raw_primitive_overuse finding missing: %+v", findings)
}

func TestRunInteropFindingsIncludesComponentLocationCanon(t *testing.T) {
	root := t.TempDir()
	writeValidationFixture(t, root, "ui/package.json", `{"dependencies":{"react":"^18.3.1"}}`)
	writeValidationFixture(t, root, "ui/src/pages/Fleet.tsx", `function DebtTable() {
  return <section>Debt</section>;
}

export function Fleet() {
  return <DebtTable />;
}`)

	findings := runInteropFindings(root, "demo")
	for _, finding := range findings {
		if finding.Code == "standard_component_location" {
			if finding.Severity != manifestvalidation.SeverityWarning {
				t.Fatalf("component-location severity = %v, want warning", finding.Severity)
			}
			if finding.Location != filepath.Join(root, "ui", "src", "pages", "Fleet.tsx") {
				t.Fatalf("component-location location = %q", finding.Location)
			}
			return
		}
	}
	t.Fatalf("standard_component_location finding missing: %+v", findings)
}

func TestRunInteropFindingsIncludesComponentVersionStaleness(t *testing.T) {
	root := t.TempDir()
	writeValidationFixture(t, root, "go.work", `go 1.24`)
	writeValidationFixture(t, root, "scenarios/react-component-library/library/components/DataTable/component.json", `{"libraryId":"react-component-library:DataTable","latest":"1.10.0","draft":"","deprecatedVersions":[]}`)
	writeValidationFixture(t, root, "ui/package.json", `{"dependencies":{"react":"^18.3.1"}}`)
	writeValidationFixture(t, root, "ui/src/components/ui/data-table.tsx", `// @vrooliComponentSource react-component-library:DataTable
// @vrooliComponentVersion 1.0.0
export function DataTable() {
  return <table />;
}`)

	findings := runInteropFindings(root, "demo")
	for _, finding := range findings {
		if finding.Code == "standard_component_version_staleness" {
			if finding.Severity != manifestvalidation.SeverityWarning {
				t.Fatalf("component-version severity = %v, want warning", finding.Severity)
			}
			if finding.Location != filepath.Join(root, "ui", "src", "components", "ui", "data-table.tsx") {
				t.Fatalf("component-version location = %q", finding.Location)
			}
			if !strings.Contains(finding.Message, "behind catalog latest 1.10.0") {
				t.Fatalf("component-version message = %q", finding.Message)
			}
			return
		}
	}
	t.Fatalf("standard_component_version_staleness finding missing: %+v", findings)
}

func writeValidationFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
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
