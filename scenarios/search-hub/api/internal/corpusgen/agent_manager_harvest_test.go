package corpusgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

type fakeRunEventsReader struct {
	pages map[string][]AgentManagerEventPage
	calls map[string]int
}

func (f *fakeRunEventsReader) GetRunEvents(_ context.Context, runID string, _ int64, _ int) (AgentManagerEventPage, error) {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	index := f.calls[runID]
	f.calls[runID]++
	if index >= len(f.pages[runID]) {
		return AgentManagerEventPage{}, nil
	}
	return f.pages[runID][index], nil
}

func TestHarvestAgentManagerCallersRetainsOnlyRealOperatorUserRecords(t *testing.T) {
	run := "run-real"
	reader := &fakeRunEventsReader{pages: map[string][]AgentManagerEventPage{run: {
		{Events: []AgentManagerEvent{
			{ID: "e1", RunID: run, Sequence: 1, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "operator", ConversationID: "conv-1", Content: "Where is retry handled?"}},
			{ID: "e2", RunID: run, Sequence: 2, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "injected_context", ConversationID: "conv-1", Content: "synthetic injected prompt"}},
			{ID: "e3", RunID: run, Sequence: 3, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "assistant", Source: "operator", ConversationID: "conv-1", Content: "assistant response"}},
			{ID: "e4", RunID: run, Sequence: 4, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "operator", ConversationID: "conv-1", Content: "deleted question"}},
			{ID: "e5", RunID: run, Sequence: 5, EventType: "RUN_EVENT_TYPE_MESSAGE_DELETED", MessageDeleted: &AgentManagerMessageDeletion{TargetEventID: "e4"}},
		}},
	}}}
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{run})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d callers, want 1: %+v", len(got), got)
	}
	if got[0].RunID != run || got[0].EventID != "e1" || got[0].Sequence != 1 || got[0].ConversationID != "conv-1" || got[0].MessageRole != "user" || got[0].MessageSource != "operator" || got[0].Query != "Where is retry handled?" {
		t.Fatalf("harvest lost source identity or caller role: %+v", got[0])
	}
}

func TestHarvestAgentManagerCallersValidatesPaginationIdentity(t *testing.T) {
	reader := &fakeRunEventsReader{pages: map[string][]AgentManagerEventPage{"run-1": {{Events: []AgentManagerEvent{{ID: "wrong-run-event", RunID: "run-other", Sequence: 1}}}}}}
	if _, err := HarvestAgentManagerCallers(context.Background(), reader, []string{"run-1"}); err == nil {
		t.Fatal("expected mismatched event run identity to be rejected")
	}
}

type agentManagerEventsFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	pages map[int64]*api.GetRunEventsResponse
	calls int
}

func (f *agentManagerEventsFixture) GetRunEvents(_ context.Context, req *connect.Request[api.GetRunEventsRequest]) (*connect.Response[api.GetRunEventsResponse], error) {
	f.calls++
	if req.Msg.GetRunId() == "" || req.Msg.GetLimit() != 1000 || len(req.Msg.GetEventTypes()) != 2 || req.Msg.GetSummaryOnly() {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	response, ok := f.pages[req.Msg.GetAfterSequence()]
	if !ok {
		return connect.NewResponse(&api.GetRunEventsResponse{}), nil
	}
	return connect.NewResponse(response), nil
}

func runEvent(id, runID string, sequence int64, data any) *domain.RunEvent {
	event := &domain.RunEvent{Id: id, RunId: runID, Sequence: sequence}
	switch data := data.(type) {
	case *domain.RunEvent_Message:
		event.EventType = domain.RunEventType_RUN_EVENT_TYPE_MESSAGE
		event.Data = data
	case *domain.RunEvent_MessageDeleted:
		event.EventType = domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED
		event.Data = data
	}
	return event
}

func serveAgentManagerEventFixture(t *testing.T, fixture *agentManagerEventsFixture) AgentManagerConnectReader {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewAgentManagerConnectReader(server.URL, server.Client())
}

func TestAgentManagerConnectReaderHarvestsDeletionFilteredCaller(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	fixture := &agentManagerEventsFixture{pages: map[int64]*api.GetRunEventsResponse{
		-1: {Events: []*domain.RunEvent{
			runEvent("event-live", runID, 1, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "Where is retry handled?", ConversationId: "conversation-1"}}),
			runEvent("event-deleted", runID, 2, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "Remove this query", ConversationId: "conversation-1"}}),
			runEvent("delete-marker", runID, 3, &domain.RunEvent_MessageDeleted{MessageDeleted: &domain.MessageDeletedEventData{TargetEventId: "event-deleted"}}),
		}},
	}}
	reader := serveAgentManagerEventFixture(t, fixture)
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != "event-live" || got[0].ConversationID != "conversation-1" || got[0].MessageRole != "user" || got[0].MessageSource != "operator" || fixture.calls != 1 {
		t.Fatalf("Connect harvest = %+v calls=%d; want only the live identified operator message", got, fixture.calls)
	}
}

func TestAgentManagerConnectReaderFollowsHasMoreCursorIncludingSequenceZero(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174001"
	fixture := &agentManagerEventsFixture{pages: map[int64]*api.GetRunEventsResponse{
		-1: {Events: []*domain.RunEvent{runEvent("event-1", runID, 0, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "first query", ConversationId: "conversation-1"}})}, HasMore: true},
		0:  {Events: []*domain.RunEvent{runEvent("event-2", runID, 1, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "second query", ConversationId: "conversation-1"}})}, HasMore: false},
	}}
	reader := serveAgentManagerEventFixture(t, fixture)
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if fixture.calls != 2 || len(got) != 2 || got[0].Sequence != 0 || got[1].Sequence != 1 {
		t.Fatalf("pagination calls=%d, callers=%+v; want sequence zero followed by one across two pages", fixture.calls, got)
	}
}

func TestAuditedAgentManagerCallerCaseRetainsConversationRoleAndReviewGate(t *testing.T) {
	record := AgentManagerCallerRecord{RunID: "run-1", EventID: "event-1", Sequence: 0, ConversationID: "conversation-1", MessageRole: "user", MessageSource: "operator", Query: "where is retry handled"}
	route := gatewayRouteFixture()
	route.SourceSequence = 0
	route.GatewayRequestID, _ = searcheval.AgentManagerReplayRequestID(record.RunID, record.EventID, int64(record.Sequence), route.TaskRole, route.Consumer)
	candidate, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.GetStatus() != "candidate" || len(candidate.GetExpectIds()) != 0 || !strings.Contains(candidate.GetNote(), "source=agent-manager://runs/run-1/events/event-1 seq=0") || !strings.Contains(candidate.GetNote(), "conversation=agent-manager://conversations/conversation-1") || !strings.Contains(candidate.GetNote(), "message_role=user") {
		t.Fatalf("candidate lost source identity or entered acceptance: %+v", candidate)
	}
	reviewed, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{Reviewer: "independent", EvidenceURI: "audit://labels/1", ExpectedIDs: []string{"src/retry.go"}, IndependentlyValidated: true})
	if err != nil {
		t.Fatal(err)
	}
	counts, err := ValidateModelRoleCoverage([]*evalv1.EvalCase{candidate, reviewed}, 1)
	if err == nil || counts["chat.default"] != 1 {
		t.Fatalf("reviewed role coverage = %v, %v; want chat.default counted and other roles reported deficient", counts, err)
	}
}

func TestAuditedAgentManagerCallerCaseRejectsUnboundGatewayRequest(t *testing.T) {
	record := AgentManagerCallerRecord{RunID: "run-1", EventID: "event-1", Sequence: 5, ConversationID: "conversation-1", MessageRole: "user", MessageSource: "operator", Query: "private text omitted"}
	route := gatewayRouteFixture()
	route.SourceSequence = record.Sequence
	route.GatewayRequestID = "gateway-request-without-source-binding"
	if _, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{}); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("unbound route accepted, error = %v", err)
	}
}
