package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	engine           suiteExecutionEngine
	executions       suiteExecutionRecorder
	collectRetention func(context.Context, string)
}

func NewSuiteExecutionService(engine suiteExecutionEngine, executions suiteExecutionRecorder) *SuiteExecutionService {
	return &SuiteExecutionService{engine: engine, executions: executions}
}

// SetRetentionCollector wires the post-persistence lifecycle trigger. The
// callback is injected by bootstrap so execution remains storage-agnostic;
// retention runs only after the SQLite row and durable run evidence agree on a
// completed run.
func (s *SuiteExecutionService) SetRetentionCollector(collect func(context.Context, string)) {
	if s != nil {
		s.collectRetention = collect
	}
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
		ID:                       uuid.New(),
		RunID:                    result.RunID,
		ScenarioName:             result.ScenarioName,
		TargetKind:               result.TargetKind,
		TargetID:                 result.TargetID,
		PresetUsed:               result.PresetUsed,
		RequestedPreset:          result.RequestedPreset,
		RequestedPhases:          append([]string(nil), result.RequestedPhases...),
		RequestedSkipPhases:      append([]string(nil), result.RequestedSkipPhases...),
		PlannedPhases:            append([]string(nil), result.PlannedPhases...),
		PhaseSetDigest:           result.PhaseSetDigest,
		DescriptorSnapshotDigest: result.DescriptorSnapshotDigest,
		ConfigurationFingerprint: result.ConfigurationFingerprint,
		FailFast:                 result.FailFast,
		Success:                  result.Success,
		Phases:                   compactPhaseResults(result.Phases),
		PreparationStages:        compactPreparationStages(result.PreparationStages),
		StartedAt:                result.StartedAt,
		CompletedAt:              result.CompletedAt,
	}

	if err := s.executions.Create(ctx, record); err != nil {
		return nil, err
	}
	if s.collectRetention != nil && result.ScenarioName != "" {
		go s.collectRetention(context.Background(), result.ScenarioName)
	}

	result.ExecutionID = record.ID
	return result, nil
}

// compactPreparationStages keeps historical execution timing bounded without
// mixing orchestration spans into the phase-history projection.
func compactPreparationStages(stages []orchestrator.PreparationStage) []orchestrator.PreparationStage {
	if len(stages) == 0 {
		return []orchestrator.PreparationStage{}
	}
	out := make([]orchestrator.PreparationStage, 0, len(stages))
	for _, stage := range stages {
		if stage.Name == "" {
			continue
		}
		stage.DurationMilliseconds = maxInt64(0, stage.DurationMilliseconds)
		out = append(out, stage)
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// compactPhaseResults is the execution-history persistence boundary. Detailed
// observations, normalized findings, provider metrics, and log payloads belong
// to immutable run evidence; SQLite retains only the compact history projection
// needed for list, timing, reliability, and phase standing queries.
func compactPhaseResults(results []phases.ExecutionResult) []phases.ExecutionResult {
	if len(results) == 0 {
		return []phases.ExecutionResult{}
	}
	compact := make([]phases.ExecutionResult, 0, len(results))
	for _, result := range results {
		compact = append(compact, phases.ExecutionResult{
			Name:               result.Name,
			Status:             result.Status,
			DurationSeconds:    result.DurationSeconds,
			Error:              result.Error,
			Classification:     result.Classification,
			Remediation:        result.Remediation,
			RunnabilityVerdict: result.RunnabilityVerdict,
			RunnabilityReason:  result.RunnabilityReason,
			FindingSource:      result.FindingSource,
			PhasePresentation:  result.PhasePresentation,
			FindingsSummary:    result.FindingsSummary,
		})
	}
	return compact
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
		TargetKind:      requestTargetKind(input.Request),
		TargetID:        requestTargetID(input.Request),
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

func requestTargetKind(req orchestrator.SuiteExecutionRequest) string {
	if strings.TrimSpace(req.Target) == "" {
		return "scenario"
	}
	if kind, _, ok := strings.Cut(req.Target, ":"); ok && strings.TrimSpace(kind) != "" {
		return strings.TrimSpace(kind)
	}
	return "scenario"
}

func requestTargetID(req orchestrator.SuiteExecutionRequest) string {
	if kind, id, ok := strings.Cut(strings.TrimSpace(req.Target), ":"); ok && strings.TrimSpace(kind) != "" && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return strings.TrimSpace(req.ScenarioName)
}
