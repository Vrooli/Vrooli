package corpusgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
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

func TestAgentManagerHTTPReaderDecodesOwnerProtoJSONAndHarvestsDeletionFilteredCaller(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/"+runID+"/events" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("event_types") != "message,message_deleted" {
			t.Errorf("event_types = %q; want both message and message_deleted", r.URL.Query().Get("event_types"))
		}
		if r.URL.Query().Get("after_sequence") != "-1" || r.URL.Query().Get("limit") != "1000" {
			t.Errorf("pagination query = %v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[{"id":"event-live","run_id":"123e4567-e89b-12d3-a456-426614174000","sequence":"1","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"Where is retry handled?","conversation_id":"conversation-1"}},{"id":"event-deleted","run_id":"123e4567-e89b-12d3-a456-426614174000","sequence":"2","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"Remove this query","conversation_id":"conversation-1"}},{"id":"delete-marker","run_id":"123e4567-e89b-12d3-a456-426614174000","sequence":"3","event_type":"RUN_EVENT_TYPE_MESSAGE_DELETED","message_deleted":{"target_event_id":"event-deleted"}}],"has_more":false}`))
	}))
	defer server.Close()

	reader := AgentManagerHTTPReader{BaseURL: server.URL, Client: server.Client()}
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != "event-live" || got[0].ConversationID != "conversation-1" || got[0].MessageRole != "user" || got[0].MessageSource != "operator" {
		t.Fatalf("owner protojson harvest = %+v; want only the live identified operator message", got)
	}
}

func TestAgentManagerHTTPReaderFollowsOwnerHasMoreCursorAcrossPages(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174001"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			if r.URL.Query().Get("after_sequence") != "-1" {
				t.Errorf("first page cursor = %q", r.URL.Query().Get("after_sequence"))
			}
			// protojson omits the proto3 sequence scalar when its value is zero.
			_, _ = w.Write([]byte(`{"events":[{"id":"event-1","run_id":"123e4567-e89b-12d3-a456-426614174001","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"first query","conversation_id":"conversation-1"}}],"has_more":true}`))
			return
		}
		if r.URL.Query().Get("after_sequence") != "0" {
			t.Errorf("second page cursor = %q; want 0", r.URL.Query().Get("after_sequence"))
		}
		_, _ = w.Write([]byte(`{"events":[{"id":"event-2","run_id":"123e4567-e89b-12d3-a456-426614174001","sequence":"1","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"second query","conversation_id":"conversation-1"}}],"has_more":false}`))
	}))
	defer server.Close()

	got, err := HarvestAgentManagerCallers(context.Background(), AgentManagerHTTPReader{BaseURL: server.URL, Client: server.Client()}, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(got) != 2 || got[0].Sequence != 0 || got[1].Sequence != 1 {
		t.Fatalf("pagination requests=%d, callers=%+v; want sequence zero followed by one across two pages", requests, got)
	}
}

func TestAuditedAgentManagerCallerCaseRetainsConversationRoleAndReviewGate(t *testing.T) {
	record := AgentManagerCallerRecord{RunID: "run-1", EventID: "event-1", Sequence: 0, ConversationID: "conversation-1", MessageRole: "user", MessageSource: "operator", Query: "where is retry handled"}
	route := gatewayRouteFixture()
	route.SourceSequence = 0
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
