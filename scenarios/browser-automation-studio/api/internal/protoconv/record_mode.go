package protoconv

import (
	"encoding/json"
	"fmt"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
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

func RecordingStatusToProto(status any) (*browser_automation_studio_v1.RecordingStatusResponse, error) {
	var pb browser_automation_studio_v1.RecordingStatusResponse
	if err := selectorPayloadToProto(status, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func RecordingSessionToProto(session any) (*browser_automation_studio_v1.CreateRecordingSessionResponse, error) {
	var pb browser_automation_studio_v1.CreateRecordingSessionResponse
	if err := selectorPayloadToProto(session, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func StartRecordingToProto(resp any) (*browser_automation_studio_v1.StartRecordingResponse, error) {
	var pb browser_automation_studio_v1.StartRecordingResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func StopRecordingToProto(resp any) (*browser_automation_studio_v1.StopRecordingResponse, error) {
	var pb browser_automation_studio_v1.StopRecordingResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func GenerateWorkflowToProto(resp any) (*browser_automation_studio_v1.GenerateWorkflowResponse, error) {
	var pb browser_automation_studio_v1.GenerateWorkflowResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func ReplayPreviewToProto(resp any) (*browser_automation_studio_v1.ReplayPreviewResponse, error) {
	var pb browser_automation_studio_v1.ReplayPreviewResponse
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

func SelectorValidationToProto(resp any) (*browser_automation_studio_v1.SelectorValidation, error) {
	var pb browser_automation_studio_v1.SelectorValidation
	if err := selectorPayloadToProto(resp, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}
