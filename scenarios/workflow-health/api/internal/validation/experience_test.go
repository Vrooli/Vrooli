package validation

import (
	"testing"

	"workflow-health/internal/workflows"

	"github.com/stretchr/testify/require"
)

func TestExperienceCoverageFindingsProgressivelyEnforcesAdoptedRoutes(t *testing.T) {
	region := readinessRegion{ID: "result-list", Required: true}
	region.Binding.TestID = "result-list"
	profile := &readinessProfile{Pages: []readinessPage{{ID: "results", Routes: []string{"/results"}, Regions: []readinessRegion{region}}}}

	uncovered := &workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{{ID: "case", Path: "bas/cases/results.json", Routes: []workflows.RouteRef{{Path: "/results"}}}}}
	findings := experienceCoverageFindings(uncovered, profile)
	require.Len(t, findings, 1)
	require.Equal(t, CodeExperienceBindingMissing, findings[0].Code)

	covered := &workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{{ID: "case", Path: "bas/cases/results.json", Routes: []workflows.RouteRef{{Path: "/results"}}, Selectors: []workflows.SelectorRef{{Raw: "[data-testid=result-list]"}}}}}
	require.Empty(t, experienceCoverageFindings(covered, profile))

	undeclared := &workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{{ID: "case", Path: "bas/cases/other.json", Routes: []workflows.RouteRef{{Path: "/other"}}}}}
	findings = experienceCoverageFindings(undeclared, profile)
	require.Len(t, findings, 1)
	require.Equal(t, CodeExperienceRouteMissing, findings[0].Code)
}

func TestExperienceCoverageFindingsMatchesParameterizedRoutesWithoutQuery(t *testing.T) {
	profile := &readinessProfile{Pages: []readinessPage{
		{ID: "asset", Routes: []string{"/assets/:id"}},
		{ID: "harness", Routes: []string{"/preview/:component/harness.html"}},
	}}
	catalog := &workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{
		{ID: "asset", Path: "bas/cases/asset.json", Routes: []workflows.RouteRef{{Path: "/assets/component-42?tab=preview"}}},
		{ID: "harness", Path: "bas/cases/harness.json", Routes: []workflows.RouteRef{{Path: "/preview/react-component-library:Button/harness.html?example=primary"}}},
	}}

	require.Empty(t, experienceCoverageFindings(catalog, profile))
}

func TestExperienceCoverageDoesNotEnforceOtherScenarioRoutes(t *testing.T) {
	profile := &readinessProfile{Pages: []readinessPage{{Routes: []string{"/results"}}}}
	catalog := &workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{{Routes: []workflows.RouteRef{{Scenario: "other", Path: "/missing"}}}}}
	require.Empty(t, experienceCoverageFindings(catalog, profile))
}

func TestExperienceCoverageRequiresAsyncLoadingAndTerminalStateEvidence(t *testing.T) {
	region := readinessRegion{ID: "results", Required: true}
	region.Binding.TestID = "results"
	region.Lifecycle.Kind = "async"
	region.Lifecycle.States = []string{"loading", "ready", "error"}
	profile := &readinessProfile{Pages: []readinessPage{{Routes: []string{"/results"}, Regions: []readinessRegion{region}}}}
	asset := workflows.WorkflowAsset{ID: "case", Path: "bas/cases/results.json", Routes: []workflows.RouteRef{{Path: "/results"}}, Selectors: []workflows.SelectorRef{{Raw: "[data-testid=results]"}, {Raw: "[data-experience-state=ready]"}}}
	findings := experienceCoverageFindings(&workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{asset}}, profile)
	require.Len(t, findings, 1)
	require.Equal(t, CodeExperienceStateMissing, findings[0].Code)
	asset.Selectors = append(asset.Selectors, workflows.SelectorRef{Raw: "[data-experience-state=loading]"})
	require.Empty(t, experienceCoverageFindings(&workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: []workflows.WorkflowAsset{asset}}, profile))
}

func TestExperienceCoverageAggregatesAsyncLifecycleAcrossRouteWorkflows(t *testing.T) {
	region := readinessRegion{ID: "results", Required: true}
	region.Binding.TestID = "results"
	region.Lifecycle.Kind = "async"
	region.Lifecycle.States = []string{"loading", "ready", "error"}
	profile := &readinessProfile{Pages: []readinessPage{{Routes: []string{"/results"}, Regions: []readinessRegion{region}}}}
	assets := []workflows.WorkflowAsset{
		{ID: "loading", Path: "bas/cases/results-loading.json", Routes: []workflows.RouteRef{{Path: "/results"}}, Selectors: []workflows.SelectorRef{{Raw: "[data-testid=results]"}, {Raw: "[data-experience-state=loading]"}}},
		{ID: "ready", Path: "bas/cases/results-ready.json", Routes: []workflows.RouteRef{{Path: "/results"}}, Selectors: []workflows.SelectorRef{{Raw: "[data-testid=results]"}, {Raw: "[data-experience-state=ready]"}}},
	}
	require.Empty(t, experienceCoverageFindings(&workflows.ScenarioWorkflowCatalog{Scenario: "demo", Assets: assets}, profile))
}
