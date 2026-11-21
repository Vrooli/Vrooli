package mocks

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

func TestImportDirectoryScannerModelsFilesAndDirectories(t *testing.T) {
	scanner := NewImportDirectoryScanner()
	scanner.PutFile("/project/workflows/a.workflow.json", []byte("workflow"))
	scanner.PutDirectory("/project/workflows", []shared.FileEntry{
		{Name: "a.workflow.json", Path: "/project/workflows/a.workflow.json"},
	})

	content, err := scanner.ReadFile(context.Background(), "/project/workflows/a.workflow.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "workflow" {
		t.Fatalf("ReadFile content = %q", content)
	}

	exists, err := scanner.Exists(context.Background(), "/project/workflows")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("directory should exist")
	}

	isDir, err := scanner.IsDir(context.Background(), "/project/workflows")
	if err != nil {
		t.Fatalf("IsDir: %v", err)
	}
	if !isDir {
		t.Fatal("path should be a directory")
	}
}

func TestImportDirectoryScannerMissingFile(t *testing.T) {
	scanner := NewImportDirectoryScanner()
	if _, err := scanner.ReadFile(context.Background(), "/missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadFile error = %v, want os.ErrNotExist", err)
	}
}

func TestImportIndexersStoreAndLookupData(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	projecter := NewImportProjectIndexer()
	indexer := NewImportWorkflowIndexer()

	projecter.PutProject(&shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/project",
	})
	workflow := &shared.WorkflowIndexData{
		ID:        workflowID,
		ProjectID: projectID,
		Name:      "Test Workflow",
		FilePath:  "workflows/test.workflow.json",
	}
	if err := indexer.CreateWorkflowIndex(context.Background(), projectID, workflow); err != nil {
		t.Fatalf("CreateWorkflowIndex: %v", err)
	}

	gotProject, err := projecter.GetProjectByFolderPath(context.Background(), "/project")
	if err != nil {
		t.Fatalf("GetProjectByFolderPath: %v", err)
	}
	if gotProject.ID != projectID {
		t.Fatalf("project ID = %s, want %s", gotProject.ID, projectID)
	}

	gotWorkflow, err := indexer.GetWorkflowByFilePath(context.Background(), projectID, "workflows/test.workflow.json")
	if err != nil {
		t.Fatalf("GetWorkflowByFilePath: %v", err)
	}
	if gotWorkflow.ID != workflowID {
		t.Fatalf("workflow ID = %s, want %s", gotWorkflow.ID, workflowID)
	}
}
