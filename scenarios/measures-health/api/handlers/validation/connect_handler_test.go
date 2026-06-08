package validation

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"measures-health/internal/measurescan"
	internal "measures-health/internal/validation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation"
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
				Domain: "backlog", Status: internal.StatusCovered, MeasureCount: 1, Tier: measurescan.TierFull,
				Measures: []internal.MeasureSummary{{Name: "backlog.completed", Tier: measurescan.TierFull, Effect: "read", QuestionCount: 2}},
			},
			{Domain: "captures", Status: internal.StatusUncovered},
		},
		Findings: []internal.Finding{
			{RuleID: "measures.uncovered-domain", Severity: internal.SeverityError, Title: "x", Remediation: "y", Scanner: "coverage"},
		},
	}
	h := NewConnectHandler(Deps{Validator: fakeValidator{rep: rep}})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "swarm-manager"}))
	if err != nil {
		t.Fatal(err)
	}
	m := resp.Msg
	if m.GetScenario() != "swarm-manager" || m.GetPassed() {
		t.Fatalf("scenario/passed mismatch: %+v", m)
	}
	if m.GetSummary().GetErrors() != 1 {
		t.Fatalf("want 1 error, got %d", m.GetSummary().GetErrors())
	}
	if len(m.GetDomains()) != 2 {
		t.Fatalf("want 2 domains, got %d", len(m.GetDomains()))
	}
	d0 := m.GetDomains()[0]
	if d0.GetStatus() != validationv1.DomainStatus_DOMAIN_STATUS_COVERED || d0.GetTier() != validationv1.Tier_TIER_FULL {
		t.Fatalf("backlog mapping wrong: %+v", d0)
	}
	if len(d0.GetMeasures()) != 1 || d0.GetMeasures()[0].GetName() != "backlog.completed" {
		t.Fatalf("measure mapping wrong: %+v", d0.GetMeasures())
	}
	f0 := m.GetFindings()[0]
	if f0.GetSeverity() != validationv1.Severity_SEVERITY_ERROR || f0.GetRuleId() != "measures.uncovered-domain" {
		t.Fatalf("finding mapping wrong: %+v", f0)
	}
}

func TestListFleetCoverage_MapsEntries(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: fakeValidator{entries: []internal.FleetEntry{
		{Scenario: "a", Passed: true, Expected: 2, Covered: 2, WorstTier: measurescan.TierFull, MeasureCount: 3},
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
