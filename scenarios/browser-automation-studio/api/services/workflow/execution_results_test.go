package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestRequiredVideoArtifactContract(t *testing.T) {
	if err := requiredVideoArtifactError(true, nil, nil); err == nil {
		t.Fatal("required video with no artifact must fail")
	}
	if err := requiredVideoArtifactError(true, []ExecutionVideoArtifact{{ArtifactID: "video-1"}}, nil); err != nil {
		t.Fatalf("required video with an artifact failed: %v", err)
	}
	want := errors.New("artifact lookup failed")
	if err := requiredVideoArtifactError(true, nil, want); !errors.Is(err, want) {
		t.Fatalf("artifact lookup error = %v, want wrapped %v", err, want)
	}
	if err := requiredVideoArtifactError(false, nil, nil); err != nil {
		t.Fatalf("optional video must not fail: %v", err)
	}
}

func TestListExecutionArtifactsDoesNotExposeCapturePath(t *testing.T) {
	// enforces invariant: protectedEvidenceHasNoPublicLocation
	root := t.TempDir()
	executionID := uuid.New()
	dir := filepath.Join(root, executionID.String(), "artifacts", "har")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capture.har"), []byte(`{"log":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	service := &WorkflowService{executionDataRoot: root}
	artifacts, err := service.listExecutionArtifacts(context.Background(), executionID, "har")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %d", len(artifacts))
	}
	artifact := artifacts[0]
	if _, ok := artifact.Payload["path"]; ok {
		t.Fatalf("payload leaked local path: %#v", artifact.Payload)
	}
	if artifact.StorageURL != "" || artifact.Payload["access_policy"] != "ACCESS_POLICY_PROTECTED_STORAGE_ONLY" || artifact.Payload["sha256"] == "" {
		t.Fatalf("artifact evidence metadata = %#v", artifact)
	}
}
