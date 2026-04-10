package integrations

import (
	"encoding/json"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// ProtoRunStatusToLocal maps a proto RunStatus enum to the local RunStatus string type.
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

// localRunnerTypeToProto maps a local RunnerType string to the proto enum.
func localRunnerTypeToProto(rt RunnerType) domainpb.RunnerType {
	switch rt {
	case RunnerTypeClaudeCode:
		return domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	case RunnerTypeCodex:
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	case RunnerTypeOpenCode:
		return domainpb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED
	}
}

// protoRunPhaseToString maps a proto RunPhase enum to a simple string.
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

// protoEventTypeToString maps a proto RunEventType enum to the simple string the UI expects.
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

// TranslateProtoEvent converts a proto RunEvent to TranslatedEvent format.
func TranslateProtoEvent(ev *domainpb.RunEvent) *TranslatedEvent {
	if ev == nil {
		return nil
	}

	event := &TranslatedEvent{
		ID:       ev.GetId(),
		Sequence: ev.GetSequence(),
		Type:     protoEventTypeToString(ev.GetEventType()),
	}

	if ts := ev.GetTimestamp(); ts != nil {
		event.Timestamp = ts.AsTime()
	}

	switch d := ev.Data.(type) {
	case *domainpb.RunEvent_Message:
		translateMessageEvent(event, d.Message)
	case *domainpb.RunEvent_ToolCall:
		translateToolCallEvent(event, d.ToolCall)
	case *domainpb.RunEvent_ToolResult:
		translateToolResultEvent(event, d.ToolResult)
	case *domainpb.RunEvent_Status:
		translateStatusEvent(event, d.Status)
	case *domainpb.RunEvent_Error:
		translateErrorEvent(event, d.Error)
	case *domainpb.RunEvent_Log:
		translateLogEvent(event, d.Log)
	case *domainpb.RunEvent_Metric:
		translateMetricEvent(event, d.Metric)
	case *domainpb.RunEvent_Artifact:
		translateArtifactEvent(event, d.Artifact)
	case *domainpb.RunEvent_MessageDeleted:
		translateMessageDeletedEvent(event, d.MessageDeleted)
	case *domainpb.RunEvent_Compaction:
		translateCompactionEvent(event, d.Compaction)
	default:
		// Unknown/unhandled oneof variant (progress, cost, rate_limit, etc.)
		event.Role = "system"
	}

	return event
}

func translateMessageEvent(event *TranslatedEvent, msg *domainpb.MessageEventData) {
	event.Role = msg.GetRole()
	event.Content = msg.GetContent()
}

func translateToolCallEvent(event *TranslatedEvent, tc *domainpb.ToolCallEventData) {
	event.Role = "assistant"
	event.ToolName = tc.GetToolName()
	event.ToolCallID = tc.GetToolCallId()
	if input := tc.GetInput(); input != nil {
		inputBytes, _ := json.Marshal(input.AsMap())
		event.ToolInput = string(inputBytes)
	}
}

func translateToolResultEvent(event *TranslatedEvent, tr *domainpb.ToolResultEventData) {
	event.Role = "tool"
	event.ToolName = tr.GetToolName()
	event.ToolCallID = tr.GetToolCallId()
	event.ToolOutput = tr.GetOutput()
	event.ToolSuccess = tr.GetSuccess()
	if errMsg := tr.GetError(); errMsg != "" {
		event.ToolOutput = errMsg
		event.ToolSuccess = false
	}
}

func translateStatusEvent(event *TranslatedEvent, st *domainpb.StatusEventData) {
	event.Role = "system"
	event.RunStatus = st.GetNewStatus()
	event.Content = st.GetReason()
}

func translateErrorEvent(event *TranslatedEvent, errData *domainpb.ErrorEventData) {
	event.Role = "system"
	event.Content = errData.GetMessage()
}

func translateLogEvent(event *TranslatedEvent, logData *domainpb.LogEventData) {
	event.Role = "system"
	event.Content = logData.GetMessage()
	rawBytes, _ := protoMarshalOpts.Marshal(logData)
	event.RawData = string(rawBytes)
}

func translateMetricEvent(event *TranslatedEvent, metricData *domainpb.MetricEventData) {
	event.Role = "system"
	event.Content = metricData.GetName()
	rawBytes, _ := protoMarshalOpts.Marshal(metricData)
	event.RawData = string(rawBytes)
}

func translateArtifactEvent(event *TranslatedEvent, artData *domainpb.ArtifactEventData) {
	event.Role = "system"
	event.Content = artData.GetType()
	rawBytes, _ := protoMarshalOpts.Marshal(artData)
	event.RawData = string(rawBytes)
}

func translateMessageDeletedEvent(event *TranslatedEvent, mdData *domainpb.MessageDeletedEventData) {
	event.Role = "system"
	event.Content = mdData.GetTargetEventId()
	rawBytes, _ := protoMarshalOpts.Marshal(mdData)
	event.RawData = string(rawBytes)
}

func translateCompactionEvent(event *TranslatedEvent, compaction *domainpb.CompactionEventData) {
	event.Role = "system"
	event.Content = compaction.GetSummary()
	event.CompactionTrigger = compaction.GetTrigger()
	event.CompactionFocus = compaction.GetFocus()
	event.CompactionMessagesCompacted = compaction.GetMessagesCompacted()
	event.CompactionTokensBefore = compaction.GetTokensBefore()
	event.CompactionTokensAfter = compaction.GetTokensAfter()
	event.CompactionOriginalCommand = compaction.GetOriginalCommand()
}
