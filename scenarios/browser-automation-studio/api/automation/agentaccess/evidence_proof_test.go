package agentaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type browserEvidenceTestTokenVerifier struct {
	principal identity.Principal
	err       error
}

type browserEvidenceTestTokenSource string

func (s browserEvidenceTestTokenSource) Token(context.Context) (string, error) { return string(s), nil }

func (v browserEvidenceTestTokenVerifier) Verify(context.Context, string) (identity.Principal, error) {
	return v.principal, v.err
}

func closedEvidenceFixture(t *testing.T) (*Broker, *entry, uuid.UUID) {
	t.Helper()
	runID := uuid.New()
	e := &entry{
		sessionID: "bas-session-1", leaseID: "immutable-lease-1", ownerID: "bas-owner-1",
		callerID: "run:" + runID.String(), grantReceipt: "browser-grant:opaque", permissionRef: "profile-observe",
		backend: "managed", openedAt: time.Now().Add(-2 * time.Minute), closedAt: time.Now().UTC(), closed: true,
		evidenceActions: 4, evidenceBytes: 8192,
	}
	b := &Broker{entries: map[string]*entry{e.sessionID: e}}
	if err := b.issueClosedEvidenceProof(e, e.closedAt); err != nil {
		t.Fatalf("issue proof: %v", err)
	}
	return b, e, runID
}

func TestClosedBrowserEvidenceProofBindsRunGrantAndFinalizedEntry(t *testing.T) {
	b, e, runID := closedEvidenceFixture(t)
	proof := e.evidenceProof
	got, err := b.VerifyBrowserEvidenceProof(context.Background(), runID, e.grantReceipt, proof.Reference)
	if err != nil {
		t.Fatalf("verify proof: %v", err)
	}
	if got.Kind != "session" || got.SessionID != e.sessionID || got.State != "closed" || got.Actions != 4 || got.Bytes != 8192 || got.SessionMinutes < 1 {
		t.Fatalf("unexpected broker summary: %+v", got)
	}
	if _, err := b.VerifyBrowserEvidenceProof(context.Background(), uuid.New(), e.grantReceipt, proof.Reference); !errors.Is(err, ErrBrowserEvidenceProof) {
		t.Fatalf("wrong run accepted: %v", err)
	}
	if _, err := b.VerifyBrowserEvidenceProof(context.Background(), runID, "different-grant", proof.Reference); !errors.Is(err, ErrBrowserEvidenceProof) {
		t.Fatalf("wrong grant accepted: %v", err)
	}
	if _, err := b.VerifyBrowserEvidenceProof(context.Background(), runID, e.grantReceipt, "caller-chosen"); !errors.Is(err, ErrBrowserEvidenceProof) {
		t.Fatalf("unknown proof accepted: %v", err)
	}
	if err := b.ConsumeBrowserEvidenceProof(context.Background(), runID, proof.Reference); err != nil {
		t.Fatalf("consume proof: %v", err)
	}
	if err := b.ConsumeBrowserEvidenceProof(context.Background(), runID, proof.Reference); err != nil {
		t.Fatalf("identical consume retry should be idempotent: %v", err)
	}
	if _, err := b.VerifyBrowserEvidenceProof(context.Background(), runID, e.grantReceipt, proof.Reference); err != nil {
		t.Fatalf("identical verify retry should remain idempotent: %v", err)
	}
}

func TestUserAttachedCloseRetainsEntryUntilProofVerification(t *testing.T) {
	runID := uuid.New()
	callerID := "run:" + runID.String()
	profile := sessionpermission.Profile{ID: "profile-attached", Effect: sessionpermission.EffectAct, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	e := &entry{
		sessionID: "user-attached-session", leaseID: "attached-lease", ownerID: "attached-owner",
		callerID: callerID, grantReceipt: "attached-grant", permissionRef: profile.ID,
		backend: string(sessionpermission.BackendUserAttached), profile: profile, policy: policy,
		openedAt: time.Now().Add(-time.Minute), evidenceActions: 2, evidenceBytes: 4096,
	}
	b := &Broker{
		caller: bridgeCaller(callerID), profiles: bridgeProfileResolver{profile: profile, authorizer: policy},
		evidence: bridgeNoopEvidence{}, entries: map[string]*entry{e.sessionID: e},
	}
	ctx := context.Background()
	if err := b.Close(ctx, e.sessionID); err != nil {
		t.Fatalf("close user-attached session: %v", err)
	}
	if e.evidenceProof == nil {
		t.Fatal("closed user-attached session did not issue proof")
	}
	observation, err := b.VerifyBrowserEvidenceProof(ctx, runID, e.grantReceipt, e.evidenceProof.Reference)
	if err != nil {
		t.Fatalf("verify proof after close: %v", err)
	}
	if observation.SessionID != e.sessionID || observation.State != "closed" || observation.Actions != 2 || observation.Bytes != 4096 {
		t.Fatalf("unexpected user-attached close observation: %+v", observation)
	}
}

type settlementReceiptReporter struct{ bridgeNoopEvidence }

func (settlementReceiptReporter) RecordClosedBrowserSession(_ context.Context, _ string, sessionID, _ string, _ string) (BrowserSettlementResult, error) {
	return BrowserSettlementResult{Status: "settled", Receipt: json.RawMessage(`{"reservation_id":"lpbs-hold","status":"settled","browser_session_minutes":2,"browser_actions":4,"browser_bytes":8192}`)}, nil
}

func TestClosedSessionExposesOnlyItsAuthenticatedSettlementReceipt(t *testing.T) {
	b, e, _ := closedEvidenceFixture(t)
	b.caller = bridgeCaller(e.callerID)
	e.identityToken = "signed-run"
	b.evidence = settlementReceiptReporter{}
	if err := b.finishCloseEvidence(context.Background(), e.sessionID, e, e.evidenceProof); err != nil {
		t.Fatalf("finish close callback: %v", err)
	}
	ctx := WithBrowserAgentCredentials(context.Background(), e.identityToken, e.grantReceipt)
	got, err := b.SettlementReceipt(ctx, e.sessionID)
	if err != nil {
		t.Fatalf("owner receipt: %v", err)
	}
	if got.Status != "settled" || string(got.Receipt) != `{"reservation_id":"lpbs-hold","status":"settled","browser_session_minutes":2,"browser_actions":4,"browser_bytes":8192}` {
		t.Fatalf("LPBS receipt changed in BAS: %+v", got)
	}
	b.caller = bridgeCaller("run:" + uuid.NewString())
	if _, err := b.SettlementReceipt(ctx, e.sessionID); err == nil {
		t.Fatal("different run read the session's settlement receipt")
	}
	b.caller = bridgeCaller(e.callerID)
	if _, err := b.SettlementReceipt(WithBrowserAgentCredentials(context.Background(), "different-identity", e.grantReceipt), e.sessionID); err == nil {
		t.Fatal("different run identity read the session's settlement receipt")
	}
	if _, err := b.SettlementReceipt(WithBrowserAgentCredentials(context.Background(), e.identityToken, "different-grant"), e.sessionID); err == nil {
		t.Fatal("different browser grant read the session's settlement receipt")
	}
}

func TestClosedSessionReceiptPreservesPendingAndNotApplicableStates(t *testing.T) {
	b, e, _ := closedEvidenceFixture(t)
	b.caller = bridgeCaller(e.callerID)
	e.identityToken = "signed-run"
	ctx := WithBrowserAgentCredentials(context.Background(), e.identityToken, e.grantReceipt)
	got, err := b.SettlementReceipt(ctx, e.sessionID)
	if err != nil || got.Status != "pending" || len(got.Receipt) != 0 {
		t.Fatalf("unconfirmed settlement must remain pending: %+v err=%v", got, err)
	}
	e.settlementResult = BrowserSettlementResult{Status: "not_applicable"}
	got, err = b.SettlementReceipt(ctx, e.sessionID)
	if err != nil || got.Status != "not_applicable" || len(got.Receipt) != 0 {
		t.Fatalf("no-reservation result must remain not_applicable: %+v err=%v", got, err)
	}
}

func TestClosedBrowserEvidenceProofExpiresAndRunBindingComesFromCaller(t *testing.T) {
	b, e, runID := closedEvidenceFixture(t)
	e.evidenceProof.ExpiresAt = time.Now().Add(-time.Second)
	e.evidenceProof.Digest = closedEvidenceDigest(e.evidenceProof)
	if _, err := b.VerifyBrowserEvidenceProof(context.Background(), runID, e.grantReceipt, e.evidenceProof.Reference); !errors.Is(err, ErrBrowserEvidenceProof) {
		t.Fatalf("expired proof accepted: %v", err)
	}
	e.callerID = "caller-supplied"
	if err := b.issueClosedEvidenceProof(e, time.Now()); err == nil {
		t.Fatal("proof issued for a caller that is not an AM run principal")
	}
}

func TestEvidenceProofRoutesFailClosedAndRejectCallerSummaryFields(t *testing.T) {
	b, e, runID := closedEvidenceFixture(t)
	request := func(handler *HTTPHandler, body string) *httptest.ResponseRecorder {
		router := chi.NewRouter()
		handler.RegisterRoutes(router)
		req := httptest.NewRequest(http.MethodPost, "/browser/internal/evidence-proof/verify", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer service-token")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	if got := request(&HTTPHandler{Broker: b}, `{"run_id":"`+runID.String()+`","grant_receipt":"`+e.grantReceipt+`","proof_reference":"`+e.evidenceProof.Reference+`"}`); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil verifier status = %d, want 503", got.Code)
	}
	principal := identity.Principal{
		Kind: identity.ActorService, Subject: BrowserEvidenceVerifierRole.Principal("agent-manager"),
		Audience: BrowserEvidenceVerifierAudience, Scopes: []string{BrowserEvidenceVerifierScope},
		Verified: true, Source: identity.SourceScenarioAuthenticator,
	}
	handler := &HTTPHandler{Broker: b, EvidenceTokenVerifier: browserEvidenceTestTokenVerifier{principal: principal}}
	body := `{"run_id":"` + runID.String() + `","grant_receipt":"` + e.grantReceipt + `","proof_reference":"` + e.evidenceProof.Reference + `"}`
	got := request(handler, body)
	if got.Code != http.StatusOK {
		t.Fatalf("valid callback status = %d body=%s", got.Code, got.Body.String())
	}
	var response struct {
		Observation BrowserEvidenceObservation `json:"observation"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Observation.SessionID != e.sessionID || response.Observation.Actions != 4 || response.Observation.Bytes != 8192 {
		t.Fatalf("response omitted snake-case owner summary fields: %+v", response.Observation)
	}
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	consumeBody := `{"run_id":"` + runID.String() + `","proof_reference":"` + e.evidenceProof.Reference + `"}`
	consumeRequest := httptest.NewRequest(http.MethodPost, "/browser/internal/evidence-proof/consume", strings.NewReader(consumeBody))
	consumeRequest.Header.Set("Authorization", "Bearer service-token")
	consumeResponse := httptest.NewRecorder()
	router.ServeHTTP(consumeResponse, consumeRequest)
	if consumeResponse.Code != http.StatusOK || !strings.Contains(consumeResponse.Body.String(), `"consumed":true`) {
		t.Fatalf("valid consume response = %d %s", consumeResponse.Code, consumeResponse.Body.String())
	}
	callerFields := `{"run_id":"` + runID.String() + `","grant_receipt":"` + e.grantReceipt + `","proof_reference":"` + e.evidenceProof.Reference + `","session_id":"forged","actions":999}`
	if got := request(handler, callerFields); got.Code != http.StatusBadRequest {
		t.Fatalf("caller-supplied session/counters status = %d, want 400", got.Code)
	}
	wrongRole := identity.Principal{Kind: identity.ActorService, Subject: "other-scenario-browser-evidence-verifier", Audience: BrowserEvidenceVerifierAudience, Scopes: []string{BrowserEvidenceVerifierScope}, Verified: true, Source: identity.SourceScenarioAuthenticator}
	if got := request(&HTTPHandler{Broker: b, EvidenceTokenVerifier: browserEvidenceTestTokenVerifier{principal: wrongRole}}, body); got.Code != http.StatusForbidden {
		t.Fatalf("wrong verifier subject status = %d, want 403", got.Code)
	}
	if got := request(handler, `{"run_id":"`+runID.String()+`","proof_reference":"`+e.evidenceProof.Reference+`"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("missing grant status = %d, want 400", got.Code)
	}
}

func TestTrustedEvidenceReporterSendsOnlyProofReferenceWithExistingServiceIdentity(t *testing.T) {
	runID := uuid.New()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/v1/browser/bas-evidence/"+runID.String() {
			t.Errorf("AM ingest path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer am-service-token" || r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser-grant:opaque" {
			t.Errorf("trusted headers missing or wrong: %v", r.Header)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload) != 1 || payload["proof_reference"] != "opaque-proof-ref" {
			t.Errorf("AM request must contain only proof reference: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"recorded":true,"run_id":"` + runID.String() + `","session_id":"bas-session-1","reported_by":"browser-automation-studio-broker","settlement":{"status":"settled","receipt":{"reservation_id":"hold-1","status":"settled","business_account_id":"acct-1","browser_session_minutes":2,"browser_actions":4,"browser_bytes":8192,"provider":"hosted","model":"m","cost_micros":12,"prompt_tokens":3,"completion_tokens":2,"credits_charged":1}}}`))
	}))
	defer server.Close()
	recorder := &AgentManagerEvidenceRecorder{
		RunContext:    &AgentManagerRunContext{BaseURL: server.URL, Client: server.Client()},
		ServiceTokens: browserEvidenceTestTokenSource("am-service-token"),
	}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-agent-token", "browser-grant:opaque")
	result, err := recorder.RecordClosedBrowserSession(ctx, "run:"+runID.String(), "bas-session-1", "browser-grant:opaque", "opaque-proof-ref")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "settled" || !strings.Contains(string(result.Receipt), `"reservation_id":"hold-1"`) {
		t.Fatalf("settlement result did not preserve LPBS receipt: %+v", result)
	}
	if requests != 1 {
		t.Fatalf("AM trusted ingests = %d, want 1", requests)
	}
	if _, err := recorder.RecordClosedBrowserSession(ctx, "run:"+runID.String(), "bas-session-1", "different-grant", "opaque-proof-ref"); err == nil {
		t.Fatal("grant not bound to trusted request context")
	}
	withoutCredential := &AgentManagerEvidenceRecorder{RunContext: recorder.RunContext}
	if _, err := withoutCredential.RecordClosedBrowserSession(ctx, "run:"+runID.String(), "bas-session-1", "browser-grant:opaque", "opaque-proof-ref"); !errors.Is(err, ErrTrustedEvidenceUnavailable) {
		t.Fatalf("missing operator-issued credential error = %v, want unavailable sentinel", err)
	}
}

func TestTrustedEvidenceReportRoundTripsThroughAuthenticatedBASProofCallbacks(t *testing.T) {
	b, e, runID := closedEvidenceFixture(t)
	principal := identity.Principal{
		Kind: identity.ActorService, Subject: BrowserEvidenceVerifierRole.Principal("agent-manager"),
		Audience: BrowserEvidenceVerifierAudience, Scopes: []string{BrowserEvidenceVerifierScope},
		Verified: true, Source: identity.SourceScenarioAuthenticator,
	}
	basHandler := &HTTPHandler{Broker: b, EvidenceTokenVerifier: browserEvidenceTestTokenVerifier{principal: principal}}
	basRouter := chi.NewRouter()
	basHandler.RegisterRoutes(basRouter)
	basServer := httptest.NewServer(basRouter)
	defer basServer.Close()

	amServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/bas-evidence/"+runID.String() || r.Header.Get("Authorization") != "Bearer am-service-token" {
			t.Errorf("unexpected BAS→AM trusted ingest request path/header: %s %v", r.URL.Path, r.Header)
		}
		if r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != e.grantReceipt {
			t.Errorf("trusted run context missing on AM ingest request: %v", r.Header)
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if len(input) != 1 || input["proof_reference"] != e.evidenceProof.Reference {
			t.Errorf("ingest request must contain only the broker proof ref: %#v", input)
		}
		callback := func(path string, body any) (*http.Response, error) {
			data, _ := json.Marshal(body)
			req, _ := http.NewRequest(http.MethodPost, basServer.URL+path, strings.NewReader(string(data)))
			req.Header.Set("Authorization", "Bearer am-verifier-token")
			req.Header.Set("Content-Type", "application/json")
			return basServer.Client().Do(req)
		}
		verify, err := callback("/browser/internal/evidence-proof/verify", map[string]string{"run_id": runID.String(), "grant_receipt": e.grantReceipt, "proof_reference": e.evidenceProof.Reference})
		if err != nil {
			t.Errorf("verify callback: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer verify.Body.Close()
		var verified struct {
			Observation BrowserEvidenceObservation `json:"observation"`
		}
		if err := json.NewDecoder(verify.Body).Decode(&verified); err != nil || verify.StatusCode != http.StatusOK || verified.Observation.SessionID != e.sessionID {
			t.Errorf("invalid proof verification result: status=%d body=%+v err=%v", verify.StatusCode, verified, err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		consumed, err := callback("/browser/internal/evidence-proof/consume", map[string]string{"run_id": runID.String(), "proof_reference": e.evidenceProof.Reference})
		if err != nil {
			t.Errorf("consume callback: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer consumed.Body.Close()
		if consumed.StatusCode != http.StatusOK {
			t.Errorf("consume callback status = %d", consumed.StatusCode)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"recorded":true,"run_id":"` + runID.String() + `","session_id":"` + e.sessionID + `","reported_by":"browser-automation-studio-broker","settlement":{"status":"settled","receipt":{"reservation_id":"hold-2","status":"settled","business_account_id":"acct-2","browser_session_minutes":2,"browser_actions":4,"browser_bytes":8192,"provider":"hosted","model":"m","cost_micros":12,"prompt_tokens":3,"completion_tokens":2,"credits_charged":1}}}`))
	}))
	defer amServer.Close()

	recorder := &AgentManagerEvidenceRecorder{
		RunContext:    &AgentManagerRunContext{BaseURL: amServer.URL, Client: amServer.Client()},
		ServiceTokens: browserEvidenceTestTokenSource("am-service-token"),
	}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-agent-token", e.grantReceipt)
	result, err := recorder.RecordClosedBrowserSession(ctx, e.callerID, e.sessionID, e.grantReceipt, e.evidenceProof.Reference)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "settled" || !strings.Contains(string(result.Receipt), `"reservation_id":"hold-2"`) {
		t.Fatalf("LPBS settlement receipt was not relayed: %+v", result)
	}
	if !e.evidenceProof.Consumed {
		t.Fatal("proof was not consumed after successful round trip")
	}
}
