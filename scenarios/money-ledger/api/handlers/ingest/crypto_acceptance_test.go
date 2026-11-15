package ingest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"

	"money-ledger/internal/testutil/ledgertest"
)

func TestCryptoAcceptanceUsesVerifiedTreasuryServiceIdentity(t *testing.T) { // [REQ:CRYP-001]
	ctx := context.Background()
	l := ledgertest.New(t)
	db, client, sign := l.DB, l.Ingest, func(subject, kind, scope, audience string, expiry time.Time) string {
		return l.Identity.Token(subject, kind, audience, expiry, scope)
	}
	book, account := l.Book, l.Account
	at := timestamppb.New(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC))
	in := &pb.AcceptCryptoReceiptRequest{
		Observation: &pb.PreviewCryptoObservationRequest{Network: "eip155:8453", AssetId: "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", AtomicAmount: "1", Decimals: 6, State: pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED, LegKind: pb.CryptoLegKind_CRYPTO_LEG_KIND_DEPOSIT, SourceReference: "fixture:check", TransactionHash: "0x" + strings.Repeat("a", 64), BlockHash: "0x" + strings.Repeat("b", 64), BookId: book.Id, AccountId: account.Id, Address: "0x" + strings.Repeat("1", 40)},
		Source:      &pb.TreasuryReceiptLink{CheckId: "auth-check", WalletId: "test-wallet", OperatorIdentity: "test-operator", SourceReference: "fixture:check"},
		Action:      pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
		Valuation:   &pb.CryptoValuation{AmountMinor: 0, Currency: "USD", BasisReference: "operator:explicit-subcent-value", ValuedAt: at, Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED},
		Attribution: &pb.CryptoPayerAttribution{PayerAddress: "0x" + strings.Repeat("2", 40)}, OccurredAt: at,
	}
	valid := sign("treasury-money-ledger", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour))
	for _, test := range []struct {
		name, token string
		code        connect.Code
	}{
		{"missing", "", connect.CodeUnauthenticated},
		{"forged", "not-a-signed-token", connect.CodeUnauthenticated},
		{"human", sign("treasury-money-ledger", "human", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		// The owner may read receipts but never accepts one: Treasury owns the
		// chain evidence behind every receipt.
		{"owner", l.Identity.Owner(), connect.CodePermissionDenied},
		// Treasury's principal at another peer, and Treasury's name before it held one
		// per peer, are not Treasury's principal at Money Ledger.
		{"treasury-principal-for-another-peer", sign("treasury-agent-manager", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"unsplit-treasury-subject", sign("treasury", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"other-service", sign("other", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"missing-capability", sign("treasury-money-ledger", "service", "money-ledger:read", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"other-audience", sign("treasury-money-ledger", "service", "money-ledger:write", "scenario-authenticator:treasury", time.Now().Add(time.Hour)), connect.CodeUnauthenticated},
		{"expired", sign("treasury-money-ledger", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(-time.Hour)), connect.CodeUnauthenticated},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := connect.NewRequest(in)
			if test.token != "" {
				request.Header().Set("Authorization", "Bearer "+test.token)
			}
			_, err := client.AcceptCryptoReceipt(ctx, request)
			require.Equal(t, test.code, connect.CodeOf(err))
		})
	}
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings`).Scan(&count))
	require.Zero(t, count)
	request := connect.NewRequest(in)
	request.Header().Set("Authorization", "Bearer "+valid)
	accepted, err := client.AcceptCryptoReceipt(ctx, request)
	require.NoError(t, err)
	require.Equal(t, "treasury", accepted.Msg.Receipt.AcceptedBy, "the journal names the actor treasury, not the principal that presented the token")
	require.Equal(t, "1", accepted.Msg.Receipt.AtomicDelta)
	require.Zero(t, accepted.Msg.Receipt.ValuationDeltaMinor, "one atomic token unit cannot become one fiat minor unit")
	_, err = client.GetCryptoReceipt(ctx, connect.NewRequest(&pb.GetCryptoReceiptRequest{Id: accepted.Msg.Receipt.Id}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	read := connect.NewRequest(&pb.GetCryptoReceiptRequest{Id: accepted.Msg.Receipt.Id})
	read.Header().Set("Authorization", "Bearer "+valid)
	stored, err := client.GetCryptoReceipt(ctx, read)
	require.NoError(t, err)
	require.Equal(t, accepted.Msg.Receipt.Id, stored.Msg.Receipt.Id)
	list := connect.NewRequest(&pb.ListCryptoReceiptsRequest{BookId: book.Id})
	list.Header().Set("Authorization", "Bearer "+valid)
	listed, err := client.ListCryptoReceipts(ctx, list)
	require.NoError(t, err)
	require.Len(t, listed.Msg.Receipts, 1)
	ownerRead := connect.NewRequest(&pb.GetCryptoReceiptRequest{Id: accepted.Msg.Receipt.Id})
	ownerRead.Header().Set("Authorization", "Bearer "+l.Identity.Owner())
	_, err = client.GetCryptoReceipt(ctx, ownerRead)
	require.NoError(t, err, "the owner reads the receipts accepted into their books")
}
