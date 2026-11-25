package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
	conversationH "web-console/handlers/conversation"
)

type nodeScenarioConnector interface {
	ScenarioURL(context.Context, string, string) (string, error)
	ConnectTransport(context.Context, string) connect.HTTPClient
}

func (a *conversationAdapter) SearchNodeArchive(ctx context.Context, request *conversationv1.SearchNodeArchiveRequest) (*conversationv1.SearchNodeArchiveResponse, error) {
	nodeID := strings.TrimSpace(request.GetNodeId())
	query := strings.TrimSpace(request.GetQuery())
	if nodeID == "" || len(nodeID) > 128 || query == "" {
		return nil, conversationH.ErrInvalidArgument
	}
	limit := request.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	owner, _ := resolveBridgeOwnerCredentials()
	reach := nodereach.New(nodereach.Config{
		ResolveBridgeURL: resolveBridgeScenarioURL,
		TokenProvider: func(context.Context) (string, error) {
			if strings.TrimSpace(owner) == "" {
				return "", errors.New("Bridge owner credential unavailable")
			}
			return owner, nil
		},
		HTTPClient: &http.Client{Timeout: 12 * time.Second},
	})
	search, err := searchNodeArchive(ctx, reach, &conversationv1.SearchNodeArchiveRequest{
		NodeId: nodeID, Query: query, Harness: request.GetHarness(), Role: request.GetRole(), Limit: limit,
	})
	if err != nil {
		return &conversationv1.SearchNodeArchiveResponse{
			NodeId: nodeID, Available: false, Degradation: "The selected node's Web Console archive search is unavailable",
		}, nil
	}
	return &conversationv1.SearchNodeArchiveResponse{NodeId: nodeID, Available: true, Search: search}, nil
}

// searchNodeArchive sends only a query over the authenticated, target-aware
// node channel. Transcript content returns with native node/session/event
// identity; the local machine's archive remains untouched.
func searchNodeArchive(ctx context.Context, connector nodeScenarioConnector, request *conversationv1.SearchNodeArchiveRequest) (*conversationv1.SearchArchivedResponse, error) {
	if connector == nil || strings.TrimSpace(request.GetNodeId()) == "" || strings.TrimSpace(request.GetQuery()) == "" {
		return nil, errors.New("node connector, node id and query are required")
	}
	baseURL, err := connector.ScenarioURL(ctx, strings.TrimSpace(request.GetNodeId()), "web-console")
	if err != nil {
		return nil, err
	}
	client := conversationconnect.NewConversationServiceClient(connector.ConnectTransport(ctx, baseURL), baseURL)
	response, err := client.SearchArchived(ctx, connect.NewRequest(&conversationv1.SearchArchivedRequest{
		Query: strings.TrimSpace(request.GetQuery()), AgentType: strings.TrimSpace(request.GetHarness()),
		Role: strings.TrimSpace(request.GetRole()), Limit: request.GetLimit(),
		Mode: conversationv1.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT,
	}))
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil {
		return nil, errors.New("node returned an empty archive search response")
	}
	return response.Msg, nil
}
