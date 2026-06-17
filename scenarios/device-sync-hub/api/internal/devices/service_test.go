package devices_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"device-sync-hub/internal/devices"
	"device-sync-hub/internal/devices/mocks"
	tmocks "device-sync-hub/internal/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService(t *testing.T) (devices.Service, *mocks.FakeRepository, *mocks.FakeSecrets, *mocks.FakeAuth, *tmocks.FakeClock) {
	t.Helper()
	repo := mocks.NewFakeRepository()
	sec := &mocks.FakeSecrets{}
	authd := &mocks.FakeAuth{}
	clk := tmocks.NewFakeClock(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	svc := devices.NewService(devices.Config{
		Repo:       repo,
		Clock:      clk,
		Secrets:    sec,
		Auth:       authd,
		PairingTTL: 15 * time.Minute,
	})
	return svc, repo, sec, authd, clk
}

// setupOwner runs the first-run bootstrap for ownerID and returns the owner's
// trusted device. Owner-authed RPCs require the hub to be claimed first, so
// every test that issues codes / approves / renames / revokes calls this.
func setupOwner(t *testing.T, svc devices.Service, ownerID string) devices.IssuedToken {
	t.Helper()
	issued, err := svc.SetupOwnerDevice(context.Background(), ownerID, devices.Profile{Name: "Owner device", Kind: "laptop"})
	require.NoError(t, err)
	require.Equal(t, devices.TrustTrusted, issued.Device.TrustState)
	return issued
}

// [REQ:REQ-P0-003] First device is admitted to the trust group via owner bootstrap.
func TestSetupOwnerDeviceClaimsHubAndTrustsDevice(t *testing.T) {
	t.Parallel()
	svc, repo, _, _, _ := newService(t)
	ctx := context.Background()

	issued, err := svc.SetupOwnerDevice(ctx, "owner-1", devices.Profile{Name: "Workstation", Kind: "laptop"})
	require.NoError(t, err)
	assert.Equal(t, devices.TrustTrusted, issued.Device.TrustState, "owner's first device is trusted directly")
	assert.Equal(t, "owner-1", issued.Device.OwnerID)
	assert.Equal(t, "Workstation", issued.Device.Name)
	assert.NotEmpty(t, issued.Token, "one-time device token returned")
	assert.True(t, repo.OwnerClaimed, "hub is claimed after setup")
	assert.Equal(t, "owner-1", repo.Owner)
}

func TestSetupOwnerDeviceRequiresOwnerID(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	_, err := svc.SetupOwnerDevice(context.Background(), "   ", devices.Profile{})
	var invalid devices.ErrInvalidDevice
	assert.ErrorAs(t, err, &invalid)
}

func TestSetupOwnerDeviceSameOwnerAddsAnotherTrustedDevice(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	first := setupOwner(t, svc, "owner-1")
	second, err := svc.SetupOwnerDevice(ctx, "owner-1", devices.Profile{Name: "Laptop"})
	require.NoError(t, err)
	assert.NotEqual(t, first.Device.ID, second.Device.ID, "a second setup registers a distinct device")
	assert.Equal(t, devices.TrustTrusted, second.Device.TrustState)
}

// [REQ:REQ-P0-005] Single-owner: a second authenticator identity cannot claim an owned hub.
func TestSetupOwnerDeviceRejectsDifferentOwner(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	setupOwner(t, svc, "owner-1")
	_, err := svc.SetupOwnerDevice(ctx, "owner-2", devices.Profile{Name: "Intruder"})
	var notOwner devices.ErrNotOwner
	assert.ErrorAs(t, err, &notOwner, "a second identity cannot claim an already-owned hub")
}

func TestOwnerAuthedRPCsRequireSetupFirst(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	var conflict devices.ErrDeviceConflict

	_, err := svc.List(ctx, "owner-1")
	assert.ErrorAs(t, err, &conflict, "List before setup is a precondition failure")

	_, err = svc.IssuePairingCode(ctx, "owner-1", "Phone")
	assert.ErrorAs(t, err, &conflict, "IssuePairingCode before setup is a precondition failure")
}

// [REQ:REQ-P0-005] Owner-authed RPCs reject a non-owner identity (single-owner pin).
func TestOwnerAuthedRPCsRejectNonOwner(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	var notOwner devices.ErrNotOwner

	_, err := svc.List(ctx, "owner-2")
	assert.ErrorAs(t, err, &notOwner, "List")

	_, err = svc.IssuePairingCode(ctx, "owner-2", "Phone")
	assert.ErrorAs(t, err, &notOwner, "IssuePairingCode")

	_, err = svc.Approve(ctx, "owner-2", "any")
	assert.ErrorAs(t, err, &notOwner, "Approve")

	_, err = svc.Rename(ctx, "owner-2", "any", "x")
	assert.ErrorAs(t, err, &notOwner, "Rename")

	_, err = svc.Revoke(ctx, "owner-2", "any")
	assert.ErrorAs(t, err, &notOwner, "Revoke")
}

func TestIssueAndRedeemPairingCode(t *testing.T) {
	t.Parallel()
	svc, repo, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	pc, err := svc.IssuePairingCode(ctx, "owner-1", "My Phone")
	require.NoError(t, err)
	assert.NotEmpty(t, pc.Code, "raw code must be returned at issue time")
	assert.Equal(t, "owner-1", pc.OwnerID)
	require.Len(t, repo.Codes, 1, "code persisted by hash")

	issued, err := svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{Kind: "phone", Platform: "android"})
	require.NoError(t, err)
	assert.Equal(t, devices.TrustTrusted, issued.Device.TrustState, "redeemed device is trusted immediately")
	assert.Equal(t, "owner-1", issued.Device.OwnerID)
	assert.Equal(t, "My Phone", issued.Device.Name, "issue-time name carries to the device")
	assert.NotEmpty(t, issued.Token, "raw device token returned once")
}

func TestRedeemPairingCodeIsSingleUse(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	pc, err := svc.IssuePairingCode(ctx, "owner-1", "")
	require.NoError(t, err)

	_, err = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	require.NoError(t, err)

	_, err = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad, "a code cannot be redeemed twice")
}

func TestRedeemPairingCodeRejectsExpired(t *testing.T) {
	t.Parallel()
	svc, _, _, _, clk := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	pc, err := svc.IssuePairingCode(ctx, "owner-1", "")
	require.NoError(t, err)

	clk.Advance(16 * time.Minute) // past the 15m TTL

	_, err = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad)
}

func TestRedeemUnknownCode(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	_, err := svc.RedeemPairingCode(context.Background(), "NOPE0-NOPE0", devices.Profile{})
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad)
}

func TestRequestPairingCreatesPendingDevice(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	issued, err := svc.RequestPairing(ctx, devices.Profile{Name: "Tablet", Kind: "tablet"})
	require.NoError(t, err)
	assert.Equal(t, devices.TrustPending, issued.Device.TrustState)
	assert.Equal(t, "owner-1", issued.Device.OwnerID, "request joins the hub's single owner")
	assert.NotEmpty(t, issued.Token, "token issued but inert until approval")
}

func TestRequestPairingWithoutOwnerFails(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	_, err := svc.RequestPairing(context.Background(), devices.Profile{Name: "First"})
	var conflict devices.ErrDeviceConflict
	assert.ErrorAs(t, err, &conflict, "first device must be set up by the owner")
}

func TestApprovePromotesPending(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	pending, err := svc.RequestPairing(ctx, devices.Profile{Name: "Tablet"})
	require.NoError(t, err)

	approved, err := svc.Approve(ctx, "owner-1", pending.Device.ID)
	require.NoError(t, err)
	assert.Equal(t, devices.TrustTrusted, approved.TrustState)

	// Idempotent.
	again, err := svc.Approve(ctx, "owner-1", pending.Device.ID)
	require.NoError(t, err)
	assert.Equal(t, devices.TrustTrusted, again.TrustState)
}

func TestApproveRevokedFails(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	owner := setupOwner(t, svc, "owner-1")

	_, err := svc.Revoke(ctx, "owner-1", owner.Device.ID)
	require.NoError(t, err)

	_, err = svc.Approve(ctx, "owner-1", owner.Device.ID)
	var conflict devices.ErrDeviceConflict
	assert.ErrorAs(t, err, &conflict)
}

func TestRevokeFlipsTrustAndRevokesSession(t *testing.T) {
	t.Parallel()
	svc, repo, _, authd, _ := newService(t)
	ctx := context.Background()
	owner := setupOwner(t, svc, "owner-1")

	// Bind an authenticator session so revoke has something to call.
	d := repo.Devices[owner.Device.ID]
	d.SessionID = "sess-9"
	repo.Devices[d.ID] = d

	revoked, err := svc.Revoke(ctx, "owner-1", owner.Device.ID)
	require.NoError(t, err)
	assert.Equal(t, devices.TrustRevoked, revoked.TrustState)
	assert.Equal(t, []string{"sess-9"}, authd.RevokedIDs, "authenticator session revoked")
}

func TestRevokeSucceedsLocallyWhenAuthUnavailable(t *testing.T) {
	t.Parallel()
	svc, repo, _, authd, _ := newService(t)
	ctx := context.Background()
	authd.RevokeErr = errors.New("authenticator down")
	owner := setupOwner(t, svc, "owner-1")

	d := repo.Devices[owner.Device.ID]
	d.SessionID = "sess-9"
	repo.Devices[d.ID] = d

	revoked, err := svc.Revoke(ctx, "owner-1", owner.Device.ID)
	require.NoError(t, err, "local revoke is the guarantee; auth failure is logged not fatal")
	assert.Equal(t, devices.TrustRevoked, revoked.TrustState)
}

func TestRenameValidates(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	owner := setupOwner(t, svc, "owner-1")

	_, err := svc.Rename(ctx, "owner-1", owner.Device.ID, "   ")
	var invalid devices.ErrInvalidDevice
	assert.ErrorAs(t, err, &invalid)

	renamed, err := svc.Rename(ctx, "owner-1", owner.Device.ID, "Laptop")
	require.NoError(t, err)
	assert.Equal(t, "Laptop", renamed.Name)
}

func TestListReturnsOwnerDevices(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()
	setupOwner(t, svc, "owner-1")

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	_, _ = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})

	list, err := svc.List(ctx, "owner-1")
	require.NoError(t, err)
	require.Len(t, list, 2, "owner device + redeemed device")
	for _, d := range list {
		assert.Equal(t, "owner-1", d.OwnerID)
	}
}
