package executionwriter

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestRecordExecutionArtifactsPublishesSanitizedHarMetadata(t *testing.T) {
	dir := t.TempDir()
	harPath := filepath.Join(dir, "capture.har")
	if err := os.WriteFile(harPath, []byte(`{"log":{"entries":[{"request":{"url":"https://example.test/?token=secret","headers":[{"name":"Authorization","value":"Bearer secret"}]}}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "har", Path: harPath, ContentType: "application/json"}}); err != nil {
		t.Fatal(err)
	}
	result := writer.getOrCreateResult(plan)
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifacts = %d", len(result.Artifacts))
	}
	artifact := result.Artifacts[0]
	if _, ok := artifact.Payload["path"]; ok {
		t.Fatal("local path leaked into artifact payload")
	}
	if _, ok := artifact.Payload["source_path"]; ok {
		t.Fatal("source path leaked into artifact payload")
	}
	if artifact.AccessPolicy != "ACCESS_POLICY_PROTECTED_STORAGE_ONLY" || !artifact.Redacted || len(artifact.SHA256) != 64 {
		t.Fatalf("unexpected protected HAR metadata: %#v", artifact)
	}
	encoded, _ := artifact.Payload["sanitized_base64"].(string)
	if strings.Contains(encoded, "secret") || encoded == "" {
		t.Fatalf("HAR derivative missing or unsafe: %q", encoded)
	}
	sanitized, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode sanitized HAR: %v", err)
	}
	digest := sha256.Sum256(sanitized)
	if artifact.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("HAR digest does not match published sanitized bytes: got %s", artifact.SHA256)
	}
}

func TestRecordExecutionArtifactsRedactsSourcePathPayload(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "capture.zip")
	if err := os.WriteFile(tracePath, []byte("trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: tracePath, ContentType: "application/zip", Payload: map[string]any{"source_path": "/private/capture.zip"}}}); err != nil {
		t.Fatal(err)
	}
	if _, leaked := writer.getOrCreateResult(plan).Artifacts[0].Payload["source_path"]; leaked {
		t.Fatal("source path leaked into execution artifact")
	}
}
