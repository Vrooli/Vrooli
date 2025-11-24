package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
)

// SimpleExecutor provides a minimal sequential executor that delegates step
// execution to an AutomationEngine and persists outcomes via a Recorder. It
// keeps telemetry/event emission contained so we can grow into the new
// architecture without touching handlers yet.
type SimpleExecutor struct {
	sequencer    events.Sequencer
	telemetrySeq map[string]uint64
	telemetryMu  sync.Mutex
}

// NewSimpleExecutor constructs a SimpleExecutor. If no sequencer is supplied,
// a per-execution sequencer is used.
func NewSimpleExecutor(seq events.Sequencer) *SimpleExecutor {
	if seq == nil {
		seq = events.NewPerExecutionSequencer()
	}
	return &SimpleExecutor{
		sequencer:    seq,
		telemetrySeq: make(map[string]uint64),
	}
}

// Execute runs the plan sequentially. Retries/branching will be layered on
// later; this is the minimal happy path required to exercise the abstraction.
func (e *SimpleExecutor) Execute(ctx context.Context, req Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.validateRequest(req); err != nil {
		return err
	}

	engineName := req.EngineName
	if engineName == "" {
		engineName = "browserless"
	}

	eng, err := req.EngineFactory.Resolve(ctx, engineName)
	if err != nil {
		return fmt.Errorf("resolve engine %q: %w", engineName, err)
	}

	// Capabilities are fetched up front to allow callers to validate or log.
	caps, err := eng.Capabilities(ctx)
	if err != nil {
		return fmt.Errorf("engine capabilities: %w", err)
	}
	if gap := caps.CheckCompatibility(deriveRequirements(req.Plan)); !gap.Satisfied() {
		return &CapabilityError{
			Engine:    eng.Name(),
			Missing:   gap.Missing,
			Warnings:  gap.Warnings,
			Execution: req.Plan.ExecutionID,
			Workflow:  req.Plan.WorkflowID,
		}
	}

	session, err := eng.StartSession(ctx, engine.SessionSpec{
		ExecutionID: req.Plan.ExecutionID,
		WorkflowID:  req.Plan.WorkflowID,
		ReuseMode:   engine.ReuseModeReuse,
		Labels:      map[string]string{},
		Capabilities: contracts.CapabilityRequirement{
			NeedsParallelTabs: false,
		},
	})
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	defer session.Close(context.Background()) // Best-effort cleanup.

	seedVars := map[string]any{}
	if metaVars, ok := req.Plan.Metadata["variables"].(map[string]any); ok {
		for k, v := range metaVars {
			seedVars[k] = v
		}
	}
	state := newFlowState(seedVars)

	if req.Plan.Graph != nil && len(req.Plan.Graph.Steps) > 0 {
		return e.executeGraph(ctx, req, session, state)
	}

	for idx := range req.Plan.Instructions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		instruction := req.Plan.Instructions[idx]
		instruction = e.interpolateInstruction(instruction, state)

		stepCtx, cancel := context.WithCancel(ctx)
		startedAt := time.Now().UTC()
		attempt := 1

		e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &instruction.Index, &attempt, map[string]any{
			"node_id":   instruction.NodeID,
			"step_type": instruction.Type,
		})

		stopHeartbeat := e.startHeartbeat(stepCtx, req, instruction.Index, attempt, startedAt)

		outcome, runErr := e.runWithRetries(stepCtx, req, session, instruction)
		if stopHeartbeat != nil {
			stopHeartbeat()
		}
		cancel()

		normalized := e.normalizeOutcome(req.Plan, instruction, attempt, startedAt, outcome, runErr)

		recordResult, recordErr := req.Recorder.RecordStepOutcome(ctx, req.Plan, normalized)
		if recordErr != nil {
			return fmt.Errorf("record step outcome: %w", recordErr)
		}

		payload := map[string]any{
			"outcome":   normalized,
			"artifacts": recordResult.ArtifactIDs,
		}
		if recordResult.TimelineArtifactID != nil {
			payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
		}

		eventKind := contracts.EventKindStepCompleted
		if !normalized.Success {
			eventKind = contracts.EventKindStepFailed
		}
		e.emitEvent(ctx, req, eventKind, &instruction.Index, &attempt, payload)

		if runErr != nil {
			return runErr
		}
		if !normalized.Success {
			return fmt.Errorf("step %d failed: %s", normalized.StepIndex, e.failureMessage(normalized))
		}
	}

	return nil
}

func (e *SimpleExecutor) validateRequest(req Request) error {
	switch {
	case req.EngineFactory == nil:
		return errors.New("engine factory is required")
	case req.Recorder == nil:
		return errors.New("recorder is required")
	case req.EventSink == nil:
		return errors.New("event sink is required")
	case req.Plan.ExecutionID == uuid.Nil:
		return errors.New("plan execution_id is required")
	case req.Plan.WorkflowID == uuid.Nil:
		return errors.New("plan workflow_id is required")
	}
	return nil
}

func (e *SimpleExecutor) normalizeOutcome(plan contracts.ExecutionPlan, instruction contracts.CompiledInstruction, attempt int, startedAt time.Time, outcome contracts.StepOutcome, runErr error) contracts.StepOutcome {
	now := time.Now().UTC()
	outcome.SchemaVersion = valueOrDefault(outcome.SchemaVersion, contracts.StepOutcomeSchemaVersion)
	outcome.PayloadVersion = valueOrDefault(outcome.PayloadVersion, contracts.PayloadVersion)
	outcome.ExecutionID = uuidOrDefault(outcome.ExecutionID, plan.ExecutionID)
	outcome.StepIndex = intOrDefault(outcome.StepIndex, instruction.Index)
	outcome.NodeID = valueOrDefault(outcome.NodeID, instruction.NodeID)
	outcome.StepType = valueOrDefault(outcome.StepType, instruction.Type)
	outcome.Attempt = intOrDefault(outcome.Attempt, attempt)

	if outcome.StartedAt.IsZero() {
		outcome.StartedAt = startedAt
	}
	if outcome.CompletedAt == nil {
		t := now
		outcome.CompletedAt = &t
	}
	if outcome.DurationMs == 0 && !outcome.StartedAt.IsZero() && outcome.CompletedAt != nil {
		outcome.DurationMs = int(outcome.CompletedAt.Sub(outcome.StartedAt).Milliseconds())
	}

	if runErr != nil {
		outcome.Success = false
		if outcome.Failure == nil {
			kind := contracts.FailureKindEngine
			retryable := true
			switch {
			case errors.Is(runErr, context.DeadlineExceeded):
				kind = contracts.FailureKindTimeout
				retryable = false
			case errors.Is(runErr, context.Canceled):
				kind = contracts.FailureKindCancelled
				retryable = false
			}
			outcome.Failure = &contracts.StepFailure{
				Kind:      kind,
				Message:   runErr.Error(),
				Retryable: retryable,
				Source:    contracts.FailureSourceEngine,
				OccurredAt: func() *time.Time {
					t := now
					return &t
				}(),
			}
		}
	}

	if !outcome.Success && outcome.Failure == nil {
		outcome.Failure = &contracts.StepFailure{
			Kind:      contracts.FailureKindEngine,
			Message:   "step failed without failure metadata",
			Retryable: false,
			Source:    contracts.FailureSourceEngine,
			OccurredAt: func() *time.Time {
				t := now
				return &t
			}(),
		}
	}

	return outcome
}

func (e *SimpleExecutor) failureMessage(outcome contracts.StepOutcome) string {
	if outcome.Failure != nil && outcome.Failure.Message != "" {
		return outcome.Failure.Message
	}
	return "unknown failure"
}

func (e *SimpleExecutor) emitEvent(ctx context.Context, req Request, kind contracts.EventKind, stepIndex *int, attempt *int, payload any) {
	if req.EventSink == nil {
		return
	}
	ev := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           kind,
		ExecutionID:    req.Plan.ExecutionID,
		WorkflowID:     req.Plan.WorkflowID,
		StepIndex:      stepIndex,
		Attempt:        attempt,
		Sequence:       e.sequencer.Next(req.Plan.ExecutionID),
		Timestamp:      time.Now().UTC(),
		Payload:        payload,
	}
	_ = req.EventSink.Publish(ctx, ev) // Sink decides drop policy; executor is fire-and-forget.
}

func (e *SimpleExecutor) emitTelemetry(ctx context.Context, req Request, telemetry contracts.StepTelemetry) {
	if req.EventSink == nil {
		return
	}
	if telemetry.Sequence == 0 {
		telemetry.Sequence = e.nextTelemetrySequence(req.Plan.ExecutionID, telemetry.StepIndex, telemetry.Attempt)
	}
	ev := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepTelemetry,
		ExecutionID:    req.Plan.ExecutionID,
		WorkflowID:     req.Plan.WorkflowID,
		StepIndex:      &telemetry.StepIndex,
		Attempt:        &telemetry.Attempt,
		Sequence:       e.sequencer.Next(req.Plan.ExecutionID),
		Timestamp:      telemetry.Timestamp,
		Payload:        telemetry,
	}
	_ = req.EventSink.Publish(ctx, ev)
}

func (e *SimpleExecutor) startHeartbeat(ctx context.Context, req Request, stepIndex int, attempt int, started time.Time) func() {
	if req.HeartbeatInterval <= 0 {
		return nil
	}
	ticker := time.NewTicker(req.HeartbeatInterval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-done:
				ticker.Stop()
				return
			case t := <-ticker.C:
				elapsed := t.Sub(started).Milliseconds()
				telemetry := contracts.StepTelemetry{
					SchemaVersion:  contracts.TelemetrySchemaVersion,
					PayloadVersion: contracts.PayloadVersion,
					ExecutionID:    req.Plan.ExecutionID,
					StepIndex:      stepIndex,
					Attempt:        attempt,
					Kind:           contracts.TelemetryKindHeartbeat,
					Sequence:       0, // Sequence within attempt can be layered later; envelope carries ordering.
					Timestamp:      t.UTC(),
					ElapsedMs:      elapsed,
					Heartbeat: &contracts.HeartbeatTelemetry{
						Progress: 0,
						Message:  "heartbeat",
					},
				}
				e.emitTelemetry(ctx, req, telemetry)
			}
		}
	}()

	return func() { close(done) }
}

func (e *SimpleExecutor) telemetryKey(executionID uuid.UUID, stepIndex, attempt int) string {
	return fmt.Sprintf("%s:%d:%d", executionID.String(), stepIndex, attempt)
}

func (e *SimpleExecutor) nextTelemetrySequence(executionID uuid.UUID, stepIndex, attempt int) uint64 {
	e.telemetryMu.Lock()
	defer e.telemetryMu.Unlock()

	if e.telemetrySeq == nil {
		e.telemetrySeq = make(map[string]uint64)
	}
	key := e.telemetryKey(executionID, stepIndex, attempt)
	e.telemetrySeq[key]++
	return e.telemetrySeq[key]
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func intOrDefault(value, fallback int) int {
	if value != 0 {
		return value
	}
	return fallback
}

func (e *SimpleExecutor) runWithRetries(ctx context.Context, req Request, session engine.EngineSession, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	cfg := retryConfigFromInstruction(instruction)
	var lastOutcome contracts.StepOutcome
	var lastErr error
	delay := cfg.Delay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attemptStart := time.Now().UTC()
		outcome, err := session.Run(ctx, instruction)
		outcome.Attempt = attempt
		outcome = e.normalizeOutcome(req.Plan, instruction, attempt, attemptStart, outcome, err)

		if req.Recorder != nil && outcome.Failure != nil {
			_ = req.Recorder.RecordTelemetry(ctx, req.Plan, contracts.StepTelemetry{
				SchemaVersion:  contracts.TelemetrySchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				ExecutionID:    req.Plan.ExecutionID,
				StepIndex:      instruction.Index,
				Attempt:        attempt,
				Kind:           contracts.TelemetryKindRetry,
				Sequence:       e.nextTelemetrySequence(req.Plan.ExecutionID, instruction.Index, attempt),
				Timestamp:      time.Now().UTC(),
				Retry: &contracts.RetryTelemetry{
					Attempt:     attempt,
					MaxAttempts: cfg.MaxAttempts,
					Reason:      outcome.Failure,
				},
			})
		}

		lastOutcome, lastErr = outcome, err
		if err == nil && outcome.Success {
			return outcome, nil
		}
		if attempt >= cfg.MaxAttempts || (outcome.Failure != nil && !outcome.Failure.Retryable) {
			return outcome, err
		}

		select {
		case <-ctx.Done():
			return outcome, ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * cfg.BackoffFactor)
	}

	return lastOutcome, lastErr
}

type retryConfig struct {
	MaxAttempts   int
	Delay         time.Duration
	BackoffFactor float64
}

func retryConfigFromInstruction(instruction contracts.CompiledInstruction) retryConfig {
	cfg := retryConfig{
		MaxAttempts:   1,
		Delay:         750 * time.Millisecond,
		BackoffFactor: 1.5,
	}
	if instruction.Params == nil {
		return cfg
	}
	if v, ok := instruction.Params["retryAttempts"].(float64); ok && v > 0 {
		cfg.MaxAttempts = int(v)
	}
	if v, ok := instruction.Params["retryDelayMs"].(float64); ok && v > 0 {
		cfg.Delay = time.Duration(v) * time.Millisecond
	}
	if v, ok := instruction.Params["retryBackoffFactor"].(float64); ok && v > 0 {
		cfg.BackoffFactor = v
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.BackoffFactor < 1 {
		cfg.BackoffFactor = 1
	}
	return cfg
}
