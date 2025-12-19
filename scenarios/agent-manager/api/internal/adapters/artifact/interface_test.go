package artifact_test

import (
	"testing"

	"agent-manager/internal/adapters/artifact"

	"github.com/google/uuid"
)

// [REQ:REQ-P1-008] Tests for artifact collector interface definitions

// TestInterfaceTypes verifies that the package compiles and types are accessible.
// This is a minimal test to satisfy test coverage requirements for interface-only packages.
func TestInterfaceTypes(t *testing.T) {
	// Verify types are defined and accessible
	var _ artifact.ArtifactType

	// Verify type constants
	types := []artifact.ArtifactType{
		artifact.ArtifactTypeDiff,
		artifact.ArtifactTypeLog,
		artifact.ArtifactTypeScreenshot,
		artifact.ArtifactTypeValidation,
		artifact.ArtifactTypeSummary,
	}

	for _, typ := range types {
		if typ == "" {
			t.Error("artifact type should not be empty")
		}
	}
}

func TestArtifact_Fields(t *testing.T) {
	// Verify Artifact struct can be instantiated
	testID := uuid.New()
	runID := uuid.New()
	a := &artifact.Artifact{
		ID:          testID,
		RunID:       runID,
		Type:        artifact.ArtifactTypeDiff,
		Name:        "changes.diff",
		ContentSize: 1024,
		Checksum:    "sha256:abc123",
	}

	if a.ID != testID {
		t.Errorf("expected ID %s, got '%s'", testID, a.ID)
	}
	if a.Type != artifact.ArtifactTypeDiff {
		t.Errorf("expected type 'diff', got '%s'", a.Type)
	}
}

func TestStoreRequest_Fields(t *testing.T) {
	// Verify StoreRequest struct
	runID := uuid.New()
	req := &artifact.StoreRequest{
		RunID: runID,
		Type:  artifact.ArtifactTypeLog,
		Name:  "execution.log",
	}

	if req.Type != artifact.ArtifactTypeLog {
		t.Errorf("expected type 'log', got '%s'", req.Type)
	}
}

func TestListOptions_Fields(t *testing.T) {
	// Verify ListOptions struct
	screenshotType := artifact.ArtifactTypeScreenshot
	opts := &artifact.ListOptions{
		Type:  &screenshotType,
		Limit: 10,
	}

	if opts.Limit != 10 {
		t.Errorf("expected limit 10, got %d", opts.Limit)
	}
}

func TestValidationType_Constants(t *testing.T) {
	// Verify validation type constants
	validationTypes := []artifact.ValidationType{
		artifact.ValidationTypeTypeCheck,
		artifact.ValidationTypeLint,
		artifact.ValidationTypeTest,
		artifact.ValidationTypeSecurity,
	}

	for _, vt := range validationTypes {
		if vt == "" {
			t.Error("validation type should not be empty")
		}
	}
}
