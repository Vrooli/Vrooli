package adapters

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters/adapters_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client adaptersconnect.AdapterServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: adaptersconnect.NewAdapterServiceClient(httpClient, baseURL),
	}
}

func (h handlers) capabilities(ctx cliapp.RunContext) error {
	resp, err := h.client.ListCapabilities(context.Background(), connect.NewRequest(&adaptersv1.ListCapabilitiesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list capabilities", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Fetched adapter capabilities."}, ResultsHeading: "Capabilities", Results: formatCapabilities(resp.Msg.GetCapabilities())})
}

func (h handlers) explain(ctx cliapp.RunContext) error {
	resp, err := h.client.ExplainUnsupportedAction(context.Background(), connect.NewRequest(&adaptersv1.ExplainUnsupportedActionRequest{Action: ctx.Positional("action")}))
	if err != nil {
		return cliapp.WrapAPIError("explain unsupported action", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{formatCapability(resp.Msg.GetCapability())}, ResultsHeading: "Manual Steps", Results: resp.Msg.GetManualSteps()})
}

func (h handlers) platform(ctx cliapp.RunContext) error {
	resp, err := h.client.GetPlatformSummary(context.Background(), connect.NewRequest(&adaptersv1.GetPlatformSummaryRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get platform summary", err, nil)
	}
	s := resp.Msg.GetSummary()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("os=%s arch=%s profile=%s", s.GetOs(), s.GetArch(), s.GetProfile())}, ResultsHeading: "Notes", Results: s.GetNotes()})
}

func formatCapabilities(caps []*adaptersv1.Capability) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		out = append(out, formatCapability(c))
	}
	return out
}

func formatCapability(c *adaptersv1.Capability) string {
	if c == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s action=%s supported=%t admin=%t rollback=%t reason=%s", c.GetAdapter(), c.GetAction(), c.GetSupported(), c.GetRequiresAdmin(), c.GetRollbackSupported(), c.GetReason())
}
