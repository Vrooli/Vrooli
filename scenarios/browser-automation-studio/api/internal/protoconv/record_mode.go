package protoconv

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
	pb.ActionKind = mapRecordedActionKind(action.ActionType)
	// Populate typed payload for the common path.
	if obj := toJsonObjectFromAny(action.Payload); obj != nil {
		pb.PayloadTyped = obj
	}
	buildTypedActionPayload(&pb, action)
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			pb.Timestamp = timestamppb.New(ts)
		}
	}
	return &pb, nil
}

func mapRecordedActionKind(actionType string) browser_automation_studio_v1.RecordedActionType {
	switch strings.ToLower(actionType) {
	case "navigate":
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_NAVIGATE
	case "click", "focus", "hover", "blur":
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_CLICK
	case "type", "input", "keypress":
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_INPUT
	case "wait":
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_WAIT
	case "assert":
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_ASSERT
	default:
		return browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_UNSPECIFIED
	}
}

func buildTypedActionPayload(pb *browser_automation_studio_v1.RecordedAction, action RecordedAction) {
	kind := pb.GetActionKind()
	payload := action.Payload
	switch kind {
	case browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_NAVIGATE:
		url := action.URL
		if val, ok := firstString(payload, "targetUrl", "url"); ok && val != "" {
			url = val
		}
		nav := &browser_automation_studio_v1.NavigateActionPayload{
			Url: url,
		}
		if selector, ok := firstString(payload, "waitForSelector", "wait_for_selector"); ok {
			nav.WaitForSelector = proto.String(selector)
		}
		if timeout, ok := firstInt32(payload, "timeoutMs", "timeout_ms"); ok {
			nav.TimeoutMs = proto.Int32(timeout)
		}
		pb.TypedAction = &browser_automation_studio_v1.RecordedAction_Navigate{Navigate: nav}
	case browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_CLICK:
		click := &browser_automation_studio_v1.ClickActionPayload{
			Button: firstStringDefault(payload, "button", ""),
		}
		if count, ok := firstInt32(payload, "clickCount", "click_count"); ok {
			click.ClickCount = proto.Int32(count)
		}
		if delay, ok := firstInt32(payload, "delayMs", "delay_ms", "delay"); ok {
			click.DelayMs = proto.Int32(delay)
		}
		if scroll, ok := firstBool(payload, "scrollIntoView", "scroll_into_view"); ok {
			click.ScrollIntoView = proto.Bool(scroll)
		}
		pb.TypedAction = &browser_automation_studio_v1.RecordedAction_Click{Click: click}
	case browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_INPUT:
		text := firstStringDefault(payload, "text", "value")
		input := &browser_automation_studio_v1.InputActionPayload{
			Value: text,
		}
		if sensitive, ok := firstBool(payload, "isSensitive", "sensitive"); ok {
			input.IsSensitive = sensitive
		}
		if submit, ok := firstBool(payload, "submit"); ok {
			input.Submit = proto.Bool(submit)
		}
		pb.TypedAction = &browser_automation_studio_v1.RecordedAction_Input{Input: input}
	case browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_WAIT:
		if duration, ok := firstInt32(payload, "durationMs", "duration_ms", "duration"); ok {
			pb.TypedAction = &browser_automation_studio_v1.RecordedAction_Wait{
				Wait: &browser_automation_studio_v1.WaitActionPayload{
					DurationMs: duration,
				},
			}
		}
	case browser_automation_studio_v1.RecordedActionType_RECORDED_ACTION_TYPE_ASSERT:
		assert := &browser_automation_studio_v1.AssertActionPayload{
			Mode:          firstStringDefault(payload, "mode", ""),
			Selector:      firstStringDefault(payload, "selector", ""),
			Negated:       firstBoolDefault(payload, "negated", false),
			CaseSensitive: firstBoolDefault(payload, "caseSensitive", false),
		}
		if expected, ok := payload["expected"]; ok {
			if val, err := structpb.NewValue(expected); err == nil {
				assert.Expected = val
			}
		}
		pb.TypedAction = &browser_automation_studio_v1.RecordedAction_Assert{Assert: assert}
	}
}

func firstString(payload map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if val, ok := payload[key]; ok {
			switch typed := val.(type) {
			case string:
				return typed, true
			case fmt.Stringer:
				return typed.String(), true
			}
		}
	}
	return "", false
}

func firstStringDefault(payload map[string]any, key string, fallback string) string {
	if payload == nil {
		return fallback
	}
	if val, ok := firstString(payload, key); ok {
		return val
	}
	return fallback
}

func firstInt32(payload map[string]any, keys ...string) (int32, bool) {
	for _, key := range keys {
		if val, ok := payload[key]; ok {
			switch v := val.(type) {
			case int:
				return int32(v), true
			case int32:
				return v, true
			case int64:
				return int32(v), true
			case float64:
				return int32(v), true
			case json.Number:
				if i, err := v.Int64(); err == nil {
					return int32(i), true
				}
			}
		}
	}
	return 0, false
}

func firstBool(payload map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if val, ok := payload[key]; ok {
			switch v := val.(type) {
			case bool:
				return v, true
			case string:
				lower := strings.ToLower(v)
				if lower == "true" {
					return true, true
				}
				if lower == "false" {
					return false, true
				}
			}
		}
	}
	return false, false
}

func firstBoolDefault(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	if val, ok := firstBool(payload, key); ok {
		return val
	}
	return fallback
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
	for _, result := range pb.Results {
		result.ActionKind = mapRecordedActionKind(result.GetActionType())
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
