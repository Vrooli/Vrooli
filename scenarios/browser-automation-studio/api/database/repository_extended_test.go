package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TestGetProjectByName tests project retrieval by name [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestGetProjectByName(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] retrieves project by name", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "FindMe Project",
			FolderPath: "/test/findme",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		retrieved, err := repo.GetProjectByName(ctx, "FindMe Project")
		if err != nil {
			t.Fatalf("Failed to get project by name: %v", err)
		}

		if retrieved.ID != project.ID {
			t.Errorf("Expected ID %s, got %s", project.ID, retrieved.ID)
		}

		if retrieved.FolderPath != project.FolderPath {
			t.Errorf("Expected folder path %s, got %s", project.FolderPath, retrieved.FolderPath)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for non-existent project name", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		_, err := repo.GetProjectByName(ctx, "NonExistentProject")
		if err == nil {
			t.Error("Expected error for non-existent project name")
		}
		// Check error contains expected text since not all methods wrap with ErrNotFound
		if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, sql.ErrNoRows) {
			t.Logf("Got error: %v", err)
		}
	})
}

// TestGetProjectByFolderPath tests project retrieval by folder path [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestGetProjectByFolderPath(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] retrieves project by folder path", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "Path Project",
			FolderPath: "/test/my-unique-path",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		retrieved, err := repo.GetProjectByFolderPath(ctx, "/test/my-unique-path")
		if err != nil {
			t.Fatalf("Failed to get project by folder path: %v", err)
		}

		if retrieved.ID != project.ID {
			t.Errorf("Expected ID %s, got %s", project.ID, retrieved.ID)
		}

		if retrieved.Name != project.Name {
			t.Errorf("Expected name %s, got %s", project.Name, retrieved.Name)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for non-existent folder path", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		_, err := repo.GetProjectByFolderPath(ctx, "/test/non-existent-path")
		if err == nil {
			t.Error("Expected error for non-existent folder path")
		}
		// Check error indicates not found (may be wrapped)
		if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, sql.ErrNoRows) {
			t.Logf("Got error: %v", err)
		}
	})
}

// TestGetProjectStats tests project statistics retrieval [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestGetProjectStats(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns project statistics", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "Stats Project",
			FolderPath: "/test/stats-project",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Create workflows for this project
		for i := 0; i < 3; i++ {
			workflow := &Workflow{
				ID:         uuid.New(),
				ProjectID:  &project.ID,
				Name:       "Workflow " + string(rune(i+'A')),
				FolderPath: "/test/stats-project/workflows",
				FlowDefinition: JSONMap{
					"nodes": []any{},
				},
				Tags:    []string{},
				Version: 1,
			}
			if err := repo.CreateWorkflow(ctx, workflow); err != nil {
				t.Fatalf("Failed to create workflow: %v", err)
			}
		}

		stats, err := repo.GetProjectStats(ctx, project.ID)
		if err != nil {
			t.Fatalf("Failed to get project stats: %v", err)
		}

		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}

		// Verify workflow count (can be int or int64 depending on driver)
		workflowCount, ok := stats["workflow_count"]
		if !ok {
			t.Fatal("Expected workflow_count in stats")
		}

		// Handle both int and int64
		var count int
		switch v := workflowCount.(type) {
		case int:
			count = v
		case int64:
			count = int(v)
		default:
			t.Fatalf("Unexpected type for workflow_count: %T", workflowCount)
		}

		if count != 3 {
			t.Errorf("Expected workflow count 3, got %d", count)
		}
	})
}

// TestDeleteProjectWorkflows tests bulk workflow deletion [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestDeleteProjectWorkflows(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] deletes multiple workflows in a project", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "Delete Workflows Project",
			FolderPath: "/test/delete-workflows",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Create workflows
		workflowIDs := make([]uuid.UUID, 3)
		for i := 0; i < 3; i++ {
			workflow := &Workflow{
				ID:         uuid.New(),
				ProjectID:  &project.ID,
				Name:       "Delete Workflow " + string(rune(i+'A')),
				FolderPath: "/test/delete-workflows/wf",
				FlowDefinition: JSONMap{
					"nodes": []any{},
				},
				Tags:    []string{},
				Version: 1,
			}
			if err := repo.CreateWorkflow(ctx, workflow); err != nil {
				t.Fatalf("Failed to create workflow: %v", err)
			}
			workflowIDs[i] = workflow.ID
		}

		// Delete workflows
		err := repo.DeleteProjectWorkflows(ctx, project.ID, workflowIDs)
		if err != nil {
			t.Fatalf("Failed to delete project workflows: %v", err)
		}

		// Verify deletion
		for _, id := range workflowIDs {
			_, err := repo.GetWorkflow(ctx, id)
			if err == nil {
				t.Errorf("Expected workflow %s to be deleted", id)
			}
		}
	})
}

// TestGetWorkflowByProjectAndName tests workflow retrieval by project and name [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestGetWorkflowByProjectAndName(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] retrieves workflow by project ID and name", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "Workflow Lookup Project",
			FolderPath: "/test/wf-lookup",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		workflow := &Workflow{
			ID:         uuid.New(),
			ProjectID:  &project.ID,
			Name:       "Unique Workflow Name",
			FolderPath: "/test/wf-lookup/workflows",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		retrieved, err := repo.GetWorkflowByProjectAndName(ctx, project.ID, "Unique Workflow Name")
		if err != nil {
			t.Fatalf("Failed to get workflow by project and name: %v", err)
		}

		if retrieved.ID != workflow.ID {
			t.Errorf("Expected ID %s, got %s", workflow.ID, retrieved.ID)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for non-existent workflow", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		project := &Project{
			ID:         uuid.New(),
			Name:       "Empty Project",
			FolderPath: "/test/empty-project",
		}

		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		_, err := repo.GetWorkflowByProjectAndName(ctx, project.ID, "NonExistentWorkflow")
		if err == nil {
			t.Error("Expected error for non-existent workflow")
		}
		// Check error indicates not found (may be wrapped)
		if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, sql.ErrNoRows) {
			t.Logf("Got error: %v", err)
		}
	})
}

// TestWorkflowVersionOperations tests workflow version management [REQ:BAS-VERSION-AUTOSAVE] [REQ:BAS-VERSION-RESTORE]
func TestWorkflowVersionOperations(t *testing.T) {
	t.Run("[REQ:BAS-VERSION-AUTOSAVE] creates workflow version", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Versioned Workflow",
			FolderPath: "/test/versioned-workflow",
			FlowDefinition: JSONMap{
				"nodes": []any{
					map[string]any{
						"id":   "node-1",
						"type": "navigate",
					},
				},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		version := &WorkflowVersion{
			WorkflowID: workflow.ID,
			Version:    1,
			FlowDefinition: JSONMap{
				"nodes": []any{
					map[string]any{
						"id":   "node-1",
						"type": "navigate",
					},
				},
			},
			CreatedAt: time.Now(),
		}

		err := repo.CreateWorkflowVersion(ctx, version)
		if err != nil {
			t.Fatalf("Failed to create workflow version: %v", err)
		}
	})

	t.Run("[REQ:BAS-VERSION-RESTORE] retrieves workflow version", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Retrieve Version Workflow",
			FolderPath: "/test/retrieve-version",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		version := &WorkflowVersion{
			WorkflowID:        workflow.ID,
			Version:           1,
			ChangeDescription: "Initial version",
			FlowDefinition: JSONMap{
				"nodes": []any{
					map[string]any{
						"id": "original-node",
					},
				},
			},
			CreatedAt: time.Now(),
		}

		if err := repo.CreateWorkflowVersion(ctx, version); err != nil {
			t.Fatalf("Failed to create workflow version: %v", err)
		}

		retrieved, err := repo.GetWorkflowVersion(ctx, workflow.ID, 1)
		if err != nil {
			t.Fatalf("Failed to get workflow version: %v", err)
		}

		if retrieved.WorkflowID != workflow.ID {
			t.Errorf("Expected workflow ID %s, got %s", workflow.ID, retrieved.WorkflowID)
		}

		if retrieved.Version != 1 {
			t.Errorf("Expected version 1, got %d", retrieved.Version)
		}

		if retrieved.ChangeDescription != "Initial version" {
			t.Errorf("Expected change description 'Initial version', got %s", retrieved.ChangeDescription)
		}
	})

	t.Run("[REQ:BAS-VERSION-RESTORE] lists workflow versions", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Multi Version Workflow",
			FolderPath: "/test/multi-version",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 3,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		// Create multiple versions
		for v := 1; v <= 3; v++ {
			version := &WorkflowVersion{
				WorkflowID: workflow.ID,
				Version:    v,
				FlowDefinition: JSONMap{
					"nodes": []any{
						map[string]any{
							"id": "node-v" + string(rune(v+'0')),
						},
					},
				},
				CreatedAt: time.Now(),
			}

			if err := repo.CreateWorkflowVersion(ctx, version); err != nil {
				t.Fatalf("Failed to create workflow version %d: %v", v, err)
			}
		}

		versions, err := repo.ListWorkflowVersions(ctx, workflow.ID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list workflow versions: %v", err)
		}

		if len(versions) != 3 {
			t.Errorf("Expected 3 versions, got %d", len(versions))
		}

		// Verify versions are returned in descending order (newest first)
		if len(versions) > 1 && versions[0].Version < versions[1].Version {
			t.Error("Expected versions to be returned in descending order")
		}
	})
}

// TestScreenshotOperations tests screenshot persistence [REQ:BAS-EXEC-TELEMETRY-STREAM]
func TestScreenshotOperations(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] creates and retrieves screenshots", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		// Create workflow and execution
		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Screenshot Workflow",
			FolderPath: "/test/screenshots",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		execution := &Execution{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "running",
			StartedAt:  time.Now(),
		}

		if err := repo.CreateExecution(ctx, execution); err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}

		// Create screenshots
		for i := 0; i < 3; i++ {
			screenshot := &Screenshot{
				ID:          uuid.New(),
				ExecutionID: execution.ID,
				StepName:    "step-" + string(rune(i+'0')),
				StorageURL:  "s3://test-bucket/screenshots/" + string(rune(i+'0')) + ".png",
				Width:       1920,
				Height:      1080,
				Metadata: JSONMap{
					"timestamp": time.Now().Unix(),
				},
				Timestamp: time.Now(),
			}

			err := repo.CreateScreenshot(ctx, screenshot)
			if err != nil {
				t.Fatalf("Failed to create screenshot: %v", err)
			}
		}

		// Retrieve screenshots
		screenshots, err := repo.GetExecutionScreenshots(ctx, execution.ID)
		if err != nil {
			t.Fatalf("Failed to get execution screenshots: %v", err)
		}

		if len(screenshots) != 3 {
			t.Errorf("Expected 3 screenshots, got %d", len(screenshots))
		}

		// Verify screenshot data
		if screenshots[0].Width != 1920 {
			t.Errorf("Expected width 1920, got %d", screenshots[0].Width)
		}

		if screenshots[0].StepName == "" {
			t.Error("Expected step name to be set")
		}
	})
}

// TestExecutionLogOperations tests execution log persistence [REQ:BAS-EXEC-TELEMETRY-STREAM]
func TestExecutionLogOperations(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] creates and retrieves execution logs", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		// Create workflow and execution
		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Log Workflow",
			FolderPath: "/test/logs",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		execution := &Execution{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "running",
			StartedAt:  time.Now(),
		}

		if err := repo.CreateExecution(ctx, execution); err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}

		// Create execution logs
		logLevels := []string{"info", "warning", "error"}
		for i, level := range logLevels {
			execLog := &ExecutionLog{
				ID:          uuid.New(),
				ExecutionID: execution.ID,
				StepName:    "step-" + string(rune(i+'0')),
				Level:       level,
				Message:     "Test log message " + level,
				Timestamp:   time.Now(),
			}

			err := repo.CreateExecutionLog(ctx, execLog)
			if err != nil {
				t.Fatalf("Failed to create execution log: %v", err)
			}
		}

		// Retrieve logs
		logs, err := repo.GetExecutionLogs(ctx, execution.ID)
		if err != nil {
			t.Fatalf("Failed to get execution logs: %v", err)
		}

		if len(logs) != 3 {
			t.Errorf("Expected 3 logs, got %d", len(logs))
		}

		// Verify log levels
		levelMap := make(map[string]bool)
		for _, l := range logs {
			levelMap[l.Level] = true
		}

		for _, expectedLevel := range logLevels {
			if !levelMap[expectedLevel] {
				t.Errorf("Expected log level %s not found", expectedLevel)
			}
		}
	})
}

// TestExtractedDataOperations tests extracted data persistence [REQ:BAS-EXEC-TELEMETRY-STREAM]
func TestExtractedDataOperations(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] creates and retrieves extracted data", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		// Create workflow and execution
		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Extract Workflow",
			FolderPath: "/test/extract",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		execution := &Execution{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "running",
			StartedAt:  time.Now(),
		}

		if err := repo.CreateExecution(ctx, execution); err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}

		// Create extracted data
		data := &ExtractedData{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			StepName:    "step-0",
			DataKey:     "headline",
			DataValue: JSONMap{
				"value": "Extracted headline text",
			},
			DataType:  "text",
			Timestamp: time.Now(),
		}

		err := repo.CreateExtractedData(ctx, data)
		if err != nil {
			t.Fatalf("Failed to create extracted data: %v", err)
		}

		// Retrieve extracted data
		extractedData, err := repo.GetExecutionExtractedData(ctx, execution.ID)
		if err != nil {
			t.Fatalf("Failed to get execution extracted data: %v", err)
		}

		if len(extractedData) != 1 {
			t.Errorf("Expected 1 extracted data entry, got %d", len(extractedData))
		}

		if extractedData[0].DataKey != "headline" {
			t.Errorf("Expected data key 'headline', got %s", extractedData[0].DataKey)
		}

		if value, ok := extractedData[0].DataValue["value"].(string); !ok || value != "Extracted headline text" {
			t.Errorf("Expected extracted value 'Extracted headline text', got %v", extractedData[0].DataValue["value"])
		}
	})
}

// TestFolderOperations tests workflow folder operations [REQ:BAS-WORKFLOW-PERSIST-CRUD]
func TestFolderOperations(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] creates and retrieves folders", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		folderPath := fmt.Sprintf("/test/my-folder-%s", uuid.NewString())
		folder := &WorkflowFolder{
			Path:      folderPath,
			Name:      "My Folder",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.CreateFolder(ctx, folder)
		if err != nil {
			t.Fatalf("Failed to create folder: %v", err)
		}

		retrieved, err := repo.GetFolder(ctx, folderPath)
		if err != nil {
			t.Fatalf("Failed to get folder: %v", err)
		}

		if retrieved.Name != "My Folder" {
			t.Errorf("Expected name 'My Folder', got %s", retrieved.Name)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] lists all folders", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		log := logrus.New()
		log.SetLevel(logrus.FatalLevel)

		repo := NewRepository(db, log)
		ctx := context.Background()

		// Create multiple folders
		for i := 0; i < 3; i++ {
			folder := &WorkflowFolder{
				Path:      fmt.Sprintf("/test/folder-%s-%d", uuid.NewString(), i),
				Name:      fmt.Sprintf("Folder %d", i),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := repo.CreateFolder(ctx, folder); err != nil {
				t.Fatalf("Failed to create folder: %v", err)
			}
		}

		folders, err := repo.ListFolders(ctx)
		if err != nil {
			t.Fatalf("Failed to list folders: %v", err)
		}

		if len(folders) < 3 {
			t.Errorf("Expected at least 3 folders, got %d", len(folders))
		}
	})
}
