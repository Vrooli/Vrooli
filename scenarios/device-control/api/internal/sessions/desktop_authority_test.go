package sessions

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
)

func desktopPeer(t *testing.T) *net.UnixConn {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix peer credential acceptance requires Linux or macOS")
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(t.TempDir(), "peer.sock"), Net: "unix"})
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	peer, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
	require.NoError(t, err)
	t.Cleanup(func() { peer.Close() })
	accepted, err := listener.AcceptUnix()
	require.NoError(t, err)
	t.Cleanup(func() { accepted.Close() })
	return accepted
}

func TestSignedDesktopAuthorityUsesKernelPeerAndRechecksRevocation(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-09]
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix peer credentials require Linux or macOS")
	}
	c, _, native, _, lease := desktopFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	now := time.Now().UTC()
	grant := DesktopGrant{ID: "grant-1", Principal: principal, Lease: lease, Operations: []string{"act", "stop"}, IssuedAt: now.Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, now)
	require.NoError(t, err)
	active := true
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return active, nil })
	require.NoError(t, err)
	ctx, err := authority.AuthenticatePeer(context.Background(), desktopPeer(t), token)
	require.NoError(t, err)
	require.NoError(t, authority.Authorize(ctx, lease, "act"))
	c.authority = authority
	receipt, err := c.Act(ctx, desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Outcome)
	require.Equal(t, 1, native.effects)
	require.ErrorIs(t, authority.Authorize(context.Background(), lease, "act"), ErrDesktopAdmission)
	require.ErrorIs(t, authority.Authorize(ctx, lease, "transfer"), ErrDesktopAdmission)
	wrong := lease
	wrong.Ref.Surface.Target.ResourceID = "wrong"
	require.ErrorIs(t, authority.Authorize(ctx, wrong, "act"), ErrDesktopAdmission)
	wrong = lease
	wrong.Epoch++
	require.ErrorIs(t, authority.Authorize(ctx, wrong, "act"), ErrDesktopAdmission)
	active = false
	command := desktopCommand(lease)
	command.ID = "after-revocation"
	_, err = c.Act(ctx, command)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Equal(t, 1, native.effects)
	require.ErrorIs(t, authority.Authorize(ctx, lease, "act"), ErrDesktopAdmission)
	active = true
	authority.now = func() time.Time { return grant.ExpiresAt }
	require.ErrorIs(t, authority.Authorize(ctx, lease, "act"), ErrDesktopAdmission)
}

func TestDesktopGrantRejectsForgedPeerSignatureAndProtocol(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix peer credentials require Linux or macOS")
	}
	_, _, _, _, lease := desktopFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	now := time.Now().UTC()
	grant := DesktopGrant{ID: "grant-1", Principal: principal, Lease: lease, Operations: []string{"act"}, IssuedAt: now, ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, now)
	require.NoError(t, err)
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	peer := desktopPeer(t)
	grant.Principal = "unix:4294967295"
	wrongPeer, err := SignDesktopGrant(private, grant, now)
	require.NoError(t, err)
	_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	forged, err := SignDesktopGrant(otherPrivate, grant, now)
	require.NoError(t, err)
	for _, candidate := range []string{wrongPeer, forged, strings.Replace(token, "DCG1.", "OS1.", 1), token + ".extra", strings.Repeat("x", 17*1024)} {
		_, err := authority.AuthenticatePeer(context.Background(), peer, candidate)
		require.ErrorIs(t, err, ErrDesktopAdmission)
	}
	authority.active = func(context.Context, string) (bool, error) {
		return false, errors.New("revocation service unavailable")
	}
	_, err = authority.AuthenticatePeer(context.Background(), peer, token)
	require.ErrorIs(t, err, ErrDesktopAdmission)
}

func TestSignedGrantCannotAdvanceBeyondAuthorizedEpoch(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix peer credentials require Linux or macOS")
	}
	c, _, native, _, lease := desktopFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	now := time.Now().UTC()
	grant := DesktopGrant{ID: "grant-1", Principal: principal, Lease: lease, Operations: []string{"transfer"}, IssuedAt: now, ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, now)
	require.NoError(t, err)
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	ctx, err := authority.AuthenticatePeer(context.Background(), desktopPeer(t), token)
	require.NoError(t, err)
	c.authority = authority
	_, err = c.Open(ctx, lease, lease.Epoch, true)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
	// A separately authorized successor epoch can take control.
	grant.Lease.Epoch++
	token, err = SignDesktopGrant(private, grant, now)
	require.NoError(t, err)
	ctx, err = authority.AuthenticatePeer(context.Background(), desktopPeer(t), token)
	require.NoError(t, err)
	successor, err := c.Open(ctx, grant.Lease, lease.Epoch, true)
	require.NoError(t, err)
	require.Equal(t, grant.Lease.Epoch, successor.Epoch)
}

func TestCleanupReadAfterExpiryNeverRestoresInputAuthority(t *testing.T) {
	c, repo, native, _, lease := desktopFixture(t)
	require.NoError(t, c.Stop(context.Background(), lease))
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	issued := time.Now().Add(-time.Second)
	grant := DesktopGrant{ID: "history", Principal: principal, Lease: lease, Operations: []string{"act", "stop"}, IssuedAt: issued, ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, issued)
	require.NoError(t, err)
	checks := 0
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { checks++; return false, errors.New("revoked") })
	require.NoError(t, err)
	authority.now = func() time.Time { return grant.ExpiresAt.Add(time.Hour) }
	peer := desktopPeer(t)
	ctx, err := authority.AuthenticateCleanupPeer(context.Background(), peer, token)
	require.NoError(t, err)
	c.authority = authority
	receipt, err := c.ReadCleanup(ctx, lease)
	require.NoError(t, err)
	require.True(t, receipt.Released)
	require.Zero(t, checks)
	require.ErrorIs(t, authority.Authorize(ctx, lease, "act"), ErrDesktopAdmission)
	require.ErrorIs(t, authority.Authorize(ctx, lease, "stop"), ErrDesktopAdmission)
	_, err = c.Act(ctx, desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.ErrorIs(t, c.Stop(ctx, lease), ErrDesktopAdmission)
	wrong := lease
	wrong.Actor = "other"
	_, err = c.ReadCleanup(ctx, wrong)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	_, err = authority.AuthenticatePeer(context.Background(), peer, token)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	for _, bad := range []string{token + ".extra", strings.Replace(token, "DCG1.", "BAD.", 1), strings.Repeat("x", 17*1024)} {
		_, err = authority.AuthenticateCleanupPeer(context.Background(), peer, bad)
		require.ErrorIs(t, err, ErrDesktopAdmission)
	}
	grant.Principal = "unix:4294967295"
	wrongPeer, err := SignDesktopGrant(private, grant, issued)
	require.NoError(t, err)
	_, err = authority.AuthenticateCleanupPeer(context.Background(), peer, wrongPeer)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
	authority.now = func() time.Time { return issued }
	authority.active = func(context.Context, string) (bool, error) { return true, nil }
	require.ErrorIs(t, authority.Authorize(ctx, lease, "act"), ErrDesktopAdmission)
	require.ErrorIs(t, authority.Authorize(ctx, lease, "stop"), ErrDesktopAdmission)
	// Read-only lookup still has exactly the same stored evidence.
	stored, err := repo.ReadCleanup(context.Background(), lease)
	require.NoError(t, err)
	require.Equal(t, receipt, stored)
}

func TestSignedRevocationProvesUnadmittedLeaseAndFencesStalePublication(t *testing.T) {
	ctx := context.Background()
	c, repo, native, _, old := desktopFixture(t)
	require.NoError(t, c.Stop(ctx, old))
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	lease := old
	lease.Ref.SessionID = "never-admitted"
	lease.Epoch++
	grant := DesktopGrant{ID: "never-admitted-grant", Principal: principal, Lease: lease, Operations: []string{"open", "act", "stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	// Even a stale publisher claiming active cannot revive a persisted revocation.
	published := false
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return published, nil })
	require.NoError(t, err)
	c.authority = authority
	token, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	peer := desktopPeer(t)
	ordinaryRead, err := authority.AuthenticateCleanupPeer(ctx, peer, token)
	require.NoError(t, err)
	_, err = c.ReadCleanup(ordinaryRead, lease)
	require.Error(t, err, "ordinary historical credential cannot prove permanent revocation")
	grant.Revoked = true
	proof, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	_, err = authority.AuthenticatePeer(ctx, peer, proof)
	require.ErrorIs(t, err, ErrDesktopAdmission, "revocation statement must never grant input")
	revokedRead, err := authority.AuthenticateCleanupPeer(ctx, peer, proof)
	require.NoError(t, err)
	_, err = repo.db.ExecContext(ctx, `CREATE TRIGGER reject_revocation BEFORE INSERT ON device_control_desktop_revocations BEGIN SELECT RAISE(FAIL,'cannot persist revocation'); END`)
	require.NoError(t, err)
	_, err = c.ReadCleanup(revokedRead, lease)
	require.Error(t, err, "an uncommitted revocation cannot establish absence")
	_, err = repo.db.ExecContext(ctx, `DROP TRIGGER reject_revocation`)
	require.NoError(t, err)
	receipt, err := c.ReadCleanup(revokedRead, lease)
	require.NoError(t, err)
	require.True(t, receipt.Released)
	require.True(t, sameDesktopLease(receipt.Lease, lease))
	require.Zero(t, native.effects)
	recovered, err := NewSQLiteDesktopRepository(ctx, repo.db, repo.destination)
	require.NoError(t, err)
	c.repo = recovered
	published = true
	original, err := authority.AuthenticatePeer(ctx, peer, token)
	require.NoError(t, err)
	_, err = c.Open(original, lease, old.Epoch, false)
	require.ErrorIs(t, err, ErrDesktopAdmission, "delayed Open cannot cross a durable revocation fence")
	require.NoError(t, repo.Update(ctx, func(state *DesktopState) error {
		require.Nil(t, state.Lease)
		require.Equal(t, old.Epoch, state.Epoch)
		return nil
	}))
}

func TestRevocationOfAdmittedLeaseRequiresSuccessfulDestinationCleanup(t *testing.T) {
	ctx := context.Background()
	c, _, native, _, old := desktopFixture(t)
	require.NoError(t, c.Stop(ctx, old))
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	lease := old
	lease.Ref.SessionID = "admitted-before-revocation"
	lease.Epoch++
	grant := DesktopGrant{ID: "admitted-grant", Principal: principal, Lease: lease, Operations: []string{"open", "act", "stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	c.authority = authority
	peer := desktopPeer(t)
	token, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	active, err := authority.AuthenticatePeer(ctx, peer, token)
	require.NoError(t, err)
	_, err = c.Open(active, lease, old.Epoch, false)
	require.NoError(t, err)
	grant.Revoked = true
	proof, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	historical, err := authority.AuthenticateCleanupPeer(ctx, peer, proof)
	require.NoError(t, err)
	pending, err := c.ReadCleanup(historical, lease)
	require.NoError(t, err)
	require.False(t, pending.Released, "live lease cannot be mistaken for never-admitted")
	_, err = c.Act(active, desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	native.releaseFail = true
	require.Error(t, c.ReapExpired(ctx))
	pending, err = c.ReadCleanup(historical, lease)
	require.NoError(t, err)
	require.False(t, pending.Released)
	native.releaseFail = false
	require.NoError(t, c.ReapExpired(ctx))
	released, err := c.ReadCleanup(historical, lease)
	require.NoError(t, err)
	require.True(t, released.Released)
	require.Zero(t, native.effects)
}

type blockedAdmissionNative struct {
	DesktopNative
	entered chan struct{}
	release chan struct{}
}

func (n *blockedAdmissionNative) ReleaseHeld(ctx context.Context) error {
	close(n.entered)
	select {
	case <-n.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestRevokedAbsenceWaitsForInFlightAdmission(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, native, _, old := desktopFixture(t)
	require.NoError(t, c.Stop(ctx, old))
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	lease := old
	lease.Ref.SessionID = "in-flight-open"
	lease.Epoch++
	grant := DesktopGrant{ID: "in-flight-grant", Principal: principal, Lease: lease, Operations: []string{"open", "stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	c.authority = authority
	gate := &blockedAdmissionNative{DesktopNative: native, entered: make(chan struct{}), release: make(chan struct{})}
	c.native = gate
	token, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	peer := desktopPeer(t)
	active, err := authority.AuthenticatePeer(ctx, peer, token)
	require.NoError(t, err)
	opened := make(chan error, 1)
	go func() { _, err := c.Open(active, lease, old.Epoch, false); opened <- err }()
	select {
	case <-gate.entered:
	case <-ctx.Done():
		t.Fatal("Open did not enter admission boundary")
	}
	grant.Revoked = true
	proof, err := SignDesktopGrant(private, grant, grant.IssuedAt)
	require.NoError(t, err)
	historical, err := authority.AuthenticateCleanupPeer(ctx, peer, proof)
	require.NoError(t, err)
	type result struct {
		receipt DesktopCleanupReceipt
		err     error
	}
	read := make(chan result, 1)
	go func() { receipt, err := c.ReadCleanup(historical, lease); read <- result{receipt, err} }()
	select {
	case <-read:
		t.Fatal("absence proof escaped in-flight admission")
	case <-time.After(30 * time.Millisecond):
	}
	close(gate.release)
	require.NoError(t, <-opened)
	outcome := <-read
	require.NoError(t, outcome.err)
	require.False(t, outcome.receipt.Released, "committed lease must remain pending")
	require.Zero(t, native.effects)
}
