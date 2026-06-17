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

func TestIssueAndRedeemPairingCode(t *testing.T) {
	t.Parallel()
	svc, repo, _, _, _ := newService(t)
	ctx := context.Background()

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

	// Bootstrap an owner via the code path first (request path needs an owner).
	pc, err := svc.IssuePairingCode(ctx, "owner-1", "")
	require.NoError(t, err)
	_, err = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	require.NoError(t, err)

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
	assert.ErrorAs(t, err, &conflict, "first device must use the code path")
}

func TestApprovePromotesPending(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	_, _ = svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
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

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	issued, _ := svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	_, err := svc.Revoke(ctx, "owner-1", issued.Device.ID)
	require.NoError(t, err)

	_, err = svc.Approve(ctx, "owner-1", issued.Device.ID)
	var conflict devices.ErrDeviceConflict
	assert.ErrorAs(t, err, &conflict)
}

func TestRevokeFlipsTrustAndRevokesSession(t *testing.T) {
	t.Parallel()
	svc, repo, _, authd, _ := newService(t)
	ctx := context.Background()

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	issued, _ := svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	// Bind an authenticator session so revoke has something to call.
	d := repo.Devices[issued.Device.ID]
	d.SessionID = "sess-9"
	repo.Devices[d.ID] = d

	revoked, err := svc.Revoke(ctx, "owner-1", issued.Device.ID)
	require.NoError(t, err)
	assert.Equal(t, devices.TrustRevoked, revoked.TrustState)
	assert.Equal(t, []string{"sess-9"}, authd.RevokedIDs, "authenticator session revoked")
}

func TestRevokeSucceedsLocallyWhenAuthUnavailable(t *testing.T) {
	t.Parallel()
	svc, repo, _, authd, _ := newService(t)
	ctx := context.Background()
	authd.RevokeErr = errors.New("authenticator down")

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	issued, _ := svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})
	d := repo.Devices[issued.Device.ID]
	d.SessionID = "sess-9"
	repo.Devices[d.ID] = d

	revoked, err := svc.Revoke(ctx, "owner-1", issued.Device.ID)
	require.NoError(t, err, "local revoke is the guarantee; auth failure is logged not fatal")
	assert.Equal(t, devices.TrustRevoked, revoked.TrustState)
}

func TestRenameValidates(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	pc, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	issued, _ := svc.RedeemPairingCode(ctx, pc.Code, devices.Profile{})

	_, err := svc.Rename(ctx, "owner-1", issued.Device.ID, "   ")
	var invalid devices.ErrInvalidDevice
	assert.ErrorAs(t, err, &invalid)

	renamed, err := svc.Rename(ctx, "owner-1", issued.Device.ID, "Laptop")
	require.NoError(t, err)
	assert.Equal(t, "Laptop", renamed.Name)
}

func TestListIsOwnerScoped(t *testing.T) {
	t.Parallel()
	svc, _, _, _, _ := newService(t)
	ctx := context.Background()

	pc1, _ := svc.IssuePairingCode(ctx, "owner-1", "")
	_, _ = svc.RedeemPairingCode(ctx, pc1.Code, devices.Profile{})
	pc2, _ := svc.IssuePairingCode(ctx, "owner-2", "")
	_, _ = svc.RedeemPairingCode(ctx, pc2.Code, devices.Profile{})

	list, err := svc.List(ctx, "owner-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "owner-1", list[0].OwnerID)
}
