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
		artifact.ValidationTypeBuild,
	}

	for _, vt := range validationTypes {
		if vt == "" {
			t.Error("validation type should not be empty")
		}
	}
}

func TestArtifactType_AllConstants(t *testing.T) {
	// Verify all artifact type constants are defined
	allTypes := []artifact.ArtifactType{
		artifact.ArtifactTypeDiff,
		artifact.ArtifactTypeLog,
		artifact.ArtifactTypeScreenshot,
		artifact.ArtifactTypeValidation,
		artifact.ArtifactTypeSummary,
		artifact.ArtifactTypeOther,
	}

	for i, typ := range allTypes {
		if typ == "" {
			t.Errorf("artifact type at index %d should not be empty", i)
		}
	}

	// Verify uniqueness
	seen := make(map[artifact.ArtifactType]bool)
	for _, typ := range allTypes {
		if seen[typ] {
			t.Errorf("duplicate artifact type: %s", typ)
		}
		seen[typ] = true
	}
}

func TestValidationResult_Fields(t *testing.T) {
	result := &artifact.ValidationResult{
		Passed: true,
		Results: []artifact.ValidationCheck{
			{
				Type:    artifact.ValidationTypeLint,
				Passed:  true,
				Message: "Linting passed",
			},
		},
	}

	if !result.Passed {
		t.Error("expected Passed to be true")
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestValidationCheck_Fields(t *testing.T) {
	check := &artifact.ValidationCheck{
		Type:     artifact.ValidationTypeTest,
		Passed:   false,
		Message:  "Tests failed",
		Errors:   []string{"test_foo.go:42: assertion failed"},
		Warnings: []string{"deprecation warning in bar.go"},
	}

	if check.Type != artifact.ValidationTypeTest {
		t.Errorf("expected type 'test', got %s", check.Type)
	}
	if check.Passed {
		t.Error("expected Passed to be false")
	}
	if len(check.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(check.Errors))
	}
	if len(check.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(check.Warnings))
	}
}

func TestValidationRequest_Fields(t *testing.T) {
	runID := uuid.New()
	req := &artifact.ValidationRequest{
		RunID:        runID,
		WorkDir:      "/tmp/workspace",
		Types:        []artifact.ValidationType{artifact.ValidationTypeLint, artifact.ValidationTypeTest},
		FilePatterns: []string{"*.go", "**/*.ts"},
	}

	if req.RunID != runID {
		t.Errorf("expected RunID %s, got %s", runID, req.RunID)
	}
	if req.WorkDir != "/tmp/workspace" {
		t.Errorf("expected WorkDir '/tmp/workspace', got %s", req.WorkDir)
	}
	if len(req.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(req.Types))
	}
	if len(req.FilePatterns) != 2 {
		t.Errorf("expected 2 file patterns, got %d", len(req.FilePatterns))
	}
}

func TestStoreRequest_AllFields(t *testing.T) {
	runID := uuid.New()
	req := &artifact.StoreRequest{
		RunID:       runID,
		Type:        artifact.ArtifactTypeDiff,
		Name:        "changes.diff",
		ContentSize: 2048,
		ContentType: "text/x-diff",
		Metadata:    map[string]string{"key": "value"},
	}

	if req.RunID != runID {
		t.Errorf("expected RunID %s, got %s", runID, req.RunID)
	}
	if req.ContentSize != 2048 {
		t.Errorf("expected ContentSize 2048, got %d", req.ContentSize)
	}
	if req.ContentType != "text/x-diff" {
		t.Errorf("expected ContentType 'text/x-diff', got %s", req.ContentType)
	}
	if req.Metadata["key"] != "value" {
		t.Errorf("expected Metadata['key'] = 'value', got %s", req.Metadata["key"])
	}
}

func TestArtifact_AllFields(t *testing.T) {
	testID := uuid.New()
	runID := uuid.New()
	a := &artifact.Artifact{
		ID:          testID,
		RunID:       runID,
		Type:        artifact.ArtifactTypeLog,
		Name:        "execution.log",
		StoragePath: "/artifacts/runs/123/execution.log",
		ContentSize: 4096,
		ContentType: "text/plain",
		Checksum:    "sha256:def456",
		Metadata:    map[string]string{"format": "text"},
	}

	if a.ID != testID {
		t.Errorf("expected ID %s, got %s", testID, a.ID)
	}
	if a.StoragePath != "/artifacts/runs/123/execution.log" {
		t.Errorf("expected StoragePath '/artifacts/runs/123/execution.log', got %s", a.StoragePath)
	}
	if a.ContentType != "text/plain" {
		t.Errorf("expected ContentType 'text/plain', got %s", a.ContentType)
	}
	if a.Metadata["format"] != "text" {
		t.Errorf("expected Metadata['format'] = 'text', got %s", a.Metadata["format"])
	}
}
