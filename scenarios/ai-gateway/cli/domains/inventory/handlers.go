package inventory

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory/inventory_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client inventoryconnect.InventoryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: inventoryconnect.NewInventoryServiceClient(httpClient, baseURL)}
}

func (h *handlers) roles(ctx cliapp.RunContext) error {
	resp, err := h.client.ListProviderRoles(context.Background(), connect.NewRequest(&inventoryv1.ListProviderRolesRequest{
		Provider: ctx.Flag("provider"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list provider roles", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no inventory response")
	}
	results := make([]string, 0, len(resp.Msg.GetRoles()))
	for _, role := range resp.Msg.GetRoles() {
		results = append(results, fmt.Sprintf("%s/%s locality=%s status=%s capabilities=%v", role.GetProvider(), role.GetRole(), role.GetLocality(), role.GetStatus(), role.GetCapabilities()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d provider role(s).", len(resp.Msg.GetRoles()))},
		ResultsHeading: "Provider roles",
		Results:        append(results, warnings(resp.Msg.GetWarnings())...),
		RetrievalHints: []string{"`inventory smoke --provider <provider>` — run a bounded resource smoke check"},
	})
}

func (h *handlers) smoke(ctx cliapp.RunContext) error {
	provider := ctx.Flag("provider")
	resp, err := h.client.SmokeProvider(context.Background(), connect.NewRequest(&inventoryv1.SmokeProviderRequest{Provider: provider}))
	if err != nil {
		return cliapp.WrapAPIError("smoke provider", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no smoke response")
	}
	results := []string{fmt.Sprintf("%s status=%s code=%s exit=%d", resp.Msg.GetProvider(), resp.Msg.GetStatus(), resp.Msg.GetCode(), resp.Msg.GetExitCode())}
	if msg := resp.Msg.GetMessage(); msg != "" {
		results = append(results, msg)
	}
	results = append(results, warnings(resp.Msg.GetWarnings())...)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Provider smoke status: %s.", resp.Msg.GetStatus())},
		ResultsHeading: "Smoke",
		Results:        results,
	})
}

func warnings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, warning := range values {
		out = append(out, "warning: "+warning)
	}
	return out
}
