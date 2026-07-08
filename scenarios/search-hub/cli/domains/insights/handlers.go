package insights

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	window := parseWindow(ctx.Flag("window"))

	resp, err := h.client.Insights(context.Background(), connect.NewRequest(&metricsv1.InsightsRequest{
		WindowDays: window,
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
	scope := "all-time"
	if window := parseWindow(ctx.Flag("window")); window > 0 {
		scope = fmt.Sprintf("last %d day(s)", window)
	}

	summary := []string{
		fmt.Sprintf("%d quer(ies) recorded (%s).", msg.GetTotalQueries(), scope),
		fmt.Sprintf("Zero-result: %d (%.1f%%). Degraded: %d. Reranked: %d.",
			msg.GetZeroResultQueries(), msg.GetZeroResultRate()*100, msg.GetDegradedQueries(), msg.GetRerankedQueries()),
		fmt.Sprintf("Latency: p50 %dms, p95 %dms.", msg.GetLatencyP50Ms(), msg.GetLatencyP95Ms()),
	}

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Per-provider utilization",
		Results:        renderUtilization(msg.GetProviders()),
		RetrievalHints: []string{
			"`--window <days>` — restrict to recent telemetry (default all-time)",
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
		line := fmt.Sprintf("• %s — routed %d×, %d hit(s), p95 %dms, degraded %.1f%%",
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

// parseWindow parses --window into a non-negative day count; an empty or invalid
// value means all-time (0).
func parseWindow(raw string) int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
		return int32(n)
	}
	return 0
}
