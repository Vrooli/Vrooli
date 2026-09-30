package guidance

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	guidancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance"
	guidanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance/guidance_v1connect"
)

type handlers struct {
	client guidanceconnect.GuidanceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: guidanceconnect.NewGuidanceServiceClient(httpClient, baseURL)}
}

func (h *handlers) nextCall(ctx cliapp.OperationContext) (*guidancev1.NextGateResponse, error) {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.NextGate(context.Background(), connect.NewRequest(&guidancev1.NextGateRequest{Scenario: scenario}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get guidance for %q", scenario), err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) nextReport(_ cliapp.OperationContext, msg *guidancev1.NextGateResponse) cliapp.ListReport {
	if msg.Complete {
		return cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("%s orientation is complete (%d/%d required).", msg.Scenario, msg.Completed, msg.Required)},
			ResultsHeading: "Guidance",
			Results: []string{
				msg.Message,
				fmt.Sprintf("finalized=%t finalize_required=%t", msg.Finalized, msg.FinalizeRequired),
			},
		}
	}
	results := []string{fmt.Sprintf("%s: %s", msg.Gate.Id, msg.Gate.Title)}
	results = append(results, msg.Gate.Description)
	for _, check := range msg.Gate.Checks {
		status := "pending"
		if check.Passed {
			status = "passed"
		}
		if check.Skipped {
			status = "skipped"
		}
		line := fmt.Sprintf("%s %s %s", status, check.Kind, check.Label)
		if check.Message != "" {
			line += ": " + check.Message
		}
		results = append(results, line)
	}
	results = append(results, msg.Gate.Remediation...)
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s next gate: %s (%d/%d required complete).", msg.Scenario, msg.Gate.Id, msg.Completed, msg.Required)},
		ResultsHeading: "Guidance",
		Results:        results,
	}
}
