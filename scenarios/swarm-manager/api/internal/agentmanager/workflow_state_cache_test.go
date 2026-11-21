package agentmanager

import (
	"context"
	"testing"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestWorkflowExecutionStateIsCachedAcrossReadersWithinTTL(t *testing.T) {
	fixture := &workflowRPCFixture{traces: map[string]*apipb.GetWorkflowExecutionTraceResponse{"workflow-cached": {Execution: &domainpb.WorkflowExecution{
		Id: "workflow-cached", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING,
	}}}}
	serverClient := workflowTestClient(t, fixture)

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	SetWorkflowStateCacheTTL(3 * time.Second)
	workflowStates.now = func() time.Time { return now }
	t.Cleanup(func() {
		workflowStates.now = time.Now
		SetWorkflowStateCacheTTL(DefaultWorkflowStateCacheTTL)
	})

	client := serverClient
	workflow := NewWorkflowServiceWithClient(client)
	agent := &AgentService{client: client, enabled: true}
	for i := 0; i < 5; i++ {
		if _, err := workflow.GetWorkflowExecutionState(context.Background(), "workflow-cached"); err != nil {
			t.Fatal(err)
		}
		if _, err := agent.GetWorkflowExecutionState(context.Background(), "workflow-cached"); err != nil {
			t.Fatal(err)
		}
	}
	if got := fixture.traceCalls; got != 1 {
		t.Fatalf("trace calls = %d, want 1 shared across both readers inside the TTL", got)
	}
	now = now.Add(3 * time.Second)
	if _, err := workflow.GetWorkflowExecutionState(context.Background(), "workflow-cached"); err != nil {
		t.Fatal(err)
	}
	if got := fixture.traceCalls; got != 2 {
		t.Fatalf("trace calls after TTL = %d, want 2", got)
	}
}
