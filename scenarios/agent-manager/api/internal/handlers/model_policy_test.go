package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelpolicy"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func setupModelPolicyHandler(t *testing.T) (*orchestration.Orchestrator, *modelpolicy.State, *mux.Router, string) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	if err := registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	source, err := os.ReadFile(modelpolicy.ResolvePath())
	if err != nil {
		t.Fatalf("read policy fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "model-policy-catalog.json")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatalf("write policy fixture: %v", err)
	}
	state, err := modelpolicy.NewState(path, modelpolicy.Requirement{Required: true, Reason: "handler test"})
	if err != nil {
		t.Fatalf("new model policy state: %v", err)
	}
	orch := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, DefaultProjectRoot: t.TempDir()}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithModelPolicyState(state),
	)
	handler := New(orch, WithModelPolicyState(state))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return orch, state, router, path
}

// [REQ:REQ-P1-004] Operators can inspect policy state without mutating it.
func TestModelPolicyStatusCatalogAndLegacyMutationCutover(t *testing.T) {
	_, state, router, _ := setupModelPolicyHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/model-policy/status", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d: %s", recorder.Code, recorder.Body.String())
	}
	var statusResponse apipb.GetModelPolicyStatusResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &statusResponse)
	if statusResponse.Status == nil || !statusResponse.Status.Ready || statusResponse.Status.ActiveDigest != state.Status().ActiveDigest {
		t.Fatalf("status response = %+v", statusResponse.Status)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/model-policy/catalog", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var catalogResponse apipb.GetModelPolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &catalogResponse)
	if catalogResponse.Catalog == nil || len(catalogResponse.Catalog.Policies) == 0 || len(catalogResponse.Catalog.Runners) == 0 {
		t.Fatalf("catalog response = %+v", catalogResponse.Catalog)
	}

	request = httptest.NewRequest(http.MethodPut, "/api/v1/runner-models", bytes.NewBufferString(`{}`))
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("legacy PUT status = %d, want 404", recorder.Code)
	}
}

// [REQ:REQ-P1-004] Failed controlled reloads preserve the active revision.
func TestModelPolicyValidateAndFailedReloadPreserveActiveRevision(t *testing.T) {
	_, state, router, path := setupModelPolicyHandler(t)
	before := state.Status().ActiveDigest

	request := httptest.NewRequest(http.MethodPost, "/api/v1/model-policy/validate", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var validateResponse apipb.ValidateModelPolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &validateResponse)
	if !validateResponse.Valid || validateResponse.CandidateDigest != before || validateResponse.ActiveDigest != before {
		t.Fatalf("validate response = %+v", &validateResponse)
	}

	if err := os.WriteFile(path, []byte(`{"schemaVersion":99}`), 0o644); err != nil {
		t.Fatalf("write invalid policy: %v", err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/model-policy/reload", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var reloadResponse apipb.ReloadModelPolicyCatalogResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &reloadResponse)
	if reloadResponse.Activated || reloadResponse.Diagnostic == nil || reloadResponse.Status == nil || reloadResponse.Status.ActiveDigest != before {
		t.Fatalf("reload response = %+v", &reloadResponse)
	}
}

// [REQ:REQ-P1-004] Profile explanation exposes the candidate snapshot run creation uses.
func TestExplainModelPolicyProfileUsesCurrentResolution(t *testing.T) {
	orch, _, router, _ := setupModelPolicyHandler(t)
	profile, err := orch.CreateProfile(context.Background(), &domain.AgentProfile{
		Name: "explain-profile", ProfileKey: "explain-profile", RunnerType: domain.RunnerTypeClaudeCode,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	body := encodeProtoJSON(t, &apipb.ExplainModelPolicyRequest{
		Target: &apipb.ExplainModelPolicyRequest_ProfileId{ProfileId: profile.ID.String()},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/model-policy/explain", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("explain status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response apipb.ExplainModelPolicyResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &response)
	if response.TargetType != "profile" || response.Snapshot == nil || response.Snapshot.CatalogDigest == "" || len(response.Snapshot.Candidates) == 0 {
		t.Fatalf("explain response = %+v", &response)
	}
}
