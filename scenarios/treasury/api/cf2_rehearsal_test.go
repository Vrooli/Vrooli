//go:build treasury_rehearsal

package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"treasury/internal/testutil/agentclaims"

	"connectrpc.com/connect"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	notificationshared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	accountsV1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsConnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	authpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	bookconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book/book_v1connect"
	budgetpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	budgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget/budget_v1connect"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
	instrumentpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	instrumentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument/instrument_v1connect"
	mandatepb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	mandateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate/mandate_v1connect"
	settlementpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
	treasurysharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/shared"
	walletpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	walletconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet/wallet_v1connect"
	domainevents "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/identity"
	"treasury/internal/nativeusdc"
	"treasury/internal/operatorauth"
	x402rail "treasury/internal/rail/x402"
	"treasury/internal/testutil/basefork"
	"treasury/internal/wallet"
	"treasury/internal/walletops"
)

// TestCF2LocalForkThroughBothOwners uses lifecycle-managed scenario processes,
// real native-USDC execution, signed service identity and persistent databases.
// The opt-in fixture directory belongs to the isolated Money Ledger rehearsal;
// no host credential or real-value wallet is touched.
func TestCF2LocalForkThroughBothOwners(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-002] [REQ:TRS-P0-006] [REQ:TRS-P0-009] [REQ:TRS-P0-011] [REQ:TRS-P1-006] [REQ:TRS-P1-007]
	runTreasuryLocalForkRehearsal(t, false)
}

// TestCF1OrdinaryAgentManagerLocalFork isolates the full CF1 ordinary-run
// showcase from CF2's unrelated Offer Desk/LPBS service-order prelude. It
// retains real Treasury authorization, local-fork settlement, Notification
// Hub approval, run-owned key rotation/revocation, and restart receipt proof.
func TestCF1OrdinaryAgentManagerLocalFork(t *testing.T) {
	runTreasuryLocalForkRehearsal(t, true)
}

func runTreasuryLocalForkRehearsal(t *testing.T, cf1Only bool) {
	enableEnv := "TREASURY_CF2_REHEARSAL"
	if cf1Only {
		enableEnv = "TREASURY_CF1_REHEARSAL"
	}
	if os.Getenv(enableEnv) != "1" {
		t.Skip("set " + enableEnv + "=1, TREASURY_BASE_FORK_TEST=1, and TREASURY_CF2_FIXTURE_DIR for the local fork rehearsal")
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
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	runPrefix := "cf2-"
	if cf1Only {
		runPrefix = "cf1-"
	}
	runDir := filepath.Join(filepath.Dir(fixtureDir), runPrefix+time.Now().UTC().Format("20060102T150405Z"))
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	t.Logf("rehearsal evidence=%s", runDir)
	t.Cleanup(func() {
		status := "PASS"
		if t.Failed() {
			status = "FAIL"
		}
		scope := "Full CF2 two-owner local-fork rehearsal."
		if cf1Only {
			scope = "CF1 ordinary Agent Manager/Treasury local-fork showcase only. LPBS service-order and Offer Desk/x402 portions were intentionally omitted and remain unverified by this run."
		}
		readme := fmt.Sprintf("# Treasury local-fork rehearsal — %s\n\nOutcome: %s\n\n%s\n\nEvidence is limited to test-only identities, keys, and the disposable local Base fork. No mainnet or real-value effect occurred.\n", runPrefix, status, scope)
		if err := os.WriteFile(filepath.Join(runDir, "README.md"), []byte(readme), 0o600); err != nil {
			t.Errorf("write rehearsal README: %v", err)
		}
	})
	// Prior completed attempts measured ~1.1 GiB filesystem growth; reserve
	// 1.5 GiB so the showcase fits the host's 120 GiB safety floor with margin
	// over observed use.
	const cf2PeakBytes int64 = 1536 * 1024 * 1024
	diskOwner, diskJob := "crypto-foundations:CF1-normal-agent-run-local-fork", "CF1/CF2 local-fork Agent Manager and receipt rehearsal"
	if cf1Only {
		diskOwner, diskJob = "crypto-foundations:CF1-ordinary-agent-manager-local-fork", "CF1 ordinary Agent Manager/Treasury local-fork showcase"
	}
	if os.Getenv("TREASURY_CF7_MULTI_CALL_REHEARSAL") == "1" {
		diskOwner, diskJob = "crypto-foundations:CF7-research-budget-multi-call-local-fork", "CF7 three-call research budget run plus CF1 reference on local Base fork"
	}
	admitTreasuryRehearsalDisk(t, filepath.Dir(fixtureDir), runDir, diskOwner, diskJob, cf2PeakBytes)
	// The pinned local facilitator test resource routes its RPC to loopback
	// port 18545. Use the same disposable fork port so paid x402 calls settle
	// on this run's chain rather than a different local fork.
	basefork.AcquireSharedFacilitatorHarness(t)
	fork := basefork.StartAt(t, cf5PinnedBaseForkPort)
	x402BuyerKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	x402BuyerAddress := gethcrypto.PubkeyToAddress(x402BuyerKey.PublicKey)
	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	provisionCF2FacilitatorSigner(t, facilitatorKey)
	startCF2Facilitator(t, ctx, runDir)
	custodyDir := filepath.Join(runDir, "custody")
	credentialPath := filepath.Join(custodyDir, "synthetic-custody.enc")
	initializeCF2Credentials(t, credentialPath)
	t.Cleanup(func() { _ = os.RemoveAll(custodyDir) })
	token, noticeToken, _, _, receiptReaderToken, _, jwksURL, exchangeURL, closeIdentity := startServiceIdentityFixture(t)
	defer closeIdentity()
	restoreLedgerEnv := map[string]string{"GOFLAGS": ""}
	for key, value := range seed.Runtime {
		if strings.HasPrefix(key, "VROOLI_AUTH_") || key == "SCENARIO_DATA_DIR" || key == "SCENARIO_NAME" || key == "VROOLI_SCENARIO" || key == "VROOLI_STORAGE_NAMESPACE" {
			restoreLedgerEnv[key] = value
		}
	}
	ledgerEnv := make(map[string]string, len(restoreLedgerEnv)+1)
	for key, value := range restoreLedgerEnv {
		ledgerEnv[key] = value
	}
	ledgerEnv["VROOLI_AUTH_SCENARIO_JWKS_URL"] = jwksURL
	configureMoneyLedgerRehearsalAuth(t, seed.Source, jwksURL)
	lifecycleRehearsal(t, ctx, runDir, "ledger-fixture-restart", ledgerEnv, "restart", "money-ledger@"+seed.Variant, "--path", seed.Source, "--json", "--timeout", "240")
	t.Cleanup(func() {
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer restoreCancel()
		lifecycleRehearsal(t, restoreCtx, runDir, "ledger-fixture-restore", restoreLedgerEnv, "restart", "money-ledger@"+seed.Variant, "--path", seed.Source, "--json", "--timeout", "240")
	})
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := filepath.Clean(filepath.Join(cwd, "../../.."))

	var identityAuditMu sync.Mutex
	var identityAudit []map[string]any
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		tokenBytes, operationTokenBytes, decodeError, classification := 0, 0, "", ""
		defer func() {
			row := map[string]any{
				"method": r.Method, "path": r.URL.Path, "content_length": r.ContentLength,
				"remote": r.RemoteAddr, "token_bytes": tokenBytes,
				"operation_token_bytes": operationTokenBytes, "decode_error": decodeError,
				"classification": classification, "duration_ms": float64(time.Since(started).Microseconds()) / 1000,
			}
			identityAuditMu.Lock()
			identityAudit = append(identityAudit, row)
			identityAuditMu.Unlock()
		}()
		var in struct {
			Token          string `json:"token"`
			OperationToken string `json:"operation_token"`
		}
		if r.URL.Path != "/api/v1/identity/verify" {
			classification = "invalid_path"
			http.Error(w, "invalid fixture identity", http.StatusUnauthorized)
			return
		}
		decodeErr := json.NewDecoder(r.Body).Decode(&in)
		tokenBytes, operationTokenBytes = len(in.Token), len(in.OperationToken)
		if decodeErr != nil {
			decodeError = decodeErr.Error()
			classification = "body_decode_failed"
			http.Error(w, "invalid fixture identity", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if in.Token == "cf2-local-agent" && in.OperationToken == "" {
			classification = "static_agent_accepted"
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": agentclaims.For("cf2-agent")})
			return
		}
		baseline, baselineOK := verifyCF1RunToken(in.Token)
		if !baselineOK {
			classification = "run_identity_rejected"
			http.Error(w, "invalid local Agent Manager run identity", http.StatusUnauthorized)
			return
		}
		if in.OperationToken == "" {
			classification = "run_identity_accepted"
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": baseline})
			return
		}
		operation, operationOK := verifyCF1RunToken(in.OperationToken)
		if !operationOK || baseline.RunID != operation.RunID || baseline.TaskID != operation.TaskID || baseline.Subject != operation.Subject || operation.Meta["authorization_id"] == "" || operation.Meta["operation_grant"] == "" {
			classification = "operation_pair_rejected"
			http.Error(w, "invalid local Agent Manager identity pair", http.StatusUnauthorized)
			return
		}
		classification = "operation_pair_accepted"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid": true, "claims": baseline,
			"operation": map[string]any{"valid": true, "claims": operation},
		})
	}))
	defer identityServer.Close()
	t.Cleanup(func() {
		identityAuditMu.Lock()
		snapshot := append([]map[string]any(nil), identityAudit...)
		identityAuditMu.Unlock()
		writeRehearsalJSON(t, filepath.Join(runDir, "cf1-identity-requests.json"), snapshot)
	})
	variantPrefix := "cf2-"
	if cf1Only {
		variantPrefix = "cf1-"
	}
	variant := variantPrefix + fmt.Sprint(time.Now().Unix())
	lpbsDeliveryDir := filepath.Join(runDir, "lpbs-deliveries")
	require.NoError(t, os.MkdirAll(lpbsDeliveryDir, 0o700))
	var offerDeskURL, lpbsURL, lpbsSource, lpbsName string
	var canonicalOffer cf2CanonicalOffer
	var offerID, offerVariantID, leaseID string
	var routingClient routingconnect.RoutingServiceClient
	var cf2LPBSOrderID string
	if !cf1Only {
		lpbsVariant := "cf2-lpbs-" + fmt.Sprint(time.Now().UnixNano())
		lpbsDatabase := "vrooli_landing_page_business_suite_" + strings.ReplaceAll(lpbsVariant, "-", "_")
		resourceDatabaseCommand(t, ctx, runDir, "lpbs-database-create", "content", "create-database", lpbsDatabase)
		t.Cleanup(func() {
			cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
			defer stop()
			resourceDatabaseCommand(t, cleanup, runDir, "lpbs-database-remove", "content", "remove", lpbsDatabase)
		})
		offerDeskVariantDir := filepath.Join(runDir, "cf2-offer-desk-owner")
		require.NoError(t, os.MkdirAll(offerDeskVariantDir, 0o700))
		offerDeskSource := prepareCF5APIScenarioVariant(t, "offer-desk", "../../offer-desk", offerDeskVariantDir, map[string]string{
			"VROOLI_STORAGE_ROOT": filepath.Join(runDir, "cf2-offer-desk-storage"),
			"TREASURY_API_URL":    "",
		})
		disableCF4OfferDeskVectorSearch(t, offerDeskSource)
		offerDeskName := "offer-desk@" + instanceName("cf2-offer-desk-"+variant)
		offerDeskStart := lifecycleRehearsal(t, ctx, runDir, "cf2-offer-desk-start", map[string]string{}, "start", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
		offerDeskURL = rehearsalURL(t, offerDeskStart)
		offerDeskStopped := false
		t.Cleanup(func() {
			if !offerDeskStopped {
				stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
				offerDeskStopped = true
			}
		})
		canonicalOffer = readCF2CanonicalServiceOffer(t, ctx, offerDeskURL)
		offerID, offerVariantID = seedCF2CanonicalServiceOfferGraph(t, ctx, offerDeskURL, runDir, canonicalOffer)
		lpbsSource = prepareLPBSRehearsal(t, repo, runDir, map[string]string{
			"LPBS_REHEARSAL_CREDENTIAL_PATH": credentialPath,
			"LPBS_REHEARSAL_PAYER_IDENTITY":  "customer:cf2-local",
			// This fixture binding serves Treasury's separate customer funding receipt.
			// The x402 service-order buyer is not that test transfer's payer.
			"LPBS_REHEARSAL_PAYER_ADDRESS":       strings.ToLower(fork.Accounts[1]),
			"LPBS_REHEARSAL_BUSINESS_ACCOUNT_ID": "cf2-customer-" + variant,
			"LPBS_REHEARSAL_CUSTOMER_EMAIL":      "cf2-local@example.test",
			"LPBS_SERVICE_ORDER_TEST_MODE":       "0",
			"LPBS_REHEARSAL_DELIVERY_DIR":        lpbsDeliveryDir,
			"SCENARIO_AUTHENTICATOR_API_URL":     exchangeURL,
			"OFFER_DESK_API_URL":                 offerDeskURL,
		})
		lpbsName = "landing-page-business-suite@" + lpbsVariant
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
		lpbsURL = rehearsalURL(t, lpbsStart)
		leaseID = "cf2-lpbs-" + variant
		routingClient = routingconnect.NewRoutingServiceClient(http.DefaultClient, lpbsURL)
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
	}
	notificationRequests := make(chan *notificationsv1.SendRequest, 4)
	notificationMux := http.NewServeMux()
	notificationMux.Handle(notificationsconnect.NotificationsServiceSendProcedure, connect.NewUnaryHandler(
		notificationsconnect.NotificationsServiceSendProcedure,
		func(_ context.Context, request *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
			serviceToken := request.Header().Get("Authorization") == "Bearer "+noticeToken
			runToken := request.Header().Get("X-Agent-Identity-Token")
			runApprovalRelay := runToken != ""
			if runApprovalRelay {
				_, runApprovalRelay = verifyCF1RunToken(runToken)
			}
			if !serviceToken && !runApprovalRelay {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("expected the Notification Hub-bound Treasury service identity or a verified leased Agent Manager run identity"))
			}
			notificationRequests <- request.Msg
			return connect.NewResponse(&notificationsv1.SendResponse{}), nil
		},
	))
	notificationServer := httptest.NewServer(notificationMux)
	defer notificationServer.Close()
	const eventSecret = "cf2-local-vrooli-events-webhook-secret-only"
	decisionBridge := newCF1DecisionEventsBridge(eventSecret, runDir)
	eventServer := httptest.NewServer(decisionBridge)
	t.Cleanup(eventServer.Close)
	accessTeamDomain, accessAudience, accessJWKSURL, accessAssertion, closeAccess := startCF1AccessAssertion(t)
	t.Cleanup(closeAccess)
	hubVariant := instanceName("cf1-notification-hub")
	hubName := "notification-hub@" + hubVariant
	hubEnvironment := map[string]string{
		"VROOLI_STORAGE_ROOT":                  filepath.Join(runDir, "cf1-notification-hub-storage"),
		"VROOLI_TEST_MODE_FORCE_ENABLE":        "1",
		"VROOLI_NOTIFICATION_RECIPIENT":        "cf1-operator",
		"VROOLI_EVENTS_API_BASE":               eventServer.URL,
		"VROOLI_EVENTS_WEBHOOK_SECRET":         eventSecret,
		"SCENARIO_AUTHENTICATOR_API_URL":       exchangeURL,
		"VROOLI_AUTH_SCENARIO_JWKS_URL":        jwksURL,
		"VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN": accessTeamDomain,
		"VROOLI_CLOUDFLARE_ACCESS_AUDIENCE":    accessAudience,
		"VROOLI_CLOUDFLARE_ACCESS_JWKS_URL":    accessJWKSURL,
		"AGENT_MANAGER_API_URL":                identityServer.URL,
	}
	hubSource := prepareCF5APIScenarioVariant(t, "notification-hub", "../../notification-hub", runDir, hubEnvironment)
	hubStarted := lifecycleRehearsal(t, ctx, runDir, "cf1-notification-hub-start", map[string]string{
		"GOFLAGS": "", "API_TOKEN": "", "VROOLI_TEST_MODE_FORCE_ENABLE": "1",
	}, "start", hubName, "--path", hubSource, "--json", "--timeout", "240")
	hubURL := rehearsalURL(t, hubStarted)
	decisionBridgeSetHub(decisionBridge, hubURL)
	hubStopped := false
	t.Cleanup(func() {
		if hubStopped {
			return
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Minute)
		defer stopCancel()
		cmd := exec.CommandContext(stopCtx, "vrooli", "scenario", "stop", hubName, "--json")
		output, stopErr := cmd.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "cf1-notification-hub-stop.json"), output, 0o600)
		if stopErr != nil {
			t.Errorf("stop isolated CF1 Notification Hub variant: %v", stopErr)
		}
		hubStopped = true
	})
	hubLeaseID := "cf1-" + hubVariant
	hubRouting := routingconnect.NewRoutingServiceClient(&http.Client{Timeout: 10 * time.Second}, hubURL)
	_, err = hubRouting.InstallTestPool(ctx, connect.NewRequest(&routingpb.InstallTestPoolRequest{
		Dsn: filepath.Join(runDir, "cf1-notification-hub-lease.db"), LeaseId: hubLeaseID,
		LeaseTtlMs: int64((10 * time.Minute).Milliseconds()),
	}))
	require.NoError(t, err, "install run-only Notification Hub decision database")
	t.Cleanup(func() {
		_, clearErr := hubRouting.ClearTestPool(context.Background(), connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: hubLeaseID}))
		if clearErr != nil {
			t.Errorf("clear run-only Notification Hub decision database: %v", clearErr)
		}
	})
	t.Setenv("VROOLI_EVENTS_API_BASE", eventServer.URL)
	awaitDecisionEvent := func(waitCtx context.Context, approvalID string) (*notificationsv1.SendRequest, error) {
		deadline, cancel := context.WithTimeout(waitCtx, 10*time.Second)
		defer cancel()
		for {
			select {
			case raw := <-decisionBridge.decisionEvents:
				var envelope domainevents.EventEnvelope
				if protojson.Unmarshal(raw, &envelope) != nil || envelope.GetEventType() != "treasury.approval.decision_requested.v1" || !bytes.Contains(raw, []byte(approvalID)) {
					continue
				}
				if !bytes.Contains(raw, []byte("110 minor units USD")) && !bytes.Contains(raw, []byte("110 USD")) {
					return nil, fmt.Errorf("isolated decision event does not bind the expected over-cap amount: %s", raw)
				}
				_ = os.WriteFile(filepath.Join(runDir, "over-cap-approval-decision-event-"+approvalID+".json"), raw, 0o600)
				askID := decisionBridge.AskID(approvalID)
				if askID == "" {
					return nil, fmt.Errorf("Notification Hub did not retain an ask for Treasury approval %s", approvalID)
				}
				return &notificationsv1.SendRequest{Body: "110 USD minor units\n" + approvalID + "\nNotification Hub ask " + askID}, nil
			case <-deadline.Done():
				return nil, fmt.Errorf("Treasury decision event was not delivered to isolated Notification Hub: %w", deadline.Err())
			}
		}
	}
	source := prepareTreasuryRehearsal(t, repo, runDir, map[string]string{
		// A variant resolves its database under ~/.vrooli/data/vrooli unless told
		// otherwise and nothing removes it; keep it in the run, where
		// cf1RunWalletX402Credential also looks for it.
		"VROOLI_STORAGE_ROOT":                 filepath.Join(runDir, "storage"),
		"VROOLI_EVENTS_API_BASE":              eventServer.URL,
		"VROOLI_EVENTS_WEBHOOK_SECRET":        eventSecret,
		"TREASURY_REHEARSAL_DIR":              custodyDir,
		"TREASURY_BASE_RPC_URL":               fork.URL,
		"TREASURY_BASE_USDC_MODE":             "local-fork",
		"X402_RUNTIME_MODE":                   "local-fork",
		"X402_FACILITATOR_URL":                "http://127.0.0.1:14021",
		"MONEY_LEDGER_API_URL":                seed.URL,
		"TREASURY_LEDGER_BOOK_ID":             seed.Book,
		"TREASURY_LEDGER_ACCOUNT_ID":          seed.Account,
		"VROOLI_AUTH_SCENARIO_JWKS_URL":       jwksURL,
		"LANDING_PAGE_BUSINESS_SUITE_API_URL": lpbsURL,
		"OFFER_DESK_BASE_URL":                 offerDeskURL,
		"SCENARIO_AUTHENTICATOR_API_URL":      exchangeURL,
		"AGENT_MANAGER_API_URL":               identityServer.URL,
		"NOTIFICATION_HUB_API_URL":            notificationServer.URL,
		"TREASURY_DECISION_HUB_API_URL":       hubURL,
		"TREASURY_DECISION_HUB_TEST_MODE":     "1",
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
	decisionBridgeSetTreasury(decisionBridge, apiURL)
	select {
	case <-decisionBridge.treasuryReady:
	case <-time.After(30 * time.Second):
		t.Fatal("Treasury did not reconcile its Notification Hub answer subscription into the isolated Events fixture")
	}
	if !cf1Only {
		setCF5VariantEnvironment(t, lpbsSource, map[string]string{"TREASURY_API_URL": apiURL, "LPBS_SERVICE_ORDER_TEST_MODE": "1"})
		// The test lease registry belongs to the running LPBS process. Clear its
		// PostgreSQL database before replacement so the restarted owner can install
		// a fresh lease with the same run-local payer binding.
		_, err = routingClient.ClearTestPool(ctx, connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: leaseID}))
		require.NoError(t, err, "release LPBS test pool before restarting its owner")
		restartedLPBS := lifecycleRehearsal(t, ctx, runDir, "cf2-lpbs-bind-treasury", map[string]string{"GOFLAGS": "-tags=lpbs_rehearsal"}, "restart", lpbsName, "--path", lpbsSource, "--json", "--timeout", "240")
		restartedLPBSURL := rehearsalURL(t, restartedLPBS)
		require.Equal(t, lpbsURL, restartedLPBSURL, "LPBS restart must retain its isolated API URL")
		lpbsURL = restartedLPBSURL
		// LPBS test-pool leases are process-owned. Reinstall after the deliberate
		// owner restart so accepted-receipt payer bindings remain available to the
		// later Treasury→LPBS credit-lot call.
		_, err = routingClient.InstallTestPool(ctx, connect.NewRequest(&routingpb.InstallTestPoolRequest{
			Dsn: "file:cf2-lpbs-isolated", LeaseId: leaseID, LeaseTtlMs: int64((10 * time.Minute).Milliseconds()),
		}))
		require.NoError(t, err, "restore the isolated LPBS test-pool lease after its owner restart")
		priceID := canonicalOffer.X402PriceID
		declaredPrice := cf4DeclareOfferDeskPrice(t, ctx, apiURL, map[string]any{
			"offer_id": offerID, "variant_id": offerVariantID,
			"price": map[string]any{
				"id": priceID, "resource_url": strings.TrimRight(lpbsURL, "/") + canonicalOffer.MPPResourcePath,
				"description": canonicalOffer.Name, "mime_type": "application/json",
				"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
				"asset": basefork.USDC, "asset_decimals": 6, "max_timeout_seconds": 300,
				"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
			},
		})
		require.Equal(t, canonicalOffer.ProposedPrice.AmountMinor, declaredPrice.AmountMinor)
		require.Equal(t, canonicalOffer.ProposedPrice.Currency, declaredPrice.Currency)
		require.Equal(t, priceID, declaredPrice.ID)
		_, err = fork.Mint(ctx, x402BuyerAddress.Hex(), declaredPrice.Amount)
		require.NoError(t, err, "fund only the ephemeral buyer with the canonical $49 test payment on this local fork")
		writeRehearsalJSON(t, filepath.Join(runDir, "cf2-service-offer-price-join.json"), map[string]any{
			"canonical_offer": canonicalOffer, "catalog_offer_id": offerID, "catalog_variant_id": offerVariantID,
			"catalog_edge": "sells_at", "price_id": declaredPrice.ID, "resource_url": declaredPrice.ResourceURL,
			"amount_minor": declaredPrice.AmountMinor, "currency": declaredPrice.Currency,
		})
		cf2LPBSOrderID = runCF2AgentManagerLPBSOrder(t, ctx, repo, runDir, lpbsDeliveryDir, name, source, apiURL, lpbsURL, declaredPrice, canonicalOffer, x402BuyerKey, fork, receiptReaderToken)
		// Link the accepted seller receipt before later CF2 customer-funding events,
		// then repeat after both owners restart below to prove durable readback.
		reconcileCF2LPBSOrder(t, ctx, runDir, apiURL, lpbsURL, cf2LPBSOrderID, receiptReaderToken)
	}
	client := authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	fundingClient := fundingconnect.NewFundingAdminServiceClient(http.DefaultClient, apiURL)
	walletClient := walletconnect.NewWalletAdminServiceClient(http.DefaultClient, apiURL)
	instrumentClient := instrumentconnect.NewInstrumentAdminServiceClient(http.DefaultClient, apiURL)
	mandateClient := mandateconnect.NewMandateAdminServiceClient(http.DefaultClient, apiURL)
	bookClient := bookconnect.NewBookAdminServiceClient(http.DefaultClient, apiURL)
	budgetAdmin := budgetconnect.NewBudgetAdminServiceClient(http.DefaultClient, apiURL)
	agent := authconnect.NewAgentSpendClient(http.DefaultClient, apiURL)
	agentFunding := fundingconnect.NewFundingAgentServiceClient(http.DefaultClient, apiURL)
	bookID := "cf2-book-" + variant
	budgetID := "cf2-budget-" + variant
	_, err = bookClient.CreateBook(ctx, operatorRequest(&bookpb.CreateBookRequest{Book: &bookpb.Book{Id: bookID, Name: "CF2 isolated research"}}))
	require.NoError(t, err)
	_, err = budgetAdmin.SetBudgetCaps(ctx, operatorRequest(&budgetpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 10000, PeriodicCapMinor: 10000, PerTransactionCapMinor: 10000, PeriodSeconds: 3600}}))
	require.NoError(t, err)
	cf1AMRunID, cf1AMAuthorizationID, cf1AMSettlementID, err := runCF1AgentManagerCLIForkShowcase(t, ctx, repo, runDir, apiURL, identityServer.URL, source, name, client, notificationRequests, fork, bookID, variant, cf1ShowcaseHooks{AnswerApprovalInHub: func(waitCtx context.Context, approvalID string) (*notificationsv1.SendRequest, error) {
		notice, awaitErr := awaitDecisionEvent(waitCtx, approvalID)
		if awaitErr != nil {
			return nil, awaitErr
		}
		askID := decisionBridge.AskID(approvalID)
		if err := answerCF1NotificationHubAsk(waitCtx, hubURL, accessAssertion, runDir, approvalID, askID, decisionBridge.Correlation(approvalID)); err != nil {
			return nil, err
		}
		if err := decisionBridge.ReconcileAnswer(waitCtx, eventServer.URL, askID); err != nil {
			return nil, fmt.Errorf("deliver Notification Hub answer through Events subscription: %w", err)
		}
		return notice, nil
	}})
	require.NoError(t, err)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-agent-manager-cli-result.json"), map[string]any{
		"run_id": cf1AMRunID, "authorization_id": cf1AMAuthorizationID, "settlement_id": cf1AMSettlementID,
		"runner_to_cli": true, "local_fork": true,
	})
	// A second independent ordinary Agent Manager invocation exercises a second
	// budget address and run-owned session key through the same Treasury and
	// Base fork. Keep its durable artifacts isolated while preserving shared
	// book policy and chain state for cross-budget comparison.
	secondCF1RunDir := filepath.Join(runDir, "cf1-second-budget-run")
	require.NoError(t, os.MkdirAll(secondCF1RunDir, 0o700))
	secondCF1Variant := variant + "-second-budget"
	secondHooks := cf1ShowcaseHooks{
		AnswerApprovalInHub: func(waitCtx context.Context, approvalID string) (*notificationsv1.SendRequest, error) {
			notice, awaitErr := awaitDecisionEvent(waitCtx, approvalID)
			if awaitErr != nil {
				return nil, awaitErr
			}
			askID := decisionBridge.AskID(approvalID)
			if err := answerCF1NotificationHubAsk(waitCtx, hubURL, accessAssertion, secondCF1RunDir, approvalID, askID, decisionBridge.Correlation(approvalID)); err != nil {
				return nil, err
			}
			if err := decisionBridge.ReconcileAnswer(waitCtx, eventServer.URL, askID); err != nil {
				return nil, fmt.Errorf("deliver Notification Hub answer through Events subscription: %w", err)
			}
			return notice, nil
		},
		TreasuryCredentialPath: filepath.Join(runDir, "custody", "synthetic-custody.enc"),
	}
	secondRunID, secondAuthorizationID, secondSettlementID, secondRunErr := runCF1AgentManagerCLIForkShowcase(
		t, ctx, repo, secondCF1RunDir, apiURL, identityServer.URL, source, name, client, notificationRequests, fork,
		bookID, secondCF1Variant, secondHooks,
	)
	require.NoError(t, secondRunErr)
	var firstSession, secondSession struct {
		WalletID string `json:"wallet_id"`
		Address  string `json:"address"`
	}
	firstSessionBytes, err := os.ReadFile(filepath.Join(runDir, "agent-manager-cli", "ordinary-run-wallet.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(firstSessionBytes, &firstSession))
	secondSessionBytes, err := os.ReadFile(filepath.Join(secondCF1RunDir, "agent-manager-cli", "ordinary-run-wallet.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(secondSessionBytes, &secondSession))
	require.NotEmpty(t, firstSession.Address)
	require.NotEmpty(t, secondSession.Address)
	require.NotEmpty(t, firstSession.WalletID)
	require.NotEmpty(t, secondSession.WalletID)
	require.NotEqual(t, firstSession.Address, secondSession.Address, "two ordinary Agent Manager runs must receive distinct budget addresses")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-two-ordinary-budget-runs.json"), map[string]any{
		"first_run_id": cf1AMRunID, "first_budget_id": "cf1-agent-manager-budget-" + variant,
		"first_wallet_id": firstSession.WalletID, "first_budget_address": firstSession.Address,
		"first_authorization_id": cf1AMAuthorizationID, "first_settlement_id": cf1AMSettlementID,
		"second_run_id": secondRunID, "second_budget_id": "cf1-agent-manager-budget-" + secondCF1Variant,
		"second_wallet_id": secondSession.WalletID, "second_budget_address": secondSession.Address,
		"second_authorization_id": secondAuthorizationID, "second_settlement_id": secondSettlementID,
		"distinct_run_bound_budgets": true, "distinct_derived_addresses": true,
	})
	if !cf1Only && os.Getenv("TREASURY_CF7_MULTI_CALL_REHEARSAL") == "1" {
		cf7RunID, cf7AuthorizationID, cf7SettlementID, cf7RunErr := runCF7SeparateLocalForkShowcase(t, ctx, repo, runDir, apiURL, hubURL, accessAssertion, source, name, client, fork, bookID, variant, func(waitCtx context.Context, approvalID string) (*notificationsv1.SendRequest, error) {
			notice, awaitErr := awaitDecisionEvent(waitCtx, approvalID)
			if awaitErr != nil {
				return nil, awaitErr
			}
			askID := decisionBridge.AskID(approvalID)
			if err := answerCF1NotificationHubAsk(waitCtx, hubURL, accessAssertion, runDir, approvalID, askID, decisionBridge.Correlation(approvalID)); err != nil {
				return nil, err
			}
			if err := decisionBridge.ReconcileAnswer(waitCtx, eventServer.URL, askID); err != nil {
				return nil, fmt.Errorf("deliver CF7 Notification Hub answer through Events subscription: %w", err)
			}
			return notice, nil
		})
		require.NoError(t, cf7RunErr)
		writeRehearsalJSON(t, filepath.Join(runDir, "cf7-research-budget-multi-call-result.json"), map[string]any{
			"run_id": cf7RunID, "fourth_authorization_id": cf7AuthorizationID,
			"fourth_settlement_id": cf7SettlementID, "fresh_ordinary_run": true,
			"settled_calls": 4, "fourth_call_dispatched_after_treasury_approval": true,
			"fourth_call_retry_reused_same_receipt": true,
		})
	}
	if cf1Only {
		return
	}
	walletResult, err := walletClient.CreateResearchWallet(ctx, operatorRequest(&walletpb.CreateResearchWalletRequest{Id: "cf2-wallet-" + variant, BookId: bookID, Network: baseusdc.Network}))
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
	check, err := walletClient.VerifyBaseUSDCReceipt(ctx, operatorRequest(&walletpb.VerifyBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx}))
	require.NoError(t, err)
	require.Len(t, check.Msg.Transfers, 1)
	logIndex := check.Msg.Transfers[0].LogIndex
	submit := func(label, checkID, hash string, action ingestpb.CryptoReceiptAction, original, previous string, value *ingestpb.CryptoValuation, attribution *ingestpb.CryptoPayerAttribution) *fundingpb.ReceiptFundingDelivery {
		t.Helper()
		reason := label
		if action == ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL {
			reason = ""
		}
		response, err := fundingClient.SubmitReceiptFunding(ctx, operatorRequest(&fundingpb.SubmitReceiptFundingRequest{WalletId: w.Id, TransactionHash: hash, CheckId: checkID, LogIndex: logIndex, BudgetId: budgetID, LedgerBookId: seed.Book, LedgerAccountId: seed.Account, Action: action, OriginalReceiptId: original, PreviousReceiptId: previous, Reason: reason, Valuation: value, Attribution: attribution}))
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
	reincludedCheck, err := walletClient.RecheckBaseUSDCReceipt(ctx, operatorRequest(&walletpb.RecheckBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx, PriorCheckId: check.Msg.CheckId}))
	require.NoError(t, err)
	require.Equal(t, walletpb.BaseUSDCReceiptRelation_BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION, reincludedCheck.Msg.Check.Relation)
	reincluded := submit("reinclusion", reincludedCheck.Msg.Check.Id, tx, ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION, original.Receipt.Id, original.Receipt.Id, valuation, nil)
	require.Equal(t, "0", reincluded.Receipt.AtomicDelta)
	headroom("reinclusion", "1000001", 100)

	correctionCheck, err := walletClient.VerifyBaseUSDCReceipt(ctx, operatorRequest(&walletpb.VerifyBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx}))
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
	removedCheck, err := walletClient.RecheckBaseUSDCReceipt(ctx, operatorRequest(&walletpb.RecheckBaseUSDCReceiptRequest{WalletId: w.Id, TransactionHash: tx, PriorCheckId: reincludedCheck.Msg.Check.Id}))
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
	budgetWalletResponse, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf2-agent", AgentRunId: "test-run"}))
	require.NoError(t, err)
	budgetWallet := budgetWalletResponse.Msg.Wallet
	agentWalletRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID})
	agentWalletRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	agentWalletResponse, err := agentFunding.GetBudgetWallet(ctx, agentWalletRequest)
	require.NoError(t, err)
	require.Equal(t, budgetWallet.Address, agentWalletResponse.Msg.Wallet.Address)
	require.Equal(t, "cf2-agent", agentWalletResponse.Msg.Wallet.AgentSubject)
	require.Equal(t, "test-run", agentWalletResponse.Msg.Wallet.AgentRunId)
	intentResponse, err := fundingClient.CreateFundingIntent(ctx, operatorRequest(&fundingpb.CreateFundingIntentRequest{
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
	writeRehearsalJSON(t, filepath.Join(runDir, "customer-funding-transfer.json"), map[string]string{"intent_id": intentResponse.Msg.Intent.Id, "transaction_hash": customerTx, "chain": "local Base fork"})
	// Anvil's finalized tag advances only after enough local blocks have been
	// mined; the Treasury receipt watcher must only credit the finalized event.
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x80"}, nil))
	customerIntent := waitForCreditedFundingIntent(t, ctx, fundingClient, runDir, intentResponse.Msg.Intent.Id)
	require.Equal(t, customerTx, customerIntent.GetTransactionHash())
	require.Equal(t, "credited", customerIntent.GetState())
	customerDelivery, err := fundingClient.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: customerIntent.GetReceiptDeliveryId()}))
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
	_, err = fundingClient.SetWithdrawalDestination(ctx, operatorRequest(&fundingpb.SetWithdrawalDestinationRequest{Destination: &fundingpb.WithdrawalDestination{BudgetId: budgetID, Address: withdrawalDestination, Label: "CF2 local Coinbase test destination"}}))
	require.NoError(t, err)
	_, err = budgetAdmin.SetBudgetCaps(ctx, operatorRequest(&budgetpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 10000, PeriodicCapMinor: 10000, PerTransactionCapMinor: 100, PeriodSeconds: 3600}}))
	require.NoError(t, err)
	mandateResponse, err := mandateClient.CreateMandate(ctx, operatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: "cf2-withdrawal-mandate", IdempotencyKey: "cf2-withdrawal-mandate-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 1000, Currency: "USD", AllowedCounterparties: []string{withdrawalDestination}, AgentSubject: "cf2-agent", AgentRunId: "test-run",
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	require.Equal(t, mandatepb.MandateStatus_MANDATE_STATUS_LIVE, mandateResponse.Msg.Mandate.Status)
	quoteResponse, err := fundingClient.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
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
	_, err = fundingClient.ExecuteWithdrawal(ctx, missingIdentity)
	require.Error(t, err)
	wrongRealm := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.Quote.Id, IdempotencyKey: "cf2-withdrawal-mixed-realm"})
	wrongRealm.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	_, err = fundingClient.ExecuteWithdrawal(ctx, wrongRealm)
	require.Error(t, err, "operator auth must keep rejecting the ordinary agent realm header")
	wrongPrincipalQuote, err := fundingClient.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.Mandate.Id, Destination: withdrawalDestination,
		AtomicAmount: "1000000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 10,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-wrong-principal", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	wrongPrincipal := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: wrongPrincipalQuote.Msg.Quote.Id, IdempotencyKey: "cf2-withdrawal-wrong-principal"})
	wrongPrincipal.Header().Set("X-Treasury-Mandate-Identity-Token", "cf2-other-agent")
	wrongPrincipalResponse, err := fundingClient.ExecuteWithdrawal(ctx, wrongPrincipal)
	require.NoError(t, err)
	require.Equal(t, "refused", wrongPrincipalResponse.Msg.Withdrawal.State)
	require.Empty(t, wrongPrincipalResponse.Msg.Withdrawal.TransactionHash)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-wrong-principal-refused.json"), wrongPrincipalResponse.Msg.Withdrawal)
	pendingResponse, err := fundingClient.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, "pending_approval", pendingResponse.Msg.Withdrawal.State)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-pending-approval.json"), pendingResponse.Msg.Withdrawal)
	queued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	require.NoError(t, err)
	require.Len(t, queued.Msg.Approvals, 1)
	approved, err := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: queued.Msg.Approvals[0].Id, Resolution: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	require.NoError(t, err)
	require.Equal(t, treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED, approved.Msg.Approval.Status)
	settledResponse, err := fundingClient.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
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
	settledRetry, err := fundingClient.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, withdrawal.TransactionHash, settledRetry.Msg.Withdrawal.TransactionHash)
	require.Equal(t, withdrawal.Id, settledRetry.Msg.Withdrawal.Id)
	headroom("after-allowlisted-withdrawal", "2500000", 250, 100)

	// CF1: bind an agent-held session key to the run-bound mandate and derived
	// budget address, then settle two exact agent-signed intents through the
	// normal authorization, approval, instrument and settlement owners.
	spendRecipient := strings.ToLower(fork.Accounts[1])
	spendMandateResponse, err := mandateClient.CreateMandate(ctx, operatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: "cf1-agent-spend-mandate", IdempotencyKey: "cf1-agent-spend-mandate-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 140, Currency: "USD", AllowedCounterparties: []string{spendRecipient}, AgentSubject: "cf2-agent", AgentRunId: "test-run",
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	mandateID := spendMandateResponse.Msg.Mandate.Id
	sessionWalletResponse, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
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
	agentSessionWalletReadback, err := agentFunding.GetBudgetWallet(ctx, agentSessionWalletRequest)
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
		_, registerErr := instrumentClient.RegisterInstrument(ctx, operatorRequest(&instrumentpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
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
	overCapQueued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	require.NoError(t, err)
	require.Len(t, overCapQueued.Msg.Approvals, 1)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-over-cap-approval-queued.json"), overCapQueued.Msg.Approvals[0])
	overCapApproval, err := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: overCapQueued.Msg.Approvals[0].Id, Resolution: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
	require.NoError(t, err)
	require.Equal(t, treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED, overCapApproval.Msg.Approval.Status)
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
	_, err = budgetAdmin.FreezeBudget(ctx, operatorRequest(&budgetpb.FreezeBudgetRequest{BudgetId: budgetID}))
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
	frozenQuote, err := fundingClient.PreviewWithdrawal(ctx, operatorRequest(&fundingpb.PreviewWithdrawalRequest{
		BudgetId: budgetID, MandateId: mandateResponse.Msg.Mandate.Id, Destination: withdrawalDestination,
		AtomicAmount: "500000", MaxFeeWei: "1000000000000000000", FeeValuationMinor: 5,
		Valuation: &ingestpb.CryptoValuation{AmountMinor: 50, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "local-fork:test-only-frozen-withdrawal", ValuedAt: timestamppb.Now()},
	}))
	require.NoError(t, err)
	frozenRequest := operatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: frozenQuote.Msg.Quote.Id, IdempotencyKey: "cf2-frozen-withdrawal-key"})
	frozenRequest.Header().Set("X-Treasury-Mandate-Identity-Token", "cf2-local-agent")
	frozenResponse, err := fundingClient.ExecuteWithdrawal(ctx, frozenRequest)
	require.NoError(t, err)
	require.Equal(t, "refused", frozenResponse.Msg.Withdrawal.State)
	writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-frozen-refused.json"), frozenResponse.Msg.Withdrawal)
	require.Empty(t, frozenResponse.Msg.Withdrawal.TransactionHash)
	unfrozen, err := budgetAdmin.UnfreezeBudget(ctx, operatorRequest(&budgetpb.UnfreezeBudgetRequest{BudgetId: budgetID}))
	require.NoError(t, err)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-budget-unfrozen-for-exact-replay.json"), unfrozen.Msg)
	headroom("after-budget-freeze", "2500000", 250, 0)

	// Process restarts use the ordinary lifecycle owner, with the same isolated
	// databases and authority. New PIDs/build identities are retained in logs.
	lifecycleRehearsal(t, ctx, runDir, "ledger-restart", ledgerEnv, "restart", "money-ledger@"+seed.Variant, "--path", seed.Source, "--json", "--timeout", "240")
	restarted := lifecycleRehearsal(t, ctx, runDir, "treasury-restart", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", name, "--path", source, "--json", "--timeout", "240")
	apiURL = rehearsalURL(t, restarted)
	reconcileCF2LPBSOrder(t, ctx, runDir, apiURL, lpbsURL, cf2LPBSOrderID, receiptReaderToken)
	client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	agent = authconnect.NewAgentSpendClient(http.DefaultClient, apiURL)
	fundingClient = fundingconnect.NewFundingAdminServiceClient(http.DefaultClient, apiURL)
	agentFunding = fundingconnect.NewFundingAgentServiceClient(http.DefaultClient, apiURL)
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
	sessionWalletReadback, err := agentFunding.GetBudgetWallet(ctx, sessionWalletReadbackRequest)
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
	rotatedWalletResponse, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf2-agent", AgentRunId: "test-run", MandateId: mandateID, RotateSession: true, RotationIdempotencyKey: "cf2-agent-spend-session-rotation",
	}))
	require.NoError(t, err)
	rotatedWallet := rotatedWalletResponse.Msg.Wallet
	require.Equal(t, sessionWallet.SessionGeneration+1, rotatedWallet.SessionGeneration)
	require.Equal(t, sessionWallet.Address, rotatedWallet.Address, "rotation replaces signing authority without changing the derived budget address")
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-session-rotated.json"), rotatedWallet)
	revokeResponse, err := mandateClient.RevokeMandateSession(ctx, operatorRequest(&mandatepb.RevokeMandateSessionRequest{MandateId: mandateID}))
	require.NoError(t, err)
	require.True(t, revokeResponse.Msg.Revoked)
	writeRehearsalProto(t, filepath.Join(runDir, "cf1-session-revoked.json"), revokeResponse.Msg)
	revokedSessionReadRequest := connect.NewRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: budgetID, MandateId: mandateID})
	revokedSessionReadRequest.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	_, revokedSessionErr := agentFunding.GetBudgetWallet(ctx, revokedSessionReadRequest)
	require.Error(t, revokedSessionErr, "revoked session must no longer be exposed to the agent")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-session-revoked-agent-read-refused.json"), map[string]string{
		"mandate_id":      mandateID,
		"read_error_code": connect.CodeOf(revokedSessionErr).String(),
	})
	persistedWithdrawal, err := fundingClient.GetWithdrawal(ctx, operatorRequest(&fundingpb.GetWithdrawalRequest{Id: withdrawal.Id}))
	require.NoError(t, err)
	require.Equal(t, "settled", persistedWithdrawal.Msg.Withdrawal.State)
	require.Equal(t, withdrawal.TransactionHash, persistedWithdrawal.Msg.Withdrawal.TransactionHash)
	withdrawalRetryAfterRestart, err := fundingClient.ExecuteWithdrawal(ctx, withdrawalRequest("cf2-withdrawal-key"))
	require.NoError(t, err)
	require.Equal(t, withdrawal.TransactionHash, withdrawalRetryAfterRestart.Msg.Withdrawal.TransactionHash)
	authorizationReadback := connect.NewRequest(&authpb.GetAuthorizationRequest{Id: withdrawal.AuthorizationId})
	authorizationReadback.Header().Set(identity.HeaderAgentIdentityToken, "cf2-local-agent")
	verifiedAuthorization, err := agent.GetAuthorization(ctx, authorizationReadback)
	require.NoError(t, err)
	require.Equal(t, authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED, verifiedAuthorization.Msg.Authorization.Verdict)
	readback, err := fundingClient.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: removed.Id}))
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
	linkedCustomer, err := fundingClient.GetReceiptFunding(ctx, operatorRequest(&fundingpb.GetReceiptFundingRequest{Id: customer.Id}))
	require.NoError(t, err)
	writeRehearsalProto(t, filepath.Join(runDir, "lpbs-credit-lot-link.json"), linkedCustomer.Msg.Delivery)
	require.Equal(t, customer.Receipt.Id, linkedCustomer.Msg.Delivery.CreditLotReceiptId)
	require.Equal(t, "landing-page-business-suite", linkedCustomer.Msg.Delivery.CreditLotOwner)
	require.NotEmpty(t, linkedCustomer.Msg.Delivery.CreditLotId, "LPBS must return an actual credit_funding_lots row ID")
	t.Log("five exact-token receipt events, actual LPBS credit_funding_lots association, mandate-bound agent session signing, in-cap and human-approved over-cap settlements, freeze denial, exact receipt attribution and idempotent replay survived Treasury restart")
}

func runCF7SeparateLocalForkShowcase(t *testing.T, ctx context.Context, repo, runDir, apiURL, hubURL, accessAssertion, treasurySource, treasuryName string, client authconnect.TreasuryAdminClient, fork *basefork.Fork, bookID, variant string, answerApprovalInHub func(context.Context, string) (*notificationsv1.SendRequest, error)) (string, string, string, error) {
	t.Helper()
	offerDeskEnv := map[string]string{
		"VROOLI_STORAGE_ROOT":                      filepath.Join(runDir, "cf7-offer-desk-storage"),
		"TREASURY_API_URL":                         apiURL,
		"OFFER_DESK_TEST_DROP_FIRST_PAID_RESPONSE": "0",
		"OFFER_DESK_CF7_TEST_SECOND_PRICE":         "1",
	}
	offerDeskSource := prepareCF5APIScenarioVariant(t, "offer-desk", "../../offer-desk", runDir, offerDeskEnv)
	disableCF4OfferDeskVectorSearch(t, offerDeskSource)
	offerDeskName := "offer-desk@" + instanceName("cf7-offer-desk-"+variant)
	offerDeskStart := lifecycleRehearsal(t, ctx, runDir, "cf7-offer-desk-start", map[string]string{}, "start", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	offerDeskURL := rehearsalURL(t, offerDeskStart)
	offerDeskStopped := false
	t.Cleanup(func() {
		if !offerDeskStopped {
			stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
			offerDeskStopped = true
		}
	})
	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	cf5ProvisionPinnedFacilitatorKey(t, facilitatorKey)
	cf5StartPinnedFacilitator(t, ctx, runDir)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf7-facilitator-local-fork.json"), map[string]any{
		"resource": "x402-facilitator-local-test", "url": "http://127.0.0.1:14021",
		"fork_rpc": "http://127.0.0.1:18545", "facilitator_signer": facilitatorAddress.Hex(), "test_only": true,
	})
	setCF5VariantEnvironment(t, treasurySource, map[string]string{
		// Reassert the root on the lifecycle restart so Treasury does not
		// reopen the host's default data tree and the signer lookup remains
		// bound to this rehearsal's database.
		"VROOLI_STORAGE_ROOT":  filepath.Join(runDir, "storage"),
		"OFFER_DESK_BASE_URL":  offerDeskURL,
		"X402_FACILITATOR_URL": "http://127.0.0.1:14021",
	})
	restarted := lifecycleRehearsal(t, ctx, runDir, "cf7-treasury-bind-offer-desk", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
	if rehearsalURL(t, restarted) != apiURL {
		return "", "", "", errors.New("CF7 Treasury restart changed its isolated API URL")
	}
	client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	offerDeskEvidenceDir := filepath.Join(runDir, "cf7-offer-desk")
	require.NoError(t, os.MkdirAll(offerDeskEvidenceDir, 0o700))
	offerID, variantID := seedCF4OfferDeskGraph(t, ctx, offerDeskURL, offerDeskEvidenceDir)
	paidURL := strings.TrimRight(offerDeskURL, "/") + "/api/v1/paid/catalog"
	firstPrice := cf4DeclareOfferDeskPrice(t, ctx, apiURL, map[string]any{
		"offer_id": offerID, "variant_id": variantID,
		"price": map[string]any{
			"id": cf4OfferDeskTestPriceID, "resource_url": paidURL,
			"description": "CF7 research inference call", "mime_type": "application/json",
			"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
			"asset": baseusdc.USDCAddress, "asset_decimals": 6, "max_timeout_seconds": 300,
			"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
		},
	})
	require.Equal(t, int64(1), firstPrice.AmountMinor)
	overOfferID, overVariantID := seedCF7OverCapOfferDeskGraph(t, ctx, offerDeskURL, offerDeskEvidenceDir, variant)
	overCapURL := strings.TrimRight(offerDeskURL, "/") + "/api/v1/paid/catalog/over-cap"
	overPrice := cf4DeclareOfferDeskPrice(t, ctx, apiURL, map[string]any{
		"offer_id": overOfferID, "variant_id": overVariantID,
		"price": map[string]any{
			"id": "cf7-research-inference-overcap", "resource_url": overCapURL,
			"description": "CF7 separate over-cap research inference quote", "mime_type": "application/json",
			"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
			"asset": baseusdc.USDCAddress, "asset_decimals": 6, "max_timeout_seconds": 300,
			"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
		},
	})
	require.Equal(t, int64(110), overPrice.AmountMinor)
	endpoint, err := url.ParseRequestURI(paidURL)
	if err != nil {
		return "", "", "", err
	}
	custodyPath := filepath.Join(runDir, "custody", "synthetic-custody.enc")
	x402 := &cf1X402ShowcaseConfig{
		CatalogURL: paidURL, PriceID: firstPrice.ID, CallID: "cf7-research-inference-call",
		Counterparty: strings.ToLower(endpoint.Hostname()), InstrumentID: "cf1-agent-manager-instrument-x402-cf7-" + variant,
		OverCapCatalogURL: overCapURL, OverCapPriceID: overPrice.ID, OverCapCallID: "cf7-research-inference-overcap-call",
		PrepareSigner: func(_ context.Context, walletID, walletAddress string) (string, error) {
			return cf1RunWalletX402Credential(t, runDir, custodyPath, walletID, walletAddress, paidURL)
		},
		PrepareOverCapSigner: func(_ context.Context, walletID, walletAddress string) (string, error) {
			return cf1RunWalletX402Credential(t, runDir, custodyPath, walletID, walletAddress, overCapURL)
		},
	}
	cf7RunID, decisionAuthorizationID, decisionSettlementID, err := runCF1AgentManagerCLIForkShowcase(t, ctx, repo, runDir, apiURL, "", treasurySource, treasuryName, client, nil, fork, bookID, "cf7-"+variant, cf1ShowcaseHooks{X402: x402, CF7Only: true, AnswerApprovalInHub: answerApprovalInHub})
	if err != nil {
		return "", "", "", err
	}
	if err := sendCF7BudgetAttention(t, ctx, hubURL, accessAssertion, cf7RunID, filepath.Join(runDir, "cf7-agent-manager-cli", "commission-checkpoint.json"), runDir); err != nil {
		return "", "", "", fmt.Errorf("publish the exact CF7 spend/result comparison to Notification Hub: %w", err)
	}
	return cf7RunID, decisionAuthorizationID, decisionSettlementID, nil
}

type cf7AttentionCheckpoint struct {
	RunID           string `json:"run_id"`
	Currency        string `json:"currency"`
	UnitCode        string `json:"unit_code"`
	CapMinor        int64  `json:"cap_minor"`
	SettledMinor    int64  `json:"settled_minor"`
	RemainingMinor  int64  `json:"remaining_minor"`
	FreeComparisons []struct {
		SettlementID    string `json:"settlement_id"`
		ReceiptID       string `json:"treasury_receipt_id"`
		PaidAmountMinor int64  `json:"paid_amount_minor"`
		PaidBenefit     string `json:"paid_benefit_summary"`
		FreeAlternative struct {
			SourceID   string `json:"source_id"`
			Label      string `json:"label"`
			ResultRef  string `json:"result_ref"`
			Digest     string `json:"sha256"`
			BodyBytes  int    `json:"body_bytes"`
			Limitation string `json:"comparison"`
		} `json:"free_alternative"`
	} `json:"free_comparisons"`
	ComputeProcurement string `json:"compute_procurement"`
	DatasetProcurement string `json:"dataset_procurement"`
}

func cf7AttentionBody(checkpoint cf7AttentionCheckpoint, expectedRunID string) (string, error) {
	if checkpoint.RunID != expectedRunID || checkpoint.Currency == "" || checkpoint.UnitCode == "" || checkpoint.CapMinor < 0 || checkpoint.SettledMinor < 0 || checkpoint.RemainingMinor != checkpoint.CapMinor-checkpoint.SettledMinor || len(checkpoint.FreeComparisons) != 4 {
		return "", errors.New("CF7 attention report is incomplete or belongs to another run")
	}
	var body strings.Builder
	fmt.Fprintf(&body, "Run %s settled %d %s of %d %s; remaining %d %s.\n", checkpoint.RunID, checkpoint.SettledMinor, checkpoint.UnitCode, checkpoint.CapMinor, checkpoint.UnitCode, checkpoint.RemainingMinor, checkpoint.UnitCode)
	settlementIDs, receiptIDs := map[string]struct{}{}, map[string]struct{}{}
	var comparedSpend int64
	for _, comparison := range checkpoint.FreeComparisons {
		if comparison.SettlementID == "" || comparison.ReceiptID == "" || comparison.PaidAmountMinor < 0 || comparison.PaidBenefit == "" || comparison.FreeAlternative.SourceID == "" || comparison.FreeAlternative.Label == "" || comparison.FreeAlternative.ResultRef == "" || comparison.FreeAlternative.Digest == "" || comparison.FreeAlternative.BodyBytes <= 0 || comparison.FreeAlternative.Limitation == "" {
			return "", errors.New("CF7 attention report has a missing paid result, receipt or free-alternative comparison")
		}
		if _, exists := settlementIDs[comparison.SettlementID]; exists {
			return "", errors.New("CF7 attention report repeats a paid settlement")
		}
		if _, exists := receiptIDs[comparison.ReceiptID]; exists {
			return "", errors.New("CF7 attention report repeats a Treasury receipt")
		}
		settlementIDs[comparison.SettlementID] = struct{}{}
		receiptIDs[comparison.ReceiptID] = struct{}{}
		if comparison.PaidAmountMinor > (1<<63-1)-comparedSpend {
			return "", errors.New("CF7 attention report spend total overflows")
		}
		comparedSpend += comparison.PaidAmountMinor
		fmt.Fprintf(&body, "Settlement %s (receipt %s): paid %d %s; result: %s; free option %s (%s) returned %d bytes, digest %s; limitation: %s.\n", comparison.SettlementID, comparison.ReceiptID, comparison.PaidAmountMinor, checkpoint.UnitCode, comparison.PaidBenefit, comparison.FreeAlternative.Label, comparison.FreeAlternative.SourceID, comparison.FreeAlternative.BodyBytes, comparison.FreeAlternative.Digest, comparison.FreeAlternative.Limitation)
	}
	if comparedSpend != checkpoint.SettledMinor {
		return "", errors.New("CF7 attention report paid comparisons do not reconcile to Treasury's settled total")
	}
	fmt.Fprintf(&body, "Compute: %s. Dataset: %s.", checkpoint.ComputeProcurement, checkpoint.DatasetProcurement)
	if body.Len() > 12<<10 {
		return "", errors.New("CF7 attention report exceeds its 12 KiB notification bound")
	}
	return body.String(), nil
}

func sendCF7BudgetAttention(t *testing.T, ctx context.Context, hubURL, accessAssertion, runID, checkpointPath, evidenceDir string) error {
	t.Helper()
	if strings.TrimSpace(accessAssertion) == "" {
		return errors.New("verified local operator identity is required for the private CF7 report notice")
	}
	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		return fmt.Errorf("read completed Agent Manager commission checkpoint: %w", err)
	}
	var checkpoint cf7AttentionCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return fmt.Errorf("decode completed Agent Manager commission checkpoint: %w", err)
	}
	body, err := cf7AttentionBody(checkpoint, runID)
	if err != nil {
		return err
	}
	client := notificationsconnect.NewNotificationsServiceClient(&http.Client{Timeout: 10 * time.Second}, hubURL)
	request := connect.NewRequest(&notificationsv1.SendRequest{
		Title: "CF7 local research spend and result",
		Body:  body, Urgency: "normal", SensitivityLabel: "private",
		IdempotencyKey: "cf7-research-budget-report-" + runID,
		DedupeKey:      "cf7-research-budget-report-" + runID,
	})
	request.Header().Set("Cf-Access-Jwt-Assertion", accessAssertion)
	request.Header().Set("X-Vrooli-Test-Mode", "1")
	response, err := client.Send(ctx, request)
	if err != nil {
		return fmt.Errorf("send private Notification Hub spend/result report: %w", err)
	}
	if response.Msg.GetNotification().GetId() == "" {
		return errors.New("Notification Hub accepted no durable CF7 report notification identity")
	}
	sum := sha256.Sum256([]byte(body))
	writeRehearsalJSON(t, filepath.Join(evidenceDir, "cf7-notification-hub-report.json"), map[string]any{
		"run_id": runID, "notification_id": response.Msg.GetNotification().GetId(),
		"idempotency_key": "cf7-research-budget-report-" + runID,
		"body_sha256":     hex.EncodeToString(sum[:]), "body": body, "private": true,
	})
	return nil
}

func TestCF7AttentionBodyReportsPaidResultBesideBoundedFreeAlternative(t *testing.T) {
	var checkpoint cf7AttentionCheckpoint
	const receipt = `{"run_id":"run-1","currency":"USD","unit_code":"USD_MINOR","cap_minor":500,"settled_minor":113,"remaining_minor":387,"free_comparisons":[{"settlement_id":"s1","treasury_receipt_id":"r1","paid_amount_minor":1,"paid_benefit_summary":"three paid records, response sha256=paid1","free_alternative":{"source_id":"offer-desk:free-sample","label":"no-signup sample","result_ref":"free1","sha256":"free-digest-1","body_bytes":81,"comparison":"shape sample only; not the current catalog"}},{"settlement_id":"s2","treasury_receipt_id":"r2","paid_amount_minor":1,"paid_benefit_summary":"three paid records, response sha256=paid2","free_alternative":{"source_id":"offer-desk:free-sample","label":"no-signup sample","result_ref":"free2","sha256":"free-digest-2","body_bytes":82,"comparison":"shape sample only; not the current catalog"}},{"settlement_id":"s3","treasury_receipt_id":"r3","paid_amount_minor":1,"paid_benefit_summary":"three paid records, response sha256=paid3","free_alternative":{"source_id":"offer-desk:free-sample","label":"no-signup sample","result_ref":"free3","sha256":"free-digest-3","body_bytes":83,"comparison":"shape sample only; not the current catalog"}},{"settlement_id":"s4","treasury_receipt_id":"r4","paid_amount_minor":110,"paid_benefit_summary":"over-cap approved paid result","free_alternative":{"source_id":"offer-desk:free-sample","label":"no-signup sample","result_ref":"free4","sha256":"free-digest-4","body_bytes":84,"comparison":"shape sample only; not the current catalog"}}],"compute_procurement":"not_run: CI4/CI9 supplier is unresolved","dataset_procurement":"not_run: no owner-bound dataset source"}`
	if err := json.Unmarshal([]byte(receipt), &checkpoint); err != nil {
		t.Fatal(err)
	}
	body, err := cf7AttentionBody(checkpoint, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"113 USD_MINOR of 500 USD_MINOR", "s4 (receipt r4): paid 110 USD_MINOR", "over-cap approved paid result", "no-signup sample (offer-desk:free-sample) returned 84 bytes", "not the current catalog", "not_run: CI4/CI9 supplier is unresolved", "not_run: no owner-bound dataset source"} {
		if !strings.Contains(body, want) {
			t.Fatalf("CF7 attention body omits %q: %s", want, body)
		}
	}
	checkpoint.RunID = "foreign-run"
	if _, err := cf7AttentionBody(checkpoint, "run-1"); err == nil {
		t.Fatal("attention report must reject another run's commission checkpoint")
	}
	checkpoint.RunID = "run-1"
	checkpoint.SettledMinor++
	if _, err := cf7AttentionBody(checkpoint, "run-1"); err == nil {
		t.Fatal("attention report must refuse spend that does not reconcile to Treasury's settled total")
	}
}

func TestSendCF7BudgetAttentionUsesPrivateIdempotentHubNotice(t *testing.T) {
	checkpointPath := filepath.Join(t.TempDir(), "commission-checkpoint.json")
	evidenceDir := t.TempDir()
	runID := "282159e3-8730-40ef-95af-307494b74ed4"
	comparisons := make([]map[string]any, 4)
	for i := range comparisons {
		comparisons[i] = map[string]any{
			"settlement_id": fmt.Sprintf("s%d", i+1), "treasury_receipt_id": fmt.Sprintf("r%d", i+1), "paid_amount_minor": 1,
			"paid_benefit_summary": "paid result", "free_alternative": map[string]any{
				"source_id": "offer-desk:free-sample", "label": "no-signup sample", "result_ref": fmt.Sprintf("free%d", i+1),
				"sha256": strings.Repeat("a", 64), "body_bytes": 81, "comparison": "shape sample only",
			},
		}
	}
	checkpointBytes, err := json.Marshal(map[string]any{
		"run_id": runID, "currency": "USD", "unit_code": "USD_MINOR", "cap_minor": 500, "settled_minor": 4, "remaining_minor": 496,
		"free_comparisons": comparisons, "compute_procurement": "not_run: supplier contract pending", "dataset_procurement": "not_run: dataset API pending",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checkpointPath, checkpointBytes, 0o600))
	var received []*notificationsv1.SendRequest
	serverMux := http.NewServeMux()
	serverMux.Handle(notificationsconnect.NotificationsServiceSendProcedure, connect.NewUnaryHandler(
		notificationsconnect.NotificationsServiceSendProcedure,
		func(_ context.Context, request *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
			received = append(received, proto.Clone(request.Msg).(*notificationsv1.SendRequest))
			if request.Header().Get("Cf-Access-Jwt-Assertion") != "operator-token" || request.Header().Get("X-Vrooli-Test-Mode") != "1" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing local operator test identity"))
			}
			return connect.NewResponse(&notificationsv1.SendResponse{Notification: &notificationshared.Notification{Id: "notice-1"}}), nil
		},
	))
	server := httptest.NewServer(serverMux)
	defer server.Close()
	for i := 0; i < 2; i++ {
		require.NoError(t, sendCF7BudgetAttention(t, context.Background(), server.URL, "operator-token", runID, checkpointPath, evidenceDir))
	}
	require.Len(t, received, 2)
	require.Equal(t, received[0], received[1])
	require.Equal(t, "private", received[0].GetSensitivityLabel())
	require.Equal(t, "cf7-research-budget-report-"+runID, received[0].GetIdempotencyKey())
	require.Contains(t, received[0].GetBody(), "s4 (receipt r4): paid 1 USD_MINOR")
	var evidence map[string]any
	evidenceBytes, err := os.ReadFile(filepath.Join(evidenceDir, "cf7-notification-hub-report.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(evidenceBytes, &evidence))
	require.Equal(t, "notice-1", evidence["notification_id"])
	require.Equal(t, runID, evidence["run_id"])
}

func seedCF7OverCapOfferDeskGraph(t *testing.T, ctx context.Context, baseURL, runDir, variant string) (string, string) {
	t.Helper()
	catalogClient := offersconnect.NewCatalogServiceClient(&http.Client{Timeout: 15 * time.Second}, baseURL)
	offer, err := catalogClient.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_OFFER, Name: "cf7-overcap-research-" + variant, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	offerVariant, err := catalogClient.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_VARIANT, Name: "cf7-overcap-call-" + variant, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	_, err = catalogClient.CreateEdge(ctx, connect.NewRequest(&offerspb.CreateEdgeRequest{Edge: &offerspb.Edge{
		FromId: offer.Msg.GetNode().GetId(), ToId: offerVariant.Msg.GetNode().GetId(), Kind: "sells_at",
		IntendedPriceMinor: 110, Currency: "USD", IntendedPriceDeclared: true,
	}}))
	require.NoError(t, err)
	writeCF5JSON(t, filepath.Join(runDir, "cf7-overcap-offer-desk-graph.json"), map[string]any{
		"offer_id": offer.Msg.GetNode().GetId(), "variant_id": offerVariant.Msg.GetNode().GetId(),
		"edge": "sells_at", "amount_minor": 110, "currency": "USD", "declared": true,
	})
	return offer.Msg.GetNode().GetId(), offerVariant.Msg.GetNode().GetId()
}

func waitForCreditedFundingIntent(t *testing.T, ctx context.Context, client fundingconnect.FundingAdminServiceClient, runDir, id string) *fundingpb.FundingIntent {
	t.Helper()
	watchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		response, err := client.GetFundingIntent(watchCtx, operatorRequest(&fundingpb.FundingIntentRequest{Id: id}))
		if err == nil && response.Msg.GetIntent().GetState() == "credited" {
			writeRehearsalProto(t, filepath.Join(runDir, "customer-funding-watcher-credit.json"), response.Msg.GetIntent())
			return response.Msg.GetIntent()
		}
		if err != nil && watchCtx.Err() == nil {
			t.Fatalf("read funding intent while waiting for Treasury's receipt watcher: %v", err)
		}
		select {
		case <-watchCtx.Done():
			t.Fatalf("Treasury receipt watcher did not credit intent %s before timeout: %v", id, watchCtx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
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

type cf1ShowcaseHooks struct {
	FundingAtomicAmount string
	// TreasuryCredentialPath lets another isolated Agent Manager run add
	// run-specific rail quotes to the one Treasury-owned local vault.
	TreasuryCredentialPath string
	// FundAgentWallet lets another owner exercise the actual funding path before
	// the Agent Manager run attempts a purchase. It must leave the exact local
	// fork balance in place and return only after accepted accounting is read back.
	FundAgentWallet func(context.Context, string, string, string, string, string) error
	// AnswerApprovalInHub exercises the real Hub answer and Treasury callback path
	// when the ordinary-run approval is delivered through Notification Hub.
	AnswerApprovalInHub func(context.Context, string) (*notificationsv1.SendRequest, error)
	// AwaitApprovalNotice supports the separate CF5 relay showcase, which still
	// resolves Treasury's local decision while exercising its funding notices.
	AwaitApprovalNotice func(context.Context, string) (*notificationsv1.SendRequest, error)
	// X402 supplies one Offer Desk-priced endpoint and a CA-only signer value
	// after the ordinary run's budget wallet and mandate exist. The lifecycle
	// harness owns price admission, response loss and Treasury restart proof.
	X402    *cf1X402ShowcaseConfig
	CF7Only bool
	// PairedRun enables actual reciprocal Treasury wallet/mandate refusal
	// probes between two concurrently active ordinary Agent Manager runs.
	PairedRun *cf1PairedRunHarness
}

type cf1PairedRunIdentity struct {
	RunID             string `json:"run_id"`
	BudgetID          string `json:"budget_id"`
	MandateID         string `json:"mandate_id"`
	WalletID          string `json:"wallet_id"`
	WalletAddress     string `json:"wallet_address"`
	WalletRunID       string `json:"wallet_run_id"`
	WalletBudgetID    string `json:"wallet_budget_id"`
	WalletMandateID   string `json:"wallet_mandate_id"`
	MandateStatus     string `json:"mandate_status"`
	SessionGeneration int64  `json:"session_generation"`
	CoordDir          string `json:"-"`
}

type cf1PairedProbeMarker struct {
	RunID                        string `json:"run_id"`
	CounterpartRunID             string `json:"counterpart_run_id"`
	TargetBudgetID               string `json:"target_budget_id"`
	TargetMandateID              string `json:"target_mandate_id"`
	CounterpartMandateStatus     string `json:"counterpart_mandate_status"`
	CounterpartSessionGeneration int64  `json:"counterpart_session_generation"`
	ForeignWalletProbeStatus     string `json:"foreign_wallet_probe_status"`
	ForeignMandateProbeStatus    string `json:"foreign_mandate_probe_status"`
}

type cf1PairedRunConfig struct {
	ForeignBudgetID       string
	ForeignMandateID      string
	CounterpartIdentity   map[string]any
	PairedProbeBarrierDir string
}

type cf1PairedIdentityEvidence struct {
	Agents []struct {
		RunID                     string `json:"run_id"`
		ProfileID                 string `json:"profile_id"`
		BudgetID                  string `json:"budget_id"`
		MandateID                 string `json:"mandate_id"`
		WalletID                  string `json:"wallet_id"`
		WalletAddress             string `json:"wallet_address"`
		OwnAuthorizationID        string `json:"own_authorization_id"`
		OwnSettlementID           string `json:"own_settlement_id"`
		OwnTxHash                 string `json:"own_tx_hash"`
		OwnHeadroomMinor          int64  `json:"own_headroom_minor"`
		ForeignWalletProbeStatus  string `json:"foreign_wallet_probe_status"`
		ForeignMandateProbeStatus string `json:"foreign_mandate_probe_status"`
	} `json:"agents"`
}

type cf1PairedRunHarness struct {
	mu               sync.Mutex
	arrivals         []cf1PairedRunIdentity
	cancels          []context.CancelFunc
	ready            chan struct{}
	readyOnce        sync.Once
	controller       sync.Once
	failureOnce      sync.Once
	apiURL           string
	identityPath     string
	barrierDir       string
	postIdentityGate chan struct{}
}

func (h *cf1PairedRunHarness) acquirePostIdentityTurn(ctx context.Context) error {
	select {
	case <-h.postIdentityGate:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for paired post-identity turn: %w", ctx.Err())
	}
}

func (h *cf1PairedRunHarness) releasePostIdentityTurn() { h.postIdentityGate <- struct{}{} }

func (h *cf1PairedRunHarness) registerChildCancel(cancel context.CancelFunc) {
	h.mu.Lock()
	h.cancels = append(h.cancels, cancel)
	h.mu.Unlock()
}

func (h *cf1PairedRunHarness) fail(err error) {
	if err == nil {
		return
	}
	h.failureOnce.Do(func() {
		h.mu.Lock()
		cancels := append([]context.CancelFunc(nil), h.cancels...)
		h.mu.Unlock()
		for _, cancel := range cancels {
			cancel()
		}
	})
}

func newCF1PairedRunHarness(apiURL, identityPath, barrierDir string) (*cf1PairedRunHarness, error) {
	for _, path := range []string{identityPath, barrierDir} {
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("paired Agent Manager evidence path must be absolute: %q", path)
		}
	}
	if err := os.MkdirAll(barrierDir, 0o700); err != nil {
		return nil, fmt.Errorf("create paired probe barrier directory: %w", err)
	}
	if err := os.Chmod(barrierDir, 0o700); err != nil {
		return nil, fmt.Errorf("secure paired probe barrier directory: %w", err)
	}
	postIdentityGate := make(chan struct{}, 1)
	postIdentityGate <- struct{}{}
	return &cf1PairedRunHarness{ready: make(chan struct{}), apiURL: apiURL, identityPath: identityPath, barrierDir: barrierDir, postIdentityGate: postIdentityGate}, nil
}

func (h *cf1PairedRunHarness) configure(ctx context.Context, own cf1PairedRunIdentity) (cf1PairedRunConfig, error) {
	h.mu.Lock()
	for _, prior := range h.arrivals {
		if prior.RunID == own.RunID || prior.BudgetID == own.BudgetID || prior.MandateID == own.MandateID || prior.WalletID == own.WalletID || strings.EqualFold(prior.WalletAddress, own.WalletAddress) {
			h.mu.Unlock()
			return cf1PairedRunConfig{}, errors.New("paired Agent Manager runs reused a run, budget, mandate, wallet, or address")
		}
	}
	h.arrivals = append(h.arrivals, own)
	if len(h.arrivals) > 2 {
		h.mu.Unlock()
		return cf1PairedRunConfig{}, errors.New("paired Agent Manager harness received more than two runs")
	}
	if len(h.arrivals) == 2 {
		h.readyOnce.Do(func() { close(h.ready) })
	}
	h.mu.Unlock()
	select {
	case <-h.ready:
	case <-ctx.Done():
		return cf1PairedRunConfig{}, fmt.Errorf("wait for counterpart Treasury budget wallet: %w", ctx.Err())
	}
	h.mu.Lock()
	arrivals := append([]cf1PairedRunIdentity(nil), h.arrivals...)
	h.mu.Unlock()
	var counterpart cf1PairedRunIdentity
	for _, candidate := range arrivals {
		if candidate.RunID != own.RunID {
			counterpart = candidate
		}
	}
	if counterpart.RunID == "" || own.SessionGeneration <= 0 || counterpart.SessionGeneration <= 0 {
		return cf1PairedRunConfig{}, errors.New("paired run identity omitted an active session generation")
	}
	if err := h.readBackLiveIdentity(ctx, counterpart); err != nil {
		return cf1PairedRunConfig{}, fmt.Errorf("read back counterpart live mandate and wallet before config: %w", err)
	}
	h.controller.Do(func() { go h.fail(h.releaseGates(ctx, arrivals)) })
	return cf1PairedRunConfig{
		ForeignBudgetID: counterpart.BudgetID, ForeignMandateID: counterpart.MandateID,
		CounterpartIdentity: map[string]any{
			"run_id": counterpart.RunID, "budget_id": counterpart.BudgetID, "mandate_id": counterpart.MandateID,
			"wallet_id": counterpart.WalletID, "wallet_address": counterpart.WalletAddress,
			"wallet_run_id": counterpart.WalletRunID, "wallet_budget_id": counterpart.WalletBudgetID,
			"wallet_mandate_id": counterpart.WalletMandateID, "mandate_status": counterpart.MandateStatus,
			"session_generation": counterpart.SessionGeneration,
		},
		PairedProbeBarrierDir: h.barrierDir,
	}, nil
}

func (h *cf1PairedRunHarness) readBackLiveIdentity(ctx context.Context, identity cf1PairedRunIdentity) error {
	mandates, err := mandateconnect.NewMandateAdminServiceClient(http.DefaultClient, h.apiURL).ListMandates(ctx, operatorRequest(&mandatepb.ListMandatesRequest{}))
	if err != nil {
		return err
	}
	var mandate *mandatepb.Mandate
	for _, candidate := range mandates.Msg.GetMandates() {
		if candidate.GetId() == identity.MandateID {
			mandate = candidate
			break
		}
	}
	if mandate == nil || mandate.GetStatus() != mandatepb.MandateStatus_MANDATE_STATUS_LIVE || mandate.GetAgentRunId() != identity.RunID || mandate.GetBudgetId() != identity.BudgetID {
		return fmt.Errorf("counterpart mandate is absent, not LIVE, or not bound to its run/budget: %+v", mandate)
	}
	walletResponse, err := fundingconnect.NewFundingAdminServiceClient(http.DefaultClient, h.apiURL).GetBudgetWallet(ctx, operatorRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: identity.BudgetID, MandateId: identity.MandateID}))
	if err != nil {
		return err
	}
	wallet := walletResponse.Msg.GetWallet()
	if wallet.GetWalletId() != identity.WalletID || !strings.EqualFold(wallet.GetAddress(), identity.WalletAddress) || wallet.GetAgentRunId() != identity.RunID || wallet.GetBudgetId() != identity.BudgetID || wallet.GetMandateId() != identity.MandateID || wallet.GetSessionGeneration() != identity.SessionGeneration {
		return fmt.Errorf("counterpart wallet readback does not match the live run tuple: %+v", wallet)
	}
	return nil
}

func (h *cf1PairedRunHarness) releaseGates(ctx context.Context, identities []cf1PairedRunIdentity) error {
	if len(identities) != 2 {
		return errors.New("paired Agent Manager barrier requires exactly two identities")
	}
	waitJSON := func(path string, target any) error {
		deadline := time.NewTimer(2 * time.Minute)
		defer deadline.Stop()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			data, err := os.ReadFile(path)
			if err == nil {
				return json.Unmarshal(data, target)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-deadline.C:
				return fmt.Errorf("timeout waiting for paired marker %s", path)
			case <-ticker.C:
			}
		}
	}
	writeRelease := func(path string, value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		if _, err = file.Write(data); err != nil {
			_ = file.Close()
			return err
		}
		if err = file.Sync(); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	}
	pair := identities
	markers := make(map[string]cf1PairedProbeMarker, 2)
	for _, identity := range pair {
		var marker cf1PairedProbeMarker
		path := filepath.Join(h.barrierDir, "probe-complete-"+identity.RunID+".json")
		if err := waitJSON(path, &marker); err != nil {
			return err
		}
		if marker.RunID != identity.RunID || marker.ForeignWalletProbeStatus != "permission_denied" || marker.ForeignMandateProbeStatus != "permission_denied" || marker.CounterpartMandateStatus != "LIVE" || marker.CounterpartSessionGeneration <= 0 {
			return fmt.Errorf("paired foreign-probe marker for %s failed identity/status validation: %+v", identity.RunID, marker)
		}
		markers[identity.RunID] = marker
	}
	for _, identity := range pair {
		if err := h.readBackLiveIdentity(ctx, identity); err != nil {
			return fmt.Errorf("read back live identity before paired probe release for %s: %w", identity.RunID, err)
		}
	}
	for _, identity := range pair {
		marker := markers[identity.RunID]
		var counterpart cf1PairedRunIdentity
		for _, candidate := range pair {
			if candidate.RunID == marker.CounterpartRunID {
				counterpart = candidate
			}
		}
		if counterpart.RunID == "" || marker.TargetBudgetID != counterpart.BudgetID || marker.TargetMandateID != counterpart.MandateID || marker.CounterpartSessionGeneration != counterpart.SessionGeneration {
			return fmt.Errorf("paired probe target does not match attested counterpart for %s: %+v", identity.RunID, marker)
		}
		if err := writeRelease(filepath.Join(h.barrierDir, "probe-release-"+identity.RunID+".json"), map[string]any{"run_id": identity.RunID, "counterpart_run_id": marker.CounterpartRunID, "both_mandates_live_during_probes": true}); err != nil {
			return fmt.Errorf("release paired probe for %s: %w", identity.RunID, err)
		}
	}
	for _, identity := range pair {
		var ready struct {
			RunID            string `json:"run_id"`
			CounterpartRunID string `json:"counterpart_run_id"`
			BudgetID         string `json:"budget_id"`
			MandateID        string `json:"mandate_id"`
			WalletID         string `json:"wallet_id"`
			WalletAddress    string `json:"wallet_address"`
		}
		if err := waitJSON(filepath.Join(h.barrierDir, "identity-ready-"+identity.RunID+".json"), &ready); err != nil {
			return err
		}
		if ready.RunID != identity.RunID || ready.BudgetID != identity.BudgetID || ready.MandateID != identity.MandateID || ready.WalletID != identity.WalletID || !strings.EqualFold(ready.WalletAddress, identity.WalletAddress) {
			return fmt.Errorf("paired identity-ready marker does not match Treasury tuple for %s: %+v", identity.RunID, ready)
		}
		for _, candidate := range pair {
			if candidate.RunID != identity.RunID && ready.CounterpartRunID != candidate.RunID {
				return fmt.Errorf("paired identity-ready marker names the wrong counterpart for %s", identity.RunID)
			}
		}
	}
	var evidence cf1PairedIdentityEvidence
	data, err := os.ReadFile(h.identityPath)
	if err != nil || json.Unmarshal(data, &evidence) != nil || len(evidence.Agents) != 2 {
		return fmt.Errorf("shared paired identity evidence is missing, malformed, or not exactly two rows: %v", err)
	}
	for _, identity := range pair {
		var row *struct {
			RunID                     string `json:"run_id"`
			ProfileID                 string `json:"profile_id"`
			BudgetID                  string `json:"budget_id"`
			MandateID                 string `json:"mandate_id"`
			WalletID                  string `json:"wallet_id"`
			WalletAddress             string `json:"wallet_address"`
			OwnAuthorizationID        string `json:"own_authorization_id"`
			OwnSettlementID           string `json:"own_settlement_id"`
			OwnTxHash                 string `json:"own_tx_hash"`
			OwnHeadroomMinor          int64  `json:"own_headroom_minor"`
			ForeignWalletProbeStatus  string `json:"foreign_wallet_probe_status"`
			ForeignMandateProbeStatus string `json:"foreign_mandate_probe_status"`
		}
		for index := range evidence.Agents {
			if evidence.Agents[index].RunID == identity.RunID {
				row = &evidence.Agents[index]
			}
		}
		if row == nil || row.BudgetID != identity.BudgetID || row.MandateID != identity.MandateID || row.WalletID != identity.WalletID || !strings.EqualFold(row.WalletAddress, identity.WalletAddress) || row.OwnAuthorizationID == "" || row.OwnSettlementID == "" || row.OwnTxHash == "" || row.OwnHeadroomMinor <= 0 || row.ForeignWalletProbeStatus != "permission_denied" || row.ForeignMandateProbeStatus != "permission_denied" {
			return fmt.Errorf("shared paired identity row does not match active Treasury tuple for %s", identity.RunID)
		}
		for _, counterpart := range pair {
			if counterpart.RunID == identity.RunID {
				continue
			}
			var counterpartRow *struct {
				RunID                     string `json:"run_id"`
				ProfileID                 string `json:"profile_id"`
				BudgetID                  string `json:"budget_id"`
				MandateID                 string `json:"mandate_id"`
				WalletID                  string `json:"wallet_id"`
				WalletAddress             string `json:"wallet_address"`
				OwnAuthorizationID        string `json:"own_authorization_id"`
				OwnSettlementID           string `json:"own_settlement_id"`
				OwnTxHash                 string `json:"own_tx_hash"`
				OwnHeadroomMinor          int64  `json:"own_headroom_minor"`
				ForeignWalletProbeStatus  string `json:"foreign_wallet_probe_status"`
				ForeignMandateProbeStatus string `json:"foreign_mandate_probe_status"`
			}
			for index := range evidence.Agents {
				if evidence.Agents[index].RunID == counterpart.RunID {
					counterpartRow = &evidence.Agents[index]
				}
			}
			if counterpartRow == nil || counterpartRow.BudgetID != counterpart.BudgetID || counterpartRow.MandateID != counterpart.MandateID || counterpartRow.WalletID != counterpart.WalletID || !strings.EqualFold(counterpartRow.WalletAddress, counterpart.WalletAddress) || counterpartRow.ForeignWalletProbeStatus != "permission_denied" || counterpartRow.ForeignMandateProbeStatus != "permission_denied" {
				return fmt.Errorf("shared paired identity row does not match counterpart Treasury tuple for %s", identity.RunID)
			}
		}
	}
	for _, identity := range pair {
		if err := h.readBackLiveIdentity(ctx, identity); err != nil {
			return fmt.Errorf("read back live identity before paired row release for %s: %w", identity.RunID, err)
		}
	}
	for _, identity := range pair {
		var counterpart string
		for _, other := range pair {
			if other.RunID != identity.RunID {
				counterpart = other.RunID
			}
		}
		if err := writeRelease(filepath.Join(h.barrierDir, "identity-release-"+identity.RunID+".json"), map[string]any{"run_id": identity.RunID, "counterpart_run_id": counterpart, "both_identity_rows_recorded": true, "both_mandates_live_at_identity_gate": true}); err != nil {
			return fmt.Errorf("release paired identity gate for %s: %w", identity.RunID, err)
		}
	}
	return nil
}

type cf1X402ShowcaseConfig struct {
	CatalogURL, PriceID, CallID, Counterparty, InstrumentID, Credential string
	BuyerAddress, FacilitatorAddress                                    string
	OverCapCatalogURL, OverCapPriceID, OverCapCallID, OverCapCredential string
	PrepareSigner                                                       func(context.Context, string, string) (string, error)
	PrepareOverCapSigner                                                func(context.Context, string, string) (string, error)
}

func runCF1AgentManagerCLIForkShowcase(t *testing.T, ctx context.Context, repo, runDir, apiURL, identityAuthorityURL, treasurySource, treasuryName string, client authconnect.TreasuryAdminClient, notificationRequests <-chan *notificationsv1.SendRequest, fork *basefork.Fork, bookID, variant string, hooks cf1ShowcaseHooks) (string, string, string, error) {
	t.Helper()
	nonceValue := func(value string) (*big.Int, bool) {
		parsed, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(value), "0x"), 16)
		return parsed, ok
	}
	instrumentClient := instrumentconnect.NewInstrumentAdminServiceClient(http.DefaultClient, apiURL)
	mandateClient := mandateconnect.NewMandateAdminServiceClient(http.DefaultClient, apiURL)
	budgetAdmin := budgetconnect.NewBudgetAdminServiceClient(http.DefaultClient, apiURL)
	fundingClient := fundingconnect.NewFundingAdminServiceClient(http.DefaultClient, apiURL)
	coordName := "agent-manager-cli"
	if hooks.CF7Only {
		// CF1 and CF7 are distinct ordinary Agent Manager runs. Keep their
		// handoff receipts in separate directories so CF7 cannot observe CF1's
		// earlier run-id/config files before its own runner starts.
		coordName = "cf7-agent-manager-cli"
	}
	coord := filepath.Join(runDir, coordName)
	if err := os.MkdirAll(coord, 0o700); err != nil {
		return "", "", "", err
	}
	budgetID := "cf1-agent-manager-budget-" + variant
	mandateID := "cf1-agent-manager-mandate-" + variant
	isolationBudgetID := "cf1-isolation-budget-" + variant
	isolationMandateID := "cf1-isolation-mandate-" + variant
	isolationRunID := "cf1-isolation-run-" + variant
	expiryMandateID := "cf1-agent-manager-mandate-expiry-" + variant
	authorizationID := "cf1-agent-manager-auth-in-cap-" + variant
	settlementID := "cf1-agent-manager-settlement-in-cap-" + variant
	idempotency := "cf1-agent-manager-idem-in-cap-" + variant
	instrumentID := "cf1-agent-manager-instrument-in-cap-" + variant
	overCapAuthorizationID := "cf1-agent-manager-auth-over-cap-" + variant
	overCapSettlementID := "cf1-agent-manager-settlement-over-cap-" + variant
	overCapIdempotency := "cf1-agent-manager-idem-over-cap-" + variant
	overCapInstrumentID := "cf1-agent-manager-instrument-over-cap-" + variant
	expiryAuthorizationID := "cf1-agent-manager-auth-expired-" + variant
	expirySettlementID := "cf1-agent-manager-settlement-expired-" + variant
	expiryIdempotency := "cf1-agent-manager-idem-expired-" + variant
	x402AuthorizationID := "cf1-agent-manager-auth-x402-" + variant
	x402SettlementID := "cf1-agent-manager-settlement-x402-" + variant
	x402Idempotency := "cf1-agent-manager-idem-x402-" + variant
	x402InstrumentID := "cf1-agent-manager-instrument-x402-" + variant
	cf7OverCapInstrumentID := "cf7-agent-manager-instrument-x402-overcap-" + variant
	expiryInstrumentID := "cf1-agent-manager-instrument-expired-" + variant
	counterparty := strings.ToLower(fork.Accounts[1])
	_, err := budgetAdmin.SetBudgetCaps(ctx, operatorRequest(&budgetpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{
		Id: budgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 1000,
		PeriodicCapMinor: 1000, PerTransactionCapMinor: 100, PeriodSeconds: 3600,
	}}))
	if err != nil {
		return "", "", "", fmt.Errorf("create ordinary-run Treasury budget: %w", err)
	}
	if !hooks.CF7Only {
		_, err = budgetAdmin.SetBudgetCaps(ctx, operatorRequest(&budgetpb.SetBudgetCapsRequest{Budget: &budgetpb.Budget{
			Id: isolationBudgetID, BookId: bookID, Currency: "USD", TotalCapMinor: 100,
			PeriodicCapMinor: 100, PerTransactionCapMinor: 100, PeriodSeconds: 3600,
		}}))
		if err != nil {
			return "", "", "", fmt.Errorf("create second identity isolation budget: %w", err)
		}
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
	amCtx, cancelAM := context.WithCancel(ctx)
	defer cancelAM()
	if hooks.PairedRun != nil {
		hooks.PairedRun.registerChildCancel(cancelAM)
	}
	cf1AuthorityDir, err := os.MkdirTemp("", "cf1-authority-")
	if err != nil {
		return "", "", "", fmt.Errorf("create isolated encrypted Agent Manager key store: %w", err)
	}
	defer os.RemoveAll(cf1AuthorityDir)
	cf1AuthorityPath := filepath.Join(cf1AuthorityDir, "agent-manager-session-keys.enc")
	amTest := exec.CommandContext(amCtx, "go", "test", "./internal/orchestration/integration", "-run", "^TestTreasuryCLIFromOrdinaryRun$", "-count=1", "-v")
	amTest.Dir = amDir
	baseEnv := os.Environ()
	if hooks.PairedRun != nil && hooks.X402 == nil {
		filtered := baseEnv[:0]
		for _, value := range baseEnv {
			if !strings.HasPrefix(value, "TREASURY_CF1_X402_MEASURED_SECOND_CALL=") {
				filtered = append(filtered, value)
			}
		}
		baseEnv = filtered
	}
	amTest.Env = append(baseEnv,
		"TREASURY_CF1_COORD_DIR="+coord,
		"TREASURY_CF1_AUTHORITY_PATH="+cf1AuthorityPath,
		"TREASURY_CF1_BUDGET_ID="+budgetID,
		"TREASURY_CF1_ISOLATION_BUDGET_ID="+isolationBudgetID,
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
		"TREASURY_CF1_EXPIRY_MANDATE_ID="+expiryMandateID,
		"TREASURY_CF1_EXPIRY_AUTHORIZATION_ID="+expiryAuthorizationID,
		"TREASURY_CF1_EXPIRY_SETTLEMENT_ID="+expirySettlementID,
		"TREASURY_CF1_EXPIRY_INSTRUMENT_ID="+expiryInstrumentID,
		"TREASURY_CF1_EXPIRY_IDEMPOTENCY_KEY="+expiryIdempotency,
	)
	if hooks.FundAgentWallet != nil {
		expectedFundingAtoms := hooks.FundingAtomicAmount
		if expectedFundingAtoms == "" {
			expectedFundingAtoms = "2500000"
		}
		amTest.Env = append(amTest.Env, "TREASURY_CF1_WAIT_FOR_FUNDING=1", "TREASURY_CF1_EXPECTED_FUNDING_ATOMS="+expectedFundingAtoms)
	}
	if hooks.CF7Only {
		amTest.Env = append(amTest.Env, "TREASURY_CF7_ONLY=1", "TREASURY_CF7_COORD_DIR="+coord)
	}
	if hooks.PairedRun != nil {
		amTest.Env = append(amTest.Env, "TREASURY_CF1_CROSS_BUDGET_IDENTITIES_PATH="+hooks.PairedRun.identityPath)
	}
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
	if !hooks.CF7Only {
		if _, err := mandateClient.CreateMandate(ctx, operatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
			Id: isolationMandateID, IdempotencyKey: isolationMandateID + "-create", BookId: bookID, BudgetId: isolationBudgetID,
			CapMinor: 100, Currency: "USD", AllowedCounterparties: []string{counterparty},
			AgentSubject: "cf1-isolation-agent", AgentRunId: isolationRunID, ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
		}})); err != nil {
			return "", "", "", fmt.Errorf("create independently bound second-budget mandate: %w", err)
		}
		if _, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
			BudgetId: isolationBudgetID, Network: baseusdc.Network, AgentSubject: "cf1-isolation-agent", AgentRunId: isolationRunID, MandateId: isolationMandateID,
		})); err != nil {
			return "", "", "", fmt.Errorf("provision second run's distinct budget address: %w", err)
		}
	}
	standingNextCharge := timestamppb.New(time.Now().Add(time.Minute))
	mandateCounterparties := []string{counterparty}
	if hooks.X402 != nil && !strings.EqualFold(hooks.X402.Counterparty, counterparty) {
		mandateCounterparties = append(mandateCounterparties, strings.ToLower(hooks.X402.Counterparty))
	}
	createdMandate, err := mandateClient.CreateMandate(ctx, operatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: mandateID, IdempotencyKey: idempotency + "-mandate", BookId: bookID, BudgetId: budgetID,
		CapMinor: 300, Currency: "USD", AllowedCounterparties: mandateCounterparties,
		AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID,
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)), RecurrenceSeconds: 3600, NextChargeAt: standingNextCharge,
	}}))
	if err != nil {
		return "", "", "", fmt.Errorf("create run-bound standing Treasury mandate: %w", err)
	}
	if createdMandate.Msg.GetMandate().GetRecurrenceSeconds() != 3600 || createdMandate.Msg.GetMandate().GetNextChargeAt() == nil || createdMandate.Msg.GetMandate().GetStatus() != mandatepb.MandateStatus_MANDATE_STATUS_LIVE {
		return "", "", "", fmt.Errorf("ordinary Agent Manager run mandate is not live/standing: %+v", createdMandate.Msg.GetMandate())
	}
	writeRehearsalProto(t, filepath.Join(coord, "standing-mandate.json"), createdMandate.Msg.GetMandate())
	expiryDeadline := time.Now().Add(30 * time.Second).UTC()
	createdExpiryMandate, err := mandateClient.CreateMandate(ctx, operatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: expiryMandateID, IdempotencyKey: expiryIdempotency + "-mandate", BookId: bookID, BudgetId: budgetID,
		CapMinor: 10, Currency: "USD", AllowedCounterparties: []string{counterparty},
		AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID, ExpiresAt: timestamppb.New(expiryDeadline),
	}}))
	if err != nil {
		return "", "", "", fmt.Errorf("create short-lived run-bound expiry-test mandate: %w", err)
	}
	if createdExpiryMandate.Msg.GetMandate().GetStatus() != mandatepb.MandateStatus_MANDATE_STATUS_LIVE || !createdExpiryMandate.Msg.GetMandate().GetExpiresAt().AsTime().Equal(expiryDeadline) {
		return "", "", "", fmt.Errorf("short-lived mandate did not preserve its live expiry: %+v", createdExpiryMandate.Msg.GetMandate())
	}
	writeRehearsalJSON(t, filepath.Join(coord, "expired-mandate-created.json"), map[string]any{"mandate_id": expiryMandateID, "expires_at": expiryDeadline})
	walletResponse, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(&fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID, MandateId: mandateID,
	}))
	if err != nil {
		return "", "", "", fmt.Errorf("provision per-run budget address: %w", err)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "ordinary-run-wallet.json"), map[string]any{
		"budget_id": budgetID, "mandate_id": mandateID, "agent_run_id": runIdentity.RunID,
		"wallet_id": walletResponse.Msg.GetWallet().GetWalletId(), "address": walletResponse.Msg.GetWallet().GetAddress(),
		"session_generation": walletResponse.Msg.GetWallet().GetSessionGeneration(),
	})
	if !hooks.CF7Only {
		isolationWallet, getErr := fundingClient.GetBudgetWallet(ctx, operatorRequest(&fundingpb.GetBudgetWalletRequest{BudgetId: isolationBudgetID, MandateId: isolationMandateID}))
		if getErr != nil {
			return "", "", "", fmt.Errorf("read independently bound second budget wallet: %w", getErr)
		}
		if isolationWallet.Msg.GetWallet().GetAddress() == "" || strings.EqualFold(isolationWallet.Msg.GetWallet().GetAddress(), walletResponse.Msg.GetWallet().GetAddress()) {
			return "", "", "", errors.New("independent budget identities did not receive distinct derived addresses")
		}
		writeRehearsalJSON(t, filepath.Join(coord, "cross-budget-wallet-identities.json"), map[string]any{
			"first_budget_id": budgetID, "first_agent_run_id": runIdentity.RunID, "first_address": walletResponse.Msg.GetWallet().GetAddress(),
			"second_budget_id": isolationBudgetID, "second_agent_run_id": isolationRunID, "second_address": isolationWallet.Msg.GetWallet().GetAddress(),
			"addresses_distinct": true, "second_budget_isolated_from_first_run": true,
		})
	}
	if hooks.FundAgentWallet != nil {
		if err := hooks.FundAgentWallet(ctx, budgetID, walletResponse.Msg.Wallet.WalletId, walletResponse.Msg.Wallet.Address, mandateID, runIdentity.RunID); err != nil {
			return "", "", "", fmt.Errorf("fund run-bound budget wallet through the configured receipt path: %w", err)
		}
	} else if _, err := fork.Mint(ctx, walletResponse.Msg.Wallet.Address, "2000000"); err != nil {
		return "", "", "", fmt.Errorf("fund only the local-fork run wallet: %w", err)
	}
	if hooks.X402 != nil && hooks.X402.PrepareSigner != nil {
		credentialValue, signerErr := hooks.X402.PrepareSigner(ctx, walletResponse.Msg.GetWallet().GetWalletId(), walletResponse.Msg.GetWallet().GetAddress())
		if signerErr != nil {
			return "", "", "", fmt.Errorf("prepare x402 signer for the provisioned research wallet: %w", signerErr)
		}
		hooks.X402.Credential = credentialValue
		hooks.X402.BuyerAddress = walletResponse.Msg.GetWallet().GetAddress()
	}
	if hooks.CF7Only && hooks.X402 != nil && hooks.X402.PrepareOverCapSigner != nil {
		credentialValue, signerErr := hooks.X402.PrepareOverCapSigner(ctx, walletResponse.Msg.GetWallet().GetWalletId(), walletResponse.Msg.GetWallet().GetAddress())
		if signerErr != nil {
			return "", "", "", fmt.Errorf("prepare CF7 over-cap signer for the provisioned research wallet: %w", signerErr)
		}
		hooks.X402.OverCapCredential = credentialValue
	}
	var pairedConfig *cf1PairedRunConfig
	if hooks.PairedRun != nil {
		identity := cf1PairedRunIdentity{
			RunID: runIdentity.RunID, BudgetID: budgetID, MandateID: mandateID,
			WalletID: walletResponse.Msg.GetWallet().GetWalletId(), WalletAddress: walletResponse.Msg.GetWallet().GetAddress(),
			WalletRunID: walletResponse.Msg.GetWallet().GetAgentRunId(), WalletBudgetID: walletResponse.Msg.GetWallet().GetBudgetId(),
			WalletMandateID: walletResponse.Msg.GetWallet().GetMandateId(), MandateStatus: "LIVE",
			SessionGeneration: walletResponse.Msg.GetWallet().GetSessionGeneration(), CoordDir: coord,
		}
		pairConfig, pairErr := hooks.PairedRun.configure(ctx, identity)
		if pairErr != nil {
			return "", "", "", fmt.Errorf("configure reciprocal live Treasury identity: %w", pairErr)
		}
		pairedConfig = &pairConfig
	}
	if err := fork.Call(ctx, "anvil_setBalance", []any{walletResponse.Msg.Wallet.Address, "0xde0b6b3a7640000"}, nil); err != nil {
		return "", "", "", fmt.Errorf("fund local-fork gas: %w", err)
	}
	quoteAdapter, err := nativeusdc.New(nil, nil, fork.URL, "local-fork")
	if err != nil {
		return "", "", "", err
	}
	credentialPath := hooks.TreasuryCredentialPath
	if credentialPath == "" {
		credentialPath = filepath.Join(runDir, "custody", "synthetic-custody.enc")
	}
	credentialAuthority, err := credentialauthority.OpenEncryptedFile(credentialPath, "CF2 local fork synthetic custody; never value-bearing")
	if err != nil {
		return "", "", "", fmt.Errorf("open local Treasury custody fixture: %w", err)
	}
	registerRunInstrument := func(instrumentMandateID, id string, amountMinor int64) error {
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
		if _, registerErr := instrumentClient.RegisterInstrument(ctx, operatorRequest(&instrumentpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
			Id: id, MandateId: instrumentMandateID, Rail: "base-usdc", CredentialReference: credentialReference, Counterparty: counterparty,
		}})); registerErr != nil {
			return fmt.Errorf("register local run rail instrument %s: %w", id, registerErr)
		}
		return nil
	}
	if err := registerRunInstrument(mandateID, instrumentID, 30); err != nil {
		return "", "", "", err
	}
	if err := registerRunInstrument(mandateID, overCapInstrumentID, 110); err != nil {
		return "", "", "", err
	}
	if err := registerRunInstrument(expiryMandateID, expiryInstrumentID, 10); err != nil {
		return "", "", "", err
	}
	if hooks.X402 != nil {
		x := hooks.X402
		if x.CatalogURL == "" || x.PriceID == "" || x.CallID == "" || x.Counterparty == "" || x.InstrumentID == "" || x.Credential == "" || x.BuyerAddress == "" {
			return "", "", "", errors.New("CF4 Offer Desk run configuration is incomplete")
		}
		var signerBinding struct {
			EndpointURL string `json:"endpoint_url"`
			Account     string `json:"account"`
		}
		if err := json.Unmarshal([]byte(x.Credential), &signerBinding); err != nil || !strings.EqualFold(signerBinding.Account, walletResponse.Msg.GetWallet().GetAddress()) || !strings.EqualFold(signerBinding.Account, x.BuyerAddress) {
			return "", "", "", errors.New("CF4 x402 signer account must equal the provisioned mandate research-wallet address")
		}
		endpoint, err := url.ParseRequestURI(signerBinding.EndpointURL)
		if err != nil || endpoint.Hostname() == "" || !strings.EqualFold(endpoint.Hostname(), x.Counterparty) {
			return "", "", "", errors.New("CF4 x402 paid endpoint host must equal its mandate counterparty")
		}
		credentialIdentity, parseErr := credentialauthority.ParseIdentity("vrooli/treasury/" + x402InstrumentID)
		if parseErr != nil {
			return "", "", "", parseErr
		}
		if putErr := credentialAuthority.Put(credentialIdentity, "value", x.Credential); putErr != nil {
			return "", "", "", fmt.Errorf("retain local x402 credential only in Credential Authority: %w", putErr)
		}
		if _, registerErr := instrumentClient.RegisterInstrument(ctx, operatorRequest(&instrumentpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
			Id: x402InstrumentID, MandateId: mandateID, Rail: "x402", CredentialReference: "vrooli/treasury/" + x402InstrumentID, Counterparty: x.Counterparty,
		}})); registerErr != nil {
			return "", "", "", fmt.Errorf("register per-call Offer Desk x402 instrument: %w", registerErr)
		}
		if hooks.CF7Only {
			if x.OverCapCatalogURL == "" || x.OverCapPriceID == "" || x.OverCapCallID == "" || x.OverCapCredential == "" {
				return "", "", "", errors.New("CF7 over-cap Offer Desk quote or credential is incomplete")
			}
			var overCapSigner struct {
				EndpointURL string `json:"endpoint_url"`
				Account     string `json:"account"`
			}
			if err := json.Unmarshal([]byte(x.OverCapCredential), &overCapSigner); err != nil || !strings.EqualFold(overCapSigner.Account, walletResponse.Msg.GetWallet().GetAddress()) || !strings.EqualFold(overCapSigner.Account, x.BuyerAddress) {
				return "", "", "", errors.New("CF7 over-cap signer account must equal the provisioned mandate research-wallet address")
			}
			overCapEndpoint, err := url.ParseRequestURI(overCapSigner.EndpointURL)
			if err != nil || overCapEndpoint.Hostname() == "" || !strings.EqualFold(overCapEndpoint.Hostname(), x.Counterparty) || overCapSigner.EndpointURL != x.OverCapCatalogURL {
				return "", "", "", errors.New("CF7 over-cap signer endpoint must match its separate priced Offer Desk resource")
			}
			overCapIdentity, err := credentialauthority.ParseIdentity("vrooli/treasury/" + cf7OverCapInstrumentID)
			if err != nil {
				return "", "", "", err
			}
			if err := credentialAuthority.Put(overCapIdentity, "value", x.OverCapCredential); err != nil {
				return "", "", "", fmt.Errorf("retain CF7 over-cap credential only in Credential Authority: %w", err)
			}
			if _, err := instrumentClient.RegisterInstrument(ctx, operatorRequest(&instrumentpb.RegisterInstrumentRequest{Instrument: &instrumentpb.Instrument{
				Id: cf7OverCapInstrumentID, MandateId: mandateID, Rail: "x402", CredentialReference: "vrooli/treasury/" + cf7OverCapInstrumentID, Counterparty: x.Counterparty,
			}})); err != nil {
				return "", "", "", fmt.Errorf("register CF7 over-cap x402 instrument: %w", err)
			}
		}
	}
	config := map[string]any{
		"api_base_url": apiURL, "cli_binary": cliBinary, "budget_id": budgetID,
		"mandate_id": mandateID, "counterparty": counterparty,
		"in_cap_instrument_id": instrumentID, "in_cap_amount_minor": 30,
		"in_cap_authorization_id": authorizationID, "in_cap_settlement_id": settlementID, "in_cap_idempotency_key": idempotency,
		"over_cap_instrument_id": overCapInstrumentID, "over_cap_amount_minor": 110,
		"over_cap_authorization_id": overCapAuthorizationID, "over_cap_settlement_id": overCapSettlementID, "over_cap_idempotency_key": overCapIdempotency,
		"expiry_mandate_id": expiryMandateID, "expiry_authorization_id": expiryAuthorizationID,
		"expiry_settlement_id": expirySettlementID, "expiry_instrument_id": expiryInstrumentID, "expiry_idempotency_key": expiryIdempotency,
	}
	if !hooks.CF7Only {
		config["isolation_budget_id"] = isolationBudgetID
		config["isolation_mandate_id"] = isolationMandateID
	}
	if pairedConfig != nil {
		config["foreign_budget_id"] = pairedConfig.ForeignBudgetID
		config["foreign_mandate_id"] = pairedConfig.ForeignMandateID
		config["counterpart_identity"] = pairedConfig.CounterpartIdentity
		config["paired_probe_barrier_dir"] = pairedConfig.PairedProbeBarrierDir
	}
	if hooks.X402 != nil {
		x := hooks.X402
		config["x402_catalog_url"] = x.CatalogURL
		config["x402_price_id"] = x.PriceID
		config["x402_call_id"] = x.CallID
		config["x402_counterparty"] = x.Counterparty
		config["x402_instrument_id"] = x402InstrumentID
		config["x402_expected_amount_minor"] = int64(1)
		config["x402_authorization_id"] = x402AuthorizationID
		config["x402_settlement_id"] = x402SettlementID
		config["x402_idempotency_key"] = x402Idempotency
		if hooks.CF7Only {
			calls := make([]map[string]any, 0, 4)
			for i := 1; i <= 3; i++ {
				calls = append(calls, map[string]any{
					"price_id": x.PriceID, "call_id": fmt.Sprintf("%s-%d", x.CallID, i), "catalog_url": x.CatalogURL,
					"counterparty": x.Counterparty, "instrument_id": x402InstrumentID,
					"authorization_id": fmt.Sprintf("%s-%d", x402AuthorizationID, i),
					"settlement_id":    fmt.Sprintf("%s-%d", x402SettlementID, i),
					"idempotency_key":  fmt.Sprintf("%s-%d", x402Idempotency, i), "amount_minor": int64(1),
				})
			}
			calls = append(calls, map[string]any{
				"price_id": x.OverCapPriceID, "call_id": x.OverCapCallID, "catalog_url": x.OverCapCatalogURL,
				"counterparty": x.Counterparty, "instrument_id": cf7OverCapInstrumentID,
				"authorization_id": x402AuthorizationID + "-overcap", "settlement_id": x402SettlementID + "-overcap",
				"idempotency_key": x402Idempotency + "-overcap", "amount_minor": int64(110),
			})
			config["x402_calls"] = calls
		}
	}
	writeRehearsalJSON(t, filepath.Join(coord, "treasury-config.json"), config)
	if hooks.CF7Only {
		return runCF7TreasuryCoordinator(t, ctx, coord, childDone, amOutputPath, fork, walletResponse.Msg.GetWallet().GetAddress(), client, bookID, hooks, config)
	}
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
	var nonceAfterInCap string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{walletResponse.Msg.Wallet.Address, "latest"}, &nonceAfterInCap); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run nonce after in-cap debit: %w", err)
	}
	preInCap, preInCapOK := nonceValue(nonceBeforeAlteredInstrument)
	postInCap, postInCapOK := nonceValue(nonceAfterInCap)
	if !preInCapOK || !postInCapOK || new(big.Int).Sub(postInCap, preInCap).Cmp(big.NewInt(1)) != 0 {
		return "", "", "", fmt.Errorf("ordinary-run in-cap debit count was not exactly one: nonce %s -> %s", nonceBeforeAlteredInstrument, nonceAfterInCap)
	}
	inCapSettlement, err := readCLISettlement(coord, "settlement-report-in-cap.log")
	if err != nil {
		return "", "", "", err
	}
	if err := requireCLILocalSettlement(inCapSettlement, settlementID); err != nil {
		return "", "", "", fmt.Errorf("validate ordinary-run in-cap settlement: %w", err)
	}
	initialSessionGeneration := walletResponse.Msg.GetWallet().GetSessionGeneration()
	if initialSessionGeneration <= 0 {
		return "", "", "", fmt.Errorf("ordinary Agent Manager run was provisioned without an active mandate session generation: %d", initialSessionGeneration)
	}
	rotationRequest := &fundingpb.ProvisionBudgetWalletRequest{
		BudgetId: budgetID, Network: baseusdc.Network, AgentSubject: "cf1-local-operator", AgentRunId: runIdentity.RunID,
		MandateId: mandateID, RotateSession: true, RotationIdempotencyKey: idempotency + "-session-rotation",
	}
	rotatedWalletResponse, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(rotationRequest))
	if err != nil {
		return "", "", "", fmt.Errorf("rotate mandate session while ordinary Agent Manager run is active: %w", err)
	}
	rotationReplay, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(proto.Clone(rotationRequest).(*fundingpb.ProvisionBudgetWalletRequest)))
	if err != nil {
		return "", "", "", fmt.Errorf("replay ordinary-run mandate session rotation: %w", err)
	}
	rotatedWallet := rotatedWalletResponse.Msg.GetWallet()
	if rotationReplay.Msg.GetWallet().GetSessionGeneration() != rotatedWallet.GetSessionGeneration() || rotationReplay.Msg.GetWallet().GetAddress() != rotatedWallet.GetAddress() {
		return "", "", "", errors.New("identical ordinary-run rotation retry changed the generation or address")
	}
	writeRehearsalProto(t, filepath.Join(coord, "mandate-session-rotation-replay.json"), rotationReplay.Msg)
	changedRotation := proto.Clone(rotationRequest).(*fundingpb.ProvisionBudgetWalletRequest)
	changedRotation.AgentRunId += "-changed"
	_, changedRotationErr := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(changedRotation))
	if connect.CodeOf(changedRotationErr) != connect.CodeAlreadyExists {
		return "", "", "", fmt.Errorf("changed run payload reused a rotation key without conflict: %v", changedRotationErr)
	}
	rotationAfterConflict, err := fundingClient.ProvisionBudgetWallet(ctx, operatorRequest(proto.Clone(rotationRequest).(*fundingpb.ProvisionBudgetWalletRequest)))
	if err != nil || rotationAfterConflict.Msg.GetWallet().GetSessionGeneration() != rotatedWallet.GetSessionGeneration() {
		return "", "", "", fmt.Errorf("changed-input conflict altered the retained rotation: response=%v err=%v", rotationAfterConflict, err)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "mandate-session-rotation-changed-input.json"), map[string]any{
		"same_idempotency_key": true, "changed_run_rejected": true, "replay_generation_unchanged": true,
	})
	if rotatedWallet.GetSessionGeneration() != initialSessionGeneration+1 || rotatedWallet.GetAddress() != walletResponse.Msg.GetWallet().GetAddress() {
		return "", "", "", fmt.Errorf("mandate session rotation changed address or failed to advance generation: old=%d new=%d address=%q", initialSessionGeneration, rotatedWallet.GetSessionGeneration(), rotatedWallet.GetAddress())
	}
	writeRehearsalJSON(t, filepath.Join(coord, "mandate-session-rotated.json"), map[string]any{
		"mandate_id": mandateID, "address": rotatedWallet.GetAddress(),
		"old_generation":         strconv.FormatInt(initialSessionGeneration, 10),
		"new_generation":         strconv.FormatInt(rotatedWallet.GetSessionGeneration(), 10),
		"old_generation_retired": true, "same_derived_budget_address": true,
	})
	if hooks.PairedRun != nil {
		if err := hooks.PairedRun.acquirePostIdentityTurn(ctx); err != nil {
			return "", "", "", err
		}
		defer hooks.PairedRun.releasePostIdentityTurn()
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
	queued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	if err != nil || len(queued.Msg.GetApprovals()) != 1 || queued.Msg.GetApprovals()[0].GetAuthorizationId() != overCapAuthorizationID {
		return "", "", "", fmt.Errorf("ordinary-run over-cap approval was not queued for its authorization: approvals=%v err=%v", queued, err)
	}
	var notification *notificationsv1.SendRequest
	approvalID := queued.Msg.GetApprovals()[0].GetId()
	if hooks.AnswerApprovalInHub != nil {
		notification, err = hooks.AnswerApprovalInHub(ctx, approvalID)
		if err != nil {
			return "", "", "", fmt.Errorf("answer over-cap decision through Notification Hub: %w", err)
		}
		approved, waitErr := waitForCF1ApprovalStatus(ctx, client, bookID, approvalID, treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED)
		if waitErr != nil {
			return "", "", "", fmt.Errorf("wait for Treasury to correlate Notification Hub answer: %w", waitErr)
		}
		writeRehearsalProto(t, filepath.Join(coord, "over-cap-approval-approved.json"), approved)
	} else {
		approved, resolveErr := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: approvalID, Resolution: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
		if resolveErr != nil || approved.Msg.GetApproval().GetStatus() != treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED {
			return "", "", "", fmt.Errorf("approve ordinary-run over-cap request: approval=%v err=%v", approved, resolveErr)
		}
		writeRehearsalProto(t, filepath.Join(coord, "over-cap-approval-approved.json"), approved.Msg.GetApproval())
		if hooks.AwaitApprovalNotice != nil {
			notification, err = hooks.AwaitApprovalNotice(ctx, approvalID)
			if err != nil {
				return "", "", "", fmt.Errorf("wait for over-cap decision notification: %w", err)
			}
		} else {
			select {
			case notification = <-notificationRequests:
			case <-ctx.Done():
				return "", "", "", fmt.Errorf("over-cap decision notification was not delivered: %w", ctx.Err())
			}
		}
	}
	if !strings.Contains(notification.GetBody(), "110 USD minor units") || !strings.Contains(notification.GetBody(), queued.Msg.GetApprovals()[0].GetId()) {
		return "", "", "", fmt.Errorf("over-cap approval notification did not match the approved AM spend: %s", notification.GetBody())
	}
	writeRehearsalProto(t, filepath.Join(coord, "over-cap-approval-notification.json"), notification)
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
	var nonceAfterOverCap string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{walletResponse.Msg.Wallet.Address, "latest"}, &nonceAfterOverCap); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run nonce after approved over-cap debit: %w", err)
	}
	postInCapValue, postInCapValueOK := nonceValue(nonceAfterInCap)
	postOverCapValue, postOverCapOK := nonceValue(nonceAfterOverCap)
	if !postInCapValueOK || !postOverCapOK || new(big.Int).Sub(postOverCapValue, postInCapValue).Cmp(big.NewInt(1)) != 0 {
		return "", "", "", fmt.Errorf("approved over-cap debit count was not exactly one: nonce %s -> %s", nonceAfterInCap, nonceAfterOverCap)
	}
	// Pin the exact JSON-RPC receipt bytes before Treasury replacement. The
	// retry below must return this same chain receipt, not merely repeat an ID.
	overCapReceiptBeforeRestart, err := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{overCapSettlement.GetSettlement().GetExternalId()})
	if err != nil {
		return "", "", "", fmt.Errorf("capture approved ordinary-run receipt before Treasury restart: %w", err)
	}
	if _, err := validateCF1RPCReceipt(overCapReceiptBeforeRestart, overCapSettlement.GetSettlement().GetExternalId(), walletResponse.Msg.GetWallet().GetAddress(), counterparty, 110); err != nil {
		return "", "", "", fmt.Errorf("validate approved ordinary-run receipt before Treasury restart: %w", err)
	}
	overCapReceiptBeforeDigest := sha256.Sum256(overCapReceiptBeforeRestart)
	// The ordinary AM run stays active while Treasury is restarted; its real CLI
	// retry runs against the replacement API after durable state is recovered.
	restarted := lifecycleRehearsal(t, ctx, runDir, "treasury-restart-during-agent-run", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
	if newURL := rehearsalURL(t, restarted); newURL == "" || newURL != apiURL {
		return "", "", "", errors.New("Treasury restart returned no API URL")
	}
	// The CLI is pinned to the same isolated variant address; rebind outer clients
	// through that address after the owner confirms its replacement process.
	client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
	// Prove the local identity authority fixture is still reachable before the
	// active Agent Manager run retries through the restarted Treasury process.
	identityCheck, identityCheckErr := http.Post(identityAuthorityURL+"/api/v1/identity/verify", "application/json", strings.NewReader(`{"token":"cf2-local-agent"}`))
	if identityCheckErr != nil {
		return "", "", "", fmt.Errorf("local identity fixture unavailable after Treasury restart: %w", identityCheckErr)
	}
	_ = identityCheck.Body.Close()
	if err := os.WriteFile(filepath.Join(coord, "over-cap-retry-release"), []byte("retry same settlement after Treasury restart"), 0o600); err != nil {
		return "", "", "", err
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "over-cap-retry-after-restart.json"), nil, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for ordinary-run exact retry after restart: %w", err)
	}
	var nonceAfterOverCapRetry string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{walletResponse.Msg.Wallet.Address, "latest"}, &nonceAfterOverCapRetry); err != nil {
		return "", "", "", fmt.Errorf("read ordinary-run nonce after restart retry: %w", err)
	}
	if nonceAfterOverCapRetry != nonceAfterOverCap {
		return "", "", "", fmt.Errorf("approved over-cap retry debited again: nonce %s -> %s", nonceAfterOverCap, nonceAfterOverCapRetry)
	}
	overCapRetry, err := readCLISettlement(coord, "settlement-report-over-cap-after-restart.log")
	if err != nil {
		return "", "", "", err
	}
	if overCapRetry.GetSettlement().GetId() != overCapSettlementID || overCapRetry.GetSettlement().GetExternalId() != overCapSettlement.GetSettlement().GetExternalId() || overCapRetry.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED {
		return "", "", "", fmt.Errorf("ordinary-run retry after restart changed or repeated settlement: %+v", overCapRetry.GetSettlement())
	}
	overCapReceiptAfterRestart, err := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{overCapRetry.GetSettlement().GetExternalId()})
	if err != nil {
		return "", "", "", fmt.Errorf("capture approved ordinary-run receipt after Treasury restart: %w", err)
	}
	if _, err := validateCF1RPCReceipt(overCapReceiptAfterRestart, overCapRetry.GetSettlement().GetExternalId(), walletResponse.Msg.GetWallet().GetAddress(), counterparty, 110); err != nil {
		return "", "", "", fmt.Errorf("validate approved ordinary-run receipt after Treasury restart: %w", err)
	}
	overCapReceiptAfterDigest := sha256.Sum256(overCapReceiptAfterRestart)
	if !bytes.Equal(overCapReceiptBeforeRestart, overCapReceiptAfterRestart) {
		return "", "", "", errors.New("ordinary-run retry after Treasury restart changed raw receipt bytes for the original transaction")
	}
	writeRehearsalJSON(t, filepath.Join(coord, "over-cap-raw-receipt-restart-continuity.json"), map[string]any{
		"transaction_hash":            overCapSettlement.GetSettlement().GetExternalId(),
		"before_restart_sha256":       fmt.Sprintf("%x", overCapReceiptBeforeDigest),
		"after_restart_sha256":        fmt.Sprintf("%x", overCapReceiptAfterDigest),
		"raw_bytes_equal":             true,
		"debit_count_before_retry":    1,
		"debit_count_after_retry":     1,
		"sender_nonce_after_in_cap":   nonceAfterInCap,
		"sender_nonce_after_over_cap": nonceAfterOverCap,
		"sender_nonce_after_retry":    nonceAfterOverCapRetry,
		"raw_receipt_before_restart":  json.RawMessage(overCapReceiptBeforeRestart),
		"raw_receipt_after_restart":   json.RawMessage(overCapReceiptAfterRestart),
	})
	if hooks.X402 != nil {
		if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "x402-authorization-ready.json"), nil, childDone); err != nil {
			return "", "", "", fmt.Errorf("wait for mandate-authorized Offer Desk call before release: %w", err)
		}
		if err := fork.Call(ctx, "evm_setNextBlockTimestamp", []any{time.Now().Unix()}, nil); err != nil {
			return "", "", "", fmt.Errorf("align only the local fork authorization clock before the x402 call: %w", err)
		}
		if err := fork.Call(ctx, "evm_mine", []any{}, nil); err != nil {
			return "", "", "", fmt.Errorf("mine the local fork authorization-clock update: %w", err)
		}
		buyerBeforeX402 := cf1WalletUSDCBalance(t, ctx, fork, walletResponse.Msg.GetWallet().GetAddress())
		if err := os.WriteFile(filepath.Join(coord, "x402-wallet-balance-before.json"), []byte(buyerBeforeX402.String()), 0o600); err != nil {
			return "", "", "", err
		}
		if err := os.WriteFile(filepath.Join(coord, "x402-call-release"), []byte("payer matches the run-bound mandate wallet"), 0o600); err != nil {
			return "", "", "", err
		}
		var unknown struct {
			AuthorizationID string `json:"authorization_id"`
			SettlementID    string `json:"settlement_id"`
			RetryKey        string `json:"retry_idempotency_key"`
			RetainedID      string `json:"retained_external_id"`
		}
		if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "x402-outcome-unknown.json"), &unknown, childDone); err != nil {
			return "", "", "", fmt.Errorf("wait for deliberately lost Offer Desk response to become unknown: %w", err)
		}
		if unknown.AuthorizationID != x402AuthorizationID || unknown.SettlementID != x402SettlementID || unknown.RetryKey != x402Idempotency+"-settlement" || unknown.RetainedID == "" {
			return "", "", "", fmt.Errorf("CF4 unknown outcome did not retain the exact authorized call input: %+v", unknown)
		}
		restarted := lifecycleRehearsal(t, ctx, runDir, "treasury-restart-during-x402-run", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
		if rehearsalURL(t, restarted) != apiURL {
			return "", "", "", errors.New("Treasury restart before exact CF4 x402 retry changed its isolated API URL")
		}
		client = authconnect.NewTreasuryAdminClient(http.DefaultClient, apiURL)
		if err := os.WriteFile(filepath.Join(coord, "x402-retry-release"), []byte("retry exact Offer Desk payment after Treasury restart"), 0o600); err != nil {
			return "", "", "", err
		}
		var settled struct {
			AuthorizationID string `json:"authorization_id"`
			SettlementID    string `json:"settlement_id"`
			SameAuth        bool   `json:"same_authorization"`
			SameSettlement  bool   `json:"same_settlement_id"`
			SameIdempotency bool   `json:"same_idempotency_key"`
			TransactionHash string `json:"transaction_hash"`
		}
		if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "x402-settled-after-restart.json"), &settled, childDone); err != nil {
			return "", "", "", fmt.Errorf("wait for exact CF4 x402 recovery after restart: %w", err)
		}
		if settled.AuthorizationID != x402AuthorizationID || settled.SettlementID != x402SettlementID || !settled.SameAuth || !settled.SameSettlement || !settled.SameIdempotency || !strings.HasPrefix(strings.ToLower(settled.TransactionHash), "0x") {
			return "", "", "", fmt.Errorf("CF4 retry lost its exact spend binding or transaction hash: %+v", settled)
		}
		buyerAfterX402 := cf1WalletUSDCBalance(t, ctx, fork, walletResponse.Msg.GetWallet().GetAddress())
		if new(big.Int).Sub(buyerBeforeX402, buyerAfterX402).Cmp(big.NewInt(10_000)) != 0 {
			return "", "", "", fmt.Errorf("CF4 x402 call did not debit the mandate wallet exactly once: before=%s after=%s", buyerBeforeX402, buyerAfterX402)
		}
		var chainTransaction struct {
			Hash  string `json:"hash"`
			From  string `json:"from"`
			Nonce string `json:"nonce"`
		}
		if err := fork.Call(ctx, "eth_getTransactionByHash", []any{settled.TransactionHash}, &chainTransaction); err != nil {
			return "", "", "", fmt.Errorf("read settled facilitator transaction and nonce: %w", err)
		}
		if !strings.EqualFold(chainTransaction.Hash, settled.TransactionHash) || chainTransaction.From == "" || chainTransaction.Nonce == "" {
			return "", "", "", fmt.Errorf("CF4 settled transaction has no chain nonce evidence: %+v", chainTransaction)
		}
		var chainReceipt struct {
			Status string `json:"status"`
		}
		if err := fork.Call(ctx, "eth_getTransactionReceipt", []any{settled.TransactionHash}, &chainReceipt); err != nil || chainReceipt.Status != "0x1" {
			return "", "", "", fmt.Errorf("CF4 settled transaction receipt is absent or failed: status=%q err=%v", chainReceipt.Status, err)
		}
		writeRehearsalJSON(t, filepath.Join(coord, "x402-restart-reconciliation.json"), map[string]any{
			"authorization_id": settled.AuthorizationID, "settlement_id": settled.SettlementID,
			"payment_digest_before_restart": unknown.RetainedID, "transaction_hash_after_restart": settled.TransactionHash,
			"same_authorization": settled.SameAuth, "same_settlement": settled.SameSettlement, "same_idempotency": settled.SameIdempotency,
			"payer_wallet_address": walletResponse.Msg.GetWallet().GetAddress(), "payer_atoms_before": buyerBeforeX402.String(), "payer_atoms_after": buyerAfterX402.String(),
			"transaction_sender": chainTransaction.From, "transaction_nonce": chainTransaction.Nonce, "receipt_status": chainReceipt.Status,
		})
	}
	if _, err = budgetAdmin.FreezeBudget(ctx, operatorRequest(&budgetpb.FreezeBudgetRequest{BudgetId: budgetID})); err != nil {
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
	var nonceBeforeExpiryAttempt string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{frozenWallet.Wallet.Address, "latest"}, &nonceBeforeExpiryAttempt); err != nil {
		return "", "", "", fmt.Errorf("read run wallet nonce before mandate-expiry denial: %w", err)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "frozen-no-dispatch.json"), map[string]string{
		"authorization_id": frozenProposal.GetAuthorization().GetId(), "violated_constraint": frozenProposal.GetAuthorization().GetViolatedConstraint(),
		"nonce_before": nonceBeforeFrozenRequest, "nonce_after": nonceAfterFrozenRequest, "dispatch": "none",
	})
	unfrozen, err := budgetAdmin.UnfreezeBudget(ctx, operatorRequest(&budgetpb.UnfreezeBudgetRequest{BudgetId: budgetID}))
	if err != nil || unfrozen.Msg.GetBudget().GetId() != budgetID || unfrozen.Msg.GetBudget().GetFrozen() {
		return "", "", "", fmt.Errorf("release local budget freeze before independent expiry test: budget=%v err=%v", unfrozen, err)
	}
	writeRehearsalProto(t, filepath.Join(coord, "budget-unfrozen-for-expiry.json"), unfrozen.Msg.GetBudget())
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
	var expiryDecision struct {
		MandateID          string `json:"mandate_id"`
		AuthorizationID    string `json:"authorization_id"`
		Verdict            string `json:"verdict"`
		ViolatedConstraint string `json:"violated_constraint"`
	}
	readRehearsalJSON(t, filepath.Join(coord, "expired-mandate-decision.json"), &expiryDecision)
	if expiryDecision.MandateID != expiryMandateID || expiryDecision.AuthorizationID != expiryAuthorizationID || expiryDecision.Verdict != "AUTHORIZATION_VERDICT_REFUSED" || expiryDecision.ViolatedConstraint != "mandate_live" {
		return "", "", "", fmt.Errorf("expired mandate did not fail closed in the ordinary Agent Manager run: %+v", expiryDecision)
	}
	var nonceAfterExpiryAttempt string
	if err := fork.Call(ctx, "eth_getTransactionCount", []any{frozenWallet.Wallet.Address, "latest"}, &nonceAfterExpiryAttempt); err != nil {
		return "", "", "", fmt.Errorf("read run wallet nonce after mandate-expiry denial: %w", err)
	}
	if nonceBeforeExpiryAttempt != nonceAfterExpiryAttempt {
		return "", "", "", fmt.Errorf("expired-mandate attempt dispatched a transaction: nonce %s -> %s", nonceBeforeExpiryAttempt, nonceAfterExpiryAttempt)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "expired-mandate-no-dispatch.json"), map[string]any{
		"mandate_id": expiryMandateID, "authorization_id": expiryAuthorizationID,
		"violated_constraint": expiryDecision.ViolatedConstraint,
		"nonce_before":        nonceBeforeExpiryAttempt, "nonce_after": nonceAfterExpiryAttempt, "dispatch": "none",
	})
	receiptContinuity := make(map[string]map[string]string)
	writeCLIReceipt := func(label string, settlement *authpb.ReportOutcomeResponse, amountMinor int64) error {
		rawReceipt, callErr := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{settlement.GetSettlement().GetExternalId()})
		if callErr != nil {
			return fmt.Errorf("read %s CLI settlement receipt: %w", label, callErr)
		}
		receipt, receiptErr := validateCF1RPCReceipt(rawReceipt, settlement.GetSettlement().GetExternalId(), walletResponse.Msg.GetWallet().GetAddress(), counterparty, amountMinor)
		if receiptErr != nil {
			return fmt.Errorf("validate %s CLI settlement receipt: %w", label, receiptErr)
		}
		expectedAtoms := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(10_000))
		rawDigest := sha256.Sum256(rawReceipt)
		rawReceiptSHA256 := fmt.Sprintf("%x", rawDigest)
		receiptContinuity[label] = map[string]string{
			"transaction_hash":   receipt.TransactionHash,
			"settlement_id":      settlement.GetSettlement().GetId(),
			"raw_receipt_sha256": rawReceiptSHA256,
		}
		writeRehearsalJSON(t, filepath.Join(coord, label+"-chain-receipt.json"), map[string]any{
			"transaction_hash": receipt.TransactionHash, "block_hash": receipt.BlockHash,
			"block_number": receipt.BlockNumber, "status": receipt.Status,
			"chain": "local-base-fork", "run_id": runIdentity.RunID, "budget_id": budgetID,
			"mandate_id": mandateID, "wallet_address": walletResponse.Msg.GetWallet().GetAddress(),
			"authorization_id": settlement.GetSettlement().GetAuthorizationId(),
			"settlement_id":    settlement.GetSettlement().GetId(), "counterparty": counterparty,
			"amount_minor": amountMinor, "transfer_atoms": expectedAtoms.String(),
			"rpc_result": json.RawMessage(rawReceipt), "rpc_result_sha256": rawReceiptSHA256,
		})
		return nil
	}
	if err := writeCLIReceipt("in-cap", inCapSettlement, 30); err != nil {
		return "", "", "", err
	}
	if err := writeCLIReceipt("over-cap", overCapSettlement, 110); err != nil {
		return "", "", "", err
	}
	var runCompletion struct {
		RunID                      string   `json:"run_id"`
		RunStatus                  string   `json:"run_status"`
		TerminalSessionRevoked     bool     `json:"terminal_session_revoked"`
		TerminalSessionKeyAbsent   bool     `json:"terminal_session_key_absent"`
		SessionKeyAuthorityBackend string   `json:"session_key_authority_backend"`
		SpendGrantIDs              []string `json:"per_spend_operation_grant_ids"`
	}
	completionBytes, readErr := os.ReadFile(filepath.Join(coord, "agent-manager-result.json"))
	expectedGrantCount := 4
	if hooks.X402 != nil {
		expectedGrantCount++
	}
	if readErr != nil || json.Unmarshal(completionBytes, &runCompletion) != nil || runCompletion.RunID != runIdentity.RunID || runCompletion.RunStatus != "complete" || !runCompletion.TerminalSessionRevoked || !runCompletion.TerminalSessionKeyAbsent || runCompletion.SessionKeyAuthorityBackend != "isolated-encrypted-file" || len(runCompletion.SpendGrantIDs) != expectedGrantCount {
		return "", "", "", fmt.Errorf("ordinary Agent Manager completion/revocation evidence is incomplete: completion=%+v err=%v", runCompletion, readErr)
	}
	for _, grantID := range runCompletion.SpendGrantIDs {
		if strings.TrimSpace(grantID) == "" {
			return "", "", "", errors.New("ordinary run continuity receipt has an empty per-spend operation grant ID")
		}
	}
	continuity := map[string]any{
		"run_id": runIdentity.RunID, "budget_id": budgetID, "mandate_id": mandateID,
		"wallet_address":                     walletResponse.Msg.GetWallet().GetAddress(),
		"session_generation_before_rotation": initialSessionGeneration,
		"session_generation_after_rotation":  rotatedWallet.GetSessionGeneration(),
		"in_cap_authorization_id":            authorizationID, "in_cap_settlement_id": settlementID,
		"in_cap_operation_grant_id": runCompletion.SpendGrantIDs[0], "in_cap_minor": 30, "remaining_minor_after_in_cap": 970,
		"in_cap_receipt":            "in-cap-chain-receipt.json",
		"in_cap_receipt_proof":      receiptContinuity["in-cap"],
		"over_cap_authorization_id": overCapAuthorizationID, "over_cap_settlement_id": overCapSettlementID,
		"over_cap_operation_grant_id": runCompletion.SpendGrantIDs[1], "over_cap_minor": 110, "remaining_minor_after_over_cap": 860,
		"over_cap_receipt":       "over-cap-chain-receipt.json",
		"over_cap_receipt_proof": receiptContinuity["over-cap"],
		"over_cap_retry_after_treasury_restart_same_settlement":  overCapRetry.GetSettlement().GetId() == overCapSettlement.GetSettlement().GetId(),
		"over_cap_retry_after_treasury_restart_same_transaction": overCapRetry.GetSettlement().GetExternalId() == overCapSettlement.GetSettlement().GetExternalId(),
		"over_cap_debit_count_before_restart_retry":              1,
		"over_cap_debit_count_after_restart_retry":               1,
		"sender_nonce_after_in_cap":                              nonceAfterInCap,
		"sender_nonce_after_over_cap":                            nonceAfterOverCap,
		"sender_nonce_after_over_cap_retry":                      nonceAfterOverCapRetry,
		"over_cap_raw_receipt_restart_continuity":                "over-cap-raw-receipt-restart-continuity.json",
		"over_cap_human_approval_notification_record":            "over-cap-approval-notification.json",
		"freeze_operation_grant_id":                              runCompletion.SpendGrantIDs[expectedGrantCount-2], "expiry_operation_grant_id": runCompletion.SpendGrantIDs[expectedGrantCount-1],
		"freeze_denial_record": "frozen-no-dispatch.json", "expiry_denial_record": "expired-mandate-no-dispatch.json",
		"terminal_run_record": "agent-manager-result.json", "terminal_session_revoked": runCompletion.TerminalSessionRevoked,
		"terminal_session_key_absent":   runCompletion.TerminalSessionKeyAbsent,
		"session_key_authority_backend": runCompletion.SessionKeyAuthorityBackend,
		"in_cap_headroom_records":       []string{"budget-headroom-before.log", "budget-headroom-after-in-cap.log"},
		"over_cap_headroom_records":     []string{"budget-headroom-after-over-cap.log", "budget-headroom-after-over-cap-retry.log"},
	}
	if hooks.X402 != nil {
		continuity["x402_operation_grant_id"] = runCompletion.SpendGrantIDs[2]
	}
	writeRehearsalJSON(t, filepath.Join(coord, "ordinary-run-continuity.json"), continuity)
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

type cf7ForkCall struct {
	PriceID         string `json:"price_id"`
	CallID          string `json:"call_id"`
	CatalogURL      string `json:"catalog_url"`
	Counterparty    string `json:"counterparty"`
	InstrumentID    string `json:"instrument_id"`
	AuthorizationID string `json:"authorization_id"`
	SettlementID    string `json:"settlement_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AmountMinor     int64  `json:"amount_minor"`
}

// runCF7TreasuryCoordinator drives the four priced calls of a fresh ordinary
// Agent Manager run. It verifies each settled Treasury response against the
// local Base fork, and sends the fourth call only after a distinct operator
// decision approves its pending over-cap authorization.
func runCF7TreasuryCoordinator(t *testing.T, ctx context.Context, coord string, childDone <-chan error, amOutputPath string, fork *basefork.Fork, buyerAddress string, client authconnect.TreasuryAdminClient, bookID string, hooks cf1ShowcaseHooks, config map[string]any) (string, string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(config["x402_calls"])
	if err != nil {
		return "", "", "", err
	}
	var calls []cf7ForkCall
	if err := json.Unmarshal(encoded, &calls); err != nil || len(calls) != 4 {
		return "", "", "", fmt.Errorf("decode four CF7 priced calls: %w", err)
	}
	var runIdentity struct {
		RunID string `json:"run_id"`
	}
	runIdentityBytes, err := os.ReadFile(filepath.Join(coord, "run-id.json"))
	if err != nil {
		return "", "", "", err
	}
	if err := json.Unmarshal(runIdentityBytes, &runIdentity); err != nil {
		return "", "", "", err
	}
	if runIdentity.RunID == "" {
		return "", "", "", errors.New("CF7 Agent Manager run ID is missing")
	}
	for i := 0; i < 3; i++ {
		index := i + 1
		call := calls[i]
		var ready struct {
			RunID           string `json:"run_id"`
			AuthorizationID string `json:"authorization_id"`
			SettlementID    string `json:"settlement_id"`
			AmountMinor     int64  `json:"amount_minor"`
			PriceID         string `json:"price_id"`
			CallID          string `json:"call_id"`
		}
		if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, fmt.Sprintf("x402-call-%d-authorization-ready.json", index)), &ready, childDone); err != nil {
			return "", "", "", fmt.Errorf("wait for CF7 call %d authorization: %w", index, err)
		}
		if ready.RunID != runIdentity.RunID || ready.AuthorizationID != call.AuthorizationID || ready.SettlementID != call.SettlementID || ready.AmountMinor != 1 || ready.PriceID != call.PriceID || ready.CallID != call.CallID {
			return "", "", "", fmt.Errorf("CF7 call %d authorization does not match its exact one-minor Offer Desk intent: expected_run=%q expected_call=%+v actual=%+v match={run:%t authorization:%t settlement:%t amount:%t price:%t call:%t}",
				index, runIdentity.RunID, call, ready,
				ready.RunID == runIdentity.RunID, ready.AuthorizationID == call.AuthorizationID,
				ready.SettlementID == call.SettlementID, ready.AmountMinor == 1,
				ready.PriceID == call.PriceID, ready.CallID == call.CallID)
		}
		proposal, err := readCLIProposal(coord, fmt.Sprintf("cf7-call-%d-authorization-propose.log", index))
		if err != nil || proposal.GetAuthorization().GetId() != call.AuthorizationID || proposal.GetAuthorization().GetSettlementId() != call.SettlementID || proposal.GetAuthorization().GetAmountMinor() != 1 || proposal.GetAuthorization().GetVerdict() != authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED || proposal.GetAuthorization().GetInstrumentId() != call.InstrumentID || proposal.GetAuthorization().GetSettlementIdempotencyKey() != call.IdempotencyKey+"-settlement" {
			return "", "", "", fmt.Errorf("CF7 call %d Treasury proposal is not approved for its exact priced intent: proposal=%+v err=%v", index, proposal, err)
		}
		// The fork is pinned to a historical Base block. Advance only this
		// disposable chain clock before signing/verification so USDC's
		// transferWithAuthorization validAfter bound is current.
		if err := fork.Call(ctx, "evm_setNextBlockTimestamp", []any{time.Now().Unix()}, nil); err != nil {
			return "", "", "", fmt.Errorf("align CF7 local-fork authorization clock for call %d: %w", index, err)
		}
		if err := fork.Call(ctx, "evm_mine", []any{}, nil); err != nil {
			return "", "", "", fmt.Errorf("mine CF7 local-fork authorization-clock update for call %d: %w", index, err)
		}
		before := cf1WalletUSDCBalance(t, ctx, fork, buyerAddress)
		if err := os.WriteFile(filepath.Join(coord, fmt.Sprintf("x402-call-%d-release", index)), []byte("Treasury verified the run-bound local-fork call"), 0o600); err != nil {
			return "", "", "", err
		}
		var settled struct {
			RunID             string `json:"run_id"`
			AuthorizationID   string `json:"authorization_id"`
			SettlementID      string `json:"settlement_id"`
			TreasuryReceiptID string `json:"treasury_receipt_id"`
			AmountMinor       int64  `json:"amount_minor"`
			Currency          string `json:"currency"`
			OperationGrantID  string `json:"operation_grant_id"`
		}
		if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, fmt.Sprintf("x402-call-%d-settled.json", index)), &settled, childDone); err != nil {
			return "", "", "", fmt.Errorf("wait for CF7 call %d settled response: %w", index, err)
		}
		if settled.RunID != runIdentity.RunID || settled.AuthorizationID != call.AuthorizationID || settled.SettlementID != call.SettlementID || settled.TreasuryReceiptID == "" || settled.AmountMinor != 1 || settled.Currency != "USD" || settled.OperationGrantID == "" {
			return "", "", "", fmt.Errorf("CF7 call %d returned incomplete or changed Treasury settlement evidence: %+v", index, settled)
		}
		response, err := readCLISettlement(coord, fmt.Sprintf("cf7-call-%d-settlement-report.log", index))
		if err != nil || response.GetSettlement().GetExternalId() != settled.TreasuryReceiptID || response.GetSettlement().GetAmountMinor() != 1 || response.GetSettlement().GetId() != call.SettlementID || response.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED {
			return "", "", "", fmt.Errorf("CF7 call %d CLI settlement response disagrees with retained report evidence: response=%+v err=%v", index, response, err)
		}
		rawReceipt, err := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{settled.TreasuryReceiptID})
		if err != nil {
			return "", "", "", fmt.Errorf("read CF7 call %d chain receipt: %w", index, err)
		}
		receipt, err := validateCF7X402RPCReceipt(rawReceipt, settled.TreasuryReceiptID, buyerAddress, fork.Accounts[2], 1)
		if err != nil {
			return "", "", "", fmt.Errorf("validate CF7 call %d exact native-USDC transfer: %w", index, err)
		}
		var transaction struct {
			Hash  string `json:"hash"`
			From  string `json:"from"`
			Nonce string `json:"nonce"`
		}
		if err := fork.Call(ctx, "eth_getTransactionByHash", []any{settled.TreasuryReceiptID}, &transaction); err != nil || transaction.Hash != settled.TreasuryReceiptID || transaction.From == "" || transaction.Nonce == "" {
			return "", "", "", fmt.Errorf("CF7 call %d missing facilitator transaction nonce: tx=%+v err=%v", index, transaction, err)
		}
		after := cf1WalletUSDCBalance(t, ctx, fork, buyerAddress)
		if new(big.Int).Sub(before, after).Cmp(big.NewInt(10_000)) != 0 {
			return "", "", "", fmt.Errorf("CF7 call %d did not debit exactly one cent from its mandate wallet: before=%s after=%s", index, before, after)
		}
		writeRehearsalJSON(t, filepath.Join(coord, fmt.Sprintf("cf7-call-%d-chain-receipt.json", index)), map[string]any{
			"authorization_id": call.AuthorizationID, "settlement_id": call.SettlementID, "operation_grant_id": settled.OperationGrantID,
			"transaction_hash": settled.TreasuryReceiptID, "transaction_nonce": transaction.Nonce,
			"transaction_sender": transaction.From, "receipt_status": receipt.Status,
			"amount_minor": settled.AmountMinor, "currency": settled.Currency,
			"wallet_atoms_before": before.String(), "wallet_atoms_after": after.String(),
		})
	}
	call := calls[3]
	var decision struct {
		AuthorizationID string `json:"authorization_id"`
		SettlementID    string `json:"settlement_id"`
		PriceID         string `json:"price_id"`
		CallID          string `json:"call_id"`
		AmountMinor     int64  `json:"amount_minor"`
		Verdict         string `json:"verdict"`
		Dispatch        string `json:"dispatch"`
		CallRelease     bool   `json:"call_release_written"`
		OperationGrant  string `json:"operation_grant_id"`
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "x402-call-4-decision-raised.json"), &decision, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for fourth CF7 decision: %w", err)
	}
	proposal, err := readCLIProposal(coord, "cf7-call-4-authorization-propose.log")
	if err != nil || decision.AuthorizationID != call.AuthorizationID || decision.SettlementID != call.SettlementID || decision.PriceID != call.PriceID || decision.CallID != call.CallID || decision.AmountMinor != 110 || decision.Verdict != "AUTHORIZATION_VERDICT_PENDING" || decision.Dispatch != "none" || decision.CallRelease || decision.OperationGrant == "" || proposal.GetAuthorization().GetVerdict() != authpb.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING || proposal.GetAuthorization().GetAmountMinor() != 110 || proposal.GetAuthorization().GetId() != call.AuthorizationID {
		return "", "", "", fmt.Errorf("fourth CF7 quote did not stop at a pending 110-minor decision: decision=%+v proposal=%+v err=%v", decision, proposal, err)
	}
	if _, err := os.Stat(filepath.Join(coord, "x402-call-4-release")); !errors.Is(err, os.ErrNotExist) {
		return "", "", "", errors.New("CF7 fourth call had a dispatch release despite the pending decision")
	}
	if _, err := os.Stat(filepath.Join(coord, "x402-call-4-settled.json")); !errors.Is(err, os.ErrNotExist) {
		return "", "", "", errors.New("CF7 fourth call produced a settlement before separate authorization")
	}
	beforeFourth := cf1WalletUSDCBalance(t, ctx, fork, buyerAddress)
	time.Sleep(100 * time.Millisecond)
	afterFourth := cf1WalletUSDCBalance(t, ctx, fork, buyerAddress)
	if beforeFourth.Cmp(afterFourth) != 0 {
		return "", "", "", errors.New("CF7 fourth pending decision changed the mandate wallet balance")
	}
	queued, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
	if err != nil {
		return "", "", "", fmt.Errorf("read queued Treasury over-cap decisions: %w", err)
	}
	var pendingApproval *treasurysharedpb.ApprovalRequest
	for _, candidate := range queued.Msg.GetApprovals() {
		if candidate.GetAuthorizationId() == call.AuthorizationID {
			if pendingApproval != nil {
				return "", "", "", fmt.Errorf("CF7 authorization %s has duplicate queued Treasury decisions", call.AuthorizationID)
			}
			pendingApproval = candidate
		}
	}
	if pendingApproval == nil || pendingApproval.GetAmountMinor() != call.AmountMinor {
		return "", "", "", fmt.Errorf("CF7 exact over-cap authorization is not queued for its configured amount: approvals=%v", queued.Msg.GetApprovals())
	}
	var approvalID, approvalSource, approvalEventID string
	var approvedRecord *treasurysharedpb.ApprovalRequest
	if hooks.AnswerApprovalInHub != nil {
		notice, answerErr := hooks.AnswerApprovalInHub(ctx, pendingApproval.GetId())
		if answerErr != nil || notice == nil || !strings.Contains(notice.GetBody(), "110 USD minor units") || !strings.Contains(notice.GetBody(), pendingApproval.GetId()) {
			return "", "", "", fmt.Errorf("Notification Hub did not resolve the exact CF7 over-cap approval: notice=%v err=%v", notice, answerErr)
		}
		approved, waitErr := waitForCF1ApprovalStatus(ctx, client, bookID, pendingApproval.GetId(), treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED)
		if waitErr != nil || approved.GetAuthorizationId() != call.AuthorizationID || approved.GetAmountMinor() != call.AmountMinor {
			return "", "", "", fmt.Errorf("Treasury did not retain the exact CF7 approval returned through Notification Hub: approval=%+v err=%v", approved, waitErr)
		}
		approvedRecord = approved
		approvalID, approvalSource = approved.GetId(), "notification-hub"
		decisionEventPath := filepath.Join(filepath.Dir(coord), "over-cap-approval-decision-event-"+pendingApproval.GetId()+".json")
		eventData, eventErr := os.ReadFile(decisionEventPath)
		if eventErr != nil {
			return "", "", "", fmt.Errorf("read exact CF7 Treasury approval event delivered to Notification Hub: %w", eventErr)
		}
		var event domainevents.EventEnvelope
		if eventErr := protojson.Unmarshal(eventData, &event); eventErr != nil || event.GetEventType() != "treasury.approval.decision_requested.v1" || event.GetEventId() == "" {
			return "", "", "", fmt.Errorf("CF7 Notification Hub approval event is invalid: event=%+v err=%v", event, eventErr)
		}
		approvalEventID = event.GetEventId()
		writeRehearsalProto(t, filepath.Join(coord, "x402-call-4-treasury-approved.proto.json"), approvedRecord)
	} else {
		approved, resolveErr := client.ResolveApproval(ctx, operatorRequest(&authpb.ResolveApprovalRequest{ApprovalId: pendingApproval.GetId(), Resolution: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
		if resolveErr != nil || approved.Msg.GetApproval().GetStatus() != treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED || approved.Msg.GetApproval().GetAuthorizationId() != call.AuthorizationID {
			return "", "", "", fmt.Errorf("explicit local operator decision did not approve the exact CF7 over-cap request: approval=%v err=%v", approved, resolveErr)
		}
		approvedRecord = approved.Msg.GetApproval()
		approvalID, approvalSource = approvedRecord.GetId(), "treasury-admin"
		writeRehearsalProto(t, filepath.Join(coord, "x402-call-4-treasury-approved.proto.json"), approvedRecord)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "x402-call-4-treasury-approved.json"), map[string]any{
		"run_id": runIdentity.RunID, "authorization_id": call.AuthorizationID,
		"settlement_id": call.SettlementID, "approval_id": approvalID, "approval_event_id": approvalEventID,
		"amount_minor": call.AmountMinor, "decision": "approved", "source": approvalSource,
	})
	if err := os.WriteFile(filepath.Join(coord, "x402-call-4-settle-release"), []byte("approved by the local Treasury operator decision route"), 0o600); err != nil {
		return "", "", "", err
	}
	var settled struct {
		RunID                    string `json:"run_id"`
		AuthorizationID          string `json:"authorization_id"`
		SettlementID             string `json:"settlement_id"`
		TreasuryReceiptID        string `json:"treasury_receipt_id"`
		AmountMinor              int64  `json:"amount_minor"`
		Currency                 string `json:"currency"`
		ApprovalID               string `json:"approval_id"`
		ApprovalSource           string `json:"approval_source"`
		SettlementReportAttempts int    `json:"settlement_report_attempts"`
		RetrySameReceipt         bool   `json:"idempotent_retry_returned_same_receipt"`
	}
	if err := waitForCF1FileOrChild(ctx, filepath.Join(coord, "x402-call-4-settled.json"), &settled, childDone); err != nil {
		return "", "", "", fmt.Errorf("wait for approved CF7 over-cap settlement: %w", err)
	}
	if settled.RunID != runIdentity.RunID || settled.AuthorizationID != call.AuthorizationID || settled.SettlementID != call.SettlementID || settled.TreasuryReceiptID == "" || settled.AmountMinor != call.AmountMinor || settled.Currency != "USD" || settled.ApprovalID != approvalID || settled.ApprovalSource != approvalSource || settled.SettlementReportAttempts < 2 || !settled.RetrySameReceipt {
		return "", "", "", fmt.Errorf("CF7 over-cap settlement did not retain its exact approval and retry identity: %+v", settled)
	}
	firstResponse, firstErr := readCLISettlement(coord, "cf7-call-4-settlement-report-first.log")
	if firstErr != nil {
		firstResponse, firstErr = readCLISettlement(coord, "cf7-call-4-settlement-report-after-report-approval.log")
	}
	if firstErr != nil || firstResponse.GetSettlement().GetId() != call.SettlementID || firstResponse.GetSettlement().GetExternalId() != settled.TreasuryReceiptID || firstResponse.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED {
		return "", "", "", fmt.Errorf("CF7 first successful over-cap settlement response does not match Treasury evidence: response=%+v err=%v", firstResponse, firstErr)
	}
	retryResponse, err := readCLISettlement(coord, "cf7-call-4-settlement-report-retry.log")
	if err != nil || retryResponse.GetSettlement().GetId() != firstResponse.GetSettlement().GetId() || retryResponse.GetSettlement().GetExternalId() != firstResponse.GetSettlement().GetExternalId() || retryResponse.GetSettlement().GetOutcome() != settlementpb.ChargeOutcome_CHARGE_OUTCOME_SETTLED {
		return "", "", "", fmt.Errorf("CF7 over-cap retry changed the settled receipt: first=%+v retry=%+v err=%v", firstResponse.GetSettlement(), retryResponse.GetSettlement(), err)
	}
	rawReceipt, err := fork.CallRaw(ctx, "eth_getTransactionReceipt", []any{settled.TreasuryReceiptID})
	if err != nil {
		return "", "", "", fmt.Errorf("read CF7 approved over-cap chain receipt: %w", err)
	}
	receipt, err := validateCF7X402RPCReceipt(rawReceipt, settled.TreasuryReceiptID, buyerAddress, fork.Accounts[2], 110)
	if err != nil {
		return "", "", "", fmt.Errorf("validate CF7 exact over-cap native-USDC transfer: %w", err)
	}
	afterSettlement := cf1WalletUSDCBalance(t, ctx, fork, buyerAddress)
	if new(big.Int).Sub(beforeFourth, afterSettlement).Cmp(big.NewInt(1_100_000)) != 0 {
		return "", "", "", fmt.Errorf("CF7 approved over-cap call did not debit exactly 110 USD minor units once: before=%s after=%s", beforeFourth, afterSettlement)
	}
	writeRehearsalJSON(t, filepath.Join(coord, "cf7-call-4-chain-receipt.json"), map[string]any{
		"run_id": runIdentity.RunID, "authorization_id": call.AuthorizationID, "settlement_id": call.SettlementID,
		"approval_id": approvalID, "transaction_hash": settled.TreasuryReceiptID,
		"receipt_status": receipt.Status, "amount_minor": settled.AmountMinor, "currency": settled.Currency,
		"wallet_atoms_before": beforeFourth.String(), "wallet_atoms_after": afterSettlement.String(),
		"single_settlement_receipt_after_retry": true,
	})
	if err := waitForCF7Artifact(ctx, filepath.Join(coord, "run-result.json")); err != nil {
		return "", "", "", fmt.Errorf("CF7 ordinary run did not complete after the over-cap approval and settlement: %w", err)
	}
	if childErr := <-childDone; childErr != nil {
		data, _ := os.ReadFile(amOutputPath)
		return "", "", "", fmt.Errorf("CF7 Agent Manager integration failed: %w: %s", childErr, strings.TrimSpace(string(data)))
	}
	return runIdentity.RunID, call.AuthorizationID, call.SettlementID, nil
}

func waitForCF7Artifact(ctx context.Context, path string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func validateCF7X402RPCReceipt(raw []byte, transactionHash, payer, payee string, amountMinor int64) (cf1RPCReceipt, error) {
	var receipt cf1RPCReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return receipt, err
	}
	if receipt.Status != "0x1" || !strings.EqualFold(receipt.TransactionHash, transactionHash) || receipt.BlockHash == "" || receipt.BlockNumber == "" || !strings.EqualFold(receipt.To, baseusdc.USDCAddress) {
		return receipt, errors.New("RPC receipt is not a successful native-USDC settlement")
	}
	expectedAtoms := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(10_000))
	for _, event := range receipt.Logs {
		if !strings.EqualFold(event.Address, baseusdc.USDCAddress) || len(event.Topics) != 3 || !strings.EqualFold(event.Topics[0], cf1ERC20TransferTopic) || len(event.Topics[1]) != 66 || len(event.Topics[2]) != 66 {
			continue
		}
		from := "0x" + strings.ToLower(event.Topics[1][26:])
		to := "0x" + strings.ToLower(event.Topics[2][26:])
		atoms, ok := new(big.Int).SetString(strings.TrimPrefix(event.Data, "0x"), 16)
		if ok && strings.EqualFold(from, payer) && strings.EqualFold(to, payee) && atoms.Cmp(expectedAtoms) == 0 {
			return receipt, nil
		}
	}
	return receipt, errors.New("RPC receipt lacks the exact x402 payer-to-payee Transfer event")
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
			// The child can publish its final marker and complete its detached
			// run between the first read above and this select. Re-read after the
			// terminal notification so completion cannot mask a durable marker.
			if data, readErr := os.ReadFile(path); readErr == nil {
				if target == nil {
					return nil
				}
				return json.Unmarshal(data, target)
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return readErr
			}
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
	tokens map[string]rehearsalServiceToken
	expiry time.Time
}

type rehearsalServiceToken struct {
	value    string
	subject  string
	audience string
	scopes   []string
}

func (s *rehearsalServiceCredentialExchange) ExchangeServiceCredential(_ context.Context, request *connect.Request[accountsV1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsV1.ExchangeServiceCredentialResponse], error) {
	serviceToken, ok := s.tokens[request.Msg.GetCredential()]
	if !ok || !time.Now().Before(s.expiry) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("rehearsal credential is invalid or expired"))
	}
	return connect.NewResponse(&accountsV1.ExchangeServiceCredentialResponse{AccessToken: serviceToken.value, AccessTokenExpiresAt: timestamppb.New(s.expiry), Audience: serviceToken.audience, Subject: serviceToken.subject, Scopes: append([]string(nil), serviceToken.scopes...)}), nil
}

func startServiceIdentityFixture(t *testing.T) (moneyLedgerToken, notificationToken, agentManagerToken, x402OrderToken, x402ReceiptToken, moneyLedgerOperatorToken, jwksURL, exchangeURL string, closeServers func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	expiry := time.Now().UTC().Add(2 * time.Hour)
	encode := func(value any) string {
		raw, marshalErr := json.Marshal(value)
		require.NoError(t, marshalErr)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	identityJWT := func(subject, actorKind, resource string, scopes []string) string {
		unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "cf2-local"}) + "." + encode(map[string]any{
			"user_id": subject, "sub": subject, "actor_kind": actorKind, "scope": scopes,
			"aud": "scenario-authenticator:" + resource, "iss": "scenario-authenticator", "iat": time.Now().Unix(), "exp": expiry.Unix(),
		})
		digest := sha256.Sum256([]byte(unsigned))
		signature, signErr := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		require.NoError(t, signErr)
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	}
	serviceJWT := func(subject, resource string, scopes []string) string {
		return identityJWT(subject, "service", resource, scopes)
	}
	moneyLedgerToken = serviceJWT("treasury-money-ledger", "money-ledger", []string{"money-ledger:write"})
	moneyLedgerOperatorToken = identityJWT("cf5-local-operator", "human", "money-ledger", []string{"money-ledger:write"})
	notificationToken = serviceJWT("treasury-notification-hub", "notification-hub", []string{"notification-hub:send"})
	agentManagerToken = serviceJWT("treasury-agent-manager", "agent-manager", []string{"agent-manager:write"})
	askReaderToken := serviceJWT("treasury-ask-reader", "notification-hub-ask-reader", []string{"notification-hub:ask-read"})
	x402OrderToken = serviceJWT("landing-page-business-suite-treasury-x402-order", "treasury-x402-order", []string{"treasury:x402-authorize", "treasury:x402-settle", "treasury:x402-cancel"})
	x402ReceiptToken = serviceJWT("landing-page-business-suite-treasury-x402-receipt-reader", "treasury-x402-receipt", []string{"treasury:x402-receipt-read"})
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
	path, handler := accountsConnect.NewAccountsServiceHandler(&rehearsalServiceCredentialExchange{
		tokens: map[string]rehearsalServiceToken{
			"cf2-local-treasury-service-only":                  {value: moneyLedgerToken, subject: "treasury-money-ledger", audience: "scenario-authenticator:money-ledger", scopes: []string{"money-ledger:write"}},
			"cf2-local-treasury-notice-only":                   {value: notificationToken, subject: "treasury-notification-hub", audience: "scenario-authenticator:notification-hub", scopes: []string{"notification-hub:send"}},
			"cf2-local-treasury-agent-manager-only":            {value: agentManagerToken, subject: "treasury-agent-manager", audience: "scenario-authenticator:agent-manager", scopes: []string{"agent-manager:write"}},
			"cf2-local-treasury-ask-reader-only":               {value: askReaderToken, subject: "treasury-ask-reader", audience: "scenario-authenticator:notification-hub-ask-reader", scopes: []string{"notification-hub:ask-read"}},
			"cf2-local-lpbs-treasury-x402-order-only":          {value: x402OrderToken, subject: "landing-page-business-suite-treasury-x402-order", audience: "scenario-authenticator:treasury-x402-order", scopes: []string{"treasury:x402-authorize", "treasury:x402-settle", "treasury:x402-cancel"}},
			"cf2-local-lpbs-treasury-x402-receipt-reader-only": {value: x402ReceiptToken, subject: "landing-page-business-suite-treasury-x402-receipt-reader", audience: "scenario-authenticator:treasury-x402-receipt", scopes: []string{"treasury:x402-receipt-read"}},
		},
		expiry: expiry,
	})
	exchangeMux.Handle(path, handler)
	exchangeServer := httptest.NewServer(exchangeMux)
	return moneyLedgerToken, notificationToken, agentManagerToken, x402OrderToken, x402ReceiptToken, moneyLedgerOperatorToken, jwksServer.URL + "/jwks", exchangeServer.URL, func() { exchangeServer.Close(); jwksServer.Close() }
}

func provisionCF2FacilitatorSigner(t *testing.T, key *ecdsa.PrivateKey) {
	t.Helper()
	const identityName = "vrooli/x402-facilitator-local-test"
	const fieldName = "facilitator-private-key"
	command := exec.Command("vrooli", "credentials", "provision", "--identity", identityName, "--field", fieldName)
	command.Stdin = strings.NewReader(fmt.Sprintf("%x", gethcrypto.FromECDSA(key)))
	output, err := command.CombinedOutput()
	require.NoError(t, err, "provision only the ephemeral local-fork facilitator signer: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		remove := exec.Command("vrooli", "credentials", "delete", "--identity", identityName, "--field", fieldName, "--yes")
		if output, err := remove.CombinedOutput(); err != nil {
			t.Errorf("remove ephemeral local-fork facilitator signer: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

func startCF2Facilitator(t *testing.T, ctx context.Context, runDir string) {
	t.Helper()
	start := exec.CommandContext(ctx, "vrooli", "resource", "start", "x402-facilitator-local-test", "--json")
	output, err := start.CombinedOutput()
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "x402-facilitator-start.json"), output, 0o600))
	require.NoError(t, err, "start the pinned local-only x402 facilitator: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stop := exec.CommandContext(cleanupCtx, "vrooli", "resource", "stop", "x402-facilitator-local-test", "--json")
		output, err := stop.CombinedOutput()
		_ = os.WriteFile(filepath.Join(runDir, "x402-facilitator-stop.json"), output, 0o600)
		if err != nil {
			t.Errorf("stop the pinned local-only x402 facilitator: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

type cf2CanonicalOffer struct {
	ID            string `json:"id"`
	Version       int    `json:"version"`
	Name          string `json:"name"`
	ProposedPrice struct {
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	} `json:"proposed_price"`
	MPPResourcePath string `json:"mpp_resource_path"`
	X402PriceID     string `json:"x402_price_id"`
}

type cf2ServiceOrder struct {
	ID                 string          `json:"order_id"`
	JobID              string          `json:"job_id"`
	OfferID            string          `json:"offer_id"`
	OfferVersion       int             `json:"offer_version"`
	JobInput           json.RawMessage `json:"job_input"`
	AmountMinor        int64           `json:"amount_minor"`
	Currency           string          `json:"currency"`
	Mode               string          `json:"mode"`
	State              string          `json:"state"`
	AuthorizationID    string          `json:"authorization_id"`
	CaptureID          string          `json:"capture_id"`
	LedgerPostingID    string          `json:"ledger_posting_id"`
	LedgerReceiptID    string          `json:"ledger_receipt_id"`
	TreasuryExternalID string          `json:"treasury_external_id"`
}

func reconcileCF2LPBSOrder(t *testing.T, ctx context.Context, runDir, treasuryURL, lpbsURL, orderID, receiptToken string) {
	t.Helper()
	endpoint := lpbsURL + "/api/v1/service-orders/" + url.PathEscape(orderID) + "/reconcile"
	deadline := time.Now().Add(30 * time.Second)
	var order cf2ServiceOrder
	var lastStatus int
	var lastBody []byte
	for time.Now().Before(deadline) {
		lastStatus, lastBody = cf2LPBSJSONRequest(t, ctx, http.MethodPost, endpoint, nil)
		if lastStatus == http.StatusOK {
			if err := json.Unmarshal(lastBody, &order); err == nil && order.State == "reconciled" {
				break
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("LPBS accepted-receipt reconciliation context ended:", ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
	require.Equalf(t, http.StatusOK, lastStatus, "LPBS accepted-receipt reconciliation: %s", strings.TrimSpace(string(lastBody)))
	require.Equal(t, "reconciled", order.State)
	require.NotEmpty(t, order.LedgerPostingID)
	require.NotEmpty(t, order.LedgerReceiptID)
	require.Equal(t, int64(4900), order.AmountMinor)
	require.Equal(t, "USD", order.Currency)
	require.True(t, strings.HasPrefix(order.TreasuryExternalID, "treasury-x402-inflow:"))
	readURL := strings.TrimRight(treasuryURL, "/") + "/api/v1/x402/receipts?" + url.Values{"external_id": []string{order.TreasuryExternalID}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, readURL, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+receiptToken)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	receiptBody, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, response.StatusCode, "Treasury accepted receipt readback: %s", strings.TrimSpace(string(receiptBody)))
	var receipt struct {
		ExternalID  string `json:"external_id"`
		PostingID   string `json:"posting_id"`
		ReceiptID   string `json:"receipt_id"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
		Basis       string `json:"basis"`
	}
	require.NoError(t, json.Unmarshal(receiptBody, &receipt))
	require.Equal(t, order.TreasuryExternalID, receipt.ExternalID)
	require.Equal(t, order.LedgerPostingID, receipt.PostingID)
	require.Equal(t, order.LedgerReceiptID, receipt.ReceiptID)
	require.Equal(t, order.AmountMinor, receipt.AmountMinor)
	require.Equal(t, order.Currency, receipt.Currency)
	require.NotEmpty(t, receipt.Basis)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf2-lpbs-treasury-accepted-receipt.json"), receipt)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf2-lpbs-order-reconciled.json"), order)
	status, body := cf2LPBSJSONRequest(t, ctx, http.MethodPost, endpoint, nil)
	require.Equalf(t, http.StatusOK, status, "LPBS reconciliation replay: %s", strings.TrimSpace(string(body)))
	var replay cf2ServiceOrder
	require.NoError(t, json.Unmarshal(body, &replay))
	require.Equal(t, order.LedgerPostingID, replay.LedgerPostingID)
	require.Equal(t, order.LedgerReceiptID, replay.LedgerReceiptID)
}

func cf2LPBSJSONRequest(t *testing.T, ctx context.Context, method, endpoint string, body []byte) (int, []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	require.NoError(t, err)
	return response.StatusCode, responseBody
}

func cf2SignLocalForkPayment(t *testing.T, challenge string, key *ecdsa.PrivateKey) string {
	t.Helper()
	decodedChallenge, err := base64.StdEncoding.DecodeString(strings.TrimSpace(challenge))
	require.NoError(t, err)
	var required struct {
		X402Version int               `json:"x402Version"`
		Resource    json.RawMessage   `json:"resource"`
		Accepts     []json.RawMessage `json:"accepts"`
		Extensions  json.RawMessage   `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(decodedChallenge, &required))
	require.Equal(t, 2, required.X402Version)
	require.Len(t, required.Accepts, 1)
	var accepted map[string]any
	require.NoError(t, json.Unmarshal(required.Accepts[0], &accepted))
	var terms struct {
		Scheme            string          `json:"scheme"`
		Network           string          `json:"network"`
		Amount            string          `json:"amount"`
		PayTo             string          `json:"payTo"`
		MaxTimeoutSeconds uint64          `json:"maxTimeoutSeconds"`
		Asset             string          `json:"asset"`
		Extra             json.RawMessage `json:"extra"`
	}
	require.NoError(t, json.Unmarshal(required.Accepts[0], &terms))
	require.Equal(t, "exact", terms.Scheme)
	require.Equal(t, "eip155:8453", terms.Network)
	var method struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(terms.Extra, &method))
	now := time.Now().UTC()
	nonceBytes := make([]byte, 32)
	_, err = rand.Read(nonceBytes)
	require.NoError(t, err)
	authorization := map[string]any{
		"from": gethcrypto.PubkeyToAddress(key.PublicKey).Hex(), "to": terms.PayTo, "value": terms.Amount,
		"validAfter":  strconv.FormatInt(now.Add(-10*time.Minute).Unix(), 10),
		"validBefore": strconv.FormatInt(now.Add(time.Duration(terms.MaxTimeoutSeconds)*time.Second).Unix(), 10),
		"nonce":       "0x" + hex.EncodeToString(nonceBytes),
	}
	chainID, err := strconv.ParseUint(strings.TrimPrefix(terms.Network, "eip155:"), 10, 64)
	require.NoError(t, err)
	typedDataMap := map[string]any{
		"types": map[string]any{
			"EIP712Domain":              []map[string]string{{"name": "name", "type": "string"}, {"name": "version", "type": "string"}, {"name": "chainId", "type": "uint256"}, {"name": "verifyingContract", "type": "address"}},
			"TransferWithAuthorization": []map[string]string{{"name": "from", "type": "address"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"}, {"name": "validAfter", "type": "uint256"}, {"name": "validBefore", "type": "uint256"}, {"name": "nonce", "type": "bytes32"}},
		},
		"primaryType": "TransferWithAuthorization",
		"domain":      map[string]any{"name": method.Name, "version": method.Version, "chainId": strconv.FormatUint(chainID, 10), "verifyingContract": terms.Asset},
		"message":     authorization,
	}
	typedBytes, err := json.Marshal(typedDataMap)
	require.NoError(t, err)
	var typedData apitypes.TypedData
	require.NoError(t, json.Unmarshal(typedBytes, &typedData))
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	require.NoError(t, err)
	signature, err := gethcrypto.Sign(digest, key)
	require.NoError(t, err)
	signature[64] += 27
	payload := map[string]any{
		"x402Version": required.X402Version, "accepted": accepted,
		"resource": json.RawMessage(required.Resource), "payload": map[string]any{
			"signature": "0x" + hex.EncodeToString(signature), "authorization": authorization,
		},
		"extensions": json.RawMessage(required.Extensions),
	}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(encoded)
}

func readCF2CanonicalServiceOffer(t *testing.T, ctx context.Context, offerDeskURL string) cf2CanonicalOffer {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(offerDeskURL, "/")+"/api/v1/proposal-preview/service-offer", nil)
	require.NoError(t, err)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var body struct {
		Offer cf2CanonicalOffer `json:"offer"`
	}
	require.NoError(t, json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body))
	require.Equal(t, "company-market-scan", body.Offer.ID)
	require.Equal(t, 1, body.Offer.Version)
	require.Equal(t, "company-market-scan-v1", body.Offer.X402PriceID)
	require.Equal(t, int64(4900), body.Offer.ProposedPrice.AmountMinor)
	require.Equal(t, "USD", body.Offer.ProposedPrice.Currency)
	require.Equal(t, "/api/v1/services/company-market-scan", body.Offer.MPPResourcePath)
	return body.Offer
}

func seedCF2CanonicalServiceOfferGraph(t *testing.T, ctx context.Context, offerDeskURL, runDir string, canonical cf2CanonicalOffer) (string, string) {
	t.Helper()
	catalog := offersconnect.NewCatalogServiceClient(&http.Client{Timeout: 15 * time.Second}, offerDeskURL)
	offer, err := catalog.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_OFFER, Name: canonical.ID, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	variantName := fmt.Sprintf("%s-v%d", canonical.ID, canonical.Version)
	variant, err := catalog.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_VARIANT, Name: variantName, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	_, err = catalog.CreateEdge(ctx, connect.NewRequest(&offerspb.CreateEdgeRequest{Edge: &offerspb.Edge{
		FromId: offer.Msg.GetNode().GetId(), ToId: variant.Msg.GetNode().GetId(), Kind: "sells_at",
		IntendedPriceMinor: canonical.ProposedPrice.AmountMinor, Currency: canonical.ProposedPrice.Currency, IntendedPriceDeclared: true,
	}}))
	require.NoError(t, err)
	writeRehearsalJSON(t, filepath.Join(runDir, "cf2-service-offer-catalog-join.json"), map[string]any{
		"service_offer_id": canonical.ID, "service_offer_version": canonical.Version,
		"x402_price_id": canonical.X402PriceID, "amount_minor": canonical.ProposedPrice.AmountMinor,
		"currency": canonical.ProposedPrice.Currency, "catalog_offer_name": canonical.ID,
		"catalog_offer_id": offer.Msg.GetNode().GetId(), "catalog_variant_name": variantName,
		"catalog_variant_id": variant.Msg.GetNode().GetId(), "edge": "sells_at",
	})
	return offer.Msg.GetNode().GetId(), variant.Msg.GetNode().GetId()
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
		{"scenario-authenticator/service-principal/treasury-money-ledger/money-ledger", "credential", "cf2-local-treasury-service-only"},
		{"scenario-authenticator/service-principal/treasury-agent-manager/agent-manager", "credential", "cf2-local-treasury-agent-manager-only"},
		{"scenario-authenticator/service-principal/treasury-ask-reader/notification-hub-ask-reader", "credential", "cf2-local-treasury-ask-reader-only"},
		{"scenario-authenticator/service-principal/landing-page-business-suite/treasury-x402-order", "credential", "cf2-local-lpbs-treasury-x402-order-only"},
		{"scenario-authenticator/service-principal/landing-page-business-suite/treasury-x402-receipt", "credential", "cf2-local-lpbs-treasury-x402-receipt-reader-only"},
		{"vrooli/landing-page-business-suite", "service-secret", "cf2-local-lpbs-service-only"},
	} {
		identity, parseErr := credentialauthority.ParseIdentity(value.identity)
		require.NoError(t, parseErr)
		require.NoError(t, authority.Put(identity, value.field, value.secret))
	}
}

type cf1DecisionEventsBridge struct {
	mu                    sync.Mutex
	secret                string
	hubURL                string
	treasuryURL           string
	artifactDir           string
	decisionEvents        chan []byte
	tasksByApproval       map[string]string
	correlationByApproval map[string]map[string]string
	answerEventByAsk      map[string]string
	pendingAnswers        map[string][]byte
	subscriptions         []cf1DecisionSubscription
	nextSubID             int64
	treasuryReady         chan struct{}
}

type cf1DecisionSubscription struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	OwnerScenario  string `json:"owner_scenario"`
	EventPattern   string `json:"event_pattern"`
	DeliveryType   string `json:"delivery_type"`
	DeliveryTarget string `json:"delivery_target"`
	Enabled        bool   `json:"enabled"`
}

func newCF1DecisionEventsBridge(secret, artifactDir string) *cf1DecisionEventsBridge {
	return &cf1DecisionEventsBridge{
		secret: secret, artifactDir: artifactDir, decisionEvents: make(chan []byte, 16),
		tasksByApproval: make(map[string]string), answerEventByAsk: make(map[string]string), pendingAnswers: make(map[string][]byte),
		correlationByApproval: make(map[string]map[string]string),
		nextSubID:             1, treasuryReady: make(chan struct{}),
	}
}

func (b *cf1DecisionEventsBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/subscriptions":
		b.mu.Lock()
		var values []cf1DecisionSubscription
		for _, subscription := range b.subscriptions {
			if subscription.OwnerScenario == r.URL.Query().Get("owner") {
				values = append(values, subscription)
			}
		}
		b.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(values)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subscriptions":
		var subscription cf1DecisionSubscription
		if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
			http.Error(w, "invalid isolated event subscription", http.StatusBadRequest)
			return
		}
		b.mu.Lock()
		for i, existing := range b.subscriptions {
			if existing.OwnerScenario == subscription.OwnerScenario && existing.Name == subscription.Name {
				subscription.ID = existing.ID
				b.subscriptions[i] = subscription
				b.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": subscription.ID})
				return
			}
		}
		subscription.ID = b.nextSubID
		b.nextSubID++
		b.subscriptions = append(b.subscriptions, subscription)
		if subscription.OwnerScenario == "treasury" && subscription.Name == "treasury-approval-decision-answers" {
			select {
			case <-b.treasuryReady:
			default:
				close(b.treasuryReady)
			}
		}
		b.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": subscription.ID})
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/subscriptions/"):
		var subscription cf1DecisionSubscription
		if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
			http.Error(w, "invalid isolated event subscription", http.StatusBadRequest)
			return
		}
		b.mu.Lock()
		for i, existing := range b.subscriptions {
			if fmt.Sprintf("/api/v1/subscriptions/%d", existing.ID) == r.URL.Path {
				subscription.ID = existing.ID
				b.subscriptions[i] = subscription
				break
			}
		}
		b.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/deliveries/reconcile"):
		var request struct {
			EventID string `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.EventID == "" {
			http.Error(w, "invalid event reconciliation request", http.StatusBadRequest)
			return
		}
		b.mu.Lock()
		answer := append([]byte(nil), b.pendingAnswers[request.EventID]...)
		b.mu.Unlock()
		if len(answer) == 0 {
			http.Error(w, "isolated event not retained", http.StatusNotFound)
			return
		}
		if err := b.deliverAnswer(r.Context(), answer); err != nil {
			http.Error(w, "Treasury answer delivery failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reconciled", "event_id": request.EventID})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/events":
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "cannot read isolated domain event", http.StatusBadRequest)
			return
		}
		var envelope domainevents.EventEnvelope
		if err := protojson.Unmarshal(body, &envelope); err != nil {
			http.Error(w, "invalid isolated domain event", http.StatusBadRequest)
			return
		}
		var payload structpb.Struct
		if envelope.GetData() == nil || envelope.GetData().UnmarshalTo(&payload) != nil {
			http.Error(w, "isolated domain event has no struct payload", http.StatusBadRequest)
			return
		}
		switch envelope.GetEventType() {
		case "treasury.approval.decision_requested.v1":
			if err := b.acceptDecision(r.Context(), body, &envelope, payload.AsMap()); err != nil {
				http.Error(w, "Notification Hub did not accept Treasury decision", http.StatusBadGateway)
				return
			}
		case "notification_hub.ask.resolved.v1":
			b.mu.Lock()
			b.pendingAnswers[envelope.GetEventId()] = append([]byte(nil), body...)
			if askID, _ := payload.AsMap()["ask_id"].(string); askID != "" {
				b.answerEventByAsk[askID] = envelope.GetEventId()
			}
			b.mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"eventId": envelope.GetEventId(), "status": "accepted"})
	default:
		http.NotFound(w, r)
	}
}

func (b *cf1DecisionEventsBridge) acceptDecision(ctx context.Context, raw []byte, event *domainevents.EventEnvelope, payload map[string]any) error {
	select {
	case b.decisionEvents <- append([]byte(nil), raw...):
	default:
		return fmt.Errorf("decision observation buffer is full")
	}
	b.mu.Lock()
	hubURL := b.hubURL
	b.mu.Unlock()
	if hubURL == "" {
		return fmt.Errorf("Notification Hub API is not connected")
	}
	hubEvent := map[string]any{
		"event_id": event.GetEventId(), "event_type": event.GetEventType(),
		"source_scenario": event.GetSource().GetScenario(), "payload": payload,
	}
	responseBody, err := json.Marshal(hubEvent)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(hubURL, "/")+"/api/v1/integrations/events", bytes.NewReader(responseBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("X-Vrooli-Events-Signature", cf1EventSignature(responseBody, b.secret))
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("Notification Hub event webhook returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var accepted struct {
		AskID string `json:"ask_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&accepted); err != nil || accepted.AskID == "" {
		return fmt.Errorf("Notification Hub did not return a durable ask id: %w", err)
	}
	correlation, _ := payload["correlation"].(map[string]any)
	approvalID, _ := correlation["approval_id"].(string)
	if approvalID == "" {
		return fmt.Errorf("Treasury decision event has no approval correlation")
	}
	stringCorrelation := make(map[string]string, len(correlation))
	for key, value := range correlation {
		valueString, ok := value.(string)
		if key == "" || !ok || valueString == "" {
			return fmt.Errorf("Treasury decision correlation contains an empty or non-string field %q", key)
		}
		stringCorrelation[key] = valueString
	}
	b.mu.Lock()
	b.tasksByApproval[approvalID] = accepted.AskID
	b.correlationByApproval[approvalID] = stringCorrelation
	artifactDir := b.artifactDir
	b.mu.Unlock()
	if err := os.WriteFile(filepath.Join(artifactDir, "notification-hub-decision-ask.json"), mustJSON(map[string]any{
		"event_id": event.GetEventId(), "ask_id": accepted.AskID, "approval_id": approvalID,
		"source_event_type": event.GetEventType(), "correlation": correlation,
		"delivery": "signed local Events fixture to real Notification Hub ingress",
	}), 0o600); err != nil {
		return err
	}
	return nil
}

func (b *cf1DecisionEventsBridge) deliverAnswer(ctx context.Context, raw []byte) error {
	var envelope domainevents.EventEnvelope
	if err := protojson.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	var payload structpb.Struct
	if envelope.GetData() == nil || envelope.GetData().UnmarshalTo(&payload) != nil {
		return fmt.Errorf("answer event has no struct payload")
	}
	b.mu.Lock()
	treasuryURL := b.treasuryURL
	artifactDir := b.artifactDir
	b.mu.Unlock()
	if treasuryURL == "" {
		return fmt.Errorf("Treasury API is not connected")
	}
	webhookEvent := map[string]any{
		"event_id": envelope.GetEventId(), "event_type": envelope.GetEventType(),
		"source_scenario": envelope.GetSource().GetScenario(), "payload": payload.AsMap(),
	}
	body, err := json.Marshal(webhookEvent)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(treasuryURL, "/")+"/api/v1/events/notification-hub", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vrooli-Events-Signature", cf1EventSignature(body, b.secret))
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Treasury decision webhook returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	return os.WriteFile(filepath.Join(artifactDir, "treasury-notification-hub-answer-delivered-"+envelope.GetEventId()+".json"), mustJSON(map[string]any{
		"event_id": envelope.GetEventId(), "ask_id": payload.AsMap()["ask_id"],
		"source_event_type": payload.AsMap()["source_event_type"], "treasury_webhook_status": response.StatusCode,
		"treasury_webhook_result": json.RawMessage(responseBody),
	}), 0o600)
}

func (b *cf1DecisionEventsBridge) AskID(approvalID string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tasksByApproval[approvalID]
}

func (b *cf1DecisionEventsBridge) Correlation(approvalID string) map[string]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	correlation := b.correlationByApproval[approvalID]
	copyOfCorrelation := make(map[string]string, len(correlation))
	for key, value := range correlation {
		copyOfCorrelation[key] = value
	}
	return copyOfCorrelation
}

func (b *cf1DecisionEventsBridge) ReconcileAnswer(ctx context.Context, eventsBase, askID string) error {
	// Notification Hub publishes its durable resolution outbox asynchronously
	// after Answer returns. Wait for the real Events publication instead of
	// racing the publisher and mistaking a queued answer for a lost event.
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var eventID string
	for eventID == "" {
		b.mu.Lock()
		eventID = b.answerEventByAsk[askID]
		b.mu.Unlock()
		if eventID != "" {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Notification Hub resolution event for ask %s: %w", askID, ctx.Err())
		case <-deadline.C:
			return fmt.Errorf("Notification Hub did not publish a resolution event for ask %s within 10 seconds", askID)
		case <-ticker.C:
		}
	}
	b.mu.Lock()
	var subscriptionID int64
	for _, subscription := range b.subscriptions {
		if subscription.OwnerScenario == "treasury" && subscription.Name == "treasury-approval-decision-answers" && subscription.Enabled {
			subscriptionID = subscription.ID
			break
		}
	}
	b.mu.Unlock()
	if subscriptionID == 0 {
		return fmt.Errorf("Vrooli Events fixture has no enabled Treasury answer subscription")
	}
	requestBody, err := json.Marshal(map[string]string{"event_id": eventID})
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/subscriptions/%d/deliveries/reconcile", subscriptionID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(eventsBase, "/")+path, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("Vrooli Events answer delivery returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func decisionBridgeSetHub(bridge *cf1DecisionEventsBridge, hubURL string) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	bridge.hubURL = hubURL
}

func decisionBridgeSetTreasury(bridge *cf1DecisionEventsBridge, treasuryURL string) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	bridge.treasuryURL = treasuryURL
}

func answerCF1NotificationHubAsk(ctx context.Context, hubURL, accessAssertion, runDir, approvalID, askID string, expectedCorrelation map[string]string) error {
	if askID == "" {
		return fmt.Errorf("Notification Hub ask id is empty for Treasury approval %s", approvalID)
	}
	client := conversationconnect.NewConversationsServiceClient(http.DefaultClient, hubURL)
	getAsk := func() (*conversationv1.Ask, error) {
		request := connect.NewRequest(&conversationv1.GetAskRequest{AskId: askID})
		request.Header().Set("Cf-Access-Jwt-Assertion", accessAssertion)
		request.Header().Set("X-Vrooli-Test-Mode", "1")
		response, err := client.GetAsk(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg.GetAsk(), nil
	}
	ask, err := getAsk()
	if err != nil {
		return fmt.Errorf("read Notification Hub approval ask: %w", err)
	}
	if ask.GetSource() != "treasury" || ask.GetSourceEventType() != "treasury.approval.decision_requested.v1" || !sameCF1Correlation(ask.GetCorrelation(), expectedCorrelation) || expectedCorrelation["approval_id"] != approvalID {
		return fmt.Errorf("Notification Hub ask does not bind Treasury approval: source=%q type=%q correlation=%v", ask.GetSource(), ask.GetSourceEventType(), ask.GetCorrelation())
	}
	answer := ""
	for _, option := range ask.GetOptions() {
		if strings.EqualFold(option.GetKey(), "approve") || strings.Contains(strings.ToLower(option.GetLabel()), "approve") {
			answer = option.GetKey()
			break
		}
	}
	if answer == "" {
		return fmt.Errorf("Notification Hub ask has no explicit approve option: %v", ask.GetOptions())
	}
	request := connect.NewRequest(&conversationv1.AnswerRequest{AskId: askID, Answer: answer, Note: "synthetic local-fork identity fixture; no live operator action"})
	request.Header().Set("Cf-Access-Jwt-Assertion", accessAssertion)
	request.Header().Set("X-Vrooli-Test-Mode", "1")
	answered, err := client.Answer(ctx, request)
	if err != nil {
		return fmt.Errorf("answer Notification Hub approval ask: %w", err)
	}
	readback, err := getAsk()
	if err != nil {
		return fmt.Errorf("read Notification Hub answer: %w", err)
	}
	if answered.Msg.GetAnswer() != answer || answered.Msg.GetAskId() != askID || readback.GetAnswer() != answer || readback.GetAnsweredByIdentityKind() != "verified_human" || readback.GetAnsweredByIdentitySource() == "" || !sameCF1Correlation(readback.GetCorrelation(), expectedCorrelation) {
		return fmt.Errorf("Notification Hub answer identity/correlation did not read back: answer=%q identity=%q source=%q correlation=%v", readback.GetAnswer(), readback.GetAnsweredByIdentityKind(), readback.GetAnsweredByIdentitySource(), readback.GetCorrelation())
	}
	answerBytes, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(readback)
	if err != nil {
		return fmt.Errorf("marshal Notification Hub answer artifact: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "notification-hub-human-answer.json"), answerBytes, 0o600); err != nil {
		return err
	}
	return nil
}

func sameCF1Correlation(actual, expected map[string]string) bool {
	if len(expected) == 0 || len(actual) != len(expected) {
		return false
	}
	for key, value := range expected {
		if actual[key] != value {
			return false
		}
	}
	return true
}

func waitForCF1ApprovalStatus(ctx context.Context, client authconnect.TreasuryAdminClient, bookID, approvalID string, status treasurysharedpb.ApprovalStatus) (*treasurysharedpb.ApprovalRequest, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		response, err := client.ListApprovals(ctx, operatorRequest(&authpb.ListApprovalsRequest{Status: status, BookId: bookID}))
		if err == nil {
			for _, approval := range response.Msg.GetApprovals() {
				if approval.GetId() == approvalID {
					return approval, nil
				}
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("approval %s did not reach %s before deadline (last read error: %v)", approvalID, status, lastErr)
		case <-ticker.C:
		}
	}
}

func cf1EventSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func mustJSON(value any) []byte {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return raw
}

func startCF1AccessAssertion(t *testing.T) (teamDomain, audience, jwksURL, assertion string, closeServer func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	teamDomain, audience = "https://cf1-local.example.test", "cf1-local-access-audience"
	const keyID = "cf1-local-human"
	public, err := json.Marshal(map[string]any{"keys": []any{map[string]string{
		"kty": "RSA", "kid": keyID, "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(public)
	}))
	encode := func(value any) string {
		raw, marshalErr := json.Marshal(value)
		require.NoError(t, marshalErr)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": keyID}) + "." + encode(map[string]any{
		"iss": teamDomain, "aud": []string{audience}, "exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(), "type": "app", "sub": "cf1-local-human", "email": "cf1-local-human@example.test",
	})
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return teamDomain, audience, server.URL + "/jwks", unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), server.Close
}

type cf4TypedDataRPCSigner struct{ key *ecdsa.PrivateKey }

func (s cf4TypedDataRPCSigner) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var rpc struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if request.Method != http.MethodPost || json.NewDecoder(io.LimitReader(request.Body, 1<<20)).Decode(&rpc) != nil || rpc.Method != "eth_signTypedData_v4" || len(rpc.Params) != 2 {
		http.Error(writer, "invalid local typed-data signing request", http.StatusBadRequest)
		return
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(rpc.Params[1], &typedData); err != nil {
		writeCF1SignerRPCError(writer, rpc.ID, err)
		return
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		writeCF1SignerRPCError(writer, rpc.ID, err)
		return
	}
	signature, err := gethcrypto.Sign(digest, s.key)
	if err != nil {
		http.Error(writer, "local test signature failed", http.StatusInternalServerError)
		return
	}
	signature[64] += 27
	result := "0x" + hex.EncodeToString(signature)
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": rpc.ID, "result": result})
}

func writeCF1SignerRPCError(writer http.ResponseWriter, id json.RawMessage, err error) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": -32602, "message": err.Error()},
	})
}

// cf1RunWalletX402Credential builds a local-only EIP-712 signer for the exact
// mandate budget wallet. It resolves the real run wallet's encrypted CA scalar
// from the isolated Treasury DB/fixture; it never accepts or records a second
// buyer key that could diverge from the wallet funded through Money Ledger.
func cf1RunWalletX402Credential(t *testing.T, runDir, custodyPath, walletID, walletAddress, endpointURL string) (string, error) {
	t.Helper()
	databases, err := filepath.Glob(filepath.Join(runDir, "storage", "data", "vrooli", "treasury_*", "treasury.db"))
	if err != nil || len(databases) != 1 {
		return "", fmt.Errorf("locate exactly one isolated Treasury wallet database: matches=%v err=%v", databases, err)
	}
	readDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databases[0])+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("open isolated Treasury wallet database read-only: %w", err)
	}
	defer readDB.Close()
	var custodyReference, storedAddress string
	if err := readDB.QueryRowContext(context.Background(), `SELECT custody_reference,address FROM research_wallets WHERE id=?`, walletID).Scan(&custodyReference, &storedAddress); err != nil {
		return "", fmt.Errorf("read the run wallet's custody identity: %w", err)
	}
	if !strings.EqualFold(storedAddress, walletAddress) || custodyReference == "" {
		return "", errors.New("isolated Treasury database wallet binding does not match the run wallet address")
	}
	authority, err := credentialauthority.OpenEncryptedFile(custodyPath, "CF2 local fork synthetic custody; never value-bearing")
	if err != nil {
		return "", fmt.Errorf("open isolated encrypted custody for local signer: %w", err)
	}
	identity, err := credentialauthority.ParseIdentity(custodyReference)
	if err != nil {
		return "", fmt.Errorf("parse run wallet CA identity: %w", err)
	}
	privateScalar, err := authority.Resolve(identity, "secp256k1-scalar")
	if err != nil {
		return "", fmt.Errorf("resolve only the isolated run wallet's signer scalar: %w", err)
	}
	key, err := gethcrypto.HexToECDSA(strings.TrimPrefix(privateScalar, "0x"))
	if err != nil {
		return "", fmt.Errorf("decode the isolated run wallet's signer scalar: %w", err)
	}
	if !strings.EqualFold(gethcrypto.PubkeyToAddress(key.PublicKey).Hex(), walletAddress) {
		return "", errors.New("resolved CA signer does not control the mandate budget wallet")
	}
	signer := httptest.NewServer(cf4TypedDataRPCSigner{key: key})
	t.Cleanup(func() {
		signer.Close()
		key.D.SetInt64(0)
	})
	credential, err := json.Marshal(x402rail.Credential{
		EndpointURL: endpointURL, SignerURL: signer.URL, Account: walletAddress,
		Networks: []string{baseusdc.Network},
		Assets:   map[string]x402rail.AssetPolicy{strings.ToLower(baseusdc.USDCAddress): {Decimals: 6, Currency: "USD"}},
	})
	if err != nil {
		return "", fmt.Errorf("encode local wallet-bound x402 credential: %w", err)
	}
	writeRehearsalJSON(t, filepath.Join(runDir, "agent-manager-cli", "x402-signer-binding.json"), map[string]any{
		"wallet_id": walletID, "wallet_address": walletAddress, "endpoint_url": endpointURL,
		"signer_is_local_test_harness": true, "private_key_recorded": false, "credential_saved_only_in_ca": true,
	})
	return string(credential), nil
}

func cf1WalletUSDCBalance(t *testing.T, ctx context.Context, fork *basefork.Fork, address string) *big.Int {
	t.Helper()
	data := "0x70a08231" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
	var raw string
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": baseusdc.USDCAddress, "data": data}, "latest"}, &raw))
	value, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16)
	require.True(t, ok, "USDC balanceOf response must be hexadecimal")
	return value
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
			require.NoError(t, ensureRehearsalWorkspaceSymlink(filepath.Join(repo, entry.Name()), filepath.Join(workspace, entry.Name())))
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

func configureMoneyLedgerRehearsalAuth(t *testing.T, source, jwksURL string) {
	t.Helper()
	manifest := filepath.Join(source, ".vrooli", "service.json")
	var config map[string]any
	readRehearsalJSON(t, manifest, &config)
	components, ok := config["components"].(map[string]any)
	require.True(t, ok, "Money Ledger rehearsal source must declare components")
	api, ok := components["api"].(map[string]any)
	require.True(t, ok, "Money Ledger rehearsal source must declare its API component")
	run, ok := api["run"].(map[string]any)
	require.True(t, ok, "Money Ledger API component must declare run configuration")
	env, _ := run["env"].(map[string]any)
	if env == nil {
		env = make(map[string]any)
	}
	env["VROOLI_AUTH_SCENARIO_AUDIENCE"] = "scenario-authenticator:money-ledger"
	env["VROOLI_AUTH_SCENARIO_JWKS_URL"] = jwksURL
	run["env"] = env
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
}

func ensureRehearsalWorkspaceSymlink(target, link string) error {
	if err := os.Symlink(target, link); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrExist) {
		return err
	}
	current, err := os.Readlink(link)
	if err != nil {
		return fmt.Errorf("existing workspace path %s is not a symlink to %s: %w", link, target, err)
	}
	if filepath.Clean(current) != filepath.Clean(target) {
		return fmt.Errorf("existing workspace symlink %s targets %s, want %s", link, current, target)
	}
	return nil
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
			require.NoError(t, ensureRehearsalWorkspaceSymlink(filepath.Join(repo, entry.Name()), filepath.Join(workspace, entry.Name())))
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
