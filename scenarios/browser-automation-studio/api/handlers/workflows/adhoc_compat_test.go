package workflows

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeAdhocRequestMapsShortExecutionModeBeforeConnectDecode(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	handler := normalizeAdhocRequest(next)
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ExecuteAdhocWorkflow", strings.NewReader(`{"flow_definition":{"metadata":{"execution_mode":"mutating"}}}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "EXECUTION_MODE_MUTATING") {
		t.Fatalf("normalized body = %s", response.Body.String())
	}
}

func TestNormalizeAdhocRequestLeavesOtherProceduresUntouched(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	})
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ValidateWorkflow", strings.NewReader(`{"execution_mode":"mutating"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	normalizeAdhocRequest(next).ServeHTTP(response, req)
	if response.Body.String() != `{"execution_mode":"mutating"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}
