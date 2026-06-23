package home

import (
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliapp"
	homev1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	homeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration/home_integration_v1connect"
)

type handlers struct{ core *cliapp.ScenarioApp }

func (h handlers) actions(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*homev1.ListActionsRequest, *homev1.ListActionsResponse](h.core, http.MethodPost, homeconnect.HomeIntegrationServiceListActionsProcedure, &homev1.ListActionsRequest{})
	if err != nil {
		return cliapp.WrapAPIError("list home actions", err, nil)
	}
	lines := make([]string, 0, len(resp.GetActions()))
	for _, a := range resp.GetActions() {
		lines = append(lines, fmt.Sprintf("%s effect=%s approval=%t %s", a.GetName(), a.GetEffect(), a.GetApprovalRequired(), a.GetDescription()))
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Fetched Home Automation actions."}, ResultsHeading: "Actions", Results: lines})
}

func (h handlers) invoke(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*homev1.InvokeActionRequest, *homev1.InvokeActionResponse](h.core, http.MethodPost, homeconnect.HomeIntegrationServiceInvokeActionProcedure, &homev1.InvokeActionRequest{Name: ctx.Positional("name"), Approved: ctx.BoolFlag("approved")})
	if err != nil {
		return cliapp.WrapAPIError("invoke home action", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp, cliapp.MutationReport{Result: []string{resp.GetMessage()}, Changes: []string{formatEvent(resp.GetEvent())}})
}

func (h handlers) events(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*homev1.ListRecentEventsRequest, *homev1.ListRecentEventsResponse](h.core, http.MethodPost, homeconnect.HomeIntegrationServiceListRecentEventsProcedure, &homev1.ListRecentEventsRequest{})
	if err != nil {
		return cliapp.WrapAPIError("list home events", err, nil)
	}
	lines := make([]string, 0, len(resp.GetEvents()))
	for _, e := range resp.GetEvents() {
		lines = append(lines, formatEvent(e))
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Fetched Home Automation integration events."}, ResultsHeading: "Events", Results: lines})
}

func formatEvent(e *homev1.HomeEvent) string {
	if e == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s type=%s at=%s %s", e.GetId(), e.GetType(), e.GetOccurredAt(), e.GetSummary())
}
