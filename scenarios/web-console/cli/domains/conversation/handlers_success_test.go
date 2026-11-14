package conversation

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
)

type conversationTestClient struct {
	conversationconnect.ConversationServiceClient
}

func (conversationTestClient) Get(context.Context, *connect.Request[conversationv1.GetRequest]) (*connect.Response[conversationv1.GetResponse], error) {
	return connect.NewResponse(&conversationv1.GetResponse{}), nil
}
func (conversationTestClient) UpdateCursor(context.Context, *connect.Request[conversationv1.UpdateCursorRequest]) (*connect.Response[conversationv1.UpdateCursorResponse], error) {
	return connect.NewResponse(&conversationv1.UpdateCursorResponse{}), nil
}
func (conversationTestClient) SummarizeEvent(context.Context, *connect.Request[conversationv1.SummarizeEventRequest]) (*connect.Response[conversationv1.SummarizeEventResponse], error) {
	return connect.NewResponse(&conversationv1.SummarizeEventResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/cursor.json"
	if err := os.WriteFile(body, []byte(`{"lastSeenSequence":2,"lastListenedSequence":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &handlers{client: conversationTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "since"}, {Name: "event"}, {Name: "body-file"}}}
	ctx := func(flags map[string]string) cliapp.RunContext {
		return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: flags, JSON: true})
	}
	if err := h.get(ctx(map[string]string{"session": "s1", "since": "2"})); err != nil {
		t.Fatal(err)
	}
	if err := h.cursorSet(ctx(map[string]string{"session": "s1", "body-file": body})); err != nil {
		t.Fatal(err)
	}
	if err := h.summarize(ctx(map[string]string{"session": "s1", "event": "e1"})); err != nil {
		t.Fatal(err)
	}
}
