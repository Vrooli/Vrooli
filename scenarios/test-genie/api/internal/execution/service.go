package execution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"

	"github.com/google/uuid"
)

type suiteExecutionEngine interface {
	ExecuteWithEvents(ctx context.Context, req orchestrator.SuiteExecutionRequest, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error)
}

type suiteExecutionRecorder interface {
	Create(ctx context.Context, record *SuiteExecutionRecord) error
}

// SuiteExecutionInput encapsulates a server-owned orchestration request.
type SuiteExecutionInput struct {
	Request orchestrator.SuiteExecutionRequest
}

// SuiteExecutionService coordinates the orchestrator and execution persistence.
type SuiteExecutionService struct {
	engine     suiteExecutionEngine
	executions suiteExecutionRecorder
}

func NewSuiteExecutionService(engine suiteExecutionEngine, executions suiteExecutionRecorder) *SuiteExecutionService {
	return &SuiteExecutionService{engine: engine, executions: executions}
}

// Execute runs the suite and persists the result.
func (s *SuiteExecutionService) Execute(ctx context.Context, input SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error) {
	return s.run(ctx, input, nil)
}

// ExecuteWithEvents runs the suite with streaming events via callback.
func (s *SuiteExecutionService) ExecuteWithEvents(ctx context.Context, input SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	return s.run(ctx, input, emit)
}

// run is the single execute implementation. The orchestrator's own per-phase
// writer no-ops a nil emit, so streaming and non-streaming callers share one
// execution and persistence path.
func (s *SuiteExecutionService) run(ctx context.Context, input SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("suite execution engine is not configured")
	}
	if s.executions == nil {
		return nil, fmt.Errorf("suite execution repository is not configured")
	}

	startedAt := time.Now().UTC()
	result, err := s.engine.ExecuteWithEvents(ctx, input.Request, emit)
	if err != nil {
		s.recordTerminalOutcome(ctx, input, startedAt, classifyTerminalError(ctx, err))
		return nil, err
	}
	if result == nil {
		s.recordTerminalOutcome(ctx, input, startedAt, classifyTerminalError(ctx, nil))
		return nil, errors.New("suite execution engine returned no result")
	}

	record := &SuiteExecutionRecord{
		ID:                  uuid.New(),
		RunID:               result.RunID,
		ScenarioName:        result.ScenarioName,
		PresetUsed:          result.PresetUsed,
		RequestedPreset:     result.RequestedPreset,
		RequestedPhases:     append([]string(nil), result.RequestedPhases...),
		RequestedSkipPhases: append([]string(nil), result.RequestedSkipPhases...),
		PlannedPhases:       append([]string(nil), result.PlannedPhases...),
		FailFast:            result.FailFast,
		Success:             result.Success,
		Phases:              append([]phases.ExecutionResult(nil), result.Phases...),
		StartedAt:           result.StartedAt,
		CompletedAt:         result.CompletedAt,
	}

	if err := s.executions.Create(ctx, record); err != nil {
		return nil, err
	}

	result.ExecutionID = record.ID
	return result, nil
}

// recordTerminalOutcome persists a minimal suite_executions row for a
// catastrophic run that never produced a result (engine error, nil result,
// abort, or timeout). Without it, availability denominators silently omit these
// outcomes. Best-effort: a persistence failure here must not mask the original
// execution error, so it is logged-by-omission (the caller already returns the
// real error). The write uses a detached context because the request context
// may already be cancelled (the very condition we are recording).
func (s *SuiteExecutionService) recordTerminalOutcome(ctx context.Context, input SuiteExecutionInput, startedAt time.Time, outcome TerminalOutcome) {
	if s.executions == nil {
		return
	}
	writeCtx := context.WithoutCancel(ctx)
	record := &SuiteExecutionRecord{
		ID:              uuid.New(),
		RunID:           input.Request.RunID,
		ScenarioName:    input.Request.ScenarioName,
		Success:         false,
		TerminalOutcome: outcome,
		// Empty (non-nil) so it marshals to a valid JSON "[]" for the
		// NOT NULL / json_valid(phases) column constraint.
		Phases:      []phases.ExecutionResult{},
		StartedAt:   startedAt,
		CompletedAt: time.Now().UTC(),
	}
	_ = s.executions.Create(writeCtx, record)
}
