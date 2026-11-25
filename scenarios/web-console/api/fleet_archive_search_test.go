package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
)

type syntheticNodeConversationHandler struct {
	conversationconnect.UnimplementedConversationServiceHandler
	request *conversationv1.SearchArchivedRequest
}

func (h *syntheticNodeConversationHandler) SearchArchived(_ context.Context, req *connect.Request[conversationv1.SearchArchivedRequest]) (*connect.Response[conversationv1.SearchArchivedResponse], error) {
	h.request = req.Msg
	return connect.NewResponse(&conversationv1.SearchArchivedResponse{
		Matches:      []*conversationv1.ArchivedSearchMatch{{EventId: "generated-event-3", SessionId: "generated-session-3", Role: "assistant", Excerpt: "generated node archive evidence"}},
		TotalMatches: 1, DistinctSessions: 1,
	}), nil
}

func TestFleetArchiveSearchUsesTargetAwareLocalBridgeFixture(t *testing.T) {
	nodeHandler := &syntheticNodeConversationHandler{}
	_, rpc := conversationconnect.NewConversationServiceHandler(nodeHandler)
	targetPrefix := "/api/v1/targets/synthetic-node/scenarios/web-console"
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != targetPrefix+conversationconnect.ConversationServiceSearchArchivedProcedure {
			t.Errorf("target route = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer generated-owner-token" {
			t.Errorf("Bridge auth header = %q", r.Header.Get("Authorization"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		proxied := r.Clone(r.Context())
		proxied.URL.Path = strings.TrimPrefix(r.URL.Path, targetPrefix)
		rpc.ServeHTTP(w, proxied)
	}))
	defer bridge.Close()
	reach := nodereach.New(nodereach.Config{BridgeURL: bridge.URL, Token: "generated-owner-token", HTTPClient: bridge.Client()})
	result, err := searchNodeArchive(context.Background(), reach, &conversationv1.SearchNodeArchiveRequest{
		NodeId: "synthetic-node", Query: "find generated evidence", Harness: "codex", Role: "assistant", Limit: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if nodeHandler.request == nil || nodeHandler.request.GetQuery() != "find generated evidence" || nodeHandler.request.GetAgentType() != "codex" || nodeHandler.request.GetRole() != "assistant" || nodeHandler.request.GetLimit() != 8 || nodeHandler.request.GetMode() != conversationv1.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT {
		t.Fatalf("node received search request %+v", nodeHandler.request)
	}
	if len(result.GetMatches()) != 1 || result.GetMatches()[0].GetEventId() != "generated-event-3" || result.GetMatches()[0].GetSessionId() != "generated-session-3" {
		t.Fatalf("node search lost native result identity: %+v", result)
	}
}

func TestFleetArchiveSearchUnavailableNodeReturnsError(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/targets/offline-node/scenarios/web-console/") {
			t.Errorf("unexpected target path %q", r.URL.Path)
		}
		http.Error(w, "synthetic node offline", http.StatusServiceUnavailable)
	}))
	defer bridge.Close()
	reach := nodereach.New(nodereach.Config{BridgeURL: bridge.URL, Token: "generated-owner-token", HTTPClient: bridge.Client()})
	if _, err := searchNodeArchive(context.Background(), reach, &conversationv1.SearchNodeArchiveRequest{NodeId: "offline-node", Query: "generated query"}); err == nil {
		t.Fatal("offline node unexpectedly returned archive search results")
	}
}
