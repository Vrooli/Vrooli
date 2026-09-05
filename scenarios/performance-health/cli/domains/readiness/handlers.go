package readiness

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness/readiness_v1connect"
)

type handlers struct {
	client readinessconnect.ReadinessServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: readinessconnect.NewReadinessServiceClient(httpClient, baseURL)}
}

// validate reports a scenario's reachable capture tier and infra findings.
func (h *handlers) validate(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate readiness for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no readiness response")
	}
	msg := resp.Msg
	summary := []string{
		fmt.Sprintf("%s: reachable tier %s (ui=%s).", msg.GetScenario(), tierLabel(msg.GetTier()), orNone(msg.GetUiFramework())),
		fmt.Sprintf("Auto-fixable findings: %d.", msg.GetAutofixableCount()),
	}
	if r := strings.TrimSpace(msg.GetDegradedReason()); r != "" {
		summary = append(summary, "Degraded: "+r)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Surfaces",
		Results:        orPlaceholder(msg.GetSurfaces(), "No surfaces detected."),
		RetrievalHints: []string{fmt.Sprintf("`performance-health readiness fix %s` — preview Tier-1 fixes", scenario)},
	})
}

// fix previews the readiness fixes (dry-run).
func (h *handlers) fix(ctx cliapp.RunContext) error {
	return h.runFix(ctx, false)
}

// apply applies the readiness fixes.
func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.runFix(ctx, true)
}

func (h *handlers) runFix(ctx cliapp.RunContext, apply bool) error {
	scenario := ctx.Positional("scenario")
	req := connect.NewRequest(&readinessv1.ReadinessFixRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		RuleIds:  ctx.FlagValues("rule"),
	})
	var (
		resp *connect.Response[readinessv1.ReadinessFixResponse]
		err  error
	)
	if apply {
		resp, err = h.client.ApplyReadinessFix(context.Background(), req)
	} else {
		resp, err = h.client.PreviewReadinessFix(context.Background(), req)
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("fix readiness for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	changes := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, c := range resp.Msg.GetCandidates() {
		changes = append(changes, fmt.Sprintf("%s — %s (%s)", c.GetRuleId(), c.GetFilePath(), c.GetDescription()))
	}
	changes = append(changes, resp.Msg.GetMessages()...)
	verb := "Previewed"
	if apply {
		verb = "Applied"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s %d readiness fix candidate(s) for %s.", verb, len(resp.Msg.GetCandidates()), resp.Msg.GetScenario())},
		Changes: changes,
	})
}

func tierLabel(t readinessv1.CaptureTier) string {
	switch t {
	case readinessv1.CaptureTier_CAPTURE_TIER_NONE:
		return "none"
	case readinessv1.CaptureTier_CAPTURE_TIER_0:
		return "0"
	case readinessv1.CaptureTier_CAPTURE_TIER_1:
		return "1"
	default:
		return "unspecified"
	}
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "none"
	}
	return s
}

func orPlaceholder(values []string, placeholder string) []string {
	if len(values) == 0 {
		return []string{placeholder}
	}
	return values
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
