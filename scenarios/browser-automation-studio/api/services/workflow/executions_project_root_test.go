package workflow

import (
	"context"
	"errors"
	"strings"
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

func TestValidateProjectRootRejectsRelativePaths(t *testing.T) {
	t.Parallel()

	if err := validateProjectRoot(""); err != nil {
		t.Fatalf("validateProjectRoot(\"\") error = %v, want nil", err)
	}
	if err := validateProjectRoot("/workspace/scenarios/audio-tools/bas"); err != nil {
		t.Fatalf("validateProjectRoot(absolute) error = %v, want nil", err)
	}
	// A relative root resolves against the server's working directory, not the
	// caller's, and silently selects the wrong selector manifest.
	err := validateProjectRoot("../../../scenarios/audio-tools/bas")
	if err == nil {
		t.Fatal("validateProjectRoot(relative) error = nil, want rejection")
	}
	if got := err.Error(); !strings.Contains(got, "absolute") {
		t.Fatalf("validateProjectRoot(relative) error = %q, want mention of absolute-path requirement", got)
	}
}
