package fixconfig

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewValidationServiceClient(httpClient, baseURL)}
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
	req := connect.NewRequest(&validationv1.FixConfigRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		RuleIds:  splitCSV(ctx.FlagValues("rule")),
		Apply:    apply,
	})

	var (
		resp *connect.Response[validationv1.FixConfigResponse]
		err  error
		verb string
	)
	if apply {
		verb = "apply config fixes"
		resp, err = h.client.ApplyFixConfig(context.Background(), req)
	} else {
		verb = "preview config fixes"
		resp, err = h.client.PreviewFixConfig(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("%s for %q", verb, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix-config response")
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

	result := []string{fmt.Sprintf("%d structure fix candidate(s) for %s.", len(msg.GetCandidates()), msg.GetScenario())}
	result = append(result, msg.GetMessages()...)

	next := []string{fmt.Sprintf("`structure-health validate scenario %s --json` - re-check structure findings", scenario)}
	if !apply && len(msg.GetCandidates()) > 0 {
		next = append([]string{fmt.Sprintf("`structure-health fix-config run %s --apply` - apply these fixes", scenario)}, next...)
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
