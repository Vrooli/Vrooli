package devices

import (
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliapp"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory/inventory_v1connect"
)

type handlers struct{ core *cliapp.ScenarioApp }

func (h handlers) refresh(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*inventoryv1.RefreshInventoryRequest, *inventoryv1.RefreshInventoryResponse](h.core, http.MethodPost, inventoryconnect.InventoryServiceRefreshInventoryProcedure, &inventoryv1.RefreshInventoryRequest{DryRun: ctx.BoolFlag("dry-run")})
	if err != nil {
		return cliapp.WrapAPIError("refresh inventory", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Inventory refresh complete."}, ResultsHeading: "Devices", Results: formatDevices(resp.GetDevices()), RetrievalHints: resp.GetFindings()})
}

func (h handlers) list(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*inventoryv1.ListDevicesRequest, *inventoryv1.ListDevicesResponse](h.core, http.MethodPost, inventoryconnect.InventoryServiceListDevicesProcedure, &inventoryv1.ListDevicesRequest{Group: ctx.Flag("group")})
	if err != nil {
		return cliapp.WrapAPIError("list devices", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{"Fetched devices."}, ResultsHeading: "Devices", Results: formatDevices(resp.GetDevices())})
}

func (h handlers) group(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*inventoryv1.UpdateDeviceGroupRequest, *inventoryv1.UpdateDeviceGroupResponse](h.core, http.MethodPost, inventoryconnect.InventoryServiceUpdateDeviceGroupProcedure, &inventoryv1.UpdateDeviceGroupRequest{Id: ctx.Positional("id"), Group: ctx.Flag("group")})
	if err != nil {
		return cliapp.WrapAPIError("update device group", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp, cliapp.MutationReport{Result: []string{"Updated device group."}, Changes: []string{formatDevice(resp.GetDevice())}})
}

func (h handlers) explain(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*inventoryv1.ExplainDeviceIdentityRequest, *inventoryv1.ExplainDeviceIdentityResponse](h.core, http.MethodPost, inventoryconnect.InventoryServiceExplainDeviceIdentityProcedure, &inventoryv1.ExplainDeviceIdentityRequest{Id: ctx.Positional("id")})
	if err != nil {
		return cliapp.WrapAPIError("explain device identity", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{formatDevice(resp.GetDevice())}, ResultsHeading: "Evidence", Results: resp.GetEvidence()})
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
