package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lpbsidentity"
	billing "landing-page-business-suite-api/handlers/commerce"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/intelligence"
	_ "modernc.org/sqlite"
)

// meteredUsageAdapter embeds the real LPBS owner and supplies only the
// unrelated legacy operations required by MeteredInferenceService's narrow
// interface. Browser settlement methods stay promoted from UsageService.
type meteredUsageAdapter struct{ *commerce.UsageService }

func (meteredUsageAdapter) ReserveAndCharge(context.Context, string, string, string, int64, intelligence.UsageReport) error {
	return errors.New("unexpected generic reserve in browser close fixture")
}
func (meteredUsageAdapter) ReserveCredits(context.Context, string, string, string, int64) (string, error) {
	return "", errors.New("unexpected generic reservation in browser close fixture")
}
func (meteredUsageAdapter) FinalizeReservation(context.Context, string, int64) error {
	return errors.New("unexpected generic finalize in browser close fixture")
}
func (meteredUsageAdapter) ReleaseReservation(context.Context, string) error {
	return errors.New("unexpected generic release in browser close fixture")
}
func (meteredUsageAdapter) AdjustUsage(context.Context, string, string, int64, string) error {
	return errors.New("unexpected generic adjustment in browser close fixture")
}
func (meteredUsageAdapter) RecordUsage(context.Context, intelligence.UsageReport) error {
	return errors.New("unexpected generic usage record in browser close fixture")
}

// fixtureAMCloseAttester is an owner-bound test double: it accepts only the
// exact run/session/grant/proof/reservation tuple seeded for account A.
type fixtureAMCloseAttester struct {
	accepted intelligence.BrowserEvidenceProof
	calls    []intelligence.BrowserEvidenceProof
}

func (a *fixtureAMCloseAttester) VerifyAndConsumeBrowserEvidence(_ context.Context, proof intelligence.BrowserEvidenceProof) (intelligence.AttestedBrowserUsage, error) {
	a.calls = append(a.calls, proof)
	if proof != a.accepted {
		return intelligence.AttestedBrowserUsage{}, errors.New("fixture AM rejected foreign or mismatched close proof")
	}
	return intelligence.AttestedBrowserUsage{
		RunID: proof.RunID, ReservationID: proof.ReservationID,
		SessionID: "bas-session-tenant-a", State: "closed",
		SessionMinutes: 4, Actions: 12, Bytes: 4096,
	}, nil
}

func TestAgentManagerPrincipalSettlesOnlyItsResolvedTenantBinding(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,billing_period TEXT NOT NULL,limit_key TEXT NOT NULL,reserved_amount INTEGER NOT NULL,status TEXT NOT NULL,actual_amount INTEGER,charged_amount INTEGER NOT NULL DEFAULT 0,overrun_amount INTEGER NOT NULL DEFAULT 0,cost_micros INTEGER NOT NULL DEFAULT 0,prompt_tokens INTEGER NOT NULL DEFAULT 0,completion_tokens INTEGER NOT NULL DEFAULT 0,app_bundle_key TEXT NOT NULL DEFAULT '',model TEXT NOT NULL DEFAULT '',provider TEXT NOT NULL DEFAULT '',livemode BOOLEAN NOT NULL DEFAULT TRUE,browser_session_minutes INTEGER NOT NULL DEFAULT 0,browser_actions INTEGER NOT NULL DEFAULT 0,browser_bytes INTEGER NOT NULL DEFAULT 0,browser_run_id TEXT NOT NULL DEFAULT '',browser_session_id TEXT NOT NULL DEFAULT '',browser_grant_fingerprint TEXT NOT NULL DEFAULT '',browser_evidence_fingerprint TEXT NOT NULL DEFAULT '',browser_tier TEXT NOT NULL DEFAULT '',settled_at DATETIME)`,
		`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT NOT NULL,gateway_root_operation_id TEXT NOT NULL UNIQUE,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,request_fingerprint TEXT NOT NULL,reserved_amount INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL,provider TEXT NOT NULL DEFAULT '',model TEXT NOT NULL DEFAULT '',credits INTEGER NOT NULL DEFAULT 0,cost_micros INTEGER NOT NULL DEFAULT 0,prompt_tokens INTEGER NOT NULL DEFAULT 0,completion_tokens INTEGER NOT NULL DEFAULT 0,receipt_fingerprint TEXT NOT NULL DEFAULT '',created_at DATETIME,settled_at DATETIME,PRIMARY KEY(reservation_id,gateway_root_operation_id))`,
		`CREATE TABLE business_account_usage_events (operation_id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,app_bundle_key TEXT NOT NULL,model TEXT NOT NULL,credits INTEGER NOT NULL,cost_micros INTEGER NOT NULL,prompt_tokens INTEGER NOT NULL,completion_tokens INTEGER NOT NULL,provider TEXT NOT NULL,browser_session_minutes INTEGER NOT NULL DEFAULT 0,browser_actions INTEGER NOT NULL DEFAULT 0,browser_bytes INTEGER NOT NULL DEFAULT 0,browser_run_id TEXT NOT NULL DEFAULT '',browser_session_id TEXT NOT NULL DEFAULT '',browser_evidence_fingerprint TEXT NOT NULL DEFAULT '',feature_id TEXT NOT NULL DEFAULT '',created_at DATETIME)`,
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,billing_period TEXT NOT NULL,limit_key TEXT NOT NULL,usage_amount INTEGER NOT NULL,app_bundle_key TEXT NOT NULL,last_operation_at DATETIME,created_at DATETIME,updated_at DATETIME,PRIMARY KEY(business_account_id,billing_period,limit_key,app_bundle_key))`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create two-account fixture schema: %v", err)
		}
	}

	const (
		reservationA = "00000000-0000-4000-8000-000000000117"
		reservationB = "00000000-0000-4000-8000-000000000217"
		runA         = "00000000-0000-4000-8000-000000000118"
		runB         = "00000000-0000-4000-8000-000000000218"
		sessionA     = "bas-session-tenant-a"
		sessionB     = "bas-session-tenant-b"
		grantA       = "fixture-grant-a"
		grantB       = "fixture-grant-b"
		proofA       = "fixture-close-proof-a"
	)
	fingerprint := func(value string) string {
		digest := sha256.Sum256([]byte(value))
		return hex.EncodeToString(digest[:])
	}
	for _, row := range []struct {
		reservation, account, actor, run, session, grant string
	}{
		{reservationA, "tenant-account-a", "member-a@example.test", runA, sessionA, fingerprint(grantA)},
		{reservationB, "tenant-account-b", "member-b@example.test", runB, sessionB, fingerprint(grantB)},
	} {
		_, err := db.Exec(`INSERT INTO business_account_credit_reservations(id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,app_bundle_key,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_tier,livemode) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			row.reservation, row.account, row.actor, "2026-11", "ai_credits", 0, "pending", "web-console", row.run, row.session, row.grant, "pro", true)
		if err != nil {
			t.Fatalf("seed %s hold: %v", row.account, err)
		}
		_, err = db.Exec(`INSERT INTO business_account_browser_gateway_receipts(reservation_id,gateway_root_operation_id,business_account_id,actor_identity,request_fingerprint,reserved_amount,state,provider,model,credits,cost_micros,prompt_tokens,completion_tokens,receipt_fingerprint) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			row.reservation, "gateway-root-"+row.account, row.account, row.actor, fingerprint("request-"+row.account), 0, "settled", "fixture-provider", "fixture-model", 0, 0, 0, 0, fingerprint("receipt-"+row.account))
		if err != nil {
			t.Fatalf("seed %s immutable child receipt: %v", row.account, err)
		}
	}

	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: db, Dialect: "sqlite"})
	meteredUsage := meteredUsageAdapter{UsageService: usage}
	attester := &fixtureAMCloseAttester{accepted: intelligence.BrowserEvidenceProof{RunID: runA, GrantReceipt: grantA, Reference: proofA, ReservationID: reservationA}}
	metered := intelligence.NewMeteredInferenceService(intelligence.MeteredInferenceServiceOptions{
		UsageService: meteredUsage, BrowserEvidenceAttester: attester,
	})
	writeError := func(w http.ResponseWriter, status int, message, code string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "code": code})
	}
	deps := billing.UsageDependencies{WriteError: writeError}
	role := lpbsidentity.AgentManagerBrowserSessionSettlement
	validPrincipal := apiidentity.Principal{Kind: apiidentity.ActorService, Subject: role.Principal("agent-manager"), Verified: true, Audience: role.Audience, Scopes: []string{role.Scope}}
	callSettlement := func(principal apiidentity.Principal, payload string) *httptest.ResponseRecorder {
		server := &Server{servicePrincipalVerifier: staticServicePrincipalVerifier{principal: principal}}
		handler := server.requireAgentManagerBrowserSessionSettlement(billing.SettleAgentManagerBrowserSession(usage, metered, deps))
		request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/browser-sessions/settle-and-receipt", strings.NewReader(payload))
		request.Header.Set("Authorization", "Bearer fixture-service-token")
		response := httptest.NewRecorder()
		handler(response, request)
		return response
	}

	// Account and reservation identifiers are neither accepted nor needed at
	// this owner boundary. Reject caller-selected IDs before owner resolution or
	// the AM proof callback can run.
	guessedIDs, _ := json.Marshal(map[string]string{"run_id": runA, "session_id": sessionA, "grant_receipt": grantA, "proof_reference": proofA, "account_id": "tenant-account-b", "reservation_id": reservationB})
	if response := callSettlement(validPrincipal, string(guessedIDs)); response.Code != http.StatusBadRequest {
		t.Fatalf("caller-supplied account/reservation fields status=%d body=%s", response.Code, response.Body.String())
	}
	if len(attester.calls) != 0 {
		t.Fatalf("invalid authority selectors reached AM attester: %+v", attester.calls)
	}
	validBody, _ := json.Marshal(map[string]string{"run_id": runA, "session_id": sessionA, "grant_receipt": grantA, "proof_reference": proofA})
	wrongRole := validPrincipal
	wrongRole.Scopes = []string{"lpbs:usage.read"}
	if response := callSettlement(wrongRole, string(validBody)); response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong service principal status=%d body=%s", response.Code, response.Body.String())
	}
	if len(attester.calls) != 0 {
		t.Fatalf("wrong service principal reached AM attester: %+v", attester.calls)
	}

	// The exact AM principal closes account A's prebound hold without any
	// tenant/account/reservation identifiers in the request.
	response := callSettlement(validPrincipal, string(validBody))
	if response.Code != http.StatusOK {
		t.Fatalf("valid AM close status=%d body=%s", response.Code, response.Body.String())
	}
	var receipt commerce.BrowserSessionReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode close receipt: %v", err)
	}
	if receipt.BusinessAccountID != "tenant-account-a" || receipt.ReservationID != reservationA || receipt.Status != "settled" || receipt.SessionMinutes != 4 || receipt.Actions != 12 || receipt.Bytes != 4096 {
		t.Fatalf("AM settlement returned wrong owner receipt: %+v", receipt)
	}
	if len(attester.calls) != 1 || attester.calls[0] != (intelligence.BrowserEvidenceProof{RunID: runA, GrantReceipt: grantA, Reference: proofA, ReservationID: reservationA}) {
		t.Fatalf("AM callback did not receive only LPBS-resolved A binding: %+v", attester.calls)
	}

	// A service-principal caller that presents account B's unrelated run tuple
	// cannot read or mutate A. The fixture AM owner rejects B, and the LPBS
	// ledger must leave both reservations and child receipts unchanged.
	foreignBody, _ := json.Marshal(map[string]string{"run_id": runB, "session_id": sessionB, "grant_receipt": grantB, "proof_reference": "fixture-close-proof-b"})
	if response := callSettlement(validPrincipal, string(foreignBody)); response.Code != http.StatusConflict {
		t.Fatalf("foreign AM close status=%d body=%s", response.Code, response.Body.String())
	}
	if len(attester.calls) != 2 || attester.calls[1].RunID != runB || attester.calls[1].ReservationID != reservationB {
		t.Fatalf("foreign request did not remain bound to B's owner record: %+v", attester.calls)
	}

	// The customer receipt read path independently resolves account membership;
	// B's requested account plus A's opaque reservation is hidden as not found.
	readDeps := billing.UsageDependencies{
		UserID:    func(context.Context) string { return "tenant-b-user" },
		UserEmail: func(context.Context) string { return "member-b@example.test" },
		ResolveBusinessAccount: func(_ context.Context, userID, _, requested string) (string, error) {
			if userID == "tenant-b-user" && requested == "tenant-account-b" {
				return "tenant-account-b", nil
			}
			return "", errors.New("account membership denied")
		},
		WriteError: writeError,
	}
	read := billing.GetBusinessAccountBrowserSessionReceipt(usage, readDeps)
	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/business-accounts/browser-sessions/"+reservationA+"?business_account_id="+url.QueryEscape("tenant-account-b"), nil)
	readRequest = mux.SetURLVars(readRequest, map[string]string{"reservationID": reservationA})
	readResponse := httptest.NewRecorder()
	read.ServeHTTP(readResponse, readRequest)
	if readResponse.Code != http.StatusNotFound {
		t.Fatalf("tenant B receipt lookup status=%d body=%s", readResponse.Code, readResponse.Body.String())
	}

	for _, row := range []struct{ reservation, account, wantStatus string }{
		{reservationA, "tenant-account-a", "finalized"},
		{reservationB, "tenant-account-b", "pending"},
	} {
		var account, status string
		if err := db.QueryRow(`SELECT business_account_id,status FROM business_account_credit_reservations WHERE id=?`, row.reservation).Scan(&account, &status); err != nil {
			t.Fatal(err)
		}
		if account != row.account || status != row.wantStatus {
			t.Fatalf("reservation %s changed unexpectedly: account=%s status=%s", row.reservation, account, status)
		}
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events WHERE business_account_id='tenant-account-b'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("foreign account B received settlement effects: events=%d", rows)
	}
	var childState, childFingerprint string
	if err := db.QueryRow(`SELECT state,receipt_fingerprint FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND business_account_id=?`, reservationB, "tenant-account-b").Scan(&childState, &childFingerprint); err != nil {
		t.Fatal(err)
	}
	if childState != "settled" || childFingerprint != fingerprint("receipt-tenant-account-b") {
		t.Fatalf("foreign request changed B child receipt: state=%s fingerprint=%s", childState, childFingerprint)
	}
}
