package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *Handler) ValidateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req apipb.ValidateWorkflowRequest
	if !h.readWorkflowProto(w, r, &req) || req.Definition == nil {
		return
	}
	data, err := json.Marshal(req.Definition.AsMap())
	if err != nil {
		writeSimpleError(w, r, "definition", "invalid workflow definition")
		return
	}
	result, err := h.svc.ValidateWorkflow(r.Context(), data)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := &apipb.ValidateWorkflowResponse{Valid: result.Valid, Digest: result.Digest, Diagnostics: workflowDiagnosticsToProto(result.Diagnostics)}
	if result.Definition != nil {
		response.Definition = workflowDefinitionToProto(*result.Definition)
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) ReconcileScenarioWorkflows(w http.ResponseWriter, r *http.Request) {
	h.reconcileScenarioWorkflows(w, r, false)
}

func (h *Handler) PlanScenarioWorkflows(w http.ResponseWriter, r *http.Request) {
	h.reconcileScenarioWorkflows(w, r, true)
}

func (h *Handler) reconcileScenarioWorkflows(w http.ResponseWriter, r *http.Request, forceDryRun bool) {
	var req apipb.ReconcileScenarioWorkflowsRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	result, err := h.svc.ReconcileScenarioWorkflows(r.Context(), orchestration.ReconcileScenarioWorkflowsRequest{Scenario: req.Scenario, DryRun: req.DryRun || forceDryRun, ValidateOnly: req.ValidateOnly})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, workflowReconcileToProto(result))
}

func (h *Handler) ListWorkflowRevisions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	revisions, err := h.svc.ListWorkflowRevisions(r.Context(), r.URL.Query().Get("owner"), r.URL.Query().Get("key"), orchestration.ListOptions{Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]*domainpb.WorkflowRevision, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, workflowRevisionToProto(revision))
	}
	writeProtoJSON(w, http.StatusOK, &apipb.ListWorkflowRevisionsResponse{Revisions: items})
}

func (h *Handler) GetWorkflowRevision(w http.ResponseWriter, r *http.Request) {
	revision, err := h.svc.GetWorkflowRevision(r.Context(), r.URL.Query().Get("owner"), r.URL.Query().Get("key"), r.URL.Query().Get("digest"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	if revision == nil {
		writeError(w, r, domain.NewNotFoundErrorWithID("WorkflowRevision", r.URL.RawQuery))
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.GetWorkflowRevisionResponse{Revision: workflowRevisionToProto(revision)})
}

func (h *Handler) StartWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	var req apipb.StartWorkflowExecutionRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	input, err := json.Marshal(req.Input.AsInterface())
	if err != nil {
		writeSimpleError(w, r, "input", "invalid workflow input")
		return
	}
	initiator := domain.WorkflowInitiatorProgrammatic
	if r.Header.Get("X-Vrooli-Workflow-Initiator") == string(domain.WorkflowInitiatorHuman) {
		initiator = domain.WorkflowInitiatorHuman
	}
	if r.Header.Get(cliutil.HeaderAgentIdentityToken) != "" {
		initiator = domain.WorkflowInitiatorAgent
	}
	execution, err := h.svc.StartWorkflowExecution(r.Context(), orchestration.StartWorkflowExecutionRequest{Owner: req.Owner, WorkflowKey: req.WorkflowKey, DefinitionDigest: req.DefinitionDigest, Input: input, IdempotencyKey: req.IdempotencyKey, Initiator: initiator, IdentityToken: r.Header.Get(cliutil.HeaderAgentIdentityToken)})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusAccepted, &apipb.WorkflowExecutionResponse{Execution: workflowExecutionToProto(execution, false)})
}

func (h *Handler) ListWorkflowExecutions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.svc.ListWorkflowExecutions(r.Context(), orchestration.ListWorkflowExecutionsRequest{
		Owner:       r.URL.Query().Get("owner"),
		WorkflowKey: r.URL.Query().Get("workflow_key"),
		Status:      domain.WorkflowExecutionStatus(r.URL.Query().Get("status")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := &apipb.ListWorkflowExecutionsResponse{Executions: make([]*domainpb.WorkflowExecution, 0, len(items))}
	for _, item := range items {
		response.Executions = append(response.Executions, workflowExecutionToProto(item, false))
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) GetWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	h.workflowExecutionByID(w, r, false)
}

func (h *Handler) GetWorkflowExecutionResult(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("explicitly_authorized") != "true" {
		writeSimpleError(w, r, "explicitly_authorized", "explicit authorization is required to inspect workflow input and output")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "id", "invalid workflow execution id")
		return
	}
	execution, err := h.svc.GetWorkflowExecution(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	pbExecution := workflowExecutionToProto(execution, true)
	if attempts, listErr := h.svc.ListWorkflowExecutionRuns(r.Context(), id); listErr == nil {
		runIDs := make([]string, 0, len(attempts))
		for _, attempt := range attempts {
			if attempt.RunID != nil {
				runIDs = append(runIDs, attempt.RunID.String())
			}
		}
		pbExecution.Observations = h.workflowObservedReceipts(r.Context(), runIDs)
	}
	writeProtoJSON(w, http.StatusOK, &apipb.WorkflowExecutionResponse{Execution: pbExecution})
}

func (h *Handler) AdvanceWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	h.workflowExecutionByID(w, r, true)
}

func (h *Handler) WaitWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	var req apipb.WaitWorkflowExecutionRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil || (req.ExecutionId != "" && req.ExecutionId != id.String()) {
		writeSimpleError(w, r, "executionId", "path and request execution ids must match")
		return
	}
	result, err := h.svc.WaitWorkflowExecution(r.Context(), id, time.Duration(req.TimeoutSeconds)*time.Second)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.WaitWorkflowExecutionResponse{
		Execution: workflowExecutionToProto(result.Execution, false),
		TimedOut:  result.TimedOut,
	})
}

func (h *Handler) GetWorkflowExecutionTrace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "id", "invalid workflow execution id")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after_sequence"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	trace, err := h.svc.GetWorkflowExecutionTrace(r.Context(), id, after, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := &apipb.GetWorkflowExecutionTraceResponse{Execution: workflowExecutionToProto(trace.Execution, false)}
	for _, attempt := range trace.Attempts {
		response.Attempts = append(response.Attempts, workflowAttemptToProto(attempt))
	}
	for _, entry := range trace.Journal {
		response.Journal = append(response.Journal, workflowJournalToProto(entry))
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) ListWorkflowExecutionRuns(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, r, domain.NewValidationError("id", "must be a UUID"))
		return
	}
	attempts, err := h.svc.ListWorkflowExecutionRuns(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := &apipb.ListWorkflowExecutionRunsResponse{Attempts: make([]*domainpb.WorkflowNodeAttempt, 0, len(attempts))}
	for _, attempt := range attempts {
		response.Attempts = append(response.Attempts, workflowAttemptToProto(attempt))
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) SignalWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	var req apipb.SignalWorkflowExecutionRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil || req.ExecutionId != id.String() {
		writeSimpleError(w, r, "executionId", "path and request execution ids must match")
		return
	}
	payload, err := json.Marshal(req.Payload.AsInterface())
	if err != nil {
		writeSimpleError(w, r, "payload", "invalid signal payload")
		return
	}
	result, err := h.svc.SignalWorkflowExecution(r.Context(), orchestration.WorkflowExecutionSignalRequest{ExecutionID: id, Signal: req.Signal, Payload: payload, IdempotencyKey: req.IdempotencyKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, workflowOperationToProto(result))
}

func (h *Handler) CancelWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	h.workflowOperation(w, r, "cancel")
}

func (h *Handler) RetryWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	h.workflowOperation(w, r, "retry")
}

func (h *Handler) ResumeWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	h.workflowOperation(w, r, "resume")
}

func (h *Handler) workflowOperation(w http.ResponseWriter, r *http.Request, operation string) {
	var req apipb.WorkflowExecutionOperationRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil || req.ExecutionId != id.String() {
		writeSimpleError(w, r, "executionId", "path and request execution ids must match")
		return
	}
	domainReq := orchestration.WorkflowExecutionOperationRequest{ExecutionID: id, IdempotencyKey: req.IdempotencyKey, ExpectedVersion: req.ExpectedVersion, Reason: req.Reason}
	var result *orchestration.WorkflowExecutionOperationResult
	switch operation {
	case "cancel":
		result, err = h.svc.CancelWorkflowExecution(r.Context(), domainReq)
	case "retry":
		result, err = h.svc.RetryWorkflowExecution(r.Context(), domainReq)
	case "resume":
		result, err = h.svc.ResumeWorkflowExecution(r.Context(), domainReq)
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, workflowOperationToProto(result))
}

func workflowOperationToProto(result *orchestration.WorkflowExecutionOperationResult) *apipb.WorkflowExecutionOperationResponse {
	if result == nil {
		return &apipb.WorkflowExecutionOperationResponse{}
	}
	return &apipb.WorkflowExecutionOperationResponse{Execution: workflowExecutionToProto(result.Execution, false), Idempotent: result.Idempotent}
}

func (h *Handler) workflowExecutionByID(w http.ResponseWriter, r *http.Request, advance bool) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "id", "invalid workflow execution id")
		return
	}
	var execution *domain.WorkflowExecution
	if advance {
		execution, err = h.svc.AdvanceWorkflowExecution(r.Context(), id)
	} else {
		execution, err = h.svc.GetWorkflowExecution(r.Context(), id)
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.WorkflowExecutionResponse{Execution: workflowExecutionToProto(execution, false)})
}

func (h *Handler) SimulateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req apipb.SimulateWorkflowRequest
	if !h.readWorkflowProto(w, r, &req) {
		return
	}
	input, err := json.Marshal(req.Input.AsInterface())
	if err != nil {
		writeSimpleError(w, r, "input", "invalid workflow input")
		return
	}
	result, err := h.svc.SimulateWorkflow(r.Context(), orchestration.SimulateWorkflowRequest{Owner: req.Owner, WorkflowKey: req.WorkflowKey, DefinitionDigest: req.DefinitionDigest, Input: input})
	if err != nil {
		writeError(w, r, err)
		return
	}
	nodes := make([]*apipb.WorkflowNodePlan, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		nodes = append(nodes, &apipb.WorkflowNodePlan{NodeId: node.NodeID, Kind: string(node.Kind), ExecutionStrategy: node.ExecutionStrategy, ProfileKey: node.ProfileKey, RoleRef: node.RoleRef, ContinuationSource: node.ContinuationSource, ChildWorkflowKey: node.ChildWorkflowKey, ChildWorkflowVersion: node.ChildWorkflowVersion, WaitSignal: node.WaitSignal, WaitTimeoutSeconds: int32(node.WaitTimeoutSeconds), JoinStrategy: node.JoinStrategy, JoinQuorum: int32(node.JoinQuorum), Parallel: node.Parallel})
	}
	writeProtoJSON(w, http.StatusOK, &apipb.SimulateWorkflowResponse{Valid: result.Valid, DefinitionDigest: result.DefinitionDigest, Nodes: nodes, PossibleTerminalNodes: result.PossibleTerminalNodes, Diagnostics: workflowDiagnosticsToProto(result.Diagnostics)})
}

func (h *Handler) readWorkflowProto(w http.ResponseWriter, r *http.Request, req proto.Message) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 300<<10))
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return false
	}
	if err := protoconv.UnmarshalJSON(body, req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return false
	}
	if !h.validateProto(w, r, req) {
		return false
	}
	return true
}

func workflowDefinitionToProto(definition domain.WorkflowDefinition) *structpb.Struct {
	data, _ := json.Marshal(definition)
	var object map[string]any
	_ = json.Unmarshal(data, &object)
	value, _ := structpb.NewStruct(object)
	return value
}

func workflowDiagnosticsToProto(items []domain.WorkflowDiagnostic) []*domainpb.WorkflowDiagnostic {
	out := make([]*domainpb.WorkflowDiagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, &domainpb.WorkflowDiagnostic{Code: item.Code, Path: item.Path, Message: item.Message, Severity: item.Severity})
	}
	return out
}

func workflowRevisionToProto(revision *domain.WorkflowRevision) *domainpb.WorkflowRevision {
	if revision == nil {
		return nil
	}
	return &domainpb.WorkflowRevision{Id: revision.ID.String(), Owner: revision.Owner, Key: revision.Key, SemanticVersion: revision.SemanticVersion, Digest: revision.Digest, Definition: workflowDefinitionToProto(revision.Definition), SourcePath: revision.SourcePath, SourceHash: revision.SourceHash, SourceUpdatedAt: timestamppb.New(revision.SourceUpdatedAt), Active: revision.Active, CreatedAt: timestamppb.New(revision.CreatedAt), PromptStale: revision.PromptStale}
}

func workflowReconcileToProto(result *orchestration.ReconcileScenarioWorkflowsResult) *apipb.ReconcileScenarioWorkflowsResponse {
	if result == nil {
		return &apipb.ReconcileScenarioWorkflowsResponse{}
	}
	items := make([]*apipb.WorkflowReconcileResult, 0, len(result.Results))
	for _, item := range result.Results {
		items = append(items, workflowReconcileResultToProto(item))
	}
	return &apipb.ReconcileScenarioWorkflowsResponse{Scenario: result.Scenario, Results: items, Created: int32(result.Created), Activated: int32(result.Activated), Unchanged: int32(result.Unchanged), Skipped: int32(result.Skipped), Failed: int32(result.Failed), DryRun: result.DryRun, ValidateOnly: result.ValidateOnly}
}

func workflowReconcileResultToProto(item orchestration.WorkflowReconcileResult) *apipb.WorkflowReconcileResult {
	status := apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_UNSPECIFIED
	switch item.Status {
	case orchestration.WorkflowReconcileCreated:
		status = apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_CREATED
	case orchestration.WorkflowReconcileActivated:
		status = apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_ACTIVATED
	case orchestration.WorkflowReconcileUnchanged:
		status = apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_UNCHANGED
	case orchestration.WorkflowReconcileSkipped:
		status = apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_SKIPPED
	case orchestration.WorkflowReconcileFailedValidation:
		status = apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_FAILED_VALIDATION
	}
	return &apipb.WorkflowReconcileResult{WorkflowKey: item.WorkflowKey, Version: item.Version, Digest: item.Digest, SourcePath: item.SourcePath, Status: status, Message: item.Message, Diagnostics: workflowDiagnosticsToProto(item.Diagnostics)}
}

func workflowExecutionToProto(x *domain.WorkflowExecution, includePayloads bool) *domainpb.WorkflowExecution {
	if x == nil {
		return nil
	}
	status := domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
	switch x.Status {
	case domain.WorkflowExecutionPending:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING
	case domain.WorkflowExecutionRunning:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING
	case domain.WorkflowExecutionWaiting:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_WAITING
	case domain.WorkflowExecutionSucceeded:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED
	case domain.WorkflowExecutionBlocked:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED
	case domain.WorkflowExecutionAbstained:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED
	case domain.WorkflowExecutionBudgetExhausted:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED
	case domain.WorkflowExecutionFailed:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED
	case domain.WorkflowExecutionCancelled:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED
	case domain.WorkflowExecutionCancelling:
		status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING
	}
	toValue := func(raw json.RawMessage) *structpb.Value {
		if len(raw) == 0 {
			return nil
		}
		var value any
		if json.Unmarshal(raw, &value) != nil {
			return nil
		}
		out, _ := structpb.NewValue(value)
		return out
	}
	edges := map[string]int32{}
	for key, value := range x.EdgeTraversals {
		edges[key] = int32(value)
	}
	out := &domainpb.WorkflowExecution{Id: x.ID.String(), Owner: x.Owner, WorkflowKey: x.WorkflowKey, DefinitionDigest: x.DefinitionDigest, Status: status, CurrentNodeId: x.CurrentNodeID, BudgetUsage: &domainpb.WorkflowBudgetUsage{Turns: int32(x.BudgetUsage.Turns), Tokens: int32(x.BudgetUsage.Tokens), CostUsd: x.BudgetUsage.CostUSD, NodeAttempts: int32(x.BudgetUsage.NodeAttempts), Children: int32(x.BudgetUsage.Children), Retries: int32(x.BudgetUsage.Retries)}, EdgeTraversals: edges, Version: x.Version, IdempotencyKey: x.IdempotencyKey, Depth: int32(x.Depth), CreatedAt: timestamppb.New(x.CreatedAt), UpdatedAt: timestamppb.New(x.UpdatedAt)}
	if x.Status.Terminal() {
		receipt := &domainpb.ChargeReceipt{Currency: "USD", MeteringBasis: "agent-manager.run.billing.metered_charge_micro_usd", Measured: x.BudgetUsage.ChargeMeasured}
		if x.BudgetUsage.ChargeMeasured {
			amount := x.BudgetUsage.ChargeMicroUSD
			receipt.AmountMicroUsd = &amount
			receipt.Note = "per-execution metered charge from child-run billing"
		} else {
			receipt.MeteringBasis = "unmeasured"
			receipt.Note = "agent-manager could not attribute a metered charge to this execution"
		}
		out.ChargeReceipt = receipt
	}
	if includePayloads {
		out.Input = toValue(x.Input)
		out.Output = toValue(x.Output)
	}
	if x.ParentExecutionID != nil {
		out.ParentExecutionId = x.ParentExecutionID.String()
	}
	if x.ParentAttemptID != nil {
		out.ParentAttemptId = x.ParentAttemptID.String()
	}
	if x.TerminalReason != nil {
		out.TerminalReason = &domainpb.WorkflowTerminalReason{Code: x.TerminalReason.Code, Message: x.TerminalReason.Message, Retryable: x.TerminalReason.Retryable, BudgetName: x.TerminalReason.BudgetName}
	}
	if x.EndedAt != nil {
		out.EndedAt = timestamppb.New(*x.EndedAt)
	}
	return out
}

func workflowAttemptToProto(a *domain.WorkflowNodeAttempt) *domainpb.WorkflowNodeAttempt {
	if a == nil {
		return nil
	}
	digest := sha256.Sum256(a.InputSnapshot)
	out := &domainpb.WorkflowNodeAttempt{Id: a.ID.String(), ExecutionId: a.ExecutionID.String(), NodeId: a.NodeID, Ordinal: int32(a.Ordinal), Strategy: string(a.Strategy), Status: string(a.Status), IdempotencyKey: a.IdempotencyKey, ConversationId: a.ConversationID, ErrorCode: a.ErrorCode, Version: a.Version, CreatedAt: timestamppb.New(a.CreatedAt), UpdatedAt: timestamppb.New(a.UpdatedAt), ProfileIdentity: a.ProfileIdentity, InputSnapshotDigest: fmt.Sprintf("sha256:%x", digest[:]), InputSnapshotSizeBytes: int64(len(a.InputSnapshot))}
	// Raw output is diagnostic evidence for failed structured extraction only;
	// successful model output remains in the protected durable result path.
	if a.ValidationError != "" {
		out.RawOutput = a.RawOutput
		out.ValidationError = a.ValidationError
	}
	if a.RunID != nil {
		out.RunId = a.RunID.String()
	}
	if a.SourceAttemptID != nil {
		out.SourceAttemptId = a.SourceAttemptID.String()
	}
	if a.ChildExecutionID != nil {
		out.ChildExecutionId = a.ChildExecutionID.String()
	}
	if a.CompletedAt != nil {
		out.CompletedAt = timestamppb.New(*a.CompletedAt)
	}
	return out
}

func workflowJournalToProto(entry *domain.WorkflowJournalEntry) *domainpb.WorkflowJournalEntry {
	if entry == nil {
		return nil
	}
	digest := sha256.Sum256(entry.Payload)
	out := &domainpb.WorkflowJournalEntry{Id: entry.ID.String(), ExecutionId: entry.ExecutionID.String(), Sequence: entry.Sequence, Kind: string(entry.Kind), NodeId: entry.NodeID, PayloadDigest: fmt.Sprintf("sha256:%x", digest[:]), PayloadSizeBytes: int64(len(entry.Payload)), CreatedAt: timestamppb.New(entry.CreatedAt)}
	if entry.AttemptID != nil {
		out.AttemptId = entry.AttemptID.String()
	}
	return out
}
