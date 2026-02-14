package integrations

import (
	"encoding/json"
	"strings"
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

func TestTranslateProtoEvent_Status(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-6",
		Sequence:  6,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_STATUS,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 35, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Status{
			Status: &domainpb.StatusEventData{
				NewStatus: "running",
				Reason:    "Agent started processing",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "status" {
		t.Errorf("expected type status, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.RunStatus != "running" {
		t.Errorf("expected run_status running, got %s", event.RunStatus)
	}
	if event.Content != "Agent started processing" {
		t.Errorf("expected content 'Agent started processing', got %s", event.Content)
	}
}

func TestTranslateProtoEvent_StatusNoReason(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-16",
		Sequence:  1,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_STATUS,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Status{
			Status: &domainpb.StatusEventData{
				NewStatus: "complete",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.RunStatus != "complete" {
		t.Errorf("expected run_status complete, got %s", event.RunStatus)
	}
	if event.Content != "" {
		t.Errorf("expected empty content when no reason, got %s", event.Content)
	}
}

func TestTranslateProtoEvent_Error(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-7",
		Sequence:  7,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_ERROR,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 36, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Error{
			Error: &domainpb.ErrorEventData{
				Message: "something went wrong",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "error" {
		t.Errorf("expected type error, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "something went wrong" {
		t.Errorf("expected content 'something went wrong', got %s", event.Content)
	}
}

func TestTranslateProtoEvent_Log(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-8",
		Sequence:  8,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_LOG,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 37, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Log{
			Log: &domainpb.LogEventData{
				Level:   "debug",
				Message: "internal log",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event for log")
	}
	if event.Type != "log" {
		t.Errorf("expected type log, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "internal log" {
		t.Errorf("expected content 'internal log', got %s", event.Content)
	}
	if event.RawData == "" {
		t.Error("expected non-empty raw_data")
	}
	if !strings.Contains(event.RawData, `"level"`) {
		t.Errorf("expected raw_data to contain level, got %s", event.RawData)
	}
}

func TestTranslateProtoEvent_Metric(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-20",
		Sequence:  20,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_METRIC,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Metric{
			Metric: &domainpb.MetricEventData{
				Name:  "tokens_used",
				Value: 1500,
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event for metric")
	}
	if event.Type != "metric" {
		t.Errorf("expected type metric, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "tokens_used" {
		t.Errorf("expected content 'tokens_used', got %s", event.Content)
	}
	if event.RawData == "" {
		t.Error("expected non-empty raw_data")
	}
	if !strings.Contains(event.RawData, `"value"`) {
		t.Errorf("expected raw_data to contain value, got %s", event.RawData)
	}
}

func TestTranslateProtoEvent_Artifact(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-21",
		Sequence:  21,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 11, 1, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Artifact{
			Artifact: &domainpb.ArtifactEventData{
				Type: "file",
				Path: "/tmp/output.txt",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event for artifact")
	}
	if event.Type != "artifact" {
		t.Errorf("expected type artifact, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "file" {
		t.Errorf("expected content 'file', got %s", event.Content)
	}
	if event.RawData == "" {
		t.Error("expected non-empty raw_data")
	}
	if !strings.Contains(event.RawData, `"path"`) {
		t.Errorf("expected raw_data to contain path, got %s", event.RawData)
	}
}

func TestTranslateProtoEvent_MessageDeleted(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-22",
		Sequence:  22,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 11, 2, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_MessageDeleted{
			MessageDeleted: &domainpb.MessageDeletedEventData{
				TargetEventId: "evt-5",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event for message_deleted")
	}
	if event.Type != "message_deleted" {
		t.Errorf("expected type message_deleted, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "evt-5" {
		t.Errorf("expected content 'evt-5', got %s", event.Content)
	}
	if event.RawData == "" {
		t.Error("expected non-empty raw_data")
	}
}

func TestTranslateProtoEvent_Nil(t *testing.T) {
	event := TranslateProtoEvent(nil)
	if event != nil {
		t.Error("expected nil event for nil input")
	}
}

func TestTranslateProtoEvent_NoDataField(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-30",
		Sequence:  30,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		// Data is nil — no oneof variant set
	}

	// Should not panic
	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Role != "system" {
		t.Errorf("expected default role system for nil data, got %s", event.Role)
	}
}

func TestTranslateProtoEvent_NoTimestamp(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-31",
		Sequence:  1,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		// Timestamp is nil
		Data: &domainpb.RunEvent_Message{
			Message: &domainpb.MessageEventData{
				Role:    "user",
				Content: "test",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if !event.Timestamp.IsZero() {
		t.Errorf("expected zero timestamp for nil proto timestamp, got %v", event.Timestamp)
	}
}

func TestTranslateProtoEvent_LargeSequence(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-32",
		Sequence:  9999999,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		Timestamp: timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_Message{
			Message: &domainpb.MessageEventData{
				Role:    "user",
				Content: "test",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Sequence != 9999999 {
		t.Errorf("expected sequence 9999999, got %d", event.Sequence)
	}
}

func TestTranslateProtoEvent_ToolCallID(t *testing.T) {
	input, _ := structpb.NewStruct(map[string]interface{}{"command": "ls"})
	ev := &domainpb.RunEvent{
		Id:        "evt-tc-id",
		Sequence:  50,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL,
		Timestamp: timestamppb.New(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolCall{
			ToolCall: &domainpb.ToolCallEventData{
				ToolName:   "Bash",
				Input:      input,
				ToolCallId: "call_abc123",
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ToolCallID != "call_abc123" {
		t.Errorf("expected ToolCallID call_abc123, got %s", event.ToolCallID)
	}
}

func TestTranslateProtoEvent_ToolResultCallID(t *testing.T) {
	ev := &domainpb.RunEvent{
		Id:        "evt-tr-id",
		Sequence:  51,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT,
		Timestamp: timestamppb.New(time.Date(2025, 6, 1, 12, 0, 1, 0, time.UTC)),
		Data: &domainpb.RunEvent_ToolResult{
			ToolResult: &domainpb.ToolResultEventData{
				ToolName:   "Bash",
				ToolCallId: "call_abc123",
				Output:     "done",
				Success:    true,
			},
		},
	}

	event := TranslateProtoEvent(ev)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ToolCallID != "call_abc123" {
		t.Errorf("expected ToolCallID call_abc123, got %s", event.ToolCallID)
	}
	if event.ToolOutput != "done" {
		t.Errorf("expected output 'done', got %s", event.ToolOutput)
	}
}

// =============================================================================
// ProtoRunStatusToLocal tests
// =============================================================================

func TestProtoRunStatusToLocal(t *testing.T) {
	tests := []struct {
		proto    domainpb.RunStatus
		expected RunStatus
	}{
		{domainpb.RunStatus_RUN_STATUS_PENDING, RunStatusPending},
		{domainpb.RunStatus_RUN_STATUS_STARTING, RunStatusStarting},
		{domainpb.RunStatus_RUN_STATUS_RUNNING, RunStatusRunning},
		{domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW, RunStatusNeedsReview},
		{domainpb.RunStatus_RUN_STATUS_COMPLETE, RunStatusComplete},
		{domainpb.RunStatus_RUN_STATUS_FAILED, RunStatusFailed},
		{domainpb.RunStatus_RUN_STATUS_CANCELLED, RunStatusCancelled},
	}

	for _, tc := range tests {
		t.Run(string(tc.expected), func(t *testing.T) {
			got := ProtoRunStatusToLocal(tc.proto)
			if got != tc.expected {
				t.Errorf("ProtoRunStatusToLocal(%v) = %q, want %q", tc.proto, got, tc.expected)
			}
		})
	}
}

func TestProtoRunStatusToLocal_Unspecified(t *testing.T) {
	got := ProtoRunStatusToLocal(domainpb.RunStatus_RUN_STATUS_UNSPECIFIED)
	// Should return the enum string representation for unknown values
	if got == "" {
		t.Error("expected non-empty status for unspecified")
	}
}
