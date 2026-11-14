package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDesktopExpiryRevokesBeforeRetryingFailedRelease(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-09]
	c, repo, native, _, lease := desktopFixture(t)
	c.now = func() time.Time { return lease.ExpiresAt }
	native.releaseFail = true
	require.Error(t, c.ReapExpired(context.Background()))
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
		require.Nil(t, s.Lease)
		require.True(t, s.CleanupPending)
		return nil
	}))
	native.releaseFail = false
	require.NoError(t, c.ReapExpired(context.Background()))
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
		require.Nil(t, s.Lease)
		require.False(t, s.CleanupPending)
		return nil
	}))
	_, err := c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
}

func TestDesktopHelperRestartFencesOldHelperAndRetainsUncertainty(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-13]
	c, repo, native, auth, lease := desktopFixture(t)
	native.fail = true
	receipt, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	next, err := NewDesktopController(repo, auth, native, c.surface, c.desktopID, "helper-generation-2")
	require.NoError(t, err)
	require.NoError(t, next.ActivateHelper(context.Background(), c.helperID, lease.Epoch))
	require.ErrorIs(t, c.ReapExpired(context.Background()), ErrDesktopAdmission)
	require.ErrorIs(t, c.Shutdown(context.Background()), ErrDesktopAdmission)
	_, err = c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
		require.Nil(t, s.Lease)
		require.Equal(t, lease.Epoch+1, s.Epoch)
		require.Equal(t, receipt, s.Receipts[receipt.CommandID])
		return nil
	}))
	require.ErrorIs(t, c.ActivateHelper(context.Background(), c.helperID, lease.Epoch), ErrDesktopAdmission)
	successor := lease
	successor.HelperID = next.helperID
	successor.Ref.SessionID = "next-session"
	_, err = c.Open(context.Background(), lease, lease.Epoch+1, true)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	successor, err = next.Open(context.Background(), successor, lease.Epoch+1, false)
	require.NoError(t, err)
	require.Equal(t, lease.Epoch+2, successor.Epoch)
}

func TestDesktopFailedTakeoverReleaseRevokesPriorLease(t *testing.T) {
	c, repo, native, _, lease := desktopFixture(t)
	native.releaseFail = true
	next := lease
	next.Ref.SessionID = "next-session"
	_, err := c.Open(context.Background(), next, lease.Epoch, true)
	require.Error(t, err)
	_, err = c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error { require.Nil(t, s.Lease); require.True(t, s.CleanupPending); return nil }))
	native.releaseFail = false
	next, err = c.Open(context.Background(), next, lease.Epoch, false)
	require.NoError(t, err)
	_, err = c.Act(context.Background(), desktopCommand(next))
	require.NoError(t, err)
	require.Equal(t, 1, native.effects)
}

func TestDesktopLifecycleCancellationUsesFreshCleanupContext(t *testing.T) {
	c, repo, _, _, lease := desktopFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, c.MaintainLease(ctx))
	_, err := c.Open(context.Background(), lease, lease.Epoch, false)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error { require.Nil(t, s.Lease); return nil }))
}

type lockedDesktopNative struct {
	desktopNative
	locked bool
}

func (n *lockedDesktopNative) CheckSession(context.Context) error {
	if n.locked {
		return ErrDesktopAdmission
	}
	return nil
}

func TestDesktopSessionLockRevokesWithoutClientRequest(t *testing.T) {
	c, repo, _, _, lease := desktopFixture(t)
	native := &lockedDesktopNative{}
	c.native = native
	require.NoError(t, c.ReapExpired(context.Background()))
	native.locked = true
	require.NoError(t, c.ReapExpired(context.Background()))
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error { require.Nil(t, s.Lease); require.Empty(t, s.GrantID); return nil }))
	_, err := c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
}

func TestDesktopCleanupReceiptSurvivesRestartAndCannotDescribeAnotherLease(t *testing.T) {
	ctx := context.Background()
	c, repo, native, auth, lease := desktopFixture(t)
	native.releaseFail = true
	require.Error(t, c.Stop(ctx, lease))
	receipt, err := repo.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.False(t, receipt.Released)
	require.True(t, sameDesktopLease(lease, receipt.Lease))
	// Open a second database connection and repository, not a process-local map.
	var seq int
	var name, path string
	require.NoError(t, repo.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&seq, &name, &path))
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	recovered, err := NewSQLiteDesktopRepository(ctx, db, repo.destination)
	require.NoError(t, err)
	receipt, err = recovered.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.False(t, receipt.Released)
	other := lease
	other.Actor = "other-actor"
	_, err = recovered.ReadCleanup(ctx, other)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	other = lease
	other.Epoch++
	_, err = recovered.ReadCleanup(ctx, other)
	require.ErrorIs(t, err, sql.ErrNoRows)
	next, err := NewDesktopController(recovered, auth, native, c.surface, c.desktopID, "next-helper")
	require.NoError(t, err)
	native.releaseFail = false
	require.NoError(t, next.ActivateHelper(ctx, c.helperID, lease.Epoch))
	receipt, err = recovered.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.True(t, receipt.Released)
	// Later cleanup failure must not revoke already-confirmed historical evidence.
	successor := lease
	successor.HelperID = next.helperID
	successor.Ref.SessionID = "successor"
	successor, err = next.Open(ctx, successor, lease.Epoch+1, false)
	require.NoError(t, err)
	native.releaseFail = true
	require.Error(t, next.Stop(ctx, successor))
	old, err := recovered.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.Equal(t, receipt, old)
	pending, err := recovered.ReadCleanup(ctx, successor)
	require.NoError(t, err)
	require.False(t, pending.Released)
	_, err = next.Act(ctx, desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
}

func TestDesktopExpiryReceiptRequiresDestinationCleanup(t *testing.T) {
	ctx := context.Background()
	c, repo, native, _, lease := desktopFixture(t)
	c.now = func() time.Time { return lease.ExpiresAt }
	_, err := repo.ReadCleanup(ctx, lease)
	require.ErrorIs(t, err, sql.ErrNoRows)
	native.releaseFail = true
	require.Error(t, c.ReapExpired(ctx))
	pending, err := repo.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.False(t, pending.Released)
	native.releaseFail = false
	require.NoError(t, c.ReapExpired(ctx))
	released, err := repo.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.True(t, released.Released)
	require.True(t, released.ObservedAt.Equal(lease.ExpiresAt))
}

func TestDesktopCleanupStorageFailureCannotProduceConfirmation(t *testing.T) {
	ctx := context.Background()
	c, repo, _, _, lease := desktopFixture(t)
	_, err := repo.db.ExecContext(ctx, `CREATE TRIGGER reject_cleanup_receipt BEFORE INSERT ON device_control_desktop_cleanup_receipts BEGIN SELECT RAISE(ABORT,'fixture storage failure'); END`)
	require.NoError(t, err)
	require.Error(t, c.Stop(ctx, lease))
	_, err = repo.ReadCleanup(ctx, lease)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDesktopConfirmedCleanupDoesNotGrowLiveInputState(t *testing.T) {
	ctx := context.Background()
	c, repo, _, _, lease := desktopFixture(t)
	require.NoError(t, c.Stop(ctx, lease))
	var raw string
	require.NoError(t, repo.db.QueryRowContext(ctx, `SELECT state FROM device_control_desktop_sessions WHERE destination=?`, repo.destination).Scan(&raw))
	var state DesktopState
	require.NoError(t, json.Unmarshal([]byte(raw), &state))
	require.Empty(t, state.CleanupReceipts)
	receipt, err := repo.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.True(t, receipt.Released)
}
