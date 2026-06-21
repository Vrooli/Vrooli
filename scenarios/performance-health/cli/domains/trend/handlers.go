package trend

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	trendv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend"
	trendconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend/trend_v1connect"
)

type handlers struct {
	client trendconnect.TrendServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: trendconnect.NewTrendServiceClient(httpClient, baseURL)}
}

// get reads a scenario's persisted performance samples, newest first.
func (h *handlers) get(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetTrend(context.Background(), connect.NewRequest(&trendv1.GetTrendRequest{
		Scenario: scenario,
		Limit:    int32(parseInt(firstFlag(ctx.FlagValues("limit")))),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("read trend for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no trend response")
	}
	results := make([]string, 0, len(resp.Msg.GetSamples()))
	for _, s := range resp.Msg.GetSamples() {
		line := fmt.Sprintf("%s — go=%dms ui=%dms bundle=%dB lcp=%dms p95=%dms startup=%dms",
			s.GetCapturedAt(), s.GetGoBuildMs(), s.GetUiBuildMs(), s.GetBundleBytes(), s.GetLcpMs(), s.GetP95Ms(), s.GetStartupMs())
		if s.GetSlowestComponent() != "" {
			line += fmt.Sprintf(" slowest=%s@%.1fms", s.GetSlowestComponent(), s.GetSlowestComponentAvgMs())
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = append(results, fmt.Sprintf("No performance samples recorded yet for %s.", scenario))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d sample(s) for %s (newest first).", len(resp.Msg.GetSamples()), scenario)},
		ResultsHeading: "Performance trend",
		Results:        results,
	})
}

func parseInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
