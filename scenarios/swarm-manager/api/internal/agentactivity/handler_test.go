package agentactivity

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func setupHandlerTest(t *testing.T, records []Record, raw *stubAgentService) (*mux.Router, *Service) {
	t.Helper()

	service := newTestService(t, raw)
	if err := service.store.Save(records); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	router := mux.NewRouter()
	NewHandler(service).RegisterRoutes(router)
	return router, service
}

func TestHandlerList_FiltersAgentActivities(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{
		enabled: true,
		runStates: map[string]agentmanager.RunState{
			"run-active": {RunID: "run-active", Status: "running"},
			"run-other":  {RunID: "run-other", Status: "running"},
		},
	}
	router, _ := setupHandlerTest(t, []Record{
		{
			ActivityID:      "act-match",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "execute",
			OwnerName:       "task-a",
			OwnerTitle:      "Task A",
			ExecutionID:     "exec-1",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-active",
			Status:          StatusRunning,
			RequestedAt:     "2026-03-28T12:00:00Z",
			UpdatedAt:       "2026-03-28T12:00:00Z",
		},
		{
			ActivityID:      "act-other-owner",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "fix",
			OwnerName:       "task-b",
			OwnerTitle:      "Task B",
			ExecutionID:     "exec-2",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-other",
			Status:          StatusRunning,
			RequestedAt:     "2026-03-28T12:01:00Z",
			UpdatedAt:       "2026-03-28T12:01:00Z",
		},
		{
			ActivityID:      "act-complete",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "execute",
			OwnerName:       "task-a",
			OwnerTitle:      "Task A",
			ExecutionID:     "exec-1",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-complete",
			Status:          StatusComplete,
			RequestedAt:     "2026-03-28T11:00:00Z",
			UpdatedAt:       "2026-03-28T11:05:00Z",
			FinishedAt:      "2026-03-28T11:05:00Z",
		},
	}, raw)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-activities?owner_type=backlog&owner_kind=execute&owner_name=task-a&execution_id=exec-1&purpose=process&status=running&run_id=run-active&active=true", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", contentType)
	}

	resp := testutil.DecodeProtoJSON(t, rec, &apipb.ListAgentActivitiesResponse{})
	if len(resp.GetItems()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.GetItems()))
	}
	if got := resp.GetItems()[0].GetActivityId(); got != "act-match" {
		t.Fatalf("expected act-match, got %q", got)
	}
	if got := resp.GetItems()[0].GetExecutionId(); got != "exec-1" {
		t.Fatalf("expected execution exec-1, got %q", got)
	}
	if raw.runStateCalls != 0 {
		t.Fatalf("list handler refreshed run state %d times, want snapshot-only read", raw.runStateCalls)
	}
}

func TestHandlerList_RejectsInvalidActiveQuery(t *testing.T) {
	t.Parallel()

	router, _ := setupHandlerTest(t, nil, &stubAgentService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-activities?active=maybe", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}

func TestHandlerGet_ReturnsAgentActivity(t *testing.T) {
	t.Parallel()

	router, _ := setupHandlerTest(t, []Record{
		{
			ActivityID:      "act-1",
			OwnerType:       OwnerScenario,
			OwnerName:       "scenario-alpha",
			OwnerTitle:      "Scenario Alpha",
			ExecutionID:     "exec-9",
			Purpose:         PurposeSpecSync,
			InteractionType: InteractionSpawn,
			RunID:           "run-9",
			TaskID:          "task-9",
			Status:          StatusComplete,
			RequestedAt:     "2026-03-28T12:00:00Z",
			StartedAt:       "2026-03-28T12:01:00Z",
			FinishedAt:      "2026-03-28T12:05:00Z",
			RequestedBy:     "tester",
			UpdatedAt:       "2026-03-28T12:05:00Z",
		},
	}, &stubAgentService{enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-activities/act-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeProtoJSON(t, rec, &apipb.AgentActivityResponse{})
	if resp.GetActivity().GetActivityId() != "act-1" {
		t.Fatalf("expected activity act-1, got %q", resp.GetActivity().GetActivityId())
	}
	if resp.GetActivity().GetOwnerType() != "scenario" {
		t.Fatalf("expected scenario owner, got %q", resp.GetActivity().GetOwnerType())
	}
	if resp.GetActivity().GetPurpose() != "spec_sync" {
		t.Fatalf("expected spec_sync purpose, got %q", resp.GetActivity().GetPurpose())
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	t.Parallel()

	router, _ := setupHandlerTest(t, nil, &stubAgentService{enabled: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-activities/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatusNotFound(t, rec)
}
