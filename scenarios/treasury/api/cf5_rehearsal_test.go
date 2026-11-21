//go:build treasury_rehearsal

package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	requestidentity "github.com/vrooli/api-core/identity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	chaincases "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	chaincasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	notificationsshared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	bookconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book/book_v1connect"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	budgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget/budget_v1connect"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
	mandatepb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	mandateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate/mandate_v1connect"
	treasurysharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/shared"
	walletpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	walletconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet/wallet_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/identity"
	x402rail "treasury/internal/rail/x402"
	treasuryresearchbudget "treasury/internal/researchbudget"
	"treasury/internal/testutil/agentclaims"
	"treasury/internal/testutil/basefork"
)

const cf5PinnedBaseForkPort = 18545
const cf5WithdrawalFeeMinor int64 = 10

func TestCF5WithdrawalPlanLeavesNoFundingAfterFee(t *testing.T) {
	amount, atomic, err := cf5WithdrawalPlan(120, cf5WithdrawalFeeMinor)
	require.NoError(t, err)
	require.Equal(t, "1.10", amount)
	require.Equal(t, "1100000", atomic)

	amount, atomic, err = cf5WithdrawalPlan(119, cf5WithdrawalFeeMinor)
	require.NoError(t, err)
	require.Equal(t, "1.09", amount)
	require.Equal(t, "1090000", atomic)

	_, _, err = cf5WithdrawalPlan(10, cf5WithdrawalFeeMinor)
	require.Error(t, err, "fees consuming all available headroom leave no withdrawable amount")
}

func cf5WithdrawalPlan(remainingMinor, feeMinor int64) (string, string, error) {
	if remainingMinor <= 0 || feeMinor < 0 || feeMinor >= remainingMinor {
		return "", "", fmt.Errorf("remaining funding %d minor units must exceed nonnegative fee %d", remainingMinor, feeMinor)
	}
	amountMinor := remainingMinor - feeMinor
	amount := fmt.Sprintf("%d.%02d", amountMinor/100, amountMinor%100)
	atomic := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(10_000)).String()
	return amount, atomic, nil
}

func TestCF5LocalPushSubscriptionFixtureSeedsLeasedSchema(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "notification-hub.db")
	db, err := sql.Open("sqlite", databasePath)
	require.NoError(t, err)
	schema, err := os.ReadFile(filepath.Join("..", "..", "notification-hub", "api", "internal", "recipients", "schema.sql"))
	require.NoError(t, err)
	_, err = db.Exec(string(schema))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	fixture := newCF5WebPushFixture(t)
	seedCF5LocalPushSubscription(t, databasePath, fixture)
	db, err = sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath)+"?mode=ro")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	var recipientCount, deviceCount, channelCount, subscriptionCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM recipients WHERE subject='e1-fixture@notification-hub.invalid'`).Scan(&recipientCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM devices d JOIN recipients r ON r.id=d.recipient_id WHERE r.subject='e1-fixture@notification-hub.invalid'`).Scan(&deviceCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM channel_addresses WHERE device_id='cf5-local-web-push-fixture' AND channel='web_push'`).Scan(&channelCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions p JOIN recipients r ON r.id=p.recipient_id WHERE r.subject='e1-fixture@notification-hub.invalid' AND p.endpoint=? AND p.p256dh=? AND p.auth=?`, fixture.Endpoint, fixture.ClientPublic, fixture.ClientAuth).Scan(&subscriptionCount))
	require.Equal(t, 1, recipientCount)
	require.Equal(t, 1, deviceCount)
	require.Equal(t, 1, channelCount)
	require.Equal(t, 1, subscriptionCount)
}

type cf5WebPushAcceptance struct {
	Endpoint         string `json:"endpoint"`
	AcceptedAt       string `json:"accepted_at"`
	ContentEncoding  string `json:"content_encoding"`
	AuthorizationSet bool   `json:"vapid_authorization_present"`
	CiphertextBytes  int    `json:"ciphertext_bytes"`
	CiphertextSHA256 string `json:"ciphertext_sha256"`
	Error            string `json:"error,omitempty"`
}

type cf5WebPushFixture struct {
	Endpoint     string
	VAPIDPrivate string
	ClientPublic string
	ClientAuth   string
	Accepted     <-chan cf5WebPushAcceptance
}

func newCF5WebPushFixture(t *testing.T) cf5WebPushFixture {
	t.Helper()
	accepted := make(chan cf5WebPushAcceptance, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ciphertext, err := io.ReadAll(r.Body)
		result := cf5WebPushAcceptance{
			Endpoint: r.URL.Path, ContentEncoding: r.Header.Get("Content-Encoding"),
			AuthorizationSet: strings.HasPrefix(r.Header.Get("Authorization"), "vapid t="),
			CiphertextBytes:  len(ciphertext), CiphertextSHA256: fmt.Sprintf("%x", sha256.Sum256(ciphertext)),
		}
		if err != nil {
			result.Error = "could not read encrypted push payload"
			w.WriteHeader(http.StatusBadRequest)
			accepted <- result
			return
		}
		if result.ContentEncoding != "aes128gcm" || !result.AuthorizationSet || result.CiphertextBytes == 0 {
			result.Error = "invalid Web Push envelope"
			w.WriteHeader(http.StatusBadRequest)
			accepted <- result
			return
		}
		result.AcceptedAt = time.Now().UTC().Format(time.RFC3339Nano)
		w.Header().Set("Location", "cf5-local-push-receipt")
		w.WriteHeader(http.StatusCreated)
		accepted <- result
	}))
	t.Cleanup(server.Close)

	vapid, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	vapidScalar := make([]byte, 32)
	vapid.D.FillBytes(vapidScalar)
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	clientAuth := make([]byte, 16)
	_, err = rand.Read(clientAuth)
	require.NoError(t, err)
	return cf5WebPushFixture{
		Endpoint: server.URL + "/push/receipt", VAPIDPrivate: base64.RawURLEncoding.EncodeToString(vapidScalar),
		ClientPublic: base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		ClientAuth:   base64.RawURLEncoding.EncodeToString(clientAuth), Accepted: accepted,
	}
}

// TestCF5MobileLocalForkReceiptWatcherQueuesReceipt is the gated local-fork
// CF5 rehearsal. It attempts BAS mobile captures while isolated Treasury,
// Money Ledger, Notification Hub and a Base fork exercise receipt acceptance,
// exactly-once headroom credit, notice delivery, paid-provider value and an
// allowlisted return. Any API fallback remains labeled separately from UI
// evidence; BAS unavailability must not be reported as a mobile pass.
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
	custodyDir := filepath.Join(runDir, "custody")
	initializeCF2Credentials(t, filepath.Join(custodyDir, "synthetic-custody.enc"))
	t.Cleanup(func() { _ = os.RemoveAll(custodyDir) })
	_, _, agentManagerServiceToken, _, _, moneyLedgerOperatorToken, jwksURL, authenticatorURL, closeIdentity := startServiceIdentityFixture(t)
	t.Cleanup(closeIdentity)
	operatorAuthVariant := instanceName("scenario-authenticator")
	operatorAuthName := "scenario-authenticator@" + operatorAuthVariant
	operatorAuthStorage := filepath.Join(runDir, "operator-auth-storage")
	t.Cleanup(func() {
		if err := os.RemoveAll(operatorAuthStorage); err != nil {
			t.Errorf("remove isolated operator-authenticator account database: %v", err)
		}
	})
	operatorAuthSource := prepareCF5APIScenarioVariant(t, "scenario-authenticator", "../../scenario-authenticator", runDir, map[string]string{
		"VROOLI_STORAGE_ROOT":            operatorAuthStorage,
		"VROOLI_AUTH_RESOURCE_AUDIENCES": "treasury=scenario-authenticator:treasury",
	})
	operatorAuthStarted := lifecycleRehearsal(t, ctx, runDir, "operator-auth-start", map[string]string{}, "start", operatorAuthName, "--path", operatorAuthSource, "--json", "--timeout", "300")
	operatorAuthURL := cf5APIURL(t, operatorAuthStarted)
	operatorAuthStopped := false
	stopOperatorAuth := func() {
		if operatorAuthStopped {
			return
		}
		stopCF5Variant(t, runDir, operatorAuthName, operatorAuthURL)
		operatorAuthStopped = true
	}
	t.Cleanup(stopOperatorAuth)
	requireHTTPStatus(t, operatorAuthURL+"/health", http.StatusOK)
	operatorEmail := "cf5-operator-" + strings.TrimPrefix(operatorAuthVariant, "cf5-") + "@example.test"
	operatorPasswordBytes := make([]byte, 24)
	_, err := rand.Read(operatorPasswordBytes)
	require.NoError(t, err)
	operatorPassword := "Cf5!" + base64.RawURLEncoding.EncodeToString(operatorPasswordBytes)
	clear(operatorPasswordBytes)
	operatorAccounts := accountsconnect.NewAccountsServiceClient(&http.Client{Timeout: 10 * time.Second}, operatorAuthURL)
	registeredOperator, err := operatorAccounts.Register(ctx, connect.NewRequest(&accountsv1.RegisterRequest{
		Email: operatorEmail, Password: operatorPassword, Username: "CF5 isolated rehearsal operator", Resource: "treasury",
	}))
	require.NoError(t, err, "register a disposable human only in the isolated scenario-authenticator variant")
	operatorSubject := registeredOperator.Msg.GetAccount().GetId()
	registeredOperator = nil // Do not retain the access/refresh token response from account registration.
	ledgerVariant := instanceName("money-ledger")
	ledgerName := "money-ledger@" + ledgerVariant
	ledgerEnv := map[string]string{
		"VROOLI_STORAGE_ROOT": filepath.Join(runDir, "money-ledger-storage"),
		"GOFLAGS":             "", "API_TOKEN": "",
		"VROOLI_AUTH_SCENARIO_JWKS_URL":  jwksURL,
		"SCENARIO_AUTHENTICATOR_API_URL": authenticatorURL,
	}
	ownerBindings, err := json.Marshal([]map[string]any{{
		"source": "scenario_authenticator", "subject": "cf5-local-operator", "canonical_subject": "cf5-local-operator",
		"scopes": []string{"money-ledger:write"}, "resources": []string{"money-ledger"},
	}})
	require.NoError(t, err)
	ledgerEnv["VROOLI_AUTH_OWNER_BINDINGS"] = string(ownerBindings)
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
	createdBook, err := ledgerBooks.CreateBook(ctx, cf5ServiceRequest(&ledgerpb.CreateBookRequest{Name: "CF5 isolated Base funding", Currency: "USD"}, moneyLedgerOperatorToken))
	require.NoError(t, err)
	createdAccount, err := ledgerBooks.CreateAccount(ctx, cf5ServiceRequest(&ledgerpb.CreateAccountRequest{BookId: createdBook.Msg.GetBook().GetId(), Name: "Base USDC deposits", AccountKind: ledgerpb.AccountKind_ASSET}, moneyLedgerOperatorToken))
	require.NoError(t, err)
	writeCF5JSON(t, filepath.Join(runDir, "money-ledger-fixture.json"), map[string]string{
		"variant": ledgerVariant, "api_url": ledgerURL, "book_id": createdBook.Msg.GetBook().GetId(), "account_id": createdAccount.Msg.GetAccount().GetId(), "service_identity": "treasury / money-ledger:write via local synthetic JWKS fixture",
	})
	var runIdentityMu sync.RWMutex
	activeRunID := "test-run"
	var budgetEvidenceMu sync.Mutex
	budgetEvidence := map[string][]byte{}
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
		if strings.HasPrefix(r.URL.Path, "/api/v1/research-budgets/") {
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+agentManagerServiceToken {
				http.Error(w, "Treasury Agent Manager service identity required", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/research-budgets/"), "/")
			if len(parts) != 2 || (parts[1] != "snapshot" && parts[1] != "settlements") {
				http.NotFound(w, r)
				return
			}
			if _, err := uuid.Parse(parts[0]); err != nil {
				http.Error(w, "valid run UUID required", http.StatusBadRequest)
				return
			}
			payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil || len(payload) == 0 || !json.Valid(payload) {
				http.Error(w, "invalid run-bound evidence", http.StatusBadRequest)
				return
			}
			key := parts[0] + "/" + parts[1]
			if parts[1] == "snapshot" {
				var snapshot treasuryresearchbudget.Snapshot
				if json.Unmarshal(payload, &snapshot) != nil || snapshot.TreasuryBudgetID == "" || snapshot.MandateID == "" || snapshot.AgentSubject == "" || snapshot.CapMinor < 0 || snapshot.Currency == "" || snapshot.UnitCode == "" || snapshot.PolicyRevision == "" || snapshot.CreatedAt.IsZero() {
					http.Error(w, "invalid Treasury run snapshot", http.StatusBadRequest)
					return
				}
			} else {
				var settlement treasuryresearchbudget.Settlement
				if json.Unmarshal(payload, &settlement) != nil || settlement.ID == "" || settlement.SettlementID == "" || settlement.TreasuryReceiptID == "" || settlement.IntegrationID == "" || settlement.Currency == "" || settlement.UnitCode == "" || settlement.AmountMinor < 0 || settlement.SettledAt.IsZero() {
					http.Error(w, "settlement evidence id required", http.StatusBadRequest)
					return
				}
				key += "/" + settlement.SettlementID
			}
			budgetEvidenceMu.Lock()
			if prior, exists := budgetEvidence[key]; exists && !bytes.Equal(prior, payload) {
				budgetEvidenceMu.Unlock()
				http.Error(w, "conflicting immutable run evidence", http.StatusConflict)
				return
			}
			budgetEvidence[key] = append([]byte(nil), payload...)
			budgetEvidenceMu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
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
	webPush := newCF5WebPushFixture(t)
	hubInstance := instanceName("notification-hub")
	hubName := "notification-hub@" + hubInstance
	hubEnv := map[string]string{
		// A variant resolves its storage under ~/.vrooli unless told otherwise, and
		// nothing removes it: each run left a notification-hub_cf5-* directory
		// there. Like the Treasury and Offer Desk variants, keep it in the run.
		"VROOLI_STORAGE_ROOT":           filepath.Join(runDir, "notification-hub-storage"),
		"VROOLI_TEST_MODE_FORCE_ENABLE": "1", "AGENT_MANAGER_API_URL": identityServer.URL,
		"VROOLI_NOTIFICATION_RECIPIENT": "cf5-operator",
	}
	hubSource := prepareCF5APIScenarioVariant(t, "notification-hub", "../../notification-hub", runDir, hubEnv)
	hubStarted := lifecycleRehearsal(t, ctx, runDir, "notification-hub-start", map[string]string{
		"API_TOKEN": "", "VROOLI_TEST_MODE_FORCE_ENABLE": "1", "VROOLI_WEBPUSH_VAPID_PRIVATE_KEY": webPush.VAPIDPrivate,
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
	seedCF5LocalPushSubscription(t, hubLeaseDB, webPush)
	writeCF5JSON(t, filepath.Join(runDir, "notification-hub-local-push-subscription.json"), map[string]any{
		"endpoint_host": "127.0.0.1", "endpoint_path": "/push/receipt", "recipient": "e1-fixture@notification-hub.invalid",
		"fixture_only": true, "seeded_in_isolated_leased_database": true, "external_credentials": false, "phone_delivery_claimed": false,
	})
	offerDeskEvidenceDir := filepath.Join(runDir, "cf5-offer-desk")
	require.NoError(t, os.MkdirAll(offerDeskEvidenceDir, 0o700))
	agentManagerEvidenceDir := filepath.Join(runDir, "agent-manager-cli")
	require.NoError(t, os.MkdirAll(agentManagerEvidenceDir, 0o700))
	agentManagerEvidenceInfo, err := os.Lstat(agentManagerEvidenceDir)
	require.NoError(t, err)
	require.True(t, agentManagerEvidenceInfo.IsDir() && agentManagerEvidenceInfo.Mode()&os.ModeSymlink == 0, "paid-body evidence child must be a real directory")
	require.Equal(t, os.FileMode(0o700), agentManagerEvidenceInfo.Mode().Perm(), "paid-body evidence child must be private")
	agentManagerEvidenceStat, ok := agentManagerEvidenceInfo.Sys().(*syscall.Stat_t)
	require.True(t, ok, "paid-body evidence child must expose its Linux device/inode identity")
	offerDeskSource := prepareCF5APIScenarioVariant(t, "offer-desk", "../../offer-desk", runDir, map[string]string{
		"VROOLI_STORAGE_ROOT":                      filepath.Join(runDir, "offer-desk-storage"),
		"TREASURY_API_URL":                         "",
		"OFFER_DESK_TEST_DROP_FIRST_PAID_RESPONSE": "1",
	})
	disableCF4OfferDeskVectorSearch(t, offerDeskSource)
	offerDeskName := "offer-desk@" + instanceName("cf5-offer-desk")
	offerDeskStarted := lifecycleRehearsal(t, ctx, runDir, "offer-desk-start", map[string]string{}, "start", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	offerDeskURL := cf5APIURL(t, offerDeskStarted)
	offerDeskStopped := false
	stopOfferDesk := func() {
		if offerDeskStopped {
			return
		}
		stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
		offerDeskStopped = true
	}
	t.Cleanup(stopOfferDesk)
	offerID, offerVariantID := seedCF4OfferDeskGraph(t, ctx, offerDeskURL, offerDeskEvidenceDir)
	paidCatalogURL := strings.TrimRight(offerDeskURL, "/") + "/api/v1/paid/catalog"
	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	cf5ProvisionPinnedFacilitatorKey(t, facilitatorKey)
	cf5StartPinnedFacilitator(t, ctx, runDir)

	instance := instanceName("treasury")
	name := "treasury@" + instance
	apiEnv := map[string]string{
		"VROOLI_STORAGE_ROOT":                                 filepath.Join(runDir, "storage"),
		"TREASURY_REHEARSAL_DIR":                              filepath.Join(runDir, "custody"),
		"TREASURY_BASE_RPC_URL":                               forkIdentity.URL,
		"TREASURY_BASE_USDC_MODE":                             "local-fork",
		"X402_RUNTIME_MODE":                                   "local-fork",
		"MONEY_LEDGER_API_URL":                                ledgerURL,
		"TREASURY_LEDGER_BOOK_ID":                             createdBook.Msg.GetBook().GetId(),
		"TREASURY_LEDGER_ACCOUNT_ID":                          createdAccount.Msg.GetAccount().GetId(),
		"SCENARIO_AUTHENTICATOR_API_URL":                      authenticatorURL,
		"TREASURY_OPERATOR_AUTH_URL":                          operatorAuthURL,
		"TREASURY_OPERATOR_JWKS_URL":                          operatorAuthURL + "/.well-known/jwks.json",
		"TREASURY_OPERATOR_HUMAN_SUBJECTS":                    operatorSubject,
		"TREASURY_X402_PAID_BODY_EVIDENCE_DIR":                agentManagerEvidenceDir,
		"TREASURY_X402_PAID_BODY_EVIDENCE_ROOT":               runDir,
		"TREASURY_X402_PAID_BODY_EVIDENCE_DIR_DEV":            fmt.Sprintf("%d", agentManagerEvidenceStat.Dev),
		"TREASURY_X402_PAID_BODY_EVIDENCE_DIR_INO":            fmt.Sprintf("%d", agentManagerEvidenceStat.Ino),
		"TREASURY_CF5_REHEARSAL":                              "1",
		"AGENT_MANAGER_API_URL":                               identityServer.URL,
		"NOTIFICATION_HUB_API_URL":                            hubAPIURL,
		"TREASURY_CF5_NOTIFICATION_TEST_MODE":                 "1",
		"TREASURY_CF5_NOTIFICATION_TEST_AGENT_IDENTITY_TOKEN": "cf5-local-agent",
		"OFFER_DESK_BASE_URL":                                 offerDeskURL,
		"X402_FACILITATOR_URL":                                "http://127.0.0.1:14021",
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
	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	sessionHTTP := &http.Client{Jar: cookieJar, Timeout: 10 * time.Second}
	signInBody, err := json.Marshal(map[string]string{"email": operatorEmail, "password": operatorPassword})
	require.NoError(t, err)
	signInRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, uiURL+"/api/v1/operator-session/sign-in", strings.NewReader(string(signInBody)))
	require.NoError(t, err)
	signInRequest.Header.Set("Content-Type", "application/json")
	signInRequest.Header.Set("Origin", uiURL)
	signInResponse, err := sessionHTTP.Do(signInRequest)
	require.NoError(t, err)
	signInResponseBody, err := io.ReadAll(signInResponse.Body)
	require.NoError(t, err)
	require.NoError(t, signInResponse.Body.Close())
	require.Equalf(t, http.StatusOK, signInResponse.StatusCode, "isolated operator sign-in response: %s", strings.TrimSpace(string(signInResponseBody)))
	require.NotContains(t, string(signInResponseBody), "access_token")
	require.NotContains(t, string(signInResponseBody), "refresh_token")
	setCookies := signInResponse.Cookies()
	require.NotEmpty(t, setCookies)
	for _, cookie := range setCookies {
		require.True(t, cookie.HttpOnly, "Treasury operator cookies must be HttpOnly")
	}
	statusRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, uiURL+"/api/v1/operator-session", nil)
	require.NoError(t, err)
	statusResponse, err := sessionHTTP.Do(statusRequest)
	require.NoError(t, err)
	var sessionStatus struct {
		Authenticated bool   `json:"authenticated"`
		Subject       string `json:"subject"`
	}
	require.NoError(t, json.NewDecoder(statusResponse.Body).Decode(&sessionStatus))
	require.NoError(t, statusResponse.Body.Close())
	require.Equal(t, http.StatusOK, statusResponse.StatusCode)
	require.True(t, sessionStatus.Authenticated)
	require.Equal(t, operatorSubject, sessionStatus.Subject)
	writeCF5JSON(t, filepath.Join(runDir, "treasury-browser-session.json"), map[string]any{
		"surface_origin": uiURL, "authenticated": true, "verified_human_subject": sessionStatus.Subject,
		"http_only_cookies": true, "token_response_fields": false,
		"subject_to_operator_binding": "isolated allowlist maps this verified human to the existing Treasury local-operator realm",
	})
	clear(operatorPasswordBytes)
	setCF5VariantEnvironment(t, offerDeskSource, map[string]string{"TREASURY_API_URL": apiURL})
	offerDeskRebound := lifecycleRehearsal(t, ctx, runDir, "offer-desk-bind-treasury", map[string]string{}, "restart", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	require.Equal(t, offerDeskURL, cf5APIURL(t, offerDeskRebound), "Offer Desk restart must preserve its isolated API address")
	cf5Price := cf5DeclareOfferDeskPriceWithSession(t, ctx, uiURL, sessionHTTP, map[string]any{
		"offer_id": offerID, "variant_id": offerVariantID,
		"price": map[string]any{
			"id": "cf4-research-inference", "resource_url": paidCatalogURL,
			"description": "Offer Desk research catalog lookup", "mime_type": "application/json",
			"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
			"asset": baseusdc.USDCAddress, "asset_decimals": 6, "max_timeout_seconds": 300,
			"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
		},
	})
	require.Equal(t, int64(1), cf5Price.AmountMinor)
	require.Equal(t, "10000", cf5Price.Amount)
	writeCF5JSON(t, filepath.Join(offerDeskEvidenceDir, "priced-call.json"), cf5Price)
	const bookID = "cf5-integrated-research-book"
	ledgerBookID := createdBook.Msg.GetBook().GetId()
	ledgerAccountID := createdAccount.Msg.GetAccount().GetId()
	operator := authorizationconnect.NewTreasuryAdminClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	walletAdmin := walletconnect.NewWalletAdminServiceClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	bookAdmin := bookconnect.NewBookAdminServiceClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	mandateAdmin := mandateconnect.NewMandateAdminServiceClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	budgetAdmin := budgetconnect.NewBudgetAdminServiceClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	agent := authorizationconnect.NewAgentSpendClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	readHeadroom := func(budgetID string) *authorizationv1.GetBudgetHeadroomResponse {
		req := connect.NewRequest(&authorizationv1.GetBudgetHeadroomRequest{BudgetId: budgetID})
		req.Header().Set(identity.HeaderAgentIdentityToken, "cf5-local-agent")
		response, readErr := agent.GetBudgetHeadroom(ctx, req)
		require.NoError(t, readErr)
		return response.Msg
	}
	_, err = bookAdmin.CreateBook(ctx, cf5OperatorRequest(&bookv1.CreateBookRequest{Book: &bookv1.Book{Id: bookID, Name: "CF5 isolated local fork"}}))
	require.NoError(t, err)
	fundingIntents := fundingconnect.NewFundingAdminServiceClient(&http.Client{Timeout: 10 * time.Second}, apiURL)
	variant := fmt.Sprintf("cf5-integrated-%d", time.Now().UnixNano())
	budgetID := "cf1-agent-manager-budget-" + variant
	mandateID := "cf1-agent-manager-mandate-" + variant
	providerURL, err := url.ParseRequestURI(paidCatalogURL)
	require.NoError(t, err)
	x402 := &cf1X402ShowcaseConfig{
		CatalogURL: paidCatalogURL, PriceID: cf5Price.ID, CallID: "cf4-research-inference-call",
		Counterparty: strings.ToLower(providerURL.Hostname()), InstrumentID: "cf1-agent-manager-instrument-x402-cf5-" + variant,
		PrepareSigner: func(_ context.Context, actualWalletID, actualAddress string) (string, error) {
			return cf1RunWalletX402Credential(t, runDir, filepath.Join(custodyDir, "synthetic-custody.enc"), actualWalletID, actualAddress, paidCatalogURL)
		},
	}
	var runID, walletID, walletAddress, fundingTxHash string
	var beforeCredit, afterCredit *authorizationv1.GetBudgetHeadroomResponse
	var basCaptureError string
	var basFundAuthenticatedCaptured bool
	notificationClient := notificationsconnect.NewNotificationsServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	hooks := cf1ShowcaseHooks{
		FundingAtomicAmount: "2600000",
		X402:                x402,
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
			cf5RecoverBudgetRootBeforeFunding(t, fundCtx, runDir, walletAdmin, bookAdmin, bookID, custodyDir)
			writeCF5JSON(t, filepath.Join(runDir, "budget-wallet.json"), map[string]any{
				"book_id": bookID, "budget_id": budgetID, "mandate_id": mandateID, "run_id": runID,
				"wallet_id": walletID, "local_fork_address": walletAddress,
				"ledger_book_id": ledgerBookID, "ledger_account_id": ledgerAccountID,
			})
			if captureErr := captureCF5MobileFund(t, fundCtx, runDir, uiURL, operatorEmail, operatorPassword, budgetID, walletAddress); captureErr != nil {
				basCaptureError = captureErr.Error()
				writeCF5JSON(t, filepath.Join(runDir, "bas-mobile-fund-capture-failed.json"), map[string]any{
					"error": basCaptureError, "bas_artifact_created": false,
					"fund_surface_verified": false, "authenticated_funding_flow_verified": false,
				})
			} else {
				basFundAuthenticatedCaptured = true
				writeCF5JSON(t, filepath.Join(runDir, "bas-mobile-authenticated-fund-link.json"), map[string]any{
					"fund_surface_verified": true, "authenticated_funding_flow_verified": true,
					"budget_id": budgetID, "atomic_amount": "2600000", "wallet_address": walletAddress,
					"exact_prefilled_base_usdc_uri_asserted": true, "mobile_wallet_app_opened": false,
				})
			}
			operatorPassword = ""
			intents, listErr := fundingIntents.ListFundingIntents(fundCtx, cf5OperatorRequest(&fundingpb.ListFundingIntentsRequest{BudgetId: budgetID}))
			if listErr != nil {
				return listErr
			}
			if len(intents.Msg.GetIntents()) == 0 && basCaptureError != "" {
				fallback, createErr := fundingIntents.CreateFundingIntent(fundCtx, cf5OperatorRequest(&fundingpb.CreateFundingIntentRequest{
					Id: "cf5-mobile-api-fallback-" + variant, BudgetId: budgetID, AtomicAmount: "2600000",
					Valuation: &ingestpb.CryptoValuation{AmountMinor: 260, Currency: "USD", BasisReference: "CF5 isolated local-fork rehearsal"},
				}))
				if createErr != nil {
					return fmt.Errorf("BAS failed and the Treasury funding API fallback also failed: %w", createErr)
				}
				writeCF5JSON(t, filepath.Join(runDir, "bas-mobile-fund-api-fallback.json"), map[string]any{
					"intent_id": fallback.Msg.GetIntent().GetId(), "created_by": "authenticated Treasury API in isolated local-fork test",
					"reason": basCaptureError, "mobile_ui_action_claimed": false,
				})
				intents, listErr = fundingIntents.ListFundingIntents(fundCtx, cf5OperatorRequest(&fundingpb.ListFundingIntentsRequest{BudgetId: budgetID}))
				if listErr != nil {
					return listErr
				}
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
				read, readErr := fundingIntents.GetFundingIntent(fundCtx, cf5OperatorRequest(&fundingpb.FundingIntentRequest{Id: intent.GetId()}))
				if readErr == nil && read.Msg.GetIntent().GetTransactionHash() != "" && read.Msg.GetIntent().GetState() == "credited" {
					observed = read.Msg.GetIntent()
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			if observed == nil || observed.GetTransactionHash() != fundingTxHash || observed.GetNotificationState() != "sent" {
				return fmt.Errorf("run wallet funding watcher did not credit and notify: %+v", observed)
			}
			delivery, deliveryErr := fundingIntents.GetReceiptFunding(fundCtx, cf5OperatorRequest(&fundingpb.GetReceiptFundingRequest{Id: observed.GetReceiptDeliveryId()}))
			if deliveryErr != nil {
				return deliveryErr
			}
			if delivery.Msg.GetDelivery().GetState() != fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED || delivery.Msg.GetDelivery().GetReceipt().GetAtomicDelta() != "2600000" || delivery.Msg.GetDelivery().GetReceipt().GetValuationDeltaMinor() != 260 {
				return fmt.Errorf("Money Ledger did not accept the exact run-wallet receipt and declared value: %+v", delivery.Msg.GetDelivery())
			}
			writeRehearsalProto(t, filepath.Join(runDir, "money-ledger-accepted-receipt.json"), delivery.Msg)
			var matched *notificationsshared.Notification
			var delivered *connect.Response[notificationsv1.GetResponse]
			pushDeadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(pushDeadline) {
				notificationReq := connect.NewRequest(&notificationsv1.ListRequest{Limit: 20})
				notificationReq.Header().Set("X-Vrooli-Test-Mode", "1")
				notificationList, noticeErr := notificationClient.List(fundCtx, notificationReq)
				if noticeErr != nil {
					return noticeErr
				}
				var matches []*notificationsshared.Notification
				for _, notice := range notificationList.Msg.GetNotifications() {
					if notice.GetDedupeKey() == delivery.Msg.GetDelivery().GetReceipt().GetId() {
						matches = append(matches, notice)
					}
				}
				if len(matches) > 1 {
					return fmt.Errorf("Notification Hub retained duplicate accepted-receipt notices: %d", len(matches))
				}
				if len(matches) == 1 {
					matched = matches[0]
					if matched.GetState() == notificationsshared.NotificationState_NOTIFICATION_STATE_DELIVERED {
						readbackReq := connect.NewRequest(&notificationsv1.GetRequest{Id: matched.GetId()})
						readbackReq.Header().Set("X-Vrooli-Test-Mode", "1")
						delivered, noticeErr = notificationClient.Get(fundCtx, readbackReq)
						if noticeErr != nil {
							return noticeErr
						}
						if len(delivered.Msg.GetReceipts()) > 0 {
							break
						}
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
			if matched == nil || !strings.Contains(matched.GetBody(), "credited to budget "+budgetID) ||
				matched.GetState() != notificationsshared.NotificationState_NOTIFICATION_STATE_DELIVERED || delivered == nil || len(delivered.Msg.GetReceipts()) != 1 {
				return fmt.Errorf("Notification Hub did not deliver one accepted-receipt notice: notification=%v receipt_count=%d", matched, func() int {
					if delivered == nil {
						return 0
					}
					return len(delivered.Msg.GetReceipts())
				}())
			}
			pushAcceptance := <-webPush.Accepted
			if pushAcceptance.Error != "" || pushAcceptance.CiphertextBytes == 0 {
				return fmt.Errorf("local Web Push endpoint did not accept the encrypted notification: %+v", pushAcceptance)
			}
			providerID, providerErr := cf5LocalPushReceiptProviderID(hubLeaseDB, matched.GetId())
			if providerErr != nil {
				return fmt.Errorf("read provider receipt from isolated Notification Hub lease: %w", providerErr)
			}
			if providerID != "cf5-local-push-receipt" {
				return fmt.Errorf("isolated Notification Hub receipt provider ID %q does not match local fixture", providerID)
			}
			acceptedTimestamp := delivery.Msg.GetDelivery().GetAcceptedAt()
			if acceptedTimestamp == nil {
				return fmt.Errorf("accepted receipt has no acceptance timestamp")
			}
			acceptedAt := acceptedTimestamp.AsTime()
			pushAcceptedAt, parseErr := time.Parse(time.RFC3339Nano, pushAcceptance.AcceptedAt)
			if parseErr != nil {
				return fmt.Errorf("parse local push acceptance time: %w", parseErr)
			}
			latency := pushAcceptedAt.Sub(acceptedAt)
			if latency < 0 || latency > 5*time.Second {
				return fmt.Errorf("accepted-receipt notification reached the local push endpoint after %s; limit is 5s", latency)
			}
			writeRehearsalProto(t, filepath.Join(runDir, "notification-hub-readback.json"), matched)
			writeRehearsalProto(t, filepath.Join(runDir, "notification-hub-delivery-receipts.json"), delivered.Msg)
			writeCF5JSON(t, filepath.Join(runDir, "notification-hub-local-push-acceptance.json"), map[string]any{
				"recipient": matched.GetRequestedBy(), "receipt_funding_accepted_at": acceptedAt.UTC().Format(time.RFC3339Nano),
				"push_accepted_at": pushAcceptedAt.UTC().Format(time.RFC3339Nano), "latency_ms": latency.Milliseconds(),
				"provider_receipt_id": providerID, "content_encoding": pushAcceptance.ContentEncoding,
				"ciphertext_bytes": pushAcceptance.CiphertextBytes, "ciphertext_sha256": pushAcceptance.CiphertextSHA256,
				"local_test_endpoint": true, "phone_delivery_claimed": false,
			})
			afterCredit = readHeadroom(budgetID)
			if afterCredit.GetHeadroom().GetFundingCreditedMinor() != 260 || afterCredit.GetHeadroom().GetFundingRemainingMinor() != 260 || afterCredit.GetHeadroom().GetFundingAtomicAmount() != "2600000" {
				return fmt.Errorf("accepted receipt did not become exact Treasury headroom: %+v", afterCredit.GetHeadroom())
			}
			writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-credit.json"), afterCredit)
			writeCF5JSON(t, filepath.Join(runDir, "bas-mobile-receipt-capture-skipped.json"), map[string]string{
				"reason": "no disposable verified-human browser session is configured; receipt UI remains unverified",
			})
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
	}
	repo, err := filepath.Abs(filepath.Join(".", "../../.."))
	require.NoError(t, err)
	// Keep the existing dropped-response recovery proof and measured useful
	// paid call inside the primary run, while a second ordinary run is active
	// with its own budget wallet and reciprocal refusal probes.
	pairedBarrierDir := filepath.Join(runDir, "cf1-paired-probes")
	pairedIdentityPath := filepath.Join(runDir, "agent-manager-cross-budget-identities.json")
	pairedRuns, pairErr := newCF1PairedRunHarness(apiURL, pairedIdentityPath, pairedBarrierDir)
	require.NoError(t, pairErr)
	hooks.PairedRun = pairedRuns
	secondCF1RunDir := filepath.Join(runDir, "cf1-second-budget-run")
	require.NoError(t, os.MkdirAll(filepath.Join(secondCF1RunDir, "custody"), 0o700))
	secondVariant := variant + "-second-budget"
	secondHooks := cf1ShowcaseHooks{PairedRun: pairedRuns, TreasuryCredentialPath: filepath.Join(secondCF1RunDir, "custody", "synthetic-custody.enc"), AwaitApprovalNotice: hooks.AwaitApprovalNotice}
	t.Setenv("TREASURY_CF1_X402_MEASURED_SECOND_CALL", "1")
	type runResult struct {
		runID, authorizationID, settlementID string
		err                                  error
	}
	firstResult, secondResult := make(chan runResult, 1), make(chan runResult, 1)
	go func() {
		result := runResult{}
		passed := t.Run("CF5 primary ordinary Agent Manager run", func(runT *testing.T) {
			result.runID, result.authorizationID, result.settlementID, result.err = runCF1AgentManagerCLIForkShowcase(runT, ctx, repo, runDir, apiURL, identityServer.URL, source, name, operator, nil, fork, bookID, variant, hooks)
		})
		if !passed && result.err == nil {
			result.err = errors.New("primary ordinary-run subtest failed through a test assertion")
		}
		firstResult <- result
	}()
	go func() {
		result := runResult{}
		passed := t.Run("CF5 counterpart ordinary Agent Manager run", func(runT *testing.T) {
			result.runID, result.authorizationID, result.settlementID, result.err = runCF1AgentManagerCLIForkShowcase(runT, ctx, repo, secondCF1RunDir, apiURL, identityServer.URL, source, name, operator, nil, fork, bookID, secondVariant, secondHooks)
		})
		if !passed && result.err == nil {
			result.err = errors.New("counterpart ordinary-run subtest failed through a test assertion")
		}
		secondResult <- result
	}()
	var first, second runResult
	select {
	case first = <-firstResult:
	case <-ctx.Done():
		require.FailNow(t, "paired primary Agent Manager run did not complete", "%v", ctx.Err())
	}
	if first.err != nil {
		cancel()
		require.NoError(t, first.err)
	}
	select {
	case second = <-secondResult:
	case <-ctx.Done():
		require.FailNow(t, "paired counterpart Agent Manager run did not complete", "%v", ctx.Err())
	}
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	cf1RunID, cf1AuthorizationID, cf1SettlementID := first.runID, first.authorizationID, first.settlementID
	require.NotEmpty(t, cf1RunID)
	require.NotEmpty(t, cf1AuthorizationID)
	require.NotEmpty(t, cf1SettlementID)
	require.Equal(t, runID, cf1RunID)
	require.NotEmpty(t, second.runID)
	require.NotEqual(t, cf1RunID, second.runID)
	var pairedEvidence cf1PairedIdentityEvidence
	pairedEvidenceBytes, err := os.ReadFile(pairedIdentityPath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(pairedEvidenceBytes, &pairedEvidence))
	require.Len(t, pairedEvidence.Agents, 2)
	var primaryPairRow, counterpartPairRow *struct {
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
	for index := range pairedEvidence.Agents {
		row := &pairedEvidence.Agents[index]
		switch row.RunID {
		case cf1RunID:
			primaryPairRow = row
		case second.runID:
			counterpartPairRow = row
		}
	}
	require.NotNil(t, primaryPairRow)
	require.NotNil(t, counterpartPairRow)
	require.Equal(t, budgetID, primaryPairRow.BudgetID)
	require.Equal(t, mandateID, primaryPairRow.MandateID)
	require.Equal(t, "cf1-agent-manager-budget-"+secondVariant, counterpartPairRow.BudgetID)
	require.Equal(t, "cf1-agent-manager-mandate-"+secondVariant, counterpartPairRow.MandateID)
	require.NotEqual(t, primaryPairRow.WalletID, counterpartPairRow.WalletID)
	require.NotEqual(t, strings.ToLower(primaryPairRow.WalletAddress), strings.ToLower(counterpartPairRow.WalletAddress))
	for _, identity := range pairedEvidence.Agents {
		require.Equal(t, "permission_denied", identity.ForeignWalletProbeStatus)
		require.Equal(t, "permission_denied", identity.ForeignMandateProbeStatus)
	}
	writeRehearsalJSON(t, filepath.Join(runDir, "cf1-two-ordinary-budget-runs.json"), map[string]any{
		"agents": pairedEvidence.Agents, "reciprocal_foreign_wallet_and_mandate_denials": true,
		"distinct_run_bound_budgets": true, "distinct_mandates_wallets_and_addresses": true,
		"shared_identity_path": pairedIdentityPath, "paired_probe_barrier_dir": pairedBarrierDir,
	})
	budgetEvidenceMu.Lock()
	snapshotEvidence := append([]byte(nil), budgetEvidence[cf1RunID+"/snapshot"]...)
	x402SettlementID := "cf1-agent-manager-settlement-x402-" + variant
	x402SettlementEvidence := append([]byte(nil), budgetEvidence[cf1RunID+"/settlements/"+x402SettlementID]...)
	measuredSettlementID := x402SettlementID + "-measured"
	measuredSettlementEvidence := append([]byte(nil), budgetEvidence[cf1RunID+"/settlements/"+measuredSettlementID]...)
	budgetEvidenceMu.Unlock()
	require.NotEmpty(t, snapshotEvidence, "the local Agent Manager receiver must retain the authenticated run snapshot")
	require.NotEmpty(t, x402SettlementEvidence, "the same x402 settlement retry must publish immutable run-budget evidence")
	var acceptedBudgetSnapshot treasuryresearchbudget.Snapshot
	require.NoError(t, json.Unmarshal(snapshotEvidence, &acceptedBudgetSnapshot))
	require.Equal(t, budgetID, acceptedBudgetSnapshot.TreasuryBudgetID)
	require.Equal(t, mandateID, acceptedBudgetSnapshot.MandateID)
	require.Equal(t, "cf1-local-operator", acceptedBudgetSnapshot.AgentSubject)
	var acceptedX402Settlement treasuryresearchbudget.Settlement
	require.NoError(t, json.Unmarshal(x402SettlementEvidence, &acceptedX402Settlement))
	require.Equal(t, x402SettlementID, acceptedX402Settlement.SettlementID)
	require.EqualValues(t, 1, acceptedX402Settlement.AmountMinor)
	require.NotEmpty(t, measuredSettlementEvidence, "the separate successful paid call must publish immutable run-budget evidence")
	var acceptedMeasuredSettlement treasuryresearchbudget.Settlement
	require.NoError(t, json.Unmarshal(measuredSettlementEvidence, &acceptedMeasuredSettlement))
	require.Equal(t, measuredSettlementID, acceptedMeasuredSettlement.SettlementID)
	require.EqualValues(t, 1, acceptedMeasuredSettlement.AmountMinor)
	require.Equal(t, "measured", acceptedMeasuredSettlement.BenefitState, "paid response must prove useful provider output")
	var measuredResult struct {
		Status          string `json:"status"`
		SettlementID    string `json:"settlement_id"`
		AmountMinor     int64  `json:"amount_minor"`
		PriceID         string `json:"price_id"`
		CallID          string `json:"call_id"`
		BodyPath        string `json:"body_path"`
		ResponseBytes   int    `json:"response_bytes"`
		ResponseSHA256  string `json:"response_sha256"`
		LedgerEmission  string `json:"ledger_emission_id"`
		TreasuryReceipt string `json:"receipt_id"`
		BenefitState    string `json:"benefit_state"`
	}
	measuredResultPath := filepath.Join(agentManagerEvidenceDir, "x402-measured-result.json")
	measuredResultBytes, err := os.ReadFile(measuredResultPath)
	require.NoError(t, err, "Agent Manager must return measured paid-result evidence")
	require.NoError(t, json.Unmarshal(measuredResultBytes, &measuredResult))
	require.Equal(t, "settled", measuredResult.Status)
	require.Equal(t, measuredSettlementID, measuredResult.SettlementID)
	require.EqualValues(t, 1, measuredResult.AmountMinor)
	require.Equal(t, cf5Price.ID, measuredResult.PriceID)
	require.Equal(t, x402.CallID, measuredResult.CallID)
	require.Equal(t, measuredSettlementID+":ledger", measuredResult.LedgerEmission)
	require.Equal(t, acceptedMeasuredSettlement.TreasuryReceiptID, measuredResult.TreasuryReceipt)
	require.Equal(t, "measured", measuredResult.BenefitState)
	require.Greater(t, measuredResult.ResponseBytes, 0)
	require.Len(t, measuredResult.ResponseSHA256, 64)
	for _, character := range measuredResult.ResponseSHA256 {
		require.True(t, character >= '0' && character <= '9' || character >= 'a' && character <= 'f', "response digest must be lowercase hex")
	}
	require.NotEmpty(t, measuredResult.BodyPath)
	require.False(t, filepath.IsAbs(measuredResult.BodyPath), "body artifact must be scoped to the run evidence directory")
	cleanBodyPath := filepath.Clean(measuredResult.BodyPath)
	require.NotEqual(t, "..", cleanBodyPath)
	require.False(t, strings.HasPrefix(cleanBodyPath, ".."+string(filepath.Separator)), "body artifact cannot escape its evidence directory")
	measuredBody, err := os.ReadFile(filepath.Join(agentManagerEvidenceDir, cleanBodyPath))
	require.NoError(t, err)
	require.Len(t, measuredBody, measuredResult.ResponseBytes)
	measuredBodyDigest := sha256.Sum256(measuredBody)
	require.Equal(t, measuredResult.ResponseSHA256, fmt.Sprintf("%x", measuredBodyDigest))
	var usefulCatalog struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(measuredBody, &usefulCatalog), "paid body must be the exact catalog response, not just a settlement receipt")
	require.NotEmpty(t, usefulCatalog.Nodes)
	var foundSeededOffer bool
	for _, node := range usefulCatalog.Nodes {
		if node.ID == offerID || node.ID == offerVariantID {
			foundSeededOffer = true
			break
		}
	}
	require.True(t, foundSeededOffer, "paid response must include the seeded useful offer graph")
	agentManagerBudgetEvidenceDir := filepath.Join(runDir, "agent-manager-research-budget")
	require.NoError(t, os.MkdirAll(agentManagerBudgetEvidenceDir, 0o700))
	writeCF5JSON(t, filepath.Join(agentManagerBudgetEvidenceDir, "snapshot.json"), json.RawMessage(snapshotEvidence))
	writeCF5JSON(t, filepath.Join(agentManagerBudgetEvidenceDir, "x402-settlement.json"), json.RawMessage(x402SettlementEvidence))
	writeCF5JSON(t, filepath.Join(agentManagerBudgetEvidenceDir, "receiver-provenance.json"), map[string]any{
		"authenticated_service_token": true, "service_subject": "treasury-agent-manager",
		"immutable_duplicate_policy": "exact repeat accepted; conflicting body rejected",
		"receiver":                   "isolated CF5 HTTP fixture sharing the identity authority origin",
		"production_agent_manager_scenario_exercised": false,
	})
	headroomRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, uiURL+"/api/v1/operator-session/budget-headroom?budget_id="+budgetID, nil)
	require.NoError(t, err)
	headroomResponse, err := sessionHTTP.Do(headroomRequest)
	require.NoError(t, err)
	headroomBody, err := io.ReadAll(headroomResponse.Body)
	require.NoError(t, err)
	require.NoError(t, headroomResponse.Body.Close())
	require.Equalf(t, http.StatusOK, headroomResponse.StatusCode, "cookie-only budget headroom response: %s", strings.TrimSpace(string(headroomBody)))
	var browserHeadroom map[string]any
	require.NoError(t, json.Unmarshal(headroomBody, &browserHeadroom))
	require.Equal(t, budgetID, browserHeadroom["budget_id"])
	sessionFunding := fundingconnect.NewFundingAdminServiceClient(sessionHTTP, uiURL)
	sessionIntents, err := sessionFunding.ListFundingIntents(ctx, connect.NewRequest(&fundingpb.ListFundingIntentsRequest{BudgetId: budgetID}))
	require.NoError(t, err, "same-origin HttpOnly session must authorize the token-free Fund Connect path")
	require.NotEmpty(t, sessionIntents.Msg.GetIntents(), "session subject mapping must resolve the same operator-owned funding intent")
	writeCF5JSON(t, filepath.Join(runDir, "treasury-browser-session-fund-readback.json"), map[string]any{
		"cookie_only_budget_headroom": true, "budget_id": budgetID, "credited_minor": browserHeadroom["funding_credited_minor"],
		"cookie_only_connect_funding_list": true, "intents_visible": len(sessionIntents.Msg.GetIntents()),
		"same_origin": true, "static_token_headers": false,
	})
	purchases := []int64{30, 110}
	if hooks.X402 != nil {
		// The Agent Manager showcase performs both a dropped-response recovery
		// attempt and a separate bounded call whose paid response is measured.
		purchases = append(purchases, 1, 1)
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
		expectedReturnMinor -= 2
	}
	require.EqualValues(t, expectedReturnMinor, afterSpend.GetHeadroom().GetFundingRemainingMinor())
	writeRehearsalProto(t, filepath.Join(runDir, "headroom-after-agent-purchases.json"), afterSpend)

	withdrawalDestination := strings.ToLower(fork.Accounts[2])
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{walletAddress, "0xde0b6b3a7640000"}, nil))
	_, err = budgetAdmin.SetBudgetCaps(ctx, cf5OperatorRequest(&budgetv1.SetBudgetCapsRequest{Budget: &budgetv1.Budget{
		Id: budgetID, BookId: bookID, TotalCapMinor: 1000, PeriodicCapMinor: 1000,
		PerTransactionCapMinor: 1000, PeriodSeconds: 3600, Currency: "USD",
	}}))
	require.NoError(t, err, "the local rehearsal explicitly permits its small return transfer after CF1's over-cap approval proof")
	_, err = fundingIntents.SetWithdrawalDestination(ctx, cf5OperatorRequest(&fundingpb.SetWithdrawalDestinationRequest{Destination: &fundingpb.WithdrawalDestination{
		BudgetId: budgetID, Address: withdrawalDestination, Label: "CF5 local Coinbase test destination",
	}}))
	require.NoError(t, err)
	withdrawalMandateID := "cf5-withdrawal-mandate-" + variant
	mandateResponse, err := mandateAdmin.CreateMandate(ctx, cf5OperatorRequest(&mandatepb.CreateMandateRequest{Mandate: &mandatepb.Mandate{
		Id: withdrawalMandateID, IdempotencyKey: withdrawalMandateID + "-key", BookId: bookID, BudgetId: budgetID,
		CapMinor: 200, Currency: "USD", AllowedCounterparties: []string{withdrawalDestination}, AgentSubject: "cf1-local-operator", AgentRunId: runID,
		ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}}))
	require.NoError(t, err)
	require.Equal(t, mandatepb.MandateStatus_MANDATE_STATUS_LIVE, mandateResponse.Msg.GetMandate().GetStatus())
	destinationBefore := cf5USDCBalance(t, ctx, fork, withdrawalDestination)
	_, withdrawalAtomic, err := cf5WithdrawalPlan(expectedReturnMinor, cf5WithdrawalFeeMinor)
	require.NoError(t, err)
	// No test-human browser identity is provisioned. Keep browser automation
	// token-free and leave all withdrawal authority decisions to the operator.
	captureErr := errors.New("withdrawal UI capture skipped: no disposable verified-human browser session or operator-approved run principal is configured")
	basCaptureError = strings.TrimSpace(strings.Join([]string{basCaptureError, captureErr.Error()}, "; "))
	writeCF5JSON(t, filepath.Join(runDir, "bas-mobile-withdrawal-capture-skipped.json"), map[string]any{
		"reason": captureErr.Error(), "bas_artifact_created": false, "withdrawal_ui_flow_verified": false,
	})
	{
		_, ok := new(big.Int).SetString(withdrawalAtomic, 10)
		require.True(t, ok)
		quoteResponse, quoteErr := fundingIntents.PreviewWithdrawal(ctx, cf5OperatorRequest(&fundingpb.PreviewWithdrawalRequest{
			BudgetId: budgetID, MandateId: mandateResponse.Msg.GetMandate().GetId(), Destination: withdrawalDestination,
			AtomicAmount: withdrawalAtomic, MaxFeeWei: "1000000000000000000", FeeValuationMinor: cf5WithdrawalFeeMinor,
			Valuation: &ingestpb.CryptoValuation{AmountMinor: expectedReturnMinor - cf5WithdrawalFeeMinor, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "CF5 isolated local-fork return value", ValuedAt: timestamppb.Now()},
		}))
		require.NoError(t, quoteErr)
		require.Equal(t, "local-fork", quoteResponse.Msg.GetQuote().GetExecutionMode())
		writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-quote-api-fallback.json"), quoteResponse.Msg)
		execute := func() *connect.Response[fundingpb.WithdrawalResponse] {
			req := cf5OperatorRequest(&fundingpb.ExecuteWithdrawalRequest{QuoteId: quoteResponse.Msg.GetQuote().GetId(), IdempotencyKey: "cf5-api-return-" + variant})
			req.Header().Set("X-Treasury-Mandate-Identity-Token", "cf5-local-agent")
			resp, executeErr := fundingIntents.ExecuteWithdrawal(ctx, req)
			require.NoError(t, executeErr)
			return resp
		}
		withdrawal := execute().Msg.GetWithdrawal()
		if withdrawal.GetState() == "pending_approval" {
			approvals, listErr := operator.ListApprovals(ctx, cf5OperatorRequest(&authorizationv1.ListApprovalsRequest{Status: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: bookID}))
			require.NoError(t, listErr)
			for _, queued := range approvals.Msg.GetApprovals() {
				if queued.GetAuthorizationId() == withdrawal.GetAuthorizationId() {
					_, resolveErr := operator.ResolveApproval(ctx, cf5OperatorRequest(&authorizationv1.ResolveApprovalRequest{ApprovalId: queued.GetId(), Resolution: treasurysharedpb.ApprovalStatus_APPROVAL_STATUS_APPROVED}))
					require.NoError(t, resolveErr)
				}
			}
			withdrawal = execute().Msg.GetWithdrawal()
		}
		require.Equal(t, "settled", withdrawal.GetState(), withdrawal.GetLastError())
		writeRehearsalProto(t, filepath.Join(runDir, "withdrawal-api-fallback.json"), withdrawal)
		writeCF5JSON(t, filepath.Join(runDir, "withdrawal-api-fallback-evidence.json"), map[string]any{
			"created_by": "authenticated Treasury API in isolated local-fork test", "mobile_ui_action_claimed": false,
			"reason": captureErr.Error(), "transaction_hash": withdrawal.GetTransactionHash(),
		})
	}
	destinationAfter := cf5USDCBalance(t, ctx, fork, withdrawalDestination)
	withdrawalAtomicBig, ok := new(big.Int).SetString(withdrawalAtomic, 10)
	require.True(t, ok)
	require.Equal(t, new(big.Int).Add(destinationBefore, withdrawalAtomicBig).String(), destinationAfter.String(), "the mobile confirmation must execute exactly the allowlisted return transfer")
	var withdrawalReceipt struct {
		Status string `json:"status"`
	}
	// The authenticated Treasury API fallback settled this local-fork return;
	// chain balance and receipt reads independently verify the transfer.
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
	stopReceipt := fork.Stop(ctx)
	writeCF5JSON(t, filepath.Join(runDir, "fork-stop.json"), stopReceipt)
	require.True(t, stopReceipt.ProcessExited)
	require.False(t, stopReceipt.KilledAfterTimeout)
	require.False(t, stopReceipt.EndpointReachableAfter, "%+v", stopReceipt)
	require.True(t, stopReceipt.Clean, "%+v", stopReceipt)
	completed := []string{
		"the CF1 budget's actual Treasury research root was exported to a key-only encrypted portable bundle, its CA scalar removed, failed to sign, restored from the separately copied bundle, and proved with a fresh challenge before receipt funding",
		"ordinary Agent Manager run wallet funded by a finalized local Base USDC transfer",
		"CF1 in-cap and approved over-cap local-fork purchases under the run-bound mandate",
		"isolated Money Ledger exact-token receipt acceptance",
		"exact Treasury budget headroom consumption after purchases and return",
		"Notification Hub delivered the accepted-receipt notice to a bounded local Web Push endpoint within five seconds; the opaque encrypted payload and durable provider receipt were read back",
	}
	remaining := []string{
		"qualify DBM off-host transport and restore on a second independent fleet owner",
		"replace the in-cap native USDC fixture purchase with the fully configured CF4 facilitator priced call",
		"prove accepted-receipt push delivery to an enrolled operator mobile device; the current endpoint is a local test fixture",
		"independent CF1/CF5 code review",
	}
	if basCaptureError == "" {
		completed = append(completed, "BAS mobile Fund, exact prefilled Base link, receipt confirmation, and allowlisted withdrawal preview/execution")
	} else if basFundAuthenticatedCaptured {
		remaining = append(remaining, "BAS verified the same-origin verified-human Fund session and exact prefilled Base USDC link; wallet-app open, on-chain send, receipt confirmation, push, priced provider result, and allowlisted return UI remain unproven. API fallback is not mobile UI evidence.")
	} else {
		remaining = append(remaining, fmt.Sprintf("BAS mobile Fund, receipt and withdrawal journey remains unproven (%s); local-fork/API fallback does not prove the mobile UI flow", basCaptureError))
	}
	writeCF5JSON(t, filepath.Join(runDir, "handoff-result.json"), map[string]any{
		"status": "partial", "completed": completed, "remaining": remaining,
		"notification_delivery": map[string]any{"status": "local_push_delivered", "phone_delivery_claimed": false, "maximum_latency_seconds": 5},
		"instance":              name, "api_url": apiURL, "ui_url": uiURL,
		"fork": forkIdentity, "fork_stop": stopReceipt,
	})
}

func cf5LocalPushReceiptProviderID(databasePath, notificationID string) (string, error) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath)+"?mode=ro")
	if err != nil {
		return "", err
	}
	defer db.Close()
	var providerID string
	err = db.QueryRow(`SELECT provider_id FROM receipts WHERE notification_id=? AND channel='web_push' ORDER BY delivered_at DESC LIMIT 1`, notificationID).Scan(&providerID)
	return providerID, err
}

func TestCF5LocalPushReceiptProviderIDReadsIsolatedReceipt(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "notification-hub.db")
	db, err := sql.Open("sqlite", databasePath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE receipts (notification_id TEXT NOT NULL, channel TEXT NOT NULL, provider_id TEXT NOT NULL, delivered_at TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO receipts (notification_id, channel, provider_id, delivered_at) VALUES ('notice-1', 'web_push', 'cf5-local-push-receipt', '2026-11-14T00:00:00Z')`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	providerID, err := cf5LocalPushReceiptProviderID(databasePath, "notice-1")
	require.NoError(t, err)
	require.Equal(t, "cf5-local-push-receipt", providerID)
}

// cf5RecoverBudgetRootBeforeFunding exports the exact receive-only root that
// parents the run's mandate wallet, removes its CA scalar, and restores it
// from the encrypted portable bundle before any receipt funding is accepted.
// Independent-CA node recovery is covered by the wallet package's restart
// proof; this joined rehearsal proves the live mandate/accounting path waits
// until the root is recoverable again.
func cf5RecoverBudgetRootBeforeFunding(t *testing.T, ctx context.Context, runDir string, admin walletconnect.WalletAdminServiceClient, books bookconnect.BookAdminServiceClient, bookID, custodyDir string) {
	t.Helper()
	bookResponse, err := books.GetBook(ctx, cf5OperatorRequest(&bookv1.GetBookRequest{BookId: bookID}))
	require.NoError(t, err)
	operatorID := bookResponse.Msg.GetBook().GetBeneficiaryIdentity()
	require.NotEmpty(t, operatorID)
	rootDigest := sha256.Sum256([]byte(strings.Join([]string{operatorID, bookID, baseusdc.Network}, "\x00")))
	rootID := fmt.Sprintf("budget-root-%x", rootDigest[:16])
	rootResponse, err := admin.GetResearchWallet(ctx, cf5OperatorRequest(&walletpb.ResearchWalletRequest{Id: rootID, Network: baseusdc.Network}))
	require.NoError(t, err, "read the actual receive-only root wallet that parents the mandate wallet")
	root := rootResponse.Msg.GetWallet()
	require.NotEmpty(t, root.GetCustodyReference())
	secret := "operator-held local rehearsal recovery secret with adequate entropy"
	exported, err := admin.ExportPortableResearchWalletRecovery(ctx, cf5OperatorRequest(&walletpb.ExportPortableResearchWalletRecoveryRequest{
		Id: rootID, Network: baseusdc.Network, OperatorSecret: secret,
	}))
	require.NoError(t, err)
	bundleDirMatches, err := filepath.Glob(filepath.Join(runDir, "storage", "data", "vrooli", "*", filepath.FromSlash(exported.Msg.GetRelativePath())))
	require.NoError(t, err)
	require.Len(t, bundleDirMatches, 1, "portable recovery bundle directory must stay inside this isolated Treasury variant's data root")
	serverBundlePath := filepath.Join(bundleDirMatches[0], "wallet.bundle")
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
	_, err = admin.ProveResearchWallet(ctx, cf5OperatorRequest(&walletpb.ProveResearchWalletRequest{Id: rootID, Network: baseusdc.Network, Nonce: "cf5-source-key-removed-before-funding"}))
	require.Error(t, err, "source root must stop signing after the scalar is removed")
	restored, err := admin.RestorePortableResearchWalletRecovery(ctx, cf5OperatorRequest(&walletpb.RestorePortableResearchWalletRecoveryRequest{
		Id: rootID, Network: baseusdc.Network, OperatorSecret: secret, Bundle: bundle,
	}))
	require.NoError(t, err)
	require.Equal(t, root.GetAddress(), restored.Msg.GetWallet().GetAddress())
	proof, err := admin.ProveResearchWallet(ctx, cf5OperatorRequest(&walletpb.ProveResearchWalletRequest{Id: rootID, Network: baseusdc.Network, Nonce: "cf5-recovered-root-before-funding"}))
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

func captureCF5MobileFund(t *testing.T, ctx context.Context, runDir, uiURL, email, password, budgetID, walletAddress string) error {
	t.Helper()
	expectedPaymentURI := fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=2600000", baseusdc.USDCAddress, walletAddress)
	flow := map[string]any{
		"settings": map[string]any{"viewport_width": 390, "viewport_height": 844},
		"nodes": []any{
			map[string]any{"id": "wait-fund", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='page-fund']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Mobile Fund page renders"}}},
			map[string]any{"id": "assert-fork-boundary", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='page-fund']", "mode": "ASSERTION_MODE_TEXT_CONTAINS", "expected": map[string]any{"string_value": "This page does not publish a production funding address or authorize customer deposits."}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify the local-fork-only funding boundary"}}},
			map[string]any{"id": "email", "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='funding-session-email']", "value": email, "clear_first": true}, "metadata": map[string]any{"label": "Sign in the disposable verified test operator"}}},
			map[string]any{"id": "password", "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='funding-session-password']", "value": password, "clear_first": true}, "metadata": map[string]any{"label": "Enter the ephemeral local-only operator password"}}},
			map[string]any{"id": "sign-in", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-session-sign-in']"}, "metadata": map[string]any{"label": "Start the same-origin HttpOnly session"}}},
			map[string]any{"id": "wait-session", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-session-identity']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Wait for the verified operator session"}}},
			map[string]any{"id": "budget", "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='funding-budget-id']", "value": budgetID, "clear_first": true}, "metadata": map[string]any{"label": "Select the run-owned research budget"}}},
			map[string]any{"id": "amount", "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='funding-atomic-amount']", "value": "2.60", "clear_first": true}, "metadata": map[string]any{"label": "Set the local test receipt amount"}}},
			map[string]any{"id": "declared-value", "action": map[string]any{"type": "ACTION_TYPE_INPUT", "input": map[string]any{"selector": "[data-testid='funding-declared-value']", "value": "2.60", "clear_first": true}, "metadata": map[string]any{"label": "Set the separately valued local test receipt"}}},
			map[string]any{"id": "create-intent", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "[data-testid='funding-create-intent']"}, "metadata": map[string]any{"label": "Tap Fund to create the Base USDC payment link"}}},
			map[string]any{"id": "wait-payment-link", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-testid='funding-open-wallet-link']", "state": "WAIT_STATE_VISIBLE", "timeout_ms": 15000}, "metadata": map[string]any{"label": "Wait for the prefilled Base wallet link"}}},
			map[string]any{"id": "assert-payment-link", "action": map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "[data-testid='funding-open-wallet-link']", "mode": "ASSERTION_MODE_ATTRIBUTE_EQUALS", "attribute_name": "href", "expected": map[string]any{"string_value": expectedPaymentURI}, "timeout_ms": 5000}, "metadata": map[string]any{"label": "Verify exact Base network, USDC token, amount, and wallet address"}}},
			map[string]any{"id": "capture-payment-link", "action": map[string]any{"type": "ACTION_TYPE_SCREENSHOT", "screenshot": map[string]any{"full_page": true}, "metadata": map[string]any{"label": "Capture the authenticated mobile Fund page and exact prefilled wallet link"}}},
		},
		"edges": []any{
			cf5Edge("e1", "wait-fund", "assert-fork-boundary"),
			cf5Edge("e2", "assert-fork-boundary", "email"), cf5Edge("e3", "email", "password"),
			cf5Edge("e4", "password", "sign-in"), cf5Edge("e5", "sign-in", "wait-session"),
			cf5Edge("e6", "wait-session", "budget"), cf5Edge("e7", "budget", "amount"),
			cf5Edge("e8", "amount", "declared-value"), cf5Edge("e9", "declared-value", "create-intent"),
			cf5Edge("e10", "create-intent", "wait-payment-link"), cf5Edge("e11", "wait-payment-link", "assert-payment-link"),
			cf5Edge("e12", "assert-payment-link", "capture-payment-link"),
		},
	}
	flowPath := filepath.Join(runDir, "mobile-fund-capture-flow.json")
	defer os.Remove(flowPath)
	raw, err := json.MarshalIndent(flow, "", "  ")
	require.NoError(t, err)
	if err = os.WriteFile(flowPath, raw, 0o600); err != nil {
		return err
	}
	artifactDir := filepath.Join(runDir, "bas-mobile-fund")
	if err = os.MkdirAll(artifactDir, 0o700); err != nil {
		return err
	}
	cli, err := exec.LookPath("browser-automation-studio")
	if err != nil {
		return fmt.Errorf("BAS CLI is required for the CF5 mobile recording: %w", err)
	}
	cmd := exec.CommandContext(ctx, cli, "capture", "--url", uiURL+"/fund", "--dimensions", "mobile", "--capture", "screenshot", "--wait-for", "[data-testid='page-fund']", "--interaction-flow-file", flowPath, "--out", artifactDir, "--label", "CF5 token-free local-fork mobile Fund surface", "--json")
	output, err := cmd.CombinedOutput()
	if writeErr := os.WriteFile(filepath.Join(runDir, "bas-mobile-fund-capture.json"), output, 0o600); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return fmt.Errorf("BAS mobile Fund surface capture failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if len(output) == 0 {
		return fmt.Errorf("BAS mobile Fund capture returned no artifact receipt")
	}
	return nil
}

func cf5Edge(id, source, target string) map[string]any {
	return map[string]any{"id": id, "source": source, "target": target}
}

func cf5DeclareOfferDeskPriceWithSession(t *testing.T, ctx context.Context, uiURL string, client *http.Client, declaration map[string]any) x402rail.Price {
	t.Helper()
	encoded, err := json.Marshal(declaration)
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(uiURL, "/")+"/api/v1/x402/prices/offer-desk", strings.NewReader(string(encoded)))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", uiURL)
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, response.StatusCode, "same-origin session price declaration: %s", strings.TrimSpace(string(body)))
	var price x402rail.Price
	require.NoError(t, json.Unmarshal(body, &price))
	return price
}

func cf5OperatorRequest[T any](value *T) *connect.Request[T] {
	request := connect.NewRequest(value)
	request.Header().Set("X-Vrooli-Operator-Token", "cf2-local-operator-only")
	return request
}

func cf5ServiceRequest[T any](value *T, token string) *connect.Request[T] {
	request := connect.NewRequest(value)
	request.Header().Set("Authorization", "Bearer "+token)
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
	if scenarioName == "treasury" {
		// The lifecycle variant builds from this isolated clone but launches the
		// API with the serving Treasury tree as its working directory. Keep the
		// generated freshness manifest's research-relative inputs resolvable
		// there for the duration of this rehearsal.
		home, homeErr := os.UserHomeDir()
		require.NoError(t, homeErr)
		servingResearch := filepath.Join(repo, ".vrooli", "research")
		if _, statErr := os.Lstat(servingResearch); errors.Is(statErr, os.ErrNotExist) {
			require.NoError(t, os.Symlink(filepath.Join(home, ".vrooli", "research"), servingResearch))
			t.Cleanup(func() { _ = os.Remove(servingResearch) })
		} else {
			require.NoError(t, statErr)
		}
	}
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
					if scenarioName == "treasury" {
						if err := copyCF5Tree(path, filepath.Join(destRoot, rel)); err != nil {
							return err
						}
						return filepath.SkipDir
					}
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

// copyCF5Tree keeps the Treasury UI build's dependency tree inside the
// isolated scenario variant. Sharing ui/node_modules makes lifecycle correctly
// reject a build while the live Treasury instance is serving; copying this
// already-provisioned tree lets the rehearsal rebuild its own UI safely.
func copyCF5Tree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := destination
		if rel != "." {
			target = filepath.Join(destination, rel)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		inCloseErr := in.Close()
		outCloseErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if inCloseErr != nil {
			return inCloseErr
		}
		return outCloseErr
	})
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

func seedCF5LocalPushSubscription(t *testing.T, databasePath string, fixture cf5WebPushFixture) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))
	_, err = db.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
	require.NoError(t, err)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	// The routed Notification Hub test pool pins the owner to its deterministic
	// test recipient. Match that owner so the delivered receipt notification can
	// resolve this isolated Web Push subscription.
	const recipientID = "e1-fixture@notification-hub.invalid"
	const deviceID = "cf5-local-web-push-fixture"
	_, err = db.ExecContext(ctx, `INSERT INTO recipients (id, subject, trust_posture, created_at, updated_at) VALUES (?, ?, 'personal', ?, ?) ON CONFLICT(subject) DO NOTHING`, recipientID, recipientID, now, now)
	require.NoError(t, err)
	var recipientRowID string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, recipientID).Scan(&recipientRowID))
	_, err = db.ExecContext(ctx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES (?, ?, ?, 'local', ?)`, deviceID, recipientRowID, "CF5 local push endpoint fixture", now)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES (?, ?, 'web_push', ?, '["public","private","sensitive","critical"]', ?)`, deviceID+"-channel", deviceID, fixture.Endpoint, now)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO push_subscriptions (id, recipient_id, device_id, endpoint, p256dh, auth, origin, created_at, updated_at, user_agent, permission, confirmed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'granted', ?)`, deviceID+"-subscription", recipientRowID, deviceID, fixture.Endpoint, fixture.ClientPublic, fixture.ClientAuth, "https://treasury.example.test", now, now, "CF5 bounded local push endpoint fixture", now)
	require.NoError(t, err)
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
