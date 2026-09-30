package workflowruntime

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TestChildRequestRendersUntil verifies the completion test is template-rendered
// with the attempt's binding values before it reaches the run. The literal
// placeholder must never be delivered to the agent.
func TestChildRequestRendersUntil(t *testing.T) {
	engine := &Engine{}
	node := &domain.WorkflowNode{
		ID:   "goal",
		Kind: domain.WorkflowNodeRun,
		Run: &domain.WorkflowRunNode{
			PromptTemplate: "do the work",
			Until:          "Stop when plan-manager exec status {{.plan_execution_id}} reports no unfinished phase.",
		},
	}
	attempt := &domain.WorkflowNodeAttempt{
		ID:             uuid.New(),
		NodeID:         "goal",
		Ordinal:        1,
		InputSnapshot:  json.RawMessage(`{"plan_execution_id":"exec-abc-123"}`),
		PromptSnapshot: "do the work",
	}
	req, err := engine.childRequest(node, &domain.WorkflowExecution{}, attempt, []*domain.WorkflowNodeAttempt{attempt}, nil)
	if err != nil {
		t.Fatalf("childRequest: %v", err)
	}
	if strings.Contains(req.Until, "{{") {
		t.Fatalf("delivered until still contains template syntax: %q", req.Until)
	}
	if !strings.Contains(req.Until, "exec-abc-123") {
		t.Fatalf("rendered until missing bound value: %q", req.Until)
	}
}

// TestChildRequestLeavesPlainUntilUnchanged verifies non-template until text is
// passed through verbatim.
func TestChildRequestLeavesPlainUntilUnchanged(t *testing.T) {
	engine := &Engine{}
	node := &domain.WorkflowNode{
		ID:   "goal",
		Kind: domain.WorkflowNodeRun,
		Run: &domain.WorkflowRunNode{
			PromptTemplate: "do the work",
			Until:          "Every phase recorded finished with evidence.",
		},
	}
	attempt := &domain.WorkflowNodeAttempt{
		ID:             uuid.New(),
		NodeID:         "goal",
		Ordinal:        1,
		InputSnapshot:  json.RawMessage(`{}`),
		PromptSnapshot: "do the work",
	}
	req, err := engine.childRequest(node, &domain.WorkflowExecution{}, attempt, []*domain.WorkflowNodeAttempt{attempt}, nil)
	if err != nil {
		t.Fatalf("childRequest: %v", err)
	}
	if req.Until != "Every phase recorded finished with evidence." {
		t.Fatalf("plain until changed: %q", req.Until)
	}
}
