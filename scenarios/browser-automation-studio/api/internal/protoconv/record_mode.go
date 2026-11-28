package protoconv

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"google.golang.org/protobuf/types/known/timestamppb"

	basrecording "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recording"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// selectorPayloadToProto marshals then unmarshals to align with proto field casing.
func selectorPayloadToProto[T any](in T, out proto.Message) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// recordingTimestamp rejects invalid receipt data instead of emitting untyped success.
func recordingTimestamp(value string) (*timestamppb.Timestamp, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("invalid recording timestamp: %w", err)
	}
	stamp := timestamppb.New(parsed)
	if err := stamp.CheckValid(); err != nil {
		return nil, err
	}
	return stamp, nil
}

func RecordingStatusToProto(status *driver.RecordingStatusResponse) (*basrecording.RecordingStatusResponse, error) {
	if status == nil || status.SessionID == "" || status.ActionCount < 0 || status.ActionCount > math.MaxInt32 {
		return nil, fmt.Errorf("invalid recording status receipt")
	}
	pb := &basrecording.RecordingStatusResponse{
		SessionId: status.SessionID, RecordingId: status.RecordingID,
		IsRecording: status.IsRecording, ActionCount: int32(status.ActionCount),
	}
	if status.StartedAt != "" {
		stamp, err := recordingTimestamp(status.StartedAt)
		if err != nil {
			return nil, err
		}
		pb.StartedAt = stamp
	}
	return pb, nil
}

func RecordingSessionToProto(session any) (*basrecording.CreateRecordingSessionResponse, error) {
	var pb basrecording.CreateRecordingSessionResponse
	if err := selectorPayloadToProto(session, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func StartRecordingToProto(resp *driver.StartRecordingResponse) (*basrecording.StartRecordingResponse, error) {
	if resp == nil || resp.SessionID == "" || resp.RecordingID == "" {
		return nil, fmt.Errorf("invalid recording start receipt")
	}
	stamp, err := recordingTimestamp(resp.StartedAt)
	if err != nil {
		return nil, err
	}
	return &basrecording.StartRecordingResponse{SessionId: resp.SessionID, RecordingId: resp.RecordingID, StartedAt: stamp}, nil
}

func StopRecordingToProto(resp *driver.StopRecordingResponse) (*basrecording.StopRecordingResponse, error) {
	if resp == nil || resp.SessionID == "" || resp.RecordingID == "" || resp.ActionCount < 0 || resp.ActionCount > math.MaxInt32 {
		return nil, fmt.Errorf("invalid recording stop receipt")
	}
	stamp, err := recordingTimestamp(resp.StoppedAt)
	if err != nil {
		return nil, err
	}
	return &basrecording.StopRecordingResponse{SessionId: resp.SessionID, RecordingId: resp.RecordingID, ActionCount: int32(resp.ActionCount), CompletedAt: stamp}, nil
}

func GenerateWorkflowToProto(resp any) (*basrecording.GenerateWorkflowResponse, error) {
	var pb basrecording.GenerateWorkflowResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func ReplayPreviewToProto(resp any) (*basrecording.ReplayPreviewResponse, error) {
	var pb basrecording.ReplayPreviewResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func SelectorValidationToProto(resp any) (*basrecording.SelectorValidation, error) {
	var pb basrecording.SelectorValidation
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}
