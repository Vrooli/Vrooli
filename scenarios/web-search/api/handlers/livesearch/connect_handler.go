package livesearch

import (
	"context"
	"log"

	"connectrpc.com/connect"

	livesearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch"

	internallivesearch "web-search/internal/livesearch"
)

// Deps wires the seams the Connect live-search handler needs. The service owns
// the SearXNG client, cache, governor, and synthesizer; this handler only
// translates wire <-> domain.
type Deps struct {
	Service *internallivesearch.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect live-search handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Search runs the L0 live web search (and optional L1 synthesis) and projects
// the domain outcome onto the wire response. A degraded outcome (budget
// exhausted) is a successful response with degraded=true and empty results —
// not a Connect error — so callers can fall back gracefully.
func (h *connectHandler) Search(ctx context.Context, req *connect.Request[livesearchv1.SearchRequest]) (*connect.Response[livesearchv1.SearchResponse], error) {
	outcome, err := h.deps.Service.Search(ctx, internallivesearch.SearchInput{
		Query:      req.Msg.GetQuery(),
		Limit:      int(req.Msg.GetLimit()),
		Synthesize: req.Msg.GetSynthesize(),
	})
	if err != nil {
		h.deps.Logger.Printf("livesearch.Search: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &livesearchv1.SearchResponse{
		Results:         make([]*livesearchv1.SearchResult, 0, len(outcome.Results)),
		Cached:          outcome.Cached,
		Degraded:        outcome.Degraded,
		DegradedReason:  outcome.DegradedReason,
		DegradedEngines: engineIssuesToProto(outcome.DegradedEngines),
	}
	for _, r := range outcome.Results {
		resp.Results = append(resp.Results, resultToProto(r))
	}
	if outcome.Synthesis != nil {
		resp.Synthesis = synthesisToProto(outcome.Synthesis)
	}
	return connect.NewResponse(resp), nil
}

// engineIssuesToProto maps the per-query engine-degradation signal onto the
// wire shape (nil-safe: empty input yields nil, not an empty slice).
func engineIssuesToProto(issues []internallivesearch.EngineIssue) []*livesearchv1.EngineIssue {
	if len(issues) == 0 {
		return nil
	}
	out := make([]*livesearchv1.EngineIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, &livesearchv1.EngineIssue{Engine: issue.Engine, Reason: issue.Reason})
	}
	return out
}

func resultToProto(r internallivesearch.Result) *livesearchv1.SearchResult {
	return &livesearchv1.SearchResult{
		Url:      r.URL,
		Title:    r.Title,
		Snippet:  r.Snippet,
		Engine:   r.Engine,
		Score:    r.Score,
		Category: r.Category,
	}
}

func synthesisToProto(s *internallivesearch.Synthesis) *livesearchv1.Synthesis {
	out := &livesearchv1.Synthesis{
		Text:      s.Text,
		Abstained: s.Abstained,
		Citations: make([]*livesearchv1.Citation, 0, len(s.Citations)),
	}
	for _, c := range s.Citations {
		out.Citations = append(out.Citations, &livesearchv1.Citation{
			ResultIndex: int32(c.ResultIndex),
			Url:         c.URL,
			Title:       c.Title,
		})
	}
	return out
}
