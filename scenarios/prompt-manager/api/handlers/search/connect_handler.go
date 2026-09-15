package search

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"connectrpc.com/connect"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/search/search_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/search"
)

type connectHandler struct {
	searchconnect.UnimplementedSearchServiceHandler
	legacy *domain.Handlers
}

// NewConnectMount exposes deterministic search through the generated contract.
// The legacy handlers remain an in-process adapter only; their REST routes are
// deliberately not mounted.
func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return searchconnect.NewSearchServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) SearchSkills(ctx context.Context, req *connect.Request[searchv1.SearchSkillsRequest]) (*connect.Response[searchv1.SearchSkillsResponse], error) {
	q := url.Values{"q": {req.Msg.GetQuery()}, "tag": {req.Msg.GetTag()}, "folder": {req.Msg.GetFolder()}}
	return invokeSearch(ctx, req.Header(), h.legacy.Search, "/search/skills?"+q.Encode(), &searchv1.SearchSkillsResponse{})
}

func (h *connectHandler) SearchSkillContent(ctx context.Context, req *connect.Request[searchv1.SearchSkillContentRequest]) (*connect.Response[searchv1.SearchSkillContentResponse], error) {
	q := contentQuery(req.Msg.GetQuery(), req.Msg.GetTags(), req.Msg.GetCaseSensitive(), req.Msg.GetWholeWord(), req.Msg.GetRegex(), req.Msg.GetLimit())
	for _, folder := range req.Msg.GetFolders() {
		q.Add("folder", folder)
	}
	return invokeSearch(ctx, req.Header(), h.legacy.ContentSearch, "/search/skills/content?"+q.Encode(), &searchv1.SearchSkillContentResponse{})
}

func (h *connectHandler) SearchAgents(ctx context.Context, req *connect.Request[searchv1.SearchAgentsRequest]) (*connect.Response[searchv1.SearchAgentsResponse], error) {
	q := url.Values{"q": {req.Msg.GetQuery()}, "tag": {req.Msg.GetTag()}, "status": {req.Msg.GetStatus()}}
	return invokeSearch(ctx, req.Header(), h.legacy.SearchAgents, "/search/agents?"+q.Encode(), &searchv1.SearchAgentsResponse{})
}

func (h *connectHandler) SearchAgentContent(ctx context.Context, req *connect.Request[searchv1.SearchAgentContentRequest]) (*connect.Response[searchv1.SearchAgentContentResponse], error) {
	q := contentQuery(req.Msg.GetQuery(), req.Msg.GetTags(), req.Msg.GetCaseSensitive(), req.Msg.GetWholeWord(), req.Msg.GetRegex(), req.Msg.GetLimit())
	return invokeSearch(ctx, req.Header(), h.legacy.AgentContentSearch, "/search/agents/content?"+q.Encode(), &searchv1.SearchAgentContentResponse{})
}

func (h *connectHandler) SearchTeams(ctx context.Context, req *connect.Request[searchv1.SearchTeamsRequest]) (*connect.Response[searchv1.SearchTeamsResponse], error) {
	q := url.Values{"q": {req.Msg.GetQuery()}}
	if req.Msg.Enabled != nil {
		q.Set("enabled", strconv.FormatBool(req.Msg.GetEnabled()))
	}
	return invokeSearch(ctx, req.Header(), h.legacy.SearchTeams, "/search/teams?"+q.Encode(), &searchv1.SearchTeamsResponse{})
}

func (h *connectHandler) SearchTeamContent(ctx context.Context, req *connect.Request[searchv1.SearchTeamContentRequest]) (*connect.Response[searchv1.SearchTeamContentResponse], error) {
	q := contentQuery(req.Msg.GetQuery(), nil, req.Msg.GetCaseSensitive(), req.Msg.GetWholeWord(), req.Msg.GetRegex(), req.Msg.GetLimit())
	return invokeSearch(ctx, req.Header(), h.legacy.TeamContentSearch, "/search/teams/content?"+q.Encode(), &searchv1.SearchTeamContentResponse{})
}

func contentQuery(query string, tags []string, caseSensitive, wholeWord, regex bool, limit int32) url.Values {
	q := url.Values{"q": {query}}
	for _, tag := range tags {
		q.Add("tag", tag)
	}
	if caseSensitive {
		q.Set("caseSensitive", "true")
	}
	if wholeWord {
		q.Set("wholeWord", "true")
	}
	if regex {
		q.Set("regex", "true")
	}
	if limit > 0 {
		q.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	return q
}

func invokeSearch[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, path string, out *T) (*connect.Response[T], error) {
	return transportbridge.InvokeProto(ctx, headers, handler, http.MethodGet, path, nil, out)
}
