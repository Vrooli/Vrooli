package insights

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client metricsconnect.MetricsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: metricsconnect.NewMetricsServiceClient(httpClient, baseURL),
	}
}

// insights aggregates per-query telemetry into federation-health signals over an
// optional recent --window. It surfaces the make-or-break questions the plan's
// metrics phase exists to answer: is federation actually used, where is it
// under-used (registered-but-never-routed leaves), and how often do queries come
// back empty.
func (h *handlers) insightsCall(ctx cliapp.OperationContext) (*metricsv1.InsightsResponse, error) {
	raw := strings.TrimSpace(ctx.Flag("window"))
	window, err := parseWindow(raw)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{
		WindowDays: window,
		Window:     raw,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("insights", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no insights response")
	}
	return resp.Msg, nil
}

func (h *handlers) insightsReport(ctx cliapp.OperationContext, msg *metricsv1.InsightsResponse) cliapp.ListReport {
	raw := strings.TrimSpace(ctx.Flag("window"))
	scope := windowDescription(raw)

	summary := []string{
		fmt.Sprintf("%d quer(ies) recorded (%s).", msg.GetTotalQueries(), scope),
		fmt.Sprintf("Zero-result: %d (%.1f%%). Degraded: %d. Reranked: %d.",
			msg.GetZeroResultQueries(), msg.GetZeroResultRate()*100, msg.GetDegradedQueries(), msg.GetRerankedQueries()),
		fmt.Sprintf("Latency in window: p50 %dms, p95 %dms.", msg.GetLatencyP50Ms(), msg.GetLatencyP95Ms()),
		fmt.Sprintf("Evidence: %d sample(s), bounds %s to %s; recent-%d latency p50 %dms, p95 %dms.", msg.GetSampleCount(), msg.GetWindowFrom(), msg.GetWindowTo(), msg.GetRecentSampleCount(), msg.GetRecentLatencyP50Ms(), msg.GetRecentLatencyP95Ms()),
		fmt.Sprintf("Address-resolution cache: %.1f%% hit rate (%d hits, %d misses).", msg.GetResolverCacheHitRate()*100, msg.GetResolverCacheHits(), msg.GetResolverCacheMisses()),
		fmt.Sprintf("Shared substrate degradation: %d provider leg(s) (%s).", msg.GetSubstrateDegradedLegs(), substrateReasons(msg.GetSubstrateDegradationReasons())),
	}
	if !msg.GetSampleSufficient() {
		summary = append(summary, fmt.Sprintf("Latency percentile is provisional: %d sample(s), minimum %d required.", msg.GetSampleCount(), msg.GetMinimumSampleCount()))
	}
	if len(msg.GetRetirementCandidates()) > 0 {
		summary = append(summary, fmt.Sprintf("Retirement candidates: %d (report-only).", len(msg.GetRetirementCandidates())))
	}
	if len(msg.GetGroupAdvisories()) > 0 {
		summary = append(summary, fmt.Sprintf("Concentrated provider groups: %d (report-only).", len(msg.GetGroupAdvisories())))
	}

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Per-provider utilization",
		Results:        renderUtilization(msg.GetProviders()),
		RetrievalHints: []string{
			"`--window <duration>` — use 15m, 2h, or a bare day count; bounds and sample sufficiency are reported",
			"`federation` — live provider reachability + model availability",
			"under-utilized leaves are registered but never routed-to — check their description",
		},
	}
}

// renderUtilization lists each provider's routed/hit totals, flagging the
// under-utilized (registered-but-never-routed) leaves the metrics phase exists
// to surface.
func renderUtilization(providers []*metricsv1.ProviderUtilization) []string {
	if len(providers) == 0 {
		return []string{"(no active providers registered)"}
	}
	out := make([]string, 0, len(providers))
	for _, p := range providers {
		line := fmt.Sprintf("• %s — window-routed %d×, %d hit(s), p95 %dms, corpus-degraded %.1f%%",
			p.GetProviderId(), p.GetTimesRouted(), p.GetTotalHits(), p.GetLatencyP95Ms(), p.GetDegradationRate()*100)
		if reasons := p.GetDegradationReasons(); len(reasons) > 0 {
			line += fmt.Sprintf(" (%s)", reasons[0].GetReason())
		}
		if p.GetUnderUtilized() {
			line += "  ⚠ under-utilized (never routed-to)"
		}
		out = append(out, line)
	}
	return out
}

func substrateReasons(reasons []*metricsv1.ProviderDegradationReason) string {
	if len(reasons) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%s=%d", reason.GetReason(), reason.GetCount()))
	}
	return strings.Join(parts, ", ")
}

func parseWindow(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n >= 0 {
		return int32(n), nil
	}
	if d, err := time.ParseDuration(raw); err == nil && d >= time.Minute {
		return 0, nil
	}
	return 0, fmt.Errorf("invalid --window %q: use a non-negative day count or a duration such as 15m or 2h", raw)
}

func windowDescription(raw string) string {
	if raw == "" || raw == "0" {
		return "all-time"
	}
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return fmt.Sprintf("last %d day(s)", n)
	}
	return "last " + raw
}
