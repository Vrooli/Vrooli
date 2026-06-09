package livesearch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	livesearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch"
	livesearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch/livesearch_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client livesearchconnect.LiveSearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: livesearchconnect.NewLiveSearchServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := ctx.Positional("query")
	resp, err := h.client.Search(context.Background(), connect.NewRequest(&livesearchv1.SearchRequest{
		Query:      query,
		Limit:      parseInt32(ctx.Flag("limit")),
		Synthesize: ctx.BoolFlag("synthesis"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("live search", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search response")
	}

	msg := resp.Msg
	results := make([]string, 0, len(msg.Results))
	for i, r := range msg.Results {
		results = append(results, formatResult(i, r))
	}

	summary := []string{fmt.Sprintf("%d result(s) for %q%s.", len(msg.Results), query, provenanceSuffix(msg))}
	if msg.Degraded {
		summary = append(summary, fmt.Sprintf("Degraded: %s", msg.DegradedReason))
	}
	if syn := msg.Synthesis; syn != nil {
		summary = append(summary, formatSynthesis(syn))
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Results",
		Results:        results,
		RetrievalHints: []string{
			"`search <query> --synthesis` — add an always-cited snippet synthesis",
			"`search <query> --limit N` — cap the number of results",
		},
	})
}

// --- helpers ---

func parseInt32(s string) int32 {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return int32(n)
}

func provenanceSuffix(msg *livesearchv1.SearchResponse) string {
	if msg.Cached {
		return " (cached)"
	}
	return ""
}

func formatResult(i int, r *livesearchv1.SearchResult) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("[%d] %s — %s (%s, score=%.3f)\n    %s", i, r.Title, r.Url, r.Engine, r.Score, r.Snippet)
}

func formatSynthesis(s *livesearchv1.Synthesis) string {
	if s.Abstained {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			text = "sources insufficient or disagree"
		}
		return fmt.Sprintf("Synthesis abstained: %s", text)
	}
	cites := make([]string, 0, len(s.Citations))
	for _, c := range s.Citations {
		cites = append(cites, fmt.Sprintf("[%d]", c.ResultIndex))
	}
	return fmt.Sprintf("Synthesis: %s (cites %s)", strings.TrimSpace(s.Text), strings.Join(cites, " "))
}
