package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

type mockRepo struct {
	executions []*database.ExecutionIndex
	updated    []*database.ExecutionIndex
	listErr    error
	updateErr  error
}

func (m *mockRepo) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.executions, nil
}

func (m *mockRepo) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, execution)
	return nil
}

func TestRecoverStaleExecutions_NoStale(t *testing.T) {
	repo := &mockRepo{
		executions: []*database.ExecutionIndex{
			{
				ID:         uuid.New(),
				WorkflowID: uuid.New(),
				Status:     database.ExecutionStatusCompleted,
				StartedAt:  time.Now().Add(-10 * time.Minute),
				CreatedAt:  time.Now().Add(-10 * time.Minute),
				UpdatedAt:  time.Now(),
			},
		},
	}
	log := logrus.New()

	svc := NewService(repo, log, WithStaleThreshold(2*time.Minute))
	result, err := svc.RecoverStaleExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalStale != 0 {
		t.Fatalf("expected TotalStale=0, got %d", result.TotalStale)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("expected no updates, got %d", len(repo.updated))
	}
}

func TestRecoverStaleExecutions_MarksInterrupted(t *testing.T) {
	staleID := uuid.New()
	workflowID := uuid.New()
	now := time.Now()
	repo := &mockRepo{
		executions: []*database.ExecutionIndex{
			{
				ID:         staleID,
				WorkflowID: workflowID,
				Status:     database.ExecutionStatusRunning,
				StartedAt:  now.Add(-20 * time.Minute),
				CreatedAt:  now.Add(-20 * time.Minute),
				UpdatedAt:  now.Add(-10 * time.Minute),
			},
		},
	}
	log := logrus.New()

	svc := NewService(repo, log, WithStaleThreshold(2*time.Minute))
	result, err := svc.RecoverStaleExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalStale != 1 {
		t.Fatalf("expected TotalStale=1, got %d", result.TotalStale)
	}
	if result.Recovered != 1 {
		t.Fatalf("expected Recovered=1, got %d", result.Recovered)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 updated execution, got %d", len(repo.updated))
	}
	updated := repo.updated[0]
	if updated.ID != staleID {
		t.Fatalf("expected updated execution id %s, got %s", staleID, updated.ID)
	}
	if updated.Status != database.ExecutionStatusFailed {
		t.Fatalf("expected status failed, got %q", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Fatalf("expected completed_at to be set")
	}
	if updated.ErrorMessage == "" {
		t.Fatalf("expected error_message to be set")
	}
}

