package dependencyhealth

import (
	"fmt"
	"strings"

	"github.com/vrooli/maturity-go/assessment"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func buildMaturityAssessment(resp *healthv1.DependencyHealthResponse, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(resp.GetFindings()))
	for _, finding := range resp.GetFindings() {
		findings = append(findings, assessment.Finding{
			Code:        firstNonEmpty(finding.GetRuleId(), finding.GetId()),
			Severity:    severityToAssessment(finding.GetSeverity()),
			Title:       finding.GetTitle(),
			Message:     finding.GetDescription(),
			Location:    finding.GetFilePath(),
			Remediation: finding.GetRemediation(),
			Source:      architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: resp.GetScenario(),
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToAssessment(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL", "HIGH", "ERROR":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String()
	case "MEDIUM", "WARN", "WARNING":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String()
	case "LOW", "INFO":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String()
	default:
		return severity
	}
}
