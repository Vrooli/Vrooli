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

// assignFixed switches a scenario's port from a range to a free in-band fixed
// port (previews unless --apply). Thin over ValidationService.AssignFixedPort.
func (h *handlers) assignFixed(ctx cliapp.RunContext) error {
	apply := ctx.FlagDeclared("apply") && ctx.BoolFlag("apply")
	scenario := ctx.Positional("scenario")
	resp, err := h.client.AssignFixedPort(context.Background(), connect.NewRequest(&validationv1.PortSwitchRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		PortName: firstFlag(ctx.FlagValues("port-name")),
		Apply:    apply,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("assign fixed port for %q", scenario), err, nil)
	}
	return renderPortSwitch(ctx, resp.Msg, apply)
}

// releaseFixed reverts a scenario's fixed port back to its canonical range
// (previews unless --apply). Thin over ValidationService.ReleaseFixedPort.
func (h *handlers) releaseFixed(ctx cliapp.RunContext) error {
	apply := ctx.FlagDeclared("apply") && ctx.BoolFlag("apply")
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ReleaseFixedPort(context.Background(), connect.NewRequest(&validationv1.PortSwitchRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		PortName: firstFlag(ctx.FlagValues("port-name")),
		Apply:    apply,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("release fixed port for %q", scenario), err, nil)
	}
	return renderPortSwitch(ctx, resp.Msg, apply)
}

func renderPortSwitch(ctx cliapp.RunContext, msg *validationv1.PortSwitchResponse, apply bool) error {
	if msg == nil {
		return fmt.Errorf("server returned no port-switch response")
	}
	result := msg.Message
	if result == "" {
		result = fmt.Sprintf("%s %s port", msg.Scenario, msg.PortName)
	}
	var changes []string
	if msg.Changed {
		state := "preview"
		if apply {
			state = "applied"
		}
		changes = append(changes, fmt.Sprintf("%s: previous_port=%d assigned_port=%d", state, msg.PreviousPort, msg.AssignedPort))
	} else {
		changes = append(changes, "no change (already in target state)")
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
	})
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
