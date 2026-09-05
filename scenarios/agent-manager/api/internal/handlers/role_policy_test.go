package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/rolepolicy"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func setupRolePolicyHandler(t *testing.T) (*rolepolicy.State, *mux.Router, string) {
	t.Helper()
	source, err := os.ReadFile(rolepolicy.ResolvePath())
	if err != nil {
		t.Fatalf("read role policy fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "role-policy-catalog.json")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatalf("write role policy fixture: %v", err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true, Reason: "handler test"})
	if err != nil {
		t.Fatalf("new role policy state: %v", err)
	}
	handler := New(orchestration.EmptyHandlerServices(), WithRolePolicyState(state))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return state, router, path
}

func TestRolePolicyStatusCatalogAndLegacySurfaceRemoval(t *testing.T) {
	state, router, _ := setupRolePolicyHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/role-policy/status", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d: %s", recorder.Code, recorder.Body.String())
	}
	var statusResponse apipb.GetRolePolicyStatusResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &statusResponse)
	if statusResponse.Status == nil || !statusResponse.Status.Ready || statusResponse.Status.ActiveDigest != state.Status().ActiveDigest {
		t.Fatalf("status response = %+v", statusResponse.Status)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/role-policy/catalog", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var catalogResponse apipb.GetRolePolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &catalogResponse)
	if catalogResponse.Catalog == nil || len(catalogResponse.Catalog.Roles) == 0 || catalogResponse.Catalog.DefaultRole == "" {
		t.Fatalf("catalog response = %+v", catalogResponse.Catalog)
	}
	for _, role := range catalogResponse.Catalog.Roles {
		for _, candidate := range role.Candidates {
			if candidate.ResourceRole == "" {
				t.Fatalf("role candidate omitted resource role: %+v", candidate)
			}
		}
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/model-policy/catalog", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("legacy catalog status = %d, want 404", recorder.Code)
	}
}

func TestRolePolicyValidateAndFailedReloadPreserveActiveRevision(t *testing.T) {
	state, router, path := setupRolePolicyHandler(t)
	before := state.Status().ActiveDigest

	request := httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/validate", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var validateResponse apipb.ValidateRolePolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &validateResponse)
	if !validateResponse.Valid || validateResponse.CandidateDigest != before || validateResponse.ActiveDigest != before {
		t.Fatalf("validate response = %+v", &validateResponse)
	}

	if err := os.WriteFile(path, []byte(`{"schemaVersion":99}`), 0o644); err != nil {
		t.Fatalf("write invalid role policy: %v", err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/reload", bytes.NewBufferString(`{}`))
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var reloadResponse apipb.ReloadRolePolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &reloadResponse)
	if reloadResponse.Activated || reloadResponse.Diagnostic == nil || reloadResponse.Status == nil || reloadResponse.Status.ActiveDigest != before {
		t.Fatalf("reload response = %+v", &reloadResponse)
	}
}
