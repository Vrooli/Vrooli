package integrations

import (
	"encoding/json"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// Proto-based TranslateProtoEvent tests
// =============================================================================

func TestTranslateProtoEvent_MessageUser(t *testing.T) {
	ts := timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	ev := &domainpb.RunEvent{
		Id:        "evt-1",
		Sequence:  1,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		Timestamp: ts,
		Data: &domainpb.RunEvent_Message{
			Message: &domainpb.MessageEventData{
				Role:    "user",
				Content: "Hello agent",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ID != "evt-1" {
		t.Errorf("expected ID evt-1, got %s", event.ID)
	}
	if event.Type != "message" {
		t.Errorf("expected type message, got %s", event.Type)
	}
	if event.Role != "user" {
		t.Errorf("expected role user, got %s", event.Role)
	}
	if event.Content != "Hello agent" {
		t.Errorf("expected content 'Hello agent', got %s", event.Content)
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
	expectedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !event.Timestamp.Equal(expectedTime) {
		t.Errorf("expected timestamp %v, got %v", expectedTime, event.Timestamp)
	}
}

func TestTranslateProtoEvent_MessageAssistant(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-2",
		Sequence:  2,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 31, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Message{
			Message: &domainpb.MessageEventData{
				Role:    "assistant",
				Content: "I'll help you with that",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Role != "assistant" {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.Content != "I'll help you with that" {
		t.Errorf("unexpected content: %s", event.Content)
	}
}

func TestTranslateProtoEvent_ToolCall(t *testing.T) {
	input, _ := structpb.NewStruct(map[string]interface{}{"path": "/tmp/test.txt"})
	ev := &domainpb.RunEvent{
		Id:        "evt-3",
		Sequence:  3,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 32, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolCall{
			ToolCall: &domainpb.ToolCallEventData{
				ToolName: "read_file",
				Input:    input,
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "tool_call" {
		t.Errorf("expected type tool_call, got %s", event.Type)
	}
	if event.Role != "assistant" {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.ToolName != "read_file" {
		t.Errorf("expected toolName read_file, got %s", event.ToolName)
	}
	if event.ToolInput == "" {
		t.Error("expected non-empty tool input")
	}
	// Parse and verify input contains expected data
	var parsedInput map[string]interface{}
	if err := json.Unmarshal([]byte(event.ToolInput), &parsedInput); err != nil {
		t.Fatalf("failed to parse tool input JSON: %v", err)
	}
	if parsedInput["path"] != "/tmp/test.txt" {
		t.Errorf("unexpected tool input: %s", event.ToolInput)
	}
}

func TestTranslateProtoEvent_ToolCallNoInput(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-14",
		Sequence:  1,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolCall{
			ToolCall: &domainpb.ToolCallEventData{
				ToolName: "list_files",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ToolName != "list_files" {
		t.Errorf("expected toolName list_files, got %s", event.ToolName)
	}
	if event.ToolInput != "" {
		t.Errorf("expected empty tool input, got %s", event.ToolInput)
	}
}

func TestTranslateProtoEvent_ToolResultSuccess(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-4",
		Sequence:  4,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 33, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolResult{
			ToolResult: &domainpb.ToolResultEventData{
				ToolName: "read_file",
				Output:   "file contents here",
				Success:  true,
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "tool_result" {
		t.Errorf("expected type tool_result, got %s", event.Type)
	}
	if event.Role != "tool" {
		t.Errorf("expected role tool, got %s", event.Role)
	}
	if event.ToolOutput != "file contents here" {
		t.Errorf("expected output 'file contents here', got %s", event.ToolOutput)
	}
	if !event.ToolSuccess {
		t.Error("expected tool_success to be true")
	}
}

func TestTranslateProtoEvent_ToolResultError(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-5",
		Sequence:  5,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 34, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolResult{
			ToolResult: &domainpb.ToolResultEventData{
				ToolName: "read_file",
				Output:   "original output",
				Success:  true,
				Error:    "permission denied",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	// Error field should override output and set success=false
	if event.ToolOutput != "permission denied" {
		t.Errorf("expected error to override output, got %s", event.ToolOutput)
	}
	if event.ToolSuccess {
		t.Error("expected tool_success to be false when error is set")
	}
}

func TestTranslateProtoEvent_ToolResultEmptyError(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-15",
		Sequence:  1,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolResult{
			ToolResult: &domainpb.ToolResultEventData{
				ToolName: "read_file",
				Output:   "file content",
				Success:  true,
				Error:    "", // Empty error string should NOT override
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ToolOutput != "file content" {
		t.Errorf("expected output 'file content', got %s", event.ToolOutput)
	}
	if !event.ToolSuccess {
		t.Error("expected tool_success to remain true for empty error")
	}
}
