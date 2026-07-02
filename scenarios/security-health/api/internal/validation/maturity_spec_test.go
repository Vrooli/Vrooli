package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversSecurityHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}

	if spec.Provider != "security-health" {
		t.Fatalf("provider = %q, want security-health", spec.Provider)
	}
	if spec.Phase != "security" {
		t.Fatalf("phase = %q, want security", spec.Phase)
	}
	if spec.Version != "2.0.0" {
		t.Fatalf("version = %q, want 2.0.0", spec.Version)
	}
	if len(spec.Capabilities) != 4 {
		t.Fatalf("capabilities = %d, want 4", len(spec.Capabilities))
	}
	for _, code := range emittedSecurityHealthRuleIDs() {
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", code)
		}
	}
	if spec.Fallback.GlobalImpact != assessment.ImpactSafetyBlocker {
		t.Fatalf("fallback impact = %q, want safety_blocker", spec.Fallback.GlobalImpact)
	}
	if spec.Fallback.CapabilityID != "security_findings" {
		t.Fatalf("fallback capability = %q, want security_findings", spec.Fallback.CapabilityID)
	}

	got, err := assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: "demo",
		Spec:     *spec,
		Findings: []assessment.Finding{{Code: "gitleaks.generic", Severity: "SEVERITY_ERROR"}},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment(dynamic scanner finding) error = %v", err)
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "security_findings" {
		t.Fatalf("dynamic scanner capability = %q, want security_findings", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() != "security_findings" {
		t.Fatalf("highest priority = %#v, want security_findings", got.GetHighestPriorityCapability())
	}
}

func emittedSecurityHealthRuleIDs() []string {
	return []string{
		"security-health.scenario-not-found",
		"security-health.scanner-absent",
		"security-health.scanner-degraded",
		"security-health.substrate-unsupported",
	}
}
