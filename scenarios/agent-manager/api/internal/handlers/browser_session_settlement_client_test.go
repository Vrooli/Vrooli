package handlers

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/servicetoken"
)

func TestDiscoveredBrowserSessionSettlementClientUsesScopedIdentityAndPreservesReceipt(t *testing.T) {
	runID := uuid.New()
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/internal/browser-sessions/settle-and-receipt" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer am-lpbs-token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "" {
			t.Error("forwarded the inbound run identity or browser grant header")
		}
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		_, _ = fmt.Fprint(w, `{"reservation_id":"reservation-1","status":"settled","business_account_id":"acct-1","provider":"multiple","model":"multiple","cost_micros":900,"prompt_tokens":10,"completion_tokens":5,"credits_charged":7,"browser_session_minutes":3,"browser_actions":4,"browser_bytes":512,"inference_operations":[{"root_operation_id":"root-1","provider":"hosted-provider","model":"hosted-model","credits_charged":7,"cost_micros":900,"prompt_tokens":10,"completion_tokens":5}]}`)
	}))
	defer server.Close()
	tokens := servicetoken.New(servicetoken.Config{
		Credential: func(context.Context) (string, error) { return "fixture-credential", nil },
		Expect:     servicetoken.ExpectIdentity(browserSessionSettlementPrincipal, browserSessionSettlementAudience, browserSessionSettlementScope),
	}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "am-lpbs-token", ExpiresAt: time.Now().Add(time.Hour), Subject: browserSessionSettlementPrincipal, Audience: browserSessionSettlementAudience, Scopes: []string{browserSessionSettlementScope}}, nil
	}))
	client := &DiscoveredBrowserSessionSettlementClient{Tokens: tokens, Client: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	receipt, err := client.SettleBrowserSession(context.Background(), runID, "broker-session", "opaque-grant", "opaque-proof")
	if err != nil {
		t.Fatal(err)
	}
	wantBody := fmt.Sprintf(`{"grant_receipt":"opaque-grant","proof_reference":"opaque-proof","run_id":%q,"session_id":"broker-session"}`, runID.String())
	if body != wantBody {
		t.Fatalf("request body = %s, want %s", body, wantBody)
	}
	if receipt.Status != "settled" || receipt.ReservationID != "reservation-1" || receipt.SessionMinutes != 3 || receipt.Actions != 4 || receipt.Bytes != 512 || len(receipt.InferenceOperations) != 1 || receipt.InferenceOperations[0].Provider != "hosted-provider" || receipt.InferenceOperations[0].PromptTokens != 10 {
		t.Fatalf("LPBS receipt = %+v", receipt)
	}
}

func TestDiscoveredBrowserSessionSettlementClientRejectsPendingReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"reservation_id":"reservation-1","status":"pending"}`)
	}))
	defer server.Close()
	tokens := servicetoken.New(servicetoken.Config{Credential: func(context.Context) (string, error) { return "fixture", nil }}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := &DiscoveredBrowserSessionSettlementClient{Tokens: tokens, Client: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	if _, err := client.SettleBrowserSession(context.Background(), uuid.New(), "session", "grant", "proof"); err == nil {
		t.Fatal("pending receipt must not be represented as settled")
	}
}

func TestValidateSettledBrowserSessionReceiptBindsAggregateToDistinctOperations(t *testing.T) {
	receipt := BrowserSessionReceipt{
		ReservationID: "reservation-1", Status: "settled", BusinessAccountID: "account-1",
		Provider: "multiple", Model: "multiple", CreditsCharged: 7, CostMicros: 900,
		PromptTokens: 10, CompletionTokens: 5, SessionMinutes: 3, Actions: 4, Bytes: 512,
		InferenceOperations: []BrowserInferenceReceipt{{
			RootOperationID: "root-1", Provider: "provider-a", Model: "model-a",
			CreditsCharged: 3, CostMicros: 400, PromptTokens: 6, CompletionTokens: 2,
		}, {
			RootOperationID: "root-2", Provider: "provider-b", Model: "model-b",
			CreditsCharged: 4, CostMicros: 500, PromptTokens: 4, CompletionTokens: 3,
		}},
	}
	if err := validateSettledBrowserSessionReceipt(receipt); err != nil {
		t.Fatalf("valid multi-provider LPBS receipt refused: %v", err)
	}

	for name, mutate := range map[string]func(*BrowserSessionReceipt){
		"aggregate-cost-drift":  func(value *BrowserSessionReceipt) { value.CostMicros++ },
		"aggregate-token-drift": func(value *BrowserSessionReceipt) { value.PromptTokens++ },
		"duplicate-operation": func(value *BrowserSessionReceipt) {
			value.InferenceOperations[1].RootOperationID = value.InferenceOperations[0].RootOperationID
		},
		"missing-operations":   func(value *BrowserSessionReceipt) { value.InferenceOperations = nil },
		"negative-measurement": func(value *BrowserSessionReceipt) { value.Bytes = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := receipt
			mutated.InferenceOperations = append([]BrowserInferenceReceipt(nil), receipt.InferenceOperations...)
			mutate(&mutated)
			if err := validateSettledBrowserSessionReceipt(mutated); err == nil {
				t.Fatal("inconsistent LPBS receipt was accepted")
			}
		})
	}

	overflow := receipt
	overflow.InferenceOperations = []BrowserInferenceReceipt{
		{RootOperationID: "root-max", Provider: "provider-a", Model: "model-a", CreditsCharged: math.MaxInt64},
		{RootOperationID: "root-overflow", Provider: "provider-b", Model: "model-b", CreditsCharged: 1},
	}
	if err := validateSettledBrowserSessionReceipt(overflow); err == nil {
		t.Fatal("overflowing LPBS child totals were accepted")
	}
}
