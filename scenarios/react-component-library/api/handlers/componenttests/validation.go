package componenttests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
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

const componentValidationWorkers = 4

type assetValidationResult struct {
	libraryID string
	detail    string
	failed    bool
}

// DescribeProvider answers Test Genie readiness from provider-owned facts.
// The catalog suite is deliberately not touched here: readiness must stay
// O(1), otherwise Test Genie falls back to running the entire browser suite
// before it can even schedule component-tests.
func (h *sharedHandler) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{
		Provider:    "react-component-library",
		Phase:       "component-tests",
		SpecVersion: "2.0.0",
		Contract:    "scenario-validation/v1",
		Capabilities: &scenariovalidationv1.ProviderCapabilities{
			SupportsExecution: true,
			DeliveryMode:      "inline",
			SupportsFixes:     false,
		},
	}), nil
}

func (h *sharedHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	assets, err := h.assets.List(ctx, components.SearchQuery{Limit: 500})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	jobs := make(chan components.Component)
	results := make(chan assetValidationResult, len(assets))
	workers := componentValidationWorkers
	if len(assets) < workers {
		workers = len(assets)
	}
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for asset := range jobs {
				if asset.LatestVersion == "" {
					continue
				}
				results <- h.validateAsset(ctx, asset)
			}
		}()
	}
	for _, asset := range assets {
		if asset.LatestVersion != "" {
			jobs <- asset
		}
	}
	close(jobs)
	wait.Wait()
	close(results)

	validated := make([]assetValidationResult, 0, len(results))
	for result := range results {
		validated = append(validated, result)
	}
	sort.Slice(validated, func(i, j int) bool { return validated[i].libraryID < validated[j].libraryID })
	failed := false
	failedAssets := []string{}
	failureDetails := map[string]string{}
	for _, result := range validated {
		if !result.failed {
			continue
		}
		failed = true
		failedAssets = append(failedAssets, result.libraryID)
		if result.detail != "" {
			failureDetails[result.libraryID] = result.detail
		}
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if failed {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   req.Msg.GetScenario(),
		Status:     status,
		Assessment: componentTestsAssessment(req.Msg.GetScenario(), !failed, failedAssets, failureDetails),
		Metrics:    collector.Stop(),
	}), nil
}

func (h *sharedHandler) validateAsset(ctx context.Context, asset components.Component) assetValidationResult {
	result := assetValidationResult{libraryID: asset.LibraryID}
	report, runErr := h.service.Run(ctx, domain.Request{ComponentID: asset.ID, Version: asset.LatestVersion, IncludeClosure: true})
	coverageGaps := h.storyCoverageGaps(ctx, asset.ID, asset.LatestVersion)
	if len(coverageGaps) > 0 {
		result.failed = true
		result.detail = fmt.Sprintf("story coverage gaps: %v", coverageGaps)
		if h.logger != nil {
			h.logger.Printf("component-test story coverage failed for %s@%s: %v", asset.LibraryID, asset.LatestVersion, coverageGaps)
		}
	}
	if runErr == nil && report.Verdict == domain.VerdictPassed {
		return result
	}
	result.failed = true
	if h.logger != nil {
		h.logger.Printf("component-test execution failed for %s@%s: run_error=%v verdict=%s results=%v", asset.LibraryID, asset.LatestVersion, runErr, report.Verdict, report.Results)
	}
	if runErr != nil {
		result.detail = "runner error: " + runErr.Error()
		return result
	}
	for _, storyResult := range report.Results {
		if storyResult.Verdict == domain.VerdictFailed && storyResult.Message != "" {
			result.detail = "failed story: " + storyResult.Message
			break
		}
	}
	return result
}

func (h *sharedHandler) storyCoverageGaps(ctx context.Context, componentID, version string) []components.StoryCoverageGap {
	rows, err := h.assets.ListStories(ctx, components.StoryQuery{ComponentID: componentID, Version: version, Limit: 1})
	if err != nil || len(rows) != 1 {
		return nil
	}
	var contract components.StoryContract
	if err := json.Unmarshal([]byte(rows[0].ContractJSON), &contract); err != nil {
		return nil
	}
	return components.StoryCoverageGaps(&contract)
}

// componentTestsAssessment is the provider contract consumed by Test Genie.
// The presentation must be derived by the shared maturity projection so the
// provider cannot drift from the run-gate contract.
func componentTestsAssessment(scenario string, clean bool, failedAssets []string, detailMaps ...map[string]string) *commonv1.MaturityAssessment {
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
	if !clean {
		currentLevel, nextLevel = "L0", levelID
	}
	blockingCodes := []string(nil)
	if !clean {
		blockingCodes = []string{"COMPONENT_TEST_FAILED"}
	}
	local := &commonv1.LocalMaturityAssessment{CurrentLevel: currentLevel, NextLevel: nextLevel, Levels: levels, Clean: clean, BlockingFindingCodes: blockingCodes}
	currentCapability := capabilityLevels[1]
	if !clean {
		currentCapability = capabilityLevels[0]
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
	assessment := &commonv1.MaturityAssessment{Scenario: scenario, Provider: provider, Phase: phase, Version: "2.0.0", Local: local, Capabilities: []*commonv1.CapabilityMaturityAssessment{capability}}
	details := map[string]string{}
	if len(detailMaps) > 0 && detailMaps[0] != nil {
		details = detailMaps[0]
	}
	if !clean {
		for _, asset := range failedAssets {
			message := fmt.Sprintf("%s has a failed component-test report", asset)
			if detail := details[asset]; detail != "" {
				message += ": " + detail
			}
			assessment.Findings = append(assessment.Findings, &commonv1.AssessmentFinding{Code: "COMPONENT_TEST_FAILED", Severity: "SEVERITY_ERROR", Title: "Component test failed", Message: message, Location: asset, Remediation: "inspect the durable component test report and repair the failing contract or source", Maturity: &commonv1.FindingMaturity{LocalLevel: "L0", GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER, Dimension: "tests", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED, CapabilityId: "component_contracts"}, FixClass: "manual"})
		}
	}
	assessment.Presentation = canonicalComponentTestsPresentation(assessment)
	return assessment
}

// canonicalComponentTestsPresentation mirrors the shared v1 projection for
// this provider's deliberately small, fixed maturity contract. Keeping the
// projection beside the assessment construction makes the response contract
// auditable without adding a second module dependency to the RCL API.
func canonicalComponentTestsPresentation(a *commonv1.MaturityAssessment) *commonv1.PhasePresentation {
	local := a.GetLocal()
	if local == nil {
		return nil
	}
	current := local.GetCurrentLevel()
	currentLabel := ""
	for _, level := range local.GetLevels() {
		if level.GetId() == current {
			currentLabel = firstNonBlank(level.GetStatusLabel(), level.GetName())
			break
		}
	}
	presentation := &commonv1.PhasePresentation{
		ContractVersion:      "v1",
		Provider:             a.GetProvider(),
		Phase:                a.GetPhase(),
		CurrentLevel:         current,
		CurrentLevelLabel:    currentLabel,
		NextLevel:            local.GetNextLevel(),
		Clean:                local.GetClean(),
		UnknownCount:         local.GetUnknownCount(),
		BlockingFindingCodes: sortedStrings(local.GetBlockingFindingCodes()),
		AtMaximum:            local.GetClean() && strings.TrimSpace(local.GetNextLevel()) == "",
	}
	if levels := local.GetLevels(); len(levels) > 0 {
		top := levels[len(levels)-1]
		presentation.CeilingLevel = top.GetId()
		presentation.NorthStar = firstNonBlank(top.GetCapabilitySummary(), top.GetName())
	}
	for _, capability := range a.GetCapabilities() {
		if capability == nil {
			continue
		}
		capabilityLabel := ""
		for _, level := range capability.GetLevels() {
			if level.GetId() == capability.GetCurrentLevel() {
				capabilityLabel = firstNonBlank(level.GetStatusLabel(), level.GetName())
				break
			}
		}
		capabilityPresentation := &commonv1.PhaseCapabilityPresentation{
			Id:                   capability.GetId(),
			Label:                capability.GetLabel(),
			CurrentLevel:         capability.GetCurrentLevel(),
			CurrentLevelLabel:    capabilityLabel,
			NextLevel:            capability.GetNextLevel(),
			CurrentSummary:       capability.GetCurrentSummary(),
			NextUnlock:           capability.GetNextUnlock(),
			Clean:                capability.GetClean(),
			UnknownCount:         capability.GetUnknownCount(),
			BlockingFindingCodes: sortedStrings(capability.GetBlockingFindingCodes()),
			PriorityRank:         capability.GetPriorityRank(),
			PriorityReason:       capability.GetPriorityReason(),
		}
		findingsByCode := make(map[string]*commonv1.PhasePresentationFinding)
		for _, finding := range a.GetFindings() {
			if finding == nil || finding.GetMaturity().GetCapabilityId() != capability.GetId() || strings.TrimSpace(finding.GetCode()) == "" {
				continue
			}
			code := strings.TrimSpace(finding.GetCode())
			if existing := findingsByCode[code]; existing != nil {
				existing.Count++
				if location := strings.TrimSpace(finding.GetLocation()); location != "" {
					existing.Locations = append(existing.Locations, location)
				}
				continue
			}
			fixAffordance := commonv1.FixAffordance_FIX_AFFORDANCE_DETECTION_ONLY
			if finding.GetAutofixAvailable() {
				fixAffordance = commonv1.FixAffordance_FIX_AFFORDANCE_PREVIEW_AVAILABLE
			} else if finding.GetFixClass() == "manual" {
				fixAffordance = commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL
			}
			entry := &commonv1.PhasePresentationFinding{
				Code:          code,
				Severity:      finding.GetSeverity(),
				Title:         finding.GetTitle(),
				Message:       finding.GetMessage(),
				Remediation:   finding.GetRemediation(),
				FixAffordance: fixAffordance,
				Count:         1,
			}
			if finding.GetLocation() != "" {
				entry.Locations = []string{finding.GetLocation()}
			}
			findingsByCode[code] = entry
		}
		findingCodes := make([]string, 0, len(findingsByCode))
		for code := range findingsByCode {
			findingCodes = append(findingCodes, code)
		}
		sort.Strings(findingCodes)
		for _, code := range findingCodes {
			entry := findingsByCode[code]
			entry.Locations = sortedStrings(entry.GetLocations())
			capabilityPresentation.Findings = append(capabilityPresentation.Findings, entry)
		}
		presentation.Capabilities = append(presentation.Capabilities, capabilityPresentation)
	}
	if !presentation.GetAtMaximum() && presentation.GetPhase() != "" {
		presentation.DocumentationTopics = []string{presentation.GetPhase() + " maturity next move"}
		for _, code := range presentation.GetBlockingFindingCodes() {
			if code != "" {
				presentation.DocumentationTopics = append(presentation.DocumentationTopics, presentation.GetPhase()+" "+code+" canonical fix")
			}
			if len(presentation.DocumentationTopics) == 3 {
				break
			}
		}
	}
	return presentation
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
