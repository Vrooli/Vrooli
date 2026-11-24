package workflow

import (
	"context"
	"testing"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

func TestWorkflowIDRoutesShareInvalidIDContract(t *testing.T) {
	svc := &WorkflowService{}
	ctx := context.Background()
	badID := "  malformed-id  "
	_, getErr := svc.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: badID})
	_, updateErr := svc.UpdateWorkflow(ctx, &basapi.UpdateWorkflowRequest{WorkflowId: badID})
	_, deleteErr := svc.DeleteWorkflow(ctx, &basapi.DeleteWorkflowRequest{WorkflowId: badID})
	errs := []error{getErr, updateErr, deleteErr}
	for _, err := range errs {
		if err == nil {
			t.Fatal("invalid workflow ID unexpectedly passed validation")
		}
		if err.Error() != getErr.Error() {
			t.Fatalf("workflow ID errors differ: %q vs %q", getErr, err)
		}
	}
	if len(getErr.Error()) < len("invalid workflow_id:") || getErr.Error()[:len("invalid workflow_id:")] != "invalid workflow_id:" {
		t.Fatalf("error = %q; want the established wrapped invalid workflow_id error", getErr)
	}
}
