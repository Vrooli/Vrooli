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
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// AutomationRunner provides an interface for running ephemeral automation sequences.
// This abstraction enables testing AI helper endpoints without requiring a real browser.
type AutomationRunner interface {
	Run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error)
}

// deviceScaleAutomationRunner is an optional extension used by screenshot
// callers that need to control browser pixel density without changing the
// shared runner contract used by other AI helpers.
type deviceScaleAutomationRunner interface {
	RunWithDeviceScale(ctx context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error)
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

func (r *inMemoryRecorder) RecordExecutionArtifacts(_ context.Context, _ autocontracts.ExecutionPlan, _ []executionwriter.ExternalArtifact) error {
	return nil // In-memory recorder does not persist artifacts
}

func (r *inMemoryRecorder) SetArtifactConfig(_ *config.ArtifactCollectionSettings) {
	// In-memory recorder ignores artifact config - collects everything
}

func (r *inMemoryRecorder) GetArtifactConfig() config.ArtifactCollectionSettings {
	return config.DefaultArtifactSettings() // In-memory recorder uses default (collect all)
}

func (r *inMemoryRecorder) SetArtifactConfigForExecution(_ uuid.UUID, _ *config.ArtifactCollectionSettings) {
	// In-memory recorder ignores per-execution artifact config - collects everything.
}

func (r *inMemoryRecorder) ForgetExecution(_ uuid.UUID) {}

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
	return r.RunWithDeviceScale(ctx, viewportWidth, viewportHeight, 0, instructions)
}

// RunWithDeviceScale executes an ephemeral run with an optional browser device
// scale factor. The profile reaches the Playwright context builder unchanged.
func (r *DefaultAutomationRunner) RunWithDeviceScale(ctx context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
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
	var browserProfile *sessionprofile.BrowserProfile
	if deviceScaleFactor != 0 {
		browserProfile = &sessionprofile.BrowserProfile{Fingerprint: &sessionprofile.FingerprintSettings{DeviceScaleFactor: deviceScaleFactor}}
	}

	engineName := autoengine.FromEnv().Resolve(r.defaultEngine)
	eventSink := autoevents.NewMemorySink(autocontracts.DefaultEventBufferLimits)
	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     r.engineFactory,
		Recorder:          recorder,
		EventSink:         eventSink,
		BrowserProfile:    browserProfile,
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
	ViewportWidth     int
	ViewportHeight    int
	DeviceScaleFactor float64
	Instructions      []autocontracts.CompiledInstruction
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
func (m *MockAutomationRunner) Run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	return m.run(ctx, viewportWidth, viewportHeight, 0, instructions)
}

func (m *MockAutomationRunner) run(_ context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	m.RunCalls = append(m.RunCalls, MockRunCall{
		ViewportWidth:     viewportWidth,
		ViewportHeight:    viewportHeight,
		DeviceScaleFactor: deviceScaleFactor,
		Instructions:      instructions,
	})

	if m.Err != nil {
		return nil, nil, m.Err
	}
	return m.Outcomes, m.Events, nil
}

// RunWithDeviceScale lets screenshot tests assert the same call path as the
// production runner while preserving the compact mock call record.
func (m *MockAutomationRunner) RunWithDeviceScale(ctx context.Context, viewportWidth, viewportHeight int, deviceScaleFactor float64, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	return m.run(ctx, viewportWidth, viewportHeight, deviceScaleFactor, instructions)
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
