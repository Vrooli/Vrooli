package workflows

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestAgentManagerDispatcherUsesGeneratedConnectProcedures(t *testing.T) {
	mux := http.NewServeMux()
	var started *apipb.StartWorkflowExecutionRequest
	var resultRequest *apipb.GetWorkflowExecutionResultRequest
	var cancelRequest *apipb.WorkflowExecutionOperationRequest
	input, err := structpb.NewValue(map[string]any{"result": map[string]any{"summary": "done", "outcome": "completed"}})
	if err != nil {
		t.Fatal(err)
	}
	execution := &domainpb.WorkflowExecution{Id: "exec-123", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: input}
	mux.Handle(apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure,
		func(_ context.Context, req *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
			started = req.Msg
			return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: execution}), nil
		},
	))
	mux.Handle(apiconnect.AgentManagerServiceWaitWorkflowExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.AgentManagerServiceWaitWorkflowExecutionProcedure,
		func(_ context.Context, req *connect.Request[apipb.WaitWorkflowExecutionRequest]) (*connect.Response[apipb.WaitWorkflowExecutionResponse], error) {
			if req.Msg.ExecutionId != execution.Id {
				t.Errorf("wait execution id = %q", req.Msg.ExecutionId)
			}
			return connect.NewResponse(&apipb.WaitWorkflowExecutionResponse{Execution: execution}), nil
		},
	))
	mux.Handle(apiconnect.AgentManagerServiceGetWorkflowExecutionResultProcedure, connect.NewUnaryHandler(
		apiconnect.AgentManagerServiceGetWorkflowExecutionResultProcedure,
		func(_ context.Context, req *connect.Request[apipb.GetWorkflowExecutionResultRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
			resultRequest = req.Msg
			return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: execution}), nil
		},
	))
	mux.Handle(apiconnect.AgentManagerServiceCancelWorkflowExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.AgentManagerServiceCancelWorkflowExecutionProcedure,
		func(_ context.Context, req *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
			cancelRequest = req.Msg
			// A proto message carries a mutex; a value copy trips vet's copylocks
			// check and shares internal state. Clone is the supported copy.
			cancelled := proto.Clone(execution).(*domainpb.WorkflowExecution)
			cancelled.Status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED
			return connect.NewResponse(&apipb.WorkflowExecutionOperationResponse{Execution: cancelled}), nil
		},
	))
	server := httptest.NewServer(mux)
	defer server.Close()
	dispatcher := &AgentManagerDispatcher{
		Resolver: discovery.NewResolver(discovery.ResolverConfig{StaticBaseURL: server.URL}),
	}

	ctx := context.Background()
	startedResult, err := dispatcher.Start(ctx, StartInput{Kind: KindExtract, AssetID: "asset-1", SourceScenario: "source", SourcePath: "Card.tsx", IdempotencyKey: "start-key"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started == nil || started.Owner != "react-component-library" || started.WorkflowKey != ExtractWorkflowKey || started.IdempotencyKey != "start-key" {
		t.Fatalf("Start request = %#v", started)
	}
	if startedResult.ExecutionID != execution.Id || startedResult.Status != StatusSucceeded {
		t.Fatalf("Start result = %#v", startedResult)
	}

	waited, err := dispatcher.Wait(ctx, execution.Id)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if waited.Summary != "done" || waited.Status != StatusSucceeded {
		t.Fatalf("Wait result = %#v", waited)
	}
	if resultRequest == nil || resultRequest.ExecutionId != execution.Id || !resultRequest.ExplicitlyAuthorized {
		t.Fatalf("result request = %#v", resultRequest)
	}

	refreshed, err := dispatcher.Refresh(ctx, execution.Id)
	if err != nil || refreshed.Summary != "done" {
		t.Fatalf("Refresh = %#v, %v", refreshed, err)
	}
	stopped, err := dispatcher.Stop(ctx, execution.Id)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopped.Status != StatusStopped || cancelRequest == nil || cancelRequest.IdempotencyKey != "rcl-cancel/"+execution.Id {
		t.Fatalf("Stop result = %#v, request = %#v", stopped, cancelRequest)
	}
}
