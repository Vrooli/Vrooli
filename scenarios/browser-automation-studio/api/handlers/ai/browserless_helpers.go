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
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
)

// inMemoryRecorder captures outcomes/telemetry without touching the database so AI
// helper endpoints can reuse the automation stack without polluting execution tables.
type inMemoryRecorder struct {
	mu        sync.Mutex
	outcomes  []autocontracts.StepOutcome
	telemetry []autocontracts.StepTelemetry
}

func (r *inMemoryRecorder) RecordStepOutcome(_ context.Context, _ autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (autorecorder.RecordResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outcomes = append(r.outcomes, outcome)
	return autorecorder.RecordResult{}, nil
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

func (r *inMemoryRecorder) Outcomes() []autocontracts.StepOutcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]autocontracts.StepOutcome, len(r.outcomes))
	copy(out, r.outcomes)
	return out
}

// automationRunner wires the engine/executor stack for ephemeral, in-process
// automation runs used by AI helper endpoints.
type automationRunner struct {
	executor      autoexecutor.Executor
	engineFactory autoengine.Factory
	log           *logrus.Logger
	defaultEngine string
}

func newAutomationRunner(log *logrus.Logger) (*automationRunner, error) {
	selection := autoengine.FromEnv()
	factory, err := autoengine.DefaultFactory(log)
	if err != nil {
		return nil, err
	}
	engineName := selection.Resolve("")
	if engineName == "" {
		engineName = "browserless"
	}
	return &automationRunner{
		executor:      autoexecutor.NewSimpleExecutor(nil),
		engineFactory: factory,
		log:           log,
		defaultEngine: engineName,
	}, nil
}

func (r *automationRunner) run(ctx context.Context, viewportWidth, viewportHeight int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
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
