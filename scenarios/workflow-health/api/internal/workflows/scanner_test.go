package workflows

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanScenarioCatalogsCasesFlowsActionsSeedsAndRegistryFacts(t *testing.T) {
	scenarioDir := makeScenarioFixture(t)

	catalog, err := ScanScenario(scenarioDir)
	require.NoError(t, err)

	require.Equal(t, "sample-scenario", catalog.Scenario)
	require.True(t, catalog.Registry.Exists)
	require.Equal(t, "observer", catalog.Registry.ExecutionMode)
	require.Len(t, catalog.Cases, 2)
	require.Len(t, catalog.Flows, 1)
	require.Len(t, catalog.Actions, 2)
	require.Len(t, catalog.Seeds, 1)

	caseAsset := findAsset(t, catalog.Assets, "bas/cases/01-foundation/create-project.json")
	require.Equal(t, AssetRoleValidationCase, caseAsset.Role)
	require.Equal(t, "01.01", caseAsset.Order)
	require.Equal(t, "Create project", caseAsset.Name)
	require.Equal(t, []RequirementLink{{ID: "REQ-CREATE", Source: "requirements.validation.ref"}}, caseAsset.Requirements)
	require.Equal(t, []DependencyEdge{
		{
			FromAssetID: "sample-scenario:bas/cases/01-foundation/create-project.json",
			FromPath:    "bas/cases/01-foundation/create-project.json",
			ToPath:      "actions/open-demo-project.json",
			ToAssetID:   "sample-scenario:bas/actions/open-demo-project.json",
			Kind:        "subflow",
			Source:      "setup.action.subflow.workflow_path",
		},
	}, caseAsset.Dependencies)
	require.Equal(t, []SelectorRef{{NodeID: "assert-list", Key: "project.list", Raw: "@selector/project.list", Path: "action.assert.selector"}}, caseAsset.Selectors)
	require.Equal(t, []RouteRef{{NodeID: "nav", Scenario: "@scenario/self", Path: "/projects", Source: "action.navigate.scenario_path"}}, caseAsset.Routes)
	require.False(t, caseAsset.Safety.Mutating)

	mutatingCase := findAsset(t, catalog.Assets, "bas/cases/02-admin/delete-project.json")
	require.Equal(t, []RequirementLink{{ID: "REQ-DELETE", Source: "workflow.metadata"}}, mutatingCase.Requirements)
	require.True(t, mutatingCase.Safety.Mutating)
	require.True(t, mutatingCase.Safety.RequiresIsolation)
	require.True(t, mutatingCase.Safety.RequiresConfirmation)
	require.Equal(t, "full", mutatingCase.Reset)

	flow := findAsset(t, catalog.Assets, "bas/flows/01-user/onboarding.json")
	require.Equal(t, AssetRoleAgentFlow, flow.Role)
	require.Equal(t, "onboard user", flow.Labels["intent"])
	require.Equal(t, []DependencyEdge{
		{
			FromAssetID: "sample-scenario:bas/flows/01-user/onboarding.json",
			FromPath:    "bas/flows/01-user/onboarding.json",
			ToPath:      "actions/load-seed-state.json",
			ToAssetID:   "sample-scenario:bas/actions/load-seed-state.json",
			Kind:        "fixture",
			Source:      "legacy.data.workflowId",
		},
	}, flow.Dependencies)

	action := findAsset(t, catalog.Assets, "bas/actions/open-demo-project.json")
	require.Equal(t, AssetRoleFragment, action.Role)

	seed := findAsset(t, catalog.Assets, "bas/seeds/seed.go")
	require.Equal(t, AssetRoleSeed, seed.Role)

	require.Equal(t, []string{"bas/cases/stale/missing.json"}, catalog.RegistryOnlyPaths)
	registryOnly := findAsset(t, catalog.Assets, "bas/cases/stale/missing.json")
	require.Equal(t, AssetRoleRegistryOnly, registryOnly.Role)

	require.Len(t, catalog.DependencyEdges, 2)
}

func TestScanScenarioToleratesMissingRegistryAndInvalidWorkflowJSON(t *testing.T) {
	scenarioDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(scenarioDir, "bas", "cases"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "bas", "cases", "broken.json"), []byte("{"), 0o644))

	catalog, err := ScanScenario(scenarioDir)
	require.NoError(t, err)

	require.False(t, catalog.Registry.Exists)
	require.Len(t, catalog.Cases, 1)
	require.NotEmpty(t, catalog.Cases[0].ParseError)
	require.Equal(t, AssetRoleValidationCase, catalog.Cases[0].Role)
}

func makeScenarioFixture(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "sample-scenario")
	writeJSON(t, filepath.Join(root, "bas", "registry.json"), map[string]any{
		"scenario":     "sample-scenario",
		"generated_at": "2026-07-02T12:00:00Z",
		"playbooks": []map[string]any{
			{
				"file":         "bas/cases/01-foundation/create-project.json",
				"description":  "registry description",
				"order":        "01.01",
				"requirements": []string{"REQ-OLD"},
				"fixtures":     []string{"open-demo-project"},
				"reset":        "none",
			},
			{
				"file":         "bas/cases/stale/missing.json",
				"description":  "stale",
				"order":        "99.01",
				"requirements": []string{"REQ-STALE"},
				"fixtures":     []string{},
				"reset":        "none",
			},
		},
		"metadata": map[string]any{"execution_mode": "observer"},
	})
	writeJSON(t, filepath.Join(root, "requirements", "index.json"), map[string]any{
		"imports": []string{"module.json"},
	})
	writeJSON(t, filepath.Join(root, "requirements", "module.json"), map[string]any{
		"requirements": []map[string]any{
			{
				"id": "REQ-CREATE",
				"validation": []map[string]any{
					{"ref": "bas/cases/01-foundation/create-project.json"},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "cases", "01-foundation", "create-project.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Create project",
			"description":    "Creates a project through the UI.",
			"version":        "1",
			"execution_mode": "observer",
			"labels":         map[string]any{"reset": "none"},
		},
		"nodes": []map[string]any{
			{
				"id": "nav",
				"action": map[string]any{
					"navigate": map[string]any{
						"scenario":      "@scenario/self",
						"scenario_path": "/projects",
					},
				},
			},
			{
				"id": "setup",
				"action": map[string]any{
					"subflow": map[string]any{"workflow_path": "actions/open-demo-project.json"},
				},
			},
			{
				"id": "assert-list",
				"action": map[string]any{
					"assert": map[string]any{"selector": "@selector/project.list"},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "cases", "02-admin", "delete-project.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Delete project",
			"execution_mode": "mutating",
			"labels": map[string]any{
				"requirements_json":     "[\"REQ-DELETE\"]",
				"reset":                 "database",
				"requires_confirmation": "true",
			},
		},
		"nodes": []map[string]any{},
	})
	writeJSON(t, filepath.Join(root, "bas", "flows", "01-user", "onboarding.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Onboarding",
			"execution_mode": "observer",
			"labels":         map[string]any{"intent": "onboard user", "reset": "none"},
		},
		"nodes": []map[string]any{
			{
				"id": "legacy",
				"data": map[string]any{
					"workflowId": "@fixture/load-seed-state(project=demo)",
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "actions", "open-demo-project.json"), map[string]any{
		"metadata": map[string]any{"name": "Open demo project", "execution_mode": "observer"},
		"nodes":    []map[string]any{},
	})
	writeJSON(t, filepath.Join(root, "bas", "actions", "load-seed-state.json"), map[string]any{
		"metadata": map[string]any{"name": "Load seed state", "execution_mode": "observer"},
		"nodes":    []map[string]any{},
	})
	require.NoError(t, os.MkdirAll(filepath.Join(root, "bas", "seeds"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "bas", "seeds", "seed.go"), []byte("package main\n"), 0o644))

	return root
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	data, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func findAsset(t *testing.T, assets []WorkflowAsset, path string) WorkflowAsset {
	t.Helper()

	for _, asset := range assets {
		if asset.Path == path {
			return asset
		}
	}
	require.Failf(t, "asset not found", "path %s", path)
	return WorkflowAsset{}
}
