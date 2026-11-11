//go:build treasury_rehearsal

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	requestidentity "github.com/vrooli/api-core/identity"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	notificationsshared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
	approvalpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	mandatepb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/identity"
	"treasury/internal/testutil/agentclaims"
	"treasury/internal/testutil/basefork"
)

// TestCF5MobileLocalForkReceiptWatcherQueuesReceipt is the gated local-fork
// CF5 rehearsal: BAS captures the funding, receipt-credit and withdrawal
// surfaces while isolated Treasury, Money Ledger, Notification Hub and a Base
// fork exercise acceptance, exactly-once headroom credit, notice delivery and
// allowlisted withdrawal. The opt-in runtime still needs a fresh shared disk
// lease before it can produce evidence for those expected outcomes.
func TestCF5MobileLocalForkReceiptWatcherQueuesReceipt(t *testing.T) {
	if os.Getenv("TREASURY_CF5_REHEARSAL") != "1" {
		t.Skip("set TREASURY_CF5_REHEARSAL=1 and TREASURY_CF5_FIXTURE_DIR for the isolated CF5 fork handoff")
	}
	runRoot := strings.TrimSpace(os.Getenv("TREASURY_CF5_FIXTURE_DIR"))
	require.True(t, filepath.IsAbs(runRoot), "fixture directory must be absolute")
	require.NoError(t, os.MkdirAll(runRoot, 0o700))
	runDir := filepath.Join(runRoot, "cf5-handoff-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	t.Logf("CF5 handoff evidence=%s", runDir)

	// BAS is a render, so admit the whole fork/Treasury/browser run before any
	// process starts and heartbeat it through cleanup.
	const cf5PeakBytes int64 = 2 * 1024 * 1024 * 1024
	admitTreasuryRehearsalDisk(t, runRoot, runDir, "crypto-foundations:CF5-local-fork-mobile-funding", "CF5 local-fork BAS mobile funding rehearsal", cf5PeakBytes)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	fork := basefork.Start(t)
	forkIdentity := fork.Identity()
	require.Equal(t, "0x2105", forkIdentity.ChainID)
	require.True(t, strings.HasPrefix(forkIdentity.URL, "http://127.0.0.1:"))
	require.NotEmpty(t, forkIdentity.BlockNumber)
	require.NotEmpty(t, forkIdentity.BlockHash)
	require.NotZero(t, forkIdentity.AnvilPID)
	writeCF5JSON(t, filepath.Join(runDir, "fork-identity.json"), forkIdentity)

	custodyDir := filepath.Join(runDir, "custody")
	initializeCF2Credentials(t, filepath.Join(custodyDir, "synthetic-custody.enc"))
	t.Cleanup(func() { _ = os.RemoveAll(custodyDir) })
	_, jwksURL, authenticatorURL, closeIdentity := startServiceIdentityFixture(t)
	t.Cleanup(closeIdentity)
	ledgerVariant := instanceName("money-ledger")
	ledgerName := "money-ledger@" + ledgerVariant
	ledgerEnv := map[string]string{
		"GOFLAGS": "", "API_TOKEN": "",
		"VROOLI_AUTH_SCENARIO_JWKS_URL":  jwksURL,
		"SCENARIO_AUTHENTICATOR_API_URL": authenticatorURL,
	}
	ledgerSource := prepareCF5APIScenarioVariant(t, "money-ledger", "../../money-ledger", runDir, ledgerEnv)
	ledgerStarted := lifecycleRehearsal(t, ctx, runDir, "money-ledger-start", map[string]string{"GOFLAGS": "", "API_TOKEN": ""}, "start", ledgerName, "--path", ledgerSource, "--json", "--timeout", "300")
	ledgerURL, _ := cf5RuntimeURLs(t, ledgerStarted)
	ledgerStopped := false
	stopLedger := func() {
		if ledgerStopped {
			return
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Minute)
		defer stopCancel()
		cmd := exec.CommandContext(stopCtx, "vrooli", "scenario", "stop", ledgerName, "--json")
		output, err := cmd.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "money-ledger-stop.json"), output, 0o600)
		if err != nil {
			t.Logf("stop isolated Money Ledger variant: %v", err)
		}
		ledgerStopped = true
	}
	t.Cleanup(stopLedger)
	requireHTTPStatus(t, ledgerURL+"/health", http.StatusOK)
	ledgerBooks := ledgerconnect.NewBooksServiceClient(&http.Client{Timeout: 10 * time.Second}, ledgerURL)
	createdBook, err := ledgerBooks.CreateBook(ctx, connect.NewRequest(&ledgerpb.CreateBookRequest{Name: "CF5 isolated Base funding", Currency: "USD"}))
	require.NoError(t, err)
	createdAccount, err := ledgerBooks.CreateAccount(ctx, connect.NewRequest(&ledgerpb.CreateAccountRequest{BookId: createdBook.Msg.GetBook().GetId(), Name: "Base USDC deposits", AccountKind: ledgerpb.AccountKind_ASSET}))
	require.NoError(t, err)
	writeCF5JSON(t, filepath.Join(runDir, "money-ledger-fixture.json"), map[string]string{
		"variant": ledgerVariant, "api_url": ledgerURL, "book_id": createdBook.Msg.GetBook().GetId(), "account_id": createdAccount.Msg.GetAccount().GetId(), "service_identity": "treasury / money-ledger:write via local synthetic JWKS fixture",
	})
	identityClaims := agentclaims.For("cf2-agent")
	identityClaims.Scopes = append(identityClaims.Scopes, "notification-hub:write")
	grant := requestidentity.OperationGrant{
		ID: "cf5-isolated-producer-grant", Approver: "cf5-test-operator",
		Operations: []string{"treasury.read", "treasury.spend", "notification.send"},
		Targets:    []string{"treasury:*", "notification-hub:recipient/e1-fixture@notification-hub.invalid"},
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	grantJSON, err := json.Marshal(grant)
	require.NoError(t, err)
	identityClaims.Meta["operation_grant"] = string(grantJSON)
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Token string `json:"token"`
		}
		if r.URL.Path != "/api/v1/identity/verify" || json.NewDecoder(r.Body).Decode(&request) != nil || request.Token != "cf5-local-agent" {
			http.Error(w, "invalid isolated CF5 agent", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": identityClaims})
	}))
	t.Cleanup(identityServer.Close)
	hubInstance := instanceName("notification-hub")
	hubName := "notification-hub@" + hubInstance
	hubEnv := map[string]string{
		"VROOLI_TEST_MODE_FORCE_ENABLE": "1", "AGENT_MANAGER_API_URL": identityServer.URL,
		"VROOLI_NOTIFICATION_RECIPIENT": "cf5-operator",
	}
	hubSource := prepareCF5APIScenarioVariant(t, "notification-hub", "../../notification-hub", runDir, hubEnv)
	hubStarted := lifecycleRehearsal(t, ctx, runDir, "notification-hub-start", map[string]string{
		"API_TOKEN": "", "VROOLI_TEST_MODE_FORCE_ENABLE": "1",
	}, "start", hubName, "--path", hubSource, "--json", "--timeout", "300")
	hubAPIURL := cf5APIURL(t, hubStarted)
	hubStopped := false
	stopHub := func() {
		if hubStopped {
			return
		}
		stopCF5Variant(t, runDir, hubName, hubAPIURL)
		hubStopped = true
	}
	t.Cleanup(stopHub)
	requireHTTPStatus(t, hubAPIURL+"/health", http.StatusOK)
	hubLeaseID := "cf5-" + hubInstance
	hubLeaseDB := filepath.Join(runDir, "notification-hub-lease.db")
	hubRouting := routingconnect.NewRoutingServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	_, err = hubRouting.InstallTestPool(ctx, connect.NewRequest(&routingpb.InstallTestPoolRequest{Dsn: hubLeaseDB, LeaseId: hubLeaseID, LeaseTtlMs: int64((8 * time.Minute).Milliseconds())}))
	require.NoError(t, err, "install isolated Notification Hub routed database lease")
	hubLeaseActive := true
	clearHubLease := func() {
		if !hubLeaseActive {
			return
		}
		_, clearErr := hubRouting.ClearTestPool(context.Background(), connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: hubLeaseID}))
		if clearErr != nil {
			t.Errorf("clear isolated Notification Hub lease: %v", clearErr)
		}
		hubLeaseActive = false
	}
	t.Cleanup(clearHubLease)

	instance := instanceName("treasury")
	name := "treasury@" + instance
	apiEnv := map[string]string{
		"TREASURY_REHEARSAL_DIR":                              filepath.Join(runDir, "custody"),
		"TREASURY_BASE_RPC_URL":                               forkIdentity.URL,
		"TREASURY_BASE_USDC_MODE":                             "local-fork",
		"MONEY_LEDGER_API_URL":                                ledgerURL,
		"SCENARIO_AUTHENTICATOR_API_URL":                      authenticatorURL,
		"AGENT_MANAGER_API_URL":                               identityServer.URL,
		"NOTIFICATION_HUB_API_URL":                            hubAPIURL,
		"TREASURY_CF5_NOTIFICATION_TEST_MODE":                 "1",
		"TREASURY_CF5_NOTIFICATION_TEST_AGENT_IDENTITY_TOKEN": "cf5-local-agent",
	}
	source := prepareCF5ScenarioVariant(t, "treasury", "..", runDir, apiEnv)
	writeCF5JSON(t, filepath.Join(runDir, "treasury-binding.json"), map[string]any{
		"instance": name, "source": source, "fork": forkIdentity, "api_environment": apiEnv,
	})
	started := lifecycleRehearsal(t, ctx, runDir, "treasury-start", map[string]string{
		"GOFLAGS": "-tags=treasury_rehearsal", "API_TOKEN": "",
		"TREASURY_OPERATOR_TOKEN": "", "TREASURY_MANDATE_SIGNING_KEY": "",
	}, "start", name, "--path", source, "--json", "--timeout", "300")
	apiURL, uiURL := cf5RuntimeURLs(t, started)
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopCF5Treasury(t, runDir, name, apiURL, uiURL)
		stopped = true
	}
	t.Cleanup(stop)
	requireHTTPStatus(t, apiURL+"/health", http.StatusOK)
	requireHTTPStatus(t, uiURL+"/health", http.StatusOK)
	requireHTTPStatus(t, uiURL+"/fund", http.StatusOK)
	const budgetID = "cf5-mobile-budget"
	const bookID = "cf5-mobile-book"
	ledgerBookID := createdBook.Msg.GetBook().GetId()
	ledgerAccountID := createdAccount.Msg.GetAccount().GetId()
	operator := authorizationconnect.NewTreasuryAdminClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	agent := authorizationconnect.NewAgentSpendClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	readHeadroom := func() *authorizationv1.GetBudgetHeadroomResponse {
		req := connect.NewRequest(&authorizationv1.GetBudgetHeadroomRequest{BudgetId: budgetID})
		req.Header().Set(identity.HeaderAgentIdentityToken, "cf5-local-agent")
		response, readErr := agent.GetBudgetHeadroom(ctx, req)
		require.NoError(t, readErr)
		return response.Msg
	}
	_, err = operator.CreateBook(ctx, cf5OperatorRequest(&authorizationv1.CreateBookRequest{Book: &bookv1.Book{Id: bookID, Name: "CF5 isolated local fork"}}))
	require.NoError(t, err)
	_, err = operator.SetBudgetCaps(ctx, cf5OperatorRequest(&authorizationv1.SetBudgetCapsRequest{Budget: &budgetv1.Budget{
		Id: budgetID, BookId: bookID, TotalCapMinor: 1000, PeriodicCapMinor: 1000,
		PerTransactionCapMinor: 1000, PeriodSeconds: 3600, Currency: "USD",
	}}))
	require.NoError(t, err)
	beforeCredit := readHeadroom()
	writeRehearsalProto(t, filepath.Join(runDir, "headroom-before-credit.json"), beforeCredit)
	fundingAdmin := operator
	provisioned, err := fundingAdmin.ProvisionBudgetWallet(ctx, cf5OperatorRequest(&fundingpb.ProvisionBudgetWalletRequest{BudgetId: budgetID, Network: baseusdc.Network}))
	require.NoError(t, err)
	require.NotEmpty(t, provisioned.Msg.GetWallet().GetAddress())
	writeCF5JSON(t, filepath.Join(runDir, "budget-wallet.json"), map[string]any{
		"book_id": bookID, "budget_id": budgetID, "wallet_id": provisioned.Msg.GetWallet().GetWalletId(),
		"local_fork_address": provisioned.Msg.GetWallet().GetAddress(), "ledger_book_id": ledgerBookID, "ledger_account_id": ledgerAccountID,
	})
	captureCF5MobileFund(t, ctx, runDir, uiURL, budgetID, ledgerBookID, ledgerAccountID, provisioned.Msg.GetWallet().GetAddress())
	intents, err := fundingAdmin.ListFundingIntents(ctx, cf5OperatorRequest(&fundingpb.ListFundingIntentsRequest{BudgetId: budgetID}))
	require.NoError(t, err)
	require.Len(t, intents.Msg.GetIntents(), 1)
	intent := intents.Msg.GetIntents()[0]
	require.Equal(t, "1000000", intent.GetAtomicAmount())
	expectedPaymentURI := fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=1000000", baseusdc.USDCAddress, provisioned.Msg.GetWallet().GetAddress())
	require.Equal(t, expectedPaymentURI, intent.GetPaymentUri(), "the Base app deep link must prefill the canonical USDC contract, Base chain, budget wallet, and exact atomic amount")
	writeCF5JSON(t, filepath.Join(runDir, "mobile-funding-intent.json"), map[string]any{
		"id": intent.GetId(), "budget_id": intent.GetBudgetId(), "atomic_amount": intent.GetAtomicAmount(),
		"payment_uri": intent.GetPaymentUri(), "state": intent.GetState(),
	})

	_, err = fork.Mint(ctx, fork.Accounts[0], intent.GetAtomicAmount())
	require.NoError(t, err)
	txHash, err := fork.Transfer(ctx, fork.Accounts[0], provisioned.Msg.GetWallet().GetAddress(), intent.GetAtomicAmount())
	require.NoError(t, err)
	// Anvil's inherited Base finalized tag trails the tip by 64 blocks. Mine a
	// local-only confirmation window so the watcher proves finalized receipt
	// handling rather than weakening production finality to "latest".
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x40"}, nil))
	var latest string
	var finalized, txReceipt map[string]any
	require.NoError(t, fork.Call(ctx, "eth_blockNumber", []any{}, &latest))
	require.NoError(t, fork.Call(ctx, "eth_getBlockByNumber", []any{"finalized", false}, &finalized))
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{txHash}, &txReceipt))
	writeCF5JSON(t, filepath.Join(runDir, "funding-chain-check.json"), map[string]any{
		"transaction_hash": txHash, "finality_advanced_blocks": 64,
		"latest_head": latest, "finalized_head": finalized, "transaction_receipt": txReceipt,
	})
	writeCF5JSON(t, filepath.Join(runDir, "mobile-funding-transfer.json"), map[string]any{
		"transaction_hash": txHash, "network": "base-local-fork", "atomic_amount": intent.GetAtomicAmount(),
		"sender": fork.Accounts[0], "recipient_wallet_id": provisioned.Msg.GetWallet().GetWalletId(),
	})
	var observed *fundingpb.FundingIntent
	var lastRead map[string]any
	var lastReadError string
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		read, readErr := fundingAdmin.GetFundingIntent(ctx, cf5OperatorRequest(&fundingpb.FundingIntentRequest{Id: intent.GetId()}))
		if readErr != nil {
			lastReadError = readErr.Error()
		} else {
			lastRead = map[string]any{"state": read.Msg.GetIntent().GetState(), "transaction_hash": read.Msg.GetIntent().GetTransactionHash(), "last_error": read.Msg.GetIntent().GetLastError()}
		}
		if readErr == nil && read.Msg.GetIntent().GetTransactionHash() != "" && read.Msg.GetIntent().GetState() == "credited" {
			observed = read.Msg.GetIntent()
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	writeCF5JSON(t, filepath.Join(runDir, "watcher-probe.json"), map[string]any{"last_read": lastRead, "last_read_error": lastReadError})
	require.NotNil(t, observed, "Treasury's background watcher must observe the finalized local-fork receipt")
	require.Equal(t, txHash, observed.GetTransactionHash())
	require.Equal(t, "credited", observed.GetState(), "the watcher must accept the finalized receipt through the isolated Money Ledger before crediting the budget")
	require.Empty(t, observed.GetLastError())
	require.Equal(t, "sent", observed.GetNotificationState(), "the verified test producer must deliver the receipt notice to its leased Hub database")
	writeCF5JSON(t, filepath.Join(runDir, "watcher-observation.json"), map[string]any{
		"state": observed.GetState(), "transaction_hash": observed.GetTransactionHash(),
		"last_error": observed.GetLastError(), "notification_state": observed.GetNotificationState(),
		"receipt_credited": true, "receipt_delivery_id": observed.GetReceiptDeliveryId(),
	})
	deliveryResponse, err := fundingAdmin.GetReceiptFunding(ctx, cf5OperatorRequest(&fundingpb.GetReceiptFundingRequest{Id: observed.GetReceiptDeliveryId()}))
	require.NoError(t, err)
	require.Equal(t, fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED, deliveryResponse.Msg.GetDelivery().GetState())
	require.Equal(t, "1000000", deliveryResponse.Msg.GetDelivery().GetReceipt().GetAtomicDelta())
	require.EqualValues(t, 100, deliveryResponse.Msg.GetDelivery().GetReceipt().GetValuationDeltaMinor())
	writeRehearsalProto(t, filepath.Join(runDir, "money-ledger-accepted-receipt.json"), deliveryResponse.Msg)
	notificationReq := connect.NewRequest(&notificationsv1.ListRequest{Limit: 20})
	notificationReq.Header().Set("X-Vrooli-Test-Mode", "1")
	notificationClient := notificationsconnect.NewNotificationsServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	notificationList, err := notificationClient.List(ctx, notificationReq)
	require.NoError(t, err)
	var matchingNotices []*notificationsshared.Notification
	for _, notice := range notificationList.Msg.GetNotifications() {
		if notice.GetDedupeKey() == deliveryResponse.Msg.GetDelivery().GetReceipt().GetId() {
			matchingNotices = append(matchingNotices, notice)
		}
	}
	require.Len(t, matchingNotices, 1, "the routed Hub fixture stores one idempotent receipt notice")
	require.Contains(t, matchingNotices[0].GetBody(), "credited to budget "+budgetID)
	require.Equal(t, "e1-fixture@notification-hub.invalid", matchingNotices[0].GetRequestedBy())
	writeRehearsalProto(t, filepath.Join(runDir, "notification-hub-readback.json"), matchingNotices[0])
	afterCredit := readHeadroom()
	require.EqualValues(t, beforeCredit.GetHeadroom().GetFundingCreditedMinor()+100, afterCredit.GetHeadroom().GetFundingCreditedMinor())
	require.EqualValues(t, beforeCredit.GetHeadroom().GetAvailableMinor()+100, afterCredit.GetHeadroom().GetAvailableMinor())
	writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-credit.json"), afterCredit)
	captureCF5MobileReceipt(t, ctx, runDir, uiURL, budgetID, txHash)

	withdrawalDestination := strings.ToLower(fork.Accounts[2])
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{provisioned.Msg.GetWallet().GetAddress(), "0xde0b6b3a7640000"}, nil))
	_, err = operator.SetWithdrawalDestination(ctx, cf5OperatorRequest(&fundingpb.SetWithdrawalDestinationRequest{Destination: &fundingpb.WithdrawalDestination{
		BudgetId: budgetID, Address: withdrawalDestination, Label: "CF5 local Coinbase test destination",
	}}))
	require.NoError(t, err)
	mandateResponse, err := operator.CreateMandate(ctx, cf5OperatorRequest(&authorizationv1.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: "cf5-withdrawal-mandate", IdempotencyKey: "cf5-withdrawal-mandate-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 1000, Currency: "USD", AllowedCounterparties: []string{withdrawalDestination}, AgentSubject: "cf2-agent", AgentRunId: "test-run",
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	require.Equal(t, mandatepb.MandateStatus_MANDATE_STATUS_LIVE, mandateResponse.Msg.GetMandate().GetStatus())
	captureCF5MobileWithdrawalPreview(t, ctx, runDir, uiURL, budgetID, withdrawalDestination, mandateResponse.Msg.GetMandate().GetId())
	quoteResponse, err := operator.PreviewWithdrawal(ctx, cf5OperatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.GetMandate().GetId(), Destination: withdrawalDestination,
		AtomicAmount: "1000000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 10,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:cf5-withdrawal-value", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	require.Equal(t, "local-fork", quoteResponse.Msg.GetQuote().GetExecutionMode())
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-preview.json"), quoteResponse.Msg.GetQuote())
	withdrawalRequest := func() *connect.Request[fundingpb.ExecuteWithdrawalRequest] {
		req := cf5OperatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.GetQuote().GetId(), IdempotencyKey: "cf5-withdrawal-idempotency"})
		req.Header().Set("X-Treasury-Mandate-Identity-Token", "cf5-local-agent")
		return req
	}
	pending, err := operator.ExecuteWithdrawal(ctx, withdrawalRequest())
	require.NoError(t, err)
	require.Equal(t, "pending_approval", pending.Msg.GetWithdrawal().GetState())
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-pending-approval.json"), pending.Msg.GetWithdrawal())
	queued, err := operator.ListApprovals(ctx, cf5OperatorRequest(&authorizationv1.ListApprovalsRequest{Status: approvalpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	require.NoError(t, err)
	require.Len(t, queued.Msg.GetApprovals(), 1)
	approved, err := operator.ResolveApproval(ctx, cf5OperatorRequest(&authorizationv1.ResolveApprovalRequest{ApprovalId: queued.Msg.GetApprovals()[0].GetId(), Resolution: approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	require.NoError(t, err)
	require.Equal(t, approvalpb.ApprovalStatus_APPROVAL_STATUS_APPROVED, approved.Msg.GetApproval().GetStatus())
	settled, err := operator.ExecuteWithdrawal(ctx, withdrawalRequest())
	require.NoError(t, err)
	require.Equal(t, "settled", settled.Msg.GetWithdrawal().GetState(), "%s", settled.Msg.GetWithdrawal().GetLastError())
	require.NotEmpty(t, settled.Msg.GetWithdrawal().GetTransactionHash())
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-settled.json"), settled.Msg.GetWithdrawal())
	var withdrawalReceipt struct {
		Status string `json:"status"`
	}
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{settled.Msg.GetWithdrawal().GetTransactionHash()}, &withdrawalReceipt))
	require.Equal(t, "0x1", withdrawalReceipt.Status)
	writeCF5JSON(t, filepath.Join(runDir, "withdrawal-chain-receipt.json"), map[string]string{
		"status": withdrawalReceipt.Status, "transaction_hash": settled.Msg.GetWithdrawal().GetTransactionHash(),
		"destination": withdrawalDestination, "atomic_amount": "1000000", "network": "base-local-fork",
	})
	clearHubLease()
	stopHub()

	// Stop in the owner body so the receipt contains explicit lifecycle and
	// socket evidence; registered test cleanup is an idempotent crash fallback.
	stop()
	stopLedger()
	stopReceipt := fork.Stop(ctx)
	writeCF5JSON(t, filepath.Join(runDir, "fork-stop.json"), stopReceipt)
	require.True(t, stopReceipt.ProcessExited)
	require.False(t, stopReceipt.KilledAfterTimeout)
	require.False(t, stopReceipt.EndpointReachableAfter, "%+v", stopReceipt)
	require.True(t, stopReceipt.Clean, "%+v", stopReceipt)
	writeCF5JSON(t, filepath.Join(runDir, "handoff-result.json"), map[string]any{
		"status": "partial", "completed": []string{"BAS mobile Fund interaction", "local Base payment intent and QR", "finalized USDC transfer", "isolated Money Ledger exact-token acceptance", "exactly-once Treasury budget headroom credit", "scoped test producer notification and Hub readback", "allowlisted fee preview and local-fork withdrawal"},
		"remaining": []string{"one continuous BAS interaction that previews and executes withdrawal", "independent CF5 code review"},
		"instance":  name, "api_url": apiURL, "ui_url": uiURL,
		"fork": forkIdentity, "fork_stop": stopReceipt,
	})
}

func captureCF5MobileFund(t *testing.T, ctx context.Context, runDir, uiURL, budgetID, ledgerBookID, ledgerAccountID, walletAddress string) {
	t.Helper()
	flow := map[string]any{
		"settings": map[string]any{"viewport_width": 390, "viewport_height": 844},
		"nodes": []any{
			map[string]any{"id": "wait-fund", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='page-fund']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Mobile Fund page renders"}}},
			map[string]any{"id": "assert-fork-boundary", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "This page does not publish a production funding address or authorize customer deposits."}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the local-fork-only funding boundary"}}},
			cf5InputNode("operator-token", "funding-operator-token", "cf2-local-operator-only"),
			cf5InputNode("budget-id", "funding-budget-id", budgetID),
			cf5InputNode("atomic-amount", "funding-atomic-amount", "1000000"),
			cf5InputNode("declared-value", "funding-declared-value", "1.00"),
			cf5InputNode("ledger-book-id", "funding-ledger-book-id", ledgerBookID),
			cf5InputNode("ledger-account-id", "funding-ledger-account-id", ledgerAccountID),
			map[string]any{"id": "create-intent", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-create-intent']"}, "metadata": map[string]any{"label": "Tap Fund to create a local-fork payment intent"}}},
			map[string]any{"id": "wait-payment-link", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-payment-uri']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Wait for the Base payment link"}}},
			map[string]any{"id": "assert-qr", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-payment-qr']", "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the funding QR is shown"}}},
			map[string]any{"id": "assert-prefilled-base-link", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-open-wallet-link']", "mode": "ASSERTION_MODE_ATTRIBUTE_EQUALS", "attribute_name": "href", "expected": map[string]any{"string_value": fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=1000000", baseusdc.USDCAddress, walletAddress)}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the Base app link pre-fills USDC, chain, budget wallet and exact amount"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "assert-fork-boundary"), cf5Edge("e2", "assert-fork-boundary", "operator-token"),
			cf5Edge("e3", "operator-token", "budget-id"), cf5Edge("e4", "budget-id", "atomic-amount"),
			cf5Edge("e5", "atomic-amount", "declared-value"), cf5Edge("e6", "declared-value", "ledger-book-id"),
			cf5Edge("e7", "ledger-book-id", "ledger-account-id"), cf5Edge("e8", "ledger-account-id", "create-intent"),
			cf5Edge("e9", "create-intent", "wait-payment-link"), cf5Edge("e10", "wait-payment-link", "assert-qr"), cf5Edge("e11", "assert-qr", "assert-prefilled-base-link"),
		},
	}
	flowPath := filepath.Join(runDir, "mobile-fund-capture-flow.json")
	raw, err := json.MarshalIndent(flow, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(flowPath, raw, 0o600))
	artifactDir := filepath.Join(runDir, "bas-mobile-fund")
	require.NoError(t, os.MkdirAll(artifactDir, 0o700))
	cli, err := exec.LookPath("browser-automation-studio")
	require.NoError(t, err, "BAS CLI is required for the CF5 mobile recording")
	cmd := exec.CommandContext(ctx, cli, "capture", "--url", uiURL+"/fund", "--dimensions", "mobile", "--capture", "screenshot,dom", "--wait-for", "[data-testid='page-fund']", "--interaction-flow-file", flowPath, "--out", artifactDir, "--label", "CF5 local-fork mobile Fund", "--json")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "BAS mobile Fund capture failed: %s", strings.TrimSpace(string(output)))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "bas-mobile-fund-capture.json"), output, 0o600))
	require.NotEmpty(t, output)
}

func captureCF5MobileReceipt(t *testing.T, ctx context.Context, runDir, uiURL, budgetID, txHash string) {
	t.Helper()
	flow := map[string]any{
		"settings": map[string]any{"viewport_width": 390, "viewport_height": 844},
		"nodes": []any{
			map[string]any{"id": "wait-fund", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='page-fund']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Open the mobile Funding surface"}}},
			cf5InputNode("operator-token", "funding-operator-token", "cf2-local-operator-only"),
			cf5InputNode("budget-id", "funding-budget-id", budgetID),
			map[string]any{"id": "refresh-intents", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-refresh-intents']"}, "metadata": map[string]any{"label": "Load the newly credited funding receipt"}}},
			map[string]any{"id": "wait-intent", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-recent-intent']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 10000}, "metadata": map[string]any{"label": "Wait for receipt status"}}},
			map[string]any{"id": "select-intent", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-recent-intent']"}, "metadata": map[string]any{"label": "Open the funded intent"}}},
			map[string]any{"id": "assert-credited", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "credited"}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify receipt credit is visible on mobile"}}},
			cf5InputNode("transaction-hash", "funding-transaction-hash", txHash),
			map[string]any{"id": "confirm-transfer", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-confirm-transfer']"}, "metadata": map[string]any{"label": "Confirm the finalized Base transfer from the mobile receipt view"}}},
			map[string]any{"id": "assert-credit-stable", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "credited"}, "timeout_ms": 10000}, "metadata": map[string]any{"label": "Verify the receipt remains credited after confirmation"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "operator-token"), cf5Edge("e2", "operator-token", "budget-id"),
			cf5Edge("e3", "budget-id", "refresh-intents"), cf5Edge("e4", "refresh-intents", "wait-intent"),
			cf5Edge("e5", "wait-intent", "select-intent"), cf5Edge("e6", "select-intent", "assert-credited"),
			cf5Edge("e7", "assert-credited", "transaction-hash"), cf5Edge("e8", "transaction-hash", "confirm-transfer"),
			cf5Edge("e9", "confirm-transfer", "assert-credit-stable"),
		},
	}
	flowPath := filepath.Join(runDir, "mobile-receipt-capture-flow.json")
	raw, err := json.MarshalIndent(flow, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(flowPath, raw, 0o600))
	artifactDir := filepath.Join(runDir, "bas-mobile-receipt")
	require.NoError(t, os.MkdirAll(artifactDir, 0o700))
	cli, err := exec.LookPath("browser-automation-studio")
	require.NoError(t, err, "BAS CLI is required for the CF5 mobile receipt recording")
	cmd := exec.CommandContext(ctx, cli, "capture", "--url", uiURL+"/fund", "--dimensions", "mobile", "--capture", "screenshot,dom", "--wait-for", "[data-testid='page-fund']", "--interaction-flow-file", flowPath, "--out", artifactDir, "--label", "CF5 local-fork mobile receipt credit", "--json")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "BAS mobile receipt capture failed: %s", strings.TrimSpace(string(output)))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "bas-mobile-receipt-capture.json"), output, 0o600))
	require.NotEmpty(t, output)
}

func captureCF5MobileWithdrawalPreview(t *testing.T, ctx context.Context, runDir, uiURL, budgetID, destination, mandateID string) {
	t.Helper()
	flow := map[string]any{
		"settings": map[string]any{"viewport_width": 390, "viewport_height": 844},
		"nodes": []any{
			map[string]any{"id": "wait-fund", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='page-fund']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Open the mobile funding and withdrawal surface"}}},
			cf5InputNode("operator-token", "funding-operator-token", "cf2-local-operator-only"),
			cf5InputNode("agent-token", "funding-agent-token", "cf5-local-agent"),
			cf5InputNode("budget-id", "funding-budget-id", budgetID),
			cf5InputNode("destination", "funding-withdrawal-destination", destination),
			cf5InputNode("mandate-id", "funding-withdrawal-mandate-id", mandateID),
			cf5InputNode("withdrawal-amount", "funding-withdrawal-amount", "1000000"),
			cf5InputNode("withdrawal-value", "funding-withdrawal-value", "1.00"),
			cf5InputNode("max-fee", "funding-withdrawal-max-fee", "1000000000000000000"),
			cf5InputNode("fee-value", "funding-withdrawal-fee-value", "0.10"),
			map[string]any{"id": "preview-withdrawal", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-withdrawal-preview']"}, "metadata": map[string]any{"label": "Request an allowlisted local-fork withdrawal fee preview"}}},
			map[string]any{"id": "assert-quote", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "local-fork"}, "timeout_ms": 10000}, "metadata": map[string]any{"label": "Verify the quote discloses its local-fork execution mode"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "operator-token"), cf5Edge("e2", "operator-token", "agent-token"),
			cf5Edge("e3", "agent-token", "budget-id"), cf5Edge("e4", "budget-id", "destination"),
			cf5Edge("e5", "destination", "mandate-id"), cf5Edge("e6", "mandate-id", "withdrawal-amount"),
			cf5Edge("e7", "withdrawal-amount", "withdrawal-value"), cf5Edge("e8", "withdrawal-value", "max-fee"),
			cf5Edge("e9", "max-fee", "fee-value"), cf5Edge("e10", "fee-value", "preview-withdrawal"),
			cf5Edge("e11", "preview-withdrawal", "assert-quote"),
		},
	}
	flowPath := filepath.Join(runDir, "mobile-withdrawal-capture-flow.json")
	raw, err := json.MarshalIndent(flow, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(flowPath, raw, 0o600))
	artifactDir := filepath.Join(runDir, "bas-mobile-withdrawal")
	require.NoError(t, os.MkdirAll(artifactDir, 0o700))
	cli, err := exec.LookPath("browser-automation-studio")
	require.NoError(t, err, "BAS CLI is required for the CF5 mobile withdrawal preview recording")
	cmd := exec.CommandContext(ctx, cli, "capture", "--url", uiURL+"/fund", "--dimensions", "mobile", "--capture", "screenshot,dom", "--wait-for", "[data-testid='page-fund']", "--interaction-flow-file", flowPath, "--out", artifactDir, "--label", "CF5 local-fork mobile withdrawal preview", "--json")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "BAS mobile withdrawal preview capture failed: %s", strings.TrimSpace(string(output)))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "bas-mobile-withdrawal-capture.json"), output, 0o600))
	require.NotEmpty(t, output)
}

func cf5InputNode(id, selector, value string) map[string]any {
	return map[string]any{"id": id, "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='" + selector + "']", "value": value, "clear_first": true}, "metadata": map[string]any{"label": "Enter isolated fixture " + id}}}
}

func cf5Edge(id, source, target string) map[string]any {
	return map[string]any{"id": id, "source": source, "target": target}
}

func cf5OperatorRequest[T any](value *T) *connect.Request[T] {
	request := connect.NewRequest(value)
	request.Header().Set("X-Vrooli-Operator-Token", "cf2-local-operator-only")
	return request
}

func instanceName(prefix string) string {
	return "cf5-" + prefix + "-" + time.Now().UTC().Format("150405") + fmt.Sprintf("-%d", os.Getpid())
}

func cf5APIURL(t *testing.T, raw []byte) string {
	t.Helper()
	var result struct {
		Success  bool `json:"success"`
		Scenario struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenario"`
		Scenarios []struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenarios"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Success)
	ports := result.Scenario.Ports
	if len(result.Scenarios) == 1 {
		ports = result.Scenarios[0].Ports
	}
	api := ports["API_PORT"]
	require.NotZero(t, api)
	return fmt.Sprintf("http://127.0.0.1:%d", api)
}

func prepareCF5ScenarioVariant(t *testing.T, scenarioName, repoScenario, runDir string, apiEnv map[string]string) string {
	t.Helper()
	repo, err := filepath.Abs(filepath.Clean(filepath.Join(".", repoScenario)))
	require.NoError(t, err)
	require.DirExists(t, repo)
	workspace := filepath.Join(runDir, "workspace")
	source := filepath.Join(workspace, "scenarios", scenarioName)
	require.NoError(t, os.MkdirAll(source, 0o700))
	repoRoot := filepath.Clean(filepath.Join(repo, "../.."))
	entries, err := os.ReadDir(repoRoot)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() == "scenarios" {
			continue
		}
		link := filepath.Join(workspace, entry.Name())
		target := filepath.Join(repoRoot, entry.Name())
		if _, statErr := os.Lstat(link); errors.Is(statErr, os.ErrNotExist) {
			require.NoError(t, os.Symlink(target, link))
		} else {
			require.NoError(t, statErr)
			actual, readErr := os.Readlink(link)
			require.NoError(t, readErr)
			require.Equal(t, target, actual, "shared test workspace link must retain its repository target")
		}
	}
	for _, component := range []string{"api", "cli", "ui", ".vrooli"} {
		original := filepath.Join(repo, component)
		if _, statErr := os.Stat(original); statErr != nil {
			continue
		}
		destRoot := filepath.Join(source, component)
		require.NoError(t, filepath.WalkDir(original, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(original, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return os.MkdirAll(destRoot, 0o700)
			}
			if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "data" || entry.Name() == "tmp") {
				if component == "ui" && entry.Name() == "node_modules" {
					if err := os.Symlink(path, filepath.Join(destRoot, rel)); err != nil {
						return err
					}
					return filepath.SkipDir
				}
				return filepath.SkipDir
			}
			dest := filepath.Join(destRoot, rel)
			if entry.Type()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				return os.Symlink(target, dest)
			}
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
	config["credentials"] = map[string]any{"descriptors": []any{}}
	config["dependencies"] = map[string]any{"scenarios": map[string]any{}, "resources": map[string]any{}}
	config["cli"].(map[string]any)["enabled"] = false
	config["components"].(map[string]any)["api"].(map[string]any)["run"].(map[string]any)["env"] = apiEnv
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
	return source
}

// prepareCF5APIScenarioVariant keeps the test-only service surface to the API
// used by this rehearsal. The standard scenario copy intentionally shares the
// source UI node_modules tree for the Treasury browser surface; doing that for
// these API-only dependencies makes lifecycle correctly reject an isolated
// rebuild while their live UI variants are serving shared build outputs.
func prepareCF5APIScenarioVariant(t *testing.T, scenarioName, repoScenario, runDir string, apiEnv map[string]string) string {
	t.Helper()
	source := prepareCF5ScenarioVariant(t, scenarioName, repoScenario, runDir, apiEnv)
	manifest := filepath.Join(source, ".vrooli", "service.json")
	var config map[string]any
	readRehearsalJSON(t, manifest, &config)
	components, ok := config["components"].(map[string]any)
	require.True(t, ok, "API-only test variant needs a component map")
	require.Contains(t, components, "api")
	delete(components, "ui")
	lifecycle, ok := config["lifecycle"].(map[string]any)
	require.True(t, ok, "API-only test variant needs a lifecycle map")
	health, ok := lifecycle["health"].(map[string]any)
	require.True(t, ok, "API-only test variant needs health configuration")
	checks, ok := health["checks"].([]any)
	require.True(t, ok, "API-only test variant needs health checks")
	apiChecks := make([]any, 0, len(checks))
	for _, check := range checks {
		entry, ok := check.(map[string]any)
		require.True(t, ok, "health check must be an object")
		target, _ := entry["target"].(string)
		if !strings.Contains(target, "${UI_PORT}") {
			apiChecks = append(apiChecks, entry)
		}
	}
	require.NotEmpty(t, apiChecks, "API-only test variant must retain an API health check")
	health["checks"] = apiChecks
	if endpoints, ok := health["endpoints"].(map[string]any); ok {
		delete(endpoints, "ui")
	}
	// Notification Hub's lifecycle setup checks duplicate Ask identities in the
	// serving database before replacing that service. This rehearsal starts a
	// new variant with its own lifecycle-assigned data directory, so the live
	// replacement preflight is neither needed nor applicable here; the variant
	// boot initializes only its own empty database.
	if setup := lifecycle["setup"]; setup != nil {
		delete(lifecycle, "setup")
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
	return source
}

func cf5RuntimeURLs(t *testing.T, raw []byte) (string, string) {
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
	api, ui := result.Scenario.Ports["API_PORT"], result.Scenario.Ports["UI_PORT"]
	require.NotZero(t, api)
	require.NotZero(t, ui, "CF5 rehearsal requires the real Treasury UI component")
	return fmt.Sprintf("http://127.0.0.1:%d", api), fmt.Sprintf("http://127.0.0.1:%d", ui)
}

func stopCF5Treasury(t *testing.T, dir, name, apiURL, uiURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	output := lifecycleRehearsal(t, ctx, dir, "treasury-stop", map[string]string{}, "stop", name, "--json")
	var status struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(output, &status))
	require.True(t, status.Success)
	statusOutput := lifecycleRehearsal(t, ctx, dir, "treasury-status-after-stop", map[string]string{}, "status", name, "--json")
	var stopped struct {
		Success  bool `json:"success"`
		Scenario struct {
			Name      string `json:"name"`
			Status    string `json:"status"`
			Processes int    `json:"processes"`
			StartOp   struct {
				Variant string `json:"variant"`
			} `json:"start_operation"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(statusOutput, &stopped))
	require.True(t, stopped.Success)
	require.Equal(t, "treasury", stopped.Scenario.Name)
	require.Equal(t, strings.TrimPrefix(name, "treasury@"), stopped.Scenario.StartOp.Variant)
	require.Equal(t, "stopped", stopped.Scenario.Status)
	require.Zero(t, stopped.Scenario.Processes)
	probes := map[string]bool{
		"api": cf5HTTPReachable(apiURL + "/health"),
		"ui":  cf5HTTPReachable(uiURL + "/health"),
	}
	writeCF5JSON(t, filepath.Join(dir, "treasury-listener-probes.json"), probes)
	require.False(t, probes["api"], "Treasury API listener remained after lifecycle stop")
	require.False(t, probes["ui"], "Treasury UI listener remained after lifecycle stop")
}

func stopCF5Variant(t *testing.T, dir, name, apiURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	stop := lifecycleRehearsal(t, ctx, dir, "variant-stop", map[string]string{}, "stop", name, "--json")
	var stopped struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(stop, &stopped))
	require.True(t, stopped.Success)
	statusRaw := lifecycleRehearsal(t, ctx, dir, "variant-status-after-stop", map[string]string{}, "status", name, "--json")
	var status struct {
		Success  bool `json:"success"`
		Scenario struct {
			Status    string `json:"status"`
			Processes int    `json:"processes"`
			StartOp   struct {
				Variant string `json:"variant"`
			} `json:"start_operation"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(statusRaw, &status))
	require.True(t, status.Success)
	parts := strings.SplitN(name, "@", 2)
	require.Len(t, parts, 2)
	require.Equal(t, parts[1], status.Scenario.StartOp.Variant)
	require.Equal(t, "stopped", status.Scenario.Status)
	require.Zero(t, status.Scenario.Processes)
	require.False(t, cf5HTTPReachable(apiURL+"/health"), "variant API listener remained after lifecycle stop")
}

func requireHTTPStatus(t *testing.T, url string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(url)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, want, response.StatusCode, url)
}

func cf5HTTPReachable(url string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return true
}

func writeCF5JSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}
