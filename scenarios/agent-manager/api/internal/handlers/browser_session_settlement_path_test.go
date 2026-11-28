package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/cli-core/cliutil"
)

// This composes the real AM close-evidence handler and its BAS/LPBS clients.
// BAS and LPBS are isolated HTTP owners in the fixture; LPBS route/ledger
// binding and receipt persistence are separately exercised in the MB17 LPBS
// integration evidence.
func TestBrowserSettlementPathKeepsCloseProofAndLPBSReceiptBoundAcrossRetry(t *testing.T) {
	runID := uuid.New()
	foreignRunID := uuid.New()
	grant, proofRef := browserTestGrantReceipt(runID), "closed-proof-ref"
	const sessionID = "broker-session-path"
	var basRequestCalls, basVerifyCalls, lpbsCalls atomic.Int64
	var effectMu sync.Mutex
	var lpbsBodies []string
	type accountEffects struct {
		reservationStatus string
		gatewayChildren   int
		receipts          int
	}
	accounts := map[string]accountEffects{
		"account-owner-a": {reservationStatus: "pending", gatewayChildren: 1},
		"account-owner-b": {reservationStatus: "pending", gatewayChildren: 1},
	}
	var finalizedReceipt string
	var committedTuple string
	bas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		basRequestCalls.Add(1)
		if r.Header.Get("Authorization") != "Bearer am-bas-proof-token" {
			t.Errorf("BAS auth = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "" {
			t.Error("AM forwarded run or browser-grant identity to BAS")
		}
		if r.URL.Path != "/api/v1/browser/internal/evidence-proof/verify" || r.Method != http.MethodPost {
			t.Errorf("BAS request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode BAS request: %v", err)
		}
		if body["run_id"] != runID.String() || body["grant_receipt"] != grant {
			t.Errorf("BAS close proof binding = %#v", body)
		}
		if body["proof_reference"] == "foreign-proof" {
			http.Error(w, "proof mismatch", http.StatusForbidden)
			return
		}
		basVerifyCalls.Add(1)
		_, _ = fmt.Fprint(w, `{"observation":{"kind":"session","session_id":"`+sessionID+`","state":"closed","session_minutes":7,"actions":19,"bytes":2048}}`)
	}))
	defer bas.Close()

	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lpbsCalls.Add(1)
		if r.Header.Get("Authorization") != "Bearer am-lpbs-settlement-token" {
			t.Errorf("LPBS auth = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "" {
			t.Error("AM forwarded inbound run identity or grant header to LPBS")
		}
		encoded, _ := io.ReadAll(r.Body)
		effectMu.Lock()
		lpbsBodies = append(lpbsBodies, string(encoded))
		effectMu.Unlock()
		var body map[string]string
		if err := json.Unmarshal(encoded, &body); err != nil {
			t.Errorf("decode LPBS request: %v", err)
		}
		if len(body) != 4 || body["run_id"] != runID.String() || body["session_id"] != sessionID || body["grant_receipt"] != grant || body["proof_reference"] != proofRef {
			t.Errorf("LPBS request did not preserve exact close binding: %#v", body)
			http.Error(w, "tuple mismatch", http.StatusConflict)
			return
		}
		effectMu.Lock()
		defer effectMu.Unlock()
		if committedTuple == "" {
			committedTuple = strings.Join([]string{body["run_id"], body["session_id"], body["grant_receipt"], body["proof_reference"]}, "/")
			state := accounts["account-owner-a"]
			state.reservationStatus = "settled"
			state.receipts = 1
			accounts["account-owner-a"] = state
			finalizedReceipt = `{"reservation_id":"reservation-owner-a","status":"settled","business_account_id":"account-owner-a","provider":"multiple","model":"multiple","cost_micros":610,"prompt_tokens":23,"completion_tokens":11,"credits_charged":8,"browser_session_minutes":7,"browser_actions":19,"browser_bytes":2048,"inference_operations":[{"root_operation_id":"gateway-root-1","provider":"hosted-provider","model":"browser-model","credits_charged":8,"cost_micros":610,"prompt_tokens":23,"completion_tokens":11}]}`
		} else if committedTuple != strings.Join([]string{body["run_id"], body["session_id"], body["grant_receipt"], body["proof_reference"]}, "/") {
			http.Error(w, "settlement tuple mismatch", http.StatusConflict)
			return
		}
		if lpbsCalls.Load() == 1 {
			http.Error(w, "settlement response uncertain", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, finalizedReceipt)
	}))
	defer lpbs.Close()

	basTokens := scopedHandlerFixtureToken(t, browserEvidenceVerifierPrincipal, browserEvidenceVerifierAudience, browserEvidenceVerifierScope, "am-bas-proof-token")
	basClient := &DiscoveredBrowserEvidenceProofVerifier{Tokens: basTokens, Client: bas.Client(), ResolveURL: func(context.Context) (string, error) { return bas.URL, nil }}
	lpbsTokens := scopedHandlerFixtureToken(t, browserSessionSettlementPrincipal, browserSessionSettlementAudience, browserSessionSettlementScope, "am-lpbs-settlement-token")
	lpbsClient := &DiscoveredBrowserSessionSettlementClient{Tokens: lpbsTokens, Client: lpbs.Client(), ResolveURL: func(context.Context) (string, error) { return lpbs.URL, nil }}

	server := browserHandlerForTest(runID)
	transcript := &browserTranscriptStoreFake{}
	server.svc.BrowserEvidenceService = transcript
	server.svc.EventService = transcript
	server.browserEvidenceProof = basClient
	server.browserSessionSettlement = lpbsClient
	call := func(targetRun uuid.UUID, subject, reference, grantHeader string, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/bas-evidence/"+targetRun.String(), strings.NewReader(body))
		req = withBASBrowserEvidenceService(req, targetRun, nil)
		if subject != orchestration.BrowserEvidenceServiceSubject {
			principal := coreidentity.Principal{Kind: coreidentity.ActorService, Subject: subject, Verified: true, Scopes: []string{"agent-manager:write"}}
			req = req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
		}
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, grantHeader)
		response := httptest.NewRecorder()
		server.RecordBASBrowserEvidence(response, req)
		return response
	}

	// A wrong AM service principal is refused before either upstream owner sees
	// the request. A mismatched BAS proof is refused before AM appends evidence
	// or calls LPBS.
	if got := call(runID, "other-service", proofRef, grant, `{"proof_reference":"`+proofRef+`"}`); got.Code != http.StatusForbidden {
		t.Fatalf("wrong principal status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(runID, orchestration.BrowserEvidenceServiceSubject, "foreign-proof", grant, `{"proof_reference":"foreign-proof"}`); got.Code != http.StatusForbidden {
		t.Fatalf("mismatched BAS proof status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(foreignRunID, orchestration.BrowserEvidenceServiceSubject, proofRef, browserTestGrantReceipt(foreignRunID), `{"proof_reference":"`+proofRef+`"}`); got.Code != http.StatusUnauthorized {
		t.Fatalf("foreign run tuple status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, "foreign-grant", `{"proof_reference":"`+proofRef+`"}`); got.Code != http.StatusForbidden {
		t.Fatalf("foreign grant tuple status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, grant, `{"proof_reference":"`+proofRef+`","account_id":"account-owner-b","reservation_id":"reservation-owner-b"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("caller-selected account/reservation status=%d body=%s", got.Code, got.Body.String())
	}
	if len(transcript.events) != 0 || basRequestCalls.Load() != 1 || basVerifyCalls.Load() != 0 || lpbsCalls.Load() != 0 {
		t.Fatalf("refused close had effects: transcript=%d BAS requests=%d valid=%d LPBS=%d", len(transcript.events), basRequestCalls.Load(), basVerifyCalls.Load(), lpbsCalls.Load())
	}
	if a, b := accounts["account-owner-a"], accounts["account-owner-b"]; a.reservationStatus != "pending" || a.receipts != 0 || a.gatewayChildren != 1 || b.reservationStatus != "pending" || b.receipts != 0 || b.gatewayChildren != 1 {
		t.Fatalf("preflight denials changed tenant effects: A=%+v B=%+v", a, b)
	}
	t.Logf("preflight zero-effect snapshot: evidence_events=%d BAS_valid_proofs=%d LPBS_requests=%d account_a=%+v account_b=%+v", len(transcript.events), basVerifyCalls.Load(), lpbsCalls.Load(), accounts["account-owner-a"], accounts["account-owner-b"])

	// LPBS returns uncertain on the first attempt. AM retains the recorded close
	// and proof for the same exact retry, then passes the finalized joint receipt
	// through without creating a second evidence event.
	first := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, grant, `{"proof_reference":"`+proofRef+`"}`)
	if first.Code != http.StatusAccepted || !strings.Contains(first.Body.String(), `"status":"pending"`) || !strings.Contains(first.Body.String(), `"recorded":true`) {
		t.Fatalf("uncertain settlement response=%d body=%s", first.Code, first.Body.String())
	}
	// Race an exact A retry batch against foreign B run attempts while the
	// settled receipt is still pending at AM. A's retry remains idempotent; B
	// must be rejected by AM's run/grant owner before another BAS proof check or
	// LPBS settlement call.
	const concurrentRetries = 12
	var wg sync.WaitGroup
	errs := make(chan string, concurrentRetries*2)
	for i := 0; i < concurrentRetries; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			got := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, grant, `{"proof_reference":"`+proofRef+`"}`)
			if got.Code != http.StatusAccepted || !strings.Contains(got.Body.String(), `"status":"settled"`) {
				errs <- fmt.Sprintf("owner retry status=%d body=%s", got.Code, got.Body.String())
			}
		}()
		go func() {
			defer wg.Done()
			foreignGrant := browserTestGrantReceipt(foreignRunID)
			got := call(foreignRunID, orchestration.BrowserEvidenceServiceSubject, proofRef, foreignGrant, `{"proof_reference":"`+proofRef+`"}`)
			if got.Code != http.StatusUnauthorized {
				errs <- fmt.Sprintf("foreign run attempt status=%d body=%s", got.Code, got.Body.String())
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	second := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, grant, `{"proof_reference":"`+proofRef+`"}`)
	if second.Code != http.StatusAccepted || !strings.Contains(second.Body.String(), `"idempotent":true`) || !strings.Contains(second.Body.String(), `"status":"settled"`) {
		t.Fatalf("settlement retry response=%d body=%s", second.Code, second.Body.String())
	}
	third := call(runID, orchestration.BrowserEvidenceServiceSubject, proofRef, grant, `{"proof_reference":"`+proofRef+`"}`)
	if third.Code != http.StatusAccepted || !strings.Contains(third.Body.String(), `"status":"settled"`) {
		t.Fatalf("settled receipt replay response=%d body=%s", third.Code, third.Body.String())
	}
	// The 12 foreign attempts above must be effect-free. A's concurrent retries
	// plus the exact sequential replays converge on one receipt, one AM event,
	// and one immutable Gateway child.
	if len(transcript.events) != 1 || basRequestCalls.Load() != 16 || basVerifyCalls.Load() != 15 || lpbsCalls.Load() != 15 {
		t.Fatalf("retry duplicated or dropped effects: evidence=%d BAS requests=%d valid=%d LPBS calls=%d", len(transcript.events), basRequestCalls.Load(), basVerifyCalls.Load(), lpbsCalls.Load())
	}
	if a, b := accounts["account-owner-a"], accounts["account-owner-b"]; a.reservationStatus != "settled" || a.receipts != 1 || a.gatewayChildren != 1 || b.reservationStatus != "pending" || b.receipts != 0 || b.gatewayChildren != 1 {
		t.Fatalf("retry or foreign attempts changed owner effects: A=%+v B=%+v", a, b)
	}
	t.Logf("settled retry snapshot: evidence_events=%d BAS_valid_proofs=%d LPBS_requests=%d account_a=%+v account_b=%+v", len(transcript.events), basVerifyCalls.Load(), lpbsCalls.Load(), accounts["account-owner-a"], accounts["account-owner-b"])
	effectMu.Lock()
	for i, body := range lpbsBodies {
		if strings.Contains(body, "account_id") || strings.Contains(body, "reservation_id") {
			t.Fatalf("request %d selected account or reservation: %s", i, body)
		}
	}
	effectMu.Unlock()
	var result struct {
		Settlement struct {
			Status  string                `json:"status"`
			Receipt BrowserSessionReceipt `json:"receipt"`
		} `json:"settlement"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	r := result.Settlement.Receipt
	if result.Settlement.Status != "settled" || r.ReservationID != "reservation-owner-a" || r.BusinessAccountID != "account-owner-a" || r.SessionMinutes != 7 || r.Actions != 19 || r.Bytes != 2048 || r.PromptTokens != 23 || r.CompletionTokens != 11 || len(r.InferenceOperations) != 1 || r.InferenceOperations[0].Provider != "hosted-provider" || r.InferenceOperations[0].Model != "browser-model" {
		t.Fatalf("joint browser/token receipt did not reach AM caller: %+v", result.Settlement)
	}
	data, ok := transcript.events[0].Data.(*domain.TypedEventData)
	if !ok || data.Type != domain.EventTypeBrowserSession {
		t.Fatalf("AM transcript event type=%T event=%+v", transcript.events[0].Data, transcript.events[0])
	}
	var payload struct {
		State          string `json:"state"`
		SessionID      string `json:"session_id"`
		SessionMinutes int64  `json:"session_minutes"`
		Actions        int64  `json:"actions"`
		Bytes          int64  `json:"bytes"`
		ReportedBy     string `json:"reported_by"`
	}
	if err := json.Unmarshal(data.Body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != "closed" || payload.SessionID != sessionID || payload.SessionMinutes != 7 || payload.Actions != 19 || payload.Bytes != 2048 || payload.ReportedBy != "browser-automation-studio-broker" {
		t.Fatalf("AM transcript event is not exact broker close evidence: %+v", payload)
	}
}

func scopedHandlerFixtureToken(t *testing.T, subject, audience, scope, access string) *servicetoken.Tokens {
	t.Helper()
	return servicetoken.New(servicetoken.Config{
		Credential: func(context.Context) (string, error) { return "fixture", nil },
		Expect:     servicetoken.ExpectIdentity(subject, audience, scope),
	}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: access, ExpiresAt: time.Now().Add(time.Hour), Subject: subject, Audience: audience, Scopes: []string{scope}}, nil
	}))
}
