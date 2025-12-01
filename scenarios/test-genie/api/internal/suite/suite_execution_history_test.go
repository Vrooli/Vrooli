package suite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/orchestrator/phases"
)

func TestExecutionHistoryService_List(t *testing.T) {
	now := time.Now().UTC()
	records := []SuiteExecutionRecord{
		{
			ID:           uuid.New(),
			ScenarioName: "demo",
			Success:      true,
			StartedAt:    now.Add(-time.Minute),
			CompletedAt:  now,
			Phases: []phases.ExecutionResult{
				{Name: "structure", Status: "passed", DurationSeconds: 1},
			},
		},
	}
	store := &stubExecutionRecordStore{list: records}
	history := NewExecutionHistoryService(store)

	results, err := history.List(context.Background(), "demo", 5, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ExecutionID != records[0].ID {
		t.Fatalf("unexpected execution id")
	}
	if results[0].PhaseSummary.Passed != 1 {
		t.Fatalf("phase summary not populated: %#v", results[0].PhaseSummary)
	}
}

func TestExecutionHistoryService_LatestNilRepo(t *testing.T) {
	var history *ExecutionHistoryService
	if _, err := history.Latest(context.Background()); err != sql.ErrConnDone {
		t.Fatalf("expected sql.ErrConnDone, got %v", err)
	}
}

func TestExecutionHistoryService_GetProps(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()
	suiteID := uuid.New()
	store := &stubExecutionRecordStore{
		record: &SuiteExecutionRecord{
			ID:             id,
			SuiteRequestID: &suiteID,
			ScenarioName:   "demo",
			PresetUsed:     "quick",
			Success:        false,
			StartedAt:      now.Add(-time.Minute),
			CompletedAt:    now,
		},
	}

	result, err := NewExecutionHistoryService(store).Get(context.Background(), id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.SuiteRequestID == nil || *result.SuiteRequestID != suiteID {
		t.Fatalf("expected suite request id to be preserved: %#v", result)
	}
	if result.PresetUsed != "quick" {
		t.Fatalf("expected preset to round-trip, got %s", result.PresetUsed)
	}
}

type stubExecutionRecordStore struct {
	list   []SuiteExecutionRecord
	record *SuiteExecutionRecord
	latest *SuiteExecutionRecord
}

func (s *stubExecutionRecordStore) ListRecent(ctx context.Context, scenario string, limit int, offset int) ([]SuiteExecutionRecord, error) {
	return s.list, nil
}

func (s *stubExecutionRecordStore) GetByID(ctx context.Context, id uuid.UUID) (*SuiteExecutionRecord, error) {
	return s.record, nil
}

func (s *stubExecutionRecordStore) Latest(ctx context.Context) (*SuiteExecutionRecord, error) {
	return s.latest, nil
}
