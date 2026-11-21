package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/database"
)

type fakeSessionInventory struct {
	sessions []driver.ObservedSession
	closed   []string
	owners   []string
	leases   []string
}

func (f *fakeSessionInventory) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return f.sessions, nil
}

func (f *fakeSessionInventory) ForceCloseSession(_ context.Context, id, ownerID, leaseID string) error {
	f.closed = append(f.closed, id)
	f.owners = append(f.owners, ownerID)
	f.leases = append(f.leases, leaseID)
	return nil
}

type fakeExecutionLookup struct {
	executions map[uuid.UUID]*database.ExecutionIndex
}

func (f *fakeExecutionLookup) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return f.executions[id], nil
}

func TestSessionReconciler_ForceClosesOnlyTerminalOwnersPastGrace(t *testing.T) {
	terminalID, runningID := uuid.New(), uuid.New()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	inventory := &fakeSessionInventory{sessions: []driver.ObservedSession{{ID: "terminal-session", OwnerExecutionID: terminalID.String(), LeaseID: "terminal-lease"}, {ID: "running-session", OwnerExecutionID: runningID.String(), LeaseID: "running-lease"}}}
	repo := &fakeExecutionLookup{executions: map[uuid.UUID]*database.ExecutionIndex{
		terminalID: {ID: terminalID, Status: database.ExecutionStatusFailed, UpdatedAt: now.Add(-time.Minute)},
		runningID:  {ID: runningID, Status: database.ExecutionStatusRunning, UpdatedAt: now.Add(-time.Hour)},
	}}
	reconciler := newSessionReconciler(inventory, repo, logrus.New(), WithSessionTerminalGrace(15*time.Second))
	reconciler.now = func() time.Time { return now }
	result, err := reconciler.ReconcileOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, result.Observed)
	require.Equal(t, 1, result.Closed)
	require.Equal(t, []string{"terminal-session"}, inventory.closed)
	require.Equal(t, []string{terminalID.String()}, inventory.owners)
	require.Equal(t, []string{"terminal-lease"}, inventory.leases)
}

func TestSessionReconciler_ClosesOnlyStaleUnknownAudioSoakOwners(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	old := now.Add(-(DefaultSoakOrphanGrace + time.Second)).Format(time.RFC3339Nano)
	fresh := now.Add(-time.Minute).Format(time.RFC3339Nano)
	inventory := &fakeSessionInventory{sessions: []driver.ObservedSession{
		{ID: "stale-soak", OwnerExecutionID: "browser-soak-legacy", LeaseID: "stale-soak-lease", WorkflowID: audioToolsSoakWorkflowID, LastUsedAt: old},
		{ID: "fresh-soak", OwnerExecutionID: uuid.NewString(), WorkflowID: audioToolsSoakWorkflowID, LastUsedAt: fresh},
		{ID: "stale-other", OwnerExecutionID: "missing-owner", WorkflowID: "another-workflow", LastUsedAt: old},
	}}
	reconciler := newSessionReconciler(inventory, &fakeExecutionLookup{executions: map[uuid.UUID]*database.ExecutionIndex{}}, logrus.New())
	reconciler.now = func() time.Time { return now }

	result, err := reconciler.ReconcileOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, result.Observed)
	require.Equal(t, 1, result.Closed)
	require.Equal(t, []string{"stale-soak"}, inventory.closed)
}
