package corpusgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
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
	ID              string                       `json:"id"`
	RunID           string                       `json:"run_id"`
	Sequence        int64                        `json:"-"`
	SequenceOmitted bool                         `json:"-"`
	EventType       string                       `json:"event_type"`
	Message         *AgentManagerMessage         `json:"message"`
	MessageDeleted  *AgentManagerMessageDeletion `json:"message_deleted"`
}

// UnmarshalJSON matches Agent Manager's protojson contract: snake_case field
// names, int64 sequence encoded as a JSON string, and oneof alternatives at
// the RunEvent object level (message/message_deleted).
func (e *AgentManagerEvent) UnmarshalJSON(data []byte) error {
	type eventJSON struct {
		ID             string                       `json:"id"`
		RunID          string                       `json:"run_id"`
		Sequence       json.RawMessage              `json:"sequence"`
		EventType      string                       `json:"event_type"`
		Message        *AgentManagerMessage         `json:"message"`
		MessageDeleted *AgentManagerMessageDeletion `json:"message_deleted"`
	}
	var decoded eventJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	sequence := strings.TrimSpace(string(decoded.Sequence))
	if sequence == "" {
		*e = AgentManagerEvent{ID: decoded.ID, RunID: decoded.RunID, EventType: decoded.EventType, Message: decoded.Message, MessageDeleted: decoded.MessageDeleted, SequenceOmitted: true}
		return nil
	}
	if len(sequence) >= 2 && sequence[0] == '"' && sequence[len(sequence)-1] == '"' {
		var value string
		if err := json.Unmarshal(decoded.Sequence, &value); err != nil {
			return err
		}
		sequence = value
	}
	parsed, err := strconv.ParseInt(sequence, 10, 64)
	if err != nil {
		return fmt.Errorf("decode Agent Manager event sequence: %w", err)
	}
	*e = AgentManagerEvent{ID: decoded.ID, RunID: decoded.RunID, Sequence: parsed, EventType: decoded.EventType, Message: decoded.Message, MessageDeleted: decoded.MessageDeleted}
	return nil
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

// AgentManagerHTTPReader implements the owner's bounded REST event projection.
// Supply the authenticated HTTP client and the Agent Manager base URL from the
// scenario's normal service wiring; this type does not manage credentials.
type AgentManagerHTTPReader struct {
	BaseURL string
	Client  *http.Client
}

func (r AgentManagerHTTPReader) GetRunEvents(ctx context.Context, runID string, after int64, limit int) (AgentManagerEventPage, error) {
	var page AgentManagerEventPage
	if r.Client == nil || strings.TrimSpace(r.BaseURL) == "" {
		return page, fmt.Errorf("Agent Manager base URL and authenticated HTTP client are required")
	}
	base, err := url.Parse(strings.TrimRight(r.BaseURL, "/") + "/api/v1/runs/" + url.PathEscape(runID) + "/events")
	if err != nil {
		return page, fmt.Errorf("parse Agent Manager URL: %w", err)
	}
	query := base.Query()
	query.Set("after_sequence", fmt.Sprint(after))
	query.Set("limit", fmt.Sprint(limit))
	query.Set("event_types", "message,message_deleted")
	base.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return page, fmt.Errorf("build Agent Manager event request: %w", err)
	}
	response, err := r.Client.Do(req)
	if err != nil {
		return page, fmt.Errorf("request Agent Manager events: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return page, fmt.Errorf("Agent Manager event request returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		return page, fmt.Errorf("decode Agent Manager event page: %w", err)
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
				if event.SequenceOmitted {
					// protojson omits proto3 scalar defaults. Agent Manager starts
					// run event sequence at zero, so only the first retained item
					// on the initial (-1) cursor may lawfully omit sequence.
					if after != -1 || len(allEvents) != 0 {
						return nil, fmt.Errorf("Agent Manager event page omitted a non-initial sequence for run %s", runID)
					}
					event.Sequence = 0
				}
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
