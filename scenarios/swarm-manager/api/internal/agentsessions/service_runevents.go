package agentsessions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"

	agentdomainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

type ListEventsRequest struct {
	SessionID     string
	AfterSequence int64
	Limit         int32
}

type ListEventsResult struct {
	Events            []RunEvent
	HasMore           bool
	NextAfterSequence int64
}

type RunEvent struct {
	ID              string `json:"id"`
	RunID           string `json:"run_id"`
	Sequence        int64  `json:"sequence"`
	CreatedAt       string `json:"created_at"`
	EventType       string `json:"event_type"`
	Role            string `json:"role,omitempty"`
	Content         string `json:"content,omitempty"`
	ToolName        string `json:"tool_name,omitempty"`
	ToolCallID      string `json:"tool_call_id,omitempty"`
	Input           string `json:"input,omitempty"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	Status          string `json:"status,omitempty"`
	PreviousStatus  string `json:"previous_status,omitempty"`
	ProgressPhase   string `json:"progress_phase,omitempty"`
	ProgressPercent int32  `json:"progress_percent,omitempty"`
	ProgressMessage string `json:"progress_message,omitempty"`
	Summary         string `json:"summary,omitempty"`
	RawJSON         string `json:"raw_json,omitempty"`
}

func (s *Service) ListEvents(ctx context.Context, req ListEventsRequest) (ListEventsResult, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return ListEventsResult{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return ListEventsResult{}, mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) == "" {
		return ListEventsResult{Events: []RunEvent{}}, nil
	}
	if s.eventReader == nil {
		return ListEventsResult{}, apierr.Unavailable("agent session events are unavailable")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	events, hasMore, err := s.eventReader.GetRunEvents(ctx, session.RunID, agentmanager.RunEventsOptions{
		AfterSequence: req.AfterSequence,
		Limit:         limit,
	})
	if err != nil {
		return ListEventsResult{}, mapSpawnError(err)
	}
	result := ListEventsResult{
		Events:  make([]RunEvent, 0, len(events)),
		HasMore: hasMore,
	}
	for _, event := range events {
		mapped := mapRunEvent(event)
		if mapped.Sequence > result.NextAfterSequence {
			result.NextAfterSequence = mapped.Sequence
		}
		result.Events = append(result.Events, mapped)
	}
	return result, nil
}

const maxRunEventFieldBytes = 6000

func mapRunEvent(event *agentdomainpb.RunEvent) RunEvent {
	if event == nil {
		return RunEvent{}
	}
	mapped := RunEvent{
		ID:        strings.TrimSpace(event.GetId()),
		RunID:     strings.TrimSpace(event.GetRunId()),
		Sequence:  event.GetSequence(),
		EventType: agentRunEventType(event.GetEventType()),
	}
	if ts := event.GetTimestamp(); ts != nil {
		mapped.CreatedAt = ts.AsTime().UTC().Format(time.RFC3339)
	}
	switch data := event.GetData().(type) {
	case *agentdomainpb.RunEvent_Message:
		mapped.Role = strings.TrimSpace(data.Message.GetRole())
		mapped.Content = boundedRunEventText(data.Message.GetContent())
	case *agentdomainpb.RunEvent_ToolCall:
		mapped.ToolName = strings.TrimSpace(data.ToolCall.GetToolName())
		mapped.ToolCallID = strings.TrimSpace(data.ToolCall.GetToolCallId())
		mapped.Input = boundedRunEventText(structJSON(data.ToolCall.GetInput()))
	case *agentdomainpb.RunEvent_ToolResult:
		mapped.ToolName = strings.TrimSpace(data.ToolResult.GetToolName())
		mapped.ToolCallID = strings.TrimSpace(data.ToolResult.GetToolCallId())
		mapped.Output = boundedRunEventText(data.ToolResult.GetOutput())
		mapped.Error = boundedRunEventText(data.ToolResult.GetError())
	case *agentdomainpb.RunEvent_Status:
		mapped.PreviousStatus = strings.TrimSpace(data.Status.GetOldStatus())
		mapped.Status = strings.TrimSpace(data.Status.GetNewStatus())
		mapped.Summary = boundedRunEventText(data.Status.GetReason())
	case *agentdomainpb.RunEvent_Error:
		mapped.Error = boundedRunEventText(data.Error.GetMessage())
		if code := strings.TrimSpace(data.Error.GetCode()); code != "" {
			mapped.Status = code
		}
		mapped.RawJSON = boundedRunEventText(structJSON(data.Error.GetDetails()))
	case *agentdomainpb.RunEvent_Progress:
		mapped.ProgressPhase = strings.TrimSpace(data.Progress.GetPhase().String())
		mapped.ProgressPercent = data.Progress.GetPercentComplete()
		mapped.ProgressMessage = boundedRunEventText(data.Progress.GetCurrentAction())
	case *agentdomainpb.RunEvent_Compaction:
		mapped.Summary = boundedRunEventText(data.Compaction.GetSummary())
		mapped.ProgressMessage = boundedRunEventText(data.Compaction.GetTrigger())
	case *agentdomainpb.RunEvent_Log:
		mapped.Summary = boundedRunEventText(data.Log.GetMessage())
		mapped.Status = strings.TrimSpace(data.Log.GetLevel())
	case *agentdomainpb.RunEvent_Metric:
		mapped.Summary = boundedRunEventText(data.Metric.GetName())
	case *agentdomainpb.RunEvent_Artifact:
		mapped.Summary = boundedRunEventText(data.Artifact.GetPath())
	case *agentdomainpb.RunEvent_Cost:
		mapped.Summary = boundedRunEventText(fmt.Sprintf("$%.4f", data.Cost.GetTotalCostUsd()))
	case *agentdomainpb.RunEvent_RateLimit:
		mapped.Error = boundedRunEventText(data.RateLimit.GetMessage())
		mapped.Status = strings.TrimSpace(data.RateLimit.GetLimitType())
	default:
		mapped.RawJSON = boundedRunEventText(protoMessageJSON(event))
	}
	return mapped
}

func agentRunEventType(eventType agentdomainpb.RunEventType) string {
	switch eventType {
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		return "log"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		return "message"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		return "tool_call"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		return "tool_result"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		return "status"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_METRIC:
		return "metric"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
		return "artifact"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		return "error"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
		return "message_deleted"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION:
		return "compaction"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_LIFECYCLE:
		return "lifecycle"
	default:
		return "unknown"
	}
}

func structJSON(value *structpb.Struct) string {
	if value == nil {
		return ""
	}
	return protoMessageJSON(value)
}

func protoMessageJSON(value interface{ ProtoReflect() protoreflect.Message }) string {
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func boundedRunEventText(value string) string {
	value = strings.TrimSpace(value)
	if normalized, ok := normalizeHTMLErrorPage(value); ok {
		value = normalized
	}
	if len(value) <= maxRunEventFieldBytes {
		return value
	}
	return value[:maxRunEventFieldBytes] + "\n[truncated]"
}

func normalizeHTMLErrorPage(value string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return "", false
	}
	if !strings.HasPrefix(lower, "<!doctype html") &&
		!strings.HasPrefix(lower, "<html") &&
		!strings.Contains(lower, "<title>") {
		return "", false
	}
	if strings.Contains(lower, "cloudflare") ||
		strings.Contains(lower, "cf-error") ||
		strings.Contains(lower, "bad gateway") ||
		strings.Contains(lower, "502") {
		return "Upstream tunnel returned an HTML 502 Bad Gateway page. The target service may be unavailable or timed out.", true
	}
	if strings.Contains(lower, "gateway timeout") ||
		strings.Contains(lower, "service unavailable") ||
		strings.Contains(lower, "error 5") {
		return "Upstream service returned an HTML 5xx error page.", true
	}
	return "", false
}
