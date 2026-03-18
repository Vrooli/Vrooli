package integrations

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProtoRunStatusToLocal tests

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
	if got == "" {
		t.Error("expected non-empty status for unspecified")
	}
}

// Compaction Event Tests

func TestTranslateProtoEvent_Compaction(t *testing.T) {
	ts := timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	ev := &domainpb.RunEvent{
		Id:        "evt-compaction-1",
		Sequence:  42,
		EventType: domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION,
		Timestamp: ts,
		Data: &domainpb.RunEvent_Compaction{
			Compaction: &domainpb.CompactionEventData{
				Summary:           "We fixed the authentication bug by...",
				Trigger:           "manual",
				Focus:             "auth",
				MessagesCompacted: 47,
				TokensBefore:      89432,
				TokensAfter:       3201,
				OriginalCommand:   "/compact focus on auth",
			},
		},
	}

	result := TranslateProtoEvent(ev)

	if result.Type != "compaction" {
		t.Errorf("Type = %s, want compaction", result.Type)
	}
	if result.Role != "system" {
		t.Errorf("Role = %s, want system", result.Role)
	}
	if result.Content != "We fixed the authentication bug by..." {
		t.Errorf("Content = %s, want summary", result.Content)
	}
	if result.CompactionTrigger != "manual" {
		t.Errorf("CompactionTrigger = %s, want manual", result.CompactionTrigger)
	}
	if result.CompactionMessagesCompacted != 47 {
		t.Errorf("CompactionMessagesCompacted = %d, want 47", result.CompactionMessagesCompacted)
	}
}

func TestProtoEventTypeToString_Compaction(t *testing.T) {
	result := protoEventTypeToString(domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION)
	if result != "compaction" {
		t.Errorf("protoEventTypeToString(COMPACTION) = %s, want compaction", result)
	}
}

// Edge case tests

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
	}

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
		t.Errorf("expected zero timestamp, got %v", event.Timestamp)
	}
}
