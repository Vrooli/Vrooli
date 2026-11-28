package workflow

import (
	"context"
	"strings"
	"testing"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

func TestWorkflowProjectIDCallersShareInvalidIDContract(t *testing.T) {
	svc := &WorkflowService{}
	ctx := context.Background()
	badID := "  invalid-project-id  "
	_, createErr := svc.CreateWorkflow(ctx, &basapi.CreateWorkflowRequest{ProjectId: badID, Name: "test"})
	_, listErr := svc.ListWorkflows(ctx, &basapi.ListWorkflowsRequest{ProjectId: &badID, FolderPath: "flows"})
	for _, err := range []error{createErr, listErr} {
		if err == nil || !strings.Contains(err.Error(), "invalid project_id:") {
			t.Fatalf("project ID error = %v; want legacy invalid project_id error", err)
		}
	}
	if createErr.Error() != listErr.Error() {
		t.Fatalf("CreateWorkflow error %q != ListWorkflows error %q", createErr, listErr)
	}
}
