package home

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	homev1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	homeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration/home_integration_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client homeconnect.HomeIntegrationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: homeconnect.NewHomeIntegrationServiceClient(httpClient, baseURL),
	}
}

func (h handlers) actions(ctx cliapp.RunContext) error {
	resp, err := h.client.ListActions(context.Background(), connect.NewRequest(&homev1.ListActionsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list home actions", err, nil)
	}
	lines := make([]string, 0, len(resp.Msg.GetActions()))
	for _, a := range resp.Msg.GetActions() {
		lines = append(lines, fmt.Sprintf("%s effect=%s approval=%t %s", a.GetName(), a.GetEffect(), a.GetApprovalRequired(), a.GetDescription()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Fetched Home Automation actions."}, ResultsHeading: "Actions", Results: lines})
}

func (h handlers) invoke(ctx cliapp.RunContext) error {
	resp, err := h.client.InvokeAction(context.Background(), connect.NewRequest(&homev1.InvokeActionRequest{Name: ctx.Positional("name"), Approved: ctx.BoolFlag("approved")}))
	if err != nil {
		return cliapp.WrapAPIError("invoke home action", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{resp.Msg.GetMessage()}, Changes: []string{formatEvent(resp.Msg.GetEvent())}})
}

func (h handlers) events(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRecentEvents(context.Background(), connect.NewRequest(&homev1.ListRecentEventsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list home events", err, nil)
	}
	lines := make([]string, 0, len(resp.Msg.GetEvents()))
	for _, e := range resp.Msg.GetEvents() {
		lines = append(lines, formatEvent(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Fetched Home Automation integration events."}, ResultsHeading: "Events", Results: lines})
}

func formatEvent(e *homev1.HomeEvent) string {
	if e == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s type=%s at=%s %s", e.GetId(), e.GetType(), e.GetOccurredAt(), e.GetSummary())
}
