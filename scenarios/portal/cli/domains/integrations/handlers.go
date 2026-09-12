package integrations

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations"
	integrationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations/integrations_v1connect"
)

type handlers struct {
	client integrationsconnect.IntegrationsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: integrationsconnect.NewIntegrationsServiceClient(httpClient, baseURL)}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&integrationsv1.StatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("show integrations status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no integrations status")
	}
	return renderStatus(ctx, resp.Msg, "Portal integrations status.")
}

func (h *handlers) override(ctx cliapp.RunContext) error {
	override, err := parseOverride(ctx.Positional("override"))
	if err != nil {
		return err
	}
	resp, err := h.client.UpdateOverride(context.Background(), connect.NewRequest(&integrationsv1.UpdateOverrideRequest{Override: override}))
	if err != nil {
		return cliapp.WrapAPIError("update integrations override", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no integrations override status")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Set behavior override to %s.", override.String())},
		Changes: statusLines(resp.Msg.GetStatus()),
		NextCommand: []string{
			"`integrations status` - show current readiness",
		},
	})
}

func renderStatus(ctx cliapp.RunContext, status *integrationsv1.StatusResponse, summary string) error {
	return cliapp.RenderProtoList(ctx, status, cliapp.ListReport{
		Summary:        append([]string{summary}, statusLines(status)...),
		ResultsHeading: "Integrations",
		Results:        integrationLines(status),
		RetrievalHints: []string{
			"`integrations override auto` - return to measured mode",
			"`integrations override force-off` - force offline behavior",
			"`integrations override force-passive` - force passive behavior",
		},
	})
}

func statusLines(status *integrationsv1.StatusResponse) []string {
	if status == nil {
		return nil
	}
	return []string{
		fmt.Sprintf("mode: %s", status.GetActiveMode().String()),
		fmt.Sprintf("override: %s", status.GetOverride().String()),
		fmt.Sprintf("reason: %s", status.GetReason()),
	}
}

func integrationLines(status *integrationsv1.StatusResponse) []string {
	if status == nil {
		return nil
	}
	lines := make([]string, 0, len(status.GetIntegrations()))
	for _, item := range status.GetIntegrations() {
		stats := item.GetStats()
		lines = append(lines, fmt.Sprintf("%s (%s) state=%s required=%t samples=%d p50=%.0fms p95=%.0fms error=%.2f degraded=%.2f reason=%s",
			item.GetId(), item.GetDisplayName(), item.GetState().String(), item.GetRequired(),
			stats.GetSampleCount(), stats.GetLatencyP50Ms(), stats.GetLatencyP95Ms(), stats.GetErrorRate(), stats.GetDegradedRate(), item.GetReason()))
	}
	return lines
}

func parseOverride(value string) (integrationsv1.BehaviorOverride, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_AUTO, nil
	case "force-off", "off":
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_OFF, nil
	case "force-passive", "passive":
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_PASSIVE, nil
	default:
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_UNSPECIFIED, fmt.Errorf("override must be one of: auto, force-off, force-passive")
	}
}
