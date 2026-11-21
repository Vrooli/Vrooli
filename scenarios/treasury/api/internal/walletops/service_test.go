package walletops

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"treasury/internal/authorization"
	"treasury/internal/baseusdc"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/nativeusdc"
	"treasury/internal/receiptfunding"
	"treasury/internal/wallet"
)

type walletopsTestStore map[string]string

type fixedFundingDestination struct{ destination ledger.Destination }

type baseMainnetFundingRPC struct{ head *big.Int }

func (r baseMainnetFundingRPC) VerifyWalletTransaction(context.Context, wallet.Wallet, string) ([]baseusdc.Observation, error) {
	return nil, errors.New("receipt verification is not part of this intent-creation test")
}

func (r baseMainnetFundingRPC) LatestBlockNumber(context.Context) (*big.Int, error) {
	return new(big.Int).Set(r.head), nil
}

func (f fixedFundingDestination) Destination(context.Context) (ledger.Destination, error) {
	return f.destination, nil
}

func (s walletopsTestStore) Put(service, key, value string) error {
	s[service+"/"+key] = value
	return nil
}

func (s walletopsTestStore) Get(service, key string) (string, error) {
	value, ok := s[service+"/"+key]
	if !ok {
		return "", credentialauthority.ErrNotFound
	}
	return value, nil
}

func (s walletopsTestStore) Delete(service, key string) error {
	delete(s, service+"/"+key)
	return nil
}

func setupWalletOpsTest(t *testing.T) (*Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(book.Schema),
		database.SchemaProviderFunc(budget.Schema),
		database.SchemaProviderFunc(mandate.Schema),
		database.SchemaProviderFunc(authorization.Schema),
		database.SchemaProviderFunc(evidence.Schema),
		database.SchemaProviderFunc(wallet.Schema),
		database.SchemaProviderFunc(Schema),
	}
	require.NoError(t, database.EnsureSchemas(ctx, db, providers...))
	clock := schedule.NewFake(time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC))
	books := book.NewService(book.NewSQLiteRepository(db), clock)
	_, err := books.Create(ctx, book.CreateInput{ID: "research", Name: "Local research", BeneficiaryIdentity: "local-operator"})
	require.NoError(t, err)
	budgets := budget.NewService(budget.NewSQLiteRepository(db), clock)
	_, err = budgets.Create(ctx, budget.Budget{
		ID: "research-budget", BookID: "research", Currency: "USD",
		TotalCapMinor: 10_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 1_000,
		Period: time.Hour, AllowedCounterparties: []string{"vendor.example"},
	})
	require.NoError(t, err)
	credentials, err := credentialauthority.NewAuthority(walletopsTestStore{})
	require.NoError(t, err)
	wallets := wallet.NewService(wallet.NewSQLiteRepository(db), books, credentials, clock, nil)
	signer, err := mandate.NewHMACSigner([]byte("walletops-test-signing-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, signer)
	authorizationService := authorization.NewService(
		authorization.NewSQLiteRepository(db), identity.UnavailableVerifier{}, mandates, budgets,
		evidence.NewRecorder(evidence.NewSQLiteRecorder(db)), clock,
	)
	service := New(Config{
		DB: db, Wallets: wallets, Books: books, Budgets: budgets, Clock: clock,
		Mandates: mandates, Authorizations: authorizationService,
	})
	wallets.SetSessionSignAuthorizer(service)
	return service, ctx
}

func TestProvisionBudgetWalletIsStableBoundAndContainsNoCustodyMaterial(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	request := &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research", AgentRunId: "run-42",
	}
	first, err := service.Provision(ctx, "local-operator", request)
	require.NoError(t, err)
	require.Equal(t, request.GetBudgetId(), first.GetBudgetId())
	require.Equal(t, request.GetAgentSubject(), first.GetAgentSubject())
	require.Equal(t, request.GetAgentRunId(), first.GetAgentRunId())
	require.Equal(t, request.GetNetwork(), first.GetNetwork())
	require.NotEmpty(t, first.GetWalletId())
	require.True(t, strings.HasPrefix(first.GetAddress(), "0x"))
	require.Len(t, first.GetAddress(), 42)

	again, err := service.Provision(ctx, "local-operator", request)
	require.NoError(t, err)
	require.Equal(t, first, again, "same budget binding must return the same derived address")

	claims := walletReadClaims(t, service.Clock.Now(), "agent:research", "run-42", "research-budget")
	// The claim target is explicitly budget-scoped; no operator credential or key
	// is needed to read the derived address.
	visible, err := service.GetForAgent(ctx, "research-budget", "", claims)
	require.NoError(t, err)
	require.Equal(t, first, visible)

	encoded, err := protojson.Marshal(first)
	require.NoError(t, err)
	response := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"private_key", "privatekey", "secp256k1", "custody_reference", "recovery_passphrase", "recovery_sha256"} {
		require.NotContains(t, response, forbidden)
	}
}

func TestProvisionBudgetWalletsAreIsolatedAcrossBudgets(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx := setupWalletOpsTest(t)
	_, err := service.Budgets.Create(ctx, budget.Budget{
		ID: "second-research-budget", BookID: "research", Currency: "USD",
		TotalCapMinor: 10_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 1_000,
		Period: time.Hour, AllowedCounterparties: []string{"vendor.example"},
	})
	require.NoError(t, err)

	first, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research-a", AgentRunId: "run-a",
	})
	require.NoError(t, err)
	second, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "second-research-budget", Network: baseusdc.Network, AgentSubject: "agent:research-b", AgentRunId: "run-b",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.GetWalletId(), second.GetWalletId())
	require.NotEqual(t, first.GetAddress(), second.GetAddress(), "different budget identities must derive different addresses")

	firstClaims := walletReadClaims(t, service.Clock.Now(), "agent:research-a", "run-a", "research-budget")
	secondClaims := walletReadClaims(t, service.Clock.Now(), "agent:research-b", "run-b", "second-research-budget")
	firstRead, err := service.GetForAgent(ctx, first.GetBudgetId(), "", firstClaims)
	require.NoError(t, err)
	require.Equal(t, first.GetAddress(), firstRead.GetAddress())
	secondRead, err := service.GetForAgent(ctx, second.GetBudgetId(), "", secondClaims)
	require.NoError(t, err)
	require.Equal(t, second.GetAddress(), secondRead.GetAddress())
	_, err = service.GetForAgent(ctx, first.GetBudgetId(), "", secondClaims)
	require.Error(t, err, "a second budget's run identity must not read the first budget wallet")
	_, err = service.GetForAgent(ctx, second.GetBudgetId(), "", firstClaims)
	require.Error(t, err, "a first budget's run identity must not read the second budget wallet")
}

func TestProvisionBudgetWalletRefusesIdentityReuseAndForeignReads(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	request := &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research", AgentRunId: "run-42",
	}
	first, err := service.Provision(ctx, "local-operator", request)
	require.NoError(t, err)

	changedSubject := *request
	changedSubject.AgentSubject = "agent:other"
	_, err = service.Provision(ctx, "local-operator", &changedSubject)
	require.ErrorIs(t, err, ErrConflict, "one budget binding cannot be reassigned to another agent")
	changedRun := *request
	changedRun.AgentRunId = "run-43"
	_, err = service.Provision(ctx, "local-operator", &changedRun)
	require.ErrorIs(t, err, ErrConflict, "one budget binding cannot be reused by a different run")
	_, err = service.Provision(ctx, "another-operator", request)
	require.Error(t, err, "a non-beneficiary cannot provision the operator-owned budget wallet")

	baseClaims := walletReadClaims(t, service.Clock.Now(), "agent:research", "run-42", "research-budget")
	foreignSubject := baseClaims
	foreignSubject.Subject = "agent:other"
	_, err = service.GetForAgent(ctx, "research-budget", "", foreignSubject)
	require.Error(t, err)
	foreignRun := baseClaims
	foreignRun.RunID = "run-43"
	_, err = service.GetForAgent(ctx, "research-budget", "", foreignRun)
	require.Error(t, err)
	wrongBudgetGrant := baseClaims
	wrongBudgetGrant = walletReadClaims(t, service.Clock.Now(), "agent:research", "run-42", "other-budget")
	_, err = service.GetForAgent(ctx, "research-budget", "", wrongBudgetGrant)
	require.Error(t, err)

	readback, err := service.Get(ctx, "local-operator", "research-budget")
	require.NoError(t, err)
	require.Equal(t, first, readback)
}

func TestFundingWatcherRejectsAmbiguousTransfersAndAdvancesFinalizedCursor(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	budgetWallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research", AgentRunId: "run-42",
	})
	require.NoError(t, err)
	intent := &fundingpb.CreateFundingIntentRequest{
		Id: "watcher-intent", BudgetId: budgetWallet.GetBudgetId(), AtomicAmount: "1000000",
		LedgerBookId: "research", LedgerAccountId: "cash", Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD"},
	}
	// The watcher persists only the intent identity and scan cursor here; no
	// receipt can be credited until it resolves an unambiguous matching log.
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(intent)
	require.NoError(t, err)
	now := service.Clock.Now().UTC().Format(time.RFC3339Nano)
	_, err = service.DB.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,created_at,updated_at) VALUES(?,?,?,?,?,?,'waiting',?,?)`, intent.GetId(), intent.GetBudgetId(), "local-operator", intent.GetAtomicAmount(), encoded, "5", now, now)
	require.NoError(t, err)
	transferTopic := gethcrypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	toTopic := common.BytesToHash(common.HexToAddress(budgetWallet.GetAddress()).Bytes())
	logs := []map[string]any{
		{"address": baseusdc.USDCAddress, "topics": []string{transferTopic.Hex(), common.Hash{}.Hex(), toTopic.Hex()}, "data": fmt.Sprintf("0x%064x", 1_000_000), "transactionHash": "0x" + strings.Repeat("1", 64)},
		{"address": baseusdc.USDCAddress, "topics": []string{transferTopic.Hex(), common.Hash{}.Hex(), toTopic.Hex()}, "data": fmt.Sprintf("0x%064x", 1_000_000), "transactionHash": "0x" + strings.Repeat("2", 64)},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		w.Header().Set("Content-Type", "application/json")
		result := any(map[string]string{"number": "0x10"})
		if request.Method == "eth_getLogs" {
			result = logs
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result}))
	}))
	defer server.Close()
	service.Native, err = nativeusdc.New(service.DB, nil, server.URL, "local-fork")
	require.NoError(t, err)
	require.Error(t, service.WatchFundingIntents(ctx), "two exact transfers matching one intent must not be auto-credited")
	var cursor string
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT next_block FROM funding_intent_watch_cursors WHERE intent_id=?`, intent.GetId()).Scan(&cursor))
	require.Equal(t, "5", cursor, "ambiguous evidence must leave the cursor at the retry point")
	logs = nil
	require.NoError(t, service.WatchFundingIntents(ctx))
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT next_block FROM funding_intent_watch_cursors WHERE intent_id=?`, intent.GetId()).Scan(&cursor))
	require.Equal(t, "17", cursor, "a completed finalized scan advances the durable cursor")
}

func TestExactFundingObservationRequiresOneFinalizedExactBaseUSDCTransfer(t *testing.T) {
	w := wallet.Wallet{ID: "wallet", Address: "0x" + strings.Repeat("a", 40)}
	intent := fundingIntent{id: "intent", operator: "operator", atoms: "1000000", startBlock: "50", request: &fundingpb.CreateFundingIntentRequest{}}
	txHash := "0x" + strings.Repeat("b", 64)
	valid := baseusdc.Observation{WalletID: w.ID, Network: baseusdc.Network, AssetID: baseusdc.TransferAsset, Decimals: baseusdc.USDCDecimals, From: "0x" + strings.Repeat("d", 40), To: w.Address, AtomicAmount: intent.atoms, TransactionHash: txHash, LogIndex: 3, BlockNumber: 51, BlockHash: "0x" + strings.Repeat("c", 64), Finality: "finalized"}
	check := baseusdc.Check{ID: "check", OperatorIdentity: intent.operator, WalletID: w.ID, Network: baseusdc.Network, TransactionHash: txHash, Outcome: "verified", Transfers: []baseusdc.Observation{valid}}

	matched, err := exactFundingObservation(intent, w, txHash, check)
	require.NoError(t, err)
	require.Equal(t, uint64(3), matched.LogIndex)

	preIntent := check
	preIntent.Transfers = []baseusdc.Observation{valid}
	preIntent.Transfers[0].BlockNumber = 49
	_, err = exactFundingObservation(intent, w, txHash, preIntent)
	require.ErrorContains(t, err, "no exact finalized")

	unfinalized := check
	unfinalized.Transfers = []baseusdc.Observation{valid}
	unfinalized.Transfers[0].Finality = "pending"
	_, err = exactFundingObservation(intent, w, txHash, unfinalized)
	require.ErrorContains(t, err, "not finalized")

	duplicate := check
	duplicate.Transfers = []baseusdc.Observation{valid, valid}
	_, err = exactFundingObservation(intent, w, txHash, duplicate)
	require.ErrorContains(t, err, "multiple exact")

	wrongRecipient := check
	wrongRecipient.Transfers = []baseusdc.Observation{valid}
	wrongRecipient.Transfers[0].To = "0x" + strings.Repeat("e", 40)
	_, err = exactFundingObservation(intent, w, txHash, wrongRecipient)
	require.ErrorContains(t, err, "outside the requested")
}

func TestFundingReceiptBindingIsStableAndPreventsOneLogFundingTwoIntents(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	service.Receipts = &baseusdc.Service{}
	service.Funding = &receiptfunding.Service{}
	created, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: "research-budget", Network: baseusdc.Network})
	require.NoError(t, err)
	w, err := service.Wallets.Get(ctx, created.GetWalletId(), baseusdc.Network, "local-operator")
	require.NoError(t, err)
	intentProto := &fundingpb.CreateFundingIntentRequest{Id: "bind-1", BudgetId: "research-budget", AtomicAmount: "1000000", LedgerBookId: "research", LedgerAccountId: "cash", Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD"}}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(intentProto)
	require.NoError(t, err)
	now := service.Clock.Now().UTC().Format(time.RFC3339Nano)
	_, err = service.DB.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,created_at,updated_at) VALUES(?,?,?,?,?,?,'waiting',?,?)`, "bind-1", "research-budget", "local-operator", "1000000", encoded, "50", now, now)
	require.NoError(t, err)
	intent, err := service.loadIntent(ctx, "local-operator", "bind-1")
	require.NoError(t, err)
	txHash := "0x" + strings.Repeat("b", 64)
	obs := baseusdc.Observation{WalletID: w.ID, Network: baseusdc.Network, AssetID: baseusdc.TransferAsset, Decimals: baseusdc.USDCDecimals, From: "0x" + strings.Repeat("d", 40), To: w.Address, AtomicAmount: intent.atoms, TransactionHash: txHash, LogIndex: 4, BlockNumber: 51, BlockHash: "0x" + strings.Repeat("c", 64), Finality: "finalized"}
	check := baseusdc.Check{ID: "retained-check", OperatorIdentity: intent.operator, WalletID: w.ID, Network: baseusdc.Network, TransactionHash: txHash, Outcome: "verified"}
	one, err := service.bindFundingReceipt(ctx, intent, w, txHash, check, obs)
	require.NoError(t, err)
	two, err := service.bindFundingReceipt(ctx, intent, w, txHash, check, obs)
	require.NoError(t, err)
	require.Equal(t, one, two, "retry must retain a deterministic check/log delivery binding")
	_, err = service.bindFundingReceipt(ctx, intent, w, "0x"+strings.Repeat("e", 64), check, obs)
	require.ErrorIs(t, err, ErrInvalid)

	second := *intentProto
	second.Id = "bind-2"
	second.BudgetId = "research-budget-two"
	_, err = budget.NewService(budget.NewSQLiteRepository(service.DB), service.Clock).Create(ctx, budget.Budget{
		ID: second.BudgetId, BookID: "research", Currency: "USD", TotalCapMinor: 1000,
		PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour,
	})
	require.NoError(t, err)
	secondWallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: second.BudgetId, Network: baseusdc.Network})
	require.NoError(t, err)
	require.NotEqual(t, w.ID, secondWallet.GetWalletId())
	encoded, err = proto.MarshalOptions{Deterministic: true}.Marshal(&second)
	require.NoError(t, err)
	_, err = service.DB.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,created_at,updated_at) VALUES(?,?,?,?,?,?,'waiting',?,?)`, "bind-2", second.BudgetId, "local-operator", "1000000", encoded, "50", now, now)
	require.NoError(t, err)
	secondIntent, err := service.loadIntent(ctx, "local-operator", "bind-2")
	require.NoError(t, err)
	secondObservation := obs
	secondObservation.WalletID = secondWallet.GetWalletId()
	secondObservation.To = secondWallet.GetAddress()
	secondCheck := check
	secondCheck.WalletID = secondWallet.GetWalletId()
	_, err = service.bindFundingReceipt(ctx, secondIntent, wallet.Wallet{ID: secondWallet.GetWalletId(), Address: secondWallet.GetAddress()}, txHash, secondCheck, secondObservation)
	require.ErrorIs(t, err, ErrConflict, "one chain log cannot be bound to two funding intents")
}

func TestConcurrentCrossWalletFundingReceiptClaimHasOneWinner(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	// Exercise simultaneous callers on separate SQLite connections. The runtime
	// writer lock may serialize their commit attempts; the invariant is that no
	// scheduling outcome can create two claims or two bindings for one log.
	pool, ok := service.DB.(interface{ SetMaxOpenConns(int) })
	require.True(t, ok, "test database should expose its SQLite connection pool")
	pool.SetMaxOpenConns(2)
	service.Receipts = &baseusdc.Service{}
	service.Funding = &receiptfunding.Service{}

	firstWallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: "research-budget", Network: baseusdc.Network})
	require.NoError(t, err)
	secondBudgetID := "research-budget-two"
	_, err = budget.NewService(budget.NewSQLiteRepository(service.DB), service.Clock).Create(ctx, budget.Budget{
		ID: secondBudgetID, BookID: "research", Currency: "USD", TotalCapMinor: 1000,
		PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour,
	})
	require.NoError(t, err)
	secondWallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: secondBudgetID, Network: baseusdc.Network})
	require.NoError(t, err)

	now := service.Clock.Now().UTC().Format(time.RFC3339Nano)
	newIntent := func(id, budgetID string) fundingIntent {
		t.Helper()
		request := &fundingpb.CreateFundingIntentRequest{Id: id, BudgetId: budgetID, AtomicAmount: "1000000", LedgerBookId: "research", LedgerAccountId: "cash", Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD"}}
		encoded, marshalErr := proto.MarshalOptions{Deterministic: true}.Marshal(request)
		require.NoError(t, marshalErr)
		_, insertErr := service.DB.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,created_at,updated_at) VALUES(?,?,?,?,?,?,'waiting',?,?)`, id, budgetID, "local-operator", "1000000", encoded, "50", now, now)
		require.NoError(t, insertErr)
		intent, loadErr := service.loadIntent(ctx, "local-operator", id)
		require.NoError(t, loadErr)
		return intent
	}
	firstIntent := newIntent("concurrent-intent-one", "research-budget")
	secondIntent := newIntent("concurrent-intent-two", secondBudgetID)
	txHash := "0x" + strings.Repeat("f", 64)
	blockHash := "0x" + strings.Repeat("c", 64)
	observation := func(w *fundingpb.BudgetWallet) baseusdc.Observation {
		return baseusdc.Observation{
			WalletID: w.GetWalletId(), Network: baseusdc.Network, AssetID: baseusdc.TransferAsset,
			Decimals: baseusdc.USDCDecimals, From: "0x" + strings.Repeat("d", 40), To: w.GetAddress(),
			AtomicAmount: "1000000", TransactionHash: txHash, LogIndex: 11,
			BlockNumber: 51, BlockHash: blockHash, Finality: "finalized",
		}
	}
	type result struct{ err error }
	type contender struct {
		intent fundingIntent
		wallet wallet.Wallet
		check  baseusdc.Check
		obs    baseusdc.Observation
	}
	contenders := []contender{
		{intent: firstIntent, wallet: wallet.Wallet{ID: firstWallet.GetWalletId(), Address: firstWallet.GetAddress()}, check: baseusdc.Check{ID: "concurrent-check-one", OperatorIdentity: "local-operator", WalletID: firstWallet.GetWalletId(), Network: baseusdc.Network, TransactionHash: txHash, Outcome: "verified"}, obs: observation(firstWallet)},
		{intent: secondIntent, wallet: wallet.Wallet{ID: secondWallet.GetWalletId(), Address: secondWallet.GetAddress()}, check: baseusdc.Check{ID: "concurrent-check-two", OperatorIdentity: "local-operator", WalletID: secondWallet.GetWalletId(), Network: baseusdc.Network, TransactionHash: txHash, Outcome: "verified"}, obs: observation(secondWallet)},
	}
	start := make(chan struct{})
	ready := make(chan struct{}, len(contenders))
	results := make(chan result, len(contenders))
	for _, current := range contenders {
		current := current
		go func() {
			ready <- struct{}{}
			<-start
			_, bindErr := service.bindFundingReceipt(ctx, current.intent, current.wallet, txHash, current.check, current.obs)
			results <- result{err: bindErr}
		}()
	}
	for range contenders {
		<-ready
	}
	close(start)

	successes, conflicts := 0, 0
	for range contenders {
		got := <-results
		if got.err == nil {
			successes++
			continue
		}
		require.ErrorIs(t, got.err, ErrConflict, "the losing caller should observe the existing global claim")
		conflicts++
	}
	require.Equal(t, 1, successes, "one intent must retain the globally unique chain-log claim")
	require.Equal(t, 1, conflicts, "the competing budget wallet must fail with a conflict")
	var claims, bindings int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_intent_receipt_log_claims WHERE network=? AND asset_id=? AND transaction_hash=? AND log_index=?`, baseusdc.Network, baseusdc.TransferAsset, txHash, 11).Scan(&claims))
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_intent_receipt_bindings WHERE network=? AND asset_id=? AND LOWER(transaction_hash)=LOWER(?) AND log_index=?`, baseusdc.Network, baseusdc.TransferAsset, txHash, 11).Scan(&bindings))
	require.Equal(t, 1, claims)
	require.Equal(t, 1, bindings)
}

func TestReceiptClaimSchemaFailsClosedAndPreservesConflictingLegacyBindings(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE budget_wallets (budget_id TEXT PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO budget_wallets(budget_id) VALUES('budget-one'),('budget-two')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE funding_intents (id TEXT PRIMARY KEY, budget_id TEXT NOT NULL, state TEXT NOT NULL, FOREIGN KEY(budget_id) REFERENCES budget_wallets(budget_id))`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,state) VALUES('intent-one','budget-one','credited'),('intent-two','budget-two','credited')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE funding_intent_receipt_bindings (
		intent_id TEXT PRIMARY KEY REFERENCES funding_intents(id), operator_identity TEXT NOT NULL,
		wallet_id TEXT NOT NULL, network TEXT NOT NULL, asset_id TEXT NOT NULL, transaction_hash TEXT NOT NULL,
		log_index INTEGER NOT NULL, atomic_amount TEXT NOT NULL, recipient TEXT NOT NULL,
		block_hash TEXT NOT NULL, check_id TEXT NOT NULL, delivery_id TEXT NOT NULL UNIQUE,
		created_at TEXT NOT NULL, UNIQUE(wallet_id,network,asset_id,transaction_hash,log_index)
	)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO funding_intent_receipt_bindings(
		intent_id,operator_identity,wallet_id,network,asset_id,transaction_hash,log_index,atomic_amount,recipient,block_hash,check_id,delivery_id,created_at
	) VALUES
		('intent-one','operator','wallet-one','base','native-usdc','0xabc',9,'100','0x111','0xaaa','check-one','delivery-one','2026-11-01T00:00:00Z'),
		('intent-two','operator','wallet-two','base','native-usdc','0xABC',9,'100','0x222','0xbbb','check-two','delivery-two','2026-11-02T00:00:00Z')`)
	require.NoError(t, err)

	err = database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema))
	require.ErrorContains(t, err, "conflicting historical funding receipt bindings require operator reconciliation")
	var conflictCount, claimCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_intent_receipt_migration_conflicts`).Scan(&conflictCount))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_intent_receipt_log_claims`).Scan(&claimCount))
	require.Equal(t, 2, conflictCount, "both original conflicting bindings must remain in the diagnostic ledger")
	require.Zero(t, claimCount, "a conflicting chain log must not receive an arbitrary canonical claim")
}

func TestCreateFundingIntentRecordsOperatorValuationEvidence(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	service.Receipts = &baseusdc.Service{}
	service.Funding = &receiptfunding.Service{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.ID) == 0 {
			http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x64"}`, request.ID)
	}))
	defer server.Close()
	var err error
	service.Native, err = nativeusdc.New(service.DB, nil, server.URL, "local-fork")
	require.NoError(t, err)
	wallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: "research-budget", Network: baseusdc.Network})
	require.NoError(t, err)

	intent, err := service.CreateFundingIntent(ctx, "local-operator", &fundingpb.CreateFundingIntentRequest{
		Id: "operator-valued-intent", BudgetId: wallet.GetBudgetId(), AtomicAmount: "1000000",
		LedgerBookId: "research", LedgerAccountId: "cash",
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", BasisReference: "operator declared local-fork value"},
	})
	require.NoError(t, err)
	require.Equal(t, "operator-valued-intent", intent.GetId())

	var encoded []byte
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT request_proto FROM funding_intents WHERE id=?`, intent.GetId()).Scan(&encoded))
	var stored fundingpb.CreateFundingIntentRequest
	require.NoError(t, proto.Unmarshal(encoded, &stored))
	require.Equal(t, sharedpb.Basis_BASIS_OPERATOR_ASSERTED, stored.GetValuation().GetBasis())
	require.True(t, stored.GetValuation().GetValuedAt().AsTime().Equal(service.Clock.Now()), "Treasury should timestamp the operator valuation at intent creation")
	require.Equal(t, "research", stored.GetLedgerBookId())
	require.Equal(t, "cash", stored.GetLedgerAccountId())
}

func TestMainnetFundingIntentRequiresExplicitGateAndUsesReadOnlyReceiptHead(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	service.Funding = &receiptfunding.Service{}
	service.Receipts = baseusdc.NewService(baseusdc.NewSQLiteStore(service.DB), baseMainnetFundingRPC{head: big.NewInt(8450000)}, "base-mainnet-fixture", service.Clock.Now)
	wallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: "research-budget", Network: baseusdc.Network})
	require.NoError(t, err)
	request := &fundingpb.CreateFundingIntentRequest{
		Id: "explicit-mainnet-intent", BudgetId: wallet.GetBudgetId(), AtomicAmount: "2600000",
		Valuation:    &ingestpb.CryptoValuation{AmountMinor: 260, Currency: "USD", BasisReference: "operator funding preview"},
		LedgerBookId: "research", LedgerAccountId: "cash",
	}

	_, err = service.CreateFundingIntent(ctx, "local-operator", request)
	require.ErrorIs(t, err, ErrInvalid, "empty funding mode must stay disabled")
	service.FundingMode = "base-mainnet"
	intent, err := service.CreateFundingIntent(ctx, "local-operator", request)
	require.NoError(t, err)
	require.Equal(t, "2600000", intent.GetAtomicAmount())
	var startBlock string
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT start_block FROM funding_intents WHERE id=?`, intent.GetId()).Scan(&startBlock))
	require.Equal(t, "8450000", startBlock)
	require.Nil(t, service.Native, "mainnet inbound mode must not need or configure a signing adapter")
}

func TestCreateFundingIntentUsesOperatorSelectedLedgerDestination(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	service.Receipts = &baseusdc.Service{}
	service.Funding = &receiptfunding.Service{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.ID) == 0 {
			http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x64"}`, request.ID)
	}))
	defer server.Close()
	var err error
	service.Native, err = nativeusdc.New(service.DB, nil, server.URL, "local-fork")
	require.NoError(t, err)
	wallet, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{BudgetId: "research-budget", Network: baseusdc.Network})
	require.NoError(t, err)
	service.LedgerDestination = fixedFundingDestination{destination: ledger.Destination{
		BookID: "operator-book", AccountID: "operator-deposit-account",
	}}

	intent, err := service.CreateFundingIntent(ctx, "local-operator", &fundingpb.CreateFundingIntentRequest{
		Id: "operator-destination-intent", BudgetId: wallet.GetBudgetId(), AtomicAmount: "1000000",
		LedgerBookId: "client-injected-book", LedgerAccountId: "client-injected-account",
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", BasisReference: "operator declared local-fork value"},
	})
	require.NoError(t, err)

	var encoded []byte
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT request_proto FROM funding_intents WHERE id=?`, intent.GetId()).Scan(&encoded))
	var stored fundingpb.CreateFundingIntentRequest
	require.NoError(t, proto.Unmarshal(encoded, &stored))
	require.Equal(t, "operator-book", stored.GetLedgerBookId())
	require.Equal(t, "operator-deposit-account", stored.GetLedgerAccountId())
}

func TestProvisionBudgetWalletRejectsPartialRunBinding(t *testing.T) {
	service, ctx := setupWalletOpsTest(t)
	_, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research",
	})
	require.ErrorIs(t, err, ErrInvalid)
}

func TestMandateSessionWalletExpiresRotatesAndRevokesByGeneration(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx, now, session, _ := provisionTestMandateSession(t)
	require.NotEmpty(t, session.Address)
	require.Equal(t, int64(1), session.Generation)
	require.True(t, session.ExpiresAt.After(now))
	require.LessOrEqual(t, session.ExpiresAt.Sub(now), mandateSessionTTL)
	require.NotEqual(t, "", session.WalletID)
	budget, err := service.load(ctx, session.BudgetID)
	require.NoError(t, err)
	rootWallet, err := service.Wallets.Get(ctx, budget.parentID, session.Network, session.OperatorIdentity)
	require.NoError(t, err)
	sessionWallet, err := service.Wallets.Get(ctx, session.WalletID, session.Network, session.OperatorIdentity)
	require.NoError(t, err)
	require.NotEqual(t, rootWallet.Address, sessionWallet.Address, "the mandate session signs for its derived budget wallet")
	require.Equal(t, session.Address, sessionWallet.Address, "the agent session is bound to the derived budget wallet")
	require.NotEqual(t, rootWallet.PublicKey, sessionWallet.PublicKey, "the budget wallet remains a distinct child of the operator root")

	claims := walletReadClaims(t, now, "agent:research", "run-42", "research-budget")
	visible, err := service.GetMandateSessionForAgent(ctx, "research-budget", "mandate-session-test", claims)
	require.NoError(t, err)
	require.Equal(t, session, visible)
	wrongRun := claims
	wrongRun.RunID = "run-other"
	_, err = service.GetMandateSessionForAgent(ctx, "research-budget", "mandate-session-test", wrongRun)
	require.Error(t, err)

	_, err = service.DB.ExecContext(ctx, `UPDATE mandate_session_keys SET expires_at=? WHERE mandate_id=?`, now.Add(-time.Second).Format(time.RFC3339Nano), session.MandateID)
	require.NoError(t, err)
	_, err = service.GetMandateSessionForAgent(ctx, "research-budget", session.MandateID, claims)
	require.ErrorIs(t, err, ErrNotFound)

	rotated, err := service.RotateMandateSession(ctx, "local-operator", session.MandateID, "rotate-session-test-key")
	require.NoError(t, err)
	require.Equal(t, int64(2), rotated.Generation)
	require.Equal(t, session.Address, rotated.Address, "rotation preserves the derived budget address")
	var rotatedKeyCount int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_agent_session_keys WHERE mandate_id=? AND generation=?`, session.MandateID, rotated.Generation).Scan(&rotatedKeyCount))
	require.Zero(t, rotatedKeyCount, "rotation revokes the old agent-held key and awaits a fresh public-key binding")
	require.NoError(t, service.RevokeMandateSession(ctx, "local-operator", session.MandateID))
	_, err = service.GetMandateSessionForAgent(ctx, "research-budget", session.MandateID, claims)
	require.Error(t, err)
}

func TestMandateSessionRotationReplayReturnsSameGenerationAcrossServiceRestart(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx, _, session, _ := provisionTestMandateSession(t)
	request := &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: session.BudgetID, Network: session.Network, AgentSubject: session.AgentSubject, AgentRunId: session.AgentRunID,
		MandateId: session.MandateID, RotateSession: true, RotationIdempotencyKey: "rotate-generation-2",
	}
	first, err := service.Provision(ctx, "local-operator", request)
	require.NoError(t, err)
	require.Equal(t, session.Generation+1, first.GetSessionGeneration())
	changedPayload := proto.Clone(request).(*fundingpb.ProvisionBudgetWalletRequest)
	changedPayload.AgentRunId = "different-run"
	_, err = service.Provision(ctx, "local-operator", changedPayload)
	require.ErrorIs(t, err, ErrConflict, "the same rotation request identity cannot be rebound to a different run")
	var unchangedGeneration int64
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT generation FROM mandate_session_keys WHERE mandate_id=? AND state='active'`, session.MandateID).Scan(&unchangedGeneration))
	require.Equal(t, first.GetSessionGeneration(), unchangedGeneration, "changed-input rejection must preserve the active generation")

	// Simulate process death after the new generation is active but before the
	// caller's result snapshot is persisted. A different request must fail
	// closed; the same request must resume that generation rather than retire it.
	_, err = service.DB.ExecContext(ctx, `UPDATE mandate_session_rotations SET result_json=X'' WHERE mandate_id=? AND operator_identity=? AND idempotency_key=?`, session.MandateID, "local-operator", request.GetRotationIdempotencyKey())
	require.NoError(t, err)
	changed := proto.Clone(request).(*fundingpb.ProvisionBudgetWalletRequest)
	changed.RotationIdempotencyKey = "different-rotation"
	_, err = service.Provision(ctx, "local-operator", changed)
	require.ErrorIs(t, err, ErrConflict)

	restarted := New(Config{
		DB: service.DB, Wallets: service.Wallets, Books: service.Books, Budgets: service.Budgets, Clock: service.Clock,
		Mandates: service.Mandates, Authorizations: service.Authorizations,
	})
	replay, err := restarted.Provision(ctx, "local-operator", proto.Clone(request).(*fundingpb.ProvisionBudgetWalletRequest))
	require.NoError(t, err)
	require.True(t, proto.Equal(first, replay))
	var activeGeneration int64
	require.NoError(t, restarted.DB.QueryRowContext(ctx, `SELECT generation FROM mandate_session_keys WHERE mandate_id=? AND state='active'`, session.MandateID).Scan(&activeGeneration))
	require.Equal(t, first.GetSessionGeneration(), activeGeneration, "replay after service reconstruction must leave exactly the original rotated generation active")

	secondRequest := proto.Clone(request).(*fundingpb.ProvisionBudgetWalletRequest)
	secondRequest.RotationIdempotencyKey = "rotate-generation-3"
	second, err := restarted.Provision(ctx, "local-operator", secondRequest)
	require.NoError(t, err)
	require.Equal(t, first.GetSessionGeneration()+1, second.GetSessionGeneration(), "a distinct key expresses a distinct rotation")
	conflict, err := restarted.Provision(ctx, "local-operator", proto.Clone(request).(*fundingpb.ProvisionBudgetWalletRequest))
	require.NoError(t, err)
	require.True(t, proto.Equal(first, conflict), "replaying a completed old key returns its retained result without rolling back or rotating again")
}

func TestProvisionBudgetWalletRotationRequiresIdempotencyKeyBeforeCreatingWallet(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx := setupWalletOpsTest(t)
	_, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research", AgentRunId: "run-42",
		MandateId: "missing-mandate", RotateSession: true,
	})
	require.ErrorIs(t, err, ErrInvalid)
	var roots, children int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM research_wallets`).Scan(&roots))
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM budget_wallets`).Scan(&children))
	require.Zero(t, roots, "invalid rotate request must fail before creating the budget root wallet")
	require.Zero(t, children, "invalid rotate request must fail before persisting the budget wallet binding")
}

func TestMandateSessionRotationAndRevocationStayWithinBudgetRun(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx, now, first, firstPrivateKey := provisionTestMandateSession(t)
	_, err := service.Budgets.Create(ctx, budget.Budget{
		ID: "second-research-budget", BookID: "research", Currency: "USD",
		TotalCapMinor: 10_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 1_000,
		Period: time.Hour, AllowedCounterparties: []string{"vendor.example"},
	})
	require.NoError(t, err)
	_, err = service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "second-research-budget", Network: baseusdc.Network, AgentSubject: "agent:research-b", AgentRunId: "run-b",
	})
	require.NoError(t, err)
	_, err = service.Mandates.Issue(ctx, mandate.IssueInput{
		ID: "second-mandate-session-test", IdempotencyKey: "second-mandate-session-test-key", BookID: "research", BudgetID: "second-research-budget",
		AgentSubject: "agent:research-b", AgentRunID: "run-b", Authorizer: "local-operator", CapMinor: 1_000,
		Currency: "USD", AllowedCounterparties: []string{"vendor.example"}, ExpiresAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	second, err := service.ProvisionMandateSession(ctx, "local-operator", "second-mandate-session-test")
	require.NoError(t, err)
	require.NotEqual(t, first.Address, second.Address)
	var phantomSessionWallets int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM research_wallets WHERE id LIKE 'mandate-session-%'`).Scan(&phantomSessionWallets))
	require.Zero(t, phantomSessionWallets, "agent session generations must not mint unused Treasury wallet keys")
	secondKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	secondClaims := walletWriteClaims(t, now, second.AgentSubject, second.AgentRunID, second.BudgetID)
	require.NoError(t, service.BindAgentSessionPublicKey(ctx, second.BudgetID, second.MandateID, secondClaims, "0x"+hex.EncodeToString(gethcrypto.FromECDSAPub(&secondKey.PublicKey))))
	secondAuthorization := authorization.Record{
		ID: "second-budget-rotation-auth", IdempotencyKey: "second-budget-rotation-auth", BookID: "research", MandateID: second.MandateID,
		BudgetID: second.BudgetID, RequestingAgent: second.AgentSubject, AmountMinor: 100, Currency: "USD",
		Counterparty: "vendor.example", Verdict: authorization.VerdictApproved, HoldMinor: 100,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	require.NoError(t, insertApprovedAuthorization(t, ctx, service, secondAuthorization))
	secondIntent := intentForSession(second, secondAuthorization.ID, "second-budget-rotation-settlement", "instrument-test", secondAuthorization.ID, secondAuthorization.AmountMinor, secondAuthorization.Currency, secondAuthorization.Counterparty)
	secondSignature, err := signAgentIntent("0x"+hex.EncodeToString(gethcrypto.FromECDSA(secondKey)), secondIntent)
	require.NoError(t, err)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, secondClaims, secondIntent, secondSignature), "second run's key must authorize only its own approved spend")

	firstAuthorization := authorization.Record{
		ID: "first-budget-rotation-auth", IdempotencyKey: "first-budget-rotation-auth", BookID: "research", MandateID: first.MandateID,
		BudgetID: first.BudgetID, RequestingAgent: first.AgentSubject, AmountMinor: 100, Currency: "USD",
		Counterparty: "0x0000000000000000000000000000000000000001", Verdict: authorization.VerdictApproved, HoldMinor: 100,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	require.NoError(t, insertApprovedAuthorization(t, ctx, service, firstAuthorization))
	firstIntent := intentForSession(first, firstAuthorization.ID, "first-budget-rotation-settlement", "instrument-test", firstAuthorization.ID, firstAuthorization.AmountMinor, firstAuthorization.Currency, firstAuthorization.Counterparty)
	firstSignature, err := signAgentIntent(firstPrivateKey, firstIntent)
	require.NoError(t, err)
	firstClaims := walletWriteClaims(t, now, first.AgentSubject, first.AgentRunID, first.BudgetID)

	firstRotated, err := service.RotateMandateSession(ctx, "local-operator", first.MandateID, "first-budget-rotation-key")
	require.NoError(t, err)
	require.Equal(t, first.Generation+1, firstRotated.Generation)
	require.Equal(t, first.Address, firstRotated.Address)
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM research_wallets WHERE id LIKE 'mandate-session-%'`).Scan(&phantomSessionWallets))
	require.Zero(t, phantomSessionWallets, "rotation must create a new run-held public-key generation without minting a Treasury key")
	require.ErrorIs(t, service.AuthorizeAgentSettlement(ctx, firstClaims, firstIntent, firstSignature), ErrSessionInactive,
		"the retired generation cannot authorize a settlement after rotation")
	secondVisible, err := service.GetMandateSessionForAgent(ctx, second.BudgetID, second.MandateID, walletReadClaims(t, now, second.AgentSubject, second.AgentRunID, second.BudgetID))
	require.NoError(t, err, "rotating one budget must not revoke another budget's run session")
	require.Equal(t, second, secondVisible)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, secondClaims, secondIntent, secondSignature),
		"rotating a different budget must not invalidate its run-owned settlement authority")

	rotatedFirstKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	require.NoError(t, service.BindAgentSessionPublicKey(ctx, first.BudgetID, first.MandateID, firstClaims,
		"0x"+hex.EncodeToString(gethcrypto.FromECDSAPub(&rotatedFirstKey.PublicKey))))
	firstIntent.Generation = firstRotated.Generation
	firstIntent.ExpiresAt = firstRotated.ExpiresAt.UTC().Format(time.RFC3339Nano)
	firstIntent.SettlementID = "first-budget-rotated-settlement"
	rotatedFirstSignature, err := signAgentIntent("0x"+hex.EncodeToString(gethcrypto.FromECDSA(rotatedFirstKey)), firstIntent)
	require.NoError(t, err)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, firstClaims, firstIntent, rotatedFirstSignature),
		"an operator-rotated generation accepts a newly bound key for that same run")
	_, err = service.GetMandateSessionForAgent(ctx, second.BudgetID, second.MandateID, walletReadClaims(t, now, first.AgentSubject, first.AgentRunID, first.BudgetID))
	require.Error(t, err, "a different budget/run cannot read the second budget's session")

	require.NoError(t, service.RevokeMandateSession(ctx, "local-operator", first.MandateID))
	_, err = service.GetMandateSessionForAgent(ctx, first.BudgetID, first.MandateID, walletReadClaims(t, now, first.AgentSubject, first.AgentRunID, first.BudgetID))
	require.Error(t, err, "revocation must disable only the bound budget/run session")
	require.ErrorIs(t, service.AuthorizeAgentSettlement(ctx, firstClaims, firstIntent, rotatedFirstSignature), ErrSessionInactive,
		"revocation prevents the previously rebound session from authorizing future work")
	secondVisible, err = service.GetMandateSessionForAgent(ctx, second.BudgetID, second.MandateID, secondClaims)
	require.NoError(t, err, "revoking one budget must leave another independently bound session active")
	require.Equal(t, second, secondVisible)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, secondClaims, secondIntent, secondSignature),
		"first-budget revocation does not disturb idempotent recovery for the second budget's already consumed exact proof")
}

func TestMandateRotationRejectsProofFromPreviousGeneration(t *testing.T) { // [REQ:TRS-P0-001]
	service, ctx, now, session, privateKeyHex := provisionTestMandateSession(t)
	request := sessionSignRequest(session, "auth-session-old-generation", "sha256:"+strings.Repeat("c", 64))
	require.NoError(t, insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: request.AuthorizationID, IdempotencyKey: request.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: request.ValueMinor, Currency: request.Currency,
		Counterparty: request.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: request.ValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}))
	oldIntent := intentForSession(session, request.AuthorizationID, "settlement-old-generation", "instrument-test", request.AuthorizationID, request.ValueMinor, request.Currency, request.Recipient)
	oldSignature, err := signAgentIntent(privateKeyHex, oldIntent)
	require.NoError(t, err)
	claims := walletWriteClaims(t, now, session.AgentSubject, session.AgentRunID, session.BudgetID)

	rotated, err := service.RotateMandateSession(ctx, "local-operator", session.MandateID, "rotate-session-test-key")
	require.NoError(t, err)
	require.Equal(t, session.Generation+1, rotated.Generation)
	require.Equal(t, session.Address, rotated.Address)
	err = service.AuthorizeAgentSettlement(ctx, claims, oldIntent, oldSignature)
	require.ErrorIs(t, err, ErrSessionInactive, "rotation retires the old generation before any new key binding")

	publicKey, err := gethcrypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	require.NoError(t, err)
	claimsRead := walletReadClaims(t, now, session.AgentSubject, session.AgentRunID, session.BudgetID)
	oldPublicKey := "0x" + hex.EncodeToString(gethcrypto.FromECDSAPub(&publicKey.PublicKey))
	require.ErrorIs(t, service.BindAgentSessionPublicKey(ctx, session.BudgetID, session.MandateID, claimsRead, oldPublicKey), ErrAgentSessionProof,
		"retired public keys cannot be rebound to a later generation")
	rotatedKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	rotatedPublicKey := "0x" + hex.EncodeToString(gethcrypto.FromECDSAPub(&rotatedKey.PublicKey))
	require.NoError(t, service.BindAgentSessionPublicKey(ctx, session.BudgetID, session.MandateID, claimsRead, rotatedPublicKey))
	newIntent := oldIntent
	newIntent.SettlementID = "settlement-new-generation"
	newIntent.Generation = rotated.Generation
	newIntent.ExpiresAt = rotated.ExpiresAt.UTC().Format(time.RFC3339Nano)
	newSignature, err := signAgentIntent("0x"+hex.EncodeToString(gethcrypto.FromECDSA(rotatedKey)), newIntent)
	require.NoError(t, err)
	require.ErrorIs(t, service.AuthorizeAgentSettlement(ctx, claims, oldIntent, oldSignature), ErrAgentSessionProof, "a rebound key cannot replay the prior generation's signed intent")
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, claims, newIntent, newSignature), "a newly signed intent can use the new generation binding")
}

func TestMandateSessionSigningConsumesApprovedAuthOnlyAfterFreezeAndScopeChecks(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-002] [REQ:TRS-P1-006]
	service, ctx, now, session, sessionPrivateKey := provisionTestMandateSession(t)
	amountMismatch := sessionSignRequest(session, "auth-session-amount-mismatch", "sha256:"+strings.Repeat("d", 64))
	amountMismatch.AtomicAmount = "50000000"
	unsignedMismatch := unsignedTestUSDCTransfer(t, amountMismatch)
	_, err := service.Wallets.SignUSDCTransfer(ctx, amountMismatch, unsignedMismatch)
	require.Error(t, err, "a $1 approval must not sign $50 of USDC")
	var count int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_session_signatures WHERE authorization_id=?`, amountMismatch.AuthorizationID).Scan(&count))
	require.Zero(t, count)

	good := sessionSignRequest(session, "auth-session-good", "sha256:"+strings.Repeat("a", 64))
	err = insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: good.AuthorizationID, IdempotencyKey: good.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: good.ValueMinor, Currency: good.Currency,
		Counterparty: good.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: good.ValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	goodIntent := intentForSession(session, good.AuthorizationID, "settlement-good", "instrument-test", good.AuthorizationID, good.ValueMinor, good.Currency, good.Recipient)
	claims := walletWriteClaims(t, now, session.AgentSubject, session.AgentRunID, session.BudgetID)
	wrongSessionKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	wrongSessionSignature, err := gethcrypto.Sign(mustAgentIntentDigest(t, goodIntent), wrongSessionKey)
	require.NoError(t, err)
	require.ErrorIs(t, service.AuthorizeAgentSettlement(ctx, claims, goodIntent, "0x"+hex.EncodeToString(wrongSessionSignature)), ErrAgentSessionProof)
	var proofCount, signatureCount int
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_agent_session_proofs WHERE authorization_id=?`, good.AuthorizationID).Scan(&proofCount))
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_session_signatures WHERE authorization_id=?`, good.AuthorizationID).Scan(&signatureCount))
	require.Zero(t, proofCount, "a wrong agent key cannot create an admitted intent")
	require.Zero(t, signatureCount, "a wrong agent key cannot cause custody to sign")
	_, err = service.Wallets.SignUSDCTransfer(ctx, good, unsignedTestUSDCTransfer(t, good))
	require.ErrorIs(t, err, ErrSessionInactive, "custody refuses dispatch when no agent proof is durably admitted")
	goodSignature, err := signAgentIntent(sessionPrivateKey, goodIntent)
	require.NoError(t, err)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, claims, goodIntent, goodSignature))
	require.NoError(t, service.AuthorizeAndConsumeSessionSign(ctx, good))
	require.NoError(t, service.AuthorizeAndConsumeSessionSign(ctx, good), "identical durable signing intents can recover safely")
	changedGood := good
	changedGood.TransactionDigest = "sha256:" + strings.Repeat("e", 64)
	require.ErrorIs(t, service.AuthorizeAndConsumeSessionSign(ctx, changedGood), ErrSessionReplay)

	frozen := sessionSignRequest(session, "auth-session-frozen", "sha256:"+strings.Repeat("b", 64))
	err = insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: frozen.AuthorizationID, IdempotencyKey: frozen.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: frozen.ValueMinor, Currency: frozen.Currency,
		Counterparty: frozen.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: frozen.ValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	_, err = service.Budgets.SetFrozen(ctx, session.BudgetID, true)
	require.NoError(t, err)
	frozenIntent := intentForSession(session, frozen.AuthorizationID, "settlement-frozen", "instrument-test", frozen.AuthorizationID, frozen.ValueMinor, frozen.Currency, frozen.Recipient)
	frozenSignature, err := signAgentIntent(sessionPrivateKey, frozenIntent)
	require.NoError(t, err)
	require.Error(t, service.AuthorizeAgentSettlement(ctx, claims, frozenIntent, frozenSignature), "a frozen budget cannot admit a new agent session proof")
	unsigned := unsignedTestUSDCTransfer(t, frozen)
	_, err = service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.Error(t, err, "freeze must block before custody creates a signature")
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_session_signatures WHERE authorization_id=?`, frozen.AuthorizationID).Scan(&count))
	require.Zero(t, count, "a frozen request must not create a durable signing claim")

	_, err = service.Budgets.SetFrozen(ctx, session.BudgetID, false)
	require.NoError(t, err)
	require.NoError(t, service.AuthorizeAgentSettlement(ctx, claims, frozenIntent, frozenSignature))
	signed, err := service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.NoError(t, err)
	r, s, v := signed.RawSignatureValues()
	require.NotNil(t, r)
	require.NotNil(t, s)
	require.NotNil(t, v)
	retriedSignature, err := service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.NoError(t, err, "crash recovery may reconstruct the identical deterministic signature")
	require.Equal(t, signed.Hash(), retriedSignature.Hash(), "an identical retry must reproduce the same transaction")
	_, err = service.Budgets.SetFrozen(ctx, session.BudgetID, true)
	require.NoError(t, err)
	_, err = service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.Error(t, err, "an already-consumed intent cannot reconstruct a new signature after the kill switch")
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_session_signatures WHERE authorization_id=?`, frozen.AuthorizationID).Scan(&count))
	require.Equal(t, 1, count, "freeze must not create another signing claim or signature dispatch opportunity")
	_, err = service.Budgets.SetFrozen(ctx, session.BudgetID, false)
	require.NoError(t, err)
	postFreezeRetry, err := service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.NoError(t, err, "the identical recovery retry remains available after policy is unfrozen")
	require.Equal(t, signed.Hash(), postFreezeRetry.Hash())
	changedPayload := unsignedTestUSDCTransfer(t, frozen)
	asset := common.HexToAddress(nativeusdc.Asset)
	changedPayload = types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(8453), Nonce: 2, To: &asset, Gas: changedPayload.Gas(), GasFeeCap: changedPayload.GasFeeCap(), GasTipCap: changedPayload.GasTipCap(), Data: changedPayload.Data()})
	_, err = service.Wallets.SignUSDCTransfer(ctx, frozen, changedPayload)
	require.ErrorIs(t, err, ErrSessionReplay, "the same authorization cannot sign a changed transaction")
	require.NoError(t, service.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_session_signatures WHERE authorization_id=?`, frozen.AuthorizationID).Scan(&count))
	require.Equal(t, 1, count)
	wrongRecipient := sessionSignRequest(session, "auth-session-wrong-recipient", "sha256:"+strings.Repeat("c", 64))
	wrongRecipient.Recipient = "0x0000000000000000000000000000000000000002"
	err = insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: wrongRecipient.AuthorizationID, IdempotencyKey: wrongRecipient.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: wrongRecipient.ValueMinor, Currency: wrongRecipient.Currency,
		Counterparty: wrongRecipient.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: wrongRecipient.ValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Error(t, service.AuthorizeAndConsumeSessionSign(ctx, wrongRecipient), "destination outside the mandate allowlist must fail before signature creation")
	require.NoError(t, service.RevokeMandateSession(ctx, "local-operator", session.MandateID))
	_, err = service.Wallets.SignUSDCTransfer(ctx, frozen, unsigned)
	require.Error(t, err, "revoking the session must disable replay of a previously consumed authorization")
}

func mustAgentIntentDigest(t *testing.T, intent AgentSettlementIntent) []byte {
	t.Helper()
	digest, err := AgentSettlementIntentDigest(intent)
	require.NoError(t, err)
	return digest
}

func TestOperatorWithdrawalUsesPersistedOperatorOwnerAndCanonicalMandateAuthorization(t *testing.T) {
	service, ctx, now, session, _ := provisionTestMandateSession(t)
	baseWallet, err := service.Get(ctx, "local-operator", session.BudgetID)
	require.NoError(t, err)
	request := wallet.SessionSignAuthorization{
		WalletID: baseWallet.GetWalletId(), Network: baseWallet.GetNetwork(), OperatorIdentity: "local-operator",
		MandateID: session.MandateID, AuthorizationID: "auth-operator-withdrawal", SigningPurpose: "operator-withdrawal",
		Recipient: "0x0000000000000000000000000000000000000001", AtomicAmount: "1000000",
		Currency: "USD", ValueMinor: 100, AuthorizationValueMinor: 110,
		TransactionDigest: "sha256:" + strings.Repeat("f", 64),
	}
	require.NoError(t, insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: request.AuthorizationID, IdempotencyKey: request.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: request.AuthorizationValueMinor, Currency: request.Currency,
		Counterparty: request.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: request.AuthorizationValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}))
	_, err = service.DB.ExecContext(ctx, `INSERT INTO withdrawal_quotes(id,budget_id,operator_identity,public_proto,execution_quote,expires_at,created_at) VALUES(?,?,?,?,?,?,?)`,
		"quote-operator-withdrawal", session.BudgetID, "local-operator", []byte("operator-created-quote"), "{}", now.Add(time.Minute).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = service.DB.ExecContext(ctx, `INSERT INTO withdrawals(id,quote_id,operator_identity,idempotency_key,authorization_id,settlement_id,state,last_error,created_at) VALUES(?,?,?,?,?,?, 'authorized','',?)`,
		"withdrawal-operator", "quote-operator-withdrawal", "local-operator", "withdrawal-operator-key", request.AuthorizationID, "settlement-operator", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	unsigned := unsignedTestUSDCTransfer(t, request)
	signed, err := service.Wallets.SignUSDCTransfer(ctx, request, unsigned)
	require.NoError(t, err, "operator-authenticated withdrawal must use the canonical authorized mandate and durable withdrawal record")
	r, s, v := signed.RawSignatureValues()
	require.NotNil(t, r)
	require.NotNil(t, s)
	require.NotNil(t, v)

	denied := request
	denied.AuthorizationID = "auth-operator-withdrawal-no-row"
	denied.TransactionDigest = "sha256:" + strings.Repeat("1", 64)
	require.NoError(t, insertApprovedAuthorization(t, ctx, service, authorization.Record{
		ID: denied.AuthorizationID, IdempotencyKey: denied.AuthorizationID, BookID: "research", MandateID: session.MandateID,
		BudgetID: session.BudgetID, RequestingAgent: session.AgentSubject, AmountMinor: denied.AuthorizationValueMinor, Currency: denied.Currency,
		Counterparty: denied.Recipient, Verdict: authorization.VerdictApproved, HoldMinor: denied.AuthorizationValueMinor,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}))
	_, err = service.Wallets.SignUSDCTransfer(ctx, denied, unsignedTestUSDCTransfer(t, denied))
	require.Error(t, err, "the operator signer must fail closed without an operator-owned withdrawal lifecycle row")
}

func unsignedTestUSDCTransfer(t *testing.T, request wallet.SessionSignAuthorization) *types.Transaction {
	t.Helper()
	data, err := nativeusdc.TransferData(request.Recipient, request.AtomicAmount)
	require.NoError(t, err)
	asset := common.HexToAddress(nativeusdc.Asset)
	return types.NewTx(&types.DynamicFeeTx{
		ChainID: big.NewInt(8453), Nonce: 1, To: &asset, Gas: 100_000,
		GasFeeCap: big.NewInt(2_000_000_000), GasTipCap: big.NewInt(1_000_000), Data: data,
	})
}

func insertApprovedAuthorization(t *testing.T, ctx context.Context, service *Service, value authorization.Record) error {
	t.Helper()
	_, err := service.DB.ExecContext(ctx, `INSERT INTO authorizations(id,idempotency_key,mandate_id,budget_id,requesting_agent,amount_minor,currency,counterparty,verdict,violated_constraint,remediation,hold_minor,created_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.ID, value.IdempotencyKey, value.MandateID, value.BudgetID, value.RequestingAgent, value.AmountMinor, value.Currency, value.Counterparty,
		string(value.Verdict), value.ViolatedConstraint, value.Remediation, value.HoldMinor, value.CreatedAt.Format(time.RFC3339Nano), value.ExpiresAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = service.DB.ExecContext(ctx, `INSERT INTO authorization_book_bindings(authorization_id,book_id) VALUES(?,?)`, value.ID, value.BookID)
	return err
}

func provisionTestMandateSession(t *testing.T) (*Service, context.Context, time.Time, MandateSessionWallet, string) {
	t.Helper()
	service, ctx := setupWalletOpsTest(t)
	now := service.Clock.Now()
	policy, err := service.Budgets.Get(ctx, "research-budget")
	require.NoError(t, err)
	policy.AllowedCounterparties = append(policy.AllowedCounterparties, "0x0000000000000000000000000000000000000001")
	_, err = service.Budgets.SetCaps(ctx, policy)
	require.NoError(t, err)
	base, err := service.Provision(ctx, "local-operator", &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: "research-budget", Network: baseusdc.Network, AgentSubject: "agent:research", AgentRunId: "run-42",
	})
	require.NoError(t, err)
	_, err = service.Mandates.Issue(ctx, mandate.IssueInput{
		ID: "mandate-session-test", IdempotencyKey: "mandate-session-test-key", BookID: "research", BudgetID: "research-budget",
		AgentSubject: "agent:research", AgentRunID: "run-42", Authorizer: "local-operator", CapMinor: 1_000,
		Currency: "USD", AllowedCounterparties: []string{"vendor.example", "0x0000000000000000000000000000000000000001"}, ExpiresAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	value, err := service.ProvisionMandateSession(ctx, "local-operator", "mandate-session-test")
	require.NoError(t, err)
	require.Equal(t, base.GetAddress(), value.Address)
	claims := walletWriteClaims(t, now, "agent:research", "run-42", "research-budget")
	privateKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	publicKey := "0x" + hex.EncodeToString(gethcrypto.FromECDSAPub(&privateKey.PublicKey))
	require.NoError(t, service.BindAgentSessionPublicKey(ctx, value.BudgetID, value.MandateID, claims, publicKey))
	return service, ctx, now, value, "0x" + hex.EncodeToString(gethcrypto.FromECDSA(privateKey))
}

func intentForSession(session MandateSessionWallet, authorizationID, settlementID, instrumentID, idempotencyKey string, amount int64, currency, counterparty string) AgentSettlementIntent {
	return AgentSettlementIntent{
		AuthorizationID: authorizationID, MandateID: session.MandateID, BudgetID: session.BudgetID,
		WalletID: session.WalletID, WalletAddress: session.Address, Network: session.Network,
		AgentSubject: session.AgentSubject, AgentRunID: session.AgentRunID, SettlementID: settlementID,
		InstrumentID: instrumentID, IdempotencyKey: idempotencyKey, AmountMinor: amount, Currency: currency,
		Counterparty: counterparty, Generation: session.Generation, ExpiresAt: session.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
}

func sessionSignRequest(session MandateSessionWallet, authorizationID, digest string) wallet.SessionSignAuthorization {
	return wallet.SessionSignAuthorization{
		WalletID: session.WalletID, Network: session.Network, OperatorIdentity: session.OperatorIdentity,
		MandateID: session.MandateID, AuthorizationID: authorizationID, Recipient: "0x0000000000000000000000000000000000000001",
		AtomicAmount: "1000000", Currency: "USD", ValueMinor: 100, TransactionDigest: digest,
	}
}

func walletReadClaims(t *testing.T, now time.Time, subject, run, budgetID string) identity.Claims {
	t.Helper()
	grant := requestidentity.OperationGrant{
		ID: "wallet-read", Approver: "local-operator", Operations: []string{"treasury.read"},
		Targets: []string{"treasury:budget/" + budgetID}, ExpiresAt: now.Add(time.Hour),
	}
	encoded, err := json.Marshal(grant)
	require.NoError(t, err)
	return identity.Claims{
		Subject: subject, RunID: run, ExpiresAt: now.Add(time.Hour).Unix(),
		Scopes: []string{"treasury:read"}, Meta: map[string]string{"operation_grant": string(encoded)},
	}
}

func walletWriteClaims(t *testing.T, now time.Time, subject, run, budgetID string) identity.Claims {
	t.Helper()
	grant := requestidentity.OperationGrant{
		ID: "wallet-spend", Approver: "local-operator", ExpiresAt: now.Add(time.Hour),
		Rules: []requestidentity.OperationRule{
			{Operations: []string{"treasury.spend"}, Targets: []string{"treasury:settlement/*"}},
			{Operations: []string{"treasury.read"}, Targets: []string{"treasury:budget/" + budgetID}},
		},
	}
	encoded, err := json.Marshal(grant)
	require.NoError(t, err)
	return identity.Claims{
		Subject: subject, RunID: run, ExpiresAt: now.Add(time.Hour).Unix(),
		Scopes: []string{"treasury:read", "treasury:write"}, Meta: map[string]string{"operation_grant": string(encoded)},
	}
}
