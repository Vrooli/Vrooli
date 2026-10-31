package fix

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	if ctx.BoolFlag("apply") {
		return h.run(ctx, true)
	}
	return h.run(ctx, false)
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.run(ctx, true)
}

func (h *handlers) run(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: scenario,
		Path:     ctx.Flag("path"),
		RuleIds:  splitCSV(ctx.Flag("rule")),
	})
	var (
		resp *connect.Response[scenariovalidationv1.FixResponse]
		err  error
	)
	if apply {
		resp, err = h.client.ApplyFix(context.Background(), req)
	} else {
		resp, err = h.client.PreviewFix(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("workflow fix for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	msg := resp.Msg
	changes := make([]string, 0, len(msg.GetCandidates()))
	for _, c := range msg.GetCandidates() {
		state := "preview"
		if c.GetApplied() {
			state = "applied"
		}
		changes = append(changes, fmt.Sprintf("[%s] %s -> %s - %s", state, c.GetRuleId(), c.GetFilePath(), c.GetDescription()))
	}
	result := []string{fmt.Sprintf("%d workflow fix candidate(s) for %s.", len(msg.GetCandidates()), msg.GetScenario())}
	result = append(result, msg.GetMessages()...)
	next := []string{fmt.Sprintf("`workflow-health validate scenario %s --json` - re-check workflow findings", scenario)}
	if !apply && len(msg.GetCandidates()) > 0 {
		next = append([]string{fmt.Sprintf("`workflow-health fix preview %s --apply` - apply these fixes", scenario)}, next...)
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: next,
	})
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
