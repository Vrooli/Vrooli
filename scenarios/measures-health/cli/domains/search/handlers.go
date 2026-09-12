package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/search/search_v1connect"
)

// searchTimeout is the HTTP client timeout for a measure query. A query may
// resolve params (deterministic) and, for a read-only measure, proxy execution
// to the owning scenario over the network — generous but bounded.
const searchTimeout = 60 * time.Second

type handlers struct {
	core   *cliapp.ScenarioApp
	client searchconnect.SearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, searchTimeout)
	return &handlers{
		core:   core,
		client: searchconnect.NewSearchServiceClient(httpClient, baseURL),
	}
}

// query calls SearchService.Search and renders the matched measure(s): the
// resolved params + the executed answer (or the needs[] / confirmation state the
// auto-execution gate produced).
func (h *handlers) query(ctx cliapp.RunContext) error {
	question := ctx.Positional("question")
	limit := parseLimit(ctx.Flag("limit"))
	resp, err := h.client.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{
		Query: question,
		Limit: limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("search query %q", question), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.GetResults()))
	for _, r := range msg.GetResults() {
		results = append(results, formatResult(r))
	}
	summary := []string{fmt.Sprintf("matched %d measure(s) via %s matcher", len(msg.GetResults()), matcherLabel(msg.GetMatcher()))}
	if len(msg.GetResults()) == 0 {
		summary = append(summary, "no measure matched — try rephrasing, or no scenario declares a measure for this question yet")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Measure answers",
		Results:        results,
		RetrievalHints: []string{"`measures-health validate coverage` — which scenarios expose measures"},
	})
}

// status calls SearchService.Status and renders index/backend availability.
func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("search status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	m := resp.Msg
	results := []string{
		fmt.Sprintf("available:     %v", m.GetAvailable()),
		fmt.Sprintf("indexed:       %d measure(s)", m.GetIndexedCount()),
		fmt.Sprintf("matcher:       %s", matcherLabel(m.GetMatcher())),
		fmt.Sprintf("ollama:        %v (constrained param extraction)", m.GetOllama()),
		fmt.Sprintf("qdrant:        %v (future hybrid index leg)", m.GetQdrant()),
	}
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary:        []string{"Measures index status"},
		ResultsHeading: "Backends",
		Results:        results,
	})
}

func formatResult(r *searchv1.MeasureResult) string {
	m := r.GetMeasure()
	if m == nil {
		return fmt.Sprintf("(score %.2f) — empty measure", r.GetScore())
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%-24s scenario=%s effect=%s score=%.2f confidence=%.2f", m.GetMeasureId(), m.GetScenario(), m.GetEffect(), r.GetScore(), m.GetConfidence())
	if ans := strings.TrimSpace(m.GetAnswer()); ans != "" {
		fmt.Fprintf(&b, "\n    answer: %s", ans)
	}
	if len(m.GetNeeds()) > 0 {
		fmt.Fprintf(&b, "\n    needs:  %s (not executed — provide these)", strings.Join(m.GetNeeds(), ", "))
	}
	if len(m.GetAnswer()) == 0 && len(m.GetNeeds()) == 0 && m.GetEffect() != "read" {
		fmt.Fprintf(&b, "\n    withheld: %s measure resolved but not auto-executed (confirm to run)", m.GetEffect())
	}
	if eq := strings.TrimSpace(m.GetExecutedQuery()); eq != "" {
		fmt.Fprintf(&b, "\n    via:    %s", eq)
	}
	if len(m.GetParams()) > 0 {
		fmt.Fprintf(&b, "\n    params: %s", formatParams(m.GetParams()))
	}
	return b.String()
}

func formatParams(p map[string]string) string {
	parts := make([]string, 0, len(p))
	for k, v := range p {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, " ")
}

func matcherLabel(m string) string {
	if strings.TrimSpace(m) == "" {
		return "none"
	}
	return m
}

// parseLimit reads the --limit flag, defaulting to 1 (a measure provider returns
// the best answer) on an empty/invalid value.
func parseLimit(raw string) int32 {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return 1
	}
	return int32(v)
}
