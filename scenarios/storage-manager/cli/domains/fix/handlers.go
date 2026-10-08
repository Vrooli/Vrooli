package fix

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 60*time.Second)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) preview(ctx cliapp.RunContext) error { return h.run(ctx, false) }
func (h *handlers) apply(ctx cliapp.RunContext) error   { return h.run(ctx, true) }

func (h *handlers) run(ctx cliapp.RunContext, apply bool) error {
	name := ctx.Positional("name")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: name, RuleIds: splitCSV(ctx.FlagValues("rule"))})
	var resp *connect.Response[scenariovalidationv1.FixResponse]
	var err error
	if apply {
		resp, err = h.client.ApplyFix(context.Background(), req)
	} else {
		resp, err = h.client.PreviewFix(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("storage fix for %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	changes := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, candidate := range resp.Msg.GetCandidates() {
		changes = append(changes, fmt.Sprintf("[%s] %s — %s (applied=%v)", candidate.GetRuleId(), candidate.GetFilePath(), candidate.GetDescription(), candidate.GetApplied()))
	}
	verb := "previewed"
	if apply {
		verb = "applied"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("%s %d storage fix candidate(s) for %s", verb, len(changes), name)}, Changes: changes})
}

func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" && !seen[part] {
				seen[part] = true
				result = append(result, part)
			}
		}
	}
	return result
}
