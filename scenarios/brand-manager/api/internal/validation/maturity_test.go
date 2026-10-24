package validation

import (
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// scenarioDir is brand-manager's root relative to this test package
// (api/internal/validation → api → brand-manager).
const scenarioDir = "../../.."

// TestEveryRuleHasMaturityMapping is the anti-drift guard between the rule
// registry and .vrooli/maturity.json: every rule the scan can emit MUST have an
// explicit findings entry, or it would silently fall back to the L1 mapping and
// mis-rank every scenario that trips it.
func TestEveryRuleHasMaturityMapping(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		t.Fatalf("load maturity spec: %v", err)
	}
	for _, s := range specs {
		mapping, ok := spec.Findings[s.id]
		if !ok {
			t.Fatalf("rule %q has no entry in .vrooli/maturity.json findings", s.id)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("rule %q has no capability_id in .vrooli/maturity.json findings", s.id)
		}
	}
	if len(spec.Capabilities) != 6 {
		t.Fatalf("capabilities = %d, want 6", len(spec.Capabilities))
	}
	if spec.Fallback.CapabilityID == "" {
		t.Fatal("fallback must declare capability_id")
	}
}

// TestMaturityAssessmentBuildsForBrandManager confirms the spec + a representative
// finding set produce a valid assessment without error (the served path).
func TestMaturityAssessmentBuildsForBrandManager(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	findings := []Finding{
		{RuleID: "open-graph", Severity: SeverityWarning, Title: "og", AutofixAvailable: true},
		{RuleID: "has-display-name", Severity: SeverityError, Title: "name"},
	}
	got, err := BuildMaturityAssessment("brand-manager", findings, *spec)
	if err != nil {
		t.Fatalf("BuildMaturityAssessment: %v", err)
	}
	if got == nil || got.GetLocal() == nil {
		t.Fatal("expected a populated maturity assessment")
	}
	if len(got.GetCapabilities()) != 6 {
		t.Fatalf("capabilities = %d, want 6", len(got.GetCapabilities()))
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "share_and_product_surfaces" {
		t.Fatalf("open-graph capability = %q, want share_and_product_surfaces", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetFindings()[1].GetMaturity().GetCapabilityId() != "identity_contract" {
		t.Fatalf("has-display-name capability = %q, want identity_contract", got.GetFindings()[1].GetMaturity().GetCapabilityId())
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() == "" {
		t.Fatalf("highest priority must identify a capability: %#v", got.GetHighestPriorityCapability())
	}
}
