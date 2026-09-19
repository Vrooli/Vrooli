package workflows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows"
)

func TestWorkflowsManifestCoversWorkflowSearchService(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	cliapp.RequireProtoServiceCoverage(t, manifest, workflowsv1.File_workflow_health_v1_workflows_workflows_proto, "WorkflowSearchService")
}
