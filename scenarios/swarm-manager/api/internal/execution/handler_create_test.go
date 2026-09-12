package execution

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workflowcontract"

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
		DataRoot:           root,
		StorePath:          filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer:       testPlanRenderer(),
		AgentService:       &testutil.AgentSpawner{Enabled: true},
		TransitionRegistry: testTransitionRegistry(t),
	})
	service.SetPhasedPlanWorkflow(&stubPhasedPlanWorkflow{startErr: fmt.Errorf("%w: status 500", agentmanager.ErrRequestFailed)})
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
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
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

// [REQ:SWM-P0-005] declared strategy registry exposed for operator selection
func TestStrategies_ReturnsDeclaredExecutionChoiceWithCost(t *testing.T) {
	service := NewService(ServiceConfig{DataRoot: t.TempDir(), StorePath: filepath.Join(t.TempDir(), "executions.json"), PlanRenderer: testPlanRenderer()})
	recorder := httptest.NewRecorder()
	NewHandlerFromService(service).Strategies(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/execution/strategies", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Items []StrategySummary `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 3 || response.Items[0].ID != defaultExecutionStrategy || response.Items[0].CostEstimate <= 0 {
		t.Fatalf("strategies=%+v", response.Items)
	}
}

func TestStrategies_UsesSettledItemAllowanceWhenRequested(t *testing.T) {
	root := t.TempDir()
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer()})
	mustWriteBacklogItem(t, root, "execute", "allowance-item", map[string]any{
		"name": "allowance-item", "title": "Allowance", "description": "desc", "kind": "execute", "status": "ready",
		"execution_limits": map[string]any{"max_slices": 4, "max_tokens": 1000, "max_wall_seconds": 100, "max_turns": 10, "max_charge_micro_usd": 100, "max_children": 2, "max_node_attempts": 4, "max_retries": 2},
	})
	item, err := service.loadBacklogItem("execute", "allowance-item")
	if err != nil {
		t.Fatal(err)
	}
	approvalDigest := digestStrings(item.PlanAcceptance.SubjectVersion, item.PlanAcceptance.PlanContentHash)
	if err := service.store.Save([]Record{{BacklogKind: "execute", BacklogName: "allowance-item", ApprovalDigest: approvalDigest, SettledUsage: &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 250, Turns: 3, WallSeconds: 20, ChargeMicroUSD: 25}}}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/execution/strategies?backlog_kind=execute&backlog_name=allowance-item", nil)
	NewHandlerFromService(service).Strategies(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Items []StrategySummary `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 3 || response.Items[0].RemainingTurns != 7 || response.Items[0].RemainingTokens != 750 || response.Items[0].RemainingWall != 80 || response.Items[0].RemainingCharge != 75 || response.Items[0].CostEstimate != 0.000075 {
		t.Fatalf("strategies=%+v", response.Items)
	}
}
