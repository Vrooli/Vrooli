package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/database"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

func TestDetachedExecutionContextPreservesTestModeWithoutRequestCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(coredb.WithTestMode(context.Background()))
	cancel()

	ctx := detachedExecutionContext(parent)
	if !coredb.IsTestMode(ctx) {
		t.Fatal("detached execution context lost routed test-mode marker")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("detached execution context inherited request cancellation: %v", err)
	}
}

func TestWithTestModeBrowserHeaderPreservesCallerProfile(t *testing.T) {
	t.Parallel()

	original := &sessionprofilepersistence.BrowserProfile{ExtraHeaders: map[string]string{"X-Existing": "value"}}
	profile := withTestModeBrowserHeader(original)
	if profile.ExtraHeaders["X-Vrooli-Test-Mode"] != "1" {
		t.Fatalf("test-mode header = %q, want 1", profile.ExtraHeaders["X-Vrooli-Test-Mode"])
	}
	if profile.ExtraHeaders["X-Existing"] != "value" {
		t.Fatalf("existing header = %q, want value", profile.ExtraHeaders["X-Existing"])
	}
	if _, ok := original.ExtraHeaders["X-Vrooli-Test-Mode"]; ok {
		t.Fatal("test-mode helper mutated caller profile")
	}
}

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
