package devices

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory/inventory_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client inventoryconnect.InventoryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: inventoryconnect.NewInventoryServiceClient(httpClient, baseURL),
	}
}

func (h handlers) refresh(ctx cliapp.RunContext) error {
	resp, err := h.client.RefreshInventory(context.Background(), connect.NewRequest(&inventoryv1.RefreshInventoryRequest{DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("refresh inventory", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Inventory refresh complete."}, ResultsHeading: "Devices", Results: formatDevices(resp.Msg.GetDevices()), RetrievalHints: resp.Msg.GetFindings()})
}

func (h handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDevices(context.Background(), connect.NewRequest(&inventoryv1.ListDevicesRequest{Group: ctx.Flag("group")}))
	if err != nil {
		return cliapp.WrapAPIError("list devices", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Fetched devices."}, ResultsHeading: "Devices", Results: formatDevices(resp.Msg.GetDevices())})
}

func (h handlers) group(ctx cliapp.RunContext) error {
	resp, err := h.client.UpdateDeviceGroup(context.Background(), connect.NewRequest(&inventoryv1.UpdateDeviceGroupRequest{Id: ctx.Positional("id"), Group: ctx.Flag("group")}))
	if err != nil {
		return cliapp.WrapAPIError("update device group", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Updated device group."}, Changes: []string{formatDevice(resp.Msg.GetDevice())}})
}

func (h handlers) explain(ctx cliapp.RunContext) error {
	resp, err := h.client.ExplainDeviceIdentity(context.Background(), connect.NewRequest(&inventoryv1.ExplainDeviceIdentityRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("explain device identity", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{formatDevice(resp.Msg.GetDevice())}, ResultsHeading: "Evidence", Results: resp.Msg.GetEvidence()})
}

func formatDevices(devices []*inventoryv1.Device) []string {
	out := make([]string, 0, len(devices))
	for _, d := range devices {
		out = append(out, formatDevice(d))
	}
	return out
}

func formatDevice(d *inventoryv1.Device) string {
	if d == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s host=%s ip=%s group=%s confidence=%s", d.GetId(), d.GetHostname(), d.GetIpAddress(), d.GetGroup(), d.GetIdentityConfidence())
}
