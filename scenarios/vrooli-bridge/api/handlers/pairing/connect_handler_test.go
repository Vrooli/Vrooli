package pairing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	internalpairing "vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
)

type fakeRegistrar struct{ n int }

func (f *fakeRegistrar) RegisterNode(_ context.Context, _ internalpairing.NodeFacts) (string, error) {
	f.n++
	return "node-1", nil
}

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalpairing.Schema),
	))
	repo := internalpairing.NewSQLiteRepository(d, clk)
	svc := internalpairing.NewService(repo, &fakeRegistrar{}, clk)
	return NewConnectHandler(Deps{Service: svc, ControlPlanePublicKey: "CP-PUBKEY"})
}

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newPubKeyB64(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(pub)
}

// [REQ:BRG-P0-002] Issuing a code is owner-gated: without an owner identity it
// fails closed.
func TestHandler_IssueRequiresOwner(t *testing.T) {
	h := newHandler(t)
	_, err := h.IssuePairingCode(context.Background(), connect.NewRequest(&pairingv1.IssuePairingCodeRequest{Name: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-002] ApprovePairing and ListPairingRequests are owner-gated.
func TestHandler_ApproveAndListRequireOwner(t *testing.T) {
	h := newHandler(t)
	_, err := h.ApprovePairing(context.Background(), connect.NewRequest(&pairingv1.ApprovePairingRequest{RequestId: "r"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListPairingRequests(context.Background(), connect.NewRequest(&pairingv1.ListPairingRequestsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-002] One-touch bootstrap end-to-end through the handler: the owner
// issues a code (and learns the CP key to pin), the node redeems it OPEN (no
// owner token) and gets its node id + the CP key.
func TestHandler_IssueThenRedeem(t *testing.T) {
	h := newHandler(t)

	issued, err := h.IssuePairingCode(ownerCtx(), connect.NewRequest(&pairingv1.IssuePairingCodeRequest{
		Name: "mac-mini", Scopes: []string{"scenario test*"},
	}))
	require.NoError(t, err)
	require.NotEmpty(t, issued.Msg.Code)
	require.Equal(t, "CP-PUBKEY", issued.Msg.ControlPlanePublicKey)

	// Redeem is open — no owner identity in context.
	redeemed, err := h.RedeemPairingCode(context.Background(), connect.NewRequest(&pairingv1.RedeemPairingCodeRequest{
		Code: issued.Msg.Code, NodePublicKey: newPubKeyB64(t), Os: "darwin", Arch: "arm64",
	}))
	require.NoError(t, err)
	require.Equal(t, "node-1", redeemed.Msg.NodeId)
	require.Equal(t, "CP-PUBKEY", redeemed.Msg.ControlPlanePublicKey)
}

// [REQ:BRG-P0-002] An unknown/used code maps to Unauthenticated (a node failing
// to authenticate), not leaking which part was wrong.
func TestHandler_RedeemUnknownCodeUnauthenticated(t *testing.T) {
	h := newHandler(t)
	_, err := h.RedeemPairingCode(context.Background(), connect.NewRequest(&pairingv1.RedeemPairingCodeRequest{
		Code: "NOPE", NodePublicKey: newPubKeyB64(t),
	}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
