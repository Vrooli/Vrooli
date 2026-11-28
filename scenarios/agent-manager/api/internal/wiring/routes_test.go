package wiring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/hookreceipt"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliutil"
)

type sandboxHealthStub struct {
	ok     bool
	reason string
}

type providerReadinessStub struct{}

func (providerReadinessStub) ReadProviderReadiness(context.Context) domain.ProviderReadiness {
	return domain.ProviderReadiness{Provider: "workspace-sandbox", State: domain.ProviderReadinessUnknown, Message: "Workspace Sandbox readiness could not be confirmed."}
}

func (s sandboxHealthStub) IsAvailable(context.Context) (bool, string) {
	return s.ok, s.reason
}

func TestSetupRoutesRegistersBaselineHealthAndMetricsWithoutOptionalServices(t *testing.T) {
	router := mux.NewRouter()
	SetupRoutes(router, RouteDependencies{})
	for _, path := range []string{"/health", "/api/v1/health", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Fatalf("route %s was not registered", path)
		}
	}
}

func TestSetupRoutesRegistersProviderReadiness(t *testing.T) {
	router := mux.NewRouter()
	SetupRoutes(router, RouteDependencies{ProviderReadiness: providerReadinessStub{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/provider-readiness", nil)
	match := &mux.RouteMatch{}
	if !router.Match(req, match) {
		t.Fatal("provider readiness route was not registered")
	}
}

func TestSetupRoutesHookReceiptUsesCanonicalServerProvenanceAndAuthnChain(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	task := &domain.Task{ID: uuid.New(), Title: "receipt route", ScopePath: ".", ProjectRoot: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	runID, invocationID := uuid.New(), uuid.New()
	run := &domain.Run{
		ID: runID, TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeInPlace,
		ExecutionMode: domain.ExecutionModeCodecPipe, Status: domain.RunStatusRunning,
		Phase: domain.RunPhaseExecuting, SessionID: "setup-routes-session", CreatedAt: now,
		StartedAt: &now, ResolvedConfig: domain.DefaultRunConfig(),
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(events))
	router := mux.NewRouter()
	SetupRoutes(router, RouteDependencies{DB: db, Orchestrator: orch})
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token != "canonical-run-token" {
			return nil, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
			RunID: runID.String(), Subject: "owner", ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}}, nil
	})
	// api-core/server.Run wraps the application router with this canonical
	// provenance middleware; SetupRoutes installs authn inside that boundary.
	served := provenance.Middleware(verifier)(router)
	store := hookreceipt.NewStore(db)
	credential, err := store.Issue(context.Background(), runID, invocationID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	occurredAt := time.Now().UTC()
	body := fmt.Sprintf(`{"receipt_id":%q,"invocation_id":%q,"session_id":"setup-routes-session","runner":"codex","evaluator_id":"policy:deny","duration_ms":17,"occurred_at":%q}`,
		uuid.NewString(), invocationID.String(), occurredAt.Format(time.RFC3339Nano))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runtime/hook-receipts", strings.NewReader(body))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "canonical-run-token")
	req.Header.Set("X-Vrooli-Hook-Receipt-Token", credential)
	response := httptest.NewRecorder()
	served.ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("SetupRoutes receipt response = %d: %s", response.Code, response.Body.String())
	}
	stored, err := store.ReceiptsForRuns(context.Background(), []uuid.UUID{runID})
	if err != nil || len(stored[runID]) != 1 || stored[runID][0].EvaluatorID != "policy:deny" || stored[runID][0].DurationMS != 17 {
		t.Fatalf("SetupRoutes persisted receipt = %#v, err=%v", stored, err)
	}
}

func TestWorkspaceSandboxHealthCheckerPreservesUnavailableReason(t *testing.T) {
	checker := workspaceSandboxHealthChecker(sandboxHealthStub{reason: "endpoint is not configured"})
	if checker == nil {
		t.Fatal("workspace sandbox health checker is nil")
	}
	result := checker.Check(context.Background())
	if result.Connected || result.Error == nil || result.Error.Error() != "endpoint is not configured" {
		t.Fatalf("health result = %+v", result)
	}
}

func TestWorkspaceSandboxHealthCheckerAcceptsAvailableProvider(t *testing.T) {
	result := workspaceSandboxHealthChecker(sandboxHealthStub{ok: true}).Check(context.Background())
	if !result.Connected || result.Error != nil {
		t.Fatalf("health result = %+v", result)
	}
}

func TestWorkspaceSandboxHealthCheckerSkipsMissingProvider(t *testing.T) {
	if checker := workspaceSandboxHealthChecker(nil); checker != nil {
		t.Fatal("missing provider should not register a health check")
	}
}

func TestCatalogFreshnessCoversEveryDeclaredHarness(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	byRunner := map[string]agentcatalog.CatalogFreshness{}
	for _, freshness := range catalogFreshnessReader(repoRoot)(context.Background()) {
		byRunner[freshness.Runner] = freshness
	}
	for _, runner := range domain.ValidRunnerTypes() {
		freshness, ok := byRunner[string(runner)]
		if !ok {
			t.Errorf("runner %q is missing from catalog freshness", runner)
			continue
		}
		if freshness.Status == "invalid" || freshness.Status == "unknown" {
			t.Errorf("runner %q catalog cannot be read: %+v", runner, freshness)
		}
	}
	if len(byRunner) != len(domain.ValidRunnerTypes()) {
		t.Fatalf("Agent Manager's runner types disagree with the harness descriptors: %v vs %v", domain.ValidRunnerTypes(), byRunner)
	}
}
