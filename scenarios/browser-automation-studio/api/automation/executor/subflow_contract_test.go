package executor

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestParseSubflowSpecRequiresTypedReference(t *testing.T) {
	_, err := parseSubflowSpec(contracts.PlanStep{
		NodeID: "subflow-node",
		Action: &basactions.ActionDefinition{
			Type:   basactions.ActionType_ACTION_TYPE_SUBFLOW,
			Params: &basactions.ActionDefinition_Subflow{Subflow: &basactions.SubflowParams{}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "workflow_id or workflow_path") {
		t.Fatalf("expected missing reference error, got %v", err)
	}
}

func TestParseSubflowSpecUsesTypedWorkflowReference(t *testing.T) {
	wfID := uuid.New()
	spec, err := parseSubflowSpec(contracts.PlanStep{
		NodeID: "subflow-node",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
			Params: &basactions.ActionDefinition_Subflow{Subflow: &basactions.SubflowParams{
				Target: &basactions.SubflowParams_WorkflowId{WorkflowId: wfID.String()},
			}},
		},
	})
	if err != nil {
		t.Fatalf("parse typed subflow: %v", err)
	}
	if spec.workflowID == nil || *spec.workflowID != wfID {
		t.Fatalf("workflow ID = %v, want %s", spec.workflowID, wfID)
	}
}
