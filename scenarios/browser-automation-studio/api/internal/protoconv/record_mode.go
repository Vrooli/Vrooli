package protoconv

import (
	"encoding/json"
	"fmt"
	"time"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecordedAction mirrors the handler's record_mode payload without depending on the handlers package.
// Keeping it here avoids import cycles when protoconv is used by handlers.
type RecordedAction struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int                    `json:"durationMs,omitempty"`
	ActionType  string                 `json:"actionType"`
	Confidence  float64                `json:"confidence"`
	Selector    *SelectorSet           `json:"selector,omitempty"`
	ElementMeta *ElementMeta           `json:"elementMeta,omitempty"`
	BoundingBox *BoundingBox           `json:"boundingBox,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	URL         string                 `json:"url"`
	FrameID     string                 `json:"frameId,omitempty"`
	CursorPos   *Point                 `json:"cursorPos,omitempty"`
}

type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates"`
}

type SelectorCandidate struct {
	Selector   string `json:"selector"`
	Confidence int    `json:"confidence"`
	Type       string `json:"type"`
}

type ElementMeta struct {
	Tag        string            `json:"tag"`
	Text       string            `json:"text"`
	Type       string            `json:"type"`
	Attributes map[string]string `json:"attributes"`
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

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

func RecordedActionToProto(action RecordedAction) (*browser_automation_studio_v1.RecordedAction, error) {
	var pb browser_automation_studio_v1.RecordedAction
	if err := selectorPayloadToProto(action, &pb); err != nil {
		return nil, err
	}
	// Populate typed payload for the common path.
	if obj := toJsonObjectFromAny(action.Payload); obj != nil {
		pb.PayloadTyped = obj
	}
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			pb.Timestamp = timestamppb.New(ts)
		}
	}
	return &pb, nil
}

func RecordedActionsToProto(actions []RecordedAction) ([]*browser_automation_studio_v1.RecordedAction, error) {
	result := make([]*browser_automation_studio_v1.RecordedAction, 0, len(actions))
	for _, act := range actions {
		pb, err := RecordedActionToProto(act)
		if err != nil {
			return nil, err
		}
		result = append(result, pb)
	}
	return result, nil
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

func GetActionsToProto(resp any) (*browser_automation_studio_v1.GetActionsResponse, error) {
	var pb browser_automation_studio_v1.GetActionsResponse
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
