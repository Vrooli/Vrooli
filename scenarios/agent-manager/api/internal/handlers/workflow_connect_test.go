package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"connectrpc.com/connect"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

type workflowConnectStub struct {
	orchestration.WorkflowService
	execution *domain.WorkflowExecution
	start     orchestration.StartWorkflowExecutionRequest
	waits     int
	cancels   []orchestration.WorkflowExecutionOperationRequest
}

func (s *workflowConnectStub) StartWorkflowExecution(_ context.Context, r orchestration.StartWorkflowExecutionRequest) (*domain.WorkflowExecution, error) {
	s.start = r
	return s.execution, nil
}
func (s *workflowConnectStub) GetWorkflowExecution(context.Context, uuid.UUID) (*domain.WorkflowExecution, error) {
	return s.execution, nil
}
func (s *workflowConnectStub) WaitWorkflowExecution(context.Context, uuid.UUID, time.Duration) (*orchestration.WaitWorkflowExecutionResult, error) {
	s.waits++
	return &orchestration.WaitWorkflowExecutionResult{Execution: s.execution, TimedOut: true}, nil
}
func (s *workflowConnectStub) ListWorkflowExecutionRuns(context.Context, uuid.UUID) ([]*domain.WorkflowNodeAttempt, error) {
	return nil, nil
}
func (s *workflowConnectStub) CancelWorkflowExecution(_ context.Context, r orchestration.WorkflowExecutionOperationRequest) (*orchestration.WorkflowExecutionOperationResult, error) {
	s.cancels = append(s.cancels, r)
	return &orchestration.WorkflowExecutionOperationResult{Execution: s.execution, Idempotent: true}, nil
}
func TestWorkflowConnectStartReadResultAndWait(t *testing.T) {
	id := uuid.New()
	s := &workflowConnectStub{execution: &domain.WorkflowExecution{ID: id, Owner: "web-search", WorkflowKey: "web-search/research", Status: domain.WorkflowExecutionRunning, Input: json.RawMessage(`{"query":"q"}`), Output: json.RawMessage(`{"result":{"summary":"evidence"}}`)}}
	h := NewAgentManagerConnectHandler(&Handler{svc: orchestration.HandlerServices{WorkflowService: s}})
	_, handler := apiconnect.NewAgentManagerServiceHandler(h)
	server := httptest.NewServer(handler)
	defer server.Close()
	c := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
	input, _ := structpb.NewValue(map[string]any{"query": "q"})
	req := connect.NewRequest(&apipb.StartWorkflowExecutionRequest{Owner: "web-search", WorkflowKey: "web-search/research", Input: input, IdempotencyKey: "stable"})
	req.Header().Set(cliutil.HeaderAgentIdentityToken, "forwarded-token")
	started, err := c.StartWorkflowExecution(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, id.String(), started.Msg.Execution.Id)
	require.Nil(t, started.Msg.Execution.Input)
	require.Equal(t, "stable", s.start.IdempotencyKey)
	require.Equal(t, domain.WorkflowInitiatorAgent, s.start.Initiator)
	require.Equal(t, "forwarded-token", s.start.IdentityToken)
	got, err := c.GetWorkflowExecution(context.Background(), connect.NewRequest(&apipb.GetWorkflowExecutionRequest{ExecutionId: id.String()}))
	require.NoError(t, err)
	require.Nil(t, got.Msg.Execution.Output)
	_, err = c.GetWorkflowExecutionResult(context.Background(), connect.NewRequest(&apipb.GetWorkflowExecutionResultRequest{ExecutionId: id.String()}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	result, err := c.GetWorkflowExecutionResult(context.Background(), connect.NewRequest(&apipb.GetWorkflowExecutionResultRequest{ExecutionId: id.String(), ExplicitlyAuthorized: true}))
	require.NoError(t, err)
	require.NotNil(t, result.Msg.Execution.Output)
	waited, err := c.WaitWorkflowExecution(context.Background(), connect.NewRequest(&apipb.WaitWorkflowExecutionRequest{ExecutionId: id.String(), TimeoutSeconds: 30}))
	require.NoError(t, err)
	require.True(t, waited.Msg.TimedOut)
	require.Equal(t, 1, s.waits)
	require.Nil(t, waited.Msg.Execution.Output)
	_, err = c.WaitWorkflowExecution(context.Background(), connect.NewRequest(&apipb.WaitWorkflowExecutionRequest{ExecutionId: "bad", TimeoutSeconds: 30}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Equal(t, 1, s.waits)
}

func TestWorkflowConnectCancelDelegatesOperation(t *testing.T) {
	id := uuid.New()
	s := &workflowConnectStub{execution: &domain.WorkflowExecution{ID: id, Owner: "web-search", Status: domain.WorkflowExecutionRunning}}
	h := NewAgentManagerConnectHandler(&Handler{svc: orchestration.HandlerServices{WorkflowService: s}})
	_, handler := apiconnect.NewAgentManagerServiceHandler(h)
	server := httptest.NewServer(handler)
	defer server.Close()
	c := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	response, err := c.CancelWorkflowExecution(context.Background(), connect.NewRequest(&apipb.WorkflowExecutionOperationRequest{
		ExecutionId: id.String(), IdempotencyKey: "cancel-key", ExpectedVersion: 7, Reason: "operator stopped it",
	}))
	require.NoError(t, err)
	require.True(t, response.Msg.Idempotent)
	require.Len(t, s.cancels, 1)
	require.Equal(t, id, s.cancels[0].ExecutionID)
	require.Equal(t, "cancel-key", s.cancels[0].IdempotencyKey)
	require.Equal(t, int64(7), s.cancels[0].ExpectedVersion)
	require.Equal(t, "operator stopped it", s.cancels[0].Reason)
}
