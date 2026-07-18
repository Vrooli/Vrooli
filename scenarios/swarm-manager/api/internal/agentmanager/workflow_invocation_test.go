package agentmanager

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestStartWorkflowRunsGuardBeforeAgentManagerTransport(t *testing.T) {
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) {
		return "", errors.New("transport must not be called")
	}, nil))
	service.SetStartGuard(func(_ context.Context, workflowKey string) error {
		if workflowKey != "swarm-manager/backlog-workshop-round" {
			t.Fatalf("key=%q", workflowKey)
		}
		return errors.New("plan-manager is stale")
	})
	input, _ := structpb.NewValue(map[string]any{})
	_, err := service.StartWorkflow(context.Background(), Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/backlog-workshop-round", Input: input})
	if err == nil || !strings.Contains(err.Error(), "plan-manager is stale") {
		t.Fatalf("error=%v", err)
	}
}
