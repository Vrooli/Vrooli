package sweep

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
	sweepconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep/sweep_v1connect"
)

type handlers struct {
	client sweepconnect.SweepServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// A sweep audits every budgeted flow (profile restart + BAS capture each),
	// running many minutes. A zero timeout means no client-side deadline; the
	// server bounds the work.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: sweepconnect.NewSweepServiceClient(httpClient, baseURL)}
}

// run drives the per-flow capture-sweep for a scenario.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RunSweep(context.Background(), connect.NewRequest(&sweepv1.RunSweepRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run sweep for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sweep response")
	}
	msg := resp.Msg
	captured, breached := 0, 0
	results := make([]string, 0, len(msg.GetResults()))
	for _, r := range msg.GetResults() {
		line := fmt.Sprintf("%s: %s", r.GetFlow(), r.GetOutcome())
		if r.GetOutcome() == "captured" {
			captured++
			if r.GetWithinBudget() {
				line += " (within budget)"
			} else if len(r.GetViolations()) > 0 {
				breached++
				line += " — OVER BUDGET: " + strings.Join(r.GetViolations(), "; ")
			}
		} else if reason := strings.TrimSpace(r.GetReason()); reason != "" {
			line += " (" + reason + ")"
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = append(results, "No budgeted flows to sweep (declare performance.budgets.flows first).")
	}
	summary := fmt.Sprintf("%s: swept %d flow(s), %d captured, %d over budget.", scenario, len(msg.GetResults()), captured, breached)
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{summary, "This call captures + persists flow samples out-of-band; the per-flow budget check then runs in the test-genie Performance phase (suite run), not this call."},
		ResultsHeading: "Flows",
		Results:        results,
	})
}
