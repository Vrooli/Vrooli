package integrations

import (
	"encoding/json"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

func ProtoRunStatusToLocal(s domainpb.RunStatus) RunStatus {
	switch s {
	case domainpb.RunStatus_RUN_STATUS_PENDING:
		return RunStatusPending
	case domainpb.RunStatus_RUN_STATUS_STARTING:
		return RunStatusStarting
	case domainpb.RunStatus_RUN_STATUS_RUNNING:
		return RunStatusRunning
	case domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return RunStatusNeedsReview
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return RunStatusComplete
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return RunStatusFailed
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return RunStatusCancelled
	default:
		return RunStatus(s.String())
	}
}

func protoRunPhaseToString(p domainpb.RunPhase) string {
	switch p {
	case domainpb.RunPhase_RUN_PHASE_QUEUED:
		return "queued"
	case domainpb.RunPhase_RUN_PHASE_INITIALIZING:
		return "initializing"
	case domainpb.RunPhase_RUN_PHASE_SANDBOX_CREATING:
		return "sandbox_creating"
	case domainpb.RunPhase_RUN_PHASE_RUNNER_ACQUIRING:
		return "runner_acquiring"
	case domainpb.RunPhase_RUN_PHASE_EXECUTING:
		return "executing"
	case domainpb.RunPhase_RUN_PHASE_COLLECTING_RESULTS:
		return "collecting_results"
	case domainpb.RunPhase_RUN_PHASE_AWAITING_REVIEW:
		return "awaiting_review"
	case domainpb.RunPhase_RUN_PHASE_APPLYING:
		return "applying"
	case domainpb.RunPhase_RUN_PHASE_CLEANING_UP:
		return "cleaning_up"
	case domainpb.RunPhase_RUN_PHASE_COMPLETED:
		return "completed"
	default:
		return ""
	}
}

func protoEventTypeToString(et domainpb.RunEventType) string {
	switch et {
	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		return "message"
	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		return "tool_call"
	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		return "tool_result"
	case domainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		return "status"
	case domainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		return "error"
	case domainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		return "log"
	case domainpb.RunEventType_RUN_EVENT_TYPE_METRIC:
		return "metric"
	case domainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
		return "artifact"
	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
		return "message_deleted"
	case domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION:
		return "compaction"
	default:
		return et.String()
	}
}

// TranslateProtoEvent preserves Agent Manager event detail while keeping the
// inbox transport independent of any runner-specific protocol.
func TranslateProtoEvent(ev *domainpb.RunEvent) *TranslatedEvent {
	if ev == nil {
		return nil
	}
	event := &TranslatedEvent{ID: ev.GetId(), Sequence: ev.GetSequence(), Type: protoEventTypeToString(ev.GetEventType()), Role: "system"}
	if ts := ev.GetTimestamp(); ts != nil {
		event.Timestamp = ts.AsTime()
	}
	switch data := ev.Data.(type) {
	case *domainpb.RunEvent_Message:
		event.Role, event.Content = data.Message.GetRole(), data.Message.GetContent()
	case *domainpb.RunEvent_ToolCall:
		event.Role, event.ToolName, event.ToolCallID = "assistant", data.ToolCall.GetToolName(), data.ToolCall.GetToolCallId()
		if input := data.ToolCall.GetInput(); input != nil {
			encoded, _ := json.Marshal(input.AsMap())
			event.ToolInput = string(encoded)
		}
	case *domainpb.RunEvent_ToolResult:
		event.Role, event.ToolName, event.ToolCallID = "tool", data.ToolResult.GetToolName(), data.ToolResult.GetToolCallId()
		event.ToolOutput, event.ToolSuccess = data.ToolResult.GetOutput(), data.ToolResult.GetSuccess()
		if message := data.ToolResult.GetError(); message != "" {
			event.ToolOutput, event.ToolSuccess = message, false
		}
	case *domainpb.RunEvent_Status:
		event.RunStatus, event.Content = data.Status.GetNewStatus(), data.Status.GetReason()
	case *domainpb.RunEvent_Error:
		event.Content = data.Error.GetMessage()
	case *domainpb.RunEvent_Log:
		event.Content = data.Log.GetMessage()
		event.RawData = marshalRaw(data.Log)
	case *domainpb.RunEvent_Metric:
		event.Content = data.Metric.GetName()
		event.RawData = marshalRaw(data.Metric)
	case *domainpb.RunEvent_Artifact:
		event.Content = data.Artifact.GetType()
		event.RawData = marshalRaw(data.Artifact)
	case *domainpb.RunEvent_MessageDeleted:
		event.Content = data.MessageDeleted.GetTargetEventId()
		event.RawData = marshalRaw(data.MessageDeleted)
	case *domainpb.RunEvent_Compaction:
		event.Content = data.Compaction.GetSummary()
		event.CompactionTrigger = data.Compaction.GetTrigger()
		event.CompactionFocus = data.Compaction.GetFocus()
		event.CompactionMessagesCompacted = data.Compaction.GetMessagesCompacted()
		event.CompactionTokensBefore = data.Compaction.GetTokensBefore()
		event.CompactionTokensAfter = data.Compaction.GetTokensAfter()
		event.CompactionOriginalCommand = data.Compaction.GetOriginalCommand()
	}
	return event
}

func marshalRaw(message proto.Message) string {
	// The generated protojson encoder is intentionally used through the shared
	// options so inbox event payloads retain the same wire names as the API.
	if encoded, err := protoMarshalOpts.Marshal(message); err == nil {
		return string(encoded)
	}
	return ""
}
