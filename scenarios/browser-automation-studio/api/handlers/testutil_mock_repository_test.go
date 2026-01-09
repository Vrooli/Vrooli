package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestMockRepository_ProjectOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	t.Run("create and get project", func(t *testing.T) {
		project := &database.ProjectIndex{
			Name:       "test-project",
			FolderPath: "/projects/test",
		}
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		if project.ID == uuid.Nil {
			t.Fatal("Expected project ID to be set")
		}

		got, err := repo.GetProject(ctx, project.ID)
		if err != nil {
			t.Fatalf("GetProject failed: %v", err)
		}
		if got.Name != project.Name {
			t.Fatalf("Expected name %s, got %s", project.Name, got.Name)
		}
	})

	t.Run("get project by name", func(t *testing.T) {
		got, err := repo.GetProjectByName(ctx, "test-project")
		if err != nil {
			t.Fatalf("GetProjectByName failed: %v", err)
		}
		if got.Name != "test-project" {
			t.Fatalf("Expected name test-project, got %s", got.Name)
		}
	})

	t.Run("get project by folder path", func(t *testing.T) {
		got, err := repo.GetProjectByFolderPath(ctx, "/projects/test")
		if err != nil {
			t.Fatalf("GetProjectByFolderPath failed: %v", err)
		}
		if got.FolderPath != "/projects/test" {
			t.Fatalf("Expected folder path /projects/test, got %s", got.FolderPath)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		_, err := repo.GetProject(ctx, uuid.New())
		if err != database.ErrNotFound {
			t.Fatalf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list projects", func(t *testing.T) {
		projects, err := repo.ListProjects(ctx, 10, 0)
		if err != nil {
			t.Fatalf("ListProjects failed: %v", err)
		}
		if len(projects) != 1 {
			t.Fatalf("Expected 1 project, got %d", len(projects))
		}
	})

	t.Run("delete project", func(t *testing.T) {
		project := &database.ProjectIndex{Name: "to-delete", FolderPath: "/delete"}
		_ = repo.CreateProject(ctx, project)

		if err := repo.DeleteProject(ctx, project.ID); err != nil {
			t.Fatalf("DeleteProject failed: %v", err)
		}

		_, err := repo.GetProject(ctx, project.ID)
		if err != database.ErrNotFound {
			t.Fatalf("Expected ErrNotFound after delete, got %v", err)
		}
	})
}

func TestMockRepository_WorkflowOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{ID: projectID, Name: "test-project", FolderPath: "/test"})

	t.Run("create and get workflow", func(t *testing.T) {
		workflow := &database.WorkflowIndex{
			ProjectID:  &projectID,
			Name:       "test-workflow",
			FolderPath: "/test/workflows",
			FilePath:   "/test/workflows/test.json",
		}
		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("CreateWorkflow failed: %v", err)
		}
		if workflow.Version != 1 {
			t.Fatalf("Expected version 1, got %d", workflow.Version)
		}

		got, err := repo.GetWorkflow(ctx, workflow.ID)
		if err != nil {
			t.Fatalf("GetWorkflow failed: %v", err)
		}
		if got.Name != workflow.Name {
			t.Fatalf("Expected name %s, got %s", workflow.Name, got.Name)
		}
	})

	t.Run("get workflow by name", func(t *testing.T) {
		got, err := repo.GetWorkflowByName(ctx, "test-workflow", "/test/workflows")
		if err != nil {
			t.Fatalf("GetWorkflowByName failed: %v", err)
		}
		if got.Name != "test-workflow" {
			t.Fatalf("Expected name test-workflow, got %s", got.Name)
		}
	})

	t.Run("list workflows by project", func(t *testing.T) {
		workflows, err := repo.ListWorkflowsByProject(ctx, projectID, 10, 0)
		if err != nil {
			t.Fatalf("ListWorkflowsByProject failed: %v", err)
		}
		if len(workflows) != 1 {
			t.Fatalf("Expected 1 workflow, got %d", len(workflows))
		}
	})
}

func TestMockRepository_ExecutionOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	workflowID := uuid.New()

	t.Run("create and get execution", func(t *testing.T) {
		execution := &database.ExecutionIndex{
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusRunning,
			StartedAt:  time.Now(),
		}
		if err := repo.CreateExecution(ctx, execution); err != nil {
			t.Fatalf("CreateExecution failed: %v", err)
		}

		got, err := repo.GetExecution(ctx, execution.ID)
		if err != nil {
			t.Fatalf("GetExecution failed: %v", err)
		}
		if got.Status != database.ExecutionStatusRunning {
			t.Fatalf("Expected status %s, got %s", database.ExecutionStatusRunning, got.Status)
		}
	})

	t.Run("update execution status", func(t *testing.T) {
		execution := &database.ExecutionIndex{
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusRunning,
			StartedAt:  time.Now(),
		}
		_ = repo.CreateExecution(ctx, execution)

		completedAt := time.Now()
		err := repo.UpdateExecutionStatus(ctx, execution.ID, database.ExecutionStatusCompleted, nil, &completedAt, time.Now())
		if err != nil {
			t.Fatalf("UpdateExecutionStatus failed: %v", err)
		}

		got, _ := repo.GetExecution(ctx, execution.ID)
		if got.Status != database.ExecutionStatusCompleted {
			t.Fatalf("Expected status %s, got %s", database.ExecutionStatusCompleted, got.Status)
		}
	})

	t.Run("list executions by status", func(t *testing.T) {
		executions, err := repo.ListExecutionsByStatus(ctx, database.ExecutionStatusCompleted, 10, 0)
		if err != nil {
			t.Fatalf("ListExecutionsByStatus failed: %v", err)
		}
		if len(executions) != 1 {
			t.Fatalf("Expected 1 execution, got %d", len(executions))
		}
	})
}

func TestMockRepository_ScheduleOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	workflowID := uuid.New()

	t.Run("create and get schedule", func(t *testing.T) {
		schedule := &database.ScheduleIndex{
			WorkflowID:     workflowID,
			Name:           "daily-run",
			CronExpression: "0 0 * * *",
			IsActive:       true,
		}
		if err := repo.CreateSchedule(ctx, schedule); err != nil {
			t.Fatalf("CreateSchedule failed: %v", err)
		}
		if schedule.Timezone != "UTC" {
			t.Fatalf("Expected default timezone UTC, got %s", schedule.Timezone)
		}

		got, err := repo.GetSchedule(ctx, schedule.ID)
		if err != nil {
			t.Fatalf("GetSchedule failed: %v", err)
		}
		if got.Name != schedule.Name {
			t.Fatalf("Expected name %s, got %s", schedule.Name, got.Name)
		}
	})

	t.Run("get active schedules due", func(t *testing.T) {
		nextRun := time.Now().Add(-1 * time.Hour) // Past time
		schedule := &database.ScheduleIndex{
			WorkflowID:     workflowID,
			Name:           "due-schedule",
			CronExpression: "*/5 * * * *",
			IsActive:       true,
			NextRunAt:      &nextRun,
		}
		_ = repo.CreateSchedule(ctx, schedule)

		due, err := repo.GetActiveSchedulesDue(ctx, time.Now())
		if err != nil {
			t.Fatalf("GetActiveSchedulesDue failed: %v", err)
		}
		if len(due) != 1 {
			t.Fatalf("Expected 1 due schedule, got %d", len(due))
		}
	})

	t.Run("list schedules active only", func(t *testing.T) {
		inactiveSchedule := &database.ScheduleIndex{
			WorkflowID:     workflowID,
			Name:           "inactive",
			CronExpression: "0 0 * * *",
			IsActive:       false,
		}
		_ = repo.CreateSchedule(ctx, inactiveSchedule)

		schedules, err := repo.ListSchedules(ctx, nil, true, 10, 0)
		if err != nil {
			t.Fatalf("ListSchedules failed: %v", err)
		}
		for _, s := range schedules {
			if !s.IsActive {
				t.Fatal("Expected only active schedules")
			}
		}
	})
}

func TestMockRepository_SettingsOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	t.Run("set and get setting", func(t *testing.T) {
		if err := repo.SetSetting(ctx, "theme", "dark"); err != nil {
			t.Fatalf("SetSetting failed: %v", err)
		}

		got, err := repo.GetSetting(ctx, "theme")
		if err != nil {
			t.Fatalf("GetSetting failed: %v", err)
		}
		if got != "dark" {
			t.Fatalf("Expected dark, got %s", got)
		}
	})

	t.Run("setting not found", func(t *testing.T) {
		_, err := repo.GetSetting(ctx, "nonexistent")
		if err != database.ErrNotFound {
			t.Fatalf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("delete setting", func(t *testing.T) {
		_ = repo.SetSetting(ctx, "to-delete", "value")
		if err := repo.DeleteSetting(ctx, "to-delete"); err != nil {
			t.Fatalf("DeleteSetting failed: %v", err)
		}

		_, err := repo.GetSetting(ctx, "to-delete")
		if err != database.ErrNotFound {
			t.Fatalf("Expected ErrNotFound after delete, got %v", err)
		}
	})
}

func TestMockRepository_ExportOperations(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	executionID := uuid.New()
	workflowID := uuid.New()

	repo.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, WorkflowID: workflowID, StartedAt: time.Now()})

	t.Run("create and get export", func(t *testing.T) {
		export := &database.ExportIndex{
			ExecutionID: executionID,
			WorkflowID:  &workflowID,
			Name:        "test-export",
			Format:      "mp4",
		}
		if err := repo.CreateExport(ctx, export); err != nil {
			t.Fatalf("CreateExport failed: %v", err)
		}
		if export.Status != "pending" {
			t.Fatalf("Expected default status pending, got %s", export.Status)
		}

		got, err := repo.GetExport(ctx, export.ID)
		if err != nil {
			t.Fatalf("GetExport failed: %v", err)
		}
		if got.Name != export.Name {
			t.Fatalf("Expected name %s, got %s", export.Name, got.Name)
		}
		if got.WorkflowName != "test-workflow" {
			t.Fatalf("Expected joined workflow name test-workflow, got %s", got.WorkflowName)
		}
	})

	t.Run("list exports by execution", func(t *testing.T) {
		exports, err := repo.ListExportsByExecution(ctx, executionID)
		if err != nil {
			t.Fatalf("ListExportsByExecution failed: %v", err)
		}
		if len(exports) != 1 {
			t.Fatalf("Expected 1 export, got %d", len(exports))
		}
	})

	t.Run("list exports by workflow", func(t *testing.T) {
		exports, err := repo.ListExportsByWorkflow(ctx, workflowID, 10, 0)
		if err != nil {
			t.Fatalf("ListExportsByWorkflow failed: %v", err)
		}
		if len(exports) != 1 {
			t.Fatalf("Expected 1 export, got %d", len(exports))
		}
	})
}

func TestMockRepository_Pagination(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Create 5 projects
	for i := 0; i < 5; i++ {
		_ = repo.CreateProject(ctx, &database.ProjectIndex{
			Name:       "project-" + string(rune('a'+i)),
			FolderPath: "/projects/" + string(rune('a'+i)),
		})
		// Small delay to ensure different timestamps
		time.Sleep(time.Millisecond)
	}

	t.Run("limit", func(t *testing.T) {
		projects, _ := repo.ListProjects(ctx, 2, 0)
		if len(projects) != 2 {
			t.Fatalf("Expected 2 projects with limit, got %d", len(projects))
		}
	})

	t.Run("offset", func(t *testing.T) {
		projects, _ := repo.ListProjects(ctx, 10, 3)
		if len(projects) != 2 {
			t.Fatalf("Expected 2 projects with offset 3, got %d", len(projects))
		}
	})

	t.Run("offset beyond range", func(t *testing.T) {
		projects, _ := repo.ListProjects(ctx, 10, 10)
		if len(projects) != 0 {
			t.Fatalf("Expected 0 projects with offset beyond range, got %d", len(projects))
		}
	})
}

func TestMockRepository_Reset(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	_ = repo.CreateProject(ctx, &database.ProjectIndex{Name: "test", FolderPath: "/test"})
	_ = repo.SetSetting(ctx, "key", "value")

	repo.Reset()

	projects, _ := repo.ListProjects(ctx, 10, 0)
	if len(projects) != 0 {
		t.Fatalf("Expected 0 projects after reset, got %d", len(projects))
	}

	_, err := repo.GetSetting(ctx, "key")
	if err != database.ErrNotFound {
		t.Fatalf("Expected ErrNotFound after reset, got %v", err)
	}
}
