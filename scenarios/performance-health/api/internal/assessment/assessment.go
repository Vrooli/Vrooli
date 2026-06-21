// Package assessment converts performance-health readiness findings into the
// shared common.v1.MaturityAssessment that the scenario-validation contract
// carries to test-genie. It composes packages/maturity-go/assessment so the
// scoring + clean-requirement rollups stay centralized.
//
// SCAFFOLD (P4): readiness emits no findings yet, so the assessment is the
// empty-findings baseline for the scenario's maturity spec. The mapping enriches
// automatically once readiness emits real findings in P5.
package assessment

import (
	"github.com/vrooli/maturity-go/assessment"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Finding is the performance-health-facing finding shape the assessment domain
// converts into the shared maturity finding.
type Finding struct {
	Code             string
	Severity         string
	Title            string
	Message          string
	Location         string
	AutofixAvailable bool
}

// Build constructs a MaturityAssessment for the scenario from the supplied
// findings and maturity spec. A nil spec yields a nil assessment (the caller
// degrades gracefully) — never a panic.
func Build(scenario string, spec *assessment.Spec, findings []Finding) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	shared := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		shared = append(shared, assessment.Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Title,
			Message:          f.Message,
			Location:         f.Location,
			AutofixAvailable: f.AutofixAvailable,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: scenario,
		Spec:     *spec,
		Findings: shared,
	})
}
