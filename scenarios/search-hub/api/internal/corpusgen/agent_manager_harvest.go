package corpusgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type AgentManagerCompletedRunReader interface {
	GetRun(context.Context, string) (*domain.Run, error)
}

type AgentManagerCodeTaskEventReader interface {
	GetCodeTaskSourceMessage(context.Context, CodeTaskSourceRef) (AgentManagerEvent, error)
	IsCodeTaskSourceDeleted(context.Context, string, string) (bool, error)
}

// AgentManagerCodeTaskSourceVerifier proves that a candidate names one exact
// operator-authored message in a completed run. Message bodies are read only
// from the selected completed run, hashed in memory, and never returned or
// persisted by this verifier.
type AgentManagerCodeTaskSourceVerifier struct {
	Runs   AgentManagerCompletedRunReader
	Events AgentManagerCodeTaskEventReader
}

func (v AgentManagerCodeTaskSourceVerifier) VerifyCodeTaskSource(ctx context.Context, source CodeTaskSourceRef) error {
	if v.Runs == nil || v.Events == nil {
		return fmt.Errorf("Agent Manager completed-run and event owners are required")
	}
	run, err := v.Runs.GetRun(ctx, source.RunID)
	if err != nil {
		return fmt.Errorf("read Agent Manager run owner: %w", err)
	}
	if run == nil || run.GetId() != source.RunID || run.GetStatus() != domain.RunStatus_RUN_STATUS_COMPLETE || source.RunStatus != "complete" {
		return fmt.Errorf("source is not from the exact completed Agent Manager run")
	}
	event, err := v.Events.GetCodeTaskSourceMessage(ctx, source)
	if err != nil {
		return fmt.Errorf("read exact Agent Manager source message: %w", err)
	}
	deleted, err := v.Events.IsCodeTaskSourceDeleted(ctx, source.RunID, source.EventID)
	if err != nil {
		return fmt.Errorf("verify Agent Manager source deletion state: %w", err)
	}
	if deleted || event.ID != source.EventID || event.RunID != source.RunID || event.Sequence < 0 || uint64(event.Sequence) != source.Sequence || event.Message == nil {
		return fmt.Errorf("exact Agent Manager source event is missing, deleted, or mismatched")
	}
	message := event.Message
	if message.Role != "user" || message.Source != "operator" || message.EvidenceOnly || message.ConversationID != source.ConversationID || source.MessageRole != "user" || source.MessageSource != "operator" || source.ActorID != "operator" {
		return fmt.Errorf("Agent Manager source is not the exact operator-authored user message")
	}
	digest := sha256.Sum256([]byte(message.Content))
	if strings.TrimSpace(message.Content) == "" || hex.EncodeToString(digest[:]) != strings.ToLower(source.MessageSHA256) {
		return fmt.Errorf("Agent Manager source body does not match its pinned SHA-256")
	}
	return nil
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

func (r AgentManagerConnectReader) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	if r.Client == nil || strings.TrimSpace(r.BaseURL) == "" || strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("Agent Manager Connect client and run ID are required")
	}
	response, err := r.Client.GetRun(ctx, connect.NewRequest(&api.GetRunRequest{RunId: runID}))
	if err != nil {
		return nil, fmt.Errorf("request Agent Manager run owner: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetRun() == nil || response.Msg.GetRun().GetId() != runID {
		return nil, fmt.Errorf("Agent Manager returned a mismatched run identity")
	}
	return response.Msg.GetRun(), nil
}

func (r AgentManagerConnectReader) GetCodeTaskSourceMessage(ctx context.Context, source CodeTaskSourceRef) (AgentManagerEvent, error) {
	if r.Client == nil || strings.TrimSpace(source.RunID) == "" || strings.TrimSpace(source.EventID) == "" || source.Sequence > uint64(1<<63-1) {
		return AgentManagerEvent{}, fmt.Errorf("Agent Manager Connect client and exact source identity are required")
	}
	after := int64(source.Sequence) - 1
	limit := int32(1)
	response, err := r.Client.GetRunEvents(ctx, connect.NewRequest(&api.GetRunEventsRequest{
		RunId: source.RunID, AfterSequence: &after, Limit: &limit,
		EventTypes: []domain.RunEventType{domain.RunEventType_RUN_EVENT_TYPE_MESSAGE},
	}))
	if err != nil {
		return AgentManagerEvent{}, fmt.Errorf("request exact Agent Manager source message: %w", err)
	}
	if response == nil || response.Msg == nil || len(response.Msg.GetEvents()) != 1 {
		return AgentManagerEvent{}, fmt.Errorf("Agent Manager did not return one exact source message")
	}
	event := response.Msg.GetEvents()[0]
	if event.GetId() != source.EventID || event.GetRunId() != source.RunID || event.GetSequence() != int64(source.Sequence) || event.GetEventType() != domain.RunEventType_RUN_EVENT_TYPE_MESSAGE || event.GetMessage() == nil {
		return AgentManagerEvent{}, fmt.Errorf("Agent Manager source message identity mismatch")
	}
	message := event.GetMessage()
	return AgentManagerEvent{ID: event.GetId(), RunID: event.GetRunId(), Sequence: event.GetSequence(), EventType: event.GetEventType().String(), Message: &AgentManagerMessage{
		Role: message.GetRole(), Source: message.GetSource(), Content: message.GetContent(), ConversationID: message.GetConversationId(), EvidenceOnly: message.GetEvidenceOnly(),
	}}, nil
}

func (r AgentManagerConnectReader) IsCodeTaskSourceDeleted(ctx context.Context, runID, eventID string) (bool, error) {
	if r.Client == nil || strings.TrimSpace(runID) == "" || strings.TrimSpace(eventID) == "" {
		return false, fmt.Errorf("Agent Manager Connect client and source identity are required")
	}
	const pageSize = 1000
	const maxDeletionEvents = 20000
	after := int64(-1)
	count := 0
	for {
		limit := int32(pageSize)
		response, err := r.Client.GetRunEvents(ctx, connect.NewRequest(&api.GetRunEventsRequest{
			RunId: runID, AfterSequence: &after, Limit: &limit,
			EventTypes: []domain.RunEventType{domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED},
		}))
		if err != nil {
			return false, fmt.Errorf("request Agent Manager source deletion metadata: %w", err)
		}
		if response == nil || response.Msg == nil || len(response.Msg.GetEvents()) == 0 && response.Msg.GetHasMore() {
			return false, fmt.Errorf("Agent Manager returned an invalid source deletion page")
		}
		for _, event := range response.Msg.GetEvents() {
			if event.GetRunId() != runID || event.GetId() == "" || event.GetSequence() <= after || event.GetEventType() != domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED || event.GetMessageDeleted() == nil {
				return false, fmt.Errorf("Agent Manager returned invalid source deletion identity")
			}
			after = event.GetSequence()
			count++
			if count > maxDeletionEvents {
				return false, fmt.Errorf("Agent Manager run exceeds the %d-deletion verification bound", maxDeletionEvents)
			}
			if event.GetMessageDeleted().GetTargetEventId() == eventID {
				return true, nil
			}
		}
		if !response.Msg.GetHasMore() {
			return false, nil
		}
	}
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
	runsReader, ok := reader.(AgentManagerCompletedRunReader)
	if !ok {
		return nil, fmt.Errorf("Agent Manager completed-run owner is required; refusing to read run transcripts without completion proof")
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
		run, err := runsReader.GetRun(ctx, runID)
		if err != nil {
			return nil, fmt.Errorf("verify Agent Manager run %s completion: %w", runID, err)
		}
		if run == nil || run.GetId() != runID || run.GetStatus() != domain.RunStatus_RUN_STATUS_COMPLETE {
			return nil, fmt.Errorf("refusing to read Agent Manager transcript %s unless the exact run is complete", runID)
		}
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
