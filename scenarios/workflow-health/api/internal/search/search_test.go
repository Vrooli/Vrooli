package search

import (
	"testing"

	"github.com/stretchr/testify/require"

	"workflow-health/internal/workflows"
)

func TestSearchCatalogPrefersObserverFlowsForRunIntent(t *testing.T) {
	catalog := fixtureCatalog()

	results := SearchCatalog(catalog, Options{Query: "run create project workflow"})

	require.NotEmpty(t, results)
	require.Equal(t, LeafTypeFlow, results[0].LeafType)
	require.Equal(t, "sample:bas/flows/create-project.json", results[0].ID)
	require.True(t, results[0].Runnable)
}

func TestSearchCatalogPrefersRequirementCasesForValidateIntent(t *testing.T) {
	catalog := fixtureCatalog()

	results := SearchCatalog(catalog, Options{Query: "prove REQ-ROUTED database isolation"})

	require.NotEmpty(t, results)
	require.Equal(t, LeafTypeTest, results[0].LeafType)
	require.Equal(t, "sample:bas/cases/routed-database/proves-test-pool-routing.json", results[0].ID)
	require.Contains(t, results[0].RequirementIDs, "REQ-ROUTED")
}

func TestSearchCatalogHidesFragmentsUnlessRequested(t *testing.T) {
	catalog := fixtureCatalog()

	defaultResults := SearchCatalog(catalog, Options{Query: "open project action"})
	for _, result := range defaultResults {
		require.NotEqual(t, LeafTypeFragment, result.LeafType)
	}

	fragmentResults := SearchCatalog(catalog, Options{Query: "open project action", Types: []string{LeafTypeFragment}})
	require.NotEmpty(t, fragmentResults)
	require.Equal(t, LeafTypeFragment, fragmentResults[0].LeafType)
}

func TestSearchCatalogMutatingResultsCarryGuardrails(t *testing.T) {
	catalog := fixtureCatalog()

	results := SearchCatalog(catalog, Options{Query: "delete project workflow", Types: []string{LeafTypeFlow}})

	require.NotEmpty(t, results)
	require.True(t, results[0].Mutating)
	require.False(t, results[0].Runnable)
	require.Contains(t, results[0].Guardrails, "requires explicit mutating confirmation")
	require.Contains(t, results[0].Guardrails, "requires routed isolation proof")
}

func fixtureCatalog() *workflows.ScenarioWorkflowCatalog {
	return &workflows.ScenarioWorkflowCatalog{
		Scenario: "sample",
		Assets: []workflows.WorkflowAsset{
			{
				ID:            "sample:bas/flows/create-project.json",
				Scenario:      "sample",
				Path:          "bas/flows/create-project.json",
				Type:          workflows.AssetTypeFlow,
				Role:          workflows.AssetRoleAgentFlow,
				Name:          "Create project",
				Description:   "Create a project through the UI.",
				ExecutionMode: "observer",
				Labels:        map[string]string{"intent": "create project"},
				Safety:        workflows.SafetyProfile{ExecutionMode: "observer"},
			},
			{
				ID:            "sample:bas/flows/delete-project.json",
				Scenario:      "sample",
				Path:          "bas/flows/delete-project.json",
				Type:          workflows.AssetTypeFlow,
				Role:          workflows.AssetRoleAgentFlow,
				Name:          "Delete project",
				Description:   "Delete a project after confirmation.",
				ExecutionMode: "mutating",
				Labels:        map[string]string{"intent": "delete project"},
				Safety: workflows.SafetyProfile{
					ExecutionMode:        "mutating",
					Mutating:             true,
					RequiresConfirmation: true,
					RequiresIsolation:    true,
				},
			},
			{
				ID:            "sample:bas/cases/routed-database/proves-test-pool-routing.json",
				Scenario:      "sample",
				Path:          "bas/cases/routed-database/proves-test-pool-routing.json",
				Type:          workflows.AssetTypeCase,
				Role:          workflows.AssetRoleValidationCase,
				Name:          "Proves test pool routing",
				Description:   "Validates routed database isolation.",
				ExecutionMode: "observer",
				Requirements:  []workflows.RequirementLink{{ID: "REQ-ROUTED", Source: "requirements.validation.ref"}},
				Safety:        workflows.SafetyProfile{ExecutionMode: "observer"},
			},
			{
				ID:            "sample:bas/actions/open-project.json",
				Scenario:      "sample",
				Path:          "bas/actions/open-project.json",
				Type:          workflows.AssetTypeAction,
				Role:          workflows.AssetRoleFragment,
				Name:          "Open project",
				Description:   "Open a project detail route.",
				ExecutionMode: "observer",
				Safety:        workflows.SafetyProfile{ExecutionMode: "observer"},
			},
		},
	}
}
