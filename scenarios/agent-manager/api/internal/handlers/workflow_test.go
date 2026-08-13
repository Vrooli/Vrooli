package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestWorkflowExecutionProjectionRedactsPayloadsByDefault(t *testing.T) {
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "example", WorkflowKey: "example/review", DefinitionDigest: "sha256:test",
		Status: domain.WorkflowExecutionSucceeded, Input: json.RawMessage(`{"secret":"prompt"}`), Output: json.RawMessage(`{"secret":"result"}`),
		BudgetUsage:    domain.WorkflowBudgetUsage{ChargeMicroUSD: 42, ChargeMeasured: true},
		EdgeTraversals: map[string]int{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	redacted := workflowExecutionToProto(execution, false)
	if redacted.Input != nil || redacted.Output != nil {
		t.Fatalf("routine projection leaked payloads: input=%v output=%v", redacted.Input, redacted.Output)
	}
	authorized := workflowExecutionToProto(execution, true)
	if authorized.Input == nil || authorized.Output == nil {
		t.Fatalf("authorized projection omitted payloads: input=%v output=%v", authorized.Input, authorized.Output)
	}
	if receipt := authorized.GetChargeReceipt(); receipt == nil || !receipt.GetMeasured() || receipt.GetAmountMicroUsd() != 42 || receipt.GetCurrency() != "USD" {
		t.Fatalf("charge receipt=%v", receipt)
	}
}

func TestWorkflowExecutionProjectionPublishesUnmeasuredChargeReceipt(t *testing.T) {
	execution := &domain.WorkflowExecution{ID: uuid.New(), Status: domain.WorkflowExecutionFailed, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	receipt := workflowExecutionToProto(execution, false).GetChargeReceipt()
	if receipt == nil || receipt.GetMeasured() || receipt.AmountMicroUsd != nil || receipt.GetMeteringBasis() != "unmeasured" {
		t.Fatalf("unmeasured receipt=%v", receipt)
	}
}

func TestWorkflowAttemptProjectionExposesIdentityAndInputMetadataWithoutPayload(t *testing.T) {
	input := json.RawMessage(`{"customer":"sensitive"}`)
	attempt := &domain.WorkflowNodeAttempt{
		ID:              uuid.New(),
		ExecutionID:     uuid.New(),
		NodeID:          "review",
		Ordinal:         1,
		Strategy:        domain.WorkflowAttemptFreshRun,
		Status:          domain.WorkflowAttemptWaiting,
		InputSnapshot:   input,
		ProfileIdentity: "role:reviewer",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	projected := workflowAttemptToProto(attempt)
	digest := sha256.Sum256(input)
	if projected.GetProfileIdentity() != "role:reviewer" {
		t.Fatalf("profile identity = %q, want role:reviewer", projected.GetProfileIdentity())
	}
	if projected.GetInputSnapshotDigest() != fmt.Sprintf("sha256:%x", digest[:]) {
		t.Fatalf("input digest = %q", projected.GetInputSnapshotDigest())
	}
	if projected.GetInputSnapshotSizeBytes() != int64(len(input)) {
		t.Fatalf("input size = %d, want %d", projected.GetInputSnapshotSizeBytes(), len(input))
	}
}

func TestWorkflowAttemptProjectionExposesStructuredFailureEvidenceOnly(t *testing.T) {
	failed := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: uuid.New(), RawOutput: `{"wrong":true}`, ValidationError: "schema_mismatch", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	projected := workflowAttemptToProto(failed)
	if projected.GetRawOutput() != failed.RawOutput || projected.GetValidationError() != failed.ValidationError {
		t.Fatalf("failed evidence projection=%+v", projected)
	}
	success := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: uuid.New(), RawOutput: `{"answer":"ok"}`, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if projected := workflowAttemptToProto(success); projected.GetRawOutput() != "" || projected.GetValidationError() != "" {
		t.Fatalf("successful output leaked through attempt projection=%+v", projected)
	}
}

func TestWorkflowRevisionAndReconcileProjectionsPreserveAPIContract(t *testing.T) {
	now := time.Now().UTC()
	revision := &domain.WorkflowRevision{ID: uuid.New(), Owner: "agent-manager", Key: "review", SemanticVersion: "1.2.3", Digest: "sha256:abc", SourcePath: "workflow.json", SourceHash: "hash", SourceUpdatedAt: now, CreatedAt: now, Active: true, PromptStale: true}
	projected := workflowRevisionToProto(revision)
	if projected.GetId() != revision.ID.String() || projected.GetOwner() != revision.Owner || !projected.GetActive() || !projected.GetPromptStale() || projected.GetDefinition() == nil {
		t.Fatalf("revision=%+v", projected)
	}
	if workflowRevisionToProto(nil) != nil {
		t.Fatal("nil revision was projected")
	}
	statuses := []orchestration.WorkflowReconcileStatus{orchestration.WorkflowReconcileCreated, orchestration.WorkflowReconcileActivated, orchestration.WorkflowReconcileUnchanged, orchestration.WorkflowReconcileSkipped, orchestration.WorkflowReconcileFailedValidation}
	items := make([]orchestration.WorkflowReconcileResult, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, orchestration.WorkflowReconcileResult{WorkflowKey: string(status), Status: status, Diagnostics: []domain.WorkflowDiagnostic{{Code: "valid", Path: "nodes", Message: "ok", Severity: "info"}}})
	}
	response := workflowReconcileToProto(&orchestration.ReconcileScenarioWorkflowsResult{Scenario: "agent-manager", Results: items, Created: 1, Activated: 2, Unchanged: 3, Skipped: 4, Failed: 5, DryRun: true, ValidateOnly: true})
	if response.GetScenario() != "agent-manager" || len(response.GetResults()) != len(statuses) || response.GetCreated() != 1 || !response.GetDryRun() || !response.GetValidateOnly() {
		t.Fatalf("response=%+v", &response)
	}
	wants := []apipb.WorkflowReconcileStatus{apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_CREATED, apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_ACTIVATED, apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_UNCHANGED, apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_SKIPPED, apipb.WorkflowReconcileStatus_WORKFLOW_RECONCILE_STATUS_FAILED_VALIDATION}
	for index, want := range wants {
		if got := response.GetResults()[index].GetStatus(); got != want || len(response.GetResults()[index].GetDiagnostics()) != 1 {
			t.Fatalf("result %d=%+v", index, response.GetResults()[index])
		}
	}
	if got := workflowReconcileToProto(nil); got == nil || got.GetScenario() != "" {
		t.Fatalf("nil reconcile=%+v", got)
	}
}

func TestWorkflowCatalogHandlersReturnEmptyListAndNotFoundRevision(t *testing.T) {
	handler, _ := setupTestHandler(t)

	list := httptest.NewRecorder()
	handler.ListWorkflowRevisions(list, httptest.NewRequest(http.MethodGet, "/api/v1/workflows?owner=missing&key=missing/workflow&limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var listResponse apipb.ListWorkflowRevisionsResponse
	decodeProtoJSON(t, list.Body.Bytes(), &listResponse)
	if len(listResponse.GetRevisions()) != 0 {
		t.Fatalf("revisions=%+v, want empty catalog", listResponse.GetRevisions())
	}

	get := httptest.NewRecorder()
	handler.GetWorkflowRevision(get, httptest.NewRequest(http.MethodGet, "/api/v1/workflows/revision?owner=missing&key=missing/workflow", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
}

func TestWorkflowExecutionReadHandlersServeRedactedAndAuthorizedViews(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeCodex))
	now := time.Now().UTC()
	revision := &domain.WorkflowRevision{ID: uuid.New(), Owner: "owner", Key: "owner/review", SemanticVersion: "1.0.0", Digest: "sha256:workflow", Active: true, Definition: domain.WorkflowDefinition{SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "owner", Key: "owner/review", Version: "1.0.0"}, SourcePath: "workflow.json", SourceHash: "sha256:source", SourceUpdatedAt: now, CreatedAt: now}
	if err := repos.Workflows.ActivateBatch(context.Background(), []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("seed workflow revision: %v", err)
	}
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "owner", WorkflowKey: "owner/review", DefinitionDigest: "sha256:workflow", Status: domain.WorkflowExecutionSucceeded,
		Input: json.RawMessage(`{"secret":"input"}`), Output: json.RawMessage(`{"answer":"ok"}`), EdgeTraversals: map[string]int{"finish": 1}, Version: 2, CreatedAt: now, UpdatedAt: now, EndedAt: &now,
	}
	initial := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 0, Kind: domain.WorkflowJournalInput, Payload: json.RawMessage(`{"started":true}`), CreatedAt: now}
	if err := repos.WorkflowExecutions.Create(context.Background(), execution, initial); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-executions?owner=owner&workflow_key=owner/review&status=succeeded", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var listed apipb.ListWorkflowExecutionsResponse
	decodeProtoJSON(t, list.Body.Bytes(), &listed)
	if len(listed.GetExecutions()) != 1 || listed.GetExecutions()[0].GetInput() != nil || listed.GetExecutions()[0].GetOutput() != nil {
		t.Fatalf("listed executions=%+v", &listed)
	}

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-executions/"+execution.ID.String(), nil))
	if get.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
	var redacted apipb.WorkflowExecutionResponse
	decodeProtoJSON(t, get.Body.Bytes(), &redacted)
	if redacted.GetExecution().GetInput() != nil || redacted.GetExecution().GetOutput() != nil {
		t.Fatalf("default get leaked payloads: %+v", redacted.GetExecution())
	}

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-executions/"+execution.ID.String()+"/result", nil))
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("unauthorized result status=%d body=%s", denied.Code, denied.Body.String())
	}
	authorized := httptest.NewRecorder()
	router.ServeHTTP(authorized, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-executions/"+execution.ID.String()+"/result?explicitly_authorized=true", nil))
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized result status=%d body=%s", authorized.Code, authorized.Body.String())
	}
	var result apipb.WorkflowExecutionResponse
	decodeProtoJSON(t, authorized.Body.Bytes(), &result)
	if result.GetExecution().GetInput() == nil || result.GetExecution().GetOutput() == nil {
		t.Fatalf("authorized result omitted payloads: %+v", result.GetExecution())
	}

	trace := httptest.NewRecorder()
	router.ServeHTTP(trace, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-executions/"+execution.ID.String()+"/trace?after_sequence=-1&limit=10", nil))
	if trace.Code != http.StatusOK {
		t.Fatalf("trace status=%d body=%s", trace.Code, trace.Body.String())
	}
	var traced apipb.GetWorkflowExecutionTraceResponse
	decodeProtoJSON(t, trace.Body.Bytes(), &traced)
	if len(traced.GetJournal()) != 1 || traced.GetExecution().GetInput() != nil {
		t.Fatalf("trace=%+v", &traced)
	}
}

func TestScenarioDeclarationHandlersHonorDryRunAndProjectResults(t *testing.T) {
	_, router := setupTestHandler(t)
	cases := []struct {
		path  string
		body  proto.Message
		check func(*testing.T, []byte)
	}{
		{
			path: "/api/v1/profiles/reconcile-scenario",
			body: &apipb.ReconcileScenarioProfilesRequest{Scenario: "agent-manager", DryRun: true},
			check: func(t *testing.T, body []byte) {
				var response apipb.ReconcileScenarioProfilesResponse
				decodeProtoJSON(t, body, &response)
				if response.GetScenario() != "agent-manager" || !response.GetDryRun() {
					t.Fatalf("profile reconcile response=%+v", &response)
				}
			},
		},
		{
			path: "/api/v1/declarations/reconcile-scenario",
			body: &apipb.ReconcileScenarioDeclarationsRequest{Scenario: "agent-manager", DryRun: true},
			check: func(t *testing.T, body []byte) {
				var response apipb.ReconcileScenarioDeclarationsResponse
				decodeProtoJSON(t, body, &response)
				if response.GetScenario() != "agent-manager" || !response.GetDryRun() {
					t.Fatalf("declaration reconcile response=%+v", &response)
				}
			},
		},
		{
			path: "/api/v1/declarations/plan",
			body: &apipb.ReconcileScenarioDeclarationsRequest{Scenario: "agent-manager", DryRun: false, ValidateOnly: true},
			check: func(t *testing.T, body []byte) {
				var response apipb.ReconcileScenarioDeclarationsResponse
				decodeProtoJSON(t, body, &response)
				if response.GetScenario() != "agent-manager" || !response.GetDryRun() || !response.GetValidateOnly() {
					t.Fatalf("declaration plan response=%+v", &response)
				}
			},
		},
	}
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, test.path, bytes.NewReader(encodeProtoJSON(t, test.body))))
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			test.check(t, rr.Body.Bytes())
		})
	}
}

func TestResumeFromFailedRunHTTPCreatesTraceableFollowUp(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeCodex))
	ctx := context.Background()
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "resume profile", ProfileKey: "resume-profile", RoleRef: "code.default"}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	task := &domain.Task{ID: uuid.New(), Title: "resume task", Description: "complete the review", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	runID := uuid.New()
	stored := &domain.Run{ID: runID, TaskID: task.ID, AgentProfileID: &profile.ID, Tag: "resume-original", Status: domain.RunStatusFailed, Phase: domain.RunPhaseCompleted}
	if err := repos.Runs.Create(ctx, stored); err != nil {
		t.Fatalf("create failed original: %v", err)
	}
	body, err := json.Marshal(map[string]any{"runId": runID.String(), "customContext": "finish the remaining checks", "attachmentIds": []string{"attachment-1"}})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/resume-from-failed", bytes.NewReader(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("resume status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response apipb.CreateRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	resumedID, err := uuid.Parse(response.GetRun().GetId())
	if err != nil || resumedID == runID {
		t.Fatalf("resumed run=%+v original=%+v", response.GetRun(), stored)
	}
	resumed, err := repos.Runs.Get(ctx, resumedID)
	if err != nil || len(resumed.SourceRunIDs) != 1 || resumed.SourceRunIDs[0] != runID {
		t.Fatalf("resumed lineage=%+v err=%v", resumed, err)
	}
}

func TestWorkflowExecutionJournalAndOperationProjectionsCoverStatuses(t *testing.T) {
	now := time.Now().UTC()
	statuses := map[domain.WorkflowExecutionStatus]domainpb.WorkflowExecutionStatus{
		domain.WorkflowExecutionPending:         domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING,
		domain.WorkflowExecutionRunning:         domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING,
		domain.WorkflowExecutionWaiting:         domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_WAITING,
		domain.WorkflowExecutionSucceeded:       domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		domain.WorkflowExecutionBlocked:         domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		domain.WorkflowExecutionAbstained:       domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
		domain.WorkflowExecutionBudgetExhausted: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED,
		domain.WorkflowExecutionFailed:          domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED,
		domain.WorkflowExecutionCancelled:       domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED,
		domain.WorkflowExecutionCancelling:      domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING,
	}
	for status, want := range statuses {
		execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: "owner", WorkflowKey: "workflow", DefinitionDigest: "sha256:def", Status: status, Input: json.RawMessage(`{"input":true}`), Output: json.RawMessage(`{"output":true}`), EdgeTraversals: map[string]int{"next": 2}, Version: 3, CreatedAt: now, UpdatedAt: now}
		got := workflowExecutionToProto(execution, true)
		if got.GetStatus() != want || got.GetInput() == nil || got.GetOutput() == nil || got.GetEdgeTraversals()["next"] != 2 {
			t.Fatalf("status %q projection=%+v", status, got)
		}
		if operation := workflowOperationToProto(&orchestration.WorkflowExecutionOperationResult{Execution: execution, Idempotent: true}); operation.GetExecution().GetStatus() != want || !operation.GetIdempotent() {
			t.Fatalf("operation=%+v", operation)
		}
	}
	if workflowExecutionToProto(nil, true) != nil || workflowOperationToProto(nil) == nil {
		t.Fatal("nil workflow projections are unsafe")
	}
	attemptID := uuid.New()
	journal := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: uuid.New(), Sequence: 7, Kind: domain.WorkflowJournalKind("signal"), NodeID: "wait", Payload: json.RawMessage(`{"ok":true}`), AttemptID: &attemptID, CreatedAt: now}
	projectedJournal := workflowJournalToProto(journal)
	if projectedJournal.GetSequence() != 7 || projectedJournal.GetAttemptId() != attemptID.String() || projectedJournal.GetPayloadSizeBytes() == 0 {
		t.Fatalf("journal=%+v", projectedJournal)
	}
	if workflowJournalToProto(nil) != nil {
		t.Fatal("nil journal was projected")
	}
}

func TestWorkflowEndpointsRejectMalformedBodiesAndInvalidExecutionIDs(t *testing.T) {
	handler := &Handler{}
	invalidBody := func(call func(http.ResponseWriter, *http.Request)) func(*httptest.ResponseRecorder) {
		return func(rr *httptest.ResponseRecorder) {
			call(rr, httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString("{")))
		}
	}
	cases := []struct {
		name string
		call func(*httptest.ResponseRecorder)
	}{
		{"validate", invalidBody(handler.ValidateWorkflow)},
		{"reconcile", invalidBody(handler.ReconcileScenarioWorkflows)},
		{"plan", invalidBody(handler.PlanScenarioWorkflows)},
		{"start", invalidBody(handler.StartWorkflowExecution)},
		{"wait", invalidBody(handler.WaitWorkflowExecution)},
		{"signal", invalidBody(handler.SignalWorkflowExecution)},
		{"cancel", invalidBody(handler.CancelWorkflowExecution)},
		{"retry", invalidBody(handler.RetryWorkflowExecution)},
		{"resume", invalidBody(handler.ResumeWorkflowExecution)},
		{"simulate", invalidBody(handler.SimulateWorkflow)},
		{"get", func(rr *httptest.ResponseRecorder) {
			handler.GetWorkflowExecution(rr, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "invalid"}))
		}},
		{"advance", func(rr *httptest.ResponseRecorder) {
			handler.AdvanceWorkflowExecution(rr, mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/", nil), map[string]string{"id": "invalid"}))
		}},
		{"trace", func(rr *httptest.ResponseRecorder) {
			handler.GetWorkflowExecutionTrace(rr, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "invalid"}))
		}},
		{"runs", func(rr *httptest.ResponseRecorder) {
			handler.ListWorkflowExecutionRuns(rr, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "invalid"}))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(rr)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestValidateWorkflowReturnsStructuredDiagnosticsForInvalidDefinition(t *testing.T) {
	_, router := setupTestHandler(t)
	definition, err := structpb.NewStruct(map[string]any{"schemaVersion": "agent-workflow/v1", "owner": "owner", "key": "owner/invalid"})
	if err != nil {
		t.Fatal(err)
	}
	body := encodeProtoJSON(t, &apipb.ValidateWorkflowRequest{Definition: definition})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response apipb.ValidateWorkflowResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	if response.GetValid() || len(response.GetDiagnostics()) == 0 || response.GetDigest() != "" {
		t.Fatalf("response=%+v", &response)
	}
}

func TestWorkflowHTTPStartSignalAndSimulationUseDurableExecutionState(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeCodex))
	ctx := context.Background()
	now := time.Now().UTC()
	revision := &domain.WorkflowRevision{
		ID: uuid.New(), Owner: "owner", Key: "owner/wait", SemanticVersion: "1.0.0", Digest: "sha256:handler-wait", Active: true,
		Definition: domain.WorkflowDefinition{
			SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "owner", Key: "owner/wait", Version: "1.0.0", InputSchema: json.RawMessage(`{}`), OutputSchema: json.RawMessage(`{}`), EntryNode: "wait",
			Nodes: []domain.WorkflowNode{{ID: "wait", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "continue", TimeoutSeconds: 60}}, {ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}},
			Edges: []domain.WorkflowEdge{{From: "wait", To: "end"}}, Budgets: domain.WorkflowBudgets{WallTimeSeconds: 600, MaxTurns: 10, MaxTokens: 10000, MaxChargeMicroUSD: 10, MaxNodeAttempts: 10, MaxChildren: 10, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 60},
		},
		SourcePath: "workflow.json", SourceHash: "sha256:source", SourceUpdatedAt: now, CreatedAt: now,
	}
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate revision: %v", err)
	}
	input, err := structpb.NewValue(map[string]any{"request": "continue"})
	if err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/api/v1/workflow-executions", bytes.NewReader(encodeProtoJSON(t, &apipb.StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: revision.Key, Input: input, IdempotencyKey: "start-handler-wait"}))))
	if start.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	var started apipb.WorkflowExecutionResponse
	decodeProtoJSON(t, start.Body.Bytes(), &started)
	if started.GetExecution().GetStatus() != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_WAITING {
		t.Fatalf("started=%+v", started.GetExecution())
	}

	simulation := httptest.NewRecorder()
	router.ServeHTTP(simulation, httptest.NewRequest(http.MethodPost, "/api/v1/workflows/simulate", bytes.NewReader(encodeProtoJSON(t, &apipb.SimulateWorkflowRequest{Owner: "owner", WorkflowKey: revision.Key, DefinitionDigest: revision.Digest, Input: input}))))
	if simulation.Code != http.StatusOK {
		t.Fatalf("simulate status=%d body=%s", simulation.Code, simulation.Body.String())
	}
	var simulated apipb.SimulateWorkflowResponse
	decodeProtoJSON(t, simulation.Body.Bytes(), &simulated)
	if !simulated.GetValid() || len(simulated.GetNodes()) != 2 || simulated.GetNodes()[0].GetWaitSignal() != "continue" {
		t.Fatalf("simulation=%+v", &simulated)
	}

	signalPayload, err := structpb.NewValue(map[string]any{"approved": true})
	if err != nil {
		t.Fatal(err)
	}
	signal := httptest.NewRecorder()
	id := started.GetExecution().GetId()
	router.ServeHTTP(signal, httptest.NewRequest(http.MethodPost, "/api/v1/workflow-executions/"+id+"/signals", bytes.NewReader(encodeProtoJSON(t, &apipb.SignalWorkflowExecutionRequest{ExecutionId: id, Signal: "continue", Payload: signalPayload, IdempotencyKey: "signal-handler-wait"}))))
	if signal.Code != http.StatusOK {
		t.Fatalf("signal status=%d body=%s", signal.Code, signal.Body.String())
	}
	var signalled apipb.WorkflowExecutionOperationResponse
	decodeProtoJSON(t, signal.Body.Bytes(), &signalled)
	if signalled.GetExecution().GetStatus() != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		t.Fatalf("signal result=%+v", signalled.GetExecution())
	}
}

func TestWorkflowDefinitionAndDiagnosticProjection(t *testing.T) {
	definition := workflowDefinitionToProto(domain.WorkflowDefinition{Owner: "owner", Key: "owner/workflow", Version: "1.0.0"})
	if definition == nil || definition.AsMap()["owner"] != "owner" {
		t.Fatalf("definition=%+v", definition)
	}
	diagnostics := workflowDiagnosticsToProto([]domain.WorkflowDiagnostic{{Code: "invalid", Path: "nodes[0]", Message: "missing", Severity: "error"}})
	if len(diagnostics) != 1 || diagnostics[0].GetCode() != "invalid" {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}
