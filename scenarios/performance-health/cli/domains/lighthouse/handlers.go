package lighthouse

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lighthousev1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse"
	lighthouseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse/lighthouse_v1connect"
)

type handlers struct {
	client lighthouseconnect.LighthouseServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: lighthouseconnect.NewLighthouseServiceClient(httpClient, baseURL)}
}

// run scores a scenario's pages with Lighthouse.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RunLighthouse(context.Background(), connect.NewRequest(&lighthousev1.RunLighthouseRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run lighthouse for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no lighthouse response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("%s: %s.", msg.GetScenario(), outcomeLabel(msg.GetOutcome()))}
	if r := strings.TrimSpace(msg.GetReason()); r != "" {
		summary = append(summary, "Reason: "+r)
	}
	results := make([]string, 0, len(msg.GetPages()))
	for _, p := range msg.GetPages() {
		results = append(results, fmt.Sprintf("%s — perf=%.2f a11y=%.2f bp=%.2f seo=%.2f", p.GetUrl(), p.GetPerformance(), p.GetAccessibility(), p.GetBestPractices(), p.GetSeo()))
	}
	if len(results) == 0 {
		results = append(results, "No pages scored.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Lighthouse scores",
		Results:        results,
	})
}

func outcomeLabel(o lighthousev1.LighthouseOutcome) string {
	switch o {
	case lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SCORED:
		return "scored"
	case lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SKIPPED:
		return "skipped"
	case lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_FAILED:
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
