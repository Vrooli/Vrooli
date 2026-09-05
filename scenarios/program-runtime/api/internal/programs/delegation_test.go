package programs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpawnsAgentManagerRunAndCollectsEvidence(t *testing.T) { // [REQ:PRT-P1-003]
	const executionID = "8d7d9f34-77b5-46d9-9e2d-9a827fe5b4e0"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workflow-executions":
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode start request: %v", err)
			}
			if request["workflowKey"] != "fixture/delegated" || request["owner"] != "fixture" {
				t.Fatalf("start request=%v", request)
			}
			_, _ = w.Write([]byte(`{"execution":{"id":"` + executionID + `","status":"running"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/wait"):
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode wait request: %v", err)
			}
			if request["executionId"] != executionID {
				t.Fatalf("wait request=%v", request)
			}
			_, _ = w.Write([]byte(`{"execution":{"id":"` + executionID + `","status":"succeeded"}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/result"):
			_, _ = w.Write([]byte(`{"execution":{"id":"` + executionID + `","status":"succeeded","output":{"summary":"delegated evidence"},"observations":[{"kind":"run","status":"succeeded"}],"charge_receipt":{"amount_micro_usd":42,"currency":"USD","metering_basis":"agent-manager.run.billing.metered_charge_micro_usd","measured":true,"note":"metered child charge"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := NewHTTPDelegator(server.URL).Delegate(t.Context(), DelegationRequest{
		SessionID:   "session-1",
		Owner:       "fixture",
		WorkflowKey: "fixture/delegated",
		Input:       map[string]any{"task": "inspect"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["execution_id"] != executionID || result["status"] != "succeeded" {
		t.Fatalf("result=%v", result)
	}
	if cost, measured, note := DelegationCharge(result); cost != 42 || !measured || note != "metered child charge" {
		t.Fatalf("receipt cost=%d measured=%t note=%q result=%v", cost, measured, note, result)
	}
	evidence, ok := result["evidence"].(map[string]any)
	if !ok || evidence["summary"] != "delegated evidence" {
		t.Fatalf("evidence=%v", result["evidence"])
	}
}

func TestDelegationChargeIsExplicitWhenAbsent(t *testing.T) {
	cost, measured, note := DelegationCharge(map[string]any{"execution_id": "run-1", "status": "succeeded"})
	if cost != 0 || measured || !strings.Contains(note, "no per-run charge") {
		t.Fatalf("charge=%d measured=%t note=%q", cost, measured, note)
	}
}

func TestDelegationChargeReadsMicroUsd(t *testing.T) {
	cost, measured, note := DelegationCharge(map[string]any{"execution": map[string]any{"total_charge_micro_usd": float64(42)}})
	if cost != 42 || !measured || !strings.Contains(note, "explicit") {
		t.Fatalf("charge=%d measured=%t note=%q", cost, measured, note)
	}
}

func TestDelegationChargeHonorsExplicitUnmeasuredReceipt(t *testing.T) {
	cost, measured, note := DelegationCharge(map[string]any{
		"charge_receipt": map[string]any{
			"currency":         "USD",
			"metering_basis":   "unmeasured",
			"measured":         false,
			"note":             "billing basis unavailable",
			"amount_micro_usd": 0,
		},
	})
	if cost != 0 || measured || note != "billing basis unavailable" {
		t.Fatalf("charge=%d measured=%t note=%q", cost, measured, note)
	}
}

func TestDelegatorsHaveBoundedHTTPClients(t *testing.T) {
	if got := NewHTTPDelegator("http://agent-manager").client.Timeout; got <= 0 {
		t.Fatalf("HTTP delegator timeout=%s", got)
	}
	provided := &http.Client{}
	delegator := NewDiscoveryDelegator(provided)
	if delegator.client == provided || delegator.client.Timeout <= 0 {
		t.Fatalf("discovery delegator did not clone an unbounded client: %#v", delegator.client)
	}
}
