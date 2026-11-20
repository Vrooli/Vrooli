package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	eventstore "agent-manager/internal/adapters/event"
	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

type browserIdentityFake struct {
	orchestration.IdentityService
	result *orchestration.IdentityVerifyResult
}

func (f browserIdentityFake) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return f.result, nil
}

type browserRunFake struct {
	orchestration.RunService
	run *domain.Run
}

type browserEvidenceFake struct {
	orchestration.BrowserEvidenceService
	called  bool
	payload eventlog.Payload
}

type browserEventReaderFake struct {
	orchestration.EventService
	events []*domain.RunEvent
}

type browserTranscriptStoreFake struct {
	orchestration.EventService
	orchestration.BrowserEvidenceService
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (f *browserTranscriptStoreFake) GetRunEvents(context.Context, uuid.UUID, eventstore.GetOptions) ([]*domain.RunEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*domain.RunEvent(nil), f.events...), nil
}

func (f *browserTranscriptStoreFake) RecordBrowserEvidence(_ context.Context, runID uuid.UUID, payload eventlog.Payload) error {
	evt, err := eventlog.BuildEvent(runID, payload)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, evt)
	return nil
}

func (f browserEventReaderFake) GetRunEvents(context.Context, uuid.UUID, eventstore.GetOptions) ([]*domain.RunEvent, error) {
	return f.events, nil
}

func (f *browserEvidenceFake) RecordBrowserEvidence(_ context.Context, _ uuid.UUID, payload eventlog.Payload) error {
	f.called = true
	f.payload = payload
	return nil
}

func (f browserRunFake) GetRun(context.Context, uuid.UUID) (*domain.Run, error) { return f.run, nil }

func browserHandlerForTest(id uuid.UUID) *Handler {
	capability := &browsercapability.Capability{
		Enabled:      true,
		GrantReceipt: browserTestGrantReceipt(id),
		Candidates:   []browsercapability.Candidate{{Harness: "claude-code", ResourceRole: "browser.session"}},
		Mode:         browsercapability.ModeAttached,
		Permissions:  browsercapability.PermissionProfileSpec{ProfileID: "profile-browser-act", ProfileMode: browsercapability.SessionProfileUserAttached, Level: browsercapability.ProfileAct, AllowedOrigins: []string{"https://example.test"}, ExtensionIDs: []string{"extension-safe"}},
		Budget:       browsercapability.Budget{SessionMinutes: 10, Actions: 80, Bytes: 2_000_000, InferenceTokens: 15_000},
	}
	return &Handler{svc: orchestration.HandlerServices{
		IdentityService: browserIdentityFake{result: &orchestration.IdentityVerifyResult{
			Valid: true, Claims: &identity.Claims{RunID: id}, RunStatus: domain.RunStatusRunning,
		}},
		RunService: browserRunFake{run: &domain.Run{ID: id, Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode, Features: domain.FeatureFlags{BrowserCapability: capability},
		}}},
	}}
}

func browserTestGrantReceipt(id uuid.UUID) string { return "browser-grant:test-" + id.String() }

func withBrowserRunAgent(req *http.Request, runID uuid.UUID) *http.Request {
	principal := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: runID.String(), Verified: true, Scopes: []string{"agent-manager:write"}}
	return req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
}

type browserEvidenceAuthorizationRaiser struct {
	req    orchestration.AuthorizationApprovalRequest
	raised bool
	err    error
}

func (f *browserEvidenceAuthorizationRaiser) RaiseAuthorizationApproval(_ context.Context, req orchestration.AuthorizationApprovalRequest) (bool, error) {
	f.req = req
	return f.raised, f.err
}

func (f *browserEvidenceAuthorizationRaiser) IsBrowserEvidenceRun(context.Context, uuid.UUID) bool {
	return true
}

type browserEvidenceOperationAuthority struct{ err error }

func (a browserEvidenceOperationAuthority) AuthorizeOperation(_ context.Context, _ coreidentity.Principal, capability, operation, target string) error {
	if capability != "agent-manager:write" || operation != browserEvidenceAuthorizationOperation || !strings.HasPrefix(target, "agent-manager:run/") {
		return authn.ErrOperation
	}
	return a.err
}

type browserEvidenceProofFake struct {
	observation       BrowserEvidenceObservation
	err               error
	called            bool
	consumed          bool
	verifiedRun       uuid.UUID
	verifiedGrant     string
	verifiedReference string
}

type retryingBrowserEvidenceProofFake struct {
	browserEvidenceProofFake
	consumeCalls int
}

func (f *retryingBrowserEvidenceProofFake) ConsumeBrowserEvidenceProof(_ context.Context, _ uuid.UUID, proofReference string) error {
	if proofReference != "opaque-proof-ref" {
		return errors.New("unknown proof")
	}
	f.consumeCalls++
	// Model BAS having consumed the reference while its successful response was
	// lost. BAS's retry contract allows this exact proof/run to be consumed again.
	f.consumed = true
	if f.consumeCalls == 1 {
		return errors.New("simulated response loss after broker consumption")
	}
	return nil
}

func (f *browserEvidenceProofFake) VerifyBrowserEvidenceProof(_ context.Context, runID uuid.UUID, grantReceipt string, proofReference string) (BrowserEvidenceObservation, error) {
	f.called = true
	f.verifiedRun = runID
	f.verifiedGrant = grantReceipt
	f.verifiedReference = proofReference
	if proofReference != "opaque-proof-ref" {
		return BrowserEvidenceObservation{}, errors.New("unknown proof")
	}
	return f.observation, f.err
}

func (f *browserEvidenceProofFake) ConsumeBrowserEvidenceProof(_ context.Context, _ uuid.UUID, proofReference string) error {
	if proofReference != "opaque-proof-ref" {
		return errors.New("unknown proof")
	}
	f.consumed = true
	return nil
}

func withBASBrowserEvidenceService(req *http.Request, runID uuid.UUID, operationErr error) *http.Request {
	principal := coreidentity.Principal{Kind: coreidentity.ActorService, Subject: orchestration.BrowserEvidenceServiceSubject,
		Verified: true, Scopes: []string{"agent-manager:write"}}
	ctx := coreidentity.WithPrincipal(req.Context(), principal)
	ctx = authn.WithRequestOperationAuthority(ctx, browserEvidenceOperationAuthority{err: operationErr})
	req = req.WithContext(ctx)
	return mux.SetURLVars(req, map[string]string{"run_id": runID.String()})
}

func TestAuthorizeBrowserEvidenceRequestsOnlyExactRunGrantAndWritesNoEvidence(t *testing.T) {
	runID := uuid.New()
	server := browserHandlerForTest(runID)
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	server.svc.EventService = browserEventReaderFake{}
	raiser := &browserEvidenceAuthorizationRaiser{raised: true}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/evidence/authorize/"+runID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"run_id": runID.String()})
	req = req.WithContext(coreidentity.WithPrincipal(req.Context(), coreidentity.Principal{Kind: coreidentity.ActorService,
		Subject: orchestration.BrowserEvidenceServiceSubject, Verified: true, Scopes: []string{"agent-manager:write"}}))
	req = req.WithContext(authn.WithRequestOperationAuthority(req.Context(), browserEvidenceOperationAuthority{err: authn.ErrNeedsApproval}))
	req = req.WithContext(context.WithValue(req.Context(), approvalRaiserKey{}, raiser))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(runID))
	response := httptest.NewRecorder()
	server.AuthorizeBrowserEvidence(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("authorization request status=%d body=%s", response.Code, response.Body.String())
	}
	if raiser.req.Subject != orchestration.BrowserEvidenceServiceSubject || raiser.req.Operation != browserEvidenceAuthorizationOperation ||
		raiser.req.Target != "agent-manager:run/"+runID.String() || raiser.req.Kind != domain.PendingAuthorizationKindOneTime ||
		raiser.req.MaxConcurrentRuns != 0 || raiser.req.SessionID != "" {
		t.Fatalf("authorization asked for broader authority: %+v", raiser.req)
	}
	if evidence.called {
		t.Fatal("authorization request must not append browser evidence")
	}
}

func TestAuthorizeBrowserEvidenceRequiresBASPrincipalAndExactGrant(t *testing.T) {
	runID := uuid.New()
	call := func(subject string, authority error, grant string) *httptest.ResponseRecorder {
		server := browserHandlerForTest(runID)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/evidence/authorize/"+runID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{"run_id": runID.String()})
		req = req.WithContext(coreidentity.WithPrincipal(req.Context(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: subject,
			Verified: true, Scopes: []string{"agent-manager:write"}}))
		req = req.WithContext(authn.WithRequestOperationAuthority(req.Context(), browserEvidenceOperationAuthority{err: authority}))
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, grant)
		response := httptest.NewRecorder()
		server.AuthorizeBrowserEvidence(response, req)
		return response
	}
	if got := call("other-service", nil, browserTestGrantReceipt(runID)); got.Code != http.StatusForbidden {
		t.Fatalf("wrong service principal status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(orchestration.BrowserEvidenceServiceSubject, authn.ErrOperation, browserTestGrantReceipt(runID)); got.Code != http.StatusForbidden {
		t.Fatalf("missing operation grant status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(orchestration.BrowserEvidenceServiceSubject, nil, "wrong-grant"); got.Code != http.StatusForbidden {
		t.Fatalf("wrong browser grant status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(orchestration.BrowserEvidenceServiceSubject, nil, browserTestGrantReceipt(runID)); got.Code != http.StatusNoContent {
		t.Fatalf("granted operation status=%d body=%s", got.Code, got.Body.String())
	}
}

func TestAttestBrowserSettlementBindsExactRunGrantProofAndReservation(t *testing.T) {
	runID, reservationID := uuid.New(), uuid.New()
	server := browserHandlerForTest(runID)
	proof := &browserEvidenceProofFake{observation: BrowserEvidenceObservation{Kind: "session", SessionID: "broker-session", State: "closed", SessionMinutes: 2, Actions: 3, Bytes: 1024}}
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	server.browserEvidenceProof = proof
	body, _ := json.Marshal(browserSettlementAttestationRequest{GrantReceipt: browserTestGrantReceipt(runID), ProofReference: "opaque-proof-ref", ReservationID: reservationID.String()})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/browser/settlement-attestation/"+runID.String(), bytes.NewReader(body))
	request = mux.SetURLVars(request, map[string]string{"run_id": runID.String()})
	request = request.WithContext(coreidentity.WithPrincipal(request.Context(), coreidentity.Principal{Kind: coreidentity.ActorService,
		Subject: browserSettlementAttestationSubject, Verified: true, Scopes: []string{browserSettlementAttestationScope}}))
	response := httptest.NewRecorder()
	server.AttestBrowserSettlement(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("attestation status=%d body=%s", response.Code, response.Body.String())
	}
	if !proof.called || !proof.consumed || proof.verifiedRun != runID || proof.verifiedGrant != browserTestGrantReceipt(runID) || proof.verifiedReference != "opaque-proof-ref" {
		t.Fatalf("proof verification/consume binding invalid: %+v", proof)
	}
	if evidence.called {
		t.Fatal("settlement attestation must not append an AM transcript event")
	}
	var got browserSettlementAttestationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != runID.String() || got.ReservationID != reservationID.String() || got.ProofFingerprint != browserEvidenceValueFingerprint("opaque-proof-ref") || got.GrantReceiptFingerprint != browserEvidenceValueFingerprint(browserTestGrantReceipt(runID)) || got.Observation.SessionID != "broker-session" || got.Observation.Actions != 3 {
		t.Fatalf("attestation did not bind the broker observation to the LPBS request: %+v", got)
	}
}

func TestAttestBrowserSettlementRejectsUntrustedRequestsBeforeConsume(t *testing.T) {
	runID, reservationID := uuid.New(), uuid.New()
	cases := []struct {
		name      string
		subject   string
		scopes    []string
		grant     string
		proofBody string
		observe   BrowserEvidenceObservation
		proofErr  error
		want      int
	}{
		{name: "wrong service", subject: "other-service", scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed"}, want: http.StatusForbidden},
		{name: "wrong scope", subject: browserSettlementAttestationSubject, scopes: []string{"agent-manager:write"}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed"}, want: http.StatusForbidden},
		{name: "wrong grant", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: "other-run-grant", proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed"}, want: http.StatusForbidden},
		{name: "caller measurement rejected", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref","actions":999`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed"}, want: http.StatusBadRequest},
		{name: "open session not consumed", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "open"}, want: http.StatusForbidden},
		{name: "incomplete observation not consumed", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", State: "closed"}, want: http.StatusForbidden},
		{name: "negative measurement not consumed", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed", Bytes: -1}, want: http.StatusForbidden},
		{name: "over budget not consumed", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, observe: BrowserEvidenceObservation{Kind: "session", SessionID: "s", State: "closed", Actions: 81}, want: http.StatusForbidden},
		{name: "proof from different run", subject: browserSettlementAttestationSubject, scopes: []string{browserSettlementAttestationScope}, grant: browserTestGrantReceipt(runID), proofBody: `"proof_reference":"opaque-proof-ref"`, proofErr: errors.New("run binding mismatch"), want: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := browserHandlerForTest(runID)
			proof := &browserEvidenceProofFake{observation: tc.observe, err: tc.proofErr}
			server.browserEvidenceProof = proof
			body := `{"grant_receipt":"` + tc.grant + `","reservation_id":"` + reservationID.String() + `",` + tc.proofBody + `}`
			request := httptest.NewRequest(http.MethodPost, "/api/v1/browser/settlement-attestation/"+runID.String(), strings.NewReader(body))
			request = mux.SetURLVars(request, map[string]string{"run_id": runID.String()})
			request = request.WithContext(coreidentity.WithPrincipal(request.Context(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: tc.subject, Verified: true, Scopes: tc.scopes}))
			response := httptest.NewRecorder()
			server.AttestBrowserSettlement(response, request)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body.String())
			}
			if proof.consumed {
				t.Fatal("refused request consumed a BAS proof")
			}
			if (tc.subject != browserSettlementAttestationSubject || tc.scopes[0] != browserSettlementAttestationScope || tc.grant != browserTestGrantReceipt(runID) || tc.name == "caller measurement rejected") && proof.called {
				t.Fatal("invalid caller reached BAS proof verification")
			}
		})
	}
}

func TestRecordBASBrowserEvidenceRequiresExactGrantAndBrokerProof(t *testing.T) {
	runID := uuid.New()
	server := browserHandlerForTest(runID)
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	server.svc.EventService = browserEventReaderFake{events: []*domain.RunEvent{
		eventlog.MustBuildEvent(runID, eventlog.BrowserSessionPayload{SessionID: "broker-session-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "opened", ReportedBy: "run-agent"}),
		eventlog.MustBuildEvent(runID, eventlog.BrowserActionPayload{SessionID: "broker-session-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", Action: "click", Outcome: "completed", ReportedBy: "run-agent"}),
	}}
	call := func(operationErr error, proof BrowserEvidenceProofVerifier) *httptest.ResponseRecorder {
		server.browserEvidenceProof = proof
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/bas-evidence/"+runID.String(), strings.NewReader(`{"proof_reference":"opaque-proof-ref"}`))
		req = withBASBrowserEvidenceService(req, runID, operationErr)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(runID))
		response := httptest.NewRecorder()
		server.RecordBASBrowserEvidence(response, req)
		return response
	}
	if got := call(authn.ErrNeedsApproval, &browserEvidenceProofFake{}); got.Code != http.StatusForbidden || evidence.called {
		t.Fatalf("missing operation grant status=%d body=%s append=%v", got.Code, got.Body.String(), evidence.called)
	}
	if got := call(nil, nil); got.Code != http.StatusServiceUnavailable || evidence.called {
		t.Fatalf("unconfigured broker verifier status=%d body=%s append=%v", got.Code, got.Body.String(), evidence.called)
	}
	if got := call(nil, &browserEvidenceProofFake{err: errors.New("revoked proof")}); got.Code != http.StatusForbidden || evidence.called {
		t.Fatalf("revoked broker proof status=%d body=%s append=%v", got.Code, got.Body.String(), evidence.called)
	}
	proof := &browserEvidenceProofFake{observation: BrowserEvidenceObservation{Kind: "session", SessionID: "broker-session-1", State: "closed", SessionMinutes: 2, Actions: 2, Bytes: 128}}
	if got := call(nil, proof); got.Code != http.StatusAccepted || !evidence.called || !proof.called {
		t.Fatalf("valid grant and broker observation status=%d body=%s append=%v proof=%v", got.Code, got.Body.String(), evidence.called, proof.called)
	}
	payload, ok := evidence.payload.(eventlog.BrowserSessionPayload)
	if !ok || payload.ReportedBy != "browser-automation-studio-broker" || payload.SessionID != "broker-session-1" || payload.State != "closed" || payload.Actions != 2 || !proof.consumed {
		t.Fatalf("trusted payload did not come from broker observation: %#v", evidence.payload)
	}
}

func TestRecordBASBrowserEvidenceRetryAfterAmbiguousConsumeIsIdempotent(t *testing.T) {
	runID := uuid.New()
	server := browserHandlerForTest(runID)
	transcript := &browserTranscriptStoreFake{}
	server.svc.BrowserEvidenceService = transcript
	server.svc.EventService = transcript
	proof := &retryingBrowserEvidenceProofFake{browserEvidenceProofFake: browserEvidenceProofFake{
		observation: BrowserEvidenceObservation{Kind: "session", SessionID: "broker-session-2", State: "closed", SessionMinutes: 1, Actions: 1, Bytes: 64},
	}}
	call := func() *httptest.ResponseRecorder {
		server.browserEvidenceProof = proof
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/bas-evidence/"+runID.String(), strings.NewReader(`{"proof_reference":"opaque-proof-ref"}`))
		req = withBASBrowserEvidenceService(req, runID, nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(runID))
		response := httptest.NewRecorder()
		server.RecordBASBrowserEvidence(response, req)
		return response
	}
	if got := call(); got.Code != http.StatusServiceUnavailable || !strings.Contains(got.Body.String(), `"recorded":true`) {
		t.Fatalf("first response-loss request status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(); got.Code != http.StatusAccepted || !strings.Contains(got.Body.String(), `"idempotent":true`) {
		t.Fatalf("retry status=%d body=%s", got.Code, got.Body.String())
	}
	if len(transcript.events) != 1 || proof.consumeCalls != 2 {
		t.Fatalf("retry duplicated evidence or missed idempotent consume: events=%d consumes=%d", len(transcript.events), proof.consumeCalls)
	}
}

func TestRunAgentBrowserEvidenceRemainsExplicitlyUntrusted(t *testing.T) {
	runID := uuid.New()
	server := browserHandlerForTest(runID)
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	server.svc.EventService = browserEventReaderFake{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/record-evidence", strings.NewReader(`{"kind":"session","session_id":"agent-session","mode":"user-attached","permission_profile":"profile-browser-act","state":"opened"}`))
	req = withBrowserRunAgent(req, runID)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(runID))
	response := httptest.NewRecorder()
	server.RecordBrowserEvidence(response, req)
	payload, ok := evidence.payload.(eventlog.BrowserSessionPayload)
	if response.Code != http.StatusAccepted || !evidence.called || !ok || payload.ReportedBy != "run-agent" {
		t.Fatalf("run-agent self-report status=%d payload=%#v called=%v", response.Code, evidence.payload, evidence.called)
	}
}

func TestAuthorizeBrowserOpenUsesRunTokenAndPersistedCapability(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	call := func(body, token, grant string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/authorize-open", strings.NewReader(body))
		if token != "" {
			req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		}
		if grant != "" {
			req.Header.Set(browserGrantReceiptHeader, grant)
		}
		response := httptest.NewRecorder()
		server.AuthorizeBrowserOpen(response, req)
		return response
	}

	response := call(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://example.test/login"}`, "signed-run-token", browserTestGrantReceipt(id))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"authorized":true`) || !strings.Contains(response.Body.String(), `"actions":80`) {
		t.Fatalf("authorized open response = %d %s", response.Code, response.Body.String())
	}
	if got := call(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://outside.test/"}`, "signed-run-token", browserTestGrantReceipt(id)); got.Code != http.StatusForbidden {
		t.Fatalf("outside-origin response = %d %s", got.Code, got.Body.String())
	}
	if got := call(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://example.test/"}`, "signed-run-token", browserTestGrantReceipt(uuid.New())); got.Code != http.StatusForbidden {
		t.Fatalf("foreign-grant response = %d %s", got.Code, got.Body.String())
	}
	if got := call(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://example.test/","resource_role":"browser.raw-cdp"}`, "signed-run-token", browserTestGrantReceipt(id)); got.Code != http.StatusForbidden {
		t.Fatalf("ineligible resource role response = %d %s", got.Code, got.Body.String())
	}
	if got := call(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://example.test/","run_id":"`+id.String()+`"}`, "signed-run-token", browserTestGrantReceipt(id)); got.Code != http.StatusBadRequest {
		t.Fatalf("caller-supplied run identity response = %d %s", got.Code, got.Body.String())
	}
}

func TestAuthorizeBrowserOpenRequiresVerifiedRunIdentity(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/authorize-open", strings.NewReader(`{"mode":"user-attached","profile_id":"profile-browser-act","initial_url":"https://example.test/"}`))
	response := httptest.NewRecorder()
	server.AuthorizeBrowserOpen(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing run identity response = %d %s", response.Code, response.Body.String())
	}
}

func TestResolveBrowserCallerDerivesRunPrincipalFromVerifiedIdentity(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-caller", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserCaller(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"caller_id":"run:`+id.String()+`"`) || !strings.Contains(response.Body.String(), `"profile_id":"profile-browser-act"`) {
		t.Fatalf("trusted caller response = %d %s", response.Code, response.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-caller", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(uuid.New()))
	response = httptest.NewRecorder()
	server.ResolveBrowserCaller(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("foreign grant response = %d %s", response.Code, response.Body.String())
	}
	server.svc.IdentityService = browserIdentityFake{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: id}, RunStatus: domain.RunStatusParked}}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-caller", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response = httptest.NewRecorder()
	server.ResolveBrowserCaller(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("yielded agent run retained browser action authority: %d %s", response.Code, response.Body.String())
	}
}

func TestResolveBrowserContextProvidesOnlyStoredRunGrant(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-context", strings.NewReader(`{"caller_id":"forged","grant_receipt":"forged"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	response := httptest.NewRecorder()
	server.ResolveBrowserContext(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("caller fields should be rejected: %d %s", response.Code, response.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-context", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	response = httptest.NewRecorder()
	server.ResolveBrowserContext(response, req)
	grant := browserTestGrantReceipt(id)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"grant_receipt":"`+grant+`"`) || !strings.Contains(response.Body.String(), `"caller_id":"run:`+id.String()+`"`) {
		t.Fatalf("trusted context response = %d %s", response.Code, response.Body.String())
	}
}

func TestRecordBrowserEvidenceRequiresTrustedRunAndRejectsContentFields(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	server.svc.EventService = browserEventReaderFake{events: []*domain.RunEvent{eventlog.MustBuildEvent(id, eventlog.BrowserSessionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "opened", ReportedBy: "run-agent"})}}
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/record-evidence", strings.NewReader(body))
		req = withBrowserRunAgent(req, id)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
		response := httptest.NewRecorder()
		server.RecordBrowserEvidence(response, req)
		return response
	}
	got := call(`{"kind":"action","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","action":"click","outcome":"completed"}`)
	if got.Code != http.StatusAccepted || !evidence.called {
		t.Fatalf("evidence response = %d %s, called=%v", got.Code, got.Body.String(), evidence.called)
	}
	if payload, ok := evidence.payload.(eventlog.BrowserActionPayload); !ok || payload.ReportedBy != "run-agent" {
		t.Fatalf("unverified reporter was mislabeled: %#v", evidence.payload)
	}
	evidence.called = false
	got = call(`{"kind":"action","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","action":"click","outcome":"completed","page_text":"secret"}`)
	if got.Code != http.StatusBadRequest || evidence.called {
		t.Fatalf("content field response = %d %s, called=%v", got.Code, got.Body.String(), evidence.called)
	}
	got = call(`{"kind":"session","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","state":"closed","session_minutes":11,"actions":1,"bytes":1}`)
	if got.Code != http.StatusBadRequest || evidence.called {
		t.Fatalf("over-budget counters response = %d %s, called=%v", got.Code, got.Body.String(), evidence.called)
	}
	server.svc.EventService = browserEventReaderFake{}
	got = call(`{"kind":"action","session_id":"missing","mode":"user-attached","permission_profile":"profile-browser-act","action":"click","outcome":"completed"}`)
	if got.Code != http.StatusConflict || evidence.called {
		t.Fatalf("action without open session response = %d %s, called=%v", got.Code, got.Body.String(), evidence.called)
	}
}

func TestRecordBrowserEvidenceRejectsDuplicateOpenAndIdempotentlyAcceptsSameClose(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	open := eventlog.MustBuildEvent(id, eventlog.BrowserSessionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "opened", ReportedBy: "run-agent"})
	close := eventlog.MustBuildEvent(id, eventlog.BrowserSessionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "closed", SessionMinutes: 2, Actions: 3, Bytes: 50, ReportedBy: "run-agent"})
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/record-evidence", strings.NewReader(body))
		req = withBrowserRunAgent(req, id)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
		rr := httptest.NewRecorder()
		server.RecordBrowserEvidence(rr, req)
		return rr
	}
	server.svc.EventService = browserEventReaderFake{events: []*domain.RunEvent{open}}
	if got := call(`{"kind":"session","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","state":"opened"}`); got.Code != http.StatusConflict || evidence.called {
		t.Fatalf("duplicate open = %d %s called=%v", got.Code, got.Body.String(), evidence.called)
	}
	server.svc.EventService = browserEventReaderFake{events: []*domain.RunEvent{open, close}}
	if got := call(`{"kind":"session","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","state":"closed","session_minutes":2,"actions":3,"bytes":50}`); got.Code != http.StatusAccepted || !strings.Contains(got.Body.String(), `"idempotent":true`) || evidence.called {
		t.Fatalf("same close replay = %d %s called=%v", got.Code, got.Body.String(), evidence.called)
	}
}

func TestRecordBrowserEvidenceCloseActionCountCannotUnderreportRecordedActions(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	open := eventlog.MustBuildEvent(id, eventlog.BrowserSessionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "opened", ReportedBy: "run-agent"})
	action := eventlog.MustBuildEvent(id, eventlog.BrowserActionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", Action: "click", Outcome: "completed", ReportedBy: "run-agent"})
	server.svc.EventService = browserEventReaderFake{events: []*domain.RunEvent{open, action}}
	evidence := &browserEvidenceFake{}
	server.svc.BrowserEvidenceService = evidence
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/record-evidence", strings.NewReader(`{"kind":"session","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","state":"closed","actions":0}`))
	req = withBrowserRunAgent(req, id)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.RecordBrowserEvidence(response, req)
	if response.Code != http.StatusConflict || evidence.called {
		t.Fatalf("underreported close = %d %s called=%v", response.Code, response.Body.String(), evidence.called)
	}
}

func TestRecordBrowserEvidenceSerializesActionBudgetCheckAndAppend(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	run := server.svc.RunService.(browserRunFake).run
	run.ResolvedConfig.Features.BrowserCapability.Budget.Actions = 1
	store := &browserTranscriptStoreFake{events: []*domain.RunEvent{eventlog.MustBuildEvent(id, eventlog.BrowserSessionPayload{SessionID: "s-1", Mode: "user-attached", PermissionProfile: "profile-browser-act", State: "opened", ReportedBy: "run-agent"})}}
	server.svc.EventService = store
	server.svc.BrowserEvidenceService = store
	call := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/record-evidence", strings.NewReader(`{"kind":"action","session_id":"s-1","mode":"user-attached","permission_profile":"profile-browser-act","action":"click","outcome":"completed"}`))
		req = withBrowserRunAgent(req, id)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
		req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
		rr := httptest.NewRecorder()
		server.RecordBrowserEvidence(rr, req)
		return rr.Code
	}
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); codes <- call() }()
	}
	wg.Wait()
	close(codes)
	accepted, conflict := 0, 0
	for code := range codes {
		if code == http.StatusAccepted {
			accepted++
		}
		if code == http.StatusConflict {
			conflict++
		}
	}
	if accepted != 1 || conflict != 1 {
		t.Fatalf("concurrent action budget results: accepted=%d conflict=%d", accepted, conflict)
	}
}

func TestResolveBrowserProfileReturnsCurrentStoredSpecAndUnsupportedFields(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	server.svc.RunService.(browserRunFake).run.ResolvedConfig.Features.BrowserCapability.Permissions.View = true
	server.svc.RunService.(browserRunFake).run.ResolvedConfig.Features.BrowserCapability.Permissions.Control = true
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"allowedOrigins":["https://example.test"]`) || !strings.Contains(response.Body.String(), `"profileMode":"user_attached"`) || !strings.Contains(response.Body.String(), `"view":true`) || !strings.Contains(response.Body.String(), `"control":true`) || !strings.Contains(response.Body.String(), `"extension_access":true`) || !strings.Contains(response.Body.String(), `"extension_permissions":{"extension-safe":[]}`) || strings.Contains(response.Body.String(), `"View"`) {
		t.Fatalf("profile response = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"profile_mode":"user_attached"`) || strings.Contains(response.Body.String(), `"ProfileMode"`) {
		t.Fatalf("attached mode mapping = %s", response.Body.String())
	}
}

func TestResolveBrowserProfileReturnsExplicitManagedExtensionInstallAndLoadOptIns(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	capability := server.svc.RunService.(browserRunFake).run.ResolvedConfig.Features.BrowserCapability
	capability.Mode = browsercapability.ModeManaged
	capability.Permissions.ProfileMode = browsercapability.SessionProfileFresh
	capability.Permissions.ExtensionInstall = true
	capability.Permissions.ExtensionLoad = true
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"extensionInstall":true`) || !strings.Contains(body, `"extensionLoad":true`) || strings.Contains(body, `"ExtensionInstall"`) || strings.Contains(body, `"ExtensionLoad"`) {
		t.Fatalf("managed extension opt-ins were not explicit and supported: %d %s", response.Code, body)
	}
}

func TestResolveBrowserProfileRejectsRunReaderReturningDifferentRun(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	server.svc.RunService = browserRunFake{run: &domain.Run{
		ID: uuid.New(), Status: domain.RunStatusRunning,
		ResolvedConfig: server.svc.RunService.(browserRunFake).run.ResolvedConfig,
	}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "browser_capability_not_enabled") {
		t.Fatalf("wrong-run browser capability response = %d %s", response.Code, response.Body.String())
	}
}

func TestResolveBrowserProfileRejectsForgedExtensionOptInInRequest(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act","extensionInstall":true,"extensionLoad":true}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("caller-forged extension opt-in response = %d %s", response.Code, response.Body.String())
	}
}

func TestSessionPermissionProfileModeIsExactOrUnsupported(t *testing.T) {
	for _, tc := range []struct {
		mode browsercapability.SessionProfileMode
		want string
		ok   bool
	}{
		{browsercapability.SessionProfileFresh, "fresh", true},
		{browsercapability.SessionProfileIsolated, "isolated", true},
		{browsercapability.SessionProfileUserAttached, "user_attached", true},
		{browsercapability.SessionProfileRemote, "remote", true},
		{"", "", false}, {"managed", "", false},
	} {
		got, ok := sessionPermissionProfileMode(tc.mode)
		if got != tc.want || ok != tc.ok {
			t.Errorf("mode %q = %q,%v; want %q,%v", tc.mode, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveBrowserProfileDefaultsViewAndControlToDenied(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"view":false`) || !strings.Contains(response.Body.String(), `"control":false`) {
		t.Fatalf("default permission response = %d %s", response.Code, response.Body.String())
	}
}

func TestResolveBrowserProfileKeepsMissingPersistedModeUnsupported(t *testing.T) {
	id := uuid.New()
	server := browserHandlerForTest(id)
	capability := server.svc.RunService.(browserRunFake).run.ResolvedConfig.Features.BrowserCapability
	capability.Mode = browsercapability.ModeManaged
	capability.Permissions.ProfileMode = ""
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/resolve-profile", strings.NewReader(`{"profile_ref":"profile-browser-act"}`))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "signed-run-token")
	req.Header.Set(browserGrantReceiptHeader, browserTestGrantReceipt(id))
	response := httptest.NewRecorder()
	server.ResolveBrowserProfile(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"profile_mode":""`) || !strings.Contains(response.Body.String(), `"ProfileMode"`) {
		t.Fatalf("profile mode must remain default-deny when absent from persisted run: %d %s", response.Code, response.Body.String())
	}
}
