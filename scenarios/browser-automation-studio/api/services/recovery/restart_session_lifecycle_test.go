package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/database"
)

// restartLifecycleRepo models the durable row transition across an API process
// restart; the driver inventory remains alive independently of that API owner.
type restartLifecycleRepo struct{ execution *database.ExecutionIndex }

func (r *restartLifecycleRepo) ListExecutions(_ context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	if r.execution == nil || r.execution.Status != query.Status {
		return nil, 0, nil
	}
	copy := *r.execution
	return []*database.ExecutionIndex{&copy}, 1, nil
}

func (r *restartLifecycleRepo) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if r.execution == nil || r.execution.ID != id {
		return nil, nil
	}
	copy := *r.execution
	return &copy, nil
}

func (r *restartLifecycleRepo) UpdateExecutionStatus(_ context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	if r.execution == nil || r.execution.ID != id {
		return nil
	}
	r.execution.Status = status
	if errorMessage != nil {
		r.execution.ErrorMessage = *errorMessage
	}
	r.execution.CompletedAt = completedAt
	r.execution.UpdatedAt = updatedAt
	return nil
}

type restartLifecycleDriver struct {
	sessions []driver.ObservedSession
	closed   []driver.ObservedSession
}

func (d *restartLifecycleDriver) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return append([]driver.ObservedSession(nil), d.sessions...), nil
}

func (d *restartLifecycleDriver) ForceCloseSession(_ context.Context, id, ownerID, leaseID string) error {
	for i, session := range d.sessions {
		if session.ID != id || session.OwnerExecutionID != ownerID || session.LeaseID != leaseID {
			continue
		}
		d.closed = append(d.closed, session)
		d.sessions = append(d.sessions[:i], d.sessions[i+1:]...)
		return nil
	}
	return nil
}

func TestStartupRecoveryThenSessionReconciliationClosesInterruptedRunOwner(t *testing.T) {
	executionID := uuid.New()
	repo := &restartLifecycleRepo{execution: &database.ExecutionIndex{
		ID: executionID, Status: database.ExecutionStatusRunning, UpdatedAt: time.Now().UTC().Add(-time.Hour),
	}}
	driverOwner := &restartLifecycleDriver{sessions: []driver.ObservedSession{{
		ID: "orphan-after-api-restart", OwnerExecutionID: executionID.String(), LeaseID: "old-driver-lease",
	}}}

	recoveryResult, err := NewService(repo, logrus.New(), t.TempDir()).RecoverInterruptedExecutions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if recoveryResult.Recovered != 1 || repo.execution.Status != database.ExecutionStatusFailed {
		t.Fatalf("startup execution recovery = %#v, row = %#v", recoveryResult, repo.execution)
	}

	reconciler := newSessionReconciler(driverOwner, repo, logrus.New(), WithSessionTerminalGrace(0))
	result, err := reconciler.ReconcileOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Observed != 1 || result.Closed != 1 || len(driverOwner.sessions) != 0 || len(driverOwner.closed) != 1 {
		t.Fatalf("post-restart session reconciliation = %#v, remaining=%#v closed=%#v", result, driverOwner.sessions, driverOwner.closed)
	}
	closed := driverOwner.closed[0]
	if closed.ID != "orphan-after-api-restart" || closed.OwnerExecutionID != executionID.String() || closed.LeaseID != "old-driver-lease" {
		t.Fatalf("reconciler did not close the exact interrupted execution lease: %#v", closed)
	}

	second, err := reconciler.ReconcileOnce(context.Background())
	if err != nil || second.Observed != 0 || second.Closed != 0 {
		t.Fatalf("second recovery pass was not idempotent: result=%#v err=%v", second, err)
	}
}
