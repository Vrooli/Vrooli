package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"treasury/internal/testutil/agentclaims"

	"connectrpc.com/connect"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	accountsV1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsConnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	approvalpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	budgetpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	instrumentpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	mandatepb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	settlementpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	walletpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/identity"
	"treasury/internal/nativeusdc"
	"treasury/internal/operatorauth"
	"treasury/internal/testutil/basefork"
	"treasury/internal/wallet"
	"treasury/internal/walletops"
)

// TestCF2LocalForkThroughBothOwners uses lifecycle-managed scenario processes,
// real native-USDC execution, signed service identity and persistent databases.
// The opt-in fixture directory belongs to the isolated Money Ledger rehearsal;
// no host credential or real-value wallet is touched.
func TestCF2LocalForkThroughBothOwners(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-002] [REQ:TRS-P0-006] [REQ:TRS-P0-009] [REQ:TRS-P0-011] [REQ:TRS-P1-006] [REQ:TRS-P1-007]
	if os.Getenv("TREASURY_CF2_REHEARSAL") != "1" {
		t.Skip("set TREASURY_CF2_REHEARSAL=1 and TREASURY_CF2_FIXTURE_DIR for the two-owner fork rehearsal")
	}
	fixtureDir := os.Getenv("TREASURY_CF2_FIXTURE_DIR")
	require.True(t, filepath.IsAbs(fixtureDir))
	var seed struct {
		URL     string            `json:"api_url"`
		Book    string            `json:"book_id"`
		Account string            `json:"account_id"`
		Variant string            `json:"variant"`
		Source  string            `json:"source_path"`
		Runtime map[string]string `json:"runtime_environment"`
	}
	readRehearsalJSON(t, filepath.Join(fixtureDir, "seed.json"), &seed)
	require.True(t, strings.HasPrefix(seed.URL, "http://127.0.0.1:"))
	fork := basefork.Start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	runDir := filepath.Join(filepath.Dir(fixtureDir), "cf2-"+time.Now().UTC().Format("20060102T150405Z"))
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	t.Logf("rehearsal evidence=%s", runDir)
	custodyDir := filepath.Join(runDir, "custody")
	credentialPath := filepath.Join(custodyDir, "synthetic-custody.enc")
	initializeCF2Credentials(t, credentialPath)
	t.Cleanup(func() { _ = os.RemoveAll(custodyDir) })
	token, jwksURL, exchangeURL, closeIdentity := startServiceIdentityFixture(t)
	defer closeIdentity()
	ledgerEnv := map[string]string{"GOFLAGS": ""}
	for key, value := range seed.Runtime {
		if strings.HasPrefix(key, "VROOLI_AUTH_") || key == "SCENARIO_DATA_DIR" || key == "SCENARIO_NAME" || key == "VROOLI_SCENARIO" || key == "VROOLI_STORAGE_NAMESPACE" {
			ledgerEnv[key] = value
		}
	}
	ledgerEnv["VROOLI_AUTH_SCENARIO_JWKS_URL"] = jwksURL
	lifecycleRehearsal(t, ctx, runDir, "ledger-fixture-restart", ledgerEnv, "restart", "money-ledger@"+seed.Variant, "--path", seed.Source, "--json", "--timeout", "240")
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := filepath.Clean(filepath.Join(cwd, "../../.."))

	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Token          string `json:"token"`
			OperationToken string `json:"operation_token"`
		}
		if r.URL.Path != "/api/v1/identity/verify" || json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "invalid fixture identity", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if in.Token == "cf2-local-agent" && in.OperationToken == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": agentclaims.For("cf2-agent")})
			return
		}
		baseline, baselineOK := verifyCF1RunToken(in.Token)
		if !baselineOK {
			t.Logf("CF1 identity fixture rejected baseline token")
			http.Error(w, "invalid local Agent Manager run identity", http.StatusUnauthorized)
			return
		}
		if in.OperationToken == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": baseline})
			return
		}
		operation, operationOK := verifyCF1RunToken(in.OperationToken)
		if !operationOK || baseline.RunID != operation.RunID || baseline.TaskID != operation.TaskID || baseline.Subject != operation.Subject || operation.Meta["authorization_id"] == "" || operation.Meta["operation_grant"] == "" {
			t.Logf("CF1 identity fixture rejected pair: operationValid=%t runMatches=%t taskMatches=%t subjectMatches=%t authorizationPresent=%t grantPresent=%t", operationOK, baseline.RunID == operation.RunID, baseline.TaskID == operation.TaskID, baseline.Subject == operation.Subject, operation.Meta["authorization_id"] != "", operation.Meta["operation_grant"] != "")
			http.Error(w, "invalid local Agent Manager identity pair", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid": true, "claims": baseline,
			"operation": map[string]any{"valid": true, "claims": operation},
		})
	}))
	defer identityServer.Close()
	variant := "cf2-" + fmt.Sprint(time.Now().Unix())
	lpbsVariant := "cf2-lpbs-" + fmt.Sprint(time.Now().UnixNano())
	lpbsDatabase := "vrooli_landing_page_business_suite_" + strings.ReplaceAll(lpbsVariant, "-", "_")
	resourceDatabaseCommand(t, ctx, runDir, "lpbs-database-create", "content", "create-database", lpbsDatabase)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		resourceDatabaseCommand(t, cleanup, runDir, "lpbs-database-remove", "content", "remove", lpbsDatabase)
	})
	lpbsSource := prepareLPBSRehearsal(t, repo, runDir, map[string]string{
		"LPBS_REHEARSAL_CREDENTIAL_PATH":     credentialPath,
		"LPBS_REHEARSAL_PAYER_IDENTITY":      "customer:cf2-local",
		"LPBS_REHEARSAL_PAYER_ADDRESS":       strings.ToLower(fork.Accounts[1]),
		"LPBS_REHEARSAL_BUSINESS_ACCOUNT_ID": "cf2-customer-" + variant,
		"LPBS_REHEARSAL_CUSTOMER_EMAIL":      "cf2-local@example.test",
	})
	lpbsName := "landing-page-business-suite@" + lpbsVariant
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		cmd := exec.CommandContext(cleanup, "vrooli", "scenario", "stop", lpbsName, "--json")
		output, err := cmd.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "lpbs-stop.json"), output, 0o600)
		if err != nil {
			t.Logf("LPBS rehearsal lifecycle cleanup: %v", err)
		}
	})
	lpbsStart := lifecycleRehearsal(t, ctx, runDir, "lpbs-start", map[string]string{"GOFLAGS": "-tags=lpbs_rehearsal"}, "start", lpbsName, "--path", lpbsSource, "--json", "--timeout", "240")
	lpbsURL := rehearsalURL(t, lpbsStart)
	leaseID := "cf2-lpbs-" + variant
	routingClient := routingconnect.NewRoutingServiceClient(http.DefaultClient, lpbsURL)
	_, err = routingClient.InstallTestPool(ctx, connect.NewRequest(&routingpb.InstallTestPoolRequest{Dsn: "file:cf2-lpbs-isolated", LeaseId: leaseID, LeaseTtlMs: int64((10 * time.Minute).Milliseconds())}))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_, clearErr := routingClient.ClearTestPool(cleanup, connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: leaseID}))
		if clearErr != nil {
			t.Logf("clear LPBS test lease: %v", clearErr)
		}
	})
	notificationRequests := make(chan *notificationsv1.SendRequest, 4)
	notificationMux := http.NewServeMux()
	notificationMux.Handle(notificationsconnect.NotificationsServiceSendProcedure, connect.NewUnaryHandler(
		notificationsconnect.NotificationsServiceSendProcedure,
		func(_ context.Context, request *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
			notificationRequests <- request.Msg
			return connect.NewResponse(&notificationsv1.SendResponse{}), nil
		},
	))
	notificationServer := httptest.NewServer(notificationMux)
	defer notificationServer.Close()
	source := prepareTreasuryRehearsal(t, repo, runDir, map[string]string{
		"TREASURY_REHEARSAL_DIR":              custodyDir,
		"TREASURY_BASE_RPC_URL":               fork.URL,
		"TREASURY_BASE_USDC_MODE":             "local-fork",
		"MONEY_LEDGER_API_URL":                seed.URL,
		"LANDING_PAGE_BUSINESS_SUITE_API_URL": lpbsURL,
		"SCENARIO_AUTHENTICATOR_API_URL":      exchangeURL,
		"AGENT_MANAGER_API_URL":               identityServer.URL,
		"NOTIFICATION_HUB_API_URL":            notificationServer.URL,
	})
	name := "treasury@" + variant
	start := lifecycleRehearsal(t, ctx, runDir, "treasury-start", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "start", name, "--path", source, "--json", "--timeout", "240")
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		cmd := exec.CommandContext(cleanup, "vrooli", "scenario", "stop", name, "--json")
		output, err := cmd.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "treasury-stop.json"), output, 0o600)
		if err != nil {
			t.Logf("rehearsal lifecycle cleanup: %v", err)
		}
	})
	apiURL := rehearsalURL(t, start)
	client := authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	agent := authconnect.NewAgentSpendClient(http.DefaultClient, apiURL)
	bookID := "cf2-book-" + variant
	budgetID := "cf2-budget-" + variant
	_, err = client.CreateBook(ctx, operatorRequest(&authpb.CreateBookRequest{Book: &bookpb.Book{Id: bookID, Name: "CF2 isolated research"}}))
	require.NoError(t, err)
	_, err = client.SetBudgetCaps(ctx, operatorRequest(&authpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 10000, PeriodicCapMinor: 10000, PerTransactionCapMinor: 10000, PeriodSeconds: 3600}}))
	require.NoError(t, err)
	cf1AMRunID, cf1AMAuthorizationID, cf1AMSettlementID, err := runCF1AgentManagerCLIForkShowcase(t, ctx, repo, runDir, apiURL, source, name, client, notificationRequests, fork, bookID, variant)
	require.NoError(t, err)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-agent-manager-cli-result.json"), map[string]any{
		"run_id": cf1AMRunID, "authorization_id": cf1AMAuthorizationID, "settlement_id": cf1AMSettlementID,
		"runner_to_cli": true, "local_fork": true,
	})
	walletResult, err := client.CreateResearchWallet(ctx, operatorRequest(&authpb.CreateResearchWalletRequest{Id: "cf2-wallet-" + variant, BookId: bookID, Network: baseusdc.Network}))
	require.NoError(t, err)
	w := walletResult.Msg.Wallet
	require.NotEmpty(t, w.Address)
	_, err = fork.Mint(ctx, fork.Accounts[0], "9000001")
	require.NoError(t, err)
	var originalSnapshot string
	require.NoError(t, fork.Call(ctx, "evm_snapshot", []any{}, &originalSnapshot))
	tx, err := fork.Transfer(ctx, fork.Accounts[0], w.Address, "1000001")
	require.NoError(t, err)
	var raw string
	require.NoError(t, fork.Call(ctx, "eth_getRawTransactionByHash", []any{tx}, &raw))
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x80"}, nil))
	valuation := &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-explicit-value-not-a-peg", ValuedAt: timestamppb.Now()}
	check, err := client.VerifyBaseUSDCReceipt(ctx, operatorRequest(&authpb.VerifyBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx}))
	require.NoError(t, err)
	require.Len(t, check.Msg.Transfers, 1)
	logIndex := check.Msg.Transfers[0].LogIndex
	submit := func(label, checkID, hash string, action ingestpb.CryptoReceiptAction, original, previous string, value *ingestpb.CryptoValuation, attribution *ingestpb.CryptoPayerAttribution) *fundingpb.ReceiptFundingDelivery {
		t.Helper()
		reason := label
		if action == ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL {
			reason = ""
		}
		response, err := client.SubmitReceiptFunding(ctx, operatorRequest(&fundingpb.SubmitReceiptFundingRequest{WalletId: w.Id, TransactionHash: hash, CheckId: checkID, LogIndex: logIndex, BudgetId: budgetID, LedgerBookId: seed.Book, LedgerAccountId: seed.Account, Action: action, OriginalReceiptId: original, PreviousReceiptId: previous, Reason: reason, Valuation: value, Attribution: attribution}))
		require.NoError(t, err)
		writeRehearsalProto(t, filepath.Join(runDir, label+".json"), response.Msg)
		require.Equal(t, fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED, response.Msg.Delivery.State, "%s: %s", label, response.Msg.Delivery.LastError)
		return response.Msg.Delivery
	}
	headroom := func(label, atoms string, credited int64, available ...int64) {
		t.Helper()
		availableMinor := credited
		if len(available) > 0 {
			availableMinor = available[0]
		}
		req := connect.NewRequest(&authpb.GetBudgetHeadroomRequest{BudgetId: budgetID})
		req.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
		response, err := agent.GetBudgetHeadroom(ctx, req)
		require.NoError(t, err)
		writeRehearsalProto(t, filepath.Join(runDir, label+"-headroom.json"), response.Msg)
		require.True(t, response.Msg.Headroom.FundingRequired)
		require.Equal(t, atoms, response.Msg.Headroom.FundingAtomicAmount)
		require.Equal(t, credited, response.Msg.Headroom.FundingCreditedMinor)
		require.Equal(t, availableMinor, response.Msg.Headroom.AvailableMinor)
		require.EqualValues(t, 10000, response.Msg.Headroom.TotalCapMinor)
	}
	original := submit("original", check.Msg.CheckId, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL, "", "", valuation, nil)
	headroom("original", "1000001", 100)
	replay := submit("replay", check.Msg.CheckId, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL, "", "", valuation, nil)
	require.Equal(t, original.Receipt.Id, replay.Receipt.Id)
	headroom("replay", "1000001", 100)

	// Republish the identical signed transaction into a different local block.
	var reverted bool
	require.NoError(t, fork.Call(ctx, "evm_revert", []any{originalSnapshot}, &reverted))
	require.True(t, reverted)
	require.NoError(t, fork.Call(ctx, "evm_mine", []any{}, nil))
	var removalSnapshot string
	require.NoError(t, fork.Call(ctx, "evm_snapshot", []any{}, &removalSnapshot))
	var reTx string
	require.NoError(t, fork.Call(ctx, "eth_sendRawTransaction", []any{raw}, &reTx))
	require.Equal(t, tx, reTx)
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x81"}, nil))
	reincludedCheck, err := client.RecheckBaseUSDCReceipt(ctx, operatorRequest(&authpb.RecheckBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx, PriorCheckId: check.Msg.CheckId}))
	require.NoError(t, err)
	require.Equal(t, walletpb.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION, reincludedCheck.Msg.Check.Relation)
	reincluded := submit("reinclusion", reincludedCheck.Msg.Check.Id, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION, original.Receipt.Id, original.Receipt.Id, valuation, nil)
	require.Equal(t, "0", reincluded.Receipt.AtomicDelta)
	headroom("reinclusion", "1000001", 100)

	correctionCheck, err := client.VerifyBaseUSDCReceipt(ctx, operatorRequest(&authpb.VerifyBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx}))
	require.NoError(t, err)
	correctedValue := proto.Clone(valuation).(*ingestpb.CryptoValuation)
	correctedValue.AmountMinor = 97
	correctedValue.BasisReference = "local-fork:documented-valuation-correction"
	corrected := submit("correction", correctionCheck.Msg.CheckId, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION, original.Receipt.Id, reincluded.Receipt.Id, correctedValue, nil)
	require.Equal(t, "0", corrected.Receipt.AtomicDelta)
	require.EqualValues(t, -3, corrected.Receipt.ValuationDeltaMinor)
	headroom("correction", "1000001", 97)

	// Remove that inclusion, then observe the changed canonical block. Merely
	// missing a transaction is insufficient evidence for an accounting debit.
	require.NoError(t, fork.Call(ctx, "evm_revert", []any{removalSnapshot}, &reverted))
	require.True(t, reverted)
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x82"}, nil))
	removedCheck, err := client.RecheckBaseUSDCReceipt(ctx, operatorRequest(&authpb.RecheckBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx, PriorCheckId: reincludedCheck.Msg.Check.Id}))
	require.NoError(t, err)
	writeRehearsalProto(t, filepath.Join(runDir, "removal-check.json"), removedCheck.Msg)
	require.Equal(t, walletpb.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_NOT_CANONICAL, removedCheck.Msg.Check.Relation)
	removed := submit("removal", removedCheck.Msg.Check.Id, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL, original.Receipt.Id, corrected.Receipt.Id, correctedValue, nil)
	require.Equal(t, "-1000001", removed.Receipt.AtomicDelta)
	require.EqualValues(t, -97, removed.Receipt.ValuationDeltaMinor)
	headroom("removal", "0", 0)

	// A distinct test customer pays a per-budget derived wallet through the
	// persisted one-tap funding intent. The payer comes from the verified
	// Transfer log; LPBS assigns the actual lot only after ledger acceptance.
	budgetWalletResponse, err := client.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf2-agent", AgentRunId: "test-run"}))
	require.NoError(t, err)
	budgetWallet := budgetWalletResponse.Msg.Wallet
	agentWalletRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID})
	agentWalletRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	agentWalletResponse, err := agent.GetBudgetWallet(ctx, agentWalletRequest)
	require.NoError(t, err)
	require.Equal(t, budgetWallet.Address, agentWalletResponse.Msg.Wallet.Address)
	require.Equal(t, "cf2-agent", agentWalletResponse.Msg.Wallet.AgentSubject)
	require.Equal(t, "test-run", agentWalletResponse.Msg.Wallet.AgentRunId)
	intentResponse, err := client.CreateFundingIntent(ctx, operatorRequest(&fundingpb.CreateFundingIntentRequest{
		Id: "customer-payment-intent", BudgetId: budgetID, AtomicAmount: "2500000", LedgerBookId: seed.Book, LedgerAccountId: seed.Account,
		Valuation:   &ingestpb.CryptoValuation{AmountMinor: 250, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-customer-value", ValuedAt: timestamppb.Now()},
		Attribution: &ingestpb.CryptoPayerAttribution{PayerAddress: strings.ToLower(fork.Accounts[1]), PayerIdentity: "customer:cf2-local"},
	}))
	require.NoError(t, err)
	require.Contains(t, intentResponse.Msg.Intent.PaymentUri, budgetWallet.Address)
	_, err = fork.Mint(ctx, fork.Accounts[1], "2500000")
	require.NoError(t, err)
	customerTx, err := fork.Transfer(ctx, fork.Accounts[1], budgetWallet.Address, "2500000")
	require.NoError(t, err)
	// Anvil's finalized tag advances only after enough local blocks have been
	// mined; a mined transaction alone remains merely included evidence.
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x80"}, nil))
	customerResponse, err := client.ConfirmFundingTransfer(ctx, operatorRequest(&fundingpb.ConfirmFundingTransferRequest{Id: intentResponse.Msg.Intent.Id, TransactionHash: customerTx}))
	require.NoError(t, err)
	require.Equal(t, "credited", customerResponse.Msg.Intent.State)
	customerDelivery, err := client.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: customerResponse.Msg.Intent.ReceiptDeliveryId}))
	require.NoError(t, err)
	customer := customerDelivery.Msg.Delivery
	require.Equal(t, fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED, customer.State)
	require.Equal(t, "2500000", customer.Receipt.AtomicDelta)
	require.Equal(t, strings.ToLower(fork.Accounts[1]), customer.Receipt.Attribution.PayerAddress)
	require.Empty(t, customer.Receipt.Attribution.CreditLotId, "immutable Money Ledger acceptance must not contain a pre-accept LPBS lot ID")
	require.Equal(t, "customer:cf2-local", customer.Receipt.Attribution.PayerIdentity)
	require.True(t, customer.Receipt.NoChargeback)
	foundCustomerNotification := false
	for index := 0; !foundCustomerNotification; index++ {
		select {
		case notification := <-notificationRequests:
			if strings.Contains(notification.GetBody(), "2.500000 USDC") && strings.Contains(notification.GetBody(), budgetID) {
				writeRehearsalProto(t, filepath.Join(runDir, "customer-funding-notification.json"), notification)
				foundCustomerNotification = true
				break
			}
			writeRehearsalProto(t, filepath.Join(runDir, fmt.Sprintf("preceding-notification-%02d.json", index+1)), notification)
		case <-ctx.Done():
			t.Fatal("accepted funding did not emit a matching Notification Hub event")
		}
	}
	headroom("customer-payment", "2500000", 250)

	// Rehearse the operator withdrawal through the same agent mandate,
	// authorization, approval, instrument and settlement owners used by spend.
	withdrawalDestination := strings.ToLower(fork.Accounts[2])
	err = fork.Call(ctx, "anvil_setBalance", []any{budgetWallet.Address, "0xde0b6b3a7640000"}, nil)
	require.NoError(t, err)
	_, err = client.SetWithdrawalDestination(ctx, operatorRequest(&fundingpb.SetWithdrawalDestinationRequest{Destination: &fundingpb.WithdrawalDestination{BudgetId: budgetID, Address: withdrawalDestination, Label: "CF2 local Coinbase test destination"}}))
	require.NoError(t, err)
	_, err = client.SetBudgetCaps(ctx, operatorRequest(&authpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 10000, PeriodicCapMinor: 10000, PerTransactionCapMinor: 100, PeriodSeconds: 3600}}))
	require.NoError(t, err)
	mandateResponse, err := client.CreateMandate(ctx, operatorRequest(&authpb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: "cf2-withdrawal-mandate", IdempotencyKey: "cf2-withdrawal-mandate-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 1000, Currency: "USD", AllowedCounterparties: []string{withdrawalDestination}, AgentSubject: "cf2-agent", AgentRunId: "test-run",
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	require.Equal(t, mandatepb.MandateStatus_MANDATE_STATUS_LIVE, mandateResponse.Msg.Mandate.Status)
	quoteResponse, err := client.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.Mandate.Id, Destination: withdrawalDestination,
		AtomicAmount: "1000000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 10,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-withdrawal-value", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	require.Equal(t, "local-fork", quoteResponse.Msg.Quote.ExecutionMode)
	withdrawalRequest := func(key string) *connect.Request[fundingpb.ExecuteWithdrawalRequest] {
		request := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.Quote.Id, IdempotencyKey: key})
		request.Header().Set("X-Treasury-Mandate-Identity-Token", "cf2-local-agent")
		return request
	}
	missingIdentity := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.Quote.Id, IdempotencyKey: "cf2-withdrawal-missing-identity"})
	_, err = client.ExecuteWithdrawal(ctx, missingIdentity)
	require.Error(t, err)
	wrongRealm := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.Quote.Id, IdempotencyKey: "cf2-withdrawal-mixed-realm"})
	wrongRealm.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	_, err = client.ExecuteWithdrawal(ctx, wrongRealm)
	require.Error(t, err, "operator auth must keep rejecting the ordinary agent realm header")
	wrongPrincipalQuote, err := client.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.Mandate.Id, Destination: withdrawalDestination,
		AtomicAmount: "1000000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 10,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-wrong-principal", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	wrongPrincipal := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: wrongPrincipalQuote.Msg.Quote.Id, IdempotencyKey: "cf2-withdrawal-wrong-principal"})
	wrongPrincipal.Header().Set("X-Treasury-Mandate-Identity-Token", "cf2-other-agent")
	wrongPrincipalResponse, err := client.ExecuteWithdrawal(ctx, wrongPrincipal)
	require.NoError(t, err)
	require.Equal(t, "refused", wrongPrincipalResponse.Msg.Withdrawal.State)
	require.Empty(t, wrongPrincipalResponse.Msg.Withdrawal.TransactionHash)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-wrong-principal-refused.json"), wrongPrincipalResponse.Msg.Withdrawal)
	pendingResponse, err := client.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, "pending_approval", pendingResponse.Msg.Withdrawal.State)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-pending-approval.json"), pendingResponse.Msg.Withdrawal)
	queued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: approvalpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	require.NoError(t, err)
	require.Len(t, queued.Msg.Approvals, 1)
	approved, err := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: queued.Msg.Approvals[0].Id, Resolution: approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	require.NoError(t, err)
	require.Equal(t, approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED, approved.Msg.Approval.Status)
	settledResponse, err := client.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	withdrawal := settledResponse.Msg.Withdrawal
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-settled.json"), withdrawal)
	require.Equal(t, "settled", withdrawal.State, "withdrawal settlement detail: %s", withdrawal.LastError)
	require.NotEmpty(t, withdrawal.TransactionHash)
	var chainReceipt struct {
		Status string `json:"status"`
	}
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{withdrawal.TransactionHash}, &chainReceipt))
	require.Equal(t, "0x1", chainReceipt.Status)
	settledRetry, err := client.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, withdrawal.TransactionHash, settledRetry.Msg.Withdrawal.TransactionHash)
	require.Equal(t, withdrawal.Id, settledRetry.Msg.Withdrawal.Id)
	headroom("after-allowlisted-withdrawal", "2500000", 250, 100)

	// CF1: bind an agent-held session key to the run-bound mandate and derived
	// budget address, then settle two exact agent-signed intents through the
	// normal authorization, approval, instrument and settlement owners.
	spendRecipient := strings.ToLower(fork.Accounts[1])
	spendMandateResponse, err := client.CreateMandate(ctx, operatorRequest(&authpb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: "cf1-agent-spend-mandate", IdempotencyKey: "cf1-agent-spend-mandate-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 140, Currency: "USD", AllowedCounterparties: []string{spendRecipient}, AgentSubject: "cf2-agent", AgentRunId: "test-run",
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	mandateID := spendMandateResponse.Msg.Mandate.Id
	sessionWalletResponse, err := client.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf2-agent", AgentRunId: "test-run", MandateId: mandateID,
	}))
	require.NoError(t, err)
	sessionWallet := sessionWalletResponse.Msg.Wallet
	require.Equal(t, budgetWallet.Address, sessionWallet.Address)
	agentSessionPrivateKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	agentSessionPublicKey := "0x" + fmt.Sprintf("%x", gethcrypto.FromECDSAPub(&agentSessionPrivateKey.PublicKey))
	agentSessionWalletRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID, MandateId: mandateID})
	agentSessionWalletRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	agentSessionWalletRequest.Header().Set(walletops.HeaderAgentSessionPublicKey, agentSessionPublicKey)
	agentSessionWalletReadback, err := agent.GetBudgetWallet(ctx, agentSessionWalletRequest)
	require.NoError(t, err)
	require.Equal(t, sessionWallet.Address, agentSessionWalletReadback.Msg.Wallet.Address)
	require.Equal(t, mandateID, agentSessionWalletReadback.Msg.Wallet.MandateId)
	require.Greater(t, sessionWallet.SessionExpiresAt.AsTime(), time.Now())
	require.LessOrEqual(t, time.Until(sessionWallet.SessionExpiresAt.AsTime()), 15*time.Minute)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-mandate-session-wallet.json"), sessionWallet)
	_, err = fork.Mint(ctx, sessionWallet.Address, "2000000")
	require.NoError(t, err)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{sessionWallet.Address, "0xde0b6b3a7640000"}, nil))

	quoteAdapter, err := nativeusdc.New(nil, nil, fork.URL, "local-fork")
	require.NoError(t, err)
	credentialAuthority, err := credentialauthority.OpenEncryptedFile(credentialPath, "CF2 local fork synthetic custody; never value-bearing")
	require.NoError(t, err)
	spendInstrumentIDs := map[int64]string{}
	for _, amount := range []int64{10, 30, 110} {
		atomicAmount := fmt.Sprint(amount * 10_000)
		quote, previewErr := quoteAdapter.Preview(ctx, wallet.Wallet{
			ID: sessionWallet.WalletId, Network: sessionWallet.Network, OperatorIdentity: "local-operator", Address: sessionWallet.Address,
		}, spendRecipient, atomicAmount, amount, "USD", "")
		require.NoError(t, previewErr)
		quoteBytes, marshalErr := json.Marshal(quote)
		require.NoError(t, marshalErr)
		credentialReference := fmt.Sprintf("vrooli/treasury/cf1-session-spend-%d", amount)
		credentialIdentity, parseErr := credentialauthority.ParseIdentity(credentialReference)
		require.NoError(t, parseErr)
		require.NoError(t, credentialAuthority.Put(credentialIdentity, "value", string(quoteBytes)))
		instrumentID := fmt.Sprintf("cf1-session-spend-%d", amount)
		_, registerErr := client.RegisterInstrument(ctx, operatorRequest(&authpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
			Id: instrumentID, MandateId: mandateID, Rail: "base-usdc", CredentialReference: credentialReference, Counterparty: spendRecipient,
		}}))
		require.NoError(t, registerErr)
		spendInstrumentIDs[amount] = instrumentID
	}
	agentProposal := func(id string, amount int64) *authpb.ProposeChargeResponse {
		t.Helper()
		request := connect.NewRequest(&authpb.ProposeChargeRequest{
			Id: id, IdempotencyKey: id + "-key", MandateId: mandateID,
			AmountMinor: amount, Currency: "USD", Counterparty: spendRecipient,
			SettlementId: id + "-settlement", InstrumentId: spendInstrumentIDs[amount], SettlementIdempotencyKey: id + "-settlement-key",
		})
		request.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
		response, proposalErr := agent.ProposeCharge(ctx, request)
		require.NoError(t, proposalErr)
		writeRehearsalProto(t, filepath.Join(runDir, id+"-authorization.json"), response.Msg)
		return response.Msg
	}
	agentSettlement := func(id string, authorizationID, instrumentID, evidenceLabel string) *authpb.ReportOutcomeResponse {
		t.Helper()
		getAuthorizationRequest := connect.NewRequest(&authpb.GetAuthorizationRequest{Id: authorizationID})
		getAuthorizationRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
		authorization, getAuthorizationErr := agent.GetAuthorization(ctx, getAuthorizationRequest)
		require.NoError(t, getAuthorizationErr)
		settlementID := id + "-settlement"
		idempotencyKey := id + "-settlement-key"
		intent := walletops.AgentSettlementIntent{
			AuthorizationID: authorizationID, MandateID: mandateID, BudgetID: budgetID,
			WalletID: sessionWallet.WalletId, WalletAddress: sessionWallet.Address, Network: sessionWallet.Network,
			AgentSubject: "cf2-agent", AgentRunID: "test-run", SettlementID: settlementID,
			InstrumentID: instrumentID, IdempotencyKey: idempotencyKey,
			AmountMinor: authorization.Msg.Authorization.AmountMinor, Currency: authorization.Msg.Authorization.Currency,
			Counterparty: authorization.Msg.Authorization.Counterparty, Generation: sessionWallet.SessionGeneration,
			ExpiresAt: sessionWallet.SessionExpiresAt.AsTime().UTC().Format(time.RFC3339Nano),
		}
		digest, digestErr := walletops.AgentSettlementIntentDigest(intent)
		require.NoError(t, digestErr)
		signature, signErr := gethcrypto.Sign(digest, agentSessionPrivateKey)
		require.NoError(t, signErr)
		signatureDigest := sha256.Sum256(signature)
		writeRehearsalJSON(t, filepath.Join(runDir, id+"-agent-session-proof.json"), map[string]string{
			"mandate_id": mandateID, "budget_wallet_id": sessionWallet.WalletId,
			"budget_wallet_address":                    sessionWallet.Address,
			"agent_session_signer":                     gethcrypto.PubkeyToAddress(agentSessionPrivateKey.PublicKey).Hex(),
			"session_generation":                       fmt.Sprint(sessionWallet.SessionGeneration),
			"intent_digest":                            "0x" + fmt.Sprintf("%x", digest),
			"signature_digest":                         "sha256:" + fmt.Sprintf("%x", signatureDigest[:]),
			"signature_dispatched_only_after_approval": "true",
		})
		request := connect.NewRequest(&authpb.ReportOutcomeRequest{
			AuthorizationId: authorizationID, SettlementId: settlementID, InstrumentId: instrumentID, IdempotencyKey: idempotencyKey,
		})
		request.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
		request.Header().Set(walletops.HeaderAgentSessionProof, "0x"+fmt.Sprintf("%x", signature))
		response, settleErr := agent.ReportOutcome(ctx, request)
		require.NoError(t, settleErr)
		writeRehearsalProto(t, filepath.Join(runDir, id+"-settlement-"+evidenceLabel+".json"), response.Msg)
		return response.Msg
	}
	inCapAuthorization := agentProposal("cf1-in-cap", 30)
	require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED, inCapAuthorization.Authorization.Verdict)
	var nonceBeforeInvalidAgentProof string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceBeforeInvalidAgentProof))
	invalidProofRequest := connect.NewRequest(&authpb.ReportOutcomeRequest{
		AuthorizationId: inCapAuthorization.Authorization.Id, SettlementId: "cf1-in-cap-settlement", InstrumentId: spendInstrumentIDs[30], IdempotencyKey: "cf1-in-cap-settlement-key",
	})
	invalidProofRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	invalidProofRequest.Header().Set(walletops.HeaderAgentSessionProof, "0x"+strings.Repeat("00", 65))
	_, invalidProofErr := agent.ReportOutcome(ctx, invalidProofRequest)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(invalidProofErr))
	var nonceAfterInvalidAgentProof string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceAfterInvalidAgentProof))
	require.Equal(t, nonceBeforeInvalidAgentProof, nonceAfterInvalidAgentProof, "invalid agent proof must not dispatch a transaction")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-invalid-agent-proof-no-dispatch.json"), map[string]string{
		"authorization_id": inCapAuthorization.Authorization.Id, "error_code": connect.CodeOf(invalidProofErr).String(),
		"nonce_before": nonceBeforeInvalidAgentProof, "nonce_after": nonceAfterInvalidAgentProof, "dispatch": "none",
	})
	inCapSettlement := agentSettlement("cf1-in-cap", inCapAuthorization.Authorization.Id, spendInstrumentIDs[30], "first-response")
	require.Equal(t, settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED, inCapSettlement.Settlement.Outcome)
	require.NotEmpty(t, inCapSettlement.Settlement.ExternalId)
	headroom("after-cf1-in-cap", "2500000", 250, 100)

	overCapAuthorization := agentProposal("cf1-over-cap", 110)
	require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING, overCapAuthorization.Authorization.Verdict)
	require.Equal(t, "cf1-over-cap-key", overCapAuthorization.Authorization.IdempotencyKey)
	overCapQueued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: approvalpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	require.NoError(t, err)
	require.Len(t, overCapQueued.Msg.Approvals, 1)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-over-cap-approval-queued.json"), overCapQueued.Msg.Approvals[0])
	overCapApproval, err := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: overCapQueued.Msg.Approvals[0].Id, Resolution: approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	require.NoError(t, err)
	require.Equal(t, approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED, overCapApproval.Msg.Approval.Status)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-over-cap-approval-approved.json"), overCapApproval.Msg.Approval)
	overCapSettlement := agentSettlement("cf1-over-cap", overCapAuthorization.Authorization.Id, spendInstrumentIDs[110], "first-response")
	require.Equal(t, settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED, overCapSettlement.Settlement.Outcome)
	var nonceAfterAgentSettlements string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceAfterAgentSettlements))
	overCapRetry := agentSettlement("cf1-over-cap", overCapAuthorization.Authorization.Id, spendInstrumentIDs[110], "before-restart-retry")
	require.Equal(t, overCapSettlement.Settlement.ExternalId, overCapRetry.Settlement.ExternalId)
	require.Equal(t, overCapSettlement.Settlement.Id, overCapRetry.Settlement.Id)
	var nonceAfterAgentRetry string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceAfterAgentRetry))
	require.Equal(t, nonceAfterAgentSettlements, nonceAfterAgentRetry, "same-id retry must not broadcast a second agent transaction")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-over-cap-retry-comparison.json"), map[string]string{
		"first_settlement_id":        overCapSettlement.Settlement.Id,
		"first_transaction_hash":     overCapSettlement.Settlement.ExternalId,
		"retry_settlement_id":        overCapRetry.Settlement.Id,
		"retry_transaction_hash":     overCapRetry.Settlement.ExternalId,
		"same_settlement":            "true",
		"same_transaction":           "true",
		"session_nonce_after_settle": nonceAfterAgentSettlements,
		"session_nonce_after_retry":  nonceAfterAgentRetry,
	})
	headroom("after-cf1-over-cap", "2500000", 250, 0)

	var agentReceipt struct {
		Status string `json:"status"`
		Logs   []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	for _, evidence := range []struct{ name, hash, atoms string }{
		{"cf1-in-cap", inCapSettlement.Settlement.ExternalId, "300000"},
		{"cf1-over-cap", overCapSettlement.Settlement.ExternalId, "1100000"},
	} {
		require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{evidence.hash}, &agentReceipt))
		require.Equal(t, "0x1", agentReceipt.Status)
		amount, ok := new(big.Int).SetString(evidence.atoms, 10)
		require.True(t, ok)
		expectedData := "0x" + fmt.Sprintf("%064x", amount)
		found := false
		for _, log := range agentReceipt.Logs {
			if strings.EqualFold(log.Address, nativeusdc.Asset) && len(log.Topics) == 3 && strings.EqualFold(log.Topics[1][26:], strings.TrimPrefix(sessionWallet.Address, "0x")) && strings.EqualFold(log.Topics[2][26:], strings.TrimPrefix(spendRecipient, "0x")) && strings.EqualFold(log.Data, expectedData) {
				found = true
			}
		}
		require.True(t, found, "%s receipt must contain the exact session-wallet USDC transfer", evidence.name)
	}

	var nonceBeforeDeniedSpend string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceBeforeDeniedSpend))
	_, err = client.FreezeBudget(ctx, operatorRequest(&authpb.FreezeBudgetRequest{BudgetId: budgetID}))
	require.NoError(t, err)
	frozenSpend := agentProposal("cf1-frozen-next-authorization", 10)
	require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED, frozenSpend.Authorization.Verdict)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-frozen-authorization-denied.json"), frozenSpend)
	deniedOutcomeRequest := connect.NewRequest(&authpb.ReportOutcomeRequest{
		AuthorizationId: frozenSpend.Authorization.Id, SettlementId: "cf1-frozen-next-authorization-settlement",
		InstrumentId: spendInstrumentIDs[10], IdempotencyKey: "cf1-frozen-next-authorization-settlement-key",
	})
	deniedOutcomeRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	_, deniedOutcomeErr := agent.ReportOutcome(ctx, deniedOutcomeRequest)
	require.Error(t, deniedOutcomeErr, "a refused authorization must not reach the signer or rail dispatch")
	var nonceAfterDeniedSpend string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceAfterDeniedSpend))
	require.Equal(t, nonceBeforeDeniedSpend, nonceAfterDeniedSpend, "denied calls cannot broadcast a transaction")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-frozen-no-dispatch.json"), map[string]string{
		"authorization_id":            frozenSpend.Authorization.Id,
		"authorization_verdict":       frozenSpend.Authorization.Verdict.String(),
		"report_outcome_error_code":   connect.CodeOf(deniedOutcomeErr).String(),
		"session_wallet_nonce_before": nonceBeforeDeniedSpend,
		"session_wallet_nonce_after":  nonceAfterDeniedSpend,
		"dispatch":                    "refused",
	})
	frozenQuote, err := client.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.Mandate.Id, Destination: withdrawalDestination,
		AtomicAmount: "500000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 5,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 50, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-frozen-withdrawal", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	frozenRequest := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: frozenQuote.Msg.Quote.Id, IdempotencyKey: "cf2-frozen-withdrawal-key"})
	frozenRequest.Header().Set("X-Treasury-Mandate-Identity-Token", "cf2-local-agent")
	frozenResponse, err := client.ExecuteWithdrawal(ctx, frozenRequest)
	require.NoError(t, err)
	require.Equal(t, "refused", frozenResponse.Msg.Withdrawal.State)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-frozen-refused.json"), frozenResponse.Msg.Withdrawal)
	require.Empty(t, frozenResponse.Msg.Withdrawal.TransactionHash)
	unfrozen, err := client.UnfreezeBudget(ctx, operatorRequest(&authpb.UnfreezeBudgetRequest{BudgetId: budgetID}))
	require.NoError(t, err)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-budget-unfrozen-for-exact-replay.json"), unfrozen.Msg)
	headroom("after-budget-freeze", "2500000", 250, 0)

	// Process restarts use the ordinary lifecycle owner, with the same isolated
	// databases and authority. New PIDs/build identities are retained in logs.
	lifecycleRehearsal(t, ctx, runDir, "ledger-restart", ledgerEnv, "restart", "money-ledger@"+seed.Variant, "--path", seed.Source, "--json", "--timeout", "240")
	restarted := lifecycleRehearsal(t, ctx, runDir, "treasury-restart", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", name, "--path", source, "--json", "--timeout", "240")
	apiURL = rehearsalURL(t, restarted)
	client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	agent = authconnect.NewAgentSpendClient(http.DefaultClient, apiURL)
	headroom("after-both-restarts", "2500000", 250, 0)
	for _, authorizationID := range []string{inCapAuthorization.Authorization.Id, overCapAuthorization.Authorization.Id} {
		readbackRequest := connect.NewRequest(&authpb.GetAuthorizationRequest{Id: authorizationID})
		readbackRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
		readback, readErr := agent.GetAuthorization(ctx, readbackRequest)
		require.NoError(t, readErr)
		require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED, readback.Msg.Authorization.Verdict)
	}
	retriedAgentSettlementAfterRestart := agentSettlement("cf1-over-cap", overCapAuthorization.Authorization.Id, spendInstrumentIDs[110], "after-restart-retry")
	require.Equal(t, settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED, retriedAgentSettlementAfterRestart.Settlement.Outcome)
	require.Equal(t, overCapSettlement.Settlement.ExternalId, retriedAgentSettlementAfterRestart.Settlement.ExternalId,
		"an approved agent settlement retried after Treasury restart must return the one original transaction")
	require.Equal(t, overCapSettlement.Settlement.Id, retriedAgentSettlementAfterRestart.Settlement.Id)
	var nonceAfterRestartRetry string
	require.NoError(t, fork.Call(ctx, "eth_getTransactionCount", []any{sessionWallet.Address, "latest"}, &nonceAfterRestartRetry))
	require.Equal(t, nonceAfterAgentRetry, nonceAfterRestartRetry, "restart replay must not broadcast a second agent transaction")
	sessionWalletReadbackRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID, MandateId: mandateID})
	sessionWalletReadbackRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	sessionWalletReadback, err := agent.GetBudgetWallet(ctx, sessionWalletReadbackRequest)
	require.NoError(t, err)
	require.Equal(t, sessionWallet.Address, sessionWalletReadback.Msg.Wallet.Address)
	require.Equal(t, sessionWallet.SessionGeneration, sessionWalletReadback.Msg.Wallet.SessionGeneration)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-over-cap-restart-retry-comparison.json"), map[string]string{
		"original_settlement_id":           overCapSettlement.Settlement.Id,
		"restarted_retry_settlement_id":    retriedAgentSettlementAfterRestart.Settlement.Id,
		"original_transaction_hash":        overCapSettlement.Settlement.ExternalId,
		"restarted_retry_transaction_hash": retriedAgentSettlementAfterRestart.Settlement.ExternalId,
		"same_settlement":                  "true",
		"same_transaction":                 "true",
		"session_nonce_before_restart":     nonceAfterAgentRetry,
		"session_nonce_after_restart":      nonceAfterRestartRetry,
	})
	rotatedWalletResponse, err := client.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf2-agent", AgentRunId: "test-run", MandateId: mandateID, RotateSession: true,
	}))
	require.NoError(t, err)
	rotatedWallet := rotatedWalletResponse.Msg.Wallet
	require.Equal(t, sessionWallet.SessionGeneration+1, rotatedWallet.SessionGeneration)
	require.Equal(t, sessionWallet.Address, rotatedWallet.Address, "rotation replaces signing authority without changing the derived budget address")
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-session-rotated.json"), rotatedWallet)
	revokeResponse, err := client.RevokeMandateSession(ctx, operatorRequest(&fundingpb.RevokeMandateSessionRequest{MandateId: mandateID}))
	require.NoError(t, err)
	require.True(t, revokeResponse.Msg.Revoked)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-session-revoked.json"), revokeResponse.Msg)
	revokedSessionReadRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID, MandateId: mandateID})
	revokedSessionReadRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	_, revokedSessionErr := agent.GetBudgetWallet(ctx, revokedSessionReadRequest)
	require.Error(t, revokedSessionErr, "revoked session must no longer be exposed to the agent")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-session-revoked-agent-read-refused.json"), map[string]string{
		"mandate_id":      mandateID,
		"read_error_code": connect.CodeOf(revokedSessionErr).String(),
	})
	persistedWithdrawal, err := client.GetWithdrawal(ctx, operatorRequest(&fundingpb.GetWithdrawalRequest{Id: withdrawal.Id}))
	require.NoError(t, err)
	require.Equal(t, "settled", persistedWithdrawal.Msg.Withdrawal.State)
	require.Equal(t, withdrawal.TransactionHash, persistedWithdrawal.Msg.Withdrawal.TransactionHash)
	withdrawalRetryAfterRestart, err := client.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, withdrawal.TransactionHash, withdrawalRetryAfterRestart.Msg.Withdrawal.TransactionHash)
	authorizationReadback := connect.NewRequest(&authpb.GetAuthorizationRequest{Id: withdrawal.AuthorizationId})
	authorizationReadback.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	verifiedAuthorization, err := agent.GetAuthorization(ctx, authorizationReadback)
	require.NoError(t, err)
	require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED, verifiedAuthorization.Msg.Authorization.Verdict)
	readback, err := client.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: removed.Id}))
	require.NoError(t, err)
	require.True(t, proto.Equal(removed, readback.Msg.Delivery))
	ledgerClient := ingestconnect.NewIngestServiceClient(http.DefaultClient, seed.URL)
	listReq := connect.NewRequest(&ingestpb.ListCryptoReceiptsRequest{BookId: seed.Book, Limit: 100})
	listReq.Header().Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	list, err := ledgerClient.ListCryptoReceipts(ctx, listReq)
	require.NoError(t, err)
	allReceipts := proto.Clone(list.Msg).(*ingestpb.ListCryptoReceiptsResponse)
	for list.Msg.NextAfterReceiptId != "" {
		listReq = connect.NewRequest(&ingestpb.ListCryptoReceiptsRequest{BookId: seed.Book, Limit: 100, AfterReceiptId: list.Msg.NextAfterReceiptId})
		listReq.Header().Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
		list, err = ledgerClient.ListCryptoReceipts(ctx, listReq)
		require.NoError(t, err)
		allReceipts.Receipts = append(allReceipts.Receipts, list.Msg.Receipts...)
		allReceipts.NextAfterReceiptId = list.Msg.NextAfterReceiptId
	}
	writeRehearsalProto(t, filepath.Join(runDir, "ledger-after-both-restarts.json"), allReceipts)
	var currentWalletReceipts []*ingestpb.AcceptedCryptoReceipt
	for _, receipt := range allReceipts.Receipts {
		if receipt.GetSource().GetWalletId() == w.Id || receipt.GetSource().GetWalletId() == budgetWallet.WalletId {
			currentWalletReceipts = append(currentWalletReceipts, receipt)
		}
	}
	require.Len(t, currentWalletReceipts, 5, "current showcase wallet should have exactly five accepted receipt events")
	require.True(t, proto.Equal(customer.Receipt, currentWalletReceipts[4]))
	linkedCustomer, err := client.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: customer.Id}))
	require.NoError(t, err)
	writeRehearsalProto(t, filepath.Join(runDir, "lpbs-credit-lot-link.json"), linkedCustomer.Msg.Delivery)
	require.Equal(t, customer.Receipt.Id, linkedCustomer.Msg.Delivery.CreditLotReceiptId)
	require.Equal(t, "landing-page-business-suite", linkedCustomer.Msg.Delivery.CreditLotOwner)
	require.NotEmpty(t, linkedCustomer.Msg.Delivery.CreditLotId, "LPBS must return an actual credit_funding_lots row ID")
	t.Log("five exact-token receipt events, actual LPBS credit_funding_lots association, mandate-bound agent session signing, in-cap and human-approved over-cap settlements, freeze denial, exact receipt attribution and idempotent replay survived Treasury restart")
}

type cf1RunIdentityClaims struct {
	Purpose   string            `json:"purpose,omitempty"`
	RunID     string            `json:"run_id"`
	TaskID    string            `json:"task_id"`
	Subject   string            `json:"subject"`
	Scopes    []string          `json:"scopes"`
	IssuedAt  int64             `json:"iat"`
	ExpiresAt int64             `json:"exp"`
	Meta      map[string]string `json:"meta"`
}

func verifyCF1RunToken(token string) (cf1RunIdentityClaims, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return cf1RunIdentityClaims{}, false
	}
	mac := hmac.New(sha256.New, []byte("local-fork-only-agent-manager-identity-secret"))
	_, _ = mac.Write([]byte(parts[0]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[1])) {
		return cf1RunIdentityClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return cf1RunIdentityClaims{}, false
	}
	var claims cf1RunIdentityClaims
	if json.Unmarshal(payload, &claims) != nil || claims.RunID == "" || claims.TaskID == "" || claims.Subject == "" || claims.ExpiresAt <= time.Now().Unix() {
		return cf1RunIdentityClaims{}, false
	}
	return claims, true
}

func runCF1AgentManagerCLIForkShowcase(t *testing.T, ctx context.Context, repo, runDir, apiURL, treasurySource, treasuryName string, client authconnect.TreasuryAdminClient, notificationRequests <-chan *notificationsv1.SendRequest, fork *basefork.Fork, bookID, variant string) (string, string, string, error) {
	t.Helper()
	coord := filepath.Join(runDir, "agent-manager-cli")
	if err := os.MkdirAll(coord, 0o700); err != nil {
		return "", "", "", err
	}
	budgetID := "cf1-agent-manager-budget-" + variant
	mandateID := "cf1-agent-manager-mandate-" + variant
	authorizationID := "cf1-agent-manager-auth-in-cap-" + variant
	settlementID := "cf1-agent-manager-settlement-in-cap-" + variant
	idempotency := "cf1-agent-manager-idem-in-cap-" + variant
	instrumentID := "cf1-agent-manager-instrument-in-cap-" + variant
	overCapAuthorizationID := "cf1-agent-manager-auth-over-cap-" + variant
	overCapSettlementID := "cf1-agent-manager-settlement-over-cap-" + variant
	overCapIdempotency := "cf1-agent-manager-idem-over-cap-" + variant
	overCapInstrumentID := "cf1-agent-manager-instrument-over-cap-" + variant
	counterparty := strings.ToLower(fork.Accounts[1])
	_, err := client.SetBudgetCaps(ctx, operatorRequest(&authpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{
		Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 1000,
		PeriodicCapMinor: 1000, PerTransactionCapMinor: 100, PeriodSeconds: 3600,
	}}))
	if err != nil {
		return "", "", "", fmt.Errorf("create ordinary-run Treasury budget: %w", err)
	}
	cliDir := filepath.Join(repo, "scenarios/treasury/cli")
	cliBinary := filepath.Join(coord, "treasury-cli")
	build := exec.CommandContext(ctx, "go", "build", "-o", cliBinary, ".")
	build.Dir = cliDir
	buildOutput, err := build.CombinedOutput()
	_ = os.WriteFile(filepath.Join(coord, "treasury-cli-build.log"), buildOutput, 0o600)
	if err != nil {
		return "", "", "", fmt.Errorf("build actual Treasury CLI: %w: %s", err, strings.TrimSpace(string(buildOutput)))
	}
	amDir := filepath.Join(repo, "scenarios/agent-manager/api")
	amTest := exec.CommandContext(ctx, "go", "test", "./internal/orchestration/integration", "-run", "^TestTreasuryCLIFromOrdinaryRun$", "-count=1", "-v")
	amTest.Dir = amDir
	amTest.Env = append(os.Environ(),
		"TREASURY_CF1_COORD_DIR="+coord,
		"TREASURY_CF1_BUDGET_ID="+budgetID,
		"TREASURY_CF1_MANDATE_ID="+mandateID,
		"TREASURY_CF1_COUNTERPARTY="+counterparty,
		"TREASURY_CF1_IN_CAP_AUTHORIZATION_ID="+authorizationID,
		"TREASURY_CF1_IN_CAP_SETTLEMENT_ID="+settlementID,
		"TREASURY_CF1_IN_CAP_INSTRUMENT_ID="+instrumentID,
		"TREASURY_CF1_IN_CAP_IDEMPOTENCY_KEY="+idempotency,
		"TREASURY_CF1_OVER_CAP_AUTHORIZATION_ID="+overCapAuthorizationID,
		"TREASURY_CF1_OVER_CAP_SETTLEMENT_ID="+overCapSettlementID,
		"TREASURY_CF1_OVER_CAP_INSTRUMENT_ID="+overCapInstrumentID,
		"TREASURY_CF1_OVER_CAP_IDEMPOTENCY_KEY="+overCapIdempotency,
	)
	amOutputPath := filepath.Join(coord, "agent-manager-go-test.log")
	amOutput, err := os.Create(amOutputPath)
	if err != nil {
		return "", "", "", err
	}
	amTest.Stdout = amOutput
	amTest.Stderr = amOutput
	if err := amTest.Start(); err != nil {
		_ = amOutput.Close()
		return "", "", "", fmt.Errorf("start Agent Manager ordinary-run showcase: %w", err)
	}
	childDone := make(chan error, 1)
	go func() {
		copyErr := amTest.Wait()
		_ = amOutput.Close()
		childDone <- copyErr
	}()
	var runIdentity struct {
		RunID string `json:"run_id"`
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "run-id.json"), &runIdentity, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for Agent Manager CreateRun: %w", err)
	}
	if strings.TrimSpace(runIdentity.RunID) == "" {
		return "", "", "", errors.New("Agent Manager run receipt omitted its run ID")
	}
	_, err = client.CreateMandate(ctx, operatorRequest(&authpb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: mandateID, IdempotencyKey: idempotency + "-mandate", BookId: bookID, BudgetId: budgetID,
		CapMinor: 300, Currency: "USD", AllowedCounterparties: []string{counterparty},
		AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID, ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	if err != nil {
		return "", "", "", fmt.Errorf("create run-bound Treasury mandate: %w", err)
	}
	walletResponse, err := client.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID, MandateId: mandateID,
	}))
	if err != nil {
		return "", "", "", fmt.Errorf("provision per-run budget address: %w", err)
	}
	if _, err := fork.Mint(ctx, walletResponse.Msg.Wallet.Address, "2000000"); err != nil {
		return "", "", "", fmt.Errorf("fund only the local-fork run wallet: %w", err)
	}
	if err := fork.Call(ctx, "anvil_setBalance", []any{walletResponse.Msg.Wallet.Address, "0xde0b6b3a7640000"}, nil); err != nil {
		return "", "", "", fmt.Errorf("fund local-fork gas: %w", err)
	}
	quoteAdapter, err := nativeusdc.New(nil, nil, fork.URL, "local-fork")
	if err != nil {
		return "", "", "", err
	}
	credentialPath := filepath.Join(runDir, "custody", "synthetic-custody.enc")
	credentialAuthority, err := credentialauthority.OpenEncryptedFile(credentialPath, "CF1 local fork synthetic custody; never value-bearing")
	if err != nil {
		return "", "", "", fmt.Errorf("open local Treasury custody fixture: %w", err)
	}
	registerRunInstrument := func(id string, amountMinor int64) error {
		atoms := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(10000)).String()
		quote, quoteErr := quoteAdapter.Preview(ctx, wallet.Wallet{ID: walletResponse.Msg.Wallet.WalletId, Network: baseusdc.Network, OperatorIdentity: "local-operator", Address: walletResponse.Msg.Wallet.Address}, counterparty, atoms, amountMinor, "USD", "")
		if quoteErr != nil {
			return fmt.Errorf("quote local run settlement %s: %w", id, quoteErr)
		}
		quoteBytes, marshalErr := json.Marshal(quote)
		if marshalErr != nil {
			return marshalErr
		}
		credentialReference := "vrooli/treasury/" + id
		credentialIdentity, parseErr := credentialauthority.ParseIdentity(credentialReference)
		if parseErr != nil {
			return parseErr
		}
		if putErr := credentialAuthority.Put(credentialIdentity, "value", string(quoteBytes)); putErr != nil {
			return fmt.Errorf("retain only local-fork rail quote: %w", putErr)
		}
		if _, registerErr := client.RegisterInstrument(ctx, operatorRequest(&authpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
			Id: id, MandateId: mandateID, Rail: "base-usdc", CredentialReference: credentialReference, Counterparty: counterparty,
		}})); registerErr != nil {
			return fmt.Errorf("register local run rail instrument %s: %w", id, registerErr)
		}
		return nil
	}
	if err := registerRunInstrument(instrumentID, 30); err != nil {
		return "", "", "", err
	}
	if err := registerRunInstrument(overCapInstrumentID, 110); err != nil {
		return "", "", "", err
	}
	config := map[string]any{
		"api_base_url": apiURL, "cli_binary": cliBinary, "budget_id": budgetID,
		"mandate_id": mandateID, "counterparty": counterparty,
		"in_cap_instrument_id": instrumentID, "in_cap_amount_minor": 30,
		"in_cap_authorization_id": authorizationID, "in_cap_settlement_id": settlementID, "in_cap_idempotency_key": idempotency,
		"over_cap_instrument_id": overCapInstrumentID, "over_cap_amount_minor": 110,
		"over_cap_authorization_id": overCapAuthorizationID, "over_cap_settlement_id": overCapSettlementID, "over_cap_idempotency_key": overCapIdempotency,
	}
	writeRehearsalJSON(t, filepath.Join(coord, "treasury-config.json"), config)
	var authorizationReady struct {
		AuthorizationID string `json:"authorization_id"`
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "authorization-ready.json"), &authorizationReady, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for run-owned CLI authorization: %w", err)
	}
	if authorizationReady.AuthorizationID != authorizationID {
		return "", "", "", errors.New("Agent Manager CLI authorization ID changed")
	}
	proposal, err := readCLIProposal(coord, "authorization-propose.log")
	if err != nil {
		return "", "", "", err
	}
	if proposal.GetAuthorization().GetId() != authorizationID || proposal.GetAuthorization().GetVerdict() != authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED || proposal.GetAuthorization().GetRequestingAgent() != "cf1-local-operator" || proposal.GetAuthorization().GetSettlementId() != settlementID || proposal.GetAuthorization().GetInstrumentId() != instrumentID || proposal.GetAuthorization().GetSettlementIdempotencyKey() != idempotency+"-settlement" {
		return "", "", "", fmt.Errorf("CLI authorization did not bind approved in-cap spend to run: %+v", proposal.GetAuthorization())
	}
	var nonceBeforeAlteredInstrument string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{walletResponse.Msg.Wallet.Address, "latest"}, &nonceBeforeAlteredInstrument); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run nonce before altered settlement attempt: %w", err)
	}
	if err := os.WriteFile(filepath.Join(coord, "settle-release"), []byte("AM local in-cap settlement"), 0o600); err != nil {
		return "", "", "", err
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "altered-instrument-denied.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("ordinary run did not reject its changed settlement instrument: %w", err)
	}
	var nonceAfterAlteredInstrument string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{walletResponse.Msg.Wallet.Address, "latest"}, &nonceAfterAlteredInstrument); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run nonce after altered settlement denial: %w", err)
	}
	if nonceBeforeAlteredInstrument != nonceAfterAlteredInstrument {
		return "", "", "", fmt.Errorf("altered-intent denial dispatched a transaction: nonce %s -> %s", nonceBeforeAlteredInstrument, nonceAfterAlteredInstrument)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "altered-intent-no-dispatch.json"), map[string]any{
		"nonce_before": nonceBeforeAlteredInstrument, "nonce_after": nonceAfterAlteredInstrument,
		"signature_created": false, "dispatch": "none",
	})
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "in-cap-settled.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for ordinary-run in-cap settlement: %w", err)
	}
	inCapSettlement, err := readCLISettlement(coord, "settlement-report-in-cap.log")
	if err != nil {
		return "", "", "", err
	}
	if err := requireCLILocalSettlement(inCapSettlement, settlementID); err != nil {
		return "", "", "", fmt.Errorf("validate ordinary-run in-cap settlement: %w", err)
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "over-cap-authorization-ready.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for ordinary-run over-cap proposal: %w", err)
	}
	overCapProposal, err := readCLIProposal(coord, "authorization-propose-over-cap.log")
	if err != nil {
		return "", "", "", err
	}
	if overCapProposal.GetAuthorization().GetId() != overCapAuthorizationID || overCapProposal.GetAuthorization().GetVerdict() != authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING || overCapProposal.GetAuthorization().GetAmountMinor() != 110 || overCapProposal.GetAuthorization().GetSettlementId() != overCapSettlementID || overCapProposal.GetAuthorization().GetInstrumentId() != overCapInstrumentID || overCapProposal.GetAuthorization().GetSettlementIdempotencyKey() != overCapIdempotency+"-settlement" {
		return "", "", "", fmt.Errorf("ordinary-run over-cap proposal did not raise a pending decision: %+v", overCapProposal.GetAuthorization())
	}
	queued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: approvalpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	if err != nil || len(queued.Msg.GetApprovals()) != 1 || queued.Msg.GetApprovals()[0].GetAuthorizationId() != overCapAuthorizationID {
		return "", "", "", fmt.Errorf("ordinary-run over-cap approval was not queued for its authorization: approvals=%v err=%v", queued, err)
	}
	approved, err := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: queued.Msg.GetApprovals()[0].GetId(), Resolution: approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	if err != nil || approved.Msg.GetApproval().GetStatus() != approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED {
		return "", "", "", fmt.Errorf("approve ordinary-run over-cap request: approval=%v err=%v", approved, err)
	}
	writeRehearsalProto(t, filepath.Join(coord, "over-cap-approval-approved.json"), approved.Msg.GetApproval())
	select {
	case notification := <-notificationRequests:
		if !strings.Contains(notification.GetBody(), "110 USD minor units") || !strings.Contains(notification.GetBody(), overCapAuthorizationID) {
			return "", "", "", fmt.Errorf("over-cap approval notification did not match the approved AM spend: %s", notification.GetBody())
		}
		writeRehearsalProto(t, filepath.Join(coord, "over-cap-approval-notification.json"), notification)
	case <-ctx.Done():
		return "", "", "", fmt.Errorf("over-cap decision notification was not delivered: %w", ctx.Err())
	}
	if err := os.WriteFile(filepath.Join(coord, "over-cap-settle-release"), []byte("human-approved local test spend"), 0o600); err != nil {
		return "", "", "", err
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "over-cap-settled.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for approved ordinary-run settlement: %w", err)
	}
	overCapSettlement, err := readCLISettlement(coord, "settlement-report-over-cap-first.log")
	if err != nil {
		return "", "", "", err
	}
	if err := requireCLILocalSettlement(overCapSettlement, overCapSettlementID); err != nil {
		return "", "", "", fmt.Errorf("validate approved ordinary-run settlement: %w", err)
	}
	// The ordinary AM run stays active while Treasury is restarted; its real CLI
	// retry runs against the replacement API after durable state is recovered.
	restarted := lifecycleRehearsal(t, ctx, runDir, "treasury-restart-during-agent-run", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
	if newURL := rehearsalURL(t, restarted); newURL == "" || newURL != apiURL {
		return "", "", "", errors.New("Treasury restart returned no API URL")
	}
	// The CLI is pinned to the same isolated variant address; rebind outer clients
	// through that address after the owner confirms its replacement process.
	client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	if err := os.WriteFile(filepath.Join(coord, "over-cap-retry-release"), []byte("retry same settlement after Treasury restart"), 0o600); err != nil {
		return "", "", "", err
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "over-cap-retry-after-restart.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for ordinary-run exact retry after restart: %w", err)
	}
	overCapRetry, err := readCLISettlement(coord, "settlement-report-over-cap-after-restart.log")
	if err != nil {
		return "", "", "", err
	}
	if overCapRetry.GetSettlement().GetId() != overCapSettlementID || overCapRetry.GetSettlement().GetExternalId() != overCapSettlement.GetSettlement().GetExternalId() || overCapRetry.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED {
		return "", "", "", fmt.Errorf("ordinary-run retry after restart changed or repeated settlement: %+v", overCapRetry.GetSettlement())
	}
	if _, err = client.FreezeBudget(ctx, operatorRequest(&authpb.FreezeBudgetRequest{BudgetId: budgetID})); err != nil {
		return "", "", "", fmt.Errorf("freeze budget before next authorization: %w", err)
	}
	var frozenWallet struct {
		Wallet struct {
			Address string `json:"address"`
		} `json:"wallet"`
	}
	walletOutput, err := os.ReadFile(filepath.Join(coord, "agent-wallet.log"))
	if err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run budget address before freeze denial: %w", err)
	}
	if err := json.Unmarshal(walletOutput, &frozenWallet); err != nil {
		return "", "", "", fmt.Errorf("decode ordinary-run budget address before freeze denial: %w", err)
	}
	if frozenWallet.Wallet.Address == "" {
		return "", "", "", fmt.Errorf("ordinary-run budget address before freeze denial is empty")
	}
	var nonceBeforeFrozenRequest string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{frozenWallet.Wallet.Address, "latest"}, &nonceBeforeFrozenRequest); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run wallet nonce before frozen request: %w", err)
	}
	if err := os.WriteFile(filepath.Join(coord, "freeze-next-auth-release"), []byte("freeze next authorization"), 0o600); err != nil {
		return "", "", "", err
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "frozen-next-authorization.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for ordinary-run frozen authorization: %w", err)
	}
	frozenProposal, err := readCLIProposal(coord, "authorization-propose-frozen.log")
	if err != nil {
		return "", "", "", err
	}
	if frozenProposal.GetAuthorization().GetVerdict() != authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED || frozenProposal.GetAuthorization().GetViolatedConstraint() != "budget_frozen" || frozenProposal.GetAuthorization().GetSettlementId() != settlementID+"-frozen" || frozenProposal.GetAuthorization().GetInstrumentId() != instrumentID || frozenProposal.GetAuthorization().GetSettlementIdempotencyKey() != idempotency+"-frozen-settlement" {
		return "", "", "", fmt.Errorf("frozen budget allowed ordinary-run authorization: %+v", frozenProposal.GetAuthorization())
	}
	var nonceAfterFrozenRequest string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{frozenWallet.Wallet.Address, "latest"}, &nonceAfterFrozenRequest); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run wallet nonce after frozen request: %w", err)
	}
	if nonceBeforeFrozenRequest != nonceAfterFrozenRequest {
		return "", "", "", fmt.Errorf("frozen request dispatched a transaction: nonce %s -> %s", nonceBeforeFrozenRequest, nonceAfterFrozenRequest)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "frozen-no-dispatch.json"), map[string]string{
		"authorization_id": frozenProposal.GetAuthorization().GetId(), "violated_constraint": frozenProposal.GetAuthorization().GetViolatedConstraint(),
		"nonce_before": nonceBeforeFrozenRequest, "nonce_after": nonceAfterFrozenRequest, "dispatch": "none",
	})
	select {
	case childErr := <-childDone:
		if childErr != nil {
			return "", "", "", fmt.Errorf("Agent Manager ordinary-run showcase failed: %w", childErr)
		}
	case <-ctx.Done():
		return "", "", "", fmt.Errorf("Agent Manager ordinary-run showcase timed out: %w", ctx.Err())
	}
	var runReceipt struct {
		RunID string `json:"run_id"`
	}
	readRehearsalJSON(t, filepath.Join(coord, "agent-manager-result.json"), &runReceipt)
	if runReceipt.RunID != runIdentity.RunID {
		return "", "", "", errors.New("Agent Manager result run ID did not match CreateRun receipt")
	}
	writeCLIReceipt := func(label string, settlement *authpb.ReportOutcomeResponse) error {
		var receipt struct {
			TransactionHash string `json:"transactionHash"`
			BlockHash       string `json:"blockHash"`
			BlockNumber     string `json:"blockNumber"`
			Status          string `json:"status"`
		}
		rawReceipt, callErr := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{settlement.GetSettlement().GetExternalId()})
		if callErr != nil {
			return fmt.Errorf("read %s CLI settlement receipt: %w", label, callErr)
		}
		if decodeErr := json.Unmarshal(rawReceipt, &receipt); decodeErr != nil {
			return fmt.Errorf("decode %s CLI settlement receipt: %w", label, decodeErr)
		}
		if receipt.Status != "0x1" || !strings.EqualFold(receipt.TransactionHash, settlement.GetSettlement().GetExternalId()) || receipt.BlockHash == "" || receipt.BlockNumber == "" {
			return fmt.Errorf("%s CLI transaction receipt is incomplete or unsuccessful: status=%q", label, receipt.Status)
		}
		writeRehearsalJSON(t, filepath.Join(coord, label+"-chain-receipt.json"), map[string]any{
			"transaction_hash": receipt.TransactionHash, "block_hash": receipt.BlockHash,
			"block_number": receipt.BlockNumber, "status": receipt.Status,
			"chain": "local-base-fork", "rpc_result": json.RawMessage(rawReceipt),
		})
		return nil
	}
	if err := writeCLIReceipt("in-cap", inCapSettlement); err != nil {
		return "", "", "", err
	}
	if err := writeCLIReceipt("over-cap", overCapSettlement); err != nil {
		return "", "", "", err
	}
	return runIdentity.RunID, authorizationID, settlementID, nil
}

func readCLIProposal(coord, name string) (*authpb.ProposeChargeResponse, error) {
	data, err := os.ReadFile(filepath.Join(coord, name))
	if err != nil {
		return nil, err
	}
	response := &authpb.ProposeChargeResponse{}
	if err := protojson.Unmarshal(data, response); err != nil {
		return nil, fmt.Errorf("decode CLI proposal %s: %w", name, err)
	}
	return response, nil
}

func readCLISettlement(coord, name string) (*authpb.ReportOutcomeResponse, error) {
	data, err := os.ReadFile(filepath.Join(coord, name))
	if err != nil {
		return nil, err
	}
	response := &authpb.ReportOutcomeResponse{}
	if err := protojson.Unmarshal(data, response); err != nil {
		return nil, fmt.Errorf("decode CLI settlement %s: %w", name, err)
	}
	return response, nil
}

func requireCLILocalSettlement(response *authpb.ReportOutcomeResponse, settlementID string) error {
	if response == nil || response.GetSettlement().GetId() != settlementID || response.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED || response.GetSettlement().GetExternalId() == "" {
		return fmt.Errorf("CLI did not settle local-fork charge %s: %+v", settlementID, response.GetSettlement())
	}
	return nil
}

func waitForCF1FileOrChild(ctx context.Context, path string, target any, childDone <-chan error) error {
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if target == nil {
				return nil
			}
			return json.Unmarshal(data, target)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case err := <-childDone:
			if err != nil {
				return fmt.Errorf("Agent Manager child exited before writing %s: %w", filepath.Base(path), err)
			}
			return fmt.Errorf("Agent Manager child exited before writing %s", filepath.Base(path))
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

type rehearsalServiceCredentialExchange struct {
	accountsConnect.UnimplementedAccountsServiceHandler
	token  string
	expiry time.Time
}

func (s *rehearsalServiceCredentialExchange) ExchangeServiceCredential(_ context.Context, request *connect.Request[accountsV1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsV1.ExchangeServiceCredentialResponse], error) {
	if request.Msg.GetCredential() != "cf2-local-treasury-service-only" || !time.Now().Before(s.expiry) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("rehearsal credential is invalid or expired"))
	}
	return connect.NewResponse(&accountsV1.ExchangeServiceCredentialResponse{AccessToken: s.token, AccessTokenExpiresAt: timestamppb.New(s.expiry), Audience: "scenario-authenticator:money-ledger", Subject: "treasury", Scopes: []string{"money-ledger:write"}}), nil
}

func startServiceIdentityFixture(t *testing.T) (token, jwksURL, exchangeURL string, closeServers func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	expiry := time.Now().UTC().Add(2 * time.Hour)
	encode := func(value any) string {
		raw, marshalErr := json.Marshal(value)
		require.NoError(t, marshalErr)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "cf2-local"}) + "." + encode(map[string]any{
		"user_id": "treasury", "sub": "treasury", "actor_kind": "service", "scope": []string{"money-ledger:write"},
		"aud": "scenario-authenticator:money-ledger", "iss": "scenario-authenticator", "iat": time.Now().Unix(), "exp": expiry.Unix(),
	})
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	token = unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	jwks := map[string]any{"keys": []any{map[string]string{
		"kty": "RSA", "kid": "cf2-local", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	public, err := json.Marshal(jwks)
	require.NoError(t, err)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(public)
	}))
	exchangeMux := http.NewServeMux()
	path, handler := accountsConnect.NewAccountsServiceHandler(&rehearsalServiceCredentialExchange{token: token, expiry: expiry})
	exchangeMux.Handle(path, handler)
	exchangeServer := httptest.NewServer(exchangeMux)
	return token, jwksServer.URL + "/jwks", exchangeServer.URL, func() { exchangeServer.Close(); jwksServer.Close() }
}

func operatorRequest[T any](value *T) *connect.Request[T] {
	request := connect.NewRequest(value)
	request.Header().Set(operatorauth.HeaderOperatorToken, "cf2-local-operator-only")
	return request
}

func readRehearsalJSON(t *testing.T, path string, out any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, out))
}

func writeRehearsalProto(t *testing.T, path string, value proto.Message) {
	t.Helper()
	raw, err := protojson.MarshalOptions{Indent: "  "}.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

func writeRehearsalJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

func initializeCF2Credentials(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	authority, err := credentialauthority.OpenEncryptedFile(path, "CF2 local fork synthetic custody; never value-bearing")
	require.NoError(t, err)
	for _, value := range []struct{ identity, field, secret string }{
		{"vrooli/treasury", "operator-token", "cf2-local-operator-only"},
		{"vrooli/treasury", "mandate-signing-key", "cf2-local-mandate-signing-key-only"},
		{"scenario-authenticator/service-principal/treasury/money-ledger", "credential", "cf2-local-treasury-service-only"},
		{"vrooli/landing-page-business-suite", "service-secret", "cf2-local-lpbs-service-only"},
	} {
		identity, parseErr := credentialauthority.ParseIdentity(value.identity)
		require.NoError(t, parseErr)
		require.NoError(t, authority.Put(identity, value.field, value.secret))
	}
}

func lifecycleRehearsal(t *testing.T, ctx context.Context, dir, label string, environment map[string]string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(ctx, "vrooli", append([]string{"scenario"}, args...)...)
	cmd.Env = os.Environ()
	for key, value := range environment {
		prefix := key + "="
		filtered := cmd.Env[:0]
		for _, entry := range cmd.Env {
			if !strings.HasPrefix(entry, prefix) {
				filtered = append(filtered, entry)
			}
		}
		cmd.Env = append(filtered, prefix+value)
	}
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	output, err := cmd.Output()
	require.NoError(t, os.WriteFile(filepath.Join(dir, label+".json"), output, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, label+".stderr.log"), diagnostics.Bytes(), 0o600))
	require.NoError(t, err, "lifecycle %s: %s\n%s", label, output, diagnostics.String())
	return output
}

func resourceDatabaseCommand(t *testing.T, ctx context.Context, dir, label string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "resource-postgres", append([]string(nil), args...)...)
	output, err := cmd.CombinedOutput()
	path := filepath.Join(dir, label+".log")
	require.NoError(t, os.WriteFile(path, output, 0o600))
	require.NoError(t, err, "resource-postgres %s: %s", strings.Join(args, " "), output)
}

func rehearsalURL(t *testing.T, raw []byte) string {
	t.Helper()
	var result struct {
		Success   bool `json:"success"`
		Scenarios []struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenarios"`
		Scenario struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Success)
	if len(result.Scenarios) == 1 {
		result.Scenario.Ports = result.Scenarios[0].Ports
	}
	require.NotZero(t, result.Scenario.Ports["API_PORT"])
	return fmt.Sprintf("http://127.0.0.1:%d", result.Scenario.Ports["API_PORT"])
}

func prepareTreasuryRehearsal(t *testing.T, repo, runDir string, environment map[string]string) string {
	t.Helper()
	workspace := filepath.Join(runDir, "workspace")
	source := filepath.Join(workspace, "scenarios", "treasury")
	require.NoError(t, os.MkdirAll(source, 0o700))
	entries, err := os.ReadDir(repo)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() != "scenarios" {
			require.NoError(t, os.Symlink(filepath.Join(repo, entry.Name()), filepath.Join(workspace, entry.Name())))
		}
	}
	for _, name := range []string{"api", "cli", ".vrooli"} {
		original := filepath.Join(repo, "scenarios", "treasury", name)
		require.NoError(t, filepath.WalkDir(original, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(original, path)
			if err != nil {
				return err
			}
			if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "data" || entry.Name() == "tmp") {
				return filepath.SkipDir
			}
			dest := filepath.Join(source, name, rel)
			if entry.IsDir() {
				return os.MkdirAll(dest, 0o700)
			}
			if strings.HasSuffix(entry.Name(), "-api") || strings.HasPrefix(entry.Name(), ".vrooli-closure") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
			if err != nil {
				return err
			}
			_, err = io.Copy(out, in)
			closeErr := out.Close()
			if err != nil {
				return err
			}
			return closeErr
		}))
	}
	manifest := filepath.Join(source, ".vrooli", "service.json")
	var config map[string]any
	readRehearsalJSON(t, manifest, &config)
	components := config["components"].(map[string]any)
	delete(components, "ui")
	components["api"].(map[string]any)["run"].(map[string]any)["env"] = environment
	delete(config["ports"].(map[string]any), "ui")
	config["credentials"] = map[string]any{}
	config["dependencies"] = map[string]any{"scenarios": map[string]any{}, "resources": map[string]any{}}
	config["cli"].(map[string]any)["enabled"] = false
	health := config["lifecycle"].(map[string]any)["health"].(map[string]any)
	health["checks"] = []any{map[string]any{"name": "api_endpoint", "type": "http", "target": "http://localhost:${API_PORT}/health", "critical": true, "timeout": 10000}}
	health["endpoints"] = map[string]string{"api": "/health"}
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
	return source
}

func prepareLPBSRehearsal(t *testing.T, repo, runDir string, environment map[string]string) string {
	t.Helper()
	workspace := filepath.Join(runDir, "lpbs-workspace")
	source := filepath.Join(workspace, "scenarios", "landing-page-business-suite")
	require.NoError(t, os.MkdirAll(source, 0o700))
	entries, err := os.ReadDir(repo)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() != "scenarios" {
			require.NoError(t, os.Symlink(filepath.Join(repo, entry.Name()), filepath.Join(workspace, entry.Name())))
		}
	}
	original := filepath.Join(repo, "scenarios", "landing-page-business-suite")
	require.NoError(t, filepath.WalkDir(original, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(original, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() && (entry.Name() == "ui" || entry.Name() == "node_modules" || entry.Name() == "data" || entry.Name() == "tmp") {
			return filepath.SkipDir
		}
		dest := filepath.Join(source, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o700)
		}
		if strings.HasSuffix(entry.Name(), "-api") || strings.HasPrefix(entry.Name(), ".vrooli-closure") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	}))
	manifest := filepath.Join(source, ".vrooli", "service.json")
	var config map[string]any
	readRehearsalJSON(t, manifest, &config)
	components := config["components"].(map[string]any)
	delete(components, "ui")
	components["api"].(map[string]any)["run"].(map[string]any)["env"] = environment
	delete(config["ports"].(map[string]any), "ui")
	config["cli"].(map[string]any)["enabled"] = false
	dependencies := config["dependencies"].(map[string]any)
	delete(dependencies["scenarios"].(map[string]any), "ai-gateway")
	health := config["lifecycle"].(map[string]any)["health"].(map[string]any)
	health["checks"] = []any{map[string]any{"name": "api_endpoint", "type": "http", "target": "http://localhost:${API_PORT}/health", "critical": true, "timeout": 10000}}
	health["endpoints"] = map[string]string{"api": "/health"}
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
	return source
}
