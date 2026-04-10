package secrets

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestLocalSecretSetGetDeleteScenarioScope(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)

	scenarioDir := filepath.Join(root, "scenarios", "landing-page-business-suite", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/local-secrets/{scope}/{key}", HandleSetLocalSecret()).Methods("PUT")
	router.HandleFunc("/api/v1/local-secrets/{scope}/{key}", HandleGetLocalSecret()).Methods("GET")
	router.HandleFunc("/api/v1/local-secrets/{scope}/{key}", HandleDeleteLocalSecret()).Methods("DELETE")

	// Set
	setBody := bytes.NewBufferString(`{"value":"abc123"}`)
	setReq := httptest.NewRequest(http.MethodPut, "/api/v1/local-secrets/scenario/LPBS_SERVICE_SECRET?scenario_id=landing-page-business-suite", setBody)
	setRec := httptest.NewRecorder()
	router.ServeHTTP(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("set expected 200, got %d body=%s", setRec.Code, setRec.Body.String())
	}

	// Get
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/local-secrets/scenario/LPBS_SERVICE_SECRET?scenario_id=landing-page-business-suite&reveal=true", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var getResp LocalSecretGetResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getResp.Value != "abc123" {
		t.Fatalf("expected value abc123, got %q", getResp.Value)
	}

	// Delete
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/local-secrets/scenario/LPBS_SERVICE_SECRET?scenario_id=landing-page-business-suite", nil)
	delRec := httptest.NewRecorder()
	router.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d body=%s", delRec.Code, delRec.Body.String())
	}
}

func TestLocalSecretGenerateHex(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir workspace .vrooli: %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/local-secrets/{scope}/{key}", HandleSetLocalSecret()).Methods("PUT")
	router.HandleFunc("/api/v1/local-secrets/{scope}/{key}", HandleGetLocalSecret()).Methods("GET")

	setReq := httptest.NewRequest(http.MethodPut, "/api/v1/local-secrets/workspace/LPBS_SERVICE_SECRET", bytes.NewBufferString(`{"generate":"hex:64"}`))
	setRec := httptest.NewRecorder()
	router.ServeHTTP(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("set expected 200, got %d body=%s", setRec.Code, setRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/local-secrets/workspace/LPBS_SERVICE_SECRET?reveal=true", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	var getResp LocalSecretGetResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if len(getResp.Value) != 64 {
		t.Fatalf("expected generated length 64, got %d", len(getResp.Value))
	}
}
