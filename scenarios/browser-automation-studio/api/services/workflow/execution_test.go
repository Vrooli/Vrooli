package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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
