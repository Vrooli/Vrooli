package aidoor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/consumeridentity"
)

func TestAppAudienceRequiresCanonicalAppScope(t *testing.T) {
	for _, app := range []string{"", "  ", "web console", "web\\console", "web\nconsole"} {
		if _, err := appAudience(app); err == nil {
			t.Errorf("appAudience(%q) accepted invalid scope", app)
		}
	}
	got, err := appAudience(" Audio-Tools ")
	if err != nil || got != consumeridentity.CreditGatewayAudience {
		t.Fatalf("appAudience() = %q, %v", got, err)
	}
}

func TestGatewayAssertionTravelsOnlyInAuthorizationHeader(t *testing.T) {
	const synthetic = "synthetic-credit-assertion"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Header.Get("Authorization") != "Bearer "+synthetic {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if strings.Contains(string(body), synthetic) {
			t.Error("assertion appeared in request body")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	door := &Door{GatewayURL: server.URL}
	response, err := door.postWithAssertion(context.Background(), "/private", map[string]string{"operation_id": "op-1"}, synthetic)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("response status = %d", response.StatusCode)
	}
}

func TestSessionScopeAudienceMustMatchPersistedApp(t *testing.T) {
	good, err := appAudience("audio-tools")
	if err != nil {
		t.Fatal(err)
	}
	parent := Key{Kind: "device", Account: "acct", Actor: "actor", App: "audio-tools", Audience: good, MaxCredits: 10}
	if !validCreditKeyScope(parent) {
		t.Fatal("app-bound audience rejected")
	}
	parent.Audience = "scenario:web-console"
	if validCreditKeyScope(parent) {
		t.Fatal("cross-app audience unexpectedly matched")
	}
}

func TestDoorRejectsFeatureOutsideGeneratedOwnerBeforeSigningOrFunding(t *testing.T) {
	calls := 0
	door := &Door{SignCreditAssertion: func(Key, string, string) (string, error) {
		calls++
		return "unexpected", nil
	}}
	key := Key{Kind: "device", Account: "acct", Actor: "actor", App: "audio-tools", Audience: consumeridentity.CreditGatewayAudience, MaxCredits: 20}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture"}`))
	door.funded(response, request, key, "agent.session", "fixture", nil, nil)
	if response.Code != http.StatusForbidden || calls != 0 {
		t.Fatalf("cross-owner feature status=%d signer_calls=%d body=%s", response.Code, calls, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "credit assertion") {
		t.Fatal("cross-owner refusal exposed signed material")
	}
}
