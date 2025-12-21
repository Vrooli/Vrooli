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
	updates    []statusUpdate
	listErr    error
	updateErr  error
}

type statusUpdate struct {
	id           uuid.UUID
	status       string
	errorMessage *string
	completedAt  *time.Time
	updatedAt    time.Time
}

func (m *mockRepo) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.executions, nil
}

func (m *mockRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updates = append(m.updates, statusUpdate{
		id:           id,
		status:       status,
		errorMessage: errorMessage,
		completedAt:  completedAt,
		updatedAt:    updatedAt,
	})
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
	if len(repo.updates) != 0 {
		t.Fatalf("expected no updates, got %d", len(repo.updates))
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
	if len(repo.updates) != 1 {
		t.Fatalf("expected 1 updated execution, got %d", len(repo.updates))
	}
	updated := repo.updates[0]
	if updated.id != staleID {
		t.Fatalf("expected updated execution id %s, got %s", staleID, updated.id)
	}
	if updated.status != database.ExecutionStatusFailed {
		t.Fatalf("expected status failed, got %q", updated.status)
	}
	if updated.completedAt == nil {
		t.Fatalf("expected completed_at to be set")
	}
	if updated.errorMessage == nil || *updated.errorMessage == "" {
		t.Fatalf("expected error_message to be set")
	}
}
