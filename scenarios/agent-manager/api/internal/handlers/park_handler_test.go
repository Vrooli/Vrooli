package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

var parkHandlerSecret = []byte("phase4-park-handler-secret-0123456789abc")

// setupParkHandler builds a handler wired with an identity secret + a claude-code
// runner, plus a directly-seeded running run, for park/wake HTTP tests.
func setupParkHandler(t *testing.T) (*mux.Router, *orchestration.Orchestrator, *domain.Run, string) {
	t.Helper()
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	if err := registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	orch := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:    5 * time.Minute,
			MaxConcurrentRuns: 10,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithIdentitySecret(parkHandlerSecret),
	)

	task := &domain.Task{ID: uuid.New(), Title: "park-handler-task", ScopePath: "src/", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	runID := uuid.New()
	token := parkHandlerToken(t, runID)
	now := time.Now()
	run := &domain.Run{
		ID:                runID,
		TaskID:            task.ID,
		Tag:               runID.String(),
		RunMode:           domain.RunModeInPlace,
		Status:            domain.RunStatusRunning,
		Phase:             domain.RunPhaseExecuting,
		SessionID:         "sess-" + uuid.New().String()[:8],
		StartedAt:         &now,
		LastHeartbeat:     &now,
		ResolvedConfig:    &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:     domain.ApprovalStateNone,
		IdentityTokenHash: identity.HashToken(token),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	h := New(orchestration.NewHandlerServices(orch))
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r, orch, run, token
}

func parkHandlerToken(t *testing.T, runID uuid.UUID) string {
	t.Helper()
	now := time.Now()
	tok, err := identity.GenerateToken(&identity.Claims{
		RunID: runID, IssuedAt: now.Unix(), ExpiresAt: now.Add(24 * time.Hour).Unix(),
	}, parkHandlerSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

// TestParkHandler_OwningTokenParks: POST /park with the owning run's token parks
// the run (200 + parked + clean message).
func TestParkHandler_OwningTokenParks(t *testing.T) {
	router, _, run, token := setupParkHandler(t)

	body := encodeProtoJSON(t, &domainpb.ParkRunRequest{
		Producer:      "test-genie",
		Key:           "agent-manager/run-1",
		IdentityToken: token,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID.String()+"/park", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	var resp domainpb.ParkRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &resp)
	if !resp.Success || resp.Run == nil {
		t.Fatalf("expected success with run, got %+v", &resp)
	}
	if resp.Run.Status != domainpb.RunStatus_RUN_STATUS_PARKED {
		t.Errorf("run status = %v, want PARKED", resp.Run.Status)
	}
	if resp.Message == "" {
		t.Error("expected a clean turn-ending message")
	}
}

// TestParkHandler_ForeignTokenForbidden: a token that does not own the run is
// rejected with 403 and the run stays running.
func TestParkHandler_ForeignTokenForbidden(t *testing.T) {
	router, orch, run, _ := setupParkHandler(t)

	body := encodeProtoJSON(t, &domainpb.ParkRunRequest{
		Producer:      "test-genie",
		Key:           "x/y",
		IdentityToken: parkHandlerToken(t, uuid.New()), // different run
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID.String()+"/park", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", rr.Code, rr.Body.String())
	}
	got, _ := orch.GetRun(context.Background(), run.ID)
	if got.Status != domain.RunStatusRunning {
		t.Errorf("run must stay running after a forbidden park, got %s", got.Status)
	}
}

// TestWakeHandler_Idempotent: waking a non-parked run is a no-op reported as
// success=false (not an error).
func TestWakeHandler_Idempotent(t *testing.T) {
	router, _, run, _ := setupParkHandler(t)

	body := encodeProtoJSON(t, &domainpb.WakeRunRequest{Result: "ignored"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID.String()+"/wake", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	var resp domainpb.WakeRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("waking a running (non-parked) run should report success=false (no-op)")
	}
}
