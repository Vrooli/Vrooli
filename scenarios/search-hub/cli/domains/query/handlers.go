package query

import (
	"context"
	"fmt"
	"regexp"
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
func (h *handlers) queryCall(ctx cliapp.OperationContext) (*routingv1.QueryResponse, error) {
	text := strings.TrimSpace(ctx.Positional("text"))
	if text == "" {
		return nil, fmt.Errorf("a query text positional is required, e.g. `search-hub query \"restart a scenario\" --type command`")
	}

	req := &routingv1.QueryRequest{
		Query:   text,
		Types:   splitTypes(ctx.Flag("type")),
		All:     ctx.BoolFlag("all"),
		Group:   strings.TrimSpace(ctx.Flag("group")),
		Scope:   strings.TrimSpace(ctx.Flag("scope")),
		Limit:   parseLimit(ctx.Flag("limit")),
		Explain: ctx.BoolFlag("explain"),
	}

	resp, err := h.client.Query(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("query %q", text), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no query response")
	}
	return resp.Msg, nil
}

func (h *handlers) queryReport(ctx cliapp.OperationContext, msg *routingv1.QueryResponse) cliapp.ListReport {
	totalHits, hitCountLabel := responseHitCount(msg)
	showAllResults := ctx != nil && ctx.BoolFlag("show-all-results")
	showDiagnostics := showAllResults || (ctx != nil && ctx.BoolFlag("explain"))

	summary := []string{
		fmt.Sprintf("Searched %d corpora; %d %s%s.",
			searchedCorpusCount(msg), totalHits, hitCountLabel, responseDegradationSummary(msg)),
	}
	if showDiagnostics {
		summary = append(summary, fmt.Sprintf("Latency: %dms. Ranking: %s (ordered by %s descending).", msg.GetLatencyMs(), rankingMode(msg.GetReranked()), orderedBy(msg)))
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
	visibleLimit := defaultVisibleHits
	if showAllResults {
		visibleLimit = 0
	}
	results := renderGroupsLimit(msg.GetGroups(), visibleLimit)
	if len(msg.GetQueryParts()) > 0 {
		resultsHeading = "Task evidence by request part"
		results = renderTaskParts(msg, visibleLimit)
	} else if len(msg.GetCorrectionCandidates()) > 0 {
		resultsHeading = "Possible entity corrections (not verified)"
		results = renderCorrections(msg, visibleLimit)
	} else if exact := directEntityMatch(ctx, msg); exact != nil {
		resultsHeading = "Direct entity match"
		results = renderDirectMatch(exact, visibleLimit)
	} else if msg.GetReranked() && len(msg.GetRanked()) > 0 {
		resultsHeading = fmt.Sprintf("Unified ranking (reranked across providers; ordered by %s)", orderedBy(msg))
		results = renderRankedLimit(msg.GetRanked(), visibleLimit)
		if showDiagnostics {
			results = append(results, "", "Provenance by provider:")
			results = append(results, renderGroupSummaries(msg.GetGroups())...)
		}
	}
	hints := nextActionHints(ctx, msg)
	switch {
	case totalHits == 0 && msg.GetDegraded():
		hints = append(hints, "Coverage was incomplete. Check the owner directly or use --explain to see which routes failed.")
	case totalHits == 0:
		hints = append(hints, "Try a broader capability phrase, or use --group if you know the owner.")
	case allWeak(visibleResponseHits(ctx, msg)):
		hints = append(hints, "The visible matches are weak. Check the owner or ask a narrower question before relying on them.")
	}

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: resultsHeading,
		Results:        results,
		RetrievalHints: hints,
	}
}

var programName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.[a-z0-9][a-z0-9-]*$`)

// A preparation read is a concrete next step, never an inferred authorization
// to execute a retrieved program. Validate identifiers before printing shell text.
func operationHints(msg *routingv1.QueryResponse) []string {
	if msg == nil {
		return nil
	}
	var candidates []*routingv1.SearchHit
	if msg.GetReranked() && len(msg.GetRanked()) > 0 {
		candidates = msg.GetRanked()[:1]
	} else {
		for _, group := range msg.GetGroups() {
			if group != nil && group.GetProviderId() == "program-runtime.library" && len(group.GetHits()) > 0 {
				candidates = group.GetHits()[:1]
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	var hit *routingv1.SearchHit
	for _, candidate := range candidates {
		if candidate == nil || candidate.GetConfidence().GetWeak() || candidate.GetProviderId() != "program-runtime.library" || !programName.MatchString(candidate.GetId()) {
			continue
		}
		hit = candidate
		break
	}
	if hit == nil {
		return nil
	}
	lines := []string{"", "Inspect the program contract and required skill:"}
	if usage, ok := libraryUsage(hit); ok {
		lines = append(lines, fmt.Sprintf("  recorded usage: %d", usage))
	}
	return append(lines, "  program-runtime library run program-runtime.prepare-operation --input name="+hit.GetId())
}

func libraryUsage(hit *routingv1.SearchHit) (int64, bool) {
	if hit == nil || hit.GetMetadata() == nil {
		return 0, false
	}
	value, ok := hit.GetMetadata().GetFields()["usage"]
	if !ok || value == nil || value.GetKind() == nil {
		return 0, false
	}
	if number := value.GetNumberValue(); number >= 0 && number == float64(int64(number)) {
		return int64(number), true
	}
	return 0, false
}

func orderedBy(msg *routingv1.QueryResponse) string {
	if value := strings.TrimSpace(msg.GetOrderedBy()); value != "" {
		return value
	}
	if msg.GetReranked() {
		return "rerank_score"
	}
	return "score"
}

// renderGroups flattens the by-provider grouping into operator-friendly lines:
// a provider header (with count or a degraded note) followed by each hit's
// title, snippet, score, and provenance path.
const defaultVisibleHits = 4

func renderGroups(groups []*routingv1.ProviderResultGroup) []string {
	return renderGroupsLimit(groups, 0)
}

func renderGroupsLimit(groups []*routingv1.ProviderResultGroup, maxHits int) []string {
	if len(groups) == 0 {
		return []string{"(no providers matched — pass --type, --all, or --group)"}
	}
	out := make([]string, 0, len(groups)*3)
	shown, total := 0, len(deduplicateHits(flattenGroupHits(groups)))
	seen := make(map[string]struct{}, total)
	for _, g := range groups {
		if g == nil {
			continue
		}
		// A degraded provider may still answer (one leg down, the rest ranked):
		// say so, then show what it returned.
		if g.GetDegraded() || len(g.GetDegradations()) > 0 {
			out = append(out, fmt.Sprintf("▸ %s (%d) — degraded: %s", truncate(g.GetProviderId(), 64), g.GetCount(), providerGroupDegradation(g)))
			if len(g.GetHits()) == 0 {
				continue
			}
		} else {
			out = append(out, fmt.Sprintf("▸ %s (%d)", truncate(g.GetProviderId(), 64), g.GetCount()))
		}
		if len(g.GetHits()) == 0 {
			out = append(out, "    (no matches)")
			continue
		}
		hits := make([]*routingv1.SearchHit, 0, len(g.GetHits()))
		for _, hit := range g.GetHits() {
			key := hitDedupeKey(hit)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if maxHits > 0 && shown+len(hits) >= maxHits {
				continue
			}
			hits = append(hits, hit)
		}
		if allWeak(hits) {
			out = append(out, "    (no confident match — all returned hits are weak)")
		}
		for i, hit := range hits {
			out = append(out, formatHit(i+1, hit))
			shown++
		}
	}
	if total > shown {
		out = append(out, fmt.Sprintf("… %d additional unique returned hit(s) omitted; use --show-all-results to display them", total-shown))
	}
	return out
}

func providerGroupDegradation(group *routingv1.ProviderResultGroup) string {
	if group == nil {
		return "details unavailable"
	}
	if note := strings.TrimSpace(group.GetNote()); note != "" {
		return truncate(note, 112)
	}
	for _, degradation := range group.GetDegradations() {
		if degradation == nil {
			continue
		}
		parts := make([]string, 0, 3)
		if leg := strings.TrimSpace(degradation.GetLeg()); leg != "" {
			parts = append(parts, truncate(leg, 32))
		}
		if reason := strings.TrimSpace(degradation.GetReason()); reason != "" {
			parts = append(parts, truncate(reason, 48))
		}
		if detail := strings.TrimSpace(degradation.GetDetail()); detail != "" {
			parts = append(parts, truncate(detail, 72))
		}
		if len(parts) > 0 {
			return strings.Join(parts, ": ")
		}
	}
	return "details unavailable"
}

// renderGroupSummaries renders the provider metadata kept beside Ranked. The
// ranked list owns the complete hit payload after successful reranking, so
// these lines must never re-render the same snippets.
func renderGroupSummaries(groups []*routingv1.ProviderResultGroup) []string {
	if len(groups) == 0 {
		return []string{"(no provider groups)"}
	}
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		line := fmt.Sprintf("▸ %s (%d)", truncate(group.GetProviderId(), 64), group.GetCount())
		if group.GetDegraded() || len(group.GetDegradations()) > 0 {
			line += " — degraded: " + providerGroupDegradation(group)
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return []string{"(no provider groups)"}
	}
	return out
}

// renderRanked renders the unified, cross-provider ranked list (Phase 6). Each
// line carries the rerank score plus the owning provider/leaf so a result is
// both comparably ranked and traceable to where it came from.
func renderRanked(hits []*routingv1.SearchHit) []string {
	return renderRankedLimit(hits, 0)
}

func renderRankedLimit(hits []*routingv1.SearchHit, maxHits int) []string {
	hits = deduplicateHits(hits)
	if len(hits) == 0 {
		return []string{"(no results)"}
	}
	shown := len(hits)
	if maxHits > 0 && shown > maxHits {
		shown = maxHits
	}
	out := make([]string, 0, shown+1)
	for i, hit := range hits[:shown] {
		title := strings.TrimSpace(hit.GetTitle())
		if title == "" {
			title = hit.GetId()
		}
		line := fmt.Sprintf("%d. %s", i+1, truncate(title, 110))
		if snippet := truncate(hit.GetSnippet(), 56); snippet != "" {
			line += " — " + snippet
		}
		line += fmt.Sprintf(" [%s %s/%s]", confidenceBand(hit), truncate(hit.GetProviderId(), 48), truncate(provenance(hit), 92))
		if intent := hit.GetMetadata().GetFields()["search_hub_intent_match"].GetStringValue(); intent != "" {
			line += " · found via: " + truncate(intent, 65)
		}
		out = append(out, line)
	}
	if len(hits) > shown {
		out = append(out, fmt.Sprintf("… %d additional unique returned hit(s) omitted; use --show-all-results to display them", len(hits)-shown))
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
	line := fmt.Sprintf("    %d. %s", n, truncate(title, 110))
	if snippet := truncate(hit.GetSnippet(), 56); snippet != "" {
		line += " — " + snippet
	}
	provider := strings.TrimSpace(hit.GetProviderId())
	if provider == "" {
		provider = strings.TrimSpace(hit.GetProviderGroup())
	}
	line += fmt.Sprintf(" [%s %s/%s]", confidenceBand(hit), truncate(provider, 48), truncate(provenance(hit), 92))
	if locs := locationSummary(hit.GetLocations()); locs != "" {
		line += " locations: " + truncate(locs, 112)
	}
	if intent := hit.GetMetadata().GetFields()["search_hub_intent_match"].GetStringValue(); intent != "" {
		line += " · found via: " + truncate(intent, 65)
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

const defaultQueryLimit int32 = 5

// parseLimit keeps the ordinary CLI view within the recommended fast-search
// budget. Callers can raise the per-provider cap explicitly when they need a
// deeper read.
func parseLimit(raw string) int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultQueryLimit
	}
	// ParseInt with bitSize=32 guarantees the result is already within the
	// int32 range, so the conversion cannot overflow on user-controlled input
	// (the gosec-clean idiom — avoids the int32(atoi) overflow pattern).
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
		return int32(n)
	}
	return defaultQueryLimit
}

func degradedSummary(groups []*routingv1.ProviderResultGroup, degraded bool) string {
	if !degraded {
		return ""
	}
	// Put the actionable degradation details in the first summary line. Search
	// results can be large and agents often only retain that line when deciding
	// whether an empty result is conclusive.
	const maxDetails = 3
	details := make([]string, 0, maxDetails)
	degradedCount := 0
	for _, group := range groups {
		if group == nil || !group.GetDegraded() {
			continue
		}
		degradedCount++
		if len(details) == maxDetails {
			continue
		}
		provider := truncate(group.GetProviderId(), 48)
		note := truncate(group.GetNote(), 96)
		if provider == "" {
			provider = "unknown provider"
		}
		if note == "" {
			details = append(details, provider)
		} else {
			details = append(details, provider+": "+note)
		}
	}
	if len(details) == 0 {
		return " (degraded; provider details unavailable)"
	}
	result := " (degraded: " + strings.Join(details, "; ")
	if degradedCount > len(details) {
		result += fmt.Sprintf("; +%d more", degradedCount-len(details))
	}
	return result + ")"
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
	s = strings.ToValidUTF8(s, "�")
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
