package perf

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	perfv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf"
	perfconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf/perf_v1connect"
)

type handlers struct {
	client perfconnect.PerfServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: perfconnect.NewPerfServiceClient(httpClient, baseURL)}
}

// measure restarts the target scenario and records a startup benchmark.
func (h *handlers) measure(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&perfv1.BenchmarkStartupRequest{
		Scenario:       scenario,
		TimeoutSeconds: int32(parseInt(firstFlag(ctx.FlagValues("timeout")))),
	})
	resp, err := h.client.BenchmarkStartup(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("benchmark startup for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMeasurement() == nil {
		return fmt.Errorf("server returned no measurement")
	}
	m := resp.Msg.GetMeasurement()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        measurementSummary(m),
		ResultsHeading: "Per-surface timing",
		Results:        surfaceLines(m.GetSurfaceTimings()),
		RetrievalHints: []string{fmt.Sprintf("`structure-health perf trend %s` - view the persisted trend", scenario)},
	})
}

// trend reads a scenario's persisted startup measurements.
func (h *handlers) trend(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&perfv1.GetPerfTrendRequest{
		Scenario: scenario,
		Limit:    int32(parseInt(firstFlag(ctx.FlagValues("limit")))),
	})
	resp, err := h.client.GetPerfTrend(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("read perf trend for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no trend response")
	}
	results := make([]string, 0, len(resp.Msg.GetMeasurements()))
	for _, m := range resp.Msg.GetMeasurements() {
		state := "healthy"
		if !m.GetHealthy() {
			state = "UNHEALTHY"
		}
		results = append(results, fmt.Sprintf("%s — %dms to %s%s", m.GetCapturedAt(), m.GetTimeToHealthyMs(), state, noteSuffix(m.GetNote())))
	}
	if len(results) == 0 {
		results = append(results, fmt.Sprintf("No startup measurements recorded yet for %s. Run `structure-health perf measure %s`.", scenario, scenario))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d measurement(s) for %s (newest first).", len(resp.Msg.GetMeasurements()), scenario)},
		ResultsHeading: "Startup trend",
		Results:        results,
	})
}

func measurementSummary(m *perfv1.PerfMeasurement) []string {
	state := "healthy"
	if !m.GetHealthy() {
		state = "did NOT reach healthy"
	}
	out := []string{
		fmt.Sprintf("%s: %dms to %s.", m.GetScenario(), m.GetTimeToHealthyMs(), state),
	}
	if env := m.GetMetrics().GetEnvironment(); env != nil {
		out = append(out, fmt.Sprintf("Host: %s/%s, %d CPU(s).", env.GetOs(), env.GetArch(), env.GetNumCpu()))
	}
	if res := m.GetMetrics().GetResources(); res != nil {
		out = append(out, fmt.Sprintf("Resource envelope: cpu_user=%dms cpu_sys=%dms peak_rss=%dB.",
			res.GetCpuUserMs(), res.GetCpuSysMs(), res.GetPeakRssBytes()))
	}
	if note := strings.TrimSpace(m.GetNote()); note != "" {
		out = append(out, "Note: "+note)
	}
	return out
}

func surfaceLines(timings []*perfv1.SurfaceTiming) []string {
	if len(timings) == 0 {
		return []string{"No per-surface timings captured."}
	}
	out := make([]string, 0, len(timings))
	for _, st := range timings {
		out = append(out, fmt.Sprintf("%s: %dms", st.GetSurface(), st.GetTimeToHealthyMs()))
	}
	return out
}

func noteSuffix(note string) string {
	if strings.TrimSpace(note) == "" {
		return ""
	}
	return " (" + note + ")"
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
