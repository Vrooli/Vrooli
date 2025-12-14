package protoconv

import (
	"encoding/json"
	"fmt"

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

func RecordingStatusToProto(status any) (*basrecording.RecordingStatusResponse, error) {
	var pb basrecording.RecordingStatusResponse
	if err := selectorPayloadToProto(status, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func RecordingSessionToProto(session any) (*basrecording.CreateRecordingSessionResponse, error) {
	var pb basrecording.CreateRecordingSessionResponse
	if err := selectorPayloadToProto(session, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func StartRecordingToProto(resp any) (*basrecording.StartRecordingResponse, error) {
	var pb basrecording.StartRecordingResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func StopRecordingToProto(resp any) (*basrecording.StopRecordingResponse, error) {
	var pb basrecording.StopRecordingResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
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
