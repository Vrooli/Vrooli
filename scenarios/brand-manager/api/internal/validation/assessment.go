package validation

import (
	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// severityToAssessment maps a normalized branding severity to the assessment
// vocabulary the maturity-go normalizer expects.
func severityToAssessment(s Severity) string {
	switch s {
	case SeverityError:
		return "SEVERITY_ERROR"
	case SeverityWarning:
		return "SEVERITY_WARNING"
	case SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_WARNING"
	}
}

// BuildMaturityAssessment projects branding findings onto the branding maturity
// ladder declared in brand-manager's .vrooli/maturity.json (spec) and returns
// the shared MaturityAssessment test-genie reads.
func BuildMaturityAssessment(scenario string, findings []Finding, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:             f.RuleID,
			Severity:         severityToAssessment(f.Severity),
			Title:            f.Title,
			Message:          f.Description,
			Location:         f.FilePath,
			Remediation:      f.RecommendedRemediation,
			Source:           architecturev1.FindingSource_FINDING_SOURCE_BRANDING,
			Phase:            spec.Phase,
			AutofixAvailable: f.AutofixAvailable,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: scenario,
		Spec:     spec,
		Findings: assessed,
	})
}
