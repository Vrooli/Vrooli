package control

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"path/filepath"
	"testing"
	"time"

	"device-control/internal/desktophelper"
	"device-control/internal/sessions"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/targetmodel"
)

func admissionPeer(t *testing.T) *net.UnixConn {
	t.Helper()
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(t.TempDir(), "peer"), Net: "unix"})
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	c, err := net.DialUnix("unix", nil, l.Addr().(*net.UnixAddr))
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	p, err := l.AcceptUnix()
	require.NoError(t, err)
	t.Cleanup(func() { p.Close() })
	return p
}

func TestLocalDesktopAdmissionSharesDeviceExclusionAndRevocation(t *testing.T) {
	svc, _ := testService(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	registration := desktophelper.Registration{Surface: surface, HelperID: "helper", SessionID: "os-session", Epoch: 1}
	owner, err := NewLocalDesktopAdmission(svc, private, "fake", surface, func(context.Context) (desktophelper.Registration, error) { return registration, nil })
	require.NoError(t, err)
	peer := admissionPeer(t)
	ctx := context.Background()
	_, _, err = owner.Acquire(ctx, nil, time.Minute, true)
	require.ErrorIs(t, err, sessions.ErrDesktopAdmission)
	ordinary, err := svc.Acquire("fake", "flow", time.Minute)
	require.NoError(t, err)
	_, _, err = owner.Acquire(ctx, peer, time.Minute, true)
	require.Error(t, err, "a desktop grant must not bypass a flow's lease")
	_, err = svc.Release(ordinary.ID)
	require.NoError(t, err)
	grant, token, err := owner.Acquire(ctx, peer, time.Minute, false)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	require.Equal(t, principal.String(), grant.Lease.Actor)
	require.Equal(t, uint64(2), grant.Lease.Epoch)
	require.Equal(t, "os-session", grant.Lease.Ref.DesktopSessionID)
	status, err := owner.GrantStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{grant.ID}, status.Active)
	require.LessOrEqual(t, status.ExpiresAt.Sub(status.ObservedAt), time.Second)
	concurrent, err := svc.Acquire("fake", "flow", time.Minute)
	require.NoError(t, err, "observation must not occupy control authority")
	_, err = svc.Release(concurrent.ID)
	require.NoError(t, err)
	authority, err := sessions.NewSignedDesktopAuthority(public, owner.Active)
	require.NoError(t, err)
	authenticated, err := authority.AuthenticatePeer(ctx, peer, token)
	require.NoError(t, err)
	require.NoError(t, authority.Authorize(authenticated, grant.Lease, "observe"))
	require.Error(t, authority.Authorize(authenticated, grant.Lease, "act"))
	require.Error(t, authority.Authorize(authenticated, grant.Lease, "transfer"))
	_, err = svc.Kill(grant.Lease.Ref.SessionID, "operator stop")
	require.NoError(t, err)
	active, err := owner.Active(ctx, grant.ID)
	require.NoError(t, err)
	require.False(t, active)
	status, err = owner.GrantStatus(ctx)
	require.NoError(t, err)
	require.Empty(t, status.Active)
	require.Error(t, authority.Authorize(authenticated, grant.Lease, "observe"))
	registration.Epoch = 7
	controlled, _, err := owner.Acquire(ctx, peer, time.Minute, true)
	require.NoError(t, err)
	require.Equal(t, uint64(8), controlled.Lease.Epoch)
	require.Contains(t, controlled.Operations, "act")
	registration.Surface.SurfaceID = "wrong"
	_, err = svc.Release(controlled.Lease.Ref.SessionID)
	require.NoError(t, err)
	_, _, err = owner.Acquire(ctx, peer, time.Minute, true)
	require.ErrorIs(t, err, sessions.ErrDesktopAdmission)
	require.Empty(t, svc.ListLiveSessions())
}
