package workflows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
