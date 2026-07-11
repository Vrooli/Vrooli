package remediation

import (
	"fmt"
	"strings"

	"test-genie/internal/orchestrator"
	sharedruns "test-genie/internal/shared/runs"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// EvidenceFromExecution converts the persisted execution result and its
// historical descriptor snapshot into planner input. It is intentionally an
// adapter: the planner itself has no dependency on filesystem or proto types.
func EvidenceFromExecution(executionID string, result *orchestrator.SuiteExecutionResult, snapshot *sharedruns.DescriptorSnapshot, snapshotErr error) Evidence {
	evidence := Evidence{SourceExecutionID: strings.TrimSpace(executionID)}
	if result == nil {
		evidence.DegradedReasons = append(evidence.DegradedReasons, "execution result is unavailable")
		return evidence
	}
	evidence.SourceRunID = strings.TrimSpace(result.RunID)
	evidence.Scenario = strings.TrimSpace(result.ScenarioName)
	evidence.CompletedAt = result.CompletedAt
	if snapshotErr != nil {
		evidence.DegradedReasons = append(evidence.DegradedReasons, "descriptor snapshot unavailable: "+snapshotErr.Error())
	}
	if snapshot == nil {
		evidence.DegradedReasons = append(evidence.DegradedReasons, "descriptor snapshot is unavailable")
	}
	descriptors := map[string]sharedruns.PhaseDescriptorSnapshot{}
	if snapshot != nil {
		for _, descriptor := range snapshot.Phases {
			descriptors[descriptor.Phase] = descriptor
		}
	}
	for _, phase := range result.Phases {
		descriptor, found := descriptors[phase.Name]
		if !found {
			evidence.DegradedReasons = append(evidence.DegradedReasons, fmt.Sprintf("descriptor for phase %q is unavailable", phase.Name))
		}
		p := Phase{Name: phase.Name, Status: phase.Status, RunnabilityVerdict: phase.RunnabilityVerdict, RunnabilityReason: phase.RunnabilityReason,
			Remediation: phase.Remediation, ResultGating: descriptor.Policy.ResultGating, DisplayName: descriptor.DisplayName, Provider: descriptor.Provider, DocsPath: descriptor.DocsPath}
		if phase.MaturityStanding != nil {
			p.MaturityStanding = phase.MaturityStanding.GetCurrentLevel()
		}
		if summary := phase.FindingsSummary; summary != nil {
			p.FindingsSummary = FindingSummary{Total: int(summary.GetTotal()), Blockers: int(summary.GetBlockers()), Errors: int(summary.GetErrors()), Warnings: int(summary.GetWarnings()), Infos: int(summary.GetInfos())}
		}
		evidence.Phases = append(evidence.Phases, p)
		for _, finding := range phase.Findings {
			evidence.Findings = append(evidence.Findings, findingFromProto(phase.Name, finding))
		}
	}
	return evidence
}

func findingFromProto(phase string, finding *architecturev1.ArchitectureFinding) Finding {
	if finding == nil {
		return Finding{Phase: phase}
	}
	return Finding{StableID: finding.GetStableId(), Code: finding.GetCode(), Source: finding.GetSource().String(), Severity: severityName(finding.GetSeverity()),
		Class: className(finding.GetFindingClass()), Locations: append([]string(nil), finding.GetLocations()...), Domains: append([]string(nil), finding.GetDomains()...),
		Message: finding.GetMessage(), Suggestion: finding.GetSuggestion(), Effort: effortName(finding.GetEffort()), Phase: phase}
}

func severityName(value architecturev1.FindingSeverity) string {
	return strings.TrimPrefix(strings.ToLower(value.String()), "finding_severity_")
}
func className(value architecturev1.FindingClass) string {
	return strings.TrimPrefix(strings.ToLower(value.String()), "finding_class_")
}
func effortName(value architecturev1.EffortHint) string {
	return strings.TrimPrefix(strings.ToLower(value.String()), "effort_hint_")
}
