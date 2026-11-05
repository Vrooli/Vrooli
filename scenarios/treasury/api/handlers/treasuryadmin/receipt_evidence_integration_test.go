package treasuryadmin_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/baseusdc"
	"treasury/internal/operatorauth"
	"treasury/internal/wallet"
)

type fixedReceiptVerifier struct{ transfers []baseusdc.Observation }

func (v fixedReceiptVerifier) VerifyWalletTransaction(context.Context, wallet.Wallet, string) ([]baseusdc.Observation, error) {
	return v.transfers, nil
}

func TestBaseReceiptEvidenceSurvivesOwnerHTTPAndDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "treasury.sqlite")
	transactionHash := "0x" + strings.Repeat("a", 64)
	blockHash := "0x" + strings.Repeat("b", 64)
	providerID := "fixture-base-rpc"
	checkedAt := time.Date(2026, 11, 2, 18, 30, 0, 0, time.UTC)
	transfer := baseusdc.Observation{
		EconomicKey:       "eip155:8453/erc20:" + baseusdc.USDCAddress + "/" + transactionHash + "/log/4/0x1111111111111111111111111111111111111111",
		InclusionRevision: transactionHash + "/log/4@" + blockHash,
		WalletID:          "wallet-1", Network: baseusdc.Network, AssetID: baseusdc.TransferAsset, Decimals: 6,
		From: "0x2222222222222222222222222222222222222222", To: "0x1111111111111111111111111111111111111111",
		AtomicAmount: "5901522", TransactionHash: transactionHash, LogIndex: 4, BlockNumber: 100,
		BlockHash: blockHash, BlockTime: checkedAt.Add(-time.Minute), Finality: "finalized",
		FinalizedHeadNumber: 100, FinalizedHeadHash: "0x" + strings.Repeat("c", 64),
		ProviderID: providerID, ObservedAt: checkedAt,
	}

	openOwner := func() (*sql.DB, *httptest.Server, authconnect.TreasuryAdminClient) {
		db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(1000)")
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(baseusdc.Schema)))
		authorizer, err := operatorauth.NewStaticToken("isolated-test-operator")
		require.NoError(t, err)
		service := baseusdc.NewService(baseusdc.NewSQLiteStore(db), fixedReceiptVerifier{transfers: []baseusdc.Observation{transfer}}, providerID, func() time.Time { return checkedAt })
		_, handler := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, &walletStub{network: baseusdc.Network}, service))
		server := httptest.NewServer(handler)
		return db, server, authconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	}
	db, server, client := openOwner()
	defer func() { server.Close(); _ = db.Close() }()
	verify := connect.NewRequest(&authv1.VerifyBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash})
	verify.Header().Set(operatorauth.HeaderOperatorToken, "isolated-test-operator")
	result, err := client.VerifyBaseUSDCReceipt(ctx, verify)
	require.NoError(t, err)
	require.NotEmpty(t, result.Msg.CheckId)
	require.Equal(t, "verified", result.Msg.CheckOutcome)
	require.Equal(t, "5901522", result.Msg.Transfers[0].AtomicAmount)

	server.Close()
	require.NoError(t, db.Close())

	freshDB, freshServer, freshClient := openOwner()
	defer func() { freshServer.Close(); _ = freshDB.Close() }()
	list := connect.NewRequest(&authv1.ListBaseUSDCReceiptChecksRequest{WalletId: "wallet-1", TransactionHash: transactionHash})
	list.Header().Set(operatorauth.HeaderOperatorToken, "isolated-test-operator")
	readback, err := freshClient.ListBaseUSDCReceiptChecks(ctx, list)
	require.NoError(t, err)
	require.Len(t, readback.Msg.Checks, 1)
	require.Equal(t, result.Msg.CheckId, readback.Msg.Checks[0].Id)
	require.Equal(t, transactionHash, readback.Msg.Checks[0].TransactionHash)
	require.Equal(t, "5901522", readback.Msg.Checks[0].Transfers[0].AtomicAmount)
	require.Equal(t, "finalized", readback.Msg.Checks[0].Transfers[0].Finality)
}
