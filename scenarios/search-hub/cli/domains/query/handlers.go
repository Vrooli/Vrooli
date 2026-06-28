package query

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so the query handler
// has typed access to the RoutingService client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client routingconnect.RoutingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: routingconnect.NewRoutingServiceClient(httpClient, baseURL),
	}
}

// query fans a natural-language query out across registered providers via
// RoutingService.Query and renders the by-provider grouping in operator-friendly
// form (which corpora were searched, per-corpus counts, provenance per hit, and
// how to expand the search). `--json` emits the raw QueryResponse for scripting.
func (h *handlers) query(ctx cliapp.RunContext) error {
	text := strings.TrimSpace(ctx.Positional("text"))
	if text == "" {
		return fmt.Errorf("a query text positional is required, e.g. `search-hub query \"restart a scenario\" --type command`")
	}

	req := &routingv1.QueryRequest{
		Query:   text,
		Types:   splitTypes(ctx.Flag("type")),
		All:     ctx.BoolFlag("all"),
		Group:   strings.TrimSpace(ctx.Flag("group")),
		Limit:   parseLimit(ctx.Flag("limit")),
		Explain: ctx.BoolFlag("explain"),
	}

	resp, err := h.client.Query(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("query %q", text), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no query response")
	}
	msg := resp.Msg

	totalHits := 0
	for _, g := range msg.GetGroups() {
		totalHits += len(g.GetHits())
	}

	summary := []string{
		fmt.Sprintf("Searched %d corpora; %d total hit(s)%s.",
			len(msg.GetGroups()), totalHits, degradedSuffix(msg.GetDegraded())),
		fmt.Sprintf("Latency: %dms. Ranking: %s.", msg.GetLatencyMs(), rankingMode(msg.GetReranked())),
	}
	if len(msg.GetRoutingExplanation()) > 0 {
		summary = append(summary, "Routing:")
		for _, line := range msg.GetRoutingExplanation() {
			summary = append(summary, "  • "+line)
		}
	}

	// When the reranker produced a unified cross-provider ordering, that ranked
	// list is the primary operator view; the by-provider grouping stays as the
	// provenance/degradation detail beneath it. Pre-rerank (or on rerank
	// degradation) we show the honest grouping alone.
	resultsHeading := "Results by provider"
	results := renderGroups(msg.GetGroups())
	if msg.GetReranked() && len(msg.GetRanked()) > 0 {
		resultsHeading = "Unified ranking (reranked across providers)"
		results = renderRanked(msg.GetRanked())
		results = append(results, "", "Provenance by provider:")
		results = append(results, renderGroups(msg.GetGroups())...)
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: resultsHeading,
		Results:        results,
		RetrievalHints: []string{
			"`--type <a,b>` — route to specific leaf types (command, doc, record, component…)",
			"`--all` — fan out to every active provider",
			"`--group <scenario>` — scope to one scenario's leaves",
			"`--limit <n>` — change the per-provider result cap (default 10)",
			"`--explain` — show why these providers were chosen",
			"`providers list` — see every registered provider and its type",
		},
	})
}

// renderGroups flattens the by-provider grouping into operator-friendly lines:
// a provider header (with count or a degraded note) followed by each hit's
// title, snippet, score, and provenance path.
func renderGroups(groups []*routingv1.ProviderResultGroup) []string {
	if len(groups) == 0 {
		return []string{"(no providers matched — pass --type, --all, or --group)"}
	}
	out := make([]string, 0, len(groups)*3)
	for _, g := range groups {
		if g.GetDegraded() {
			out = append(out, fmt.Sprintf("▸ %s — degraded: %s", g.GetProviderId(), g.GetNote()))
			continue
		}
		out = append(out, fmt.Sprintf("▸ %s (%d)", g.GetProviderId(), g.GetCount()))
		if len(g.GetHits()) == 0 {
			out = append(out, "    (no matches)")
			continue
		}
		if allWeak(g.GetHits()) {
			out = append(out, "    (no confident match — all returned hits are weak)")
		}
		for i, hit := range g.GetHits() {
			out = append(out, formatHit(i+1, hit))
		}
	}
	return out
}

// renderRanked renders the unified, cross-provider ranked list (Phase 6). Each
// line carries the rerank score plus the owning provider/leaf so a result is
// both comparably ranked and traceable to where it came from.
func renderRanked(hits []*routingv1.SearchHit) []string {
	if len(hits) == 0 {
		return []string{"(no results)"}
	}
	out := make([]string, 0, len(hits))
	for i, hit := range hits {
		title := strings.TrimSpace(hit.GetTitle())
		if title == "" {
			title = hit.GetId()
		}
		line := fmt.Sprintf("%d. %s", i+1, title)
		if snippet := truncate(hit.GetSnippet(), 80); snippet != "" {
			line += " — " + snippet
		}
		line += fmt.Sprintf(" [%s %s/%s]", confidenceBand(hit), hit.GetProviderId(), provenance(hit))
		out = append(out, line)
	}
	return out
}

// formatHit renders one SearchHit with its provenance (provider + path/id) so
// every result is traceable to where it came from.
func formatHit(n int, hit *routingv1.SearchHit) string {
	title := strings.TrimSpace(hit.GetTitle())
	if title == "" {
		title = hit.GetId()
	}
	line := fmt.Sprintf("    %d. %s", n, title)
	if snippet := truncate(hit.GetSnippet(), 80); snippet != "" {
		line += " — " + snippet
	}
	line += fmt.Sprintf(" [%s %s/%s]", confidenceBand(hit), hit.GetProviderGroup(), provenance(hit))
	if locs := locationSummary(hit.GetLocations()); locs != "" {
		line += " locations: " + locs
	}
	return line
}

func allWeak(hits []*routingv1.SearchHit) bool {
	if len(hits) == 0 {
		return false
	}
	for _, hit := range hits {
		if hit.GetConfidence() == nil || !hit.GetConfidence().GetWeak() {
			return false
		}
	}
	return true
}

func confidenceBand(hit *routingv1.SearchHit) string {
	c := hit.GetConfidence()
	if c == nil {
		return "confidence=unknown"
	}
	if c.GetWeak() {
		if regime := strings.TrimSpace(c.GetRegime()); regime != "" {
			return "confidence=weak/" + regime
		}
		return "confidence=weak"
	}
	if regime := strings.TrimSpace(c.GetRegime()); regime != "" {
		return "confidence=strong/" + regime
	}
	return "confidence=strong"
}

func provenance(hit *routingv1.SearchHit) string {
	if locs := hit.GetLocations(); len(locs) > 0 && strings.TrimSpace(locs[0]) != "" {
		return strings.TrimSpace(locs[0])
	}
	if path := strings.TrimSpace(hit.GetPath()); path != "" {
		return path
	}
	return hit.GetId()
}

func locationSummary(locations []string) string {
	clean := make([]string, 0, len(locations))
	for _, loc := range locations {
		loc = strings.TrimSpace(loc)
		if loc != "" {
			clean = append(clean, loc)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	if len(clean) <= 2 {
		return strings.Join(clean, ", ")
	}
	return strings.Join(clean[:2], ", ") + fmt.Sprintf(" (+%d more)", len(clean)-2)
}

// splitTypes parses a comma-separated --type value into trimmed, non-empty
// tokens.
func splitTypes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseLimit parses --limit, falling back to the server default (0 ⇒ server
// applies its own default of 10) on an empty or invalid value.
func parseLimit(raw string) int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	// ParseInt with bitSize=32 guarantees the result is already within the
	// int32 range, so the conversion cannot overflow on user-controlled input
	// (the gosec-clean idiom — avoids the int32(atoi) overflow pattern).
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
		return int32(n)
	}
	return 0
}

func degradedSuffix(degraded bool) string {
	if degraded {
		return " (some providers degraded — see notes below)"
	}
	return ""
}

func rankingMode(reranked bool) string {
	if reranked {
		return "unified cross-provider rerank"
	}
	return "by-provider grouping (no rerank — reranker disabled or unavailable)"
}

// truncate collapses internal whitespace (provider snippets often carry
// embedded newlines) to a single clean line, then caps the length.
func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
