package treasuryadmin_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
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
	walletv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/baseusdc"
	"treasury/internal/operatorauth"
	"treasury/internal/wallet"
)

type fixedReceiptVerifier struct{ transfers []baseusdc.Observation }

func (v fixedReceiptVerifier) VerifyWalletTransaction(context.Context, wallet.Wallet, string) ([]baseusdc.Observation, error) {
	return v.transfers, nil
}

type receiptResult struct {
	transfers []baseusdc.Observation
	err       error
}

type sequenceReceiptVerifier struct {
	results []receiptResult
	calls   int
}

func (v *sequenceReceiptVerifier) VerifyWalletTransaction(context.Context, wallet.Wallet, string) ([]baseusdc.Observation, error) {
	v.calls++
	if v.calls > len(v.results) {
		return nil, errors.New("unexpected synthetic receipt read")
	}
	result := v.results[v.calls-1]
	return result.transfers, result.err
}

type failSecondReceiptAppendStore struct {
	baseusdc.Store
	appendCalls int
}

func (s *failSecondReceiptAppendStore) Append(ctx context.Context, check baseusdc.Check) error {
	s.appendCalls++
	if s.appendCalls == 2 {
		return errors.New("synthetic receipt persistence failure")
	}
	return s.Store.Append(ctx, check)
}

type subjectAuthorizer struct{}

func (subjectAuthorizer) Authorize(_ context.Context, headers http.Header) (operatorauth.Identity, error) {
	switch headers.Get(operatorauth.HeaderOperatorToken) {
	case "operator-a-token":
		return operatorauth.Identity{Subject: "operator-a"}, nil
	case "operator-b-token":
		return operatorauth.Identity{Subject: "operator-b"}, nil
	default:
		return operatorauth.Identity{}, operatorauth.ErrDenied
	}
}

type scopedReceiptWallets struct{ owners map[string]string }

func (w *scopedReceiptWallets) Create(context.Context, wallet.CreateInput) (wallet.Wallet, error) {
	return wallet.Wallet{}, errors.New("not used")
}

func (w *scopedReceiptWallets) Get(_ context.Context, id, network, operator string) (wallet.Wallet, error) {
	owner, ok := w.owners[id]
	if !ok || network != baseusdc.Network {
		return wallet.Wallet{}, wallet.ErrNotFound
	}
	if owner != operator {
		return wallet.Wallet{}, wallet.ErrMismatch
	}
	return wallet.Wallet{ID: id, OperatorIdentity: owner, Network: network, Address: "0x1111111111111111111111111111111111111111", CustodyReference: "fixture:receive-only"}, nil
}

func (w *scopedReceiptWallets) Prove(context.Context, string, string, string, string) (wallet.Proof, error) {
	return wallet.Proof{}, errors.New("not used")
}

func (w *scopedReceiptWallets) QualifyRecovery(context.Context, string, string, string) (wallet.Wallet, error) {
	return wallet.Wallet{}, errors.New("not used")
}

func (w *scopedReceiptWallets) Recover(context.Context, string, string, string) (wallet.Wallet, error) {
	return wallet.Wallet{}, errors.New("not used")
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

func TestAuthenticatedRecheckLinksReinclusionAndProviderReportedNoncanonicalAcrossReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "treasury-recheck.sqlite")
	transactionHash := "0x" + strings.Repeat("a", 64)
	firstBlock := "0x" + strings.Repeat("b", 64)
	secondBlock := "0x" + strings.Repeat("d", 64)
	providerID := "fixture-base-rpc"
	checkedAt := time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
	serviceClock := checkedAt
	wallets := &scopedReceiptWallets{owners: map[string]string{"wallet-1": "operator-a", "wallet-2": "operator-a"}}
	verifier := &sequenceReceiptVerifier{results: []receiptResult{
		{transfers: []baseusdc.Observation{receiptObservation("wallet-1", transactionHash, firstBlock, providerID, checkedAt)}},
		{transfers: []baseusdc.Observation{receiptObservation("wallet-1", transactionHash, secondBlock, providerID, checkedAt.Add(time.Second))}},
		{err: baseusdc.ErrNotCanonical},
	}}

	openOwner := func(v baseusdc.Verification) (*sql.DB, *httptest.Server, authconnect.TreasuryAdminClient) {
		db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(1000)")
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(baseusdc.Schema)))
		service := baseusdc.NewService(baseusdc.NewSQLiteStore(db), v, providerID, func() time.Time {
			serviceClock = serviceClock.Add(time.Second)
			return serviceClock
		})
		_, handler := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(subjectAuthorizer{}, wallets, service))
		server := httptest.NewServer(handler)
		return db, server, authconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	}
	authorized := func(token string) *connect.Request[authv1.VerifyBaseUSDCReceiptRequest] {
		request := connect.NewRequest(&authv1.VerifyBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash})
		request.Header().Set(operatorauth.HeaderOperatorToken, token)
		return request
	}
	db, server, client := openOwner(verifier)
	unauthenticated := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash, PriorCheckId: "synthetic-prior"})
	_, err := client.RecheckBaseUSDCReceipt(ctx, unauthenticated)
	require.Error(t, err)
	require.Zero(t, verifier.calls, "unauthenticated recheck must not query the provider")
	initial, err := client.VerifyBaseUSDCReceipt(ctx, authorized("operator-a-token"))
	require.NoError(t, err)
	require.Equal(t, "verified", initial.Msg.CheckOutcome)
	priorID := initial.Msg.CheckId

	recheck := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash, PriorCheckId: priorID})
	recheck.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	reincluded, err := client.RecheckBaseUSDCReceipt(ctx, recheck)
	require.NoError(t, err)
	require.Equal(t, priorID, reincluded.Msg.Check.PriorCheckId)
	require.Equal(t, walletv1.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION, reincluded.Msg.Check.Relation)
	require.Equal(t, "verified", reincluded.Msg.Check.Outcome)
	require.Equal(t, initial.Msg.Transfers[0].EconomicKey, reincluded.Msg.Check.Transfers[0].EconomicKey)
	require.NotEqual(t, initial.Msg.Transfers[0].InclusionRevision, reincluded.Msg.Check.Transfers[0].InclusionRevision)

	badWallet := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-2", TransactionHash: transactionHash, PriorCheckId: reincluded.Msg.Check.Id})
	badWallet.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	_, err = client.RecheckBaseUSDCReceipt(ctx, badWallet)
	require.Error(t, err)
	badTransaction := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: "0x" + strings.Repeat("f", 64), PriorCheckId: reincluded.Msg.Check.Id})
	badTransaction.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	_, err = client.RecheckBaseUSDCReceipt(ctx, badTransaction)
	require.Error(t, err)
	otherOperator := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash, PriorCheckId: reincluded.Msg.Check.Id})
	otherOperator.Header().Set(operatorauth.HeaderOperatorToken, "operator-b-token")
	_, err = client.RecheckBaseUSDCReceipt(ctx, otherOperator)
	require.Error(t, err)
	require.Equal(t, 2, verifier.calls, "invalid scopes must be rejected before another provider call")

	noncanonicalRequest := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash, PriorCheckId: reincluded.Msg.Check.Id})
	noncanonicalRequest.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	noncanonical, err := client.RecheckBaseUSDCReceipt(ctx, noncanonicalRequest)
	require.NoError(t, err)
	require.Equal(t, reincluded.Msg.Check.Id, noncanonical.Msg.Check.PriorCheckId)
	require.Equal(t, walletv1.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_NOT_CANONICAL, noncanonical.Msg.Check.Relation)
	require.Equal(t, "provider_reports_not_canonical", noncanonical.Msg.Check.Outcome)
	require.Empty(t, noncanonical.Msg.Check.Transfers)
	require.Equal(t, "not_canonical", noncanonical.Msg.Check.ErrorCode)
	server.Close()
	require.NoError(t, db.Close())

	freshDB, freshServer, freshClient := openOwner(fixedReceiptVerifier{})
	defer func() { freshServer.Close(); _ = freshDB.Close() }()
	list := connect.NewRequest(&authv1.ListBaseUSDCReceiptChecksRequest{WalletId: "wallet-1", TransactionHash: transactionHash})
	list.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	history, err := freshClient.ListBaseUSDCReceiptChecks(ctx, list)
	require.NoError(t, err)
	require.Len(t, history.Msg.Checks, 3)
	require.Equal(t, noncanonical.Msg.Check.Id, history.Msg.Checks[0].Id)
	require.Equal(t, reincluded.Msg.Check.Id, history.Msg.Checks[0].PriorCheckId)
	require.Equal(t, walletv1.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_NOT_CANONICAL, history.Msg.Checks[0].Relation)
	require.Equal(t, "provider_reports_not_canonical", history.Msg.Checks[0].Outcome)
	require.Equal(t, reincluded.Msg.Check.Id, history.Msg.Checks[1].Id)
	require.Equal(t, priorID, history.Msg.Checks[1].PriorCheckId)
	require.Equal(t, walletv1.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION, history.Msg.Checks[1].Relation)
	require.Equal(t, initial.Msg.CheckId, history.Msg.Checks[2].Id)
	require.Empty(t, history.Msg.Checks[2].PriorCheckId)
	require.Equal(t, "verified", history.Msg.Checks[2].Outcome, "historical check remains unchanged")
	require.Len(t, history.Msg.Checks[2].Transfers, 1)
	require.Equal(t, 3, verifier.calls, "reopening and listing must not recheck the provider")
}

func TestAuthenticatedRecheckPersistenceFailureReturnsInternal(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "treasury-recheck-failure.sqlite")+"?_pragma=busy_timeout(1000)")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(baseusdc.Schema)))

	transactionHash := "0x" + strings.Repeat("a", 64)
	checkedAt := time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
	providerID := "fixture-base-rpc"
	transfer := receiptObservation("wallet-1", transactionHash, "0x"+strings.Repeat("b", 64), providerID, checkedAt)
	store := &failSecondReceiptAppendStore{Store: baseusdc.NewSQLiteStore(db)}
	service := baseusdc.NewService(store, fixedReceiptVerifier{transfers: []baseusdc.Observation{transfer}}, providerID, func() time.Time { return checkedAt })
	wallets := &scopedReceiptWallets{owners: map[string]string{"wallet-1": "operator-a"}}
	_, handler := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(subjectAuthorizer{}, wallets, service))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := authconnect.NewTreasuryAdminClient(server.Client(), server.URL)

	initial := connect.NewRequest(&authv1.VerifyBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash})
	initial.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	verified, err := client.VerifyBaseUSDCReceipt(ctx, initial)
	require.NoError(t, err)
	require.NotEmpty(t, verified.Msg.CheckId)

	recheck := connect.NewRequest(&authv1.RecheckBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: transactionHash, PriorCheckId: verified.Msg.CheckId})
	recheck.Header().Set(operatorauth.HeaderOperatorToken, "operator-a-token")
	_, err = client.RecheckBaseUSDCReceipt(ctx, recheck)
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err), "failed durable append is a server failure")
	require.Equal(t, 2, store.appendCalls)

	checks, err := baseusdc.NewSQLiteStore(db).List(ctx, "operator-a", "wallet-1", transactionHash, 10)
	require.NoError(t, err)
	require.Len(t, checks, 1, "failed recheck must not create receipt evidence")
	require.Equal(t, verified.Msg.CheckId, checks[0].ID)
}

func receiptObservation(walletID, txHash, blockHash, provider string, observedAt time.Time) baseusdc.Observation {
	walletAddress := "0x1111111111111111111111111111111111111111"
	return baseusdc.Observation{
		EconomicKey:       baseusdc.Network + "/" + baseusdc.USDCAddress + "/" + txHash + "/log/4/" + walletAddress,
		InclusionRevision: txHash + "/log/4@" + blockHash,
		WalletID:          walletID, Network: baseusdc.Network, AssetID: baseusdc.TransferAsset, Decimals: 6,
		From: "0x2222222222222222222222222222222222222222", To: walletAddress, AtomicAmount: "5901522",
		TransactionHash: txHash, LogIndex: 4, BlockNumber: 100, BlockHash: blockHash,
		BlockTime: observedAt.Add(-time.Minute), Finality: "finalized", FinalizedHeadNumber: 100,
		FinalizedHeadHash: "0x" + strings.Repeat("c", 64), ProviderID: provider, ObservedAt: observedAt,
	}
}
