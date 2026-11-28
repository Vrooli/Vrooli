package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/servicetoken"
)

func TestDiscoveredBrowserEvidenceProofVerifierContract(t *testing.T) {
	runID := uuid.New()
	var verifyBody, consumeBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer am-service-token" {
			t.Errorf("authorization = %q", got)
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "" {
			t.Error("forwarded run/browser identity to BAS")
		}
		switch r.URL.Path {
		case "/api/v1/browser/internal/evidence-proof/verify":
			if r.Method != http.MethodPost {
				t.Errorf("verify method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			verifyBody = string(body)
			_, _ = fmt.Fprint(w, `{"observation":{"kind":"session","session_id":"broker-session","state":"closed","session_minutes":3,"actions":4,"bytes":512}}`)
		case "/api/v1/browser/internal/evidence-proof/consume":
			if r.Method != http.MethodPost {
				t.Errorf("consume method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			consumeBody = string(body)
			_, _ = fmt.Fprint(w, `{"consumed":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	tokens := servicetoken.New(servicetoken.Config{
		Credential: func(context.Context) (string, error) { return "fixture-credential", nil },
		Expect:     servicetoken.ExpectIdentity(browserEvidenceVerifierPrincipal, browserEvidenceVerifierAudience, browserEvidenceVerifierScope),
	}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "am-service-token", ExpiresAt: time.Now().Add(time.Hour), Subject: browserEvidenceVerifierPrincipal, Audience: browserEvidenceVerifierAudience, Scopes: []string{browserEvidenceVerifierScope}}, nil
	}))
	client := &DiscoveredBrowserEvidenceProofVerifier{Tokens: tokens, Client: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	observation, err := client.VerifyBrowserEvidenceProof(context.Background(), runID, "opaque-grant", "opaque-proof")
	if err != nil {
		t.Fatal(err)
	}
	if observation.SessionID != "broker-session" || observation.SessionMinutes != 3 || observation.Actions != 4 || observation.Bytes != 512 {
		t.Fatalf("unexpected observation: %+v", observation)
	}
	if verifyBody != fmt.Sprintf(`{"grant_receipt":"opaque-grant","proof_reference":"opaque-proof","run_id":%q}`, runID.String()) {
		t.Fatalf("verify request = %s", verifyBody)
	}
	if err := client.ConsumeBrowserEvidenceProof(context.Background(), runID, "opaque-proof"); err != nil {
		t.Fatal(err)
	}
	if consumeBody != fmt.Sprintf(`{"proof_reference":"opaque-proof","run_id":%q}`, runID.String()) {
		t.Fatalf("consume request = %s", consumeBody)
	}
}

func TestDiscoveredBrowserEvidenceProofVerifierRejectsInvalidObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"observation":{"kind":"session","session_id":"s","state":"open","session_minutes":0,"actions":0,"bytes":0}}`)
	}))
	defer server.Close()
	tokens := servicetoken.New(servicetoken.Config{Credential: func(context.Context) (string, error) { return "fixture", nil }}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := &DiscoveredBrowserEvidenceProofVerifier{Tokens: tokens, Client: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	if _, err := client.VerifyBrowserEvidenceProof(context.Background(), uuid.New(), "grant", "proof"); err == nil {
		t.Fatal("expected open observation rejection")
	}
}
