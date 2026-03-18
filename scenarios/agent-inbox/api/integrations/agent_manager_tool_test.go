package integrations

import (
	"strings"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
