package execution

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/testutil/mocks"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestCreate_ReturnsBadGatewayForAgentManagerRequestFailure(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "request-fail-idea", map[string]any{
		"name":        "request-fail-idea",
		"title":       "Request Fail Idea",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "request-fail-idea")

	service := NewService(ServiceConfig{
		DataRoot:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: &mocks.AgentSpawner{
			Enabled:  true,
			SpawnErr: fmt.Errorf("%w: status 500", agentmanager.ErrRequestFailed),
		},
	})
	handler := NewHandlerFromService(service)

	reqBody := `{"backlogKind":"idea","backlogName":"request-fail-idea","mode":"yolo"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadGateway, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "check agent-manager health/logs and retry") {
		t.Fatalf("expected remediation message, got %q", rec.Body.String())
	}
}

func TestList_UsesSnapshotWithoutRefreshingRunState(t *testing.T) {
	root := t.TempDir()
	agent := &snapshotAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
	})
	if err := service.store.Save([]Record{{
		ExecutionID: "exec-1",
		BacklogKind: "idea",
		BacklogName: "slow-read",
		Status:      StatusRunning,
		RunID:       "run-1",
		Mode:        ModeYOLO,
		CreatedAt:   "2026-05-14T00:00:00Z",
		UpdatedAt:   "2026-05-14T00:00:00Z",
	}}); err != nil {
		t.Fatalf("save executions: %v", err)
	}
	handler := NewHandlerFromService(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeProtoJSON(t, rec, &apipb.ListExecutionResponse{})
	if len(resp.GetItems()) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(resp.GetItems()))
	}
	if agent.runStateCalls != 0 {
		t.Fatalf("list handler refreshed run state %d times, want snapshot-only read", agent.runStateCalls)
	}
}
