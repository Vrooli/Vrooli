package workflows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeBASDefinitionAcceptsBrowserAutomationStudioCatalog(t *testing.T) {
	root := "../../../../browser-automation-studio/bas"
	for _, family := range []string{"actions", "cases", "flows"} {
		files, err := collectJSONFiles(filepath.Join(root, family))
		require.NoError(t, err)
		for _, path := range files {
			t.Run(path, func(t *testing.T) {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				_, err = DecodeBASDefinitionJSON(data)
				require.NoError(t, err)
			})
		}
	}
}

func TestDecodeBASDefinitionAcceptsCanonicalWorkflow(t *testing.T) {
	definition, err := DecodeBASDefinition(map[string]any{
		"metadata": map[string]any{
			"name":           "smoke",
			"description":    "Smoke workflow.",
			"execution_mode": "observer",
		},
		"nodes": []any{},
		"edges": []any{},
	})
	require.NoError(t, err)
	require.Equal(t, "smoke", definition.GetMetadata().GetName())
}

func TestDecodeBASDefinitionAcceptsTypedSubflowFixtureArguments(t *testing.T) {
	// This contract is argument type preservation, independent of the node count
	// of a consumer workflow. The real BAS catalog is decoded separately above.
	definition, err := DecodeBASDefinitionJSON([]byte(`{"nodes":[{"id":"fixture-seed-state","action":{"type":"ACTION_TYPE_SUBFLOW","subflow":{"workflow_path":"actions/load-seed-state.json","args":{"projectId":"${@params/projectId}","enabled":true,"attempt":2}}}}]}`))
	require.NoError(t, err)
	require.Len(t, definition.GetNodes(), 1)
	subflow := definition.GetNodes()[0].GetAction().GetSubflow()
	require.Equal(t, "actions/load-seed-state.json", subflow.GetWorkflowPath())
	require.Equal(t, "${@params/projectId}", subflow.GetArgs()["projectId"].GetStringValue())
	require.True(t, subflow.GetArgs()["enabled"].GetBoolValue())
	require.Equal(t, int64(2), subflow.GetArgs()["attempt"].GetIntValue())
}

func TestDecodeBASDefinitionRejectsUnknownMetadataField(t *testing.T) {
	_, err := DecodeBASDefinition(map[string]any{
		"metadata": map[string]any{
			"name":         "invalid",
			"requirements": []string{"REQ-1"},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field \"requirements\"")
}

func TestDecodeBASDefinitionAcceptsPersistedBASEnvelope(t *testing.T) {
	definition, err := DecodeBASDefinition(map[string]any{
		"id":          "workflow-1",
		"project_id":  "project-1",
		"folder_path": "cases",
		"created_at":  "2026-07-23T12:00:00Z",
		"metadata": map[string]any{
			"name": "catalog name", "execution_mode": "observer",
		},
		"flow_definition": map[string]any{
			"metadata": map[string]any{
				"name": "nested workflow", "description": "A valid V2 workflow.", "execution_mode": "observer",
			},
			"nodes": []any{}, "edges": []any{},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "nested workflow", definition.GetMetadata().GetName())
}

func TestDecodeBASDefinitionRejectsUnknownFieldInsidePersistedEnvelope(t *testing.T) {
	_, err := DecodeBASDefinition(map[string]any{
		"created_at": "2026-07-23T12:00:00Z",
		"flow_definition": map[string]any{
			"metadata":            map[string]any{"name": "invalid"},
			"unknown_graph_field": true,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown field \"unknown_graph_field\"")
}

func TestResolveBASDocumentRejectsInvalidEnvelopeBoundary(t *testing.T) {
	_, err := ResolveBASDocument(map[string]any{"flow_definition": "not an object"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "flow_definition")

	_, err = ResolveBASDocument(map[string]any{
		"metadata":        map[string]any{"execution_mode": "observer"},
		"flow_definition": map[string]any{"metadata": map[string]any{"execution_mode": "mutating"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflicts")
}
