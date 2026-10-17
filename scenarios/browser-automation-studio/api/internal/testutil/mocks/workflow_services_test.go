package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

func TestWorkflowCatalogServiceDelegatesConfiguredFunctions(t *testing.T) {
	expectedID := uuid.New()
	service := &WorkflowCatalogService{
		ListWorkflowsFunc: func(_ context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
			if req.GetLimit() != 7 {
				t.Fatalf("expected limit 7, got %d", req.GetLimit())
			}
			return &basapi.ListWorkflowsResponse{
				Workflows: []*basapi.WorkflowSummary{{Id: expectedID.String(), Name: "Imported workflow"}},
				Total:     1,
			}, nil
		},
	}

	limit := int32(7)
	response, err := service.ListWorkflows(context.Background(), &basapi.ListWorkflowsRequest{Limit: &limit})
	if err != nil {
		t.Fatalf("list workflows: %v", err)
	}
	if response.GetTotal() != 1 {
		t.Fatalf("expected one workflow, got %d", response.GetTotal())
	}
	if response.GetWorkflows()[0].GetId() != expectedID.String() {
		t.Fatalf("expected workflow id %s, got %s", expectedID, response.GetWorkflows()[0].GetId())
	}
}

func TestWorkflowCatalogServiceDefaultsAreDeterministic(t *testing.T) {
	service := &WorkflowCatalogService{}

	if got := service.CheckHealth(); got != "ok" {
		t.Fatalf("expected default health ok, got %q", got)
	}
	healthy, err := service.CheckAutomationHealth(context.Background())
	if err != nil {
		t.Fatalf("check automation health: %v", err)
	}
	if !healthy {
		t.Fatal("expected default automation health to be healthy")
	}
	response, err := service.ListWorkflows(context.Background(), &basapi.ListWorkflowsRequest{})
	if err != nil {
		t.Fatalf("list workflows: %v", err)
	}
	if response.GetTotal() != 0 || len(response.GetWorkflows()) != 0 {
		t.Fatalf("expected empty workflow list, got total=%d len=%d", response.GetTotal(), len(response.GetWorkflows()))
	}
}

func TestWorkflowExecutionServiceDelegatesConfiguredFunctions(t *testing.T) {
	expectedID := uuid.New()
	service := &WorkflowExecutionService{
		GetExecutionFunc: func(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
			if id != expectedID {
				t.Fatalf("expected execution id %s, got %s", expectedID, id)
			}
			return &database.ExecutionIndex{ID: id, Status: database.ExecutionStatusCompleted}, nil
		},
	}

	execution, err := service.GetExecution(context.Background(), expectedID)
	if err != nil {
		t.Fatalf("get execution: %v", err)
	}
	if execution.Status != database.ExecutionStatusCompleted {
		t.Fatalf("expected completed status, got %s", execution.Status)
	}
}

func TestWorkflowExecutionServiceDefaultsAreDeterministic(t *testing.T) {
	service := &WorkflowExecutionService{}

	response, err := service.ExecuteWorkflowAPI(context.Background(), &basapi.ExecuteWorkflowRequest{})
	if err != nil {
		t.Fatalf("execute workflow api: %v", err)
	}
	if response.GetExecutionId() == "" {
		t.Fatal("expected default execution id")
	}
	timeline, err := service.GetExecutionTimeline(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("get execution timeline: %v", err)
	}
	if timeline == nil || len(timeline.Frames) != 0 {
		t.Fatalf("expected empty timeline, got %#v", timeline)
	}
}

func TestWorkflowServiceFakesLeaveUnarrangedBranchesExplicit(t *testing.T) {
	catalog := &WorkflowCatalogService{}
	if _, err := catalog.GetProject(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected unarranged catalog method to return an error")
	}

	execution := &WorkflowExecutionService{}
	if _, err := execution.ExecuteWorkflow(context.Background(), uuid.New(), nil); err == nil {
		t.Fatal("expected unarranged execution method to return an error")
	}
}

var (
	_ workflow.CatalogService   = (*WorkflowCatalogService)(nil)
	_ workflow.ExecutionService = (*WorkflowExecutionService)(nil)
)
