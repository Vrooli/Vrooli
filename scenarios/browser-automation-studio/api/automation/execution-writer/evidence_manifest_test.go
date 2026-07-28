package executionwriter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRecordExecutionArtifactsWritesStorageIndependentEvidenceManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.zip")
	if err := os.WriteFile(path, []byte("trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: path, ContentType: "application/zip"}}); err != nil {
		t.Fatal(err)
	}
	manifestPath, err := writer.evidenceManifestFilePath(context.Background(), plan.ExecutionID)
	if err != nil {
		t.Fatalf("evidence manifest path: %v", err)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var pack basevidence.ReplayPackage
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &pack); err != nil {
		t.Fatal(err)
	}
	if pack.GetEvidence().GetArtifacts()[0].GetSha256() == "" || strings.Contains(string(raw), path) {
		t.Fatalf("unsafe or incomplete evidence manifest: %s", raw)
	}
}

func TestWriteEvidenceManifestRejectsArtifactWithoutByteDigest(t *testing.T) {
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(t.TempDir()))
	executionID := uuid.New()
	result := &ExecutionResultData{ExecutionID: executionID.String(), Artifacts: []ArtifactData{{ArtifactID: uuid.NewString(), ArtifactType: "console", Payload: map[string]any{"text": "missing digest"}}}}
	err := writer.writeEvidenceManifest(context.Background(), executionID, result, nil)
	if err == nil || !strings.Contains(err.Error(), "no byte-derived SHA-256") {
		t.Fatalf("expected missing digest error, got %v", err)
	}
}
