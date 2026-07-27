package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"scenario-to-desktop-api/agentmanager"
	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/fix"
	"scenario-to-desktop-api/tasks/shared"
	"strings"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// --- Mock implementations ---

// mockInvStore implements InvestigationStore for testing.
type mockInvStore struct {
	investigations map[string]*domain.Investigation
	activeByPipe   map[string]*domain.Investigation
	createErr      error
	getActiveErr   error
	updateStatusFn func(id string, status domain.InvestigationStatus) error
	cancelFuncs    map[string]context.CancelFunc
}

func newMockInvStore() *mockInvStore {
	return &mockInvStore{
		investigations: make(map[string]*domain.Investigation),
		activeByPipe:   make(map[string]*domain.Investigation),
		cancelFuncs:    make(map[string]context.CancelFunc),
	}
}

func (m *mockInvStore) Create(inv *domain.Investigation) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.investigations[inv.ID] = inv
	return nil
}

func (m *mockInvStore) Get(id string) (*domain.Investigation, error) {
	inv, ok := m.investigations[id]
	if !ok {
		return nil, nil
	}
	return inv, nil
}

func (m *mockInvStore) GetForPipeline(pipelineID, id string) (*domain.Investigation, error) {
	inv, ok := m.investigations[id]
	if !ok || inv.PipelineID != pipelineID {
		return nil, nil
	}
	return inv, nil
}

func (m *mockInvStore) GetActive(pipelineID string) (*domain.Investigation, error) {
	if m.getActiveErr != nil {
		return nil, m.getActiveErr
	}
	return m.activeByPipe[pipelineID], nil
}

func (m *mockInvStore) List(pipelineID string, limit int) ([]*domain.Investigation, error) {
	var result []*domain.Investigation
	for _, inv := range m.investigations {
		if inv.PipelineID == pipelineID {
			result = append(result, inv)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockInvStore) Update(id string, fn func(inv *domain.Investigation)) error {
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	fn(inv)
	return nil
}

func (m *mockInvStore) UpdateStatus(id string, status domain.InvestigationStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, status)
	}
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	inv.Status = status
	return nil
}

func (m *mockInvStore) UpdateProgress(id string, progress int) error {
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	inv.Progress = progress
	return nil
}

func (m *mockInvStore) UpdateRunID(id, runID string) error {
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	inv.AgentRunID = &runID
	return nil
}

func (m *mockInvStore) UpdateFindings(id, findings string, details json.RawMessage) error {
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	inv.Findings = &findings
	inv.Status = domain.InvestigationStatusCompleted
	inv.Details = details
	return nil
}

func (m *mockInvStore) UpdateError(id, errorMsg string) error {
	inv, ok := m.investigations[id]
	if !ok {
		return fmt.Errorf("not found: %s", id)
	}
	inv.ErrorMessage = &errorMsg
	inv.Status = domain.InvestigationStatusFailed
	return nil
}

func (m *mockInvStore) UpdateErrorWithDetails(id, errorMsg string, details json.RawMessage) error {
	return m.UpdateError(id, errorMsg)
}

func (m *mockInvStore) SetCancel(id string, cancel context.CancelFunc) {
	m.cancelFuncs[id] = cancel
}

func (m *mockInvStore) TakeCancel(id string) context.CancelFunc {
	fn := m.cancelFuncs[id]
	delete(m.cancelFuncs, id)
	return fn
}

func (m *mockInvStore) ClearCancel(id string) {
	delete(m.cancelFuncs, id)
}

// mockPipelineStore implements PipelineStore.
type mockPipelineStore struct {
	statuses map[string]*pipeline.Status
}

func newMockPipelineStore() *mockPipelineStore {
	return &mockPipelineStore{statuses: make(map[string]*pipeline.Status)}
}

func (m *mockPipelineStore) Get(pipelineID string) (*pipeline.Status, bool) {
	s, ok := m.statuses[pipelineID]
	return s, ok
}

// mockAgentExecutor implements AgentExecutor.
type mockAgentExecutor struct {
	available bool
	url       string
	urlErr    error
}

func (m *mockAgentExecutor) IsAvailable(ctx context.Context) bool { return m.available }
func (m *mockAgentExecutor) ExecuteAsync(ctx context.Context, req agentmanager.ExecuteRequest) (string, error) {
	return "run-123", nil
}

func (m *mockAgentExecutor) GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	return nil, nil
}
func (m *mockAgentExecutor) StopRun(ctx context.Context, runID string) error { return nil }
func (m *mockAgentExecutor) ResolveURL(ctx context.Context) (string, error) {
	return m.url, m.urlErr
}

// mockProgressHub implements ProgressBroadcaster.
type mockProgressHub struct {
	events []progressEvent
}

type progressEvent struct {
	pipelineID, invID, eventType, message string
	progress                              float64
}

func (m *mockProgressHub) BroadcastInvestigation(pipelineID, invID, eventType string, progress float64, message string) {
	m.events = append(m.events, progressEvent{pipelineID, invID, eventType, message, progress})
}

// mockTaskHandler implements TaskHandler for testing.
type mockTaskHandler struct {
	taskType       domain.TaskType
	tag            string
	shouldContinue bool
}

func (m *mockTaskHandler) TaskType() domain.TaskType { return m.taskType }
func (m *mockTaskHandler) BuildPromptAndContext(ctx context.Context, input TaskInput) (PromptResult, error) {
	return shared.PromptResult{Prompt: "test prompt"}, nil
}
func (m *mockTaskHandler) AgentTag() string { return m.tag }
func (m *mockTaskHandler) ShouldContinue(ctx context.Context, task *domain.Investigation, result *AgentResult) (bool, string) {
	return m.shouldContinue, ""
}

// --- Helper to build a valid investigate request ---

func validInvestigateRequest(pipelineID string) domain.CreateTaskRequest {
	return domain.CreateTaskRequest{
		PipelineID: pipelineID,
		TaskType:   domain.TaskTypeInvestigate,
		Focus:      domain.TaskFocus{Harness: true},
		Effort:     domain.EffortLogs,
	}
}

func validFixRequest(pipelineID, sourceInvID string) domain.CreateTaskRequest {
	return domain.CreateTaskRequest{
		PipelineID:            pipelineID,
		TaskType:              domain.TaskTypeFix,
		Focus:                 domain.TaskFocus{Subject: true},
		Permissions:           domain.FixPermissions{Immediate: true},
		SourceInvestigationID: sourceInvID,
		MaxIterations:         3,
	}
}

// --- Helper to create a standard test service ---

func newTestService() (*Service, *mockInvStore, *mockPipelineStore, *mockAgentExecutor, *mockProgressHub) {
	invStore := newMockInvStore()
	pipeStore := newMockPipelineStore()
	agentExec := &mockAgentExecutor{available: true}
	hub := &mockProgressHub{}

	// Use NewService but we need to override handlers since it imports investigate/fix
	svc := NewService(invStore, pipeStore, agentExec, hub, "http://127.0.0.1:19001")

	return svc, invStore, pipeStore, agentExec, hub
}

// --- TriggerTask validation tests ---

func TestTriggerTask_InvalidRequest(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	// Empty pipeline ID
	req := domain.CreateTaskRequest{
		TaskType: domain.TaskTypeInvestigate,
		Focus:    domain.TaskFocus{Harness: true},
	}
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty pipeline ID")
	}
	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("error = %q, want 'invalid request'", err.Error())
	}
}

func TestTriggerTask_UnknownTaskType(t *testing.T) {
	svc, _, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	// Register a custom handler for an unknown type to bypass the validate step
	// Actually, Validate() will catch invalid task types first
	req := domain.CreateTaskRequest{
		PipelineID: "pipe-1",
		TaskType:   "unknown",
		Focus:      domain.TaskFocus{Harness: true},
	}
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for unknown task type")
	}
	_ = pipeStore // suppress unused
}

func TestTriggerTask_PipelineNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	req := validInvestigateRequest("nonexistent-pipe")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing pipeline")
	}
	if !strings.Contains(err.Error(), "pipeline not found") {
		t.Errorf("error = %q, want 'pipeline not found'", err.Error())
	}
}

func TestTriggerTask_AgentNotAvailable(t *testing.T) {
	svc, _, pipeStore, agentExec, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	agentExec.available = false

	req := validInvestigateRequest("pipe-1")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error when agent unavailable")
	}
	if !strings.Contains(err.Error(), "agent-manager is not available") {
		t.Errorf("error = %q, want 'agent-manager is not available'", err.Error())
	}
}

func TestTriggerTask_ActiveInvestigationExists(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	invStore.activeByPipe["pipe-1"] = &domain.Investigation{
		ID:         "existing-inv",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusRunning,
	}

	req := validInvestigateRequest("pipe-1")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error when active investigation exists")
	}
	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("error = %q, want 'already in progress'", err.Error())
	}
}

func TestTriggerTask_GetActiveError(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	invStore.getActiveErr = fmt.Errorf("db connection lost")

	req := validInvestigateRequest("pipe-1")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error when GetActive fails")
	}
	if !strings.Contains(err.Error(), "failed to check active") {
		t.Errorf("error = %q, want 'failed to check active'", err.Error())
	}
}

func TestStoreInvestigationResultsPersistsFindingsAndFailureDetails(t *testing.T) {
	for _, test := range []struct {
		name       string
		result     *AgentResult
		wantStatus domain.InvestigationStatus
		wantEvent  string
	}{
		{"findings", &AgentResult{RunID: "run-1", Success: true, Output: "Root cause identified", DurationSeconds: 12, TokensUsed: 34}, domain.InvestigationStatusCompleted, EventInvestigationCompleted},
		{"empty output", &AgentResult{RunID: "run-2", Success: true}, domain.InvestigationStatusFailed, EventInvestigationFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, store, _, _, hub := newTestService()
			store.investigations["task"] = &domain.Investigation{ID: "task", PipelineID: "pipe"}
			status := &pipeline.Status{PipelineID: "pipe", Stages: map[string]*pipeline.StageResult{
				pipeline.StageBuild: {Status: pipeline.StatusFailed},
			}}
			svc.storeInvestigationResults(context.Background(), "task", status, &domain.CreateTaskRequest{
				TaskType: domain.TaskTypeInvestigate, Effort: domain.EffortLogs,
			}, test.result)
			stored := store.investigations["task"]
			if stored.Status != test.wantStatus || len(hub.events) != 1 || hub.events[0].eventType != test.wantEvent {
				t.Fatalf("stored=%#v events=%#v", stored, hub.events)
			}
			if test.wantStatus == domain.InvestigationStatusCompleted && (stored.Findings == nil || *stored.Findings != "Root cause identified") {
				t.Fatalf("findings were not stored: %#v", stored)
			}
			if test.wantStatus == domain.InvestigationStatusFailed && (stored.ErrorMessage == nil || !strings.Contains(*stored.ErrorMessage, "did not produce")) {
				t.Fatalf("empty output error was not stored: %#v", stored)
			}
		})
	}
}

func TestStoreFixResultsReportsSuccessAndTermination(t *testing.T) {
	for _, test := range []struct {
		name       string
		final      string
		wantStatus domain.InvestigationStatus
		wantEvent  string
	}{
		{"success", FixStatusSuccess, domain.InvestigationStatusCompleted, EventFixCompleted},
		{"max iterations", FixStatusMaxIterations, domain.InvestigationStatusFailed, EventFixFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, store, _, _, hub := newTestService()
			store.investigations["task"] = &domain.Investigation{ID: "task", PipelineID: "pipe"}
			findings := "prior investigation"
			loop := fix.NewLoopState(fix.DefaultLoopConfig(2, "http://pipeline"))
			loop.StartIteration()
			loop.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "changed", VerifyResult: "passed"})
			loop.FinalStatus = test.final
			svc.storeFixResults(context.Background(), "task", &pipeline.Status{PipelineID: "pipe"}, &domain.CreateTaskRequest{
				TaskType: domain.TaskTypeFix, SourceInvestigationID: "source", Permissions: domain.FixPermissions{Immediate: true},
			}, loop, &findings)
			stored := store.investigations["task"]
			if stored.Status != test.wantStatus || len(hub.events) != 1 || hub.events[0].eventType != test.wantEvent {
				t.Fatalf("stored=%#v events=%#v", stored, hub.events)
			}
			if test.wantStatus == domain.InvestigationStatusCompleted && (stored.Findings == nil || !strings.Contains(*stored.Findings, "Fix Task Summary")) {
				t.Fatalf("success findings missing: %#v", stored)
			}
		})
	}
}

func TestUpdateLoopStatePersistsSerializableFixProgress(t *testing.T) {
	svc, store, _, _, _ := newTestService()
	store.investigations["task"] = &domain.Investigation{ID: "task", PipelineID: "pipe"}
	loop := fix.NewLoopState(fix.DefaultLoopConfig(3, "http://pipeline"))
	loop.StartIteration()
	loop.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "changed", VerifyResult: "passed"})

	svc.updateLoopState(context.Background(), "task", loop)
	if !strings.Contains(string(store.investigations["task"].Details), "current_iteration") {
		t.Fatalf("loop state was not persisted: %s", store.investigations["task"].Details)
	}
}

func TestTriggerTask_FixWithoutSourceInvestigation(t *testing.T) {
	svc, _, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}

	req := validFixRequest("pipe-1", "")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for fix without source investigation")
	}
	if !strings.Contains(err.Error(), "source_investigation_id is required") {
		t.Errorf("error = %q, want 'source_investigation_id is required'", err.Error())
	}
}

func TestTriggerTask_FixSourceNotFound(t *testing.T) {
	svc, _, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}

	req := validFixRequest("pipe-1", "nonexistent-inv")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing source investigation")
	}
	if !strings.Contains(err.Error(), "source investigation not found") {
		t.Errorf("error = %q, want 'source investigation not found'", err.Error())
	}
}

func TestTriggerTask_FixSourceNotCompleted(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	invStore.investigations["src-inv"] = &domain.Investigation{
		ID:         "src-inv",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusRunning,
	}

	req := validFixRequest("pipe-1", "src-inv")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for non-completed source")
	}
	if !strings.Contains(err.Error(), "expected completed") {
		t.Errorf("error = %q, want 'expected completed'", err.Error())
	}
}

func TestTriggerTask_FixSourceNoFindings(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	invStore.investigations["src-inv"] = &domain.Investigation{
		ID:         "src-inv",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusCompleted,
		Findings:   nil,
	}

	req := validFixRequest("pipe-1", "src-inv")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for source with no findings")
	}
	if !strings.Contains(err.Error(), "no findings") {
		t.Errorf("error = %q, want 'no findings'", err.Error())
	}
}

func TestTriggerTask_FixSourceEmptyFindings(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	empty := ""
	invStore.investigations["src-inv"] = &domain.Investigation{
		ID:         "src-inv",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusCompleted,
		Findings:   &empty,
	}

	req := validFixRequest("pipe-1", "src-inv")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty findings")
	}
	if !strings.Contains(err.Error(), "no findings") {
		t.Errorf("error = %q, want 'no findings'", err.Error())
	}
}

func TestTriggerTask_CreateStoreError(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	invStore.createErr = fmt.Errorf("disk full")

	req := validInvestigateRequest("pipe-1")
	_, err := svc.TriggerTask(ctx, req)
	if err == nil {
		t.Fatal("expected error when Create fails")
	}
	if !strings.Contains(err.Error(), "failed to create investigation") {
		t.Errorf("error = %q, want 'failed to create investigation'", err.Error())
	}
}

func TestTriggerTask_InvestigateSuccess(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}

	req := validInvestigateRequest("pipe-1")
	inv, err := svc.TriggerTask(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected non-nil investigation")
	}
	if inv.ID == "" {
		t.Error("investigation ID should be set")
	}
	if inv.PipelineID != "pipe-1" {
		t.Errorf("PipelineID = %q, want pipe-1", inv.PipelineID)
	}
	if inv.Status != domain.InvestigationStatusPending && inv.Status != domain.InvestigationStatusRunning {
		t.Errorf("Status = %q, want pending or running", inv.Status)
	}

	// Verify it was stored
	if _, ok := invStore.investigations[inv.ID]; !ok {
		t.Error("investigation should be stored in invStore")
	}

	// Verify cancel was set
	if _, ok := invStore.cancelFuncs[inv.ID]; !ok {
		t.Error("cancel func should be set for the investigation")
	}

	// The worker intentionally outlives the request. Shut it down explicitly so
	// this test does not leave an asynchronous poller behind.
	ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Shutdown(ctxShutdown); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestTriggerTask_FixSuccess(t *testing.T) {
	svc, invStore, pipeStore, _, _ := newTestService()
	ctx := context.Background()

	pipeStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}
	findings := "Found issue: missing dependency"
	invStore.investigations["src-inv"] = &domain.Investigation{
		ID:         "src-inv",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusCompleted,
		Findings:   &findings,
	}

	req := validFixRequest("pipe-1", "src-inv")
	inv, err := svc.TriggerTask(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected non-nil investigation")
	}
	if inv.Status != domain.InvestigationStatusPending && inv.Status != domain.InvestigationStatusRunning {
		t.Errorf("Status = %q, want pending or running", inv.Status)
	}
	ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Shutdown(ctxShutdown); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
