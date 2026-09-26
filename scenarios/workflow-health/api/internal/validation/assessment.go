package validation

import (
	"path/filepath"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func LoadSpec(scenarioRoot string) (*assessment.Spec, error) {
	return assessment.LoadSpecFromScenario(scenarioRoot)
}

func BuildMaturityAssessment(scenario string, findings []Finding, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:             f.Code,
			Severity:         severityToAssessment(f.Severity),
			Title:            f.Title,
			Message:          f.Description,
			Location:         filepath.ToSlash(f.FilePath),
			Remediation:      f.Remediation,
			Source:           architecturev1.FindingSource_FINDING_SOURCE_WORKFLOW,
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
