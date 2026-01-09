package execution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/queue"
	"test-genie/internal/shared"
)

// ErrSuiteRequestNotFound indicates the linked suite request does not exist.
var ErrSuiteRequestNotFound = errors.New("suite request not found")

type suiteExecutionEngine interface {
	Execute(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*orchestrator.SuiteExecutionResult, error)
	ExecuteWithEvents(ctx context.Context, req orchestrator.SuiteExecutionRequest, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error)
}

type suiteRequestManager interface {
	Get(ctx context.Context, id uuid.UUID) (*queue.SuiteRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// SuiteExecutionInput encapsulates the orchestration request plus optional linkage to a queued suite.
type SuiteExecutionInput struct {
	Request        orchestrator.SuiteExecutionRequest
	SuiteRequestID *uuid.UUID
}

// SuiteExecutionService coordinates the orchestrator, queue state transitions, and execution persistence.
type SuiteExecutionService struct {
	engine        suiteExecutionEngine
	executions    *SuiteExecutionRepository
	suiteRequests suiteRequestManager
}

func NewSuiteExecutionService(engine suiteExecutionEngine, executions *SuiteExecutionRepository, suiteRequests suiteRequestManager) *SuiteExecutionService {
	return &SuiteExecutionService{
		engine:        engine,
		executions:    executions,
		suiteRequests: suiteRequests,
	}
}

// Execute runs the suite, persists the result, and keeps queue state in sync.
func (s *SuiteExecutionService) Execute(ctx context.Context, input SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("suite execution engine is not configured")
	}
	if s.executions == nil {
		return nil, fmt.Errorf("suite execution repository is not configured")
	}

	var suiteID *uuid.UUID
	if input.SuiteRequestID != nil {
		suiteID = input.SuiteRequestID
		if err := s.loadAndMarkSuiteRequest(ctx, *suiteID, input.Request.ScenarioName); err != nil {
			return nil, err
		}
	}

	result, err := s.engine.Execute(ctx, input.Request)
	if err != nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, err
	}
	if result == nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, errors.New("suite execution engine returned no result")
	}

	record := &SuiteExecutionRecord{
		ID:             uuid.New(),
		SuiteRequestID: suiteID,
		ScenarioName:   result.ScenarioName,
		PresetUsed:     result.PresetUsed,
		Success:        result.Success,
		Phases:         append([]phases.ExecutionResult(nil), result.Phases...),
		StartedAt:      result.StartedAt,
		CompletedAt:    result.CompletedAt,
	}

	if err := s.executions.Create(ctx, record); err != nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, err
	}

	if suiteID != nil {
		if err := s.finalizeSuiteRequest(ctx, *suiteID, result.Success); err != nil {
			return nil, err
		}
		result.SuiteRequestID = suiteID
	}

	result.ExecutionID = record.ID
	return result, nil
}

// ExecuteWithEvents runs the suite with streaming events via callback.
func (s *SuiteExecutionService) ExecuteWithEvents(ctx context.Context, input SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	if s.engine == nil {
		return nil, fmt.Errorf("suite execution engine is not configured")
	}
	if s.executions == nil {
		return nil, fmt.Errorf("suite execution repository is not configured")
	}

	var suiteID *uuid.UUID
	if input.SuiteRequestID != nil {
		suiteID = input.SuiteRequestID
		if err := s.loadAndMarkSuiteRequest(ctx, *suiteID, input.Request.ScenarioName); err != nil {
			return nil, err
		}
	}

	result, err := s.engine.ExecuteWithEvents(ctx, input.Request, emit)
	if err != nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, err
	}
	if result == nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, errors.New("suite execution engine returned no result")
	}

	record := &SuiteExecutionRecord{
		ID:             uuid.New(),
		SuiteRequestID: suiteID,
		ScenarioName:   result.ScenarioName,
		PresetUsed:     result.PresetUsed,
		Success:        result.Success,
		Phases:         append([]phases.ExecutionResult(nil), result.Phases...),
		StartedAt:      result.StartedAt,
		CompletedAt:    result.CompletedAt,
	}

	if err := s.executions.Create(ctx, record); err != nil {
		s.markSuiteFailed(ctx, suiteID)
		return nil, err
	}

	if suiteID != nil {
		if err := s.finalizeSuiteRequest(ctx, *suiteID, result.Success); err != nil {
			return nil, err
		}
		result.SuiteRequestID = suiteID
	}

	result.ExecutionID = record.ID
	return result, nil
}

func (s *SuiteExecutionService) loadAndMarkSuiteRequest(ctx context.Context, suiteID uuid.UUID, scenario string) error {
	if s.suiteRequests == nil {
		return fmt.Errorf("suite request service is not configured")
	}
	req, err := s.suiteRequests.Get(ctx, suiteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSuiteRequestNotFound
		}
		return err
	}
	if !strings.EqualFold(req.ScenarioName, scenario) {
		return shared.NewValidationError("suiteRequestId does not match scenarioName")
	}
	if err := s.suiteRequests.UpdateStatus(ctx, suiteID, queue.StatusRunning); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSuiteRequestNotFound
		}
		return err
	}
	return nil
}

func (s *SuiteExecutionService) finalizeSuiteRequest(ctx context.Context, suiteID uuid.UUID, success bool) error {
	if s.suiteRequests == nil {
		return fmt.Errorf("suite request service is not configured")
	}
	status := queue.StatusFailed
	if success {
		status = queue.StatusCompleted
	}
	return s.suiteRequests.UpdateStatus(ctx, suiteID, status)
}

func (s *SuiteExecutionService) markSuiteFailed(ctx context.Context, suiteID *uuid.UUID) {
	if suiteID == nil || s.suiteRequests == nil {
		return
	}
	_ = s.suiteRequests.UpdateStatus(ctx, *suiteID, queue.StatusFailed)
}
