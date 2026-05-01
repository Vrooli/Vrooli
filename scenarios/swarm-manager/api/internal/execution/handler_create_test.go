package execution

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil/mocks"
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
		RootDir:   root,
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
