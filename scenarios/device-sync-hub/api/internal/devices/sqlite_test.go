package devices_test

import (
	"context"
	"testing"
	"time"

	"device-sync-hub/internal/devices"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "device-sync-hub/internal/database"
)

// newRepo returns a sqlite-backed devices.Repository with the production schema
// applied — the canonical repository-test compose pattern.
func newRepo(t *testing.T) (devices.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(devices.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	return devices.NewSQLiteRepository(d, clk), clk
}

func TestSQLiteDeviceRoundTrip(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	created, err := repo.CreateDevice(ctx, devices.Device{
		OwnerID:      "owner-1",
		Name:         "Pixel",
		Kind:         "phone",
		Platform:     "android",
		Capabilities: []string{"camera", "clipboard"},
		TrustState:   devices.TrustTrusted,
	}, "hash-abc")
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, clk.Now(), created.CreatedAt)

	got, err := repo.GetDevice(ctx, "owner-1", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pixel", got.Name)
	assert.Equal(t, []string{"camera", "clipboard"}, got.Capabilities)
	assert.Equal(t, devices.TrustTrusted, got.TrustState)
	assert.False(t, got.Online, "online is runtime presence, 0 at rest")
}

func TestSQLiteGetIsOwnerScoped(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	created, err := repo.CreateDevice(ctx, devices.Device{OwnerID: "owner-1", Name: "A", TrustState: devices.TrustTrusted}, "h")
	require.NoError(t, err)

	_, err = repo.GetDevice(ctx, "owner-2", created.ID)
	var nf devices.ErrDeviceNotFound
	assert.ErrorAs(t, err, &nf, "a device must not be readable by another owner")
}

func TestSQLiteDeviceByToken(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	created, err := repo.CreateDevice(ctx, devices.Device{OwnerID: "o", Name: "A", TrustState: devices.TrustTrusted}, "tokenhash-1")
	require.NoError(t, err)

	got, err := repo.DeviceByToken(ctx, "tokenhash-1")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	_, err = repo.DeviceByToken(ctx, "")
	var nf devices.ErrDeviceNotFound
	assert.ErrorAs(t, err, &nf, "empty token hash never matches")

	_, err = repo.DeviceByToken(ctx, "nope")
	assert.ErrorAs(t, err, &nf)
}

func TestSQLiteSetTrustAndRename(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	created, _ := repo.CreateDevice(ctx, devices.Device{OwnerID: "o", Name: "A", TrustState: devices.TrustPending}, "h")

	trusted, err := repo.SetTrust(ctx, "o", created.ID, devices.TrustTrusted)
	require.NoError(t, err)
	assert.Equal(t, devices.TrustTrusted, trusted.TrustState)

	renamed, err := repo.Rename(ctx, "o", created.ID, "B")
	require.NoError(t, err)
	assert.Equal(t, "B", renamed.Name)

	_, err = repo.SetTrust(ctx, "o", "missing", devices.TrustRevoked)
	var nf devices.ErrDeviceNotFound
	assert.ErrorAs(t, err, &nf)
}

func TestSQLiteHubOwnerClaim(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	_, err := repo.HubOwner(ctx)
	assert.ErrorIs(t, err, devices.ErrNoOwner, "an unclaimed hub has no owner")

	// First claim wins and is returned.
	holder, err := repo.ClaimOwner(ctx, "owner-1", clk.Now())
	require.NoError(t, err)
	assert.Equal(t, "owner-1", holder)

	// A competing claim does not displace the established owner.
	clk.Advance(time.Minute)
	holder, err = repo.ClaimOwner(ctx, "owner-2", clk.Now())
	require.NoError(t, err)
	assert.Equal(t, "owner-1", holder, "second claimant loses; ownership is first-owner-wins")

	owner, err := repo.HubOwner(ctx)
	require.NoError(t, err)
	assert.Equal(t, "owner-1", owner)
}

func TestSQLiteClaimPairingCodeSingleUse(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	now := clk.Now()

	require.NoError(t, repo.CreatePairingCode(ctx, devices.PairingCode{
		CodeHash:   "codehash-1",
		OwnerID:    "owner-1",
		DeviceName: "Phone slot",
		ExpiresAt:  now.Add(15 * time.Minute),
		CreatedAt:  now,
	}))

	claimed, err := repo.ClaimPairingCode(ctx, "codehash-1", now)
	require.NoError(t, err)
	assert.Equal(t, "owner-1", claimed.OwnerID)
	assert.Equal(t, "Phone slot", claimed.DeviceName)

	// Second claim must fail — single use.
	_, err = repo.ClaimPairingCode(ctx, "codehash-1", now)
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad)
}

func TestSQLiteClaimPairingCodeRejectsExpired(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	now := clk.Now()

	require.NoError(t, repo.CreatePairingCode(ctx, devices.PairingCode{
		CodeHash:  "codehash-2",
		OwnerID:   "owner-1",
		ExpiresAt: now.Add(time.Minute),
		CreatedAt: now,
	}))

	_, err := repo.ClaimPairingCode(ctx, "codehash-2", now.Add(2*time.Minute))
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad)
}

func TestSQLiteClaimUnknownCode(t *testing.T) {
	repo, clk := newRepo(t)
	_, err := repo.ClaimPairingCode(context.Background(), "ghost", clk.Now())
	var bad devices.ErrInvalidPairingCode
	assert.ErrorAs(t, err, &bad)
}
