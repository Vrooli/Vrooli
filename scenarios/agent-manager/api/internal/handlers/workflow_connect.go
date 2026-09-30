package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/proto"
)

// Workflow Connect adapters share orchestration and payload projections with the
// operator HTTP surface. They do not implement execution or wait loops.
func (h *AgentManagerConnectHandler) validateWorkflowMessage(msg proto.Message) error {
	if h.h == nil || h.h.svc.WorkflowService == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("workflow service unavailable"))
	}
	if h.h.validator != nil {
		if err := h.h.validator.Validate(msg); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
		}
	}
	return nil
}

func (h *AgentManagerConnectHandler) StartWorkflowExecution(ctx context.Context, req *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.Input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workflow input is required"))
	}
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	input, err := json.Marshal(req.Msg.Input.AsInterface())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	initiator := domain.WorkflowInitiatorProgrammatic
	if req.Header().Get("X-Vrooli-Workflow-Initiator") == string(domain.WorkflowInitiatorHuman) {
		initiator = domain.WorkflowInitiatorHuman
	}
	token := req.Header().Get(cliutil.HeaderAgentIdentityToken)
	if token != "" {
		initiator = domain.WorkflowInitiatorAgent
	}
	grant := engagementGrantFromProto(req.Msg.EngagementGrant)
	e, err := h.h.svc.StartWorkflowExecution(ctx, orchestration.StartWorkflowExecutionRequest{Owner: req.Msg.Owner, WorkflowKey: req.Msg.WorkflowKey, DefinitionDigest: req.Msg.DefinitionDigest, Input: input, IdempotencyKey: req.Msg.IdempotencyKey, Initiator: initiator, IdentityToken: token, EngagementGrant: grant, ApprovalDigest: req.Msg.ApprovalDigest, GrantDigest: req.Msg.GrantDigest, ExecutionPreferences: executionPreferencesFromProto(req.Msg.ExecutionPreferences)})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: workflowExecutionToProto(e, false)}), nil
}

func (h *AgentManagerConnectHandler) GetWorkflowExecution(ctx context.Context, req *connect.Request[apipb.GetWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	e, err := h.h.svc.GetWorkflowExecution(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: workflowExecutionToProto(e, false)}), nil
}

func (h *AgentManagerConnectHandler) GetWorkflowExecutionResult(ctx context.Context, req *connect.Request[apipb.GetWorkflowExecutionResultRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if req == nil || req.Msg == nil || !req.Msg.ExplicitlyAuthorized {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("explicit result authorization is required"))
	}
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	e, err := h.h.svc.GetWorkflowExecution(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	result := workflowExecutionToProto(e, true)
	if attempts, err := h.h.svc.ListWorkflowExecutionRuns(ctx, id); err == nil {
		ids := make([]string, 0, len(attempts))
		for _, a := range attempts {
			if a.RunID != nil {
				ids = append(ids, a.RunID.String())
			}
		}
		result.Observations = h.h.workflowObservedReceipts(ctx, ids)
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: result}), nil
}

func (h *AgentManagerConnectHandler) WaitWorkflowExecution(ctx context.Context, req *connect.Request[apipb.WaitWorkflowExecutionRequest]) (*connect.Response[apipb.WaitWorkflowExecutionResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.TimeoutSeconds < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timeout must be nonnegative"))
	}
	r, err := h.h.svc.WaitWorkflowExecution(ctx, id, time.Duration(req.Msg.TimeoutSeconds)*time.Second)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.WaitWorkflowExecutionResponse{Execution: workflowExecutionToProto(r.Execution, false), TimedOut: r.TimedOut}), nil
}

func (h *AgentManagerConnectHandler) CancelWorkflowExecution(ctx context.Context, req *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.h.svc.CancelWorkflowExecution(ctx, orchestration.WorkflowExecutionOperationRequest{
		ExecutionID:     id,
		IdempotencyKey:  req.Msg.IdempotencyKey,
		ExpectedVersion: req.Msg.ExpectedVersion,
		Reason:          req.Msg.Reason,
	})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(workflowOperationToProto(result)), nil
}

func workflowConnectError(err error) error {
	code := connect.CodeInternal
	switch mapErrorCodeToStatus(domain.GetErrorCode(err)) {
	case http.StatusBadRequest:
		code = connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		code = connect.CodeUnauthenticated
	case http.StatusForbidden:
		code = connect.CodePermissionDenied
	case http.StatusNotFound:
		code = connect.CodeNotFound
	case http.StatusConflict:
		code = connect.CodeFailedPrecondition
	case http.StatusTooManyRequests:
		code = connect.CodeResourceExhausted
	case http.StatusServiceUnavailable:
		code = connect.CodeUnavailable
	}
	if errors.Is(err, context.Canceled) {
		code = connect.CodeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code = connect.CodeDeadlineExceeded
	}
	return connect.NewError(code, err)
}
