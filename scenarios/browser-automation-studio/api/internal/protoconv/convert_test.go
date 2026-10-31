package protoconv

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basexports "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func containsJSONField(data []byte, field string) bool {
	return bytes.Contains(data, []byte(`"`+field+`"`))
}

func TestExecutionExportPreviewUsesGeneratedReplaySpec(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()
	preview := &workflow.ExecutionExportPreview{
		ExecutionID: executionID,
		SpecID:      uuid.NewString(),
		Status:      "ready",
		Package: &basexports.ReplaySpec{
			Version:     "2025-11-07",
			GeneratedAt: timestamppb.New(time.Date(2026, 9, 15, 12, 30, 0, 0, time.UTC)),
			Execution: &basexports.ReplayExecutionMetadata{
				ExecutionId: executionID.String(),
				WorkflowId:  workflowID.String(),
				Status:      "completed",
				StartedAt:   timestamppb.New(time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)),
			},
			Frames: []*basexports.ReplayFrame{},
			Assets: []*basexports.ReplayAsset{},
		},
	}

	got, err := ExecutionExportPreviewToProto(preview)
	if err != nil {
		t.Fatalf("convert replay preview: %v", err)
	}
	if got.GetPackage().GetVersion() != preview.Package.Version {
		t.Fatalf("package version = %q", got.GetPackage().GetVersion())
	}
	if got.GetPackage().GetExecution().GetExecutionId() != executionID.String() {
		t.Fatalf("package execution id = %q", got.GetPackage().GetExecution().GetExecutionId())
	}
}

func TestExecutionToProto(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(2 * time.Minute)

	exec := &database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Status:     database.ExecutionStatusRunning,
		StartedAt:  now,
		CompletedAt: func() *time.Time {
			return &completed
		}(),
		CreatedAt:    now.Add(-time.Minute),
		UpdatedAt:    now,
		ErrorMessage: "boom",
	}

	pb, err := ExecutionToProto(exec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pb.GetExecutionId() != exec.ID.String() {
		t.Fatalf("expected id %s, got %s", exec.ID, pb.GetExecutionId())
	}
	if pb.GetWorkflowId() != exec.WorkflowID.String() {
		t.Fatalf("expected workflow_id %s, got %s", exec.WorkflowID, pb.GetWorkflowId())
	}
	if pb.Error == nil || pb.GetError() != "boom" {
		t.Fatalf("expected error \"boom\", got %v", pb.Error)
	}

	// Ensure JSON uses proto field names (snake_case).
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !containsJSONField(data, "workflow_id") {
		t.Fatalf("expected workflow_id field in marshalled JSON: %s", string(data))
	}
}

// [REQ:BAS-RH-J24] Malformed receipts must not become successful public responses.
func TestRecordingReceiptValidation(t *testing.T) {
	for _, stamp := range []string{"", "invalid", "2026-09-22T12:00:00+25:00"} {
		if _, err := StartRecordingToProto(&driver.StartRecordingResponse{SessionID: "s", RecordingID: "r", StartedAt: stamp}); err == nil {
			t.Errorf("accepted start timestamp %q", stamp)
		}
		if _, err := StopRecordingToProto(&driver.StopRecordingResponse{SessionID: "s", RecordingID: "r", StoppedAt: stamp}); err == nil {
			t.Errorf("accepted stop timestamp %q", stamp)
		}
	}
	if _, err := StartRecordingToProto(nil); err == nil {
		t.Error("accepted nil start")
	}
	if _, err := StopRecordingToProto(nil); err == nil {
		t.Error("accepted nil stop")
	}
	if _, err := RecordingStatusToProto(nil); err == nil {
		t.Error("accepted nil status")
	}
	for _, count := range []int{-1, 2147483648} {
		if _, err := StopRecordingToProto(&driver.StopRecordingResponse{SessionID: "s", RecordingID: "r", StoppedAt: "2026-09-22T12:00:00Z", ActionCount: count}); err == nil {
			t.Errorf("accepted stop count %d", count)
		}
		if _, err := RecordingStatusToProto(&driver.RecordingStatusResponse{SessionID: "s", ActionCount: count}); err == nil {
			t.Errorf("accepted status count %d", count)
		}
	}
	idle, err := RecordingStatusToProto(&driver.RecordingStatusResponse{SessionID: "s"})
	if err != nil || idle.GetStartedAt() != nil || idle.GetIsRecording() {
		t.Fatalf("idle receipt not preserved: %v, %v", idle, err)
	}
}
