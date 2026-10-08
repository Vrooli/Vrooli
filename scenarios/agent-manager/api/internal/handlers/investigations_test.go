package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/investigation"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type investigationRepositoryStub struct {
	item   *investigation.Lifecycle
	reused bool
}

func (s *investigationRepositoryStub) Reserve(_ context.Context, request investigation.Request) (*investigation.Lifecycle, bool, error) {
	if s.item == nil {
		s.item = &investigation.Lifecycle{ID: "inv-api-1", Request: request, RequestDigest: "digest", OperationStatus: investigation.OperationQueued, CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()}
		return s.item, false, nil
	}
	s.reused = true
	return s.item, true, nil
}
func (s *investigationRepositoryStub) Get(context.Context, string) (*investigation.Lifecycle, error) {
	return s.item, nil
}
func (s *investigationRepositoryStub) List(context.Context, string, int) ([]*investigation.Lifecycle, error) {
	return []*investigation.Lifecycle{s.item}, nil
}
func (s *investigationRepositoryStub) Wait(context.Context, string, time.Duration) (*investigation.Lifecycle, bool, error) {
	return s.item, s.item != nil && s.item.OperationStatus == investigation.OperationCompleted, nil
}

func (s *investigationRepositoryStub) UpdateStatus(_ context.Context, _ string, status, workflowRef string) (*investigation.Lifecycle, error) {
	if s.item != nil {
		s.item.OperationStatus = status
		if workflowRef != "" {
			s.item.WorkflowRef = workflowRef
		}
	}
	return s.item, nil
}
func (s *investigationRepositoryStub) Complete(context.Context, string, investigation.Result) (*investigation.Lifecycle, error) {
	return s.item, nil
}
func (s *investigationRepositoryStub) Cancel(context.Context, string) (*investigation.Lifecycle, error) {
	return s.item, nil
}

func apiRequest() investigation.Request {
	return investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "api-key",
		Subject:        investigation.Subject{Owner: "run-set-owner", Kind: "run-set", Ref: "set-1", Revision: "1"},
		Question:       "Is the run progressing?",
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", MaxEvents: 8, MaxEvidenceBytes: 1024, MaxReconciliations: 1},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
}

type investigationWorkflowStub struct {
	orchestration.WorkflowService
	execution        *domain.WorkflowExecution
	settledExecution *domain.WorkflowExecution
	repository       *investigationRepositoryStub
	request          orchestration.StartWorkflowExecutionRequest
	settled          bool
}

func (s *investigationWorkflowStub) ReconcileTypedInvestigation(_ context.Context, _ *domain.WorkflowExecution) {
	s.settled = true
	if s.repository != nil && s.repository.item != nil {
		s.repository.item.OperationStatus = investigation.OperationCompleted
	}
}

func (s *investigationWorkflowStub) StartWorkflowExecution(_ context.Context, request orchestration.StartWorkflowExecutionRequest) (*domain.WorkflowExecution, error) {
	s.request = request
	return s.execution, nil
}

func (s *investigationWorkflowStub) GetWorkflowExecution(context.Context, uuid.UUID) (*domain.WorkflowExecution, error) {
	if s.settledExecution != nil {
		return s.settledExecution, nil
	}
	return s.execution, nil
}

func TestInvestigationAdmissionDispatchesDiagnosisOnlyWorkflow(t *testing.T) {
	repository := &investigationRepositoryStub{}
	executionID := uuid.New()
	workflow := &investigationWorkflowStub{execution: &domain.WorkflowExecution{ID: executionID, Status: domain.WorkflowExecutionSucceeded}}
	services := orchestration.EmptyHandlerServices()
	services.WorkflowService = workflow
	handler := New(services, WithInvestigationLifecycle(repository))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	body, _ := json.Marshal(apiRequest())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if recorder.Code != http.StatusCreated || repository.item.OperationStatus != investigation.OperationCollecting || repository.item.WorkflowRef != executionID.String() {
		t.Fatalf("dispatch status=%d item=%+v body=%s", recorder.Code, repository.item, recorder.Body.String())
	}
	if workflow.request.WorkflowKey != investigation.WorkflowKey || workflow.request.Initiator != domain.WorkflowInitiatorProgrammatic || workflow.request.IdempotencyKey != "typed-investigation/"+repository.item.ID {
		t.Fatalf("workflow request=%+v", workflow.request)
	}
	if !workflow.settled {
		t.Fatal("terminal workflow was not synchronously reconciled")
	}
	var input map[string]any
	if err := json.Unmarshal(workflow.request.Input, &input); err != nil || input["context"] != apiRequest().Question {
		t.Fatalf("workflow input=%s err=%v", workflow.request.Input, err)
	}
	if refs, ok := input["evidenceRefs"].([]any); !ok || refs == nil {
		t.Fatalf("workflow evidenceRefs=%#v, want an empty array when omitted", input["evidenceRefs"])
	}
}

func TestInvestigationAdmissionPassesTypedDomainEvidenceToWorkflow(t *testing.T) {
	repository := &investigationRepositoryStub{}
	executionID := uuid.New()
	workflow := &investigationWorkflowStub{execution: &domain.WorkflowExecution{ID: executionID, Status: domain.WorkflowExecutionRunning}}
	services := orchestration.EmptyHandlerServices()
	services.WorkflowService = workflow
	handler := New(services, WithInvestigationLifecycle(repository))
	request := apiRequest()
	request.DomainEvidence = []investigation.EvidenceReference{{Owner: "plan-manager", Kind: "brief", Ref: "brief-1", Revision: "7", SchemaVersion: "brief/v1"}}
	body, _ := json.Marshal(request)
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("admission status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var input map[string]any
	if err := json.Unmarshal(workflow.request.Input, &input); err != nil {
		t.Fatal(err)
	}
	refs, ok := input["evidenceRefs"].([]any)
	if !ok || len(refs) != 1 {
		t.Fatalf("workflow evidenceRefs=%#v", input["evidenceRefs"])
	}
	if refs[0].(map[string]any)["ref"] != "brief-1" {
		t.Fatalf("workflow evidence ref=%#v", refs[0])
	}
}

func TestInvestigationAdmissionReconcilesFastWorkflowAfterLifecycleLink(t *testing.T) {
	repository := &investigationRepositoryStub{}
	executionID := uuid.New()
	workflow := &investigationWorkflowStub{
		execution:        &domain.WorkflowExecution{ID: executionID, Status: domain.WorkflowExecutionRunning},
		settledExecution: &domain.WorkflowExecution{ID: executionID, Status: domain.WorkflowExecutionSucceeded, Output: json.RawMessage(`{"findings":{"summary":"bounded result","primaryCategory":"complete","categories":[]}}`)},
		repository:       repository,
	}
	services := orchestration.EmptyHandlerServices()
	services.WorkflowService = workflow
	handler := New(services, WithInvestigationLifecycle(repository))
	body, _ := json.Marshal(apiRequest())
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if recorder.Code != http.StatusCreated || !workflow.settled {
		t.Fatalf("admission status=%d settled=%v body=%s", recorder.Code, workflow.settled, recorder.Body.String())
	}
	if repository.item.OperationStatus != investigation.OperationCompleted {
		t.Fatalf("operation status=%q, want completed", repository.item.OperationStatus)
	}
}

func TestInvestigationAdmissionIncludesAuthoritativeBoundedRunReports(t *testing.T) {
	repository := &investigationRepositoryStub{}
	executionID := uuid.New()
	workflow := &investigationWorkflowStub{execution: &domain.WorkflowExecution{ID: executionID, Status: domain.WorkflowExecutionRunning}}
	runID := uuid.New()
	services := orchestration.EmptyHandlerServices()
	services.WorkflowService = workflow
	services.RunReportService = runReportServiceStub{report: &runreport.RunReport{RunID: runID, Status: "failed", Error: "heartbeat timeout"}}
	handler := New(services, WithInvestigationLifecycle(repository))
	request := apiRequest()
	request.Subject.RunIDs = []string{runID.String()}
	body, _ := json.Marshal(request)
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("admission status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var input map[string]any
	if err := json.Unmarshal(workflow.request.Input, &input); err != nil {
		t.Fatal(err)
	}
	contextText, _ := input["context"].(string)
	if !strings.Contains(contextText, "Question: Is the run progressing?") || !strings.Contains(contextText, runID.String()) || !strings.Contains(contextText, "heartbeat timeout") {
		t.Fatalf("workflow context omitted bounded run evidence: %q", contextText)
	}
}

func TestInvestigationRoutesAdmitAndReuseStableRequest(t *testing.T) {
	repository := &investigationRepositoryStub{}
	handler := New(orchestration.EmptyHandlerServices(), WithInvestigationLifecycle(repository))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	body, err := json.Marshal(apiRequest())
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if first.Code != http.StatusCreated || repository.item == nil {
		t.Fatalf("first admission status=%d body=%s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if second.Code != http.StatusOK || !repository.reused {
		t.Fatalf("second admission status=%d body=%s reused=%v", second.Code, second.Body.String(), repository.reused)
	}
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/investigations/inv-api-1", nil))
	if get.Code != http.StatusOK || !bytes.Contains(get.Body.Bytes(), []byte("inv-api-1")) {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
}

func TestInvestigationRouteRejectsUnknownAuthorityBeforeReserve(t *testing.T) {
	repository := &investigationRepositoryStub{}
	handler := New(orchestration.EmptyHandlerServices(), WithInvestigationLifecycle(repository))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	request := apiRequest()
	request.CallerAuthority = "not-authorized"
	body, _ := json.Marshal(request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/investigations", bytes.NewReader(body)))
	if recorder.Code != http.StatusBadRequest || repository.item != nil {
		t.Fatalf("invalid authority status=%d body=%s item=%+v", recorder.Code, recorder.Body.String(), repository.item)
	}
}
