package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type workflowRPCFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	started, waited *domainpb.WorkflowExecution
	executions      map[string]*domainpb.WorkflowExecution
	listed          []*domainpb.WorkflowExecution
	traces          map[string]*apipb.GetWorkflowExecutionTraceResponse
	runs            map[string]*domainpb.Run
	resultErrors    map[string]error
	traceErrors     map[string]error
	runErrors       map[string]error
	startRequest    *apipb.StartWorkflowExecutionRequest
	listRequest     *apipb.ListWorkflowExecutionsRequest
	signal          *apipb.SignalWorkflowExecutionRequest
	cancel          *apipb.WorkflowExecutionOperationRequest
	traceCalls      int
}

func (f *workflowRPCFixture) StartWorkflowExecution(_ context.Context, r *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	f.startRequest = r.Msg
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: f.started}), nil
}
func (f *workflowRPCFixture) ListWorkflowExecutions(_ context.Context, r *connect.Request[apipb.ListWorkflowExecutionsRequest]) (*connect.Response[apipb.ListWorkflowExecutionsResponse], error) {
	f.listRequest = r.Msg
	return connect.NewResponse(&apipb.ListWorkflowExecutionsResponse{Executions: f.listed}), nil
}
func (f *workflowRPCFixture) GetWorkflowExecutionResult(_ context.Context, r *connect.Request[apipb.GetWorkflowExecutionResultRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if err := f.resultErrors[r.Msg.ExecutionId]; err != nil {
		return nil, err
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: f.executions[r.Msg.ExecutionId]}), nil
}
func (f *workflowRPCFixture) WaitWorkflowExecution(_ context.Context, _ *connect.Request[apipb.WaitWorkflowExecutionRequest]) (*connect.Response[apipb.WaitWorkflowExecutionResponse], error) {
	return connect.NewResponse(&apipb.WaitWorkflowExecutionResponse{Execution: f.waited}), nil
}
func (f *workflowRPCFixture) GetWorkflowExecutionTrace(_ context.Context, r *connect.Request[apipb.GetWorkflowExecutionTraceRequest]) (*connect.Response[apipb.GetWorkflowExecutionTraceResponse], error) {
	f.traceCalls++
	if err := f.traceErrors[r.Msg.ExecutionId]; err != nil {
		return nil, err
	}
	return connect.NewResponse(f.traces[r.Msg.ExecutionId]), nil
}
func (f *workflowRPCFixture) GetRun(_ context.Context, r *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	if err := f.runErrors[r.Msg.RunId]; err != nil {
		return nil, err
	}
	return connect.NewResponse(&apipb.GetRunResponse{Run: f.runs[r.Msg.RunId]}), nil
}
func (f *workflowRPCFixture) SignalWorkflowExecution(_ context.Context, r *connect.Request[apipb.SignalWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	f.signal = r.Msg
	return connect.NewResponse(&apipb.WorkflowExecutionOperationResponse{}), nil
}
func (f *workflowRPCFixture) CancelWorkflowExecution(_ context.Context, r *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	f.cancel = r.Msg
	return connect.NewResponse(&apipb.WorkflowExecutionOperationResponse{}), nil
}
func workflowTestClient(t *testing.T, f *workflowRPCFixture) *HTTPClient {
	t.Helper()
	path, h := apiconnect.NewAgentManagerServiceHandler(f)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewHTTPClientWithResolver(func(context.Context) (string, error) { return srv.URL, nil }, srv.Client())
}

func TestWorkflowStartInspectionUsesExactReadOnlyKeyFilter(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{"consumer": map[string]any{"entityVersion": "v1"}})
	x := &domainpb.WorkflowExecution{Id: "original", Owner: "swarm-manager", WorkflowKey: "swarm-manager/phased-plan-drain", IdempotencyKey: "original-start-key", Input: input}
	f := &workflowRPCFixture{listed: []*domainpb.WorkflowExecution{x}, executions: map[string]*domainpb.WorkflowExecution{"original": x}}
	service := NewWorkflowServiceWithClient(workflowTestClient(t, f))
	got, err := service.InspectWorkflowStart(t.Context(), x.WorkflowKey, x.IdempotencyKey)
	if err != nil || !proto.Equal(got, x) {
		t.Fatalf("lookup=%+v err=%v", got, err)
	}
	if f.listRequest.GetIdempotencyKey() != x.IdempotencyKey || f.listRequest.GetOwner() != "swarm-manager" || f.listRequest.GetLimit() != 1 {
		t.Fatalf("list filter=%+v", f.listRequest)
	}
}

func TestWorkflowServiceProgressReadsLiveTrace(t *testing.T) {
	now := timestamppb.New(time.Now().UTC())
	execution := &domainpb.WorkflowExecution{Id: "workflow-1", CurrentNodeId: "slice_review", UpdatedAt: now, BudgetUsage: &domainpb.WorkflowBudgetUsage{Turns: 4, Tokens: 900, CostUsd: .12}, EdgeTraversals: map[string]int32{"review": 2}}
	f := &workflowRPCFixture{traces: map[string]*apipb.GetWorkflowExecutionTraceResponse{"workflow-1": {Execution: execution, Journal: []*domainpb.WorkflowJournalEntry{{NodeId: "slice"}, {NodeId: "slice"}, {NodeId: "review"}}}}}
	progress, err := NewWorkflowServiceWithClient(workflowTestClient(t, f)).GetWorkflowProgress(t.Context(), "workflow-1")
	if err != nil || progress.CurrentNode != "slice_review" || progress.SliceCount != 2 || progress.Turns != 4 || progress.Tokens != 900 || progress.CostUSD != .12 || progress.EdgeTraversals["review"] != 2 || progress.UpdatedAt == "" {
		t.Fatalf("progress=%#v err=%v", progress, err)
	}
}

func TestWorkflowExecutionStateReadsAuthorizedTerminalEvidence(t *testing.T) {
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "complete"}})
	trace := &domainpb.WorkflowExecution{Id: "workflow-1", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED}
	result := proto.Clone(trace).(*domainpb.WorkflowExecution)
	result.Output = output
	f := &workflowRPCFixture{traces: map[string]*apipb.GetWorkflowExecutionTraceResponse{"workflow-1": {Execution: trace}}, executions: map[string]*domainpb.WorkflowExecution{"workflow-1": result}}
	state, err := NewWorkflowServiceWithClient(workflowTestClient(t, f)).GetWorkflowExecutionState(t.Context(), "workflow-1")
	if err != nil || !state.TerminalEvidence {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}

func TestWorkflowServiceStartCollectAndTypedControls(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{"consumer": map[string]any{"executionId": "consumer-1"}})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "complete"}})
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", Owner: "swarm-manager", WorkflowKey: "swarm-manager/phased-plan-drain", IdempotencyKey: "stable", DefinitionDigest: "sha256:def", ApprovalDigest: "sha256:approval", GrantDigest: "sha256:grant", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output, CreatedAt: timestamppb.New(time.Now().Add(-time.Second)), EndedAt: timestamppb.Now()}
	attempt := &domainpb.WorkflowNodeAttempt{NodeId: "slice", RunId: "22222222-2222-2222-2222-222222222222"}
	f := &workflowRPCFixture{started: execution, waited: execution, executions: map[string]*domainpb.WorkflowExecution{execution.Id: execution}, traces: map[string]*apipb.GetWorkflowExecutionTraceResponse{execution.Id: {Execution: execution, Attempts: []*domainpb.WorkflowNodeAttempt{attempt}}}}
	service := NewWorkflowServiceWithClient(workflowTestClient(t, f))
	start, err := service.StartWorkflow(t.Context(), Invocation{Owner: "swarm-manager", WorkflowKey: execution.WorkflowKey, WorkflowDigest: "sha256:def", ApprovalDigest: "sha256:approval", GrantDigest: "sha256:grant", Input: input, IdempotencyKey: "stable", FirstRunNodeID: "slice", EngagementGrant: &domainpb.WorkflowEngagementGrant{MaxTokens: 200, MaxWallTimeSeconds: 30}, ExecutionPreferences: &domainpb.ExecutionPreferences{PreferredRunner: "claude-code", Model: "claude-sonnet", Effort: "high"}})
	if err != nil || start.ExecutionID != execution.Id || start.RunID != attempt.RunId {
		t.Fatalf("start=%+v err=%v", start, err)
	}
	if f.startRequest.GetEngagementGrant().GetMaxTokens() != 200 || f.startRequest.GetExecutionPreferences().GetModel() != "claude-sonnet" {
		t.Fatalf("start request=%+v", f.startRequest)
	}
	completion, err := service.CollectWorkflow(t.Context(), execution.Id)
	if err != nil || completion.Input == nil || completion.Output == nil || len(completion.Attempts) != 1 {
		t.Fatalf("completion=%#v err=%v", completion, err)
	}
	payload, _ := structpb.NewValue(map[string]any{"executionId": "consumer-1"})
	if err = service.SignalWorkflow(t.Context(), execution.Id, "slice_approved", payload, "approve-1"); err != nil {
		t.Fatal(err)
	}
	if f.signal.GetIdempotencyKey() != "approve-1" || f.signal.GetSignal() != "slice_approved" {
		t.Fatalf("signal=%+v", f.signal)
	}
	if err = service.CancelWorkflow(t.Context(), execution.Id, "cancel-1", "operator canceled"); err != nil {
		t.Fatal(err)
	}
	if f.cancel.GetIdempotencyKey() != "cancel-1" || f.cancel.GetReason() != "operator canceled" {
		t.Fatalf("cancel=%+v", f.cancel)
	}
}

func TestWorkflowServiceCollectsCanceledCompletion(t *testing.T) {
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED}
	f := &workflowRPCFixture{waited: execution, executions: map[string]*domainpb.WorkflowExecution{execution.Id: execution}, traces: map[string]*apipb.GetWorkflowExecutionTraceResponse{execution.Id: {Execution: execution}}}
	completion, err := NewWorkflowServiceWithClient(workflowTestClient(t, f)).CollectWorkflow(t.Context(), execution.Id)
	if err != nil || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED {
		t.Fatalf("completion=%#v err=%v", completion, err)
	}
}

func TestPhasedPlanTerminalStatusesIncludeBlockedAndAbstained(t *testing.T) {
	for _, status := range []domainpb.WorkflowExecutionStatus{domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED, domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED} {
		if !terminalWorkflowStatus(status) {
			t.Fatalf("status %s was not terminal", status)
		}
	}
}

func writeWorkflowProto(t *testing.T, w http.ResponseWriter, message proto.Message, status int) {
	t.Helper()
	data, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
