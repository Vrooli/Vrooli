package suite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrSuiteRequestNotFound = errors.New("suite request not found")

type suiteExecutionEngine interface {
	Execute(ctx context.Context, req SuiteExecutionRequest) (*SuiteExecutionResult, error)
}

// SuiteExecutionInput encapsulates the orchestration request plus optional linkage to a queued suite.
type SuiteExecutionInput struct {
	Request        SuiteExecutionRequest
	SuiteRequestID *uuid.UUID
}

// SuiteExecutionService coordinates the orchestrator, queue state transitions, and execution persistence.
type SuiteExecutionService struct {
	engine        suiteExecutionEngine
	executions    *SuiteExecutionRepository
	suiteRequests *SuiteRequestService
}

func NewSuiteExecutionService(engine suiteExecutionEngine, executions *SuiteExecutionRepository, suiteRequests *SuiteRequestService) *SuiteExecutionService {
	return &SuiteExecutionService{
		engine:        engine,
		executions:    executions,
		suiteRequests: suiteRequests,
	}
}

// Execute runs the suite, persists the result, and keeps queue state in sync.
func (s *SuiteExecutionService) Execute(ctx context.Context, input SuiteExecutionInput) (*SuiteExecutionResult, error) {
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
		Phases:         append([]PhaseExecutionResult(nil), result.Phases...),
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
		return NewValidationError("suiteRequestId does not match scenarioName")
	}
	if err := s.suiteRequests.UpdateStatus(ctx, suiteID, suiteStatusRunning); err != nil {
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
	status := suiteStatusFailed
	if success {
		status = suiteStatusCompleted
	}
	return s.suiteRequests.UpdateStatus(ctx, suiteID, status)
}

func (s *SuiteExecutionService) markSuiteFailed(ctx context.Context, suiteID *uuid.UUID) {
	if suiteID == nil || s.suiteRequests == nil {
		return
	}
	_ = s.suiteRequests.UpdateStatus(ctx, *suiteID, suiteStatusFailed)
}
