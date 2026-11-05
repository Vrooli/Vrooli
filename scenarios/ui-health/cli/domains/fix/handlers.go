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

// run previews fixes (or applies them when --apply is set).
func (h *handlers) run(ctx cliapp.RunContext) error {
	apply := ctx.FlagDeclared("apply") && ctx.BoolFlag("apply")
	return h.fix(ctx, apply)
}

// apply always applies fixes.
func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.fix(ctx, true)
}

func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		RuleIds:  splitCSV(ctx.FlagValues("rule")),
	})

	var (
		resp *connect.Response[scenariovalidationv1.FixResponse]
		err  error
		verb string
	)
	if apply {
		verb = "apply ui fixes"
		resp, err = h.client.ApplyFix(context.Background(), req)
	} else {
		verb = "preview ui fixes"
		resp, err = h.client.PreviewFix(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("%s for %q", verb, scenario), err, nil)
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
		changes = append(changes, fmt.Sprintf("[%s] %s → %s — %s", state, c.GetRuleId(), c.GetFilePath(), c.GetDescription()))
	}

	result := []string{fmt.Sprintf("%d ui fix candidate(s) for %s.", len(msg.GetCandidates()), msg.GetScenario())}
	result = append(result, msg.GetMessages()...)

	next := []string{fmt.Sprintf("`ui-health validate scenario %s --json` - re-check ui findings", scenario)}
	if !apply && len(msg.GetCandidates()) > 0 {
		next = append([]string{fmt.Sprintf("`ui-health fix run %s --apply` - apply these fixes", scenario)}, next...)
	}

	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: next,
	})
}

func splitCSV(values []string) []string {
	var out []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
