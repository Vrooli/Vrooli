package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pairVerifierFixture struct {
	claims Claims
	pairs  int
}

func (v *pairVerifierFixture) Verify(context.Context, string) (Claims, error) {
	return Claims{}, errors.New("baseline-only verification must not be used")
}

func (v *pairVerifierFixture) VerifyPair(_ context.Context, baseline, operation string) (Claims, error) {
	v.pairs++
	if baseline != "run-token" || operation != "operation-token" {
		return Claims{}, ErrUnverifiable
	}
	return v.claims, nil
}

func TestVerifyRequestUsesCanonicalPairWhenOperationTokenPresent(t *testing.T) {
	want := Claims{RunID: "run-1", Subject: "agent:research", Scopes: []string{"treasury:write"}, Meta: map[string]string{"operation_grant": "verified"}}
	verifier := &pairVerifierFixture{claims: want}
	headers := make(http.Header)
	headers.Set(HeaderAgentIdentityToken, "run-token")
	headers.Set(HeaderAgentOperationToken, "operation-token")
	got, err := VerifyRequest(context.Background(), verifier, headers)
	if err != nil || got.RunID != want.RunID || got.Subject != want.Subject || got.Meta["operation_grant"] != "verified" || verifier.pairs != 1 {
		t.Fatalf("pair verification did not supply the verified operation claims: got=%+v calls=%d err=%v", got, verifier.pairs, err)
	}
}

func TestVerifyRequestNeverFallsBackWhenPairVerifierMissing(t *testing.T) {
	headers := make(http.Header)
	headers.Set(HeaderAgentIdentityToken, "run-token")
	headers.Set(HeaderAgentOperationToken, "operation-token")
	if _, err := VerifyRequest(context.Background(), singleTokenVerifier{}, headers); err == nil {
		t.Fatal("operation token was silently ignored by a baseline-only verifier")
	}
}

type singleTokenVerifier struct{}

func (singleTokenVerifier) Verify(context.Context, string) (Claims, error) { return Claims{}, nil }

func TestHTTPVerifierVerifyPairSendsBothTokensAndReturnsOperationClaims(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/identity/verify" {
			t.Errorf("unexpected verification request %s %s", r.Method, r.URL.Path)
		}
		var request struct {
			Token          string `json:"token"`
			OperationToken string `json:"operation_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Token != "run-token" || request.OperationToken != "operation-token" {
			t.Errorf("pair request omitted a token: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"operation":{"valid":true,"claims":{"run_id":"run-1","subject":"agent:research","scopes":["treasury:write"],"exp":2000000000,"meta":{"operation_grant":"verified"}}}}`))
	}))
	defer server.Close()
	verifier, err := NewHTTPVerifier(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := verifier.VerifyPair(context.Background(), "run-token", "operation-token")
	if err != nil || claims.RunID != "run-1" || claims.Meta["operation_grant"] != "verified" {
		t.Fatalf("VerifyPair returned wrong operation claims: %+v err=%v", claims, err)
	}
}
