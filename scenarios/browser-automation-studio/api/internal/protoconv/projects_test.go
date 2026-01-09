package protoconv

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestProjectToProto(t *testing.T) {
	now := time.Now()
	project := &database.ProjectIndex{
		ID:          uuid.New(),
		Name:        "Test Project",
		FolderPath:  "/folder",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	pb := ProjectToProto(project)
	if pb == nil {
		t.Fatalf("expected proto project")
	}
	if pb.Id != project.ID.String() {
		t.Fatalf("id mismatch: %s vs %s", pb.Id, project.ID)
	}
	if pb.Name != project.Name {
		t.Fatalf("name mismatch")
	}
	if pb.FolderPath != project.FolderPath {
		t.Fatalf("folder_path mismatch")
	}
}

func TestProjectStatsToProto(t *testing.T) {
	now := time.Now()
	stats := &database.ProjectStats{
		ProjectID:      uuid.New(),
		WorkflowCount:  5,
		ExecutionCount: 7,
		LastExecution:  &now,
	}

	pb := ProjectStatsToProto(stats)
	if pb.ProjectId != stats.ProjectID.String() {
		t.Fatalf("project_id mismatch")
	}
	if pb.WorkflowCount != int32(stats.WorkflowCount) {
		t.Fatalf("workflow_count mismatch")
	}
	if pb.ExecutionCount != int32(stats.ExecutionCount) {
		t.Fatalf("execution_count mismatch")
	}
	if pb.LastExecution == nil {
		t.Fatalf("expected last_execution set")
	}
}

func TestProjectStatsFromMap(t *testing.T) {
	now := time.Now()
	projectID := uuid.New()
	statsMap := map[string]any{
		"workflow_count":  3,
		"execution_count": float64(4),
		"last_execution":  &now,
	}

	stats := ProjectStatsFromMap(statsMap, projectID)
	if stats.WorkflowCount != 3 || stats.ExecutionCount != 4 {
		t.Fatalf("unexpected stats conversion: %+v", stats)
	}
	if stats.LastExecution == nil || !stats.LastExecution.Equal(now) {
		t.Fatalf("expected last_execution to match")
	}
}
