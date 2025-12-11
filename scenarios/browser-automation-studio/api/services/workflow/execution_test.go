package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestUnsupportedAutomationNodes(t *testing.T) {
	tests := []struct {
		name         string
		flow         database.JSONMap
		expectUnsafe bool
		expected     []string
	}{
		{
			name: "simple linear nodes allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "navigate"},
					map[string]any{"id": "2", "type": "click"},
					map[string]any{"id": "3", "type": "assert"},
				},
			},
		},
		{
			name: "condition node allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "condition"},
					map[string]any{"id": "2", "type": "click"},
				},
			},
		},
		{
			name: "repeat loop allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "loop", "data": map[string]any{"loopType": "repeat", "loopCount": 2}},
					map[string]any{"id": "2", "type": "click"},
				},
			},
		},
		{
			name: "missing nodes ignored",
			flow: database.JSONMap{},
		},
		{
			name: "loop type mismatch flagged",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "loop1", "type": "loop", "data": map[string]any{"loopType": "until"}},
				},
			},
			expectUnsafe: true,
			expected:     []string{"loop:until"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			unsupported := unsupportedAutomationNodes(tt.flow)
			if tt.expectUnsafe && len(unsupported) == 0 {
				t.Fatalf("expected unsupported nodes, got none")
			}
			if !tt.expectUnsafe && len(unsupported) > 0 {
				t.Fatalf("expected no unsupported nodes, got %v", unsupported)
			}
			if len(tt.expected) > 0 {
				if len(unsupported) != len(tt.expected) {
					t.Fatalf("expected %v, got %v", tt.expected, unsupported)
				}
				for i := range tt.expected {
					if unsupported[i] != tt.expected[i] {
						t.Fatalf("expected %v, got %v", tt.expected, unsupported)
					}
				}
			}
		})
	}
}

type closableSink struct {
	closed []uuid.UUID
}

func (c *closableSink) Publish(_ context.Context, _ autocontracts.EventEnvelope) error { return nil }
func (c *closableSink) Limits() autocontracts.EventBufferLimits {
	return autocontracts.DefaultEventBufferLimits
}
func (c *closableSink) CloseExecution(id uuid.UUID) { c.closed = append(c.closed, id) }

func TestCloseEventSink(t *testing.T) {
	execID := uuid.New()
	sink := &closableSink{}
	closeEventSink(sink, execID)
	if len(sink.closed) != 1 || sink.closed[0] != execID {
		t.Fatalf("expected close to be invoked with %s, got %+v", execID, sink.closed)
	}

	var nilSink autoevents.Sink
	closeEventSink(nilSink, execID) // should not panic
}

type markerRecorder struct {
	marked []autocontracts.StepFailure
}

func (m *markerRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure autocontracts.StepFailure) error {
	m.marked = append(m.marked, failure)
	return nil
}

func (m *markerRecorder) RecordStepOutcome(ctx context.Context, plan autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (autorecorder.RecordResult, error) {
	return autorecorder.RecordResult{}, nil
}

func (m *markerRecorder) RecordTelemetry(ctx context.Context, plan autocontracts.ExecutionPlan, telemetry autocontracts.StepTelemetry) error {
	return nil
}

func (m *markerRecorder) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	return nil
}

// Ensure recordExecutionMarker uses recorder without panicking when absent.
func TestRecordExecutionMarker(t *testing.T) {
	service := &WorkflowService{}
	// No recorder configured should be a no-op.
	service.recordExecutionMarker(context.Background(), uuid.New(), autocontracts.StepFailure{Kind: autocontracts.FailureKindTimeout})

	rec := &markerRecorder{}
	service.artifactRecorder = rec
	service.recordExecutionMarker(context.Background(), uuid.New(), autocontracts.StepFailure{Kind: autocontracts.FailureKindTimeout})

	if len(rec.marked) != 1 {
		t.Fatalf("expected one crash marker, got %d", len(rec.marked))
	}
	if rec.marked[0].Kind != autocontracts.FailureKindTimeout {
		t.Fatalf("expected timeout failure kind, got %+v", rec.marked[0])
	}
}

func TestApplyCapabilityError(t *testing.T) {
	exec := &database.Execution{}
	capErr := &autoexecutor.CapabilityError{
		Engine:   "stub",
		Missing:  []string{"har", "downloads"},
		Warnings: []string{"viewport_width>=2000"},
		Reasons: map[string][]string{
			"har":         {"step 1 (mock): network mock"},
			"downloads":   {"step 2 (download): type download"},
			"tracing":     {"metadata.requiresTracing"},
			"parallel":    {},
			"file_upload": nil,
		},
	}

	failure := applyCapabilityError(exec, capErr)

	if exec.Status != "failed" {
		t.Fatalf("expected status failed, got %s", exec.Status)
	}
	if exec.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to be set")
	}
	if !exec.Error.Valid || !strings.Contains(exec.Error.String, "har") {
		t.Fatalf("expected error string to mention missing capabilities, got %q", exec.Error.String)
	}
	if failure.Kind != autocontracts.FailureKindOrchestration || failure.Code != "capability_mismatch" {
		t.Fatalf("unexpected failure metadata: %+v", failure)
	}
	if missing, ok := failure.Details["missing"].([]string); !ok || len(missing) == 0 {
		t.Fatalf("expected missing details to be populated, got %+v", failure.Details)
	}
}

type stubPlanCompiler struct {
	called bool
	plan   autocontracts.ExecutionPlan
	instr  []autocontracts.CompiledInstruction
}

func (s *stubPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (autocontracts.ExecutionPlan, []autocontracts.CompiledInstruction, error) {
	s.called = true
	return s.plan, s.instr, nil
}

type stubEngine struct {
	session *stubSession
}

func (s *stubEngine) Name() string { return "stub" }

func (s *stubEngine) Capabilities(ctx context.Context) (autocontracts.EngineCapabilities, error) {
	return autocontracts.EngineCapabilities{
		SchemaVersion:         autocontracts.CapabilitiesSchemaVersion,
		Engine:                s.Name(),
		MaxViewportWidth:      1920,
		MaxViewportHeight:     1080,
		AllowsParallelTabs:    true,
		SupportsHAR:           true,
		SupportsVideo:         true,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     true,
		SupportsTracing:       true,
		MaxConcurrentSessions: 1,
	}, nil
}

func (s *stubEngine) StartSession(ctx context.Context, spec autoengine.SessionSpec) (autoengine.EngineSession, error) {
	if s.session == nil {
		s.session = &stubSession{}
	}
	return s.session, nil
}

type stubSession struct {
	runs int
}

func (s *stubSession) Run(ctx context.Context, instruction autocontracts.CompiledInstruction) (autocontracts.StepOutcome, error) {
	s.runs++
	now := time.Now().UTC()
	return autocontracts.StepOutcome{
		SchemaVersion:  autocontracts.StepOutcomeSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		StepIndex:      instruction.Index,
		NodeID:         instruction.NodeID,
		StepType:       instruction.Type,
		Success:        true,
		StartedAt:      now,
		CompletedAt: func() *time.Time {
			t := now
			return &t
		}(),
	}, nil
}

func (s *stubSession) Reset(ctx context.Context) error { return nil }
func (s *stubSession) Close(ctx context.Context) error { return nil }

type stubRecorder struct {
	outcomes []autocontracts.StepOutcome
}

func (r *stubRecorder) RecordStepOutcome(ctx context.Context, plan autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (autorecorder.RecordResult, error) {
	r.outcomes = append(r.outcomes, outcome)
	return autorecorder.RecordResult{}, nil
}

func (r *stubRecorder) RecordTelemetry(ctx context.Context, plan autocontracts.ExecutionPlan, telemetry autocontracts.StepTelemetry) error {
	return nil
}

func (r *stubRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure autocontracts.StepFailure) error {
	return nil
}

func (r *stubRecorder) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	return nil
}

// Verify executeWithAutomationEngine respects injected plan compiler / engine / recorder.
func TestExecuteWithAutomationEngine_UsesInjectedCompilerAndEngine(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	execution := &database.Execution{ID: execID, WorkflowID: wfID}
	workflow := &database.Workflow{ID: wfID}

	compiler := &stubPlanCompiler{
		plan: autocontracts.ExecutionPlan{
			SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
			PayloadVersion: autocontracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     wfID,
			CreatedAt:      time.Now().UTC(),
			Instructions: []autocontracts.CompiledInstruction{
				{Index: 0, NodeID: "n1", Type: "noop"},
			},
		},
		instr: []autocontracts.CompiledInstruction{
			{Index: 0, NodeID: "n1", Type: "noop"},
		},
	}

	engine := &stubEngine{}
	rec := &stubRecorder{}
	exec := autoexecutor.NewSimpleExecutor(nil)
	eventSink := autoevents.NewMemorySink(autocontracts.DefaultEventBufferLimits)

	svc := NewWorkflowServiceWithDeps(nil, nil, nil, WorkflowServiceOptions{
		Executor:         exec,
		EngineFactory:    autoengine.NewStaticFactory(engine),
		ArtifactRecorder: rec,
		PlanCompiler:     compiler,
	})

	err := svc.executeWithAutomationEngine(context.Background(), execution, workflow, autoengine.SelectionConfig{DefaultEngine: "stub"}, eventSink)
	if err != nil {
		t.Fatalf("executeWithAutomationEngine returned error: %v", err)
	}

	if !compiler.called {
		t.Fatalf("expected custom plan compiler to be used")
	}
	if len(rec.outcomes) != 1 {
		t.Fatalf("expected one recorded outcome, got %d", len(rec.outcomes))
	}
	if engine.session == nil || engine.session.runs != 1 {
		t.Fatalf("expected stub engine session to run once, got %+v", engine.session)
	}
	if events := eventSink.Events(); len(events) == 0 {
		t.Fatalf("expected events emitted to sink")
	}
}

// resumeRepositoryStub provides a minimal stub for testing resume execution.
// It only implements the methods required by ResumeExecution.
type resumeRepositoryStub struct {
	database.Repository
	execution        *database.Execution
	workflow         *database.Workflow
	lastStepIndex    int
	completedSteps   []*database.ExecutionStep
	createdExecution *database.Execution
	getError         error
}

func (r *resumeRepositoryStub) GetResumableExecution(ctx context.Context, id uuid.UUID) (*database.Execution, int, error) {
	if r.getError != nil {
		return nil, -1, r.getError
	}
	if r.execution == nil {
		return nil, -1, database.ErrNotFound
	}
	return r.execution, r.lastStepIndex, nil
}

func (r *resumeRepositoryStub) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	if r.workflow == nil {
		return nil, database.ErrNotFound
	}
	return r.workflow, nil
}

func (r *resumeRepositoryStub) GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return r.completedSteps, nil
}

func (r *resumeRepositoryStub) CreateExecution(ctx context.Context, execution *database.Execution) error {
	r.createdExecution = execution
	return nil
}

func (r *resumeRepositoryStub) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}

func (r *resumeRepositoryStub) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}

func (r *resumeRepositoryStub) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}

func TestResumeExecution_CreatesCorrectTriggerMetadata(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()

	repo := &resumeRepositoryStub{
		execution: &database.Execution{
			ID:         execID,
			WorkflowID: wfID,
			Status:     "interrupted",
			Progress:   50,
		},
		workflow: &database.Workflow{
			ID:      wfID,
			Name:    "Test Workflow",
			Version: 1,
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "navigate"},
					map[string]any{"id": "2", "type": "click"},
					map[string]any{"id": "3", "type": "click"},
				},
			},
		},
		lastStepIndex: 1,
		completedSteps: []*database.ExecutionStep{
			{
				StepIndex: 0,
				Output: database.JSONMap{
					"stored_as": "result_0",
					"value":     "test_value",
				},
			},
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	svc := NewWorkflowServiceWithDeps(repo, nil, log, WorkflowServiceOptions{})

	newExec, err := svc.ResumeExecution(context.Background(), execID, map[string]any{"extra_param": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newExec == nil {
		t.Fatal("expected new execution to be created")
	}

	if newExec.TriggerType != "resume" {
		t.Errorf("expected trigger type 'resume', got %s", newExec.TriggerType)
	}

	if newExec.WorkflowID != wfID {
		t.Errorf("expected workflow id %s, got %s", wfID, newExec.WorkflowID)
	}

	// Verify trigger metadata
	triggerMeta := newExec.TriggerMetadata
	if triggerMeta == nil {
		t.Fatal("expected trigger metadata to be set")
	}
	if triggerMeta["resumed_from_execution_id"] != execID.String() {
		t.Errorf("expected resumed_from_execution_id %s, got %v", execID, triggerMeta["resumed_from_execution_id"])
	}
	if stepIndex, ok := triggerMeta["resume_from_step_index"].(int); !ok || stepIndex != 1 {
		t.Errorf("expected resume_from_step_index 1, got %v", triggerMeta["resume_from_step_index"])
	}

	// Verify parameters include both restored variables and new parameters
	params := newExec.Parameters
	if params == nil {
		t.Fatal("expected parameters to be set")
	}
	if params["result_0"] != "test_value" {
		t.Errorf("expected result_0 to be restored, got %v", params["result_0"])
	}
	if params["extra_param"] != "value" {
		t.Errorf("expected extra_param to be merged, got %v", params["extra_param"])
	}
}

func TestResumeExecution_FailsForNonResumableExecution(t *testing.T) {
	repo := &resumeRepositoryStub{
		getError: database.ErrNotFound,
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	svc := NewWorkflowServiceWithDeps(repo, nil, log, WorkflowServiceOptions{})

	_, err := svc.ResumeExecution(context.Background(), uuid.New(), nil)

	if err == nil {
		t.Fatal("expected error for non-resumable execution")
	}
	if !strings.Contains(err.Error(), "cannot be resumed") {
		t.Errorf("expected error to mention 'cannot be resumed', got %v", err)
	}
}

func TestResumeExecution_FailsWhenWorkflowNotFound(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()

	repo := &resumeRepositoryStub{
		execution: &database.Execution{
			ID:         execID,
			WorkflowID: wfID,
			Status:     "interrupted",
		},
		workflow:      nil, // Workflow not found
		lastStepIndex: 1,
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	svc := NewWorkflowServiceWithDeps(repo, nil, log, WorkflowServiceOptions{})

	_, err := svc.ResumeExecution(context.Background(), execID, nil)

	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !strings.Contains(err.Error(), "workflow not found") {
		t.Errorf("expected error to mention 'workflow not found', got %v", err)
	}
}

func TestExtractVariablesFromCompletedSteps(t *testing.T) {
	svc := &WorkflowService{
		repo: &resumeRepositoryStub{
			completedSteps: []*database.ExecutionStep{
				{
					StepIndex: 0,
					Output: database.JSONMap{
						"stored_as": "url",
						"value":     "https://example.com",
					},
				},
				{
					StepIndex: 1,
					Output: database.JSONMap{
						"extracted_data": "Some text content",
					},
					Metadata: database.JSONMap{
						"store_result": "page_content",
					},
				},
				{
					StepIndex: 2,
					Output:    database.JSONMap{}, // No stored result
				},
			},
		},
	}

	vars, err := svc.extractVariablesFromCompletedSteps(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["url"] != "https://example.com" {
		t.Errorf("expected url to be restored, got %v", vars["url"])
	}
	if vars["page_content"] != "Some text content" {
		t.Errorf("expected page_content to be restored, got %v", vars["page_content"])
	}
}
