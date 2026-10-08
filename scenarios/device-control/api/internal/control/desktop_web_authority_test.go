package control

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"device-control/internal/desktophelper"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
)

type desktopIdentityFixture struct {
	identity owneridentity.Identity
	err      error
}

func (f desktopIdentityFixture) Validate(context.Context, string) (owneridentity.Identity, error) {
	return f.identity, f.err
}

func TestForwardedDesktopIdentityRequiresPinScopesAndActorBinding(t *testing.T) {
	svc, _ := testService(t)
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	owner, err := NewLocalDesktopAdmission(svc, key, "fake", surface, func(context.Context) (desktophelper.Registration, error) {
		return desktophelper.Registration{Surface: surface, HelperID: "helper", SessionID: "login", Epoch: 1}, nil
	})
	require.NoError(t, err)
	identity := owneridentity.Identity{Subject: "operator", Scopes: []string{"device-control:read"}, ExpiresAt: time.Now().Add(20 * time.Second)}
	owner.webIdentity = desktopIdentityFixture{identity: identity}
	_, err = owner.authenticateForwarded(context.Background(), "Bearer token")
	require.Error(t, err, "valid account without a local pin is not authorized")
	owner.webSubject = "operator"
	ctx, err := owner.authenticateForwarded(context.Background(), "Bearer token")
	require.NoError(t, err)
	peer := admissionPeer(t)
	_, _, err = owner.Acquire(ctx, peer, time.Minute, true)
	require.Error(t, err, "read scope cannot authorize control")
	grant, _, err := owner.Acquire(ctx, peer, time.Minute, false)
	require.NoError(t, err)
	require.Equal(t, "operator", grant.Lease.Actor)
	require.False(t, grant.Lease.ExpiresAt.After(identity.ExpiresAt))
	require.True(t, owner.sessionActorAllowed(ctx, "operator", false))
	require.False(t, owner.sessionActorAllowed(context.Background(), "operator", false), "dropping bearer cannot inherit a web session")
	require.False(t, owner.sessionActorAllowed(ctx, "another-operator", false))
	_, err = svc.Release(grant.Lease.Ref.SessionID)
	require.NoError(t, err)
	identity.Scopes = append(identity.Scopes, "device-control:write")
	owner.webIdentity = desktopIdentityFixture{identity: identity}
	ctx, err = owner.authenticateForwarded(context.Background(), "Bearer token")
	require.NoError(t, err)
	_, _, err = owner.Acquire(ctx, peer, time.Minute, true)
	require.NoError(t, err)
	identity.Subject = "someone-else"
	owner.webIdentity = desktopIdentityFixture{identity: identity}
	_, err = owner.authenticateForwarded(context.Background(), "Bearer token")
	require.Error(t, err)
	owner.webIdentity = desktopIdentityFixture{err: owneridentity.ErrUnavailable}
	_, err = owner.authenticateForwarded(context.Background(), "Bearer token")
	require.Error(t, err, "provider failure never falls back to local peer authority")
}
