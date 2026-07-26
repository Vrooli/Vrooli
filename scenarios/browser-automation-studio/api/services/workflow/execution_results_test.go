package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

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
