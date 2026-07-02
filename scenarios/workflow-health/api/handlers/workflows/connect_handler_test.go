package workflows

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	internalsearch "workflow-health/internal/search"
	internalvalidation "workflow-health/internal/validation"

	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows"
)

func TestSearchWorkflowsReturnsTypedLeaves(t *testing.T) {
	target := filepath.Join(t.TempDir(), "searchable-workflows")
	writeWorkflowJSON(t, filepath.Join(target, "bas", "flows", "create-project.json"), `{
  "metadata": {
    "name": "Create project",
    "description": "Create a project through the UI.",
    "execution_mode": "observer",
    "labels": { "intent": "create project", "reset": "none" }
  },
  "nodes": []
}`)
	writeWorkflowJSON(t, filepath.Join(target, "bas", "cases", "prove-project.json"), `{
  "metadata": {
    "name": "Prove project creation",
    "description": "Validates project creation.",
    "execution_mode": "observer",
    "labels": { "requirements_json": "[\"REQ-PROJECT\"]", "reset": "none" }
  },
  "nodes": []
}`)

	handler := NewConnectHandler(Deps{Engine: internalvalidation.NewEngine()})
	resp, err := handler.SearchWorkflows(context.Background(), connect.NewRequest(&workflowsv1.SearchWorkflowsRequest{
		Path:  target,
		Query: "run create project workflow",
		Types: []string{internalsearch.LeafTypeFlow},
	}))
	require.NoError(t, err)
	require.Equal(t, "searchable-workflows", resp.Msg.GetScenario())
	require.Len(t, resp.Msg.GetResults(), 1)
	require.Equal(t, internalsearch.LeafTypeFlow, resp.Msg.GetResults()[0].GetLeafType())
	require.True(t, resp.Msg.GetResults()[0].GetRunnable())
}

func writeWorkflowJSON(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body+"\n"), 0o644))
}
