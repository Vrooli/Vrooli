package remote

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/vrooli/cli-core/cliutil"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSessionIntentSignatureMatchesTreasuryCanonicalMessage(t *testing.T) {
	privateKey, err := parseSessionPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	intent := agentSettlementIntent{
		AuthorizationID: "auth-7", MandateID: "mandate-2", BudgetID: "research", WalletID: "wallet-9",
		WalletAddress: "0x0000000000000000000000000000000000000001", Network: "base",
		AgentSubject: "agent:researcher", AgentRunID: "run-42", SettlementID: "settle-3",
		InstrumentID: "native-usdc", IdempotencyKey: "idem-8", AmountMinor: 125,
		Currency: "USD", Counterparty: "model:fixture", Generation: 4,
		ExpiresAt: "2026-11-09T22:30:00Z",
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := `{"authorization_id":"auth-7","mandate_id":"mandate-2","budget_id":"research","wallet_id":"wallet-9","wallet_address":"0x0000000000000000000000000000000000000001","network":"base","agent_subject":"agent:researcher","agent_run_id":"run-42","settlement_id":"settle-3","instrument_id":"native-usdc","idempotency_key":"idem-8","amount_minor":125,"currency":"USD","counterparty":"model:fixture","generation":4,"expires_at":"2026-11-09T22:30:00Z"}`
	if string(encoded) != wantJSON {
		t.Fatalf("session intent JSON drifted from Treasury contract:\n got %s\nwant %s", encoded, wantJSON)
	}
	signatureHex, err := signSessionIntent(privateKey, intent)
	if err != nil {
		t.Fatal(err)
	}
	retriedSignature, err := signSessionIntent(privateKey, intent)
	if err != nil || retriedSignature != signatureHex {
		t.Fatalf("same intent retry changed its deterministic signature: first=%q retry=%q err=%v", signatureHex, retriedSignature, err)
	}
	signature, err := hex.DecodeString(signatureHex[2:])
	if err != nil || len(signature) != 65 || signature[64] > 1 {
		t.Fatalf("invalid Ethereum signature encoding: len=%d err=%v", len(signature), err)
	}
	message := "Vrooli Treasury mandate session settlement v1\n" + wantJSON
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len([]byte(message))) + message))
	digest := hash.Sum(nil)
	compact := make([]byte, 65)
	compact[0] = 27 + signature[64]
	copy(compact[1:], signature[:64])
	recovered, compressed, err := secpECDSA.RecoverCompact(compact, digest)
	if err != nil || compressed || string(recovered.SerializeUncompressed()) != string(privateKey.PubKey().SerializeUncompressed()) {
		t.Fatalf("signature does not recover to the run-owned session key: compressed=%t err=%v", compressed, err)
	}
}

func TestParseSessionPrivateKeyRejectsInvalidScalars(t *testing.T) {
	for _, value := range []string{"", "1", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"} {
		if _, err := parseSessionPrivateKey(value); err == nil {
			t.Errorf("accepted invalid session scalar %q", value)
		}
	}
}

type proofReaderFixture struct {
	authorization      *authorizationv1.AuthorizationRecord
	wallet             *fundingv1.BudgetWallet
	authorizationCalls int
	walletCalls        int
}

func (f *proofReaderFixture) GetAuthorization(_ context.Context, request *connect.Request[authorizationv1.GetAuthorizationRequest]) (*connect.Response[authorizationv1.GetAuthorizationResponse], error) {
	f.authorizationCalls++
	if request.Msg.GetId() != f.authorization.GetId() {
		return nil, context.Canceled
	}
	return connect.NewResponse(&authorizationv1.GetAuthorizationResponse{Authorization: f.authorization}), nil
}

func (f *proofReaderFixture) GetBudgetWallet(_ context.Context, request *connect.Request[fundingv1.GetBudgetWalletRequest]) (*connect.Response[fundingv1.BudgetWalletResponse], error) {
	f.walletCalls++
	if request.Msg.GetBudgetId() != f.wallet.GetBudgetId() || request.Msg.GetMandateId() != f.wallet.GetMandateId() || request.Header().Get(sessionPublicKeyHeader) == "" {
		return nil, context.Canceled
	}
	return connect.NewResponse(&fundingv1.BudgetWalletResponse{Wallet: f.wallet}), nil
}

func TestPrepareSettlementProofDoesNotSignPendingAuthorization(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	key, err := parseSessionPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &proofReaderFixture{authorization: &authorizationv1.AuthorizationRecord{
		Id: "auth-pending", Verdict: authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING,
	}}
	request := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-pending", SettlementId: "settle-1", InstrumentId: "instrument-1", IdempotencyKey: "idem-1"})
	if _, err := prepareSettlementProofFromClient(fixture, request.Msg, key); err == nil {
		t.Fatal("pending authorization produced an agent session signature")
	}
	if fixture.authorizationCalls != 1 || fixture.walletCalls != 0 {
		t.Fatalf("pending authorization reached wallet/signature path: auth=%d wallet=%d", fixture.authorizationCalls, fixture.walletCalls)
	}
}

func TestSessionPublicKeyHeaderUsesInjectedKeySource(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	request := connect.NewRequest(&fundingv1.GetBudgetWalletRequest{BudgetId: "budget-1", MandateId: "mandate-1"})
	prepare := treasurySessionRequestHeadersWithKeySource(nil, func() string {
		return "0000000000000000000000000000000000000000000000000000000000000001"
	})
	headers, err := prepare(nil, "AgentSpend.GetBudgetWallet", request.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if headers[sessionPublicKeyHeader] == "" || headers[sessionProofHeader] != "" {
		t.Fatalf("unexpected first-binding headers: %+v", headers)
	}
}

func TestTreasuryAgentCallForwardsCurrentPairedOperationGrant(t *testing.T) {
	tmp, err := os.MkdirTemp("/tmp", "cf1-agent-cred-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	t.Setenv("TMPDIR", tmp)
	t.Setenv(cliutil.EnvIdentityToken, "baseline-inherited")
	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/credential" || r.URL.Query().Get("operation") != "treasury.read" || r.URL.Query().Get("target") != "treasury:budget/budget-1" || r.Header.Get(cliutil.HeaderAgentIdentityToken) != "baseline-inherited" {
			http.Error(w, "unexpected credential request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": "baseline-inherited", "operation_token": "operation-current"})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	request := connect.NewRequest(&fundingv1.GetBudgetWalletRequest{BudgetId: "budget-1", MandateId: "mandate-1"})
	prepare := treasurySessionRequestHeadersWithKeySource(nil, func() string {
		return "0000000000000000000000000000000000000000000000000000000000000001"
	})
	headers, err := prepare(nil, "AgentSpend.GetBudgetWallet", request.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if headers[cliutil.HeaderAgentIdentityToken] != "baseline-inherited" || headers[cliutil.HeaderAgentOperationToken] != "operation-current" || headers[sessionPublicKeyHeader] == "" {
		t.Fatalf("Treasury agent request did not carry the refreshed baseline/operation pair and session key: %#v", headers)
	}
}

func TestProposeChargeRequestsItsExactImmutableGrantTarget(t *testing.T) {
	tmp, err := os.MkdirTemp("/tmp", "cf1-propose-grant-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	t.Setenv("TMPDIR", tmp)
	t.Setenv(cliutil.EnvIdentityToken, "baseline-run-token")
	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	proposalTarget := cliutil.TreasurySpendIntentAuthorizationTarget(cliutil.TreasurySpendIntent{
		AuthorizationID: "auth-17", MandateID: "mandate-5", AmountMinor: 425, Currency: "USD", Counterparty: "research-provider",
		AuthorizationIdempotencyKey: "proposal-idem-17", SettlementID: "settlement-17", InstrumentID: "base-usdc-budget-2", SettlementIdempotencyKey: "settlement-idem-17",
	})
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/credential" || r.URL.Query().Get("operation") != "treasury.spend" || r.URL.Query().Get("target") != proposalTarget || r.Header.Get(cliutil.HeaderAgentIdentityToken) != "baseline-run-token" {
			http.Error(w, "unexpected spend intent credential request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": "baseline-run-token", "operation_token": "intent-grant-token"})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	request := connect.NewRequest(&authorizationv1.ProposeChargeRequest{
		Id: "auth-17", MandateId: "mandate-5", AmountMinor: 425, Currency: "USD",
		Counterparty: "research-provider", IdempotencyKey: "proposal-idem-17", SettlementId: "settlement-17",
		InstrumentId: "base-usdc-budget-2", SettlementIdempotencyKey: "settlement-idem-17",
	})
	prepare := treasurySessionRequestHeadersWithKeySource(nil, nil)
	headers, err := prepare(nil, "AgentSpend.ProposeCharge", request.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if headers[cliutil.HeaderAgentIdentityToken] != "baseline-run-token" || headers[cliutil.HeaderAgentOperationToken] != "intent-grant-token" {
		t.Fatalf("proposal did not carry its scoped identity pair: %#v", headers)
	}
}

func TestPrepareSettlementProofSignsApprovedRunBoundWalletIntent(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	key, err := parseSessionPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &proofReaderFixture{
		authorization: &authorizationv1.AuthorizationRecord{
			Id: "auth-approved", MandateId: "mandate-1", BudgetId: "budget-1", RequestingAgent: "agent:research",
			AmountMinor: 125, Currency: "USD", Counterparty: "model:fixture",
			SettlementId: "settle-1", InstrumentId: "instrument-1", SettlementIdempotencyKey: "idem-1",
			Verdict:   authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED,
			ExpiresAt: timestamppb.New(time.Date(2026, 11, 9, 23, 30, 0, 0, time.UTC)),
		},
		wallet: &fundingv1.BudgetWallet{
			BudgetId: "budget-1", WalletId: "wallet-1", Network: "base", Address: "0x0000000000000000000000000000000000000001",
			AgentSubject: "agent:research", AgentRunId: "run-42", MandateId: "mandate-1", SessionGeneration: 4,
			SessionExpiresAt: timestamppb.New(time.Date(2026, 11, 9, 22, 30, 0, 0, time.UTC)),
		},
	}
	request := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-approved", SettlementId: "settle-1", InstrumentId: "instrument-1", IdempotencyKey: "idem-1"})
	proof, err := prepareSettlementProofFromClientAt(fixture, request.Msg, key, time.Date(2026, 11, 9, 22, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if fixture.authorizationCalls != 1 || fixture.walletCalls != 1 {
		t.Fatalf("approved proof did not read exact authorization and wallet: auth=%d wallet=%d", fixture.authorizationCalls, fixture.walletCalls)
	}
	signature, err := hex.DecodeString(proof[2:])
	if err != nil || len(signature) != 65 || signature[64] > 1 {
		t.Fatalf("invalid proof signature encoding: len=%d err=%v", len(signature), err)
	}
}

func TestPrepareSettlementProofDoesNotSignExpiredAuthorizationOrSession(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	key, err := parseSessionPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 22, 0, 0, 0, time.UTC)
	request := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-expired", SettlementId: "settle-1", InstrumentId: "instrument-1", IdempotencyKey: "idem-1"})
	approved := authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED

	t.Run("authorization expired", func(t *testing.T) {
		fixture := &proofReaderFixture{authorization: &authorizationv1.AuthorizationRecord{
			Id: "auth-expired", MandateId: "mandate-1", BudgetId: "budget-1", RequestingAgent: "agent:research",
			AmountMinor: 125, Currency: "USD", Counterparty: "model:fixture", SettlementId: "settle-1", InstrumentId: "instrument-1", SettlementIdempotencyKey: "idem-1", Verdict: approved,
			ExpiresAt: timestamppb.New(now.Add(-time.Second)),
		}}
		if _, err := prepareSettlementProofFromClientAt(fixture, request.Msg, key, now); err == nil {
			t.Fatal("expired authorization produced a session proof")
		}
		if fixture.walletCalls != 0 {
			t.Fatalf("expired authorization reached wallet/session key lookup: calls=%d", fixture.walletCalls)
		}
	})

	t.Run("session expired", func(t *testing.T) {
		fixture := &proofReaderFixture{
			authorization: &authorizationv1.AuthorizationRecord{
				Id: "auth-expired", MandateId: "mandate-1", BudgetId: "budget-1", RequestingAgent: "agent:research",
				AmountMinor: 125, Currency: "USD", Counterparty: "model:fixture", SettlementId: "settle-1", InstrumentId: "instrument-1", SettlementIdempotencyKey: "idem-1", Verdict: approved,
				ExpiresAt: timestamppb.New(now.Add(time.Hour)),
			},
			wallet: &fundingv1.BudgetWallet{
				BudgetId: "budget-1", WalletId: "wallet-1", Network: "base", Address: "0x0000000000000000000000000000000000000001",
				AgentSubject: "agent:research", AgentRunId: "run-42", MandateId: "mandate-1", SessionGeneration: 4,
				SessionExpiresAt: timestamppb.New(now.Add(-time.Second)),
			},
		}
		if _, err := prepareSettlementProofFromClientAt(fixture, request.Msg, key, now); err == nil {
			t.Fatal("expired session produced a settlement proof")
		}
		if fixture.authorizationCalls != 1 || fixture.walletCalls != 1 {
			t.Fatalf("expired session path read wrong records: authorization=%d wallet=%d", fixture.authorizationCalls, fixture.walletCalls)
		}
	})
}

func TestPrepareSettlementProofRejectsChangedPreapprovedInstrumentBeforeWalletLookup(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	key, err := parseSessionPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 22, 0, 0, 0, time.UTC)
	fixture := &proofReaderFixture{
		authorization: &authorizationv1.AuthorizationRecord{
			Id: "auth-approved", MandateId: "mandate-1", BudgetId: "budget-1", RequestingAgent: "agent:research",
			AmountMinor: 125, Currency: "USD", Counterparty: "model:fixture", SettlementId: "settle-1",
			InstrumentId: "instrument-1", SettlementIdempotencyKey: "idem-1", Verdict: authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED,
			ExpiresAt: timestamppb.New(now.Add(time.Hour)),
		},
	}
	request := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-approved", SettlementId: "settle-1", InstrumentId: "instrument-other", IdempotencyKey: "idem-1"})
	if _, err := prepareSettlementProofFromClientAt(fixture, request.Msg, key, now); err == nil {
		t.Fatal("changed instrument produced a session signature")
	}
	if fixture.authorizationCalls != 1 || fixture.walletCalls != 0 {
		t.Fatalf("changed intent reached wallet/signature path: auth=%d wallet=%d", fixture.authorizationCalls, fixture.walletCalls)
	}
}
