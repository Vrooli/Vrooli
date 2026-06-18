package discovery

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery/discovery_v1connect"
)

type handlers struct {
	client discoveryconnect.DiscoveryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: discoveryconnect.NewDiscoveryServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) resources(ctx cliapp.RunContext) error {
	resp, err := h.client.ListResources(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListResourcesRequest{Refresh: ctx.BoolFlag("refresh")}),
	)
	if err != nil {
		return err
	}
	out := ctx.Stdout()
	fmt.Fprintf(out, "%d resource(s):\n", resp.Msg.GetCount())
	for _, r := range resp.Msg.GetResources() {
		health := "down"
		if r.GetHealthy() {
			health = "ok"
		}
		fmt.Fprintf(out, "  %-28s [%s] %s %s\n", r.GetName(), health, r.GetCategory(), r.GetStatus())
	}
	return nil
}

func (h *handlers) scenarios(ctx cliapp.RunContext) error {
	resp, err := h.client.ListScenarios(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListScenariosRequest{Refresh: ctx.BoolFlag("refresh")}),
	)
	if err != nil {
		return err
	}
	out := ctx.Stdout()
	fmt.Fprintf(out, "%d scenario(s):\n", resp.Msg.GetCount())
	for _, s := range resp.Msg.GetScenarios() {
		fmt.Fprintf(out, "  %-28s %s %s\n", s.GetName(), s.GetCategory(), s.GetStatus())
	}
	return nil
}
