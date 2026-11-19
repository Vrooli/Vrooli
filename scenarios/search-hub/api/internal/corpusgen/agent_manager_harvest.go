package corpusgen

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

// RunEventsReader is the narrow Agent Manager read surface used by the caller
// audit. The concrete client must be authenticated by its owner at wiring time.
type RunEventsReader interface {
	GetRunEvents(context.Context, string, int64, int) (AgentManagerEventPage, error)
}

type AgentManagerEventPage struct {
	Events  []AgentManagerEvent `json:"events"`
	HasMore bool                `json:"has_more"`
}

type AgentManagerEvent struct {
	ID             string                       `json:"id"`
	RunID          string                       `json:"run_id"`
	Sequence       int64                        `json:"-"`
	EventType      string                       `json:"event_type"`
	Message        *AgentManagerMessage         `json:"message"`
	MessageDeleted *AgentManagerMessageDeletion `json:"message_deleted"`
}

type AgentManagerMessage struct {
	Role           string `json:"role"`
	Source         string `json:"source"`
	Content        string `json:"content"`
	ConversationID string `json:"conversation_id"`
	EvidenceOnly   bool   `json:"evidence_only"`
}

type AgentManagerMessageDeletion struct {
	TargetEventID string `json:"target_event_id"`
}

// AgentManagerConnectReader uses the generated event RPC. A configured client
// carries the scenario's authentication and transport policy.
type AgentManagerConnectReader struct {
	BaseURL string
	Client  apiconnect.AgentManagerServiceClient
}

func NewAgentManagerConnectReader(baseURL string, client *http.Client) AgentManagerConnectReader {
	if client == nil {
		client = http.DefaultClient
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return AgentManagerConnectReader{BaseURL: baseURL, Client: apiconnect.NewAgentManagerServiceClient(client, baseURL)}
}

func (r AgentManagerConnectReader) GetRunEvents(ctx context.Context, runID string, after int64, limit int) (AgentManagerEventPage, error) {
	var page AgentManagerEventPage
	if r.Client == nil || strings.TrimSpace(r.BaseURL) == "" || strings.TrimSpace(runID) == "" || limit < 1 || limit > 1000 {
		return page, fmt.Errorf("Agent Manager Connect client, run ID and bounded page size are required")
	}
	limit32 := int32(limit)
	response, err := r.Client.GetRunEvents(ctx, connect.NewRequest(&api.GetRunEventsRequest{
		RunId: runID, AfterSequence: &after, Limit: &limit32,
		EventTypes: []domain.RunEventType{domain.RunEventType_RUN_EVENT_TYPE_MESSAGE, domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED},
	}))
	if err != nil {
		return page, fmt.Errorf("request Agent Manager events: %w", err)
	}
	if response == nil || response.Msg == nil {
		return page, fmt.Errorf("Agent Manager returned an empty Connect event response")
	}
	page.HasMore = response.Msg.GetHasMore()
	for _, event := range response.Msg.GetEvents() {
		converted := AgentManagerEvent{
			ID: event.GetId(), RunID: event.GetRunId(), Sequence: event.GetSequence(), EventType: event.GetEventType().String(),
		}
		if message := event.GetMessage(); message != nil {
			converted.Message = &AgentManagerMessage{
				Role: message.GetRole(), Source: message.GetSource(), Content: message.GetContent(),
				ConversationID: message.GetConversationId(), EvidenceOnly: message.GetEvidenceOnly(),
			}
		}
		if deleted := event.GetMessageDeleted(); deleted != nil {
			converted.MessageDeleted = &AgentManagerMessageDeletion{TargetEventID: deleted.GetTargetEventId()}
		}
		page.Events = append(page.Events, converted)
	}
	return page, nil
}

// AgentManagerCallerRecord retains the original event and conversation
// identity. Query content is held only in memory for corpus review.
type AgentManagerCallerRecord struct {
	RunID, EventID, ConversationID string
	Sequence                       uint64
	MessageRole, MessageSource     string
	Query                          string
}

// AuditedAgentManagerCallerCase joins a harvested source record to the executed
// Gateway route and independent label review. Conversation identity is carried
// into the durable case note alongside the event identity.
func AuditedAgentManagerCallerCase(record AgentManagerCallerRecord, route GatewayRouteProof, review CallerLabelReview) (*evalv1.EvalCase, error) {
	if record.MessageRole != "user" || record.MessageSource != "operator" || record.ConversationID == "" {
		return nil, fmt.Errorf("caller record must be an operator-authored user message with a conversation identity")
	}
	expectedRequestID, err := searcheval.AgentManagerReplayRequestID(record.RunID, record.EventID, int64(record.Sequence), route.TaskRole, route.Consumer)
	if err != nil || route.GatewayRequestID != expectedRequestID {
		return nil, fmt.Errorf("Gateway request identity does not bind this Agent Manager source, task role, and consumer")
	}
	event := CallerEvent{
		RunID: record.RunID, EventID: record.EventID, Sequence: record.Sequence,
		Harness: "agent-manager", ContentClass: "user-message",
		EvidenceRef: fmt.Sprintf("agent-manager://runs/%s/events/%s", record.RunID, record.EventID),
		Query:       record.Query, SourceVerified: true,
	}
	result, err := AuditedCallerCase(event, route, review)
	if err != nil {
		return nil, err
	}
	result.Note += fmt.Sprintf(" conversation=agent-manager://conversations/%s message_role=%s message_source=%s", record.ConversationID, record.MessageRole, record.MessageSource)
	return result, nil
}

// HarvestAgentManagerCallers reads the selected real runs from Agent Manager,
// retaining only explicit operator-authored user messages. Generated prompt
// context, tool output, assistant text, and deleted messages are excluded.
func HarvestAgentManagerCallers(ctx context.Context, reader RunEventsReader, runIDs []string) ([]AgentManagerCallerRecord, error) {
	if reader == nil {
		return nil, fmt.Errorf("Agent Manager run-event reader is required")
	}
	if len(runIDs) == 0 {
		return nil, fmt.Errorf("at least one Agent Manager run ID is required")
	}
	runs := append([]string(nil), runIDs...)
	sort.Strings(runs)
	seenRuns := map[string]bool{}
	for _, runID := range runs {
		if strings.TrimSpace(runID) == "" || seenRuns[runID] {
			return nil, fmt.Errorf("run IDs must be non-empty and unique")
		}
		seenRuns[runID] = true
	}

	var result []AgentManagerCallerRecord
	for _, runID := range runs {
		// Agent Manager's first retained event may be sequence zero. Its REST
		// owner defaults an omitted cursor to -1 so sequence zero is included.
		after := int64(-1)
		allEvents := make([]AgentManagerEvent, 0)
		for {
			page, err := reader.GetRunEvents(ctx, runID, after, 1000)
			if err != nil {
				return nil, fmt.Errorf("read Agent Manager events for run %s: %w", runID, err)
			}
			if len(page.Events) == 0 && page.HasMore {
				return nil, fmt.Errorf("Agent Manager returned an empty page with has_more for run %s", runID)
			}
			for _, event := range page.Events {
				if event.RunID != runID || event.ID == "" || event.Sequence <= after {
					return nil, fmt.Errorf("Agent Manager event page has invalid run/event/sequence identity for run %s", runID)
				}
				after = event.Sequence
				allEvents = append(allEvents, event)
			}
			if !page.HasMore {
				break
			}
		}
		deleted := map[string]bool{}
		for _, event := range allEvents {
			if event.EventType == "RUN_EVENT_TYPE_MESSAGE_DELETED" && event.MessageDeleted != nil {
				deleted[event.MessageDeleted.TargetEventID] = true
			}
		}
		for _, event := range allEvents {
			message := event.Message
			if event.EventType != "RUN_EVENT_TYPE_MESSAGE" || message == nil || message.Role != "user" || message.Source != "operator" || message.EvidenceOnly || deleted[event.ID] || strings.TrimSpace(message.ConversationID) == "" || strings.TrimSpace(message.Content) == "" {
				continue
			}
			result = append(result, AgentManagerCallerRecord{
				RunID: runID, EventID: event.ID, ConversationID: message.ConversationID,
				Sequence: uint64(event.Sequence), MessageRole: message.Role, MessageSource: message.Source, Query: strings.TrimSpace(message.Content),
			})
		}
	}
	return result, nil
}
