package describe

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ProtoHealthServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: validationconnect.NewProtoHealthServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) describeScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.DescribeScenarioProtos(context.Background(), connect.NewRequest(&validationv1.DescribeScenarioProtosRequest{
		Scenario: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("describe scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Surface == nil {
		return fmt.Errorf("server returned no proto surface")
	}
	surface := resp.Msg.Surface
	results := []string{
		fmt.Sprintf("files=%d services=%d messages=%d imports=%d cross_scenario_imports=%d",
			len(surface.Files),
			len(surface.Services),
			len(surface.Messages),
			len(surface.IntraScenarioImports),
			len(surface.CrossScenarioImports),
		),
		fmt.Sprintf("transport_world=%s", surface.TransportWorld.String()),
	}
	for _, service := range surface.Services {
		results = append(results, fmt.Sprintf("service %s (%s): %d rpc(s)", service.Name, service.Domain, len(service.Rpcs)))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Proto surface for %s.", surface.Scenario)},
		ResultsHeading: "Surface",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`validate scenario %s` - run policy checks against this surface", name),
		},
	})
}
