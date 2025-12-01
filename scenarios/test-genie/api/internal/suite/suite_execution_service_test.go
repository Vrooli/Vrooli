package suite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestSuiteExecutionService_ExecuteWithoutLinkedRequest(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO suite_executions").
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(1, 1))

	engine := &stubExecutionEngine{
		result: &SuiteExecutionResult{
			ScenarioName: "demo",
			StartedAt:    time.Now().Add(-time.Minute),
			CompletedAt:  time.Now(),
			Success:      true,
		},
	}

	service := NewSuiteExecutionService(
		engine,
		NewSuiteExecutionRepository(db),
		NewSuiteRequestService(&fakeSuiteRequestRepo{}),
	)

	output, err := service.Execute(ctx, SuiteExecutionInput{
		Request: SuiteExecutionRequest{ScenarioName: "demo"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output == nil {
		t.Fatalf("expected execution result")
	}
	if output.ExecutionID == uuid.Nil {
		t.Fatalf("execution id was not assigned")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSuiteExecutionService_RejectsMismatchedSuiteRequest(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	suiteID := uuid.New()
	repo := &fakeSuiteRequestRepo{
		suites: map[uuid.UUID]*SuiteRequest{
			suiteID: {
				ID:           suiteID,
				ScenarioName: "ecosystem-manager",
			},
		},
	}

	service := NewSuiteExecutionService(
		&stubExecutionEngine{},
		NewSuiteExecutionRepository(db),
		NewSuiteRequestService(repo),
	)

	_, err = service.Execute(ctx, SuiteExecutionInput{
		Request:        SuiteExecutionRequest{ScenarioName: "test-genie"},
		SuiteRequestID: &suiteID,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var vErr ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(repo.updates) != 0 {
		t.Fatalf("expected no status updates, got %v", repo.updates)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSuiteExecutionService_MarksSuiteRequestFailedOnRunnerError(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	suiteID := uuid.New()
	repo := &fakeSuiteRequestRepo{
		suites: map[uuid.UUID]*SuiteRequest{
			suiteID: {
				ID:           suiteID,
				ScenarioName: "demo",
			},
		},
	}

	service := NewSuiteExecutionService(
		&stubExecutionEngine{err: errors.New("runner failed")},
		NewSuiteExecutionRepository(db),
		NewSuiteRequestService(repo),
	)

	_, err = service.Execute(ctx, SuiteExecutionInput{
		Request:        SuiteExecutionRequest{ScenarioName: "demo"},
		SuiteRequestID: &suiteID,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(repo.updates) != 2 {
		t.Fatalf("expected two status updates, got %d", len(repo.updates))
	}
	if repo.updates[0].status != suiteStatusRunning {
		t.Fatalf("first status should be running, got %s", repo.updates[0].status)
	}
	if repo.updates[1].status != suiteStatusFailed {
		t.Fatalf("second status should be failed, got %s", repo.updates[1].status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

type stubExecutionEngine struct {
	result *SuiteExecutionResult
	err    error
}

func (s *stubExecutionEngine) Execute(ctx context.Context, req SuiteExecutionRequest) (*SuiteExecutionResult, error) {
	return s.result, s.err
}

type statusChange struct {
	id     uuid.UUID
	status string
}

type fakeSuiteRequestRepo struct {
	suites  map[uuid.UUID]*SuiteRequest
	updates []statusChange
}

func (f *fakeSuiteRequestRepo) Create(ctx context.Context, req *SuiteRequest) error {
	return nil
}

func (f *fakeSuiteRequestRepo) List(ctx context.Context, limit int) ([]SuiteRequest, error) {
	return nil, nil
}

func (f *fakeSuiteRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*SuiteRequest, error) {
	if f.suites == nil {
		return nil, sql.ErrNoRows
	}
	if req, ok := f.suites[id]; ok {
		copy := *req
		return &copy, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeSuiteRequestRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if f.suites == nil {
		return sql.ErrNoRows
	}
	if _, ok := f.suites[id]; !ok {
		return sql.ErrNoRows
	}
	f.updates = append(f.updates, statusChange{id: id, status: status})
	return nil
}

func (f *fakeSuiteRequestRepo) StatusSnapshot(ctx context.Context) (SuiteRequestSnapshot, error) {
	return SuiteRequestSnapshot{}, nil
}
