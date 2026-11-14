package ingest_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"

	ingestion "money-ledger/handlers/ingest"
	"money-ledger/internal/ledger"
)

func TestCryptoAcceptanceUsesVerifiedTreasuryServiceIdentity(t *testing.T) { // [REQ:CRYP-001]
	ctx := context.Background()
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	journal := ledger.NewStore(db, nil)
	require.NoError(t, database.EnsureSchemas(ctx, db.Primary(), journal))
	book, err := journal.CreateBook(ctx, "Authenticated rehearsal", "USD")
	require.NoError(t, err)
	account, err := journal.CreateAccount(ctx, book.Id, "USDC", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "fixture", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer keys.Close()
	t.Setenv("VROOLI_AUTH_PROVIDERS", "scenario_authenticator")
	t.Setenv("VROOLI_AUTH_SCENARIO_JWKS_URL", keys.URL)
	t.Setenv("VROOLI_AUTH_SCENARIO_AUDIENCE", "scenario-authenticator:money-ledger")
	t.Setenv("VROOLI_AUTH_SCENARIO_ISSUER", "scenario-authenticator")
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", "[]")
	router := mux.NewRouter()
	ingestion.Module(db, nil, log.New(io.Discard, "", 0)).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := rpc.NewIngestServiceClient(server.Client(), server.URL)
	sign := func(subject, kind, scope, audience string, expiry time.Time) string {
		header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "fixture", "typ": "JWT"})
		claims, _ := json.Marshal(map[string]any{"sub": subject, "user_id": subject, "actor_kind": kind, "iss": "scenario-authenticator", "aud": audience, "scope": []string{scope}, "exp": expiry.Unix(), "iat": time.Now().Add(-time.Minute).Unix()})
		unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
		digest := sha256.Sum256([]byte(unsigned))
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		require.NoError(t, err)
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
	}
	at := timestamppb.New(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC))
	in := &pb.AcceptCryptoReceiptRequest{
		Observation: &pb.PreviewCryptoObservationRequest{Network: "eip155:8453", AssetId: "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", AtomicAmount: "1", Decimals: 6, State: pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED, LegKind: pb.CryptoLegKind_CRYPTO_LEG_KIND_DEPOSIT, SourceReference: "fixture:check", TransactionHash: "0x" + strings.Repeat("a", 64), BlockHash: "0x" + strings.Repeat("b", 64), BookId: book.Id, AccountId: account.Id, Address: "0x" + strings.Repeat("1", 40)},
		Source:      &pb.TreasuryReceiptLink{CheckId: "auth-check", WalletId: "test-wallet", OperatorIdentity: "test-operator", SourceReference: "fixture:check"},
		Action:      pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
		Valuation:   &pb.CryptoValuation{AmountMinor: 0, Currency: "USD", BasisReference: "operator:explicit-subcent-value", ValuedAt: at, Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED},
		Attribution: &pb.CryptoPayerAttribution{PayerAddress: "0x" + strings.Repeat("2", 40)}, OccurredAt: at,
	}
	valid := sign("treasury", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour))
	for _, test := range []struct {
		name, token string
		code        connect.Code
	}{
		{"missing", "", connect.CodeUnauthenticated},
		{"forged", "not-a-signed-token", connect.CodeUnauthenticated},
		{"human", sign("treasury", "human", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodeUnauthenticated},
		{"other-service", sign("other", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"missing-capability", sign("treasury", "service", "money-ledger:read", "scenario-authenticator:money-ledger", time.Now().Add(time.Hour)), connect.CodePermissionDenied},
		{"other-audience", sign("treasury", "service", "money-ledger:write", "scenario-authenticator:treasury", time.Now().Add(time.Hour)), connect.CodeUnauthenticated},
		{"expired", sign("treasury", "service", "money-ledger:write", "scenario-authenticator:money-ledger", time.Now().Add(-time.Hour)), connect.CodeUnauthenticated},
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
	require.Equal(t, "treasury", accepted.Msg.Receipt.AcceptedBy)
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
}
