package livesearch

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	livesearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch"
	livesearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch/livesearch_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-search/cli/internal/cliutil"
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
		Limit:      cliutil.ParseInt32(ctx.Flag("limit")),
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
	if warning := formatEngineWarning(msg.DegradedEngines); warning != "" {
		summary = append(summary, warning)
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

// formatEngineWarning renders the per-query engine-degradation signal as a
// single warning line ("results may be partial"), or "" when every engine
// answered.
func formatEngineWarning(issues []*livesearchv1.EngineIssue) string {
	if len(issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Engine, issue.Reason))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("⚠ %d engine(s) unavailable (%s) — results may be partial.", len(parts), strings.Join(parts, "; "))
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
