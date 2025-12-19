package ai

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/config"
)

// AutomationRunner provides an interface for running ephemeral automation sequences.
// This abstraction enables testing AI helper endpoints without requiring a real browser.
type AutomationRunner interface {
	Run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error)
}

// inMemoryRecorder captures outcomes/telemetry without touching the database so AI
// helper endpoints can reuse the automation stack without polluting execution tables.
type inMemoryRecorder struct {
	mu        sync.Mutex
	outcomes  []autocontracts.StepOutcome
	telemetry []autocontracts.StepTelemetry
}

func (r *inMemoryRecorder) RecordStepOutcome(_ context.Context, _ autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (executionwriter.RecordResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outcomes = append(r.outcomes, outcome)
	return executionwriter.RecordResult{}, nil
}

func (r *inMemoryRecorder) RecordTelemetry(_ context.Context, _ autocontracts.ExecutionPlan, telemetry autocontracts.StepTelemetry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.telemetry = append(r.telemetry, telemetry)
	return nil
}

func (r *inMemoryRecorder) MarkCrash(_ context.Context, _ uuid.UUID, _ autocontracts.StepFailure) error {
	return nil
}

func (r *inMemoryRecorder) UpdateCheckpoint(_ context.Context, _ uuid.UUID, _ int, _ int) error {
	return nil // In-memory recorder doesn't persist checkpoints
}

func (r *inMemoryRecorder) SetArtifactConfig(_ *config.ArtifactCollectionSettings) {
	// In-memory recorder ignores artifact config - collects everything
}

func (r *inMemoryRecorder) GetArtifactConfig() config.ArtifactCollectionSettings {
	return config.DefaultArtifactSettings() // In-memory recorder uses default (collect all)
}

func (r *inMemoryRecorder) Outcomes() []autocontracts.StepOutcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]autocontracts.StepOutcome, len(r.outcomes))
	copy(out, r.outcomes)
	return out
}

// DefaultAutomationRunner wires the engine/executor stack for ephemeral, in-process
// automation runs used by AI helper endpoints.
type DefaultAutomationRunner struct {
	executor      autoexecutor.Executor
	engineFactory autoengine.Factory
	log           *logrus.Logger
	defaultEngine string
}

// AutomationRunnerOption configures the DefaultAutomationRunner.
type AutomationRunnerOption func(*DefaultAutomationRunner)

// WithEngineFactory sets a custom engine factory.
func WithEngineFactory(factory autoengine.Factory) AutomationRunnerOption {
	return func(r *DefaultAutomationRunner) {
		r.engineFactory = factory
	}
}

// WithExecutor sets a custom executor.
func WithExecutor(executor autoexecutor.Executor) AutomationRunnerOption {
	return func(r *DefaultAutomationRunner) {
		r.executor = executor
	}
}

// NewDefaultAutomationRunner creates a DefaultAutomationRunner with optional configuration.
func NewDefaultAutomationRunner(log *logrus.Logger, opts ...AutomationRunnerOption) (*DefaultAutomationRunner, error) {
	runner := &DefaultAutomationRunner{
		executor:      autoexecutor.NewSimpleExecutor(nil),
		log:           log,
		defaultEngine: "playwright",
	}

	// Apply options first to allow factory injection
	for _, opt := range opts {
		opt(runner)
	}

	// Only create default factory if not provided via options
	if runner.engineFactory == nil {
		factory, err := autoengine.DefaultFactory(log)
		if err != nil {
			return nil, err
		}
		runner.engineFactory = factory
	}

	return runner, nil
}

// newAutomationRunner is a convenience wrapper for backward compatibility.
func newAutomationRunner(log *logrus.Logger) (*DefaultAutomationRunner, error) {
	return NewDefaultAutomationRunner(log)
}

// Run executes a sequence of automation instructions and returns outcomes.
func (r *DefaultAutomationRunner) Run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	recorder := &inMemoryRecorder{}
	plan := autocontracts.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions:   instructions,
		Metadata: map[string]any{
			"executionViewport": map[string]any{
				"width":  viewportWidth,
				"height": viewportHeight,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	engineName := autoengine.FromEnv().Resolve(r.defaultEngine)
	eventSink := autoevents.NewMemorySink(autocontracts.DefaultEventBufferLimits)
	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     r.engineFactory,
		Recorder:          recorder,
		EventSink:         eventSink,
		HeartbeatInterval: 0,
	}

	if err := r.executor.Execute(ctx, req); err != nil {
		return recorder.Outcomes(), nil, err
	}

	events := eventSink.Events()

	return recorder.Outcomes(), events, nil
}

// MockAutomationRunner is a test double for AutomationRunner.
type MockAutomationRunner struct {
	Outcomes []autocontracts.StepOutcome
	Events   []autocontracts.EventEnvelope
	Err      error
	RunCalls []MockRunCall
}

// MockRunCall records a call to Run.
type MockRunCall struct {
	ViewportWidth  int
	ViewportHeight int
	Instructions   []autocontracts.CompiledInstruction
}

// NewMockAutomationRunner creates a MockAutomationRunner with default successful outcomes.
func NewMockAutomationRunner() *MockAutomationRunner {
	return &MockAutomationRunner{
		Outcomes: []autocontracts.StepOutcome{
			{
				Success:   true,
				StepIndex: 0,
				NodeID:    "mock-step",
				StepType:  "navigate",
			},
		},
	}
}

// Run records the call and returns configured outcomes or error.
func (m *MockAutomationRunner) Run(_ context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	m.RunCalls = append(m.RunCalls, MockRunCall{
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
		Instructions:   instructions,
	})

	if m.Err != nil {
		return nil, nil, m.Err
	}
	return m.Outcomes, m.Events, nil
}

// Reset clears recorded calls for reuse between tests.
func (m *MockAutomationRunner) Reset() {
	m.RunCalls = nil
}

// Compile-time interface enforcement
var (
	_ AutomationRunner = (*DefaultAutomationRunner)(nil)
	_ AutomationRunner = (*MockAutomationRunner)(nil)
)
