package dependencyhealth

import (
	"fmt"
	"path/filepath"

	"scenario-dependency-analyzer/internal/dependencygovernance"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

const approvedDependencyGuidance = dependencygovernance.Guidance

func (h *connectHandler) evaluateGovernance(scenario string, surfaces []*healthv1.DependencyHealthSurface) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, *healthv1.DependencyGovernanceSummary) {
	repoRoot := filepath.Dir(h.resolveScenariosDir())
	registry := dependencygovernance.NewRegistry(repoRoot)
	observed := dependencygovernance.ScanSurfaceDependencies(governanceSurfaces(surfaces))
	result, err := registry.ValidateObserved(scenario, observed)
	if err != nil {
		finding := &healthv1.DependencyHealthFinding{
			Id:           "governance.registry.unreadable",
			Severity:     "ERROR",
			SourceDomain: "governance",
			Title:        "Approved dependency registry unavailable",
			Description:  "SDA could not read or parse the approved dependency registry.",
			Remediation:  "Fix .vrooli/dependencies/approved-dependencies.json, then rerun dependency health.",
			FilePath:     ".vrooli/dependencies/approved-dependencies.json",
			RuleId:       "dependency.governance.registry_readable",
			Observed:     err.Error(),
			Expected:     "readable approved dependency registry",
		}
		summary := &healthv1.DependencyGovernanceSummary{Status: "fail", Guidance: approvedDependencyGuidance}
		return sectionWithFindingIDs("governance", "Approved dependency governance", "fail", "Approved dependency registry could not be evaluated.", []string{finding.GetId()}), []*healthv1.DependencyHealthFinding{finding}, summary
	}

	findings := make([]*healthv1.DependencyHealthFinding, 0, len(result.GetFindings()))
	for _, finding := range result.GetFindings() {
		findings = append(findings, governanceHealthFinding(finding))
	}
	summary := governanceHealthSummary(result.GetSummary())
	status := summary.GetStatus()
	if status == "" {
		status = statusFromFindings(findings, "governance")
	}
	text := fmt.Sprintf("%d observed third-party dependency declaration(s) checked against %d recorded governance decision(s).", result.GetSummary().GetObserved(), governanceRecordCount(result.GetSummary()))
	if status == "not_configured" {
		text = "Approved dependency registry is present but has no records yet; observed dependencies are reported as needs-review guidance, not allowlist failures."
	}
	return sectionWithFindingIDs("governance", "Approved dependency governance", status, text, findingIDs(findings, "governance")), findings, summary
}

func governanceSurfaces(surfaces []*healthv1.DependencyHealthSurface) []dependencygovernance.Surface {
	out := make([]dependencygovernance.Surface, 0, len(surfaces))
	for _, surface := range surfaces {
		out = append(out, dependencygovernance.Surface{
			ID:       surface.GetId(),
			Language: surface.GetLanguage(),
			RootPath: surface.GetRootPath(),
		})
	}
	return out
}

func governanceHealthSummary(summary *governancev1.DependencyGovernanceSummary) *healthv1.DependencyGovernanceSummary {
	if summary == nil {
		return &healthv1.DependencyGovernanceSummary{Status: "not_configured", Guidance: approvedDependencyGuidance}
	}
	return &healthv1.DependencyGovernanceSummary{
		Status:                  summary.GetStatus(),
		Approved:                summary.GetApproved(),
		ApprovedWithConstraints: summary.GetApprovedWithConstraints(),
		NeedsReview:             summary.GetNeedsReview() + summary.GetUnrecorded(),
		Blocked:                 summary.GetBlocked(),
		Deprecated:              summary.GetDeprecated(),
		Guidance:                approvedDependencyGuidance,
	}
}

func governanceRecordCount(summary *governancev1.DependencyGovernanceSummary) int32 {
	if summary == nil {
		return 0
	}
	return summary.GetApproved() + summary.GetApprovedWithConstraints() + summary.GetNeedsReview() + summary.GetBlocked() + summary.GetDeprecated()
}

func governanceHealthFinding(finding *governancev1.ApprovedDependencyFinding) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           finding.GetId(),
		Severity:     finding.GetSeverity(),
		SourceDomain: "governance",
		Title:        finding.GetTitle(),
		Description:  finding.GetDescription(),
		Remediation:  finding.GetRemediation(),
		FilePath:     finding.GetFilePath(),
		RuleId:       "dependency.governance.approved_dependency",
		Observed:     finding.GetObserved(),
		Expected:     finding.GetExpected(),
	}
}
