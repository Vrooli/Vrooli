package execution

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"test-genie/internal/orchestrator"
)

// ExecutionHistory exposes read-only execution data without leaking repository details to callers.
type ExecutionHistory interface {
	List(ctx context.Context, scenario string, limit int, offset int) ([]orchestrator.SuiteExecutionResult, error)
	Get(ctx context.Context, id uuid.UUID) (*orchestrator.SuiteExecutionResult, error)
	Latest(ctx context.Context) (*orchestrator.SuiteExecutionResult, error)
}

type executionRecordStore interface {
	ListRecent(ctx context.Context, scenario string, limit int, offset int) ([]SuiteExecutionRecord, error)
	GetByID(ctx context.Context, id uuid.UUID) (*SuiteExecutionRecord, error)
	Latest(ctx context.Context) (*SuiteExecutionRecord, error)
}

// ExecutionHistoryService converts repository records into orchestrator-facing payloads.
type ExecutionHistoryService struct {
	repo executionRecordStore
}

func NewExecutionHistoryService(repo executionRecordStore) *ExecutionHistoryService {
	return &ExecutionHistoryService{repo: repo}
}

// List returns paginated execution results for an optional scenario filter.
func (s *ExecutionHistoryService) List(ctx context.Context, scenario string, limit int, offset int) ([]orchestrator.SuiteExecutionResult, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	records, err := s.repo.ListRecent(ctx, scenario, limit, offset)
	if err != nil {
		return nil, err
	}
	results := make([]orchestrator.SuiteExecutionResult, 0, len(records))
	for i := range records {
		if result := records[i].ToExecutionResult(); result != nil {
			results = append(results, *result)
		}
	}
	return results, nil
}

// Get loads a single execution result by identifier.
func (s *ExecutionHistoryService) Get(ctx context.Context, id uuid.UUID) (*orchestrator.SuiteExecutionResult, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return record.ToExecutionResult(), nil
}

// Latest returns the most recently completed execution result.
func (s *ExecutionHistoryService) Latest(ctx context.Context) (*orchestrator.SuiteExecutionResult, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	record, err := s.repo.Latest(ctx)
	if err != nil || record == nil {
		return nil, err
	}
	return record.ToExecutionResult(), nil
}

var _ ExecutionHistory = (*ExecutionHistoryService)(nil)
