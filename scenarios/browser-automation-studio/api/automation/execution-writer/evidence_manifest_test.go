package executionwriter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/storage"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRecordExecutionArtifactsWritesStorageIndependentEvidenceManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.zip")
	if err := os.WriteFile(path, []byte("trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
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
	var inventory purgeInventory
	inventoryRaw, err := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), purgeInventoryFileName))
	if err != nil || json.Unmarshal(inventoryRaw, &inventory) != nil {
		t.Fatalf("read private purge inventory: %v", err)
	}
	if len(inventory.ArtifactBindings) != 1 || inventory.ArtifactBindings[0].Scope != "evidence" || inventory.ArtifactBindings[0].Storage != "external" || inventory.ArtifactBindings[0].Path != "" {
		t.Fatalf("external artifact did not receive a typed storage binding: %+v", inventory.ArtifactBindings)
	}
}

func TestRecordExecutionArtifactsBindsLocalDiskBytesByIdentity(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "trace.zip")
	if err := os.WriteFile(source, []byte("local trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	local, err := storage.NewFileStorage(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, local, nil, NewStaticRoot(root))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: source, ContentType: "application/zip"}}); err != nil {
		t.Fatal(err)
	}
	var inventory purgeInventory
	raw, err := os.ReadFile(filepath.Join(root, plan.ExecutionID.String(), purgeInventoryFileName))
	if err != nil || json.Unmarshal(raw, &inventory) != nil {
		t.Fatalf("read private purge inventory: %v", err)
	}
	if len(inventory.ArtifactBindings) != 1 {
		t.Fatalf("expected one disk-backed binding, got %+v", inventory.ArtifactBindings)
	}
	binding := inventory.ArtifactBindings[0]
	// FileStorage's generic artifact objects live outside the UUID execution
	// directory. The exact purge inventory must mark them external and not imply
	// it will delete them.
	if binding.Scope != "evidence" || binding.Storage != "external" || binding.Path != "" {
		t.Fatalf("out-of-directory local object was misclassified: %+v", binding)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", plan.ExecutionID.String())); err != nil {
		t.Fatalf("expected the storage adapter's separate object namespace: %v", err)
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

func TestPurgeBindingsRejectMissingWriterStorageClass(t *testing.T) {
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(t.TempDir()))
	size := int64(1)
	result := &ExecutionResultData{Artifacts: []ArtifactData{{
		ArtifactID: uuid.NewString(), ArtifactType: "screenshot", SizeBytes: &size, SHA256: strings.Repeat("a", 64),
	}}}
	if _, err := writer.purgeArtifactBindings(context.Background(), uuid.New(), result); err == nil || !strings.Contains(err.Error(), "no writer storage receipt") {
		t.Fatalf("missing storage class did not fail closed: %v", err)
	}
}

func TestEvidenceManifestRemovesTimelineStorageLocatorsAndPayloads(t *testing.T) {
	root := t.TempDir()
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(root))
	executionID, workflowID := uuid.New(), uuid.New()
	artifactID := uuid.NewString()
	path := "/private/execution/artifacts/screenshot.png"
	thumb := "https://private.invalid/private-thumb"
	size := int64(5)
	result := &ExecutionResultData{
		ExecutionID: executionID.String(), WorkflowID: workflowID.String(),
		Artifacts: []ArtifactData{{ArtifactID: artifactID, ArtifactType: "screenshot", SizeBytes: &size, SHA256: strings.Repeat("a", 64)}},
	}
	timeline := &executionTimelineData{pb: &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{{
		Id: uuid.NewString(), Timestamp: timestamppb.Now(),
		Telemetry: &basdomain.ActionTelemetry{
			Screenshot:  &basdomain.TimelineScreenshot{ArtifactId: artifactID, Url: "https://private.invalid/private-shot", ThumbnailUrl: thumb, Path: &path},
			DomSnapshot: &basdomain.TelemetryArtifact{ArtifactId: uuid.NewString(), StorageUrl: "memory://private-bucket/dom-key", Path: &path},
		},
		Aggregates: &bastimeline.TimelineEntryAggregates{Artifacts: []*bastimeline.TimelineArtifact{{
			Id: uuid.NewString(), StorageUrl: "/api/v1/artifacts/private-key", ThumbnailUrl: &thumb,
			Payload: map[string]*commonv1.JsonValue{"storage_object": {Kind: &commonv1.JsonValue_StringValue{StringValue: "private-object-key"}}},
		}}},
	}}}}
	if err := writer.writeEvidenceManifest(context.Background(), executionID, result, timeline); err != nil {
		t.Fatal(err)
	}
	manifestPath, err := writer.evidenceManifestFilePath(context.Background(), executionID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private.invalid", "private/execution", "memory://", "private-key", "private-object-key", "storage_object"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("storage locator leaked into portable evidence manifest (%q): %s", secret, raw)
		}
	}
}
