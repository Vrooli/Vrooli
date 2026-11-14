package recovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

type mockRepo struct {
	executions []*database.ExecutionIndex
	updates    []statusUpdate
	queries    []database.ExecutionQuery
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

func (m *mockRepo) ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	m.queries = append(m.queries, query)
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var matches []*database.ExecutionIndex
	for _, execution := range m.executions {
		if query.Status != "" && execution.Status != query.Status {
			continue
		}
		matches = append(matches, execution)
	}
	total := len(matches)
	if query.Offset >= total {
		return nil, total, nil
	}
	matches = matches[query.Offset:]
	if query.Limit > 0 && query.Limit < len(matches) {
		matches = matches[:query.Limit]
	}
	return matches, total, nil
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

func TestRecoverInterruptedExecutions_NoActiveRows(t *testing.T) {
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

	svc := NewService(repo, log)
	result, err := svc.RecoverInterruptedExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalActive != 0 {
		t.Fatalf("expected TotalActive=0, got %d", result.TotalActive)
	}
	if len(repo.updates) != 0 {
		t.Fatalf("expected no updates, got %d", len(repo.updates))
	}
}

func TestRecoverInterruptedExecutions_MarksInterrupted(t *testing.T) {
	t.Log("[REQ:BAS-RH-J07] persisted running execution becomes terminally failed on API restart")
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

	svc := NewService(repo, log)
	result, err := svc.RecoverInterruptedExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalActive != 1 {
		t.Fatalf("expected TotalActive=1, got %d", result.TotalActive)
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

func TestRecoverInterruptedExecutionsReturnsErrorWhenAnyOwnerRemainsActive(t *testing.T) {
	t.Log("[REQ:BAS-RH-J07] startup recovery rejects an unresolved active owner")
	repo := &mockRepo{
		executions: []*database.ExecutionIndex{{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     database.ExecutionStatusRunning,
		}},
		updateErr: errors.New("database unavailable"),
	}

	result, err := NewService(repo, logrus.New()).RecoverInterruptedExecutions(context.Background())
	if err == nil {
		t.Fatal("RecoverInterruptedExecutions returned nil error while an active execution remained unrecovered")
	}
	if result == nil || result.TotalActive != 1 || result.Failed != 1 || result.Recovered != 0 {
		t.Fatalf("recovery result = %+v, want one failed active execution", result)
	}
}

func TestRecoverInterruptedExecutions_RecoversRecentRowsFromPreviousProcess(t *testing.T) {
	t.Log("[REQ:BAS-RH-J07] recent running and pending executions are both recovered before serving work")
	now := time.Now()
	runningID := uuid.New()
	pendingID := uuid.New()
	repo := &mockRepo{
		executions: []*database.ExecutionIndex{
			{
				ID:         runningID,
				WorkflowID: uuid.New(),
				Status:     database.ExecutionStatusRunning,
				StartedAt:  now.Add(-2 * time.Second),
				CreatedAt:  now.Add(-2 * time.Second),
				UpdatedAt:  now.Add(-time.Second),
			},
			{
				ID:         pendingID,
				WorkflowID: uuid.New(),
				Status:     database.ExecutionStatusPending,
				StartedAt:  now.Add(-2 * time.Second),
				CreatedAt:  now.Add(-2 * time.Second),
				UpdatedAt:  now.Add(-time.Second),
			},
		},
	}

	result, err := NewService(repo, logrus.New()).RecoverInterruptedExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalActive != 2 || result.Recovered != 2 {
		t.Fatalf("expected recent rows from the previous process to be recovered, got total=%d recovered=%d", result.TotalActive, result.Recovered)
	}
	if len(repo.updates) != 2 {
		t.Fatalf("expected both active rows updated, got %d", len(repo.updates))
	}
	updatedIDs := map[uuid.UUID]bool{}
	for _, update := range repo.updates {
		updatedIDs[update.id] = true
		if update.status != database.ExecutionStatusFailed {
			t.Errorf("expected interrupted row to be failed, got %q", update.status)
		}
	}
	if !updatedIDs[runningID] || !updatedIDs[pendingID] {
		t.Fatalf("expected running and pending rows updated, got %v", updatedIDs)
	}
}

func TestRecoverInterruptedExecutions_FindsActiveRowsBeyondRecentHistory(t *testing.T) {
	now := time.Now()
	workflowID := uuid.New()
	executions := make([]*database.ExecutionIndex, 0, 1003)
	for i := range 1000 {
		executions = append(executions, &database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-time.Duration(i) * time.Second),
			UpdatedAt:  now.Add(-time.Duration(i) * time.Second),
		})
	}
	staleRunningID := uuid.New()
	stalePendingID := uuid.New()
	freshRunningID := uuid.New()
	executions = append(executions,
		&database.ExecutionIndex{
			ID:         staleRunningID,
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusRunning,
			StartedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  now.Add(-time.Hour),
		},
		&database.ExecutionIndex{
			ID:         stalePendingID,
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusPending,
			StartedAt:  now.Add(-90 * time.Minute),
			UpdatedAt:  now.Add(-45 * time.Minute),
		},
		&database.ExecutionIndex{
			ID:         freshRunningID,
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusRunning,
			StartedAt:  now.Add(-time.Minute),
			UpdatedAt:  now.Add(-time.Minute),
		},
	)
	repo := &mockRepo{executions: executions}
	svc := NewService(repo, logrus.New())

	result, err := svc.RecoverInterruptedExecutions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalActive != 3 || result.Recovered != 3 || result.Failed != 0 {
		t.Fatalf("expected all active rows recovered, got total=%d recovered=%d failed=%d", result.TotalActive, result.Recovered, result.Failed)
	}
	if len(repo.updates) != 3 {
		t.Fatalf("expected 3 status updates, got %d", len(repo.updates))
	}
	updatedIDs := map[uuid.UUID]bool{}
	for _, update := range repo.updates {
		updatedIDs[update.id] = true
	}
	if !updatedIDs[staleRunningID] || !updatedIDs[stalePendingID] || !updatedIDs[freshRunningID] {
		t.Fatalf("expected all active rows updated, got %v", updatedIDs)
	}
	if len(repo.queries) != 2 ||
		repo.queries[0].Status != database.ExecutionStatusRunning ||
		repo.queries[1].Status != database.ExecutionStatusPending ||
		repo.queries[0].Limit != 0 || repo.queries[1].Limit != 0 {
		t.Fatalf("expected unbounded status-filtered recovery queries, got %+v", repo.queries)
	}
}
