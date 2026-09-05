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
	return h.fix(ctx, false)
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.fix(ctx, true)
}

func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: scenario,
		RuleIds:  splitCSV(ctx.FlagValues("rule")),
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
		return cliapp.WrapAPIError("fix security", err, nil)
	}

	results := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, c := range resp.Msg.GetCandidates() {
		results = append(results, fmt.Sprintf("%s %s applied=%v", c.GetRuleId(), c.GetFilePath(), c.GetApplied()))
	}
	summary := []string{fmt.Sprintf("%d security fix candidate(s).", len(resp.Msg.GetCandidates()))}
	summary = append(summary, resp.Msg.GetMessages()...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  summary,
		Changes: results,
		NextCommand: []string{
			fmt.Sprintf("`security-health validate scenario %s --json` - refresh security findings", scenario),
		},
	})
}

func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
