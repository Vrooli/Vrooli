package pairing_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
	localdb "vrooli-bridge/internal/database"
	"vrooli-bridge/internal/pairing"

	"github.com/vrooli/api-core/scheduletest"
)

func newPairingDB(t *testing.T) (*sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(pairing.Schema),
	))
	return d, clk
}

// fakeRegistrar records the facts it was asked to register and hands back
// sequential node ids.
type fakeRegistrar struct {
	calls []pairing.NodeFacts
	err   error
}

func (f *fakeRegistrar) RegisterNode(_ context.Context, facts pairing.NodeFacts) (string, error) {
	f.calls = append(f.calls, facts)
	if f.err != nil {
		return "", f.err
	}
	return "node-" + strconv.Itoa(len(f.calls)), nil
}

func newPubKeyB64(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(pub)
}

// [REQ:BRG-P0-002] One-touch bootstrap: an issued code redeems exactly once,
// registering the node (owner-assigned name/scopes win) and storing its key so
// the node can immediately authenticate.
func TestService_IssueAndRedeem(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	reg := &fakeRegistrar{}
	svc := pairing.NewService(repo, reg, clk)
	ctx := context.Background()

	issued, err := svc.IssueCode(ctx, "mac-mini", []string{"scenario test*"}, 0)
	require.NoError(t, err)
	require.NotEmpty(t, issued.Code)

	pub := newPubKeyB64(t)
	nodeID, err := svc.Redeem(ctx, issued.Code, pub, pairing.NodeFacts{
		Name: "node-proposed", OS: "darwin", Arch: "arm64", Capabilities: []string{"scenario test*"},
	})
	require.NoError(t, err)
	require.Equal(t, "node-1", nodeID)

	// Owner-assigned name/scopes from the code win over the node's proposal.
	require.Len(t, reg.calls, 1)
	require.Equal(t, "mac-mini", reg.calls[0].Name)
	require.Equal(t, []string{"scenario test*"}, reg.calls[0].Scopes)

	// The node's key is now active and verifiable.
	gotPub, ok, err := repo.ActivePublicKey(ctx, nodeID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, pub, base64.StdEncoding.EncodeToString(gotPub))
}

// [REQ:BRG-P0-002] A code is single-use: the second redeem is rejected and does
// NOT register a second node (no rogue enrolment by replay).
func TestService_RedeemTwiceRejected(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	reg := &fakeRegistrar{}
	svc := pairing.NewService(repo, reg, clk)
	ctx := context.Background()

	issued, err := svc.IssueCode(ctx, "", nil, 0)
	require.NoError(t, err)

	_, err = svc.Redeem(ctx, issued.Code, newPubKeyB64(t), pairing.NodeFacts{})
	require.NoError(t, err)

	_, err = svc.Redeem(ctx, issued.Code, newPubKeyB64(t), pairing.NodeFacts{})
	require.ErrorIs(t, err, pairing.ErrCodeUsed)
	require.Len(t, reg.calls, 1, "the second redeem must not register a node")
}

// [REQ:BRG-P0-002] An expired code is burned by TTL: redeem after expiry fails.
func TestService_RedeemExpired(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	svc := pairing.NewService(repo, &fakeRegistrar{}, clk)
	ctx := context.Background()

	issued, err := svc.IssueCode(ctx, "", nil, time.Minute)
	require.NoError(t, err)

	clk.Advance(2 * time.Minute)
	_, err = svc.Redeem(ctx, issued.Code, newPubKeyB64(t), pairing.NodeFacts{})
	require.ErrorIs(t, err, pairing.ErrCodeExpired)
}

// [REQ:BRG-P0-002] An unknown code is rejected (a rogue node cannot guess one).
func TestService_RedeemUnknownCode(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	svc := pairing.NewService(repo, &fakeRegistrar{}, clk)

	_, err := svc.Redeem(context.Background(), "TOTALLYNOTAREALCODE", newPubKeyB64(t), pairing.NodeFacts{})
	require.ErrorIs(t, err, pairing.ErrCodeNotFound)
}

// [REQ:BRG-P0-002] A malformed node public key is rejected before any node is
// registered.
func TestService_RedeemRejectsBadKey(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	reg := &fakeRegistrar{}
	svc := pairing.NewService(repo, reg, clk)

	issued, err := svc.IssueCode(context.Background(), "", nil, 0)
	require.NoError(t, err)
	_, err = svc.Redeem(context.Background(), issued.Code, "not-base64!!", pairing.NodeFacts{})
	var invalid pairing.ErrInvalid
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "node_public_key", invalid.Field)
	require.Empty(t, reg.calls)
}

// [REQ:BRG-P0-002] Request/approve fallback: a pending request mints a node only
// on owner approval, and storing its credential makes it active.
func TestService_RequestApprove(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	reg := &fakeRegistrar{}
	svc := pairing.NewService(repo, reg, clk)
	ctx := context.Background()

	pub := newPubKeyB64(t)
	req, err := svc.RequestPairing(ctx, pub, pairing.NodeFacts{Name: "ci-box", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)
	require.Equal(t, pairing.RequestPending, req.Status)

	pending, err := svc.ListRequests(ctx, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	status, nodeID, err := svc.Approve(ctx, req.ID, true, []string{"scenario test*"})
	require.NoError(t, err)
	require.Equal(t, pairing.RequestApproved, status)
	require.Equal(t, "node-1", nodeID)

	_, ok, err := repo.ActivePublicKey(ctx, nodeID)
	require.NoError(t, err)
	require.True(t, ok)

	// Re-approving an already-decided request is rejected.
	_, _, err = svc.Approve(ctx, req.ID, true, nil)
	require.ErrorIs(t, err, pairing.ErrRequestDecided)
}

// [REQ:BRG-P0-002] Rejecting a request mints no node.
func TestService_RequestReject(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	reg := &fakeRegistrar{}
	svc := pairing.NewService(repo, reg, clk)
	ctx := context.Background()

	req, err := svc.RequestPairing(ctx, newPubKeyB64(t), pairing.NodeFacts{Name: "stranger"})
	require.NoError(t, err)

	status, nodeID, err := svc.Approve(ctx, req.ID, false, nil)
	require.NoError(t, err)
	require.Equal(t, pairing.RequestRejected, status)
	require.Empty(t, nodeID)
	require.Empty(t, reg.calls)
}

// [REQ:BRG-P0-002] Revoking a credential makes it inactive: nodeauth (and the
// ActivePublicKey lookup) treat the node as having no credential.
func TestService_RevokeCredential(t *testing.T) {
	d, clk := newPairingDB(t)
	repo := pairing.NewSQLiteRepository(d, clk)
	svc := pairing.NewService(repo, &fakeRegistrar{}, clk)
	ctx := context.Background()

	issued, err := svc.IssueCode(ctx, "", nil, 0)
	require.NoError(t, err)
	nodeID, err := svc.Redeem(ctx, issued.Code, newPubKeyB64(t), pairing.NodeFacts{})
	require.NoError(t, err)

	_, ok, _ := repo.ActivePublicKey(ctx, nodeID)
	require.True(t, ok)

	require.NoError(t, svc.RevokeCredential(ctx, nodeID))

	_, ok, err = repo.ActivePublicKey(ctx, nodeID)
	require.NoError(t, err)
	require.False(t, ok, "a revoked credential is no longer active")

	// Idempotent.
	require.NoError(t, svc.RevokeCredential(ctx, nodeID))
}
