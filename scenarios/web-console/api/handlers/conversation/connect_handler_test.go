package conversation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
)

type fakeConversationService struct{ err error }

func (f fakeConversationService) Get(context.Context, string, int64, int, int64) (SessionState, error) {
	return SessionState{SessionID: "s1", Events: []Event{{ID: "e1", SessionID: "s1", Role: "assistant", Text: "hello", SpeechParagraphs: []string{"hello"}, OriginalSpeechParagraphs: []string{"hello"}, Sequence: 2, DeliveryState: "complete", TTSState: "ready", ConsumptionState: "new"}}, Cursor: Cursor{LastSeenSequence: 1}, HasMore: true, OldestSequence: 1, NewestSequence: 2, TotalCount: 2}, f.err
}

func (f fakeConversationService) CaptureStatus(context.Context, string) CaptureStatus {
	return CaptureStatus{State: CaptureCapturing, Summary: "Messages are being captured."}
}

func (f fakeConversationService) Search(context.Context, string, SearchQuery) (SearchResult, error) {
	return SearchResult{Matches: []SearchMatch{{EventID: "e1", Sequence: 2, Excerpt: "hello"}}, Truncated: true, TotalMatches: 1}, f.err
}

func (f fakeConversationService) SearchArchived(context.Context, ArchivedSearchFilter) (ArchivedSearchResult, error) {
	return ArchivedSearchResult{Matches: []ArchivedSearchMatch{{EventID: "e1", SessionID: "s1", Sequence: 2, Role: "assistant", CreatedAt: "now", Excerpt: "hello"}}, Truncated: true, TotalMatches: 1, DistinctSessions: 1}, f.err
}

func (f fakeConversationService) GetRange(context.Context, string, int64, int64) (SessionState, error) {
	return SessionState{SessionID: "s1", Cursor: Cursor{LastListenedSequence: 2}}, f.err
}

func (f fakeConversationService) Project(_ context.Context, _ string, query ProjectionQuery) (SessionState, error) {
	work := &WorkSpan{FromSequence: 1, ToSequence: 3, ToolCalls: 2, ToolResults: 1, StartedAt: "2026-11-26T10:00:00Z", EndedAt: "2026-11-26T10:00:20Z"}
	return SessionState{SessionID: "s1", Events: []Event{
		{ID: "work:" + query.ThreadID + ":1", SessionID: "s1", Role: "assistant", Kind: "work", ThreadID: query.ThreadID, Sequence: 3, Work: work},
		{ID: "e4", SessionID: "s1", Role: "assistant", Kind: "message", Author: "agent", ThreadID: query.ThreadID, Text: "done", Sequence: 4},
	}, Threads: []Thread{{ThreadID: "01a217a4", Kind: "subagent", Label: "Hilbert", Description: "/root/sc9_owner", FirstSequence: 2, LastSequence: 9, MessageCount: 2, WorkCount: 5}}, OldestSequence: 1, NewestSequence: 4}, f.err
}

func (f fakeConversationService) WorkRange(_ context.Context, _ string, threadID string, from, _ int64) (SessionState, error) {
	return SessionState{SessionID: "s1", Events: []Event{{ID: "t1", SessionID: "s1", Role: "assistant", Kind: "tool_call", Label: "Bash", ThreadID: threadID, Text: "[Tool call: Bash]", Sequence: from, TextTruncated: true}}}, f.err
}

func (f fakeConversationService) ResolveEvents(_ context.Context, _ string, ids []string) (ResolveEventsResult, error) {
	return ResolveEventsResult{Events: []Event{{ID: "e1", SessionID: "s1", Role: "assistant", Text: "hello", Sequence: 2}}, MissingEventIDs: []string{"gone"}}, f.err
}

func (f fakeConversationService) UpdateCursor(context.Context, string, CursorPatch) (Cursor, error) {
	return Cursor{LastSeenSequence: 3}, f.err
}

func (f fakeConversationService) SummarizeEvent(context.Context, string, string) (SummarizeResult, error) {
	return SummarizeResult{Summarized: true, SpeechParagraphs: []string{"short"}}, f.err
}

type fakeOptionalSearchService struct{}

func (fakeOptionalSearchService) SearchSemantic(_ context.Context, request *conversationv1.SearchSemanticRequest) (*conversationv1.SearchSemanticResponse, error) {
	return &conversationv1.SearchSemanticResponse{
		Local:             &conversationv1.SearchArchivedResponse{Matches: []*conversationv1.ArchivedSearchMatch{{EventId: "local-e1"}}},
		SemanticHits:      []*conversationv1.SemanticSearchHit{{StableHitId: "am-hit-1", Harness: "codex", SourceSessionId: "native-1"}},
		SemanticAvailable: true,
	}, nil
}

func (fakeOptionalSearchService) SearchNodeArchive(_ context.Context, request *conversationv1.SearchNodeArchiveRequest) (*conversationv1.SearchNodeArchiveResponse, error) {
	return &conversationv1.SearchNodeArchiveResponse{
		NodeId: request.GetNodeId(), Available: true,
		Search: &conversationv1.SearchArchivedResponse{Matches: []*conversationv1.ArchivedSearchMatch{{EventId: "remote-e1", SessionId: "remote-s1"}}},
	}, nil
}

func TestOptionalSearchConnectRPCsAndDegradation(t *testing.T) {
	_, handler := conversationconnect.NewConversationServiceHandler(NewConnectHandler(Deps{Service: fakeConversationService{}}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	defer server.Close()
	client := conversationconnect.NewConversationServiceClient(server.Client(), server.URL)
	localOnly, err := client.SearchSemantic(context.Background(), connect.NewRequest(&conversationv1.SearchSemanticRequest{Query: "generated"}))
	if err != nil || len(localOnly.Msg.GetLocal().GetMatches()) != 1 || localOnly.Msg.GetLocal().GetMatches()[0].GetEventId() != "e1" || localOnly.Msg.GetSemanticAvailable() {
		t.Fatalf("local-only semantic fallback = %+v, %v", localOnly, err)
	}
	withoutNode, err := client.SearchNodeArchive(context.Background(), connect.NewRequest(&conversationv1.SearchNodeArchiveRequest{NodeId: "offline", Query: "generated"}))
	if err != nil || withoutNode.Msg.GetAvailable() || withoutNode.Msg.GetDegradation() == "" {
		t.Fatalf("node absence response = %+v, %v", withoutNode, err)
	}

	_, handler = conversationconnect.NewConversationServiceHandler(NewConnectHandler(Deps{Service: fakeConversationService{}, OptionalSearch: fakeOptionalSearchService{}}))
	serverWithOptional := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	defer serverWithOptional.Close()
	clientWithOptional := conversationconnect.NewConversationServiceClient(serverWithOptional.Client(), serverWithOptional.URL)
	semantic, err := clientWithOptional.SearchSemantic(context.Background(), connect.NewRequest(&conversationv1.SearchSemanticRequest{Query: "generated"}))
	if err != nil || len(semantic.Msg.GetSemanticHits()) != 1 || semantic.Msg.GetSemanticHits()[0].GetSourceSessionId() != "native-1" {
		t.Fatalf("semantic Connect response = %+v, %v", semantic, err)
	}
	node, err := clientWithOptional.SearchNodeArchive(context.Background(), connect.NewRequest(&conversationv1.SearchNodeArchiveRequest{NodeId: "synthetic-node", Query: "generated"}))
	if err != nil || !node.Msg.GetAvailable() || node.Msg.GetNodeId() != "synthetic-node" || node.Msg.GetSearch().GetMatches()[0].GetEventId() != "remote-e1" {
		t.Fatalf("node Connect response = %+v, %v", node, err)
	}
}

func TestConnectHandlerConversationOperations(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeConversationService{}})
	ctx := context.Background()
	get, err := h.Get(ctx, connect.NewRequest(&conversationv1.GetRequest{SessionId: "s1", Limit: 10}))
	if err != nil || len(get.Msg.Events) != 1 || get.Msg.Events[0].Id != "e1" || get.Msg.Cursor.LastSeenSequence != 1 {
		t.Fatalf("get: %#v %v", get, err)
	}
	search, err := h.Search(ctx, connect.NewRequest(&conversationv1.SearchRequest{SessionId: "s1", Query: "hello", Limit: 10}))
	if err != nil || len(search.Msg.Matches) != 1 || !search.Msg.Truncated {
		t.Fatalf("search: %#v %v", search, err)
	}
	arch, err := h.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{Query: "hello", AgentType: "codex", Role: "assistant"}))
	if err != nil || len(arch.Msg.Matches) != 1 || arch.Msg.DistinctSessions != 1 {
		t.Fatalf("archived: %#v %v", arch, err)
	}
	empty, err := h.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{}))
	if err != nil || len(empty.Msg.Matches) != 0 {
		t.Fatalf("empty archived: %#v %v", empty, err)
	}
	if _, err := h.GetRange(ctx, connect.NewRequest(&conversationv1.GetRangeRequest{SessionId: "s1", FromSequence: 1, ToSequence: 2})); err != nil {
		t.Fatal(err)
	}
	resolved, err := h.ResolveEvents(ctx, connect.NewRequest(&conversationv1.ResolveEventsRequest{SessionId: "s1", EventIds: []string{"e1", "gone"}}))
	if err != nil || len(resolved.Msg.Events) != 1 || resolved.Msg.Events[0].Id != "e1" || len(resolved.Msg.MissingEventIds) != 1 {
		t.Fatalf("resolve events: %#v %v", resolved, err)
	}
	if _, err := h.UpdateCursor(ctx, connect.NewRequest(&conversationv1.UpdateCursorRequest{SessionId: "s1", HasLastSeenSequence: true, LastSeenSequence: 3})); err != nil {
		t.Fatal(err)
	}
	if resp, err := h.SummarizeEvent(ctx, connect.NewRequest(&conversationv1.SummarizeEventRequest{SessionId: "s1", EventId: "e1"})); err != nil || !resp.Msg.Summarized {
		t.Fatalf("summary: %#v %v", resp, err)
	}
}

func TestConnectHandlerMessagesProjectionAndWork(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeConversationService{}})
	ctx := context.Background()
	projected, err := h.Get(ctx, connect.NewRequest(&conversationv1.GetRequest{
		SessionId: "s1", Limit: 50, Projection: conversationv1.ConversationProjection_CONVERSATION_PROJECTION_MESSAGES,
	}))
	if err != nil {
		t.Fatal(err)
	}
	events := projected.Msg.GetEvents()
	if len(events) != 2 || events[0].GetKind() != "work" || events[0].GetWork().GetToolCalls() != 2 || events[0].GetWork().GetFromSequence() != 1 {
		t.Fatalf("projection must carry the work row first: %#v", events)
	}
	if events[1].GetKind() != "message" || events[1].GetAuthor() != "agent" {
		t.Fatalf("conversation row: %#v", events[1])
	}
	threads := projected.Msg.GetThreads()
	if len(threads) != 1 || threads[0].GetLabel() != "Hilbert" || threads[0].GetMessageCount() != 2 || threads[0].GetFirstSequence() != 2 {
		t.Fatalf("threads: %#v", threads)
	}
	work, err := h.GetRange(ctx, connect.NewRequest(&conversationv1.GetRangeRequest{
		SessionId: "s1", FromSequence: 5, ToSequence: 9, ThreadId: "01a217a4", Projection: conversationv1.ConversationProjection_CONVERSATION_PROJECTION_WORK,
	}))
	if err != nil || len(work.Msg.GetEvents()) != 1 || work.Msg.GetEvents()[0].GetLabel() != "Bash" || !work.Msg.GetEvents()[0].GetTextTruncated() || work.Msg.GetEvents()[0].GetThreadId() != "01a217a4" {
		t.Fatalf("work range: %#v %v", work, err)
	}
}

func TestConnectHandlerConversationValidationAndErrors(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeConversationService{}})
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, e := h.Get(ctx, connect.NewRequest(&conversationv1.GetRequest{})); return e },
		func() error {
			_, e := h.UpdateCursor(ctx, connect.NewRequest(&conversationv1.UpdateCursorRequest{}))
			return e
		},
		func() error {
			_, e := h.SummarizeEvent(ctx, connect.NewRequest(&conversationv1.SummarizeEventRequest{SessionId: "s"}))
			return e
		},
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("got %v", err)
		}
	}
	for _, tc := range []struct {
		in   error
		want connect.Code
	}{{ErrSessionNotFound, connect.CodeNotFound}, {ErrNotFound, connect.CodeNotFound}, {ErrInvalidArgument, connect.CodeInvalidArgument}, {errors.New("x"), connect.CodeInternal}} {
		var ce *connect.Error
		if err := h.classify(tc.in, "test"); !errors.As(err, &ce) || ce.Code() != tc.want {
			t.Errorf("%v: got %v want %v", tc.in, err, tc.want)
		}
	}
}

func TestConversationEventProtoCarriesSourceOccurrenceTime(t *testing.T) {
	event := eventToProto(Event{ID: "e1", CreatedAt: "ingested", OccurredAt: "source-time"})
	if event.GetCreatedAt() != "ingested" || event.GetOccurredAt() != "source-time" {
		t.Fatalf("event timestamps created=%q occurred=%q", event.GetCreatedAt(), event.GetOccurredAt())
	}
}
