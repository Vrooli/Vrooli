package componenttests

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
)

// sharedHandler turns the catalog suite into a Test Genie provider phase.
// Test Genie owns the enclosing run's lifecycle; this handler executes no
// arbitrary process or client-owned background work.
type sharedHandler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	service *domain.Service
	assets  components.Service
	logger  *log.Logger
}

func (h *sharedHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	assets, err := h.assets.List(ctx, components.SearchQuery{Limit: 500})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	failed := false
	failedAssets := []string{}
	for _, asset := range assets {
		if asset.LatestVersion == "" {
			continue
		}
		report, runErr := h.service.Run(ctx, domain.Request{ComponentID: asset.ID, Version: asset.LatestVersion, IncludeClosure: true})
		if runErr != nil || report.Verdict != domain.VerdictPassed {
			if runErr != nil || report.Verdict == domain.VerdictFailed {
				failed = true
				failedAssets = append(failedAssets, asset.LibraryID)
			}
		}
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if failed {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   req.Msg.GetScenario(),
		Status:     status,
		Assessment: componentTestsAssessment(req.Msg.GetScenario(), !failed, failedAssets),
	}), nil
}

// componentTestsAssessment is the self-contained provider contract consumed by
// Test Genie. Keep this projection local: the RCL API deliberately does not
// take a dependency on maturity-go merely to construct protobuf data.
func componentTestsAssessment(scenario string, clean bool, failedAssets []string) *commonv1.MaturityAssessment {
	const (
		provider = "react-component-library"
		phase    = "component-tests"
		levelID  = "L1"
		label    = "Contracts validated"
	)
	levels := []*commonv1.LocalMaturityLevel{
		{Id: "L0", Name: "Contracts unavailable", Description: "Catalog contracts cannot be evaluated."},
		{Id: levelID, Name: label, Description: "Versioned component and hook contracts are parsed and checked against declared examples."},
	}
	capabilityLevels := []*commonv1.LocalMaturityLevel{
		{Id: "L0", Name: "Unavailable", StatusLabel: "Unavailable", CapabilitySummary: "No validated component contract evidence.", NextUnlock: "A valid versioned contract."},
		{Id: levelID, Name: "Validated", StatusLabel: "Ready", CapabilitySummary: "Component contract evidence is available.", NextUnlock: "No unresolved contract-test findings."},
	}
	currentLevel := levelID
	nextLevel := ""
	currentLocal := levels[1]
	if !clean {
		currentLevel, nextLevel = "L0", levelID
		currentLocal = levels[0]
	}
	local := &commonv1.LocalMaturityAssessment{CurrentLevel: currentLevel, NextLevel: nextLevel, Levels: levels, Clean: clean}
	currentCapability := capabilityLevels[1]
	if !clean {
		currentCapability = capabilityLevels[0]
	}
	presentation := &commonv1.PhasePresentation{
		ContractVersion:   "v1",
		Provider:          provider,
		Phase:             phase,
		CurrentLevel:      currentLevel,
		CurrentLevelLabel: currentLocal.Name,
		NextLevel:         nextLevel,
		CeilingLevel:      levelID,
		NorthStar:         levels[1].Name,
		Clean:             clean,
		AtMaximum:         clean,
		Capabilities: []*commonv1.PhaseCapabilityPresentation{{
			Id:                "component_contracts",
			Label:             "Component Contracts",
			CurrentLevel:      currentLevel,
			CurrentLevelLabel: currentCapability.StatusLabel,
			NextLevel:         nextLevel,
			CurrentSummary:    currentCapability.CapabilitySummary,
			NextUnlock:        currentCapability.NextUnlock,
			Clean:             clean,
			PriorityRank:      1,
		}},
	}
	if !clean {
		presentation.DocumentationTopics = []string{phase + " maturity next move"}
	}
	capability := &commonv1.CapabilityMaturityAssessment{
		Id:             "component_contracts",
		Label:          "Component Contracts",
		Description:    "Version-pinned declarative component and hook behavior contracts.",
		CurrentLevel:   currentLevel,
		NextLevel:      nextLevel,
		Levels:         capabilityLevels,
		CurrentSummary: currentCapability.CapabilitySummary,
		NextUnlock:     currentCapability.NextUnlock,
		Clean:          clean,
		PriorityRank:   1,
	}
	assessment := &commonv1.MaturityAssessment{Scenario: scenario, Provider: provider, Phase: phase, Version: "2.0.0", Local: local, Capabilities: []*commonv1.CapabilityMaturityAssessment{capability}, Presentation: presentation}
	if !clean {
		for _, asset := range failedAssets {
			assessment.Findings = append(assessment.Findings, &commonv1.AssessmentFinding{Code: "COMPONENT_TEST_FAILED", Severity: "SEVERITY_ERROR", Title: "Component test failed", Message: fmt.Sprintf("%s has a failed component-test report", asset), Location: asset, Remediation: "inspect the durable component test report and repair the failing contract or source", Maturity: &commonv1.FindingMaturity{LocalLevel: "L0", GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER, Dimension: "tests", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED, CapabilityId: "component_contracts"}, FixClass: "manual"})
		}
	}
	return assessment
}
