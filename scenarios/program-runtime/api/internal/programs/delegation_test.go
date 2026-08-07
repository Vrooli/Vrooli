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
			_, _ = w.Write([]byte(`{"execution":{"id":"` + executionID + `","status":"succeeded","output":{"summary":"delegated evidence"},"observations":[{"kind":"run","status":"succeeded"}]}}`))
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
	evidence, ok := result["evidence"].(map[string]any)
	if !ok || evidence["summary"] != "delegated evidence" {
		t.Fatalf("evidence=%v", result["evidence"])
	}
}
