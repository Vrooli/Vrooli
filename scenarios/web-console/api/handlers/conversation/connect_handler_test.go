package conversation

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
)

type fakeConversationService struct{ err error }

func (f fakeConversationService) Get(string, int64, int, int64) (SessionState, error) {
	return SessionState{SessionID: "s1", Events: []Event{{ID: "e1", SessionID: "s1", Role: "assistant", Text: "hello", SpeechParagraphs: []string{"hello"}, OriginalSpeechParagraphs: []string{"hello"}, Sequence: 2, DeliveryState: "complete", TTSState: "ready", ConsumptionState: "new"}}, Cursor: Cursor{LastSeenSequence: 1}, HasMore: true, OldestSequence: 1, NewestSequence: 2, TotalCount: 2}, f.err
}
func (f fakeConversationService) Search(string, string, int) ([]SearchMatch, bool, int64, error) {
	return []SearchMatch{{EventID: "e1", Sequence: 2, Excerpt: "hello"}}, true, 1, f.err
}
func (f fakeConversationService) SearchArchived(context.Context, ArchivedSearchFilter) (ArchivedSearchResult, error) {
	return ArchivedSearchResult{Matches: []ArchivedSearchMatch{{EventID: "e1", SessionID: "s1", Sequence: 2, Role: "assistant", CreatedAt: "now", Excerpt: "hello"}}, Truncated: true, TotalMatches: 1, DistinctSessions: 1}, f.err
}
func (f fakeConversationService) GetRange(string, int64, int64) (SessionState, error) {
	return SessionState{SessionID: "s1", Cursor: Cursor{LastListenedSequence: 2}}, f.err
}
func (f fakeConversationService) UpdateCursor(string, CursorPatch) (Cursor, error) {
	return Cursor{LastSeenSequence: 3}, f.err
}
func (f fakeConversationService) SummarizeEvent(context.Context, string, string) (SummarizeResult, error) {
	return SummarizeResult{Summarized: true, SpeechParagraphs: []string{"short"}}, f.err
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
	if _, err := h.UpdateCursor(ctx, connect.NewRequest(&conversationv1.UpdateCursorRequest{SessionId: "s1", HasLastSeenSequence: true, LastSeenSequence: 3})); err != nil {
		t.Fatal(err)
	}
	if resp, err := h.SummarizeEvent(ctx, connect.NewRequest(&conversationv1.SummarizeEventRequest{SessionId: "s1", EventId: "e1"})); err != nil || !resp.Msg.Summarized {
		t.Fatalf("summary: %#v %v", resp, err)
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
