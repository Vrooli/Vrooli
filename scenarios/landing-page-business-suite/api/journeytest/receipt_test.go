package journeytest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"landing-page-business-suite-api/internal/aidoor"
)

type lostReceiptResponseTransport struct{}

func (lostReceiptResponseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	return nil, errors.New("fixture: entire response lost before caller received headers or body")
}

type receiptReadTransport struct{ reads atomic.Int32 }

func (tr *receiptReadTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "/provider-routes/receipt/") {
		tr.reads.Add(1)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func (s *spine) readReceipt(t *testing.T, token, operation string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/v1/ai/receipt", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if operation != "" {
		req.Header.Set("Idempotency-Key", operation)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("receipt is cacheable: %v", response.Header)
	}
	return response.StatusCode, string(body)
}

func requirePublicReceipt(t *testing.T, code int, body string, wantCode int, want string) {
	t.Helper()
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &fields) != nil {
		t.Fatalf("invalid receipt: %s", body)
	}
	// Exact wire allowlist also proves private content/model/supplier/cost/operation
	// identities and tokens cannot leak through receipt projection.
	if code != wantCode || strings.TrimSpace(body) != want {
		t.Fatalf("receipt=%d %s; want %d %s", code, body, wantCode, want)
	}
	for field := range fields {
		if field != "status" && field != "charged_credits" {
			t.Fatalf("private field %q in public receipt", field)
		}
	}
}

func TestPublicReceiptRecoversLostResponseWithoutRedispatchOrPrivateContent(t *testing.T) {
	var dispatches atomic.Int32
	s := newSpine(t, func([]byte) { dispatches.Add(1) })
	// The real Gateway commits its receipt, but LPBS cannot settle until this
	// temporary local database failure is repaired. The durable hold must survive.
	_, err := s.db.Exec(`CREATE FUNCTION fixture_receipt_settlement_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture settlement unavailable'; END $$;
 CREATE TRIGGER fixture_receipt_failure BEFORE INSERT ON business_account_usage_events FOR EACH ROW EXECUTE FUNCTION fixture_receipt_settlement_failure()`)
	if err != nil {
		t.Fatal(err)
	}
	const operation = "lost-response-recovery"
	req, err := http.NewRequest(http.MethodPost, s.server.URL+"/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture-model","stream":true,"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.key.Token)
	req.Header.Set("Idempotency-Key", operation)
	req.Header.Set("Content-Type", "application/json")
	if response, err := (&http.Client{Transport: lostReceiptResponseTransport{}}).Do(req); err == nil || response != nil {
		t.Fatalf("response was not completely lost: response=%v err=%v", response, err)
	}
	assertCharged(t, s, 0)
	code, body := s.readReceipt(t, s.key.Token, operation)
	requirePublicReceipt(t, code, body, http.StatusAccepted, `{"status":"pending"}`)
	retry := s.request(t, s.key.Token, operation, "hello")
	io.Copy(io.Discard, retry.Body)
	if retry.StatusCode != http.StatusConflict {
		t.Fatalf("pending retry status=%d", retry.StatusCode)
	}
	if _, err := s.db.Exec(`DROP TRIGGER fixture_receipt_failure ON business_account_usage_events`); err != nil {
		t.Fatal(err)
	}

	// Recreate the LPBS handler to prove recovery uses durable owner rows, not
	// memory retained by the request that lost its response.
	door := s.server.Config.Handler.(*aidoor.Door)
	s.server.Close()
	s.server = httptest.NewServer(&aidoor.Door{Keys: door.Keys, Wallet: door.Wallet, GatewayURL: door.GatewayURL})
	t.Cleanup(s.server.Close)
	for i := 0; i < 2; i++ {
		code, body = s.readReceipt(t, s.key.Token, operation)
		requirePublicReceipt(t, code, body, http.StatusOK, `{"status":"settled","charged_credits":200}`)
	}
	retry = s.request(t, s.key.Token, operation, "hello")
	io.Copy(io.Discard, retry.Body)
	if retry.StatusCode != http.StatusConflict {
		t.Fatalf("settled retry status=%d", retry.StatusCode)
	}
	assertCharged(t, s, 200)
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || dispatches.Load() != 1 {
		t.Fatalf("duplicated effects: charges=%d dispatches=%d", count, dispatches.Load())
	}
	var charged, capActual int64
	if err := s.db.QueryRow(`SELECT r.charged_amount,c.actual_credits FROM business_account_credit_reservations r JOIN ai_key_charges c ON c.operation_id=CAST(r.id AS TEXT) WHERE c.key_id=$1`, s.key.ID).Scan(&charged, &capActual); err != nil {
		t.Fatal(err)
	}
	if charged != 200 || capActual != 200 {
		t.Fatalf("wallet/cap receipt mismatch: %d/%d", charged, capActual)
	}
}

func TestPublicReceiptRequiresOriginalKeyAndAccountAndDoesNotProbeUnknownOperations(t *testing.T) {
	s := newSpine(t)
	response := s.request(t, s.key.Token, "owned-operation", "hello")
	io.Copy(io.Discard, response.Body)
	keys := aidoor.KeyStore{DB: s.db}
	sameAccount, err := keys.IssueDevice(context.Background(), "account", "actor", "journey", 10000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO business_accounts(id,display_name,billing_email) VALUES('other-account','Other','other@example.invalid'); INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('other-account','other-actor','owner')`); err != nil {
		t.Fatal(err)
	}
	otherAccount, err := keys.IssueDevice(context.Background(), "other-account", "other-actor", "", 10000)
	if err != nil {
		t.Fatal(err)
	}
	tracker := &receiptReadTransport{}
	door := s.server.Config.Handler.(*aidoor.Door)
	s.server.Close()
	s.server = httptest.NewServer(&aidoor.Door{Keys: door.Keys, Wallet: door.Wallet, GatewayURL: door.GatewayURL, Client: &http.Client{Transport: tracker}})
	t.Cleanup(s.server.Close)
	for _, tc := range []struct{ name, token, operation string }{
		{"unknown", s.key.Token, "never-dispatched"},
		{"same-account-wrong-key", sameAccount.Token, "owned-operation"},
		{"wrong-account", otherAccount.Token, "owned-operation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, body := s.readReceipt(t, tc.token, tc.operation)
			requirePublicReceipt(t, code, body, http.StatusNotFound, `{"status":"unknown"}`)
		})
	}
	if tracker.reads.Load() != 0 {
		t.Fatalf("unowned request reached private Gateway: %d", tracker.reads.Load())
	}
	var internalOperation string
	if err := s.db.QueryRow(`SELECT id::text FROM business_account_credit_reservations`).Scan(&internalOperation); err != nil {
		t.Fatal(err)
	}
	code, body := s.readReceipt(t, s.key.Token, internalOperation)
	requirePublicReceipt(t, code, body, http.StatusNotFound, `{"status":"unknown"}`)
	// The claim cannot override a conflicting reservation account, even in an
	// inconsistent test DB. No private owner read may follow this mismatch.
	if _, err := s.db.Exec(`UPDATE business_account_credit_reservations SET business_account_id='other-account'`); err != nil {
		t.Fatal(err)
	}
	code, body = s.readReceipt(t, s.key.Token, "owned-operation")
	requirePublicReceipt(t, code, body, http.StatusNotFound, `{"status":"unknown"}`)
	if tracker.reads.Load() != 0 {
		t.Fatal("raw operation or conflicting account reached private Gateway")
	}
	if _, err := s.db.Exec(`UPDATE business_account_credit_reservations SET business_account_id='account'`); err != nil {
		t.Fatal(err)
	}
	code, body = s.readReceipt(t, s.key.Token, "owned-operation")
	requirePublicReceipt(t, code, body, http.StatusOK, `{"status":"settled","charged_credits":200}`)
	if code, _ := s.readReceipt(t, s.key.Token, ""); code != http.StatusBadRequest {
		t.Fatalf("missing caller key status=%d", code)
	}
	if err := keys.Revoke(context.Background(), s.key, s.key.ID); err != nil {
		t.Fatal(err)
	}
	if code, _ := s.readReceipt(t, s.key.Token, "owned-operation"); code != http.StatusUnauthorized {
		t.Fatalf("revoked key status=%d", code)
	}
	assertCharged(t, s, 200)
}

func TestPublicReceiptDistinguishesPendingFromUnknownAndReleased(t *testing.T) {
	s := newSpine(t)
	// A fully admitted hold without any Gateway receipt is unresolved, never a
	// zero-cost completion and never permission to dispatch from a read route.
	door := s.server.Config.Handler.(*aidoor.Door)
	const operation = "receipt-pending-no-gateway-result"
	// Let the Gateway plan and LPBS admission run, then fail only stream dispatch
	// ambiguously. This transport deliberately cannot prove non-dispatch.
	door.Client = &http.Client{Transport: pendingReceiptTransport{}}
	response := s.request(t, s.key.Token, operation, "hello")
	io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("uncertain dispatch=%d", response.StatusCode)
	}
	code, body := s.readReceipt(t, s.key.Token, operation)
	requirePublicReceipt(t, code, body, http.StatusAccepted, `{"status":"pending"}`)
	assertCharged(t, s, 0)
	door.Client = nil
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET remaining=0`); err != nil {
		t.Fatal(err)
	}
	response = s.request(t, s.key.Token, "refused-no-credit", "hello")
	io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("refused admission=%d", response.StatusCode)
	}
	code, body = s.readReceipt(t, s.key.Token, "refused-no-credit")
	requirePublicReceipt(t, code, body, http.StatusOK, `{"status":"released","charged_credits":0}`)
}

type pendingReceiptTransport struct{}

func (pendingReceiptTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasSuffix(r.URL.Path, "/provider-routes/stream") {
		return nil, errors.New("fixture: ambiguous dispatch")
	}
	return http.DefaultTransport.RoundTrip(r)
}

func TestPublicReceiptReturnsCappedWalletChargeForUnknownProviderOutcome(t *testing.T) {
	var dispatches atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatches.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"private incomplete content\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"cost\":0.000020}}\n\n")
		// Missing DONE means completion is unknown even though cost is known.
	}))
	defer upstream.Close()
	t.Setenv("VROOLI_JOURNEY_ROUTER_URL", upstream.URL)
	s := newSpine(t)
	response := s.request(t, s.key.Token, "unknown-outcome-overrun", "hello")
	io.Copy(io.Discard, response.Body)
	operation := response.Header.Get("X-Operation-ID")
	privateResponse, err := http.Get(s.gateway + "/api/v1/private/provider-routes/receipt/" + operation)
	if err != nil {
		t.Fatal(err)
	}
	defer privateResponse.Body.Close()
	var private struct {
		State     string `json:"state"`
		Credits   int64  `json:"credits"`
		CostKnown bool   `json:"cost_known"`
	}
	if err := json.NewDecoder(privateResponse.Body).Decode(&private); err != nil {
		t.Fatal(err)
	}
	if privateResponse.StatusCode != http.StatusOK || private.State != "unknown" || private.Credits != 2000 || !private.CostKnown {
		t.Fatalf("expected unknown outcome with known measured overrun: %+v status=%d", private, privateResponse.StatusCode)
	}
	code, body := s.readReceipt(t, s.key.Token, "unknown-outcome-overrun")
	requirePublicReceipt(t, code, body, http.StatusOK, `{"status":"settled","charged_credits":1000}`)
	assertCharged(t, s, 1000)
	if dispatches.Load() != 1 {
		t.Fatalf("dispatches=%d", dispatches.Load())
	}
}
