package devices_test

import (
	"context"
	"testing"

	"device-sync-hub/internal/devices"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authRepo composes a sqlite repo and registers a device in a given trust state
// via the service's pairing paths is overkill here; we exercise the
// Authenticator against the real repo by inserting through CreateDevice.
func TestAuthenticator_ResolvesTrustedOnly(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	authn := devices.NewAuthenticator(repo)

	// The authenticator hashes the raw token with the same one-way hash the
	// service uses at issue time. We can't see that hash from the test package,
	// so drive the real bootstrap path: SetupOwnerDevice mints a TRUSTED device
	// and returns its raw token once.
	svc := devices.NewService(devices.Config{Repo: repo})
	issued, err := svc.SetupOwnerDevice(ctx, "owner-1", devices.Profile{Name: "Phone"})
	require.NoError(t, err)

	// A valid TRUSTED token resolves to its device.
	got, err := authn.Authenticate(ctx, issued.Token)
	require.NoError(t, err)
	assert.Equal(t, issued.Device.ID, got.ID)
	assert.Equal(t, devices.TrustTrusted, got.TrustState)

	// Unknown / blank tokens are indistinguishably untrusted.
	_, err = authn.Authenticate(ctx, "not-a-real-token")
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)
	_, err = authn.Authenticate(ctx, "")
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)

	// Revoking the device makes its token stop resolving immediately.
	_, err = svc.Revoke(ctx, "owner-1", issued.Device.ID)
	require.NoError(t, err)
	_, err = authn.Authenticate(ctx, issued.Token)
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)
}

func TestAuthenticator_PendingTokenIsInert(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	authn := devices.NewAuthenticator(repo)
	svc := devices.NewService(devices.Config{Repo: repo})

	// Bootstrap the owner's first device, then a fallback PENDING request.
	_, err := svc.SetupOwnerDevice(ctx, "owner-1", devices.Profile{Name: "First"})
	require.NoError(t, err)

	pending, err := svc.RequestPairing(ctx, devices.Profile{Name: "Pending"})
	require.NoError(t, err)

	// A PENDING device's token must not authenticate until approved.
	_, err = authn.Authenticate(ctx, pending.Token)
	assert.ErrorIs(t, err, devices.ErrUntrustedDevice)

	_, err = svc.Approve(ctx, "owner-1", pending.Device.ID)
	require.NoError(t, err)
	got, err := authn.Authenticate(ctx, pending.Token)
	require.NoError(t, err)
	assert.Equal(t, pending.Device.ID, got.ID)
}
