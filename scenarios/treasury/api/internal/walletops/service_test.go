package walletops

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	"treasury/internal/mandate"
	"treasury/internal/nativeusdc"
	"treasury/internal/receiptfunding"
	"treasury/internal/wallet"
)

type walletopsTestStore map[string]string

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

	rotated, err := service.RotateMandateSession(ctx, "local-operator", session.MandateID)
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

	rotated, err := service.RotateMandateSession(ctx, "local-operator", session.MandateID)
	require.NoError(t, err)
	require.Equal(t, session.Generation+1, rotated.Generation)
	require.Equal(t, session.Address, rotated.Address)
	err = service.AuthorizeAgentSettlement(ctx, claims, oldIntent, oldSignature)
	require.ErrorIs(t, err, ErrSessionInactive, "rotation retires the old generation before any new key binding")

	publicKey, err := gethcrypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	require.NoError(t, err)
	claimsRead := walletReadClaims(t, now, session.AgentSubject, session.AgentRunID, session.BudgetID)
	require.NoError(t, service.BindAgentSessionPublicKey(ctx, session.BudgetID, session.MandateID, claimsRead, "0x"+hex.EncodeToString(gethcrypto.FromECDSAPub(&publicKey.PublicKey))))
	newIntent := oldIntent
	newIntent.SettlementID = "settlement-new-generation"
	newIntent.Generation = rotated.Generation
	newIntent.ExpiresAt = rotated.ExpiresAt.UTC().Format(time.RFC3339Nano)
	newSignature, err := signAgentIntent(privateKeyHex, newIntent)
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
