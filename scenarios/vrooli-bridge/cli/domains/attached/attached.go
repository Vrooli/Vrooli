package attached

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
	"vrooli-bridge/cli/internal/session"
)

const groupName = "attached"

type handlers struct {
	client attachedconnect.AttachedDeviceServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := &handlers{}
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	h.client = attachedconnect.NewAttachedDeviceServiceClient(httpClient, baseURL)
	bindings := map[string]func(cliapp.RunContext) error{
		"AttachedDeviceService.ListAttachedDevices":  h.list,
		"AttachedDeviceService.PairAttachedDevice":   h.pair,
		"AttachedDeviceService.RevokeAttachedDevice": h.revoke,
	}
	group, err := cliapp.LoadFromManifest(manifest, groupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("attached: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAttachedDevices(context.Background(), connect.NewRequest(&attachedv1.ListAttachedDevicesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list attached devices", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no attached-device response")
	}
	results := make([]string, 0, len(resp.Msg.Devices))
	for _, device := range resp.Msg.Devices {
		results = append(results, fmt.Sprintf("%s · serial=%s · transport=%s · host=%s · reachability=%s", device.Name, device.Serial, device.Transport, device.HostNodeId, device.Reachability))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d attached device(s).", len(resp.Msg.Devices))}, ResultsHeading: "Attached devices", Results: results})
}

func (h *handlers) pair(ctx cliapp.RunContext) error {
	resp, err := h.client.PairAttachedDevice(context.Background(), connect.NewRequest(&attachedv1.PairAttachedDeviceRequest{
		Name:           ctx.Flag("name"),
		HostNodeId:     ctx.Flag("host-node-id"),
		Kind:           ctx.Flag("kind"),
		Transport:      ctx.Flag("transport"),
		Serial:         ctx.Flag("serial"),
		OsVersion:      ctx.Flag("os-version"),
		HostNodeOnline: ctx.BoolFlag("host-node-online"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("pair attached device", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no attached-device response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Attached device paired."}, Changes: []string{fmt.Sprintf("id: %s", resp.Msg.Device.GetId()), fmt.Sprintf("serial: %s", resp.Msg.Device.GetSerial()), fmt.Sprintf("host: %s", resp.Msg.Device.GetHostNodeId())}})
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	resp, err := h.client.RevokeAttachedDevice(context.Background(), connect.NewRequest(&attachedv1.RevokeAttachedDeviceRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("revoke attached device", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no attached-device response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Attached device revoked."}, Changes: []string{fmt.Sprintf("id: %s", resp.Msg.Device.GetId()), fmt.Sprintf("reachability: %s", resp.Msg.Device.GetReachability())}})
}
