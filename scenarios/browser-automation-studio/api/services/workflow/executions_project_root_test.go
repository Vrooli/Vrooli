package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

type executionProjectCatalogStub struct {
	workflow *database.WorkflowIndex
	project  *database.ProjectIndex
	err      error
}

func (s executionProjectCatalogStub) GetWorkflow(context.Context, uuid.UUID) (*database.WorkflowIndex, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.workflow, nil
}

func (s executionProjectCatalogStub) GetProject(context.Context, uuid.UUID) (*database.ProjectIndex, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.project, nil
}

func TestExecutionProjectRootUsesPersistedProjectFolder(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	root, err := executionProjectRoot(context.Background(), executionProjectCatalogStub{
		workflow: &database.WorkflowIndex{ID: uuid.New(), ProjectID: &projectID},
		project:  &database.ProjectIndex{ID: projectID, FolderPath: " /workspace/audio-tools "},
	}, uuid.New())
	if err != nil {
		t.Fatalf("executionProjectRoot() error = %v", err)
	}
	if root != "/workspace/audio-tools" {
		t.Fatalf("executionProjectRoot() = %q, want project folder", root)
	}
}

func TestExecutionProjectRootAllowsProjectlessWorkflows(t *testing.T) {
	t.Parallel()

	root, err := executionProjectRoot(context.Background(), executionProjectCatalogStub{
		workflow: &database.WorkflowIndex{ID: uuid.New()},
	}, uuid.New())
	if err != nil {
		t.Fatalf("executionProjectRoot() error = %v", err)
	}
	if root != "" {
		t.Fatalf("executionProjectRoot() = %q, want empty root", root)
	}
}

func TestExecutionProjectRootPropagatesLookupFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("catalog unavailable")
	_, err := executionProjectRoot(context.Background(), executionProjectCatalogStub{err: want}, uuid.New())
	if !errors.Is(err, want) {
		t.Fatalf("executionProjectRoot() error = %v, want %v", err, want)
	}
}
