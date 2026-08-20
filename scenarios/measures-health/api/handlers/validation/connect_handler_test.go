package validation

import (
	"context"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	internal "measures-health/internal/validation"

	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/measures-go/manifestscan"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeValidator struct {
	rep     internal.Report
	entries []internal.FleetEntry
}

func (f fakeValidator) ValidateScenario(_ context.Context, _ string, _ bool) (internal.Report, error) {
	return f.rep, nil
}

func (f fakeValidator) ListFleetCoverage(_ context.Context, _ []string) ([]internal.FleetEntry, error) {
	return f.entries, nil
}

func TestValidateScenario_MapsReportToProto(t *testing.T) {
	rep := internal.Report{
		Scenario: "swarm-manager",
		Passed:   false,
		Domains: []internal.DomainCoverage{
			{
				Domain: "backlog", Status: internal.StatusCovered, MeasureCount: 1, Tier: manifestscan.TierFull,
				Measures: []internal.MeasureSummary{{Name: "backlog.completed", Tier: manifestscan.TierFull, Effect: "read", QuestionCount: 2}},
			},
			{Domain: "captures", Status: internal.StatusUncovered},
		},
		Findings: []internal.Finding{
			{RuleID: "measures.uncovered-domain", Severity: internal.SeverityError, Title: "x", Remediation: "y", Scanner: "coverage"},
		},
	}
	h := NewConnectHandler(Deps{Validator: fakeValidator{rep: rep}, MaturitySpec: testMaturitySpec()})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "swarm-manager"}))
	if err != nil {
		t.Fatal(err)
	}
	m := resp.Msg
	if m.GetScenario() != "swarm-manager" || m.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("scenario/passed mismatch: %+v", m)
	}
	native := &validationv1.ScenarioCoverageReport{}
	if err := m.GetNativeDetail().UnmarshalTo(native); err != nil {
		t.Fatalf("unpack native detail: %v", err)
	}
	if native.GetSummary().GetErrors() != 1 {
		t.Fatalf("want 1 error, got %d", native.GetSummary().GetErrors())
	}
	if len(native.GetDomains()) != 2 {
		t.Fatalf("want 2 domains, got %d", len(native.GetDomains()))
	}
	d0 := native.GetDomains()[0]
	if d0.GetStatus() != validationv1.DomainStatus_DOMAIN_STATUS_COVERED || d0.GetTier() != validationv1.Tier_TIER_FULL {
		t.Fatalf("backlog mapping wrong: %+v", d0)
	}
	if len(d0.GetMeasures()) != 1 || d0.GetMeasures()[0].GetName() != "backlog.completed" {
		t.Fatalf("measure mapping wrong: %+v", d0.GetMeasures())
	}
	f0 := native.GetFindings()[0]
	if f0.GetSeverity() != validationv1.Severity_SEVERITY_ERROR || f0.GetRuleId() != "measures.uncovered-domain" {
		t.Fatalf("finding mapping wrong: %+v", f0)
	}
	assessment := m.GetAssessment()
	if assessment.GetProvider() != "measures-health" || assessment.GetPhase() != "measures" {
		t.Fatalf("assessment identity wrong: %+v", assessment)
	}
	if assessment.GetLocal().GetCurrentLevel() != "L1" || assessment.GetLocal().GetNextLevel() != "L2" {
		t.Fatalf("assessment local maturity wrong: %+v", assessment.GetLocal())
	}
	if got := assessment.GetFindingsByGlobalImpact()["capability_gap"]; got != 1 {
		t.Fatalf("global impact count = %d, want 1", got)
	}
	if assessment.GetFindings()[0].GetMaturity().GetGlobalImpact() != commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP {
		t.Fatalf("finding maturity impact wrong: %+v", assessment.GetFindings()[0].GetMaturity())
	}
}

func TestListFleetCoverage_MapsEntries(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: fakeValidator{entries: []internal.FleetEntry{
		{Scenario: "a", Passed: true, Expected: 2, Covered: 2, WorstTier: manifestscan.TierFull, MeasureCount: 3},
		{Scenario: "b", Passed: false, Expected: 1, Uncovered: 1},
	}}})
	resp, err := h.ListFleetCoverage(context.Background(), connect.NewRequest(&validationv1.ListFleetCoverageRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetEntries()) != 2 {
		t.Fatalf("want 2 entries, got %d", len(resp.Msg.GetEntries()))
	}
	if resp.Msg.GetEntries()[0].GetWorstTier() != validationv1.Tier_TIER_FULL {
		t.Fatalf("tier mapping wrong: %+v", resp.Msg.GetEntries()[0])
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	rep := internal.Report{
		Scenario: "swarm-manager",
		Passed:   true,
		Domains: []internal.DomainCoverage{
			{Domain: "backlog", Status: internal.StatusCovered, MeasureCount: 1, Tier: manifestscan.TierFull},
		},
	}
	h := NewConnectHandler(Deps{
		Validator:    fakeValidator{rep: rep},
		MaturitySpec: testMaturitySpec(),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "swarm-manager"}))
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
	// native_detail must still be populated (coverage report payload unchanged).
	native := &validationv1.ScenarioCoverageReport{}
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(native); err != nil {
		t.Fatalf("native_detail must remain populated: %v", err)
	}
	if native.GetScenario() != "swarm-manager" {
		t.Fatalf("native_detail scenario = %q, want %q", native.GetScenario(), "swarm-manager")
	}
}

func testMaturitySpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "measures-health",
		Phase:    "measures",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "contract readable"},
			{ID: "L1", Name: "domains derived"},
			{ID: "L2", Name: "domains covered"},
		},
		Findings: map[string]assessment.FindingMapping{
			"measures.uncovered-domain": {
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactCapabilityGap,
				Dimension:           "measures",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"measures-adoption"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "measures",
			SeverityDefault:  "SEVERITY_WARNING",
		},
	}
}
