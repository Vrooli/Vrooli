package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fixedBrowserEvidenceToken string

func (t fixedBrowserEvidenceToken) Token(context.Context) (string, error) { return string(t), nil }

type failingBrowserEvidenceToken struct{}

func (failingBrowserEvidenceToken) Token(context.Context) (string, error) {
	return "", errors.New("missing credential")
}

func TestAgentManagerBrowserEvidenceAttesterUsesBoundProofAndReservation(t *testing.T) {
	runID, reservationID := uuid.New(), uuid.New()
	proof := BrowserEvidenceProof{RunID: runID.String(), GrantReceipt: "exact-run-grant", Reference: "opaque-proof-ref", ReservationID: reservationID.String()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != agentManagerBrowserEvidencePath+runID.String() {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer attester-token" {
			t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 3 || body["grant_receipt"] != proof.GrantReceipt || body["proof_reference"] != proof.Reference || body["reservation_id"] != reservationID.String() {
			t.Errorf("unexpected attestation request fields: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(browserSettlementAttestationResponse{
			RunID: runID.String(), ReservationID: reservationID.String(),
			GrantReceiptFingerprint: fingerprint(proof.GrantReceipt), ProofFingerprint: fingerprint(proof.Reference),
			Observation: browserEvidenceObservation{Kind: "session", SessionID: "broker-session", State: "closed", SessionMinutes: 2, Actions: 3, Bytes: 2048},
		})
	}))
	defer server.Close()
	attester := &AgentManagerBrowserEvidenceAttester{
		tokens: fixedBrowserEvidenceToken("attester-token"), client: &http.Client{Timeout: time.Second},
		resolveURL: func(context.Context) (string, error) { return server.URL, nil },
	}
	got, err := attester.VerifyAndConsumeBrowserEvidence(context.Background(), proof)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != proof.RunID || got.ReservationID != proof.ReservationID || got.SessionID != "broker-session" || got.State != "closed" || got.SessionMinutes != 2 || got.Actions != 3 || got.Bytes != 2048 {
		t.Fatalf("attested browser usage=%+v", got)
	}
}

func TestAgentManagerBrowserEvidenceAttesterRejectsBindingAndMeasurementMismatch(t *testing.T) {
	runID, reservationID := uuid.New(), uuid.New()
	proof := BrowserEvidenceProof{RunID: runID.String(), GrantReceipt: "exact-run-grant", Reference: "opaque-proof-ref", ReservationID: reservationID.String()}
	cases := []struct {
		name   string
		mutate func(*browserSettlementAttestationResponse)
	}{
		{name: "cross run", mutate: func(a *browserSettlementAttestationResponse) { a.RunID = uuid.NewString() }},
		{name: "cross reservation", mutate: func(a *browserSettlementAttestationResponse) { a.ReservationID = uuid.NewString() }},
		{name: "cross grant", mutate: func(a *browserSettlementAttestationResponse) { a.GrantReceiptFingerprint = fingerprint("other-grant") }},
		{name: "cross proof", mutate: func(a *browserSettlementAttestationResponse) { a.ProofFingerprint = fingerprint("other-proof") }},
		{name: "open session", mutate: func(a *browserSettlementAttestationResponse) { a.Observation.State = "open" }},
		{name: "negative actions", mutate: func(a *browserSettlementAttestationResponse) { a.Observation.Actions = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attestation := browserSettlementAttestationResponse{
					RunID: runID.String(), ReservationID: reservationID.String(), GrantReceiptFingerprint: fingerprint(proof.GrantReceipt), ProofFingerprint: fingerprint(proof.Reference),
					Observation: browserEvidenceObservation{Kind: "session", SessionID: "broker-session", State: "closed", SessionMinutes: 2, Actions: 3, Bytes: 10},
				}
				tc.mutate(&attestation)
				_ = json.NewEncoder(w).Encode(attestation)
			}))
			defer server.Close()
			attester := &AgentManagerBrowserEvidenceAttester{tokens: fixedBrowserEvidenceToken("token"), client: server.Client(), resolveURL: func(context.Context) (string, error) { return server.URL, nil }}
			if _, err := attester.VerifyAndConsumeBrowserEvidence(context.Background(), proof); err == nil {
				t.Fatal("mismatched attestation was accepted")
			}
		})
	}
}

func TestAgentManagerBrowserEvidenceAttesterFailsClosedForMissingIdentityAndUnsafeOrigin(t *testing.T) {
	proof := BrowserEvidenceProof{RunID: uuid.NewString(), GrantReceipt: "grant", Reference: "proof", ReservationID: uuid.NewString()}
	cases := []struct {
		name   string
		tokens browserEvidenceServiceTokenSource
		url    string
	}{
		{name: "missing principal credential", tokens: failingBrowserEvidenceToken{}, url: "https://agent-manager.example.test"},
		{name: "nonloopback cleartext", tokens: fixedBrowserEvidenceToken("token"), url: "http://agent-manager.example.test"},
		{name: "credential endpoint path", tokens: fixedBrowserEvidenceToken("token"), url: "https://agent-manager.example.test/path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attester := &AgentManagerBrowserEvidenceAttester{tokens: tc.tokens, client: &http.Client{Timeout: time.Second}, resolveURL: func(context.Context) (string, error) { return tc.url, nil }}
			if _, err := attester.VerifyAndConsumeBrowserEvidence(context.Background(), proof); err == nil {
				t.Fatal("unsafe or unconfigured attester succeeded")
			}
		})
	}
}

func TestAgentManagerBrowserEvidenceAttesterRejectsRedirectAndNonSuccess(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusForbidden, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == http.StatusTemporaryRedirect {
					http.Redirect(w, r, "https://other.example.test/attest", status)
					return
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(strings.Repeat("x", 16)))
			}))
			defer server.Close()
			client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			proof := BrowserEvidenceProof{RunID: uuid.NewString(), GrantReceipt: "grant", Reference: "proof", ReservationID: uuid.NewString()}
			attester := &AgentManagerBrowserEvidenceAttester{tokens: fixedBrowserEvidenceToken("token"), client: client, resolveURL: func(context.Context) (string, error) { return server.URL, nil }}
			if _, err := attester.VerifyAndConsumeBrowserEvidence(context.Background(), proof); err == nil {
				t.Fatal("redirect or non-success response was accepted")
			}
		})
	}
}
