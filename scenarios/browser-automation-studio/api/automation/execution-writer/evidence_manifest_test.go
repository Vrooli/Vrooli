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
	writer := NewFileWriter(nil, nil, nil, dir)
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: path, ContentType: "application/zip"}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(writer.evidenceManifestFilePath(plan.ExecutionID))
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
