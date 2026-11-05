package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

// Workflow Connect adapters share orchestration and payload projections with the
// operator HTTP surface. They do not implement execution or wait loops.
func (h *AgentManagerConnectHandler) validateWorkflowMessage(msg proto.Message) error {
	if h.h == nil || h.h.svc.WorkflowService == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("workflow service unavailable"))
	}
	return h.validateMessage(msg)
}

func (h *AgentManagerConnectHandler) ValidateWorkflow(ctx context.Context, req *connect.Request[apipb.ValidateWorkflowRequest]) (*connect.Response[apipb.ValidateWorkflowResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	if req.Msg.Definition == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workflow definition is required"))
	}
	data, err := json.Marshal(req.Msg.Definition.AsMap())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workflow definition"))
	}
	result, err := h.h.svc.ValidateWorkflow(ctx, data)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	response := &apipb.ValidateWorkflowResponse{Valid: result.Valid, Digest: result.Digest, Diagnostics: workflowDiagnosticsToProto(result.Diagnostics)}
	if result.Definition != nil {
		response.Definition = workflowDefinitionToProto(*result.Definition)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ReconcileScenarioWorkflows(ctx context.Context, req *connect.Request[apipb.ReconcileScenarioWorkflowsRequest]) (*connect.Response[apipb.ReconcileScenarioWorkflowsResponse], error) {
	return h.reconcileWorkflows(ctx, req.Msg, false)
}

// ReloadScenarioWorkflows is the operator verb for a non-dry-run reconcile.
func (h *AgentManagerConnectHandler) ReloadScenarioWorkflows(ctx context.Context, req *connect.Request[apipb.ReconcileScenarioWorkflowsRequest]) (*connect.Response[apipb.ReconcileScenarioWorkflowsResponse], error) {
	return h.reconcileWorkflows(ctx, req.Msg, false)
}

func (h *AgentManagerConnectHandler) PlanScenarioWorkflows(ctx context.Context, req *connect.Request[apipb.ReconcileScenarioWorkflowsRequest]) (*connect.Response[apipb.ReconcileScenarioWorkflowsResponse], error) {
	return h.reconcileWorkflows(ctx, req.Msg, true)
}

func (h *AgentManagerConnectHandler) reconcileWorkflows(ctx context.Context, msg *apipb.ReconcileScenarioWorkflowsRequest, forceDryRun bool) (*connect.Response[apipb.ReconcileScenarioWorkflowsResponse], error) {
	if err := h.validateWorkflowMessage(msg); err != nil {
		return nil, err
	}
	result, err := h.h.svc.ReconcileScenarioWorkflows(ctx, orchestration.ReconcileScenarioWorkflowsRequest{Scenario: msg.Scenario, DryRun: msg.DryRun || forceDryRun, ValidateOnly: msg.ValidateOnly})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(workflowReconcileToProto(result)), nil
}

func (h *AgentManagerConnectHandler) ListWorkflowRevisions(ctx context.Context, req *connect.Request[apipb.ListWorkflowRevisionsRequest]) (*connect.Response[apipb.ListWorkflowRevisionsResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	revisions, err := h.h.svc.ListWorkflowRevisions(ctx, req.Msg.Owner, req.Msg.Key, orchestration.ListOptions{Limit: int(req.Msg.Limit), Offset: int(req.Msg.Offset)})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	items := make([]*domainpb.WorkflowRevision, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, workflowRevisionToProto(revision))
	}
	return connect.NewResponse(&apipb.ListWorkflowRevisionsResponse{Revisions: items}), nil
}

func (h *AgentManagerConnectHandler) GetWorkflowRevision(ctx context.Context, req *connect.Request[apipb.GetWorkflowRevisionRequest]) (*connect.Response[apipb.GetWorkflowRevisionResponse], error) {
	return h.workflowRevision(ctx, req.Msg)
}

// ExplainWorkflow returns the same revision projection as GetWorkflowRevision.
func (h *AgentManagerConnectHandler) ExplainWorkflow(ctx context.Context, req *connect.Request[apipb.GetWorkflowRevisionRequest]) (*connect.Response[apipb.GetWorkflowRevisionResponse], error) {
	return h.workflowRevision(ctx, req.Msg)
}

func (h *AgentManagerConnectHandler) workflowRevision(ctx context.Context, msg *apipb.GetWorkflowRevisionRequest) (*connect.Response[apipb.GetWorkflowRevisionResponse], error) {
	if err := h.validateWorkflowMessage(msg); err != nil {
		return nil, err
	}
	revision, err := h.h.svc.GetWorkflowRevision(ctx, msg.Owner, msg.Key, msg.Digest)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	if revision == nil {
		return nil, workflowConnectError(domain.NewNotFoundErrorWithID("WorkflowRevision", msg.Owner+"/"+msg.Key))
	}
	return connect.NewResponse(&apipb.GetWorkflowRevisionResponse{Revision: workflowRevisionToProto(revision)}), nil
}

func (h *AgentManagerConnectHandler) SimulateWorkflow(ctx context.Context, req *connect.Request[apipb.SimulateWorkflowRequest]) (*connect.Response[apipb.SimulateWorkflowResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	input, err := json.Marshal(req.Msg.Input.AsInterface())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workflow input"))
	}
	result, err := h.h.svc.SimulateWorkflow(ctx, orchestration.SimulateWorkflowRequest{Owner: req.Msg.Owner, WorkflowKey: req.Msg.WorkflowKey, DefinitionDigest: req.Msg.DefinitionDigest, Input: input})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	nodes := make([]*apipb.WorkflowNodePlan, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		nodes = append(nodes, &apipb.WorkflowNodePlan{NodeId: node.NodeID, Kind: string(node.Kind), ExecutionStrategy: node.ExecutionStrategy, ProfileKey: node.ProfileKey, RoleRef: node.RoleRef, ContinuationSource: node.ContinuationSource, ChildWorkflowKey: node.ChildWorkflowKey, ChildWorkflowVersion: node.ChildWorkflowVersion, WaitSignal: node.WaitSignal, WaitTimeoutSeconds: int32(node.WaitTimeoutSeconds), JoinStrategy: node.JoinStrategy, JoinQuorum: int32(node.JoinQuorum), Parallel: node.Parallel})
	}
	return connect.NewResponse(&apipb.SimulateWorkflowResponse{Valid: result.Valid, DefinitionDigest: result.DefinitionDigest, Nodes: nodes, PossibleTerminalNodes: result.PossibleTerminalNodes, Diagnostics: workflowDiagnosticsToProto(result.Diagnostics)}), nil
}

func (h *AgentManagerConnectHandler) ListWorkflowExecutionRuns(ctx context.Context, req *connect.Request[apipb.ListWorkflowExecutionRunsRequest]) (*connect.Response[apipb.ListWorkflowExecutionRunsResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	attempts, err := h.h.svc.ListWorkflowExecutionRuns(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	response := &apipb.ListWorkflowExecutionRunsResponse{Attempts: make([]*domainpb.WorkflowNodeAttempt, 0, len(attempts))}
	for _, attempt := range attempts {
		response.Attempts = append(response.Attempts, workflowAttemptToProto(attempt))
	}
	return connect.NewResponse(response), nil
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
	return h.workflowOperation(ctx, req.Msg, h.h.svc.CancelWorkflowExecution)
}

func (h *AgentManagerConnectHandler) ListWorkflowExecutions(ctx context.Context, req *connect.Request[apipb.ListWorkflowExecutionsRequest]) (*connect.Response[apipb.ListWorkflowExecutionsResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	// The domain status is the lower-cased enum suffix (BUDGET_EXHAUSTED -> budget_exhausted).
	status := strings.ToLower(strings.TrimPrefix(req.Msg.Status.String(), "WORKFLOW_EXECUTION_STATUS_"))
	if req.Msg.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_UNSPECIFIED {
		status = ""
	}
	items, err := h.h.svc.ListWorkflowExecutions(ctx, orchestration.ListWorkflowExecutionsRequest{
		Owner:       req.Msg.Owner,
		WorkflowKey: req.Msg.WorkflowKey,
		Status:      domain.WorkflowExecutionStatus(status),
		Limit:       int(req.Msg.Limit),
		Offset:      int(req.Msg.Offset),
	})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	response := &apipb.ListWorkflowExecutionsResponse{Executions: make([]*domainpb.WorkflowExecution, 0, len(items))}
	for _, item := range items {
		response.Executions = append(response.Executions, workflowExecutionToProto(item, false))
	}
	return connect.NewResponse(response), nil
}

// AdvanceWorkflowExecution runs one scheduler step and returns the execution.
func (h *AgentManagerConnectHandler) AdvanceWorkflowExecution(ctx context.Context, req *connect.Request[apipb.GetWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	e, err := h.h.svc.AdvanceWorkflowExecution(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: workflowExecutionToProto(e, false)}), nil
}

func (h *AgentManagerConnectHandler) GetWorkflowExecutionTrace(ctx context.Context, req *connect.Request[apipb.GetWorkflowExecutionTraceRequest]) (*connect.Response[apipb.GetWorkflowExecutionTraceResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	trace, err := h.h.svc.GetWorkflowExecutionTrace(ctx, id, req.Msg.AfterSequence, int(req.Msg.Limit))
	if err != nil {
		return nil, workflowConnectError(err)
	}
	response := &apipb.GetWorkflowExecutionTraceResponse{Execution: workflowExecutionToProto(trace.Execution, false)}
	for _, attempt := range trace.Attempts {
		response.Attempts = append(response.Attempts, workflowAttemptToProto(attempt))
	}
	for _, entry := range trace.Journal {
		response.Journal = append(response.Journal, workflowJournalToProto(entry))
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) SignalWorkflowExecution(ctx context.Context, req *connect.Request[apipb.SignalWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	if err := h.validateWorkflowMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	payload, err := json.Marshal(req.Msg.Payload.AsInterface())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid signal payload"))
	}
	result, err := h.h.svc.SignalWorkflowExecution(ctx, orchestration.WorkflowExecutionSignalRequest{ExecutionID: id, Signal: req.Msg.Signal, Payload: payload, IdempotencyKey: req.Msg.IdempotencyKey, ExpectedVersion: req.Msg.ExpectedVersion})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(workflowOperationToProto(result)), nil
}

func (h *AgentManagerConnectHandler) RetryWorkflowExecution(ctx context.Context, req *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	return h.workflowOperation(ctx, req.Msg, h.h.svc.RetryWorkflowExecution)
}

func (h *AgentManagerConnectHandler) ResumeWorkflowExecution(ctx context.Context, req *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	return h.workflowOperation(ctx, req.Msg, h.h.svc.ResumeWorkflowExecution)
}

func (h *AgentManagerConnectHandler) workflowOperation(ctx context.Context, msg *apipb.WorkflowExecutionOperationRequest, operate func(context.Context, orchestration.WorkflowExecutionOperationRequest) (*orchestration.WorkflowExecutionOperationResult, error)) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	if err := h.validateWorkflowMessage(msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(msg.ExecutionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := operate(ctx, orchestration.WorkflowExecutionOperationRequest{ExecutionID: id, IdempotencyKey: msg.IdempotencyKey, ExpectedVersion: msg.ExpectedVersion, Reason: msg.Reason})
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
