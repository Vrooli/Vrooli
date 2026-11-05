package benchmark

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	benchmarkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark"
	benchmarkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark/benchmark_v1connect"
)

type handlers struct {
	client benchmarkconnect.BenchmarkServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: benchmarkconnect.NewBenchmarkServiceClient(httpClient, baseURL)}
}

// run times a scenario's build surfaces.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RunBenchmark(context.Background(), connect.NewRequest(&benchmarkv1.RunBenchmarkRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run benchmark for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no benchmark response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("%s: %s.", msg.GetScenario(), outcomeLabel(msg.GetOutcome()))}
	if r := strings.TrimSpace(msg.GetReason()); r != "" {
		summary = append(summary, "Reason: "+r)
	}
	results := make([]string, 0, len(msg.GetTimings()))
	for _, t := range msg.GetTimings() {
		flag := ""
		if t.GetOverBudget() {
			flag = " OVER BUDGET"
		}
		results = append(results, fmt.Sprintf("%s — %dms (budget %dms)%s", t.GetSurface(), t.GetDurationMs(), t.GetBudgetMs(), flag))
	}
	if len(results) == 0 {
		results = append(results, "No build timings captured.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Build timings",
		Results:        results,
	})
}

func outcomeLabel(o benchmarkv1.BenchmarkOutcome) string {
	switch o {
	case benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_MEASURED:
		return "measured"
	case benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_SKIPPED:
		return "skipped"
	case benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
