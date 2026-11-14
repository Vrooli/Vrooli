//go:build treasury_rehearsal

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
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
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	requestidentity "github.com/vrooli/api-core/identity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	chaincases "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	chaincasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	notificationsshared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	mandatepb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/identity"
	"treasury/internal/testutil/agentclaims"
	"treasury/internal/testutil/basefork"
)

const cf5PinnedBaseForkPort = 18545

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

	basefork.AcquireSharedFacilitatorHarness(t)
	fork := basefork.StartAt(t, cf5PinnedBaseForkPort)
	forkIdentity := fork.Identity()
	require.Equal(t, "0x2105", forkIdentity.ChainID)
	require.True(t, strings.HasPrefix(forkIdentity.URL, "http://127.0.0.1:"))
	require.NotEmpty(t, forkIdentity.BlockNumber)
	require.NotEmpty(t, forkIdentity.BlockHash)
	require.NotZero(t, forkIdentity.AnvilPID)
	writeCF5JSON(t, filepath.Join(runDir, "fork-identity.json"), forkIdentity)
	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	cf5ProvisionPinnedFacilitatorKey(t, facilitatorKey)
	cf5StartPinnedFacilitator(t, ctx, runDir)
	writeCF5JSON(t, filepath.Join(runDir, "facilitator-local-fork.json"), map[string]any{
		"resource": "x402-facilitator-local-test", "url": "http://127.0.0.1:14021",
		"fork_rpc": forkIdentity.URL, "facilitator_signer": facilitatorAddress.Hex(), "test_only": true,
	})

	custodyDir := filepath.Join(runDir, "custody")
	initializeCF2Credentials(t, filepath.Join(custodyDir, "synthetic-custody.enc"))
	t.Cleanup(func() { _ = os.RemoveAll(custodyDir) })
	offerDeskEnv := map[string]string{
		"VROOLI_STORAGE_ROOT": filepath.Join(runDir, "offer-desk-storage"),
		"TREASURY_API_URL":    "", "OFFER_DESK_TEST_DROP_FIRST_PAID_RESPONSE": "1",
	}
	offerDeskSource := prepareCF5APIScenarioVariant(t, "offer-desk", "../../offer-desk", runDir, offerDeskEnv)
	disableCF4OfferDeskVectorSearch(t, offerDeskSource)
	offerDeskName := "offer-desk@" + instanceName("offer-desk")
	offerDeskStart := lifecycleRehearsal(t, ctx, runDir, "offer-desk-start", map[string]string{}, "start", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	offerDeskURL := rehearsalURL(t, offerDeskStart)
	offerDeskStopped := false
	stopOfferDesk := func() {
		if offerDeskStopped {
			return
		}
		stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
		offerDeskStopped = true
	}
	t.Cleanup(stopOfferDesk)
	offerID, offerVariantID := seedCF4OfferDeskGraph(t, ctx, offerDeskURL, runDir)
	paidCatalogURL := offerDeskURL + "/api/v1/paid/catalog"
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
	var runIdentityMu sync.RWMutex
	activeRunID := "test-run"
	claimsForRun := func(runID string) identity.Claims {
		claims := agentclaims.For("cf1-local-operator")
		claims.RunID = runID
		claims.Scopes = append(claims.Scopes, "notification-hub:write")
		grant := requestidentity.OperationGrant{
			ID: "cf5-isolated-producer-grant", Approver: "cf5-test-operator",
			Operations: []string{"treasury.read", "treasury.spend", "notification.send"},
			Targets:    []string{"treasury:*", "notification-hub:recipient/e1-fixture@notification-hub.invalid"},
			ExpiresAt:  time.Now().Add(time.Hour),
		}
		grantJSON, err := json.Marshal(grant)
		if err != nil {
			return identity.Claims{}
		}
		claims.Meta["operation_grant"] = string(grantJSON)
		return claims
	}
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Token          string `json:"token"`
			OperationToken string `json:"operation_token"`
		}
		if r.URL.Path != "/api/v1/identity/verify" || json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "invalid isolated CF5 agent", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if request.Token == "cf5-local-agent" && request.OperationToken == "" {
			runIdentityMu.RLock()
			runID := activeRunID
			runIdentityMu.RUnlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": claimsForRun(runID)})
			return
		}
		baseline, baselineOK := verifyCF1RunToken(request.Token)
		if !baselineOK {
			http.Error(w, "invalid isolated CF1 run identity", http.StatusUnauthorized)
			return
		}
		if request.OperationToken == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": baseline})
			return
		}
		operation, operationOK := verifyCF1RunToken(request.OperationToken)
		if !operationOK || baseline.RunID != operation.RunID || baseline.TaskID != operation.TaskID || baseline.Subject != operation.Subject || operation.Meta["authorization_id"] == "" || operation.Meta["operation_grant"] == "" {
			http.Error(w, "invalid isolated CF1 operation identity", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": baseline, "operation": map[string]any{"valid": true, "claims": operation}})
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
		"VROOLI_STORAGE_ROOT":                                 filepath.Join(runDir, "storage"),
		"TREASURY_REHEARSAL_DIR":                              filepath.Join(runDir, "custody"),
		"TREASURY_BASE_RPC_URL":                               forkIdentity.URL,
		"TREASURY_BASE_USDC_MODE":                             "local-fork",
		"MONEY_LEDGER_API_URL":                                ledgerURL,
		"TREASURY_LEDGER_BOOK_ID":                             createdBook.Msg.GetBook().GetId(),
		"TREASURY_LEDGER_ACCOUNT_ID":                          createdAccount.Msg.GetAccount().GetId(),
		"SCENARIO_AUTHENTICATOR_API_URL":                      authenticatorURL,
		"AGENT_MANAGER_API_URL":                               identityServer.URL,
		"NOTIFICATION_HUB_API_URL":                            hubAPIURL,
		"TREASURY_CF5_NOTIFICATION_TEST_MODE":                 "1",
		"TREASURY_CF5_NOTIFICATION_TEST_AGENT_IDENTITY_TOKEN": "cf5-local-agent",
		"X402_FACILITATOR_URL":                                "http://127.0.0.1:14021",
		"OFFER_DESK_BASE_URL":                                 offerDeskURL,
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
	setCF5VariantEnvironment(t, offerDeskSource, map[string]string{"TREASURY_API_URL": apiURL})
	boundOfferDesk := lifecycleRehearsal(t, ctx, runDir, "offer-desk-bind-treasury", map[string]string{}, "restart", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	require.Equal(t, offerDeskURL, rehearsalURL(t, boundOfferDesk), "Offer Desk keeps its test API address when binding the isolated Treasury variant")
	offerPrice := map[string]any{
		"id": cf4OfferDeskTestPriceID, "resource_url": paidCatalogURL,
		"description": "Offer Desk research catalog lookup", "mime_type": "application/json",
		"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
		"asset": baseusdc.USDCAddress, "asset_decimals": 6, "max_timeout_seconds": 300,
		"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
	}
	declaredOfferPrice := cf4DeclareOfferDeskPrice(t, ctx, apiURL, map[string]any{"offer_id": offerID, "variant_id": offerVariantID, "price": offerPrice})
	require.Equal(t, int64(1), declaredOfferPrice.AmountMinor)
	require.Equal(t, "10000", declaredOfferPrice.Amount)
	writeCF5JSON(t, filepath.Join(runDir, "offer-desk-price.json"), declaredOfferPrice)
	requireHTTPStatus(t, uiURL+"/health", http.StatusOK)
	requireHTTPStatus(t, uiURL+"/fund", http.StatusOK)
	const bookID = "cf5-integrated-research-book"
	ledgerBookID := createdBook.Msg.GetBook().GetId()
	ledgerAccountID := createdAccount.Msg.GetAccount().GetId()
	operator := authorizationconnect.NewTreasuryAdminClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	agent := authorizationconnect.NewAgentSpendClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	readHeadroom := func(budgetID string) *authorizationv1.GetBudgetHeadroomResponse {
		req := connect.NewRequest(&authorizationv1.GetBudgetHeadroomRequest{BudgetId: budgetID})
		req.Header().Set(identity.HeaderAgentIdentityToken, "cf5-local-agent")
		response, readErr := agent.GetBudgetHeadroom(ctx, req)
		require.NoError(t, readErr)
		return response.Msg
	}
	_, err = operator.CreateBook(ctx, cf5OperatorRequest(&authorizationv1.CreateBookRequest{Book: &bookv1.Book{Id: bookID, Name: "CF5 isolated local fork"}}))
	require.NoError(t, err)
	fundingAdmin := operator
	variant := fmt.Sprintf("cf5-integrated-%d", time.Now().UnixNano())
	budgetID := "cf1-agent-manager-budget-" + variant
	mandateID := "cf1-agent-manager-mandate-" + variant
	var runID, walletID, walletAddress, fundingTxHash string
	var beforeCredit, afterCredit *authorizationv1.GetBudgetHeadroomResponse
	notificationClient := notificationsconnect.NewNotificationsServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	hooks := cf1ShowcaseHooks{
		FundingAtomicAmount: "2600000",
		FundAgentWallet: func(fundCtx context.Context, actualBudgetID, actualWalletID, actualAddress, actualMandateID, actualRunID string) error {
			if actualBudgetID != budgetID || actualMandateID != mandateID {
				return fmt.Errorf("unexpected run-owned budget binding: budget=%s mandate=%s", actualBudgetID, actualMandateID)
			}
			runIdentityMu.Lock()
			activeRunID = actualRunID
			runIdentityMu.Unlock()
			runID, walletID, walletAddress = actualRunID, actualWalletID, actualAddress
			beforeCredit = readHeadroom(budgetID)
			writeRehearsalProto(t, filepath.Join(runDir, "headroom-before-credit.json"), beforeCredit)
			cf5RecoverBudgetRootBeforeFunding(t, fundCtx, runDir, operator, bookID, custodyDir)
			writeCF5JSON(t, filepath.Join(runDir, "budget-wallet.json"), map[string]any{
				"book_id": bookID, "budget_id": budgetID, "mandate_id": mandateID, "run_id": runID,
				"wallet_id": walletID, "local_fork_address": walletAddress,
				"ledger_book_id": ledgerBookID, "ledger_account_id": ledgerAccountID,
			})
			captureCF5MobileFund(t, fundCtx, runDir, uiURL, budgetID, ledgerBookID, ledgerAccountID, walletAddress, "2600000", "2.60")
			intents, listErr := fundingAdmin.ListFundingIntents(fundCtx, cf5OperatorRequest(&fundingpb.ListFundingIntentsRequest{BudgetId: budgetID}))
			if listErr != nil {
				return listErr
			}
			if len(intents.Msg.GetIntents()) != 1 {
				return fmt.Errorf("Fund surface created %d intents for run budget", len(intents.Msg.GetIntents()))
			}
			intent := intents.Msg.GetIntents()[0]
			if intent.GetAtomicAmount() != "2600000" {
				return fmt.Errorf("unexpected exact token amount in funding intent: %s", intent.GetAtomicAmount())
			}
			expectedURI := fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=2600000", baseusdc.USDCAddress, walletAddress)
			if intent.GetPaymentUri() != expectedURI {
				return fmt.Errorf("Base app link does not bind the run wallet and exact atoms: %s", intent.GetPaymentUri())
			}
			writeCF5JSON(t, filepath.Join(runDir, "mobile-funding-intent.json"), map[string]any{
				"id": intent.GetId(), "budget_id": budgetID, "wallet_id": walletID, "agent_run_id": runID,
				"atomic_amount": intent.GetAtomicAmount(), "payment_uri": intent.GetPaymentUri(), "state": intent.GetState(),
			})
			if _, mintErr := fork.Mint(fundCtx, fork.Accounts[0], intent.GetAtomicAmount()); mintErr != nil {
				return mintErr
			}
			fundingTxHash, err = fork.Transfer(fundCtx, fork.Accounts[0], walletAddress, intent.GetAtomicAmount())
			if err != nil {
				return err
			}
			// Anvil's inherited Base finalized tag trails the tip by 64 blocks. Mine
			// a local-only confirmation window rather than weakening production finality.
			if err = fork.Call(fundCtx, "anvil_mine", []any{"0x40"}, nil); err != nil {
				return err
			}
			var latest string
			var finalized, txReceipt map[string]any
			if err = fork.Call(fundCtx, "eth_blockNumber", []any{}, &latest); err != nil {
				return err
			}
			if err = fork.Call(fundCtx, "eth_getBlockByNumber", []any{"finalized", false}, &finalized); err != nil {
				return err
			}
			if err = fork.Call(fundCtx, "eth_getTransactionReceipt", []any{fundingTxHash}, &txReceipt); err != nil {
				return err
			}
			writeCF5JSON(t, filepath.Join(runDir, "funding-chain-check.json"), map[string]any{
				"transaction_hash": fundingTxHash, "finality_advanced_blocks": 64,
				"latest_head": latest, "finalized_head": finalized, "transaction_receipt": txReceipt,
			})
			writeCF5JSON(t, filepath.Join(runDir, "mobile-funding-transfer.json"), map[string]any{
				"transaction_hash": fundingTxHash, "network": "base-local-fork", "atomic_amount": intent.GetAtomicAmount(),
				"sender": fork.Accounts[0], "recipient_wallet_id": walletID, "agent_run_id": runID,
			})
			var observed *fundingpb.FundingIntent
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				read, readErr := fundingAdmin.GetFundingIntent(fundCtx, cf5OperatorRequest(&fundingpb.FundingIntentRequest{Id: intent.GetId()}))
				if readErr == nil && read.Msg.GetIntent().GetTransactionHash() != "" && read.Msg.GetIntent().GetState() == "credited" {
					observed = read.Msg.GetIntent()
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			if observed == nil || observed.GetTransactionHash() != fundingTxHash || observed.GetNotificationState() != "sent" {
				return fmt.Errorf("run wallet funding watcher did not credit and notify: %+v", observed)
			}
			delivery, deliveryErr := fundingAdmin.GetReceiptFunding(fundCtx, cf5OperatorRequest(&fundingpb.GetReceiptFundingRequest{Id: observed.GetReceiptDeliveryId()}))
			if deliveryErr != nil {
				return deliveryErr
			}
			if delivery.Msg.GetDelivery().GetState() != fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED || delivery.Msg.GetDelivery().GetReceipt().GetAtomicDelta() != "2600000" || delivery.Msg.GetDelivery().GetReceipt().GetValuationDeltaMinor() != 260 {
				return fmt.Errorf("Money Ledger did not accept the exact run-wallet receipt and declared value: %+v", delivery.Msg.GetDelivery())
			}
			writeRehearsalProto(t, filepath.Join(runDir, "money-ledger-accepted-receipt.json"), delivery.Msg)
			notificationReq := connect.NewRequest(&notificationsv1.ListRequest{Limit: 20})
			notificationReq.Header().Set("X-Vrooli-Test-Mode", "1")
			notificationList, noticeErr := notificationClient.List(fundCtx, notificationReq)
			if noticeErr != nil {
				return noticeErr
			}
			var matched []*notificationsshared.Notification
			for _, notice := range notificationList.Msg.GetNotifications() {
				if notice.GetDedupeKey() == delivery.Msg.GetDelivery().GetReceipt().GetId() {
					matched = append(matched, notice)
				}
			}
			if len(matched) != 1 || !strings.Contains(matched[0].GetBody(), "credited to budget "+budgetID) {
				return fmt.Errorf("Notification Hub did not retain one accepted-receipt notice: %v", matched)
			}
			writeRehearsalProto(t, filepath.Join(runDir, "notification-hub-readback.json"), matched[0])
			afterCredit = readHeadroom(budgetID)
			if afterCredit.GetHeadroom().GetFundingCreditedMinor() != 260 || afterCredit.GetHeadroom().GetFundingRemainingMinor() != 260 || afterCredit.GetHeadroom().GetFundingAtomicAmount() != "2600000" {
				return fmt.Errorf("accepted receipt did not become exact Treasury headroom: %+v", afterCredit.GetHeadroom())
			}
			writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-credit.json"), afterCredit)
			captureCF5MobileReceipt(t, fundCtx, runDir, uiURL, budgetID, fundingTxHash)
			writeCF5JSON(t, filepath.Join(runDir, "agent-manager-cli", "funding-accepted.json"), map[string]any{
				"budget_id": budgetID, "wallet_id": walletID, "address": walletAddress,
				"transaction_hash": fundingTxHash, "atomic_amount": "2600000", "accepted": true,
			})
			return nil
		},
		AwaitApprovalNotice: func(waitCtx context.Context, approvalID string) (*notificationsv1.SendRequest, error) {
			deadline := time.Now().Add(20 * time.Second)
			var lastBodies []string
			for time.Now().Before(deadline) {
				req := connect.NewRequest(&notificationsv1.ListRequest{Limit: 100})
				req.Header().Set("X-Vrooli-Test-Mode", "1")
				list, listErr := notificationClient.List(waitCtx, req)
				if listErr != nil {
					return nil, listErr
				}
				lastBodies = lastBodies[:0]
				for _, notice := range list.Msg.GetNotifications() {
					lastBodies = append(lastBodies, notice.GetBody())
					if strings.Contains(notice.GetBody(), "110 USD minor units") && strings.Contains(notice.GetBody(), approvalID) {
						return &notificationsv1.SendRequest{Body: notice.GetBody()}, nil
					}
				}
				time.Sleep(250 * time.Millisecond)
			}
			return nil, fmt.Errorf("Notification Hub did not deliver approval %s before timeout; latest bodies: %s", approvalID, strings.Join(lastBodies, " | "))
		},
		X402: &cf1X402ShowcaseConfig{
			CatalogURL: paidCatalogURL, PriceID: cf4OfferDeskTestPriceID,
			CallID: cf4OfferDeskTestVariantName, Counterparty: "127.0.0.1", InstrumentID: "cf1-agent-manager-x402",
			PrepareSigner: func(_ context.Context, walletID, walletAddress string) (string, error) {
				return cf1RunWalletX402Credential(t, runDir, filepath.Join(custodyDir, "synthetic-custody.enc"), walletID, walletAddress, paidCatalogURL)
			},
		},
	}
	repo, err := filepath.Abs(filepath.Join(".", "../../.."))
	require.NoError(t, err)
	cf1RunID, cf1AuthorizationID, cf1SettlementID, err := runCF1AgentManagerCLIForkShowcase(t, ctx, repo, runDir, apiURL, identityServer.URL, source, name, operator, nil, fork, bookID, variant, hooks)
	require.NoError(t, err)
	require.NotEmpty(t, cf1RunID)
	require.NotEmpty(t, cf1AuthorizationID)
	require.NotEmpty(t, cf1SettlementID)
	require.Equal(t, runID, cf1RunID)
	purchases := []int64{30, 110}
	if hooks.X402 != nil {
		purchases = append(purchases, 1)
	}
	writeCF5JSON(t, filepath.Join(runDir, "integrated-wallet-spend.json"), map[string]any{
		"run_id": runID, "budget_id": budgetID, "mandate_id": mandateID, "wallet_id": walletID,
		"wallet_address": walletAddress, "in_cap_authorization_id": cf1AuthorizationID,
		"in_cap_settlement_id": cf1SettlementID, "accepted_funding_atoms": "2600000",
		"bounded_local_fork_purchases_minor": purchases,
	})
	reconcileCF6TreasuryForkWallets(t, ctx, runDir, forkIdentity.URL, runID)
	afterSpend := readHeadroom(budgetID)
	require.EqualValues(t, 260, afterSpend.GetHeadroom().GetFundingCreditedMinor())
	expectedReturnMinor := int64(120)
	if hooks.X402 != nil {
		expectedReturnMinor--
	}
	require.EqualValues(t, expectedReturnMinor, afterSpend.GetHeadroom().GetFundingRemainingMinor())
	writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-agent-purchases.json"), afterSpend)

	withdrawalDestination := strings.ToLower(fork.Accounts[2])
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{walletAddress, "0xde0b6b3a7640000"}, nil))
	_, err = operator.SetBudgetCaps(ctx, cf5OperatorRequest(&authorizationv1.SetBudgetCapsRequest{Budget: &budgetv1.Budget{
		Id: budgetID, BookId: bookID, TotalCapMinor: 1000, PeriodicCapMinor: 1000,
		PerTransactionCapMinor: 1000, PeriodSeconds: 3600, Currency: "USD",
	}}))
	require.NoError(t, err, "the local rehearsal explicitly permits its small return transfer after CF1's over-cap approval proof")
	_, err = operator.SetWithdrawalDestination(ctx, cf5OperatorRequest(&fundingpb.SetWithdrawalDestinationRequest{Destination: &fundingpb.WithdrawalDestination{
		BudgetId: budgetID, Address: withdrawalDestination, Label: "CF5 local Coinbase test destination",
	}}))
	require.NoError(t, err)
	withdrawalMandateID := "cf5-withdrawal-mandate-" + variant
	mandateResponse, err := operator.CreateMandate(ctx, cf5OperatorRequest(&authorizationv1.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: withdrawalMandateID, IdempotencyKey: withdrawalMandateID + "-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 200, Currency: "USD", AllowedCounterparties: []string{withdrawalDestination}, AgentSubject: "cf1-local-operator", AgentRunId: runID,
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	require.Equal(t, mandatepb.MandateStatus_MANDATE_STATUS_LIVE, mandateResponse.Msg.GetMandate().GetStatus())
	destinationBefore := cf5USDCBalance(t, ctx, fork, withdrawalDestination)
	withdrawalAmount := "1.00"
	withdrawalAtomic := "1000000"
	if hooks.X402 != nil {
		withdrawalAmount = "1.09"
		withdrawalAtomic = "1090000"
	}
	captureCF5MobileWithdrawalPreview(t, ctx, runDir, uiURL, budgetID, withdrawalDestination, mandateResponse.Msg.GetMandate().GetId(), withdrawalAmount)
	destinationAfter := cf5USDCBalance(t, ctx, fork, withdrawalDestination)
	withdrawalAtomicBig, ok := new(big.Int).SetString(withdrawalAtomic, 10)
	require.True(t, ok)
	require.Equal(t, new(big.Int).Add(destinationBefore, withdrawalAtomicBig).String(), destinationAfter.String(), "the mobile confirmation must execute exactly the allowlisted return transfer")
	var withdrawalReceipt struct {
		Status string `json:"status"`
	}
	// The BAS assertion captures the settled withdrawal status; the destination
	// balance delta is the independent local-fork chain readback.
	withdrawalTxHash := cf5WithdrawalTx(t, ctx, fork, withdrawalDestination, withdrawalAtomic)
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{withdrawalTxHash}, &withdrawalReceipt))
	require.Equal(t, "0x1", withdrawalReceipt.Status)
	writeCF5JSON(t, filepath.Join(runDir, "withdrawal-chain-receipt.json"), map[string]string{
		"status": withdrawalReceipt.Status, "transaction_hash": withdrawalTxHash,
		"destination": withdrawalDestination, "atomic_amount": withdrawalAtomic, "network": "base-local-fork",
	})
	afterWithdrawal := readHeadroom(budgetID)
	require.EqualValues(t, 260, afterWithdrawal.GetHeadroom().GetFundingCreditedMinor())
	require.EqualValues(t, 0, afterWithdrawal.GetHeadroom().GetFundingRemainingMinor())
	writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-allowlisted-return.json"), afterWithdrawal)
	clearHubLease()
	stopHub()

	// Stop in the owner body so the receipt contains explicit lifecycle and
	// socket evidence; registered test cleanup is an idempotent crash fallback.
	stop()
	stopLedger()
	stopOfferDesk()
	facilitatorStop := cf5SourceVrooliCommand(ctx, "resource", "stop", "x402-facilitator-local-test", "--json")
	facilitatorStopOutput, facilitatorStopErr := facilitatorStop.CombinedOutput()
	require.NoErrorf(t, facilitatorStopErr, "stop the isolated facilitator before its fork: %s", strings.TrimSpace(string(facilitatorStopOutput)))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "facilitator-stop.json"), facilitatorStopOutput, 0o600))
	stopReceipt := fork.Stop(ctx)
	writeCF5JSON(t, filepath.Join(runDir, "fork-stop.json"), stopReceipt)
	require.True(t, stopReceipt.ProcessExited)
	require.False(t, stopReceipt.KilledAfterTimeout)
	require.False(t, stopReceipt.EndpointReachableAfter, "%+v", stopReceipt)
	require.True(t, stopReceipt.Clean, "%+v", stopReceipt)
	writeCF5JSON(t, filepath.Join(runDir, "handoff-result.json"), map[string]any{
		"status": "partial", "completed": []string{"the CF1 budget's actual Treasury research root was exported to a key-only encrypted portable bundle, its CA scalar removed, failed to sign, restored from the separately copied bundle, and proved with a fresh challenge before receipt funding", "ordinary Agent Manager run wallet funded by a finalized local Base USDC transfer", "CF1 in-cap and approved over-cap local-fork purchases under the run-bound mandate", "isolated Money Ledger exact-token receipt acceptance", "exact Treasury budget headroom consumption after purchases and return", "scoped Notification Hub receipt delivery and readback", "BAS mobile Fund surface and exact prefilled Base link", "BAS mobile allowlisted withdrawal preview and execution with chain balance readback"},
		"remaining": []string{"qualify DBM off-host transport and restore on a second independent fleet owner", "replace the in-cap native USDC fixture purchase with the fully configured CF4 facilitator priced call", "independent CF1/CF5 code review"},
		"instance":  name, "api_url": apiURL, "ui_url": uiURL,
		"fork": forkIdentity, "fork_stop": stopReceipt,
	})
}

// cf5RecoverBudgetRootBeforeFunding exports the exact receive-only root that
// parents the run's mandate wallet, removes its CA scalar, and restores it
// from the encrypted portable bundle before any receipt funding is accepted.
// Independent-CA node recovery is covered by the wallet package's restart
// proof; this joined rehearsal proves the live mandate/accounting path waits
// until the root is recoverable again.
func cf5RecoverBudgetRootBeforeFunding(t *testing.T, ctx context.Context, runDir string, admin authorizationconnect.TreasuryAdminClient, bookID, custodyDir string) {
	t.Helper()
	bookResponse, err := admin.GetBook(ctx, cf5OperatorRequest(&authorizationv1.GetBookRequest{BookId: bookID}))
	require.NoError(t, err)
	operatorID := bookResponse.Msg.GetBook().GetBeneficiaryIdentity()
	require.NotEmpty(t, operatorID)
	rootDigest := sha256.Sum256([]byte(strings.Join([]string{operatorID, bookID, baseusdc.Network}, "\x00")))
	rootID := fmt.Sprintf("budget-root-%x", rootDigest[:16])
	rootResponse, err := admin.GetResearchWallet(ctx, cf5OperatorRequest(&authorizationv1.ResearchWalletRequest{Id: rootID, Network: baseusdc.Network}))
	require.NoError(t, err, "read the actual receive-only root wallet that parents the mandate wallet")
	root := rootResponse.Msg.GetWallet()
	require.NotEmpty(t, root.GetCustodyReference())
	secret := "operator-held local rehearsal recovery secret with adequate entropy"
	exported, err := admin.ExportPortableResearchWalletRecovery(ctx, cf5OperatorRequest(&authorizationv1.ExportPortableResearchWalletRecoveryRequest{
		Id: rootID, Network: baseusdc.Network, OperatorSecret: secret,
	}))
	require.NoError(t, err)
	bundleMatches, err := filepath.Glob(filepath.Join(runDir, "storage", "data", "vrooli", "*", filepath.FromSlash(exported.Msg.GetRelativePath())))
	require.NoError(t, err)
	require.Len(t, bundleMatches, 1, "portable bundle must stay inside this isolated Treasury variant's data root")
	serverBundlePath := bundleMatches[0]
	bundle, err := os.ReadFile(serverBundlePath)
	require.NoError(t, err)
	require.Equal(t, int(exported.Msg.GetSizeBytes()), len(bundle))
	require.NotEmpty(t, exported.Msg.GetSha256())
	portableDir := filepath.Join(runDir, "portable-recovery")
	require.NoError(t, os.MkdirAll(portableDir, 0o700))
	bundlePath := filepath.Join(portableDir, rootID+".bundle")
	require.NoError(t, os.WriteFile(bundlePath, bundle, 0o600))
	identity, err := credentialauthority.ParseIdentity(root.GetCustodyReference())
	require.NoError(t, err)
	localAuthority, err := credentialauthority.OpenEncryptedFile(filepath.Join(custodyDir, "synthetic-custody.enc"), rehearsalAuthorityPassphrase)
	require.NoError(t, err)
	require.NoError(t, localAuthority.Delete(identity, "secp256k1-scalar"), "simulate loss of source-host root key before funding")
	_, err = admin.ProveResearchWallet(ctx, cf5OperatorRequest(&authorizationv1.ProveResearchWalletRequest{Id: rootID, Network: baseusdc.Network, Nonce: "cf5-source-key-removed-before-funding"}))
	require.Error(t, err, "source root must stop signing after the scalar is removed")
	restored, err := admin.RestorePortableResearchWalletRecovery(ctx, cf5OperatorRequest(&authorizationv1.RestorePortableResearchWalletRecoveryRequest{
		Id: rootID, Network: baseusdc.Network, OperatorSecret: secret, Bundle: bundle,
	}))
	require.NoError(t, err)
	require.Equal(t, root.GetAddress(), restored.Msg.GetWallet().GetAddress())
	proof, err := admin.ProveResearchWallet(ctx, cf5OperatorRequest(&authorizationv1.ProveResearchWalletRequest{Id: rootID, Network: baseusdc.Network, Nonce: "cf5-recovered-root-before-funding"}))
	require.NoError(t, err)
	require.True(t, proof.Msg.GetProof().GetVerified())
	require.Equal(t, root.GetAddress(), proof.Msg.GetProof().GetWallet().GetAddress())
	writeCF5JSON(t, filepath.Join(runDir, "portable-recovery", "recovery-before-funding.json"), map[string]any{
		"root_wallet_id": rootID, "address": root.GetAddress(), "bundle_path": bundlePath,
		"bundle_sha256": exported.Msg.GetSha256(), "bundle_size_bytes": exported.Msg.GetSizeBytes(),
		"source_scalar_removed": true, "source_proof_after_removal": "rejected",
		"restored_root_address_matches": true, "fresh_challenge_proof_verified": proof.Msg.GetProof().GetVerified(),
		"funding_started_after_restore": true, "bundle_is_key_only": true,
	})
}

func cf5USDCBalance(t *testing.T, ctx context.Context, fork *basefork.Fork, address string) *big.Int {
	t.Helper()
	data := "0x70a08231" + strings.TrimPrefix(cf5AddressTopic(address), "0x")
	var raw string
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": baseusdc.USDCAddress, "data": data}, "latest"}, &raw))
	value, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16)
	require.True(t, ok, "USDC balanceOf response must be hexadecimal")
	return value
}

func cf5WithdrawalTx(t *testing.T, ctx context.Context, fork *basefork.Fork, destination, amount string) string {
	t.Helper()
	var logs []struct {
		TransactionHash string   `json:"transactionHash"`
		Topics          []string `json:"topics"`
		Data            string   `json:"data"`
	}
	filter := map[string]any{
		"address":   baseusdc.USDCAddress,
		"fromBlock": fork.Identity().BlockNumber, "toBlock": "latest",
		"topics": []any{cf1ERC20TransferTopic, nil, cf5AddressTopic(destination)},
	}
	require.NoError(t, fork.Call(ctx, "eth_getLogs", []any{filter}, &logs))
	amountValue, ok := new(big.Int).SetString(amount, 10)
	require.True(t, ok)
	wanted := fmt.Sprintf("0x%064x", amountValue)
	var hashes []string
	for _, log := range logs {
		if strings.EqualFold(log.Data, wanted) && log.TransactionHash != "" {
			hashes = append(hashes, log.TransactionHash)
		}
	}
	require.Len(t, hashes, 1, "one exact USDC transfer must settle to the allowlisted destination")
	return hashes[0]
}

func cf5AddressTopic(address string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}

func formatAtomicUSDCForUI(atoms string) string {
	value, ok := new(big.Int).SetString(atoms, 10)
	if !ok {
		return atoms
	}
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(value, big.NewInt(1_000_000), fraction)
	return fmt.Sprintf("%s.%06d", whole.String(), fraction.Int64())
}

func captureCF5MobileFund(t *testing.T, ctx context.Context, runDir, uiURL, budgetID, ledgerBookID, ledgerAccountID, walletAddress, atomicAmount, declaredValue string) {
	t.Helper()
	flow := map[string]any{
		"settings": map[string]any{"viewport_width": 390, "viewport_height": 844},
		"nodes": []any{
			map[string]any{"id": "wait-fund", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='page-fund']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Mobile Fund page renders"}}},
			map[string]any{"id": "assert-fork-boundary", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "This page does not publish a production funding address or authorize customer deposits."}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the local-fork-only funding boundary"}}},
			cf5InputNode("operator-token", "funding-operator-token", "cf2-local-operator-only"),
			cf5InputNode("budget-id", "funding-budget-id", budgetID),
			cf5InputNode("token-amount", "funding-atomic-amount", formatAtomicUSDCForUI(atomicAmount)),
			cf5InputNode("declared-value", "funding-declared-value", declaredValue),
			cf5InputNode("ledger-book-id", "funding-ledger-book-id", ledgerBookID),
			cf5InputNode("ledger-account-id", "funding-ledger-account-id", ledgerAccountID),
			map[string]any{"id": "create-intent", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-create-intent']"}, "metadata": map[string]any{"label": "Tap Fund to create a local-fork payment intent"}}},
			map[string]any{"id": "wait-payment-link", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-payment-uri']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Wait for the Base payment link"}}},
			map[string]any{"id": "assert-qr", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-payment-qr']", "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the funding QR is shown"}}},
			map[string]any{"id": "assert-prefilled-base-link", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-open-wallet-link']", "mode": "ASSERTION_MODE_ATTRIBUTE_EQUALS", "attribute_name": "href", "expected": map[string]any{"string_value": fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=%s", baseusdc.USDCAddress, walletAddress, atomicAmount)}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the Base app link pre-fills USDC, chain, budget wallet and exact amount"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "assert-fork-boundary"), cf5Edge("e2", "assert-fork-boundary", "operator-token"),
			cf5Edge("e3", "operator-token", "budget-id"), cf5Edge("e4", "budget-id", "token-amount"),
			cf5Edge("e5", "token-amount", "declared-value"), cf5Edge("e6", "declared-value", "ledger-book-id"),
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

func captureCF5MobileWithdrawalPreview(t *testing.T, ctx context.Context, runDir, uiURL, budgetID, destination, mandateID, amount string) {
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
			cf5InputNode("withdrawal-amount", "funding-withdrawal-amount", amount),
			cf5InputNode("withdrawal-value", "funding-withdrawal-value", amount),
			cf5InputNode("max-fee", "funding-withdrawal-max-fee", "1000000000000000000"),
			cf5InputNode("fee-value", "funding-withdrawal-fee-value", "0.10"),
			map[string]any{"id": "preview-withdrawal", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-withdrawal-preview']"}, "metadata": map[string]any{"label": "Request an allowlisted local-fork withdrawal fee preview"}}},
			map[string]any{"id": "assert-quote", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "local-fork"}, "timeout_ms": 10000}, "metadata": map[string]any{"label": "Verify the quote discloses its local-fork execution mode"}}},
			map[string]any{"id": "execute-withdrawal", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-withdrawal-execute']"}, "metadata": map[string]any{"label": "Confirm and execute the allowlisted withdrawal from mobile"}}},
			map[string]any{"id": "wait-settled", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-withdrawal-status']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 20000}, "metadata": map[string]any{"label": "Wait for the withdrawal settlement status"}}},
			map[string]any{"id": "assert-settled", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-withdrawal-status']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "settled"}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Confirm the allowlisted withdrawal settled"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "operator-token"), cf5Edge("e2", "operator-token", "agent-token"),
			cf5Edge("e3", "agent-token", "budget-id"), cf5Edge("e4", "budget-id", "destination"),
			cf5Edge("e5", "destination", "mandate-id"), cf5Edge("e6", "mandate-id", "withdrawal-amount"),
			cf5Edge("e7", "withdrawal-amount", "withdrawal-value"), cf5Edge("e8", "withdrawal-value", "max-fee"),
			cf5Edge("e9", "max-fee", "fee-value"), cf5Edge("e10", "fee-value", "preview-withdrawal"),
			cf5Edge("e11", "preview-withdrawal", "assert-quote"), cf5Edge("e12", "assert-quote", "execute-withdrawal"),
			cf5Edge("e13", "execute-withdrawal", "wait-settled"), cf5Edge("e14", "wait-settled", "assert-settled"),
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

func cf5ProvisionPinnedFacilitatorKey(t *testing.T, key *ecdsa.PrivateKey) {
	t.Helper()
	secret := []byte(fmt.Sprintf("%x", gethcrypto.FromECDSA(key)))
	defer clear(secret)
	command := exec.Command("vrooli", "credentials", "provision", "--identity", "vrooli/x402-facilitator-local-test", "--field", "facilitator-private-key")
	command.Stdin = strings.NewReader(string(secret))
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "provision ephemeral local-fork facilitator test signer: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		delete := exec.Command("vrooli", "credentials", "delete", "--identity", "vrooli/x402-facilitator-local-test", "--field", "facilitator-private-key", "--yes")
		if output, err := delete.CombinedOutput(); err != nil {
			t.Errorf("remove ephemeral local-fork facilitator signer: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

func cf5StartPinnedFacilitator(t *testing.T, ctx context.Context, runDir string) {
	t.Helper()
	command := cf5SourceVrooliCommand(ctx, "resource", "start", "x402-facilitator-local-test", "--json")
	output, err := command.CombinedOutput()
	_ = os.WriteFile(filepath.Join(runDir, "facilitator-start.json"), output, 0o600)
	require.NoErrorf(t, err, "start lifecycle-managed local-fork facilitator: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		stop := cf5SourceVrooliCommand(stopCtx, "resource", "stop", "x402-facilitator-local-test", "--json")
		output, err := stop.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "facilitator-stop.json"), output, 0o600)
		if err != nil {
			t.Errorf("stop lifecycle-managed local-fork facilitator: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

func cf5SourceVrooliCommand(ctx context.Context, args ...string) *exec.Cmd {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	repoRoot := filepath.Clean(filepath.Join(workingDir, "../../.."))
	commandArgs := append([]string{"run", "./cmd/vrooli"}, args...)
	command := exec.CommandContext(ctx, "go", commandArgs...)
	command.Dir = repoRoot
	return command
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
			if strings.HasSuffix(entry.Name(), "-api") || strings.Contains(entry.Name(), "-api.freshness.json") || strings.HasPrefix(entry.Name(), ".vrooli-closure") {
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

// reconcileCF6TreasuryForkWallets feeds Treasury's address-only wallet set
// into the served chain-forensics snapshot route while this rehearsal's local
// Base fork is still live. The separate API variant owns its isolated case DB.
func reconcileCF6TreasuryForkWallets(t *testing.T, ctx context.Context, runDir, forkURL, runID string) {
	t.Helper()
	databases, err := filepath.Glob(filepath.Join(runDir, "storage", "data", "vrooli", "treasury_*", "treasury.db"))
	require.NoError(t, err)
	require.Len(t, databases, 1, "reconciliation needs one isolated Treasury database")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databases[0])+"?mode=ro")
	require.NoError(t, err)
	rows, err := db.QueryContext(ctx, `SELECT id,address FROM research_wallets WHERE network=? AND address<>'' ORDER BY id`, baseusdc.Network)
	require.NoError(t, err)
	type treasuryWallet struct {
		ID      string `json:"wallet_id"`
		Address string `json:"address"`
	}
	wallets := make([]treasuryWallet, 0, 8)
	for rows.Next() {
		var wallet treasuryWallet
		require.NoError(t, rows.Scan(&wallet.ID, &wallet.Address))
		wallets = append(wallets, wallet)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.NoError(t, db.Close())
	require.NotEmpty(t, wallets, "Treasury should have Base research wallet addresses to reconcile")

	caseRoot := filepath.Join(runDir, "chain-forensics-storage")
	source := prepareCF5APIScenarioVariant(t, "chain-forensics", "../../chain-forensics", runDir, map[string]string{
		"VROOLI_STORAGE_ROOT":          caseRoot,
		"CHAIN_FORENSICS_BASE_RPC_URL": forkURL,
	})
	name := "chain-forensics@" + instanceName("chain-forensics")
	started := lifecycleRehearsal(t, ctx, runDir, "chain-forensics-start", map[string]string{}, "start", name, "--path", source, "--json", "--timeout", "300")
	var start struct {
		Success  bool `json:"success"`
		Scenario struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenario"`
		Scenarios []struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenarios"`
	}
	require.NoError(t, json.Unmarshal(started, &start))
	require.True(t, start.Success)
	ports := start.Scenario.Ports
	if len(start.Scenarios) == 1 {
		ports = start.Scenarios[0].Ports
	}
	apiPort := ports["API_PORT"]
	require.NotZero(t, apiPort)
	apiURL := fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		output := lifecycleRehearsal(t, stopCtx, runDir, "chain-forensics-stop", map[string]string{}, "stop", name, "--json")
		writeCF5JSON(t, filepath.Join(runDir, "chain-forensics-stop.json"), json.RawMessage(output))
		stopped = true
	}
	t.Cleanup(stop)
	requireHTTPStatus(t, apiURL+"/health", http.StatusOK)
	client := chaincasesconnect.NewCasesServiceClient(&http.Client{Timeout: 20 * time.Second}, apiURL)
	caseID := "treasury-fork-wallets-" + strings.ReplaceAll(runID, "-", "")
	created, err := client.ImportCase(ctx, connect.NewRequest(&chaincases.ImportCaseRequest{Case: &chaincases.EvidenceCase{
		Id: caseID, Title: "Private Treasury local Base fork wallet reconciliation", Private: true,
		Gaps: []string{"This records chain balances only; it does not infer control, ownership or liabilities."},
	}}))
	require.NoError(t, err)
	require.Equal(t, caseID, created.Msg.GetCase().GetId())

	for _, wallet := range wallets {
		captured, captureErr := client.CaptureEVMWalletSnapshot(ctx, connect.NewRequest(&chaincases.CaptureEVMWalletSnapshotRequest{
			CaseId: caseID, Chain: chaincases.Chain_CHAIN_BASE, Address: wallet.Address,
			TokenContracts: []string{baseusdc.USDCAddress},
		}))
		require.NoError(t, captureErr, "capture Treasury wallet %s from the fork RPC", wallet.ID)
		snapshots := captured.Msg.GetCase().GetWalletSnapshots()
		require.NotEmpty(t, snapshots)
		snapshot := snapshots[len(snapshots)-1]
		require.Equal(t, strings.ToLower(wallet.Address), strings.ToLower(snapshot.GetAddress()))
		require.Equal(t, chaincases.Chain_CHAIN_BASE, snapshot.GetChain())
		require.NotEmpty(t, snapshot.GetBlockHash())
		require.NotEmpty(t, snapshot.GetSourceIds())
	}
	final, err := client.GetCase(ctx, connect.NewRequest(&chaincases.GetCaseRequest{Id: caseID}))
	require.NoError(t, err)
	snapshots := final.Msg.GetCase().GetWalletSnapshots()
	require.Len(t, snapshots, len(wallets))
	block, blockHash := snapshots[0].GetBlockHeight(), snapshots[0].GetBlockHash()
	for _, snapshot := range snapshots {
		require.Equal(t, block, snapshot.GetBlockHeight(), "all wallet balances must share one pinned fork block")
		require.Equal(t, blockHash, snapshot.GetBlockHash(), "all wallet balances must share one pinned fork block hash")
	}
	rawCase, err := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(final.Msg.GetCase())
	require.NoError(t, err)
	evidenceDir := filepath.Join(runDir, "chain-forensics")
	require.NoError(t, os.MkdirAll(evidenceDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(evidenceDir, "treasury-wallets-case.json"), rawCase, 0o600))
	writeCF5JSON(t, filepath.Join(evidenceDir, "treasury-wallet-reconciliation.json"), map[string]any{
		"run_id": runID, "chain": "base-local-fork", "fork_rpc": forkURL,
		"case_id": caseID, "wallet_set_source": "read-only Treasury research_wallets rows (id/address only)",
		"wallets": wallets, "snapshot_count": len(snapshots), "block_height": block,
		"block_hash": blockHash, "chain_facts_only": true,
	})
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
