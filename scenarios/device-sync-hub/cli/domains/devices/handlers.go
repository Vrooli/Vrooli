package devices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client devicesconnect.DevicesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: devicesconnect.NewDevicesServiceClient(httpClient, baseURL),
	}
}

// setup calls DevicesService.SetupOwnerDevice (owner-authed): it claims the hub
// for the signed-in owner if unclaimed and trusts this client directly. The
// owner JWT rides the configured token source (populate it with `auth login`).
// The one-time hub device token is printed for the caller to export as
// $DEVICE_SYNC_HUB_DEVICE_TOKEN before running `transfer` commands.
func (h *handlers) setup(ctx cliapp.RunContext) error {
	resp, err := h.client.SetupOwnerDevice(context.Background(), connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{
		Profile: profileFromFlags(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("set up owner device (run `auth login` first if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Set up hub owner device %s (TRUSTED).", resp.Msg.Device.Id)},
		Changes: []string{
			formatDevice(resp.Msg.Device),
			fmt.Sprintf("device_token=%s (shown once)", resp.Msg.DeviceToken),
		},
		NextCommand: []string{
			fmt.Sprintf("export DEVICE_SYNC_HUB_DEVICE_TOKEN=%s", resp.Msg.DeviceToken),
			"`devices pair --name <name>` — issue a code for an additional device",
		},
	})
}

// list calls DevicesService.ListDevices. Owner-authed: cli-core attaches the
// owner JWT from the configured token source.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDevices(context.Background(), connect.NewRequest(&devicesv1.ListDevicesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list devices", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no devices response")
	}
	results := make([]string, 0, len(resp.Msg.Devices))
	for _, d := range resp.Msg.Devices {
		results = append(results, formatDevice(d))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d device(s) in the trust group.", len(resp.Msg.Devices))},
		ResultsHeading: "Devices",
		Results:        results,
		RetrievalHints: []string{
			"`devices get <id>` — show a single device",
			"`devices pair --name <name>` — issue a pairing code for a new device",
			"`devices revoke <device-id>` — sever a device's access",
		},
	})
}

// get calls DevicesService.GetDevice for a single device id.
func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDevice(context.Background(), connect.NewRequest(&devicesv1.GetDeviceRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get device %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched device %s.", resp.Msg.Device.Id)},
		ResultsHeading: "Device",
		Results:        []string{formatDevice(resp.Msg.Device)},
	})
}

// pair calls DevicesService.IssuePairingCode (owner-authed). The raw code is
// returned only once, here — render it for the owner to type or QR-encode on
// the joining device.
func (h *handlers) pair(ctx cliapp.RunContext) error {
	resp, err := h.client.IssuePairingCode(context.Background(), connect.NewRequest(&devicesv1.IssuePairingCodeRequest{
		DeviceName: strings.TrimSpace(ctx.Flag("name")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("issue pairing code", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.PairingCode == nil {
		return fmt.Errorf("server returned no pairing code")
	}
	pc := resp.Msg.PairingCode
	expires := ""
	if pc.ExpiresAt != nil {
		expires = pc.ExpiresAt.AsTime().Format(time.RFC3339)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Issued a single-use pairing code."},
		Changes: []string{fmt.Sprintf("code=%s expires=%s", pc.Code, expires)},
		NextCommand: []string{
			fmt.Sprintf("On the new device: `device-sync-hub devices redeem --code %s --name <device-name>`", pc.Code),
			"`devices list` — see the device once it redeems",
		},
	})
}

// redeem calls DevicesService.RedeemPairingCode as a NEW device. No owner token
// is required. The one-time hub device token is printed for the caller to
// export as $DEVICE_SYNC_HUB_DEVICE_TOKEN before running `transfer` commands.
func (h *handlers) redeem(ctx cliapp.RunContext) error {
	resp, err := h.client.RedeemPairingCode(context.Background(), connect.NewRequest(&devicesv1.RedeemPairingCodeRequest{
		Code:    strings.TrimSpace(ctx.Flag("code")),
		Profile: profileFromFlags(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("redeem pairing code", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Paired as device %s (TRUSTED).", resp.Msg.Device.Id)},
		Changes: []string{
			formatDevice(resp.Msg.Device),
			fmt.Sprintf("device_token=%s (shown once)", resp.Msg.DeviceToken),
		},
		NextCommand: []string{
			fmt.Sprintf("export DEVICE_SYNC_HUB_DEVICE_TOKEN=%s", resp.Msg.DeviceToken),
			"`transfer list` — see what's waiting for this device",
		},
	})
}

// request calls DevicesService.RequestPairing as a NEW device (fallback path).
// The returned token is inert until the owner approves; it activates in place.
func (h *handlers) request(ctx cliapp.RunContext) error {
	resp, err := h.client.RequestPairing(context.Background(), connect.NewRequest(&devicesv1.RequestPairingRequest{
		Profile: profileFromFlags(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("request pairing", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Requested pairing as device %s (PENDING owner approval).", resp.Msg.Device.Id)},
		Changes: []string{
			formatDevice(resp.Msg.Device),
			fmt.Sprintf("device_token=%s (inert until approved)", resp.Msg.DeviceToken),
		},
		NextCommand: []string{
			fmt.Sprintf("Ask the owner to run: `device-sync-hub devices approve %s`", resp.Msg.Device.Id),
		},
	})
}

// approve calls DevicesService.ApprovePairing (owner-authed), promoting a
// PENDING device to TRUSTED.
func (h *handlers) approve(ctx cliapp.RunContext) error {
	id := ctx.Positional("device-id")
	resp, err := h.client.ApprovePairing(context.Background(), connect.NewRequest(&devicesv1.ApprovePairingRequest{
		DeviceId: id,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("approve device %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Approved device %s (now TRUSTED).", resp.Msg.Device.Id)},
		Changes:     []string{formatDevice(resp.Msg.Device)},
		NextCommand: []string{"`devices list` — confirm the trust group"},
	})
}

// rename calls DevicesService.RenameDevice (owner-authed).
func (h *handlers) rename(ctx cliapp.RunContext) error {
	id := ctx.Positional("device-id")
	resp, err := h.client.RenameDevice(context.Background(), connect.NewRequest(&devicesv1.RenameDeviceRequest{
		DeviceId: id,
		Name:     strings.TrimSpace(ctx.Flag("name")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("rename device %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Renamed device %s.", resp.Msg.Device.Id)},
		Changes: []string{formatDevice(resp.Msg.Device)},
	})
}

// revoke calls DevicesService.RevokeDevice (owner-authed). Destructive: severs
// the device's access immediately and revokes its authenticator session.
func (h *handlers) revoke(ctx cliapp.RunContext) error {
	id := ctx.Positional("device-id")
	resp, err := h.client.RevokeDevice(context.Background(), connect.NewRequest(&devicesv1.RevokeDeviceRequest{
		DeviceId: id,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("revoke device %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Device == nil {
		return fmt.Errorf("server returned no device")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Revoked device %s — access severed.", resp.Msg.Device.Id)},
		Changes:     []string{formatDevice(resp.Msg.Device)},
		NextCommand: []string{"`devices list` — the device now shows REVOKED"},
	})
}

// profileFromFlags builds the self-description a joining device supplies on both
// pairing paths.
func profileFromFlags(ctx cliapp.RunContext) *devicesv1.DeviceProfile {
	return &devicesv1.DeviceProfile{
		DeviceName: strings.TrimSpace(ctx.Flag("device-name")),
		Kind:       strings.TrimSpace(ctx.Flag("kind")),
		Platform:   strings.TrimSpace(ctx.Flag("platform")),
	}
}

// trustStateLabel renders the trust lifecycle position as the short human token
// (TRUSTED/PENDING/REVOKED) rather than the verbose proto enum name.
func trustStateLabel(s devicesv1.TrustState) string {
	switch s {
	case devicesv1.TrustState_TRUST_STATE_PENDING:
		return "PENDING"
	case devicesv1.TrustState_TRUST_STATE_TRUSTED:
		return "TRUSTED"
	case devicesv1.TrustState_TRUST_STATE_REVOKED:
		return "REVOKED"
	default:
		return "UNKNOWN"
	}
}

// formatDevice produces a one-line representation suitable for both ListReport
// and MutationReport result blocks.
func formatDevice(d *devicesv1.Device) string {
	if d == nil {
		return "(nil)"
	}
	presence := "offline"
	if d.Online {
		presence = "online"
	}
	name := d.Name
	if name == "" {
		name = "(unnamed)"
	}
	return fmt.Sprintf("%s — %s [%s, %s, kind=%s]", d.Id, name, trustStateLabel(d.TrustState), presence, d.Kind)
}
