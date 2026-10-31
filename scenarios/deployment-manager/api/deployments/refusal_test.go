package deployments

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestDeployRefusesUnimplementedDispatcher(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["reason"] != "dispatcher_not_implemented" {
		t.Fatalf("reason = %v", response["reason"])
	}
	if _, exists := response["deployment_id"]; exists {
		t.Fatal("refusal fabricated a deployment identifier")
	}
}

func TestDeploymentStatusRefusesUnknownIdentifier(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/unknown", nil)
	req = mux.SetURLVars(req, map[string]string{"deployment_id": "unknown"})
	rec := httptest.NewRecorder()

	h.Status(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
