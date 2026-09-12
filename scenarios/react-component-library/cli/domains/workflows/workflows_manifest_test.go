package workflows

import (
	"os"
	"path/filepath"
	"testing"

	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"

	"github.com/vrooli/cli-core/cliapp"
)

func TestWorkflowsManifestCoversWorkflowsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, workflowsv1.File_react_component_library_v1_workflows_workflows_proto, "WorkflowsService")
}
