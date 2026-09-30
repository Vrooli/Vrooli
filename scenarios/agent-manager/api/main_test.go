package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/identity"
	searchregister "github.com/vrooli/searchregister-go"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type maintenanceOwnerFixture struct{}

func (maintenanceOwnerFixture) Verify(context.Context, string) (identity.Principal, error) {
	return identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "fixture-owner", Scopes: []string{"agent-manager:write"}}, nil
}

func TestRouterMountsDurableMaintenanceWithSharedOrchestrationGate(t *testing.T) {
	// The real registry ensures the maintenance schema alongside all other
	// domains. No startup scheduler, credential exchange or host process runs.
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithMaintenanceGate(gate))
	interlock, err := maintenance.NewScenarioInterlock(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reader := maintenance.NewInventory(db, func(ctx context.Context, ref maintenance.ExecutorRef) (maintenance.ExecutorEvidence, error) {
		t.Error("empty fixture unexpectedly has an executor")
		return maintenance.ExecutorEvidence{}, errors.New("unknown executor")
	})
	h, err := maintenance.NewHandler(gate, reader.Observe, maintenanceOwnerFixture{}, interlock)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, router: mux.NewRouter(), orchestrator: orch, maintenance: h}
	router := s.Router()
	request := httptest.NewRequest(http.MethodPost, maintenance.AdmissionPath+"/enter", strings.NewReader(`{"reason":"fixture rollout"}`))
	request.Header.Set("Authorization", "Bearer fixture")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("mounted begin=%d %s", response.Code, response.Body.String())
	}
	if _, err := orch.CreateRun(t.Context(), orchestration.CreateRunRequest{TaskID: uuid.New(), Force: true}); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("HTTP closed a different gate from orchestration: %v", err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, maintenance.AdmissionPath, nil))
	var standing maintenance.Standing
	if err := json.Unmarshal(response.Body.Bytes(), &standing); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !standing.Drained || !standing.Closed || standing.Revision <= 0 || standing.LifecycleInterlock != maintenance.ScenarioLockV1 || standing.Inventory == nil || standing.Inventory.Remaining == nil || *standing.Inventory.Remaining != 0 {
		t.Fatalf("mounted owner proof=%d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, maintenance.AdmissionPath+"/resume", strings.NewReader(`{"revision":1}`))
	request.Header.Set("Authorization", "Bearer fixture")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("mounted resume=%d %s", response.Code, response.Body.String())
	}
	release, err := gate.Admit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestRouterReportsInitializingBeforeRecoveryAndUsesExistingHealthWhenReady(t *testing.T) {
	recovery := maintenance.NewRecovery()
	srv := &Server{router: mux.NewRouter(), db: &database.DB{}, recovery: recovery}
	for _, path := range []string{"/health", "/api/v1/health"} {
		srv.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	}
	srv.router.HandleFunc(apiconnect.AgentManagerServiceHealthProcedure, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := srv.Router()
	for _, path := range []string{"/health", "/api/v1/health"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		wantCode := http.StatusServiceUnavailable
		if path == "/health" {
			wantCode = http.StatusOK
		}
		if response.Code != wantCode || !strings.Contains(response.Body.String(), `"status":"initializing"`) || !strings.Contains(response.Body.String(), `"readiness":false`) {
			t.Errorf("before recovery %s=%d %s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, apiconnect.AgentManagerServiceHealthProcedure, nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"unavailable"`) {
		t.Errorf("before recovery RPC=%d %s", response.Code, response.Body.String())
	}
	recovery.Start(t.Context(), nil)
	<-recovery.Done()
	defer recovery.Stop()
	for _, path := range []string{"/health", "/api/v1/health", apiconnect.AgentManagerServiceHealthProcedure} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Errorf("ready %s=%d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestDatabaseConfigUsesGovernedPoolLevers(t *testing.T) {
	config := databaseConfigFromLevers("file:test.db", agentconfig.StorageLevers{
		MaxOpenConns: 17, MaxIdleConns: 4, ConnMaxLifetime: 7 * time.Minute,
	})
	if config.MaxOpenConns != 5 || config.MaxIdleConns != 4 || config.ConnMaxLifetime != 7*time.Minute {
		t.Fatalf("database pool ignored storage levers: %+v", config)
	}
}

// TestNewServerBuildsAndShutsDownTheRealCompositionRoot is a lightweight
// composition-root proof: it uses a disposable SQLite store and a bounded
// empty project root, but otherwise exercises the same graph construction,
// recovery ordering, middleware, and cleanup path as production startup.
func TestNewServerBuildsAndShutsDownTheRealCompositionRoot(t *testing.T) {
	// Use the current storage identity seam. AM_SQLITE_PATH is no longer read.
	t.Setenv("SCENARIO_NAME", "agent-manager")
	t.Setenv("VROOLI_SCENARIO", "agent-manager")
	t.Setenv("SCENARIO_DATA_DIR", t.TempDir())
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "agent-manager")
	t.Setenv("VROOLI_VARIANT", "live")
	t.Setenv("UPLOAD_DIR", t.TempDir())
	t.Setenv("PROJECT_ROOT", t.TempDir())
	server, err := NewServer()
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	if server.Router() == nil || server.orchestrator == nil || server.reconciler == nil || server.awaitRegistry == nil || server.workflowNudger == nil {
		t.Fatalf("incomplete service graph: %+v", server)
	}
	t.Cleanup(func() {
		if err := server.Cleanup(); err != nil {
			t.Errorf("cleanup server: %v", err)
		}
	})
	<-server.recovery.Done()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	server.Router().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEnvOrEmptyReflectsConfiguredAndAbsentValues(t *testing.T) {
	t.Setenv("AM_TEST_ENV", "configured")
	if got := envOrEmpty("AM_TEST_ENV"); got != "configured" {
		t.Fatalf("configured env=%q", got)
	}
	if got := envOrEmpty("AM_UNSET_TEST_ENV"); got != "" {
		t.Fatalf("unset env=%q", got)
	}
}

func TestConversationSearchDescriptorConformsToLiveProviderContract(t *testing.T) {
	file, err := aisearch.LoadSearchFile("../.vrooli/search.json")
	if err != nil {
		t.Fatalf("load committed search descriptor: %v", err)
	}
	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map committed search descriptor: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptor count=%d, want 1", len(descriptors))
	}
	if err := searchregister.ValidateRegistration(file.Providers[0]); err != nil {
		t.Fatalf("descriptor would be rejected during live self-registration: %v", err)
	}
	d := descriptors[0]
	if d.GetProviderId() != "agent-manager.runs" || d.GetLifecycle() != registryv1.Lifecycle_LIFECYCLE_PRODUCTION || d.GetScope() != registryv1.Scope_SCOPE_PROJECT {
		t.Fatalf("unexpected provider identity/state/scope: %+v", d)
	}
	if d.GetType() != "run" || d.GetEndpoint().GetHttpJson().GetScenarioId() != "agent-manager" || d.GetStatusEndpoint() == nil || d.GetReindexEndpoint() == nil || d.GetConfigEndpoint() == nil {
		t.Fatalf("incomplete live provider contract: %+v", d)
	}
	if got := d.GetResultMapping().GetIdField(); got != "stableHitId" {
		t.Fatalf("id mapping=%q, want native stableHitId", got)
	}
	mapping := d.GetResultMapping()
	if got := mapping.GetMetadataFields()["run_id"]; got != "runId" {
		t.Fatalf("run identity mapping=%q, want runId", got)
	}
	if mapping.GetRankEvidenceField() != "rankEvidence" || mapping.GetCoverageField() != "coverage" || mapping.GetDegradationsField() != "degradations" || mapping.GetNextCursorField() != "nextPageCursor" {
		t.Fatalf("incomplete native evidence mappings: %+v", mapping)
	}
}

func TestSearchRegistrationStartsBeforeListener(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	registerAt := strings.Index(text, "srv.startSearchRegistration(")
	serveAt := strings.Index(text, "server.Run(")
	if registerAt < 0 || serveAt < 0 || registerAt >= serveAt {
		t.Fatalf("registration must start before listener: register=%d serve=%d", registerAt, serveAt)
	}
}

func TestRecoveryStepsPrioritizeSelfDeclarations(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	selfAt := strings.Index(text, `add("self_declarations"`)
	workflowAt := strings.Index(text, `add("workflow_accounting"`)
	otherAt := strings.Index(text, `add("scenario_declarations"`)
	if selfAt < 0 || workflowAt < 0 || otherAt < 0 {
		t.Fatalf("recovery steps missing self/workflow/scenario declarations")
	}
	if selfAt >= workflowAt || selfAt >= otherAt {
		t.Fatalf("self declarations must precede historical reconciliation: self=%d workflow=%d scenario=%d", selfAt, workflowAt, otherAt)
	}
}

func TestSearchControlTokensRejectEmptyAndRoundTripMintedToken(t *testing.T) {
	tokens := newSearchControlTokens()
	tokens.set("agent-manager.runs", "")
	if got := tokens.get("agent-manager.runs"); got != "" {
		t.Fatalf("empty token persisted: %q", got)
	}
	tokens.set("agent-manager.runs", "minted-token")
	if got := tokens.get("agent-manager.runs"); got != "minted-token" {
		t.Fatalf("token=%q", got)
	}
}
