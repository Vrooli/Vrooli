package recipients

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients/recipients_v1connect"
)

const GroupName = "recipients"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h, base := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewRecipientsServiceClient(h, base)
	bindings := map[string]func(cliapp.RunContext) error{
		"RecipientsService.GetRecipient": func(ctx cliapp.RunContext) error {
			resp, err := client.GetRecipient(context.Background(), connect.NewRequest(&v1.GetRecipientRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("get recipient", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Recipient %s (%s).", resp.Msg.GetId(), resp.Msg.GetTrustPosture())}})
		},
		"RecipientsService.RegisterPushSubscription": func(ctx cliapp.RunContext) error {
			resp, err := client.RegisterPushSubscription(context.Background(), connect.NewRequest(&v1.RegisterPushSubscriptionRequest{Endpoint: ctx.Flag("endpoint"), P256Dh: ctx.Flag("p256dh"), Auth: ctx.Flag("auth"), Origin: ctx.Flag("origin")}))
			if err != nil {
				return cliapp.WrapAPIError("register push subscription", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Push subscription registered."}})
		},
		"RecipientsService.RemovePushSubscription": func(ctx cliapp.RunContext) error {
			resp, err := client.RemovePushSubscription(context.Background(), connect.NewRequest(&v1.RemovePushSubscriptionRequest{Endpoint: ctx.Flag("endpoint")}))
			if err != nil {
				return cliapp.WrapAPIError("remove push subscription", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Push subscription removed."}})
		},
		"RecipientsService.ListDevices": func(ctx cliapp.RunContext) error {
			resp, err := client.ListDevices(context.Background(), connect.NewRequest(&v1.ListDevicesRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("list devices", err, nil)
			}
			rows := make([]string, 0, len(resp.Msg.GetDevices()))
			for _, device := range resp.Msg.GetDevices() {
				rows = append(rows, fmt.Sprintf("%s %s (%s): %s", device.GetId(), device.GetName(), device.GetMachineId(), strings.Join(device.GetChannels(), ",")))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Devices: %d", len(rows))}, ResultsHeading: "Devices", Results: rows})
		},
		"RecipientsService.UpsertDevice": func(ctx cliapp.RunContext) error {
			resp, err := client.UpsertDevice(context.Background(), connect.NewRequest(&v1.UpsertDeviceRequest{Id: ctx.Flag("id"), Name: ctx.Flag("name"), MachineId: ctx.Flag("machine-id")}))
			if err != nil {
				return cliapp.WrapAPIError("upsert device", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Device %s saved.", resp.Msg.GetId())}})
		},
		"RecipientsService.RemoveDevice": func(ctx cliapp.RunContext) error {
			resp, err := client.RemoveDevice(context.Background(), connect.NewRequest(&v1.RemoveDeviceRequest{Id: ctx.Positional("id")}))
			if err != nil {
				return cliapp.WrapAPIError("remove device", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Device removed."}})
		},
		"RecipientsService.UpsertChannelAddress": func(ctx cliapp.RunContext) error {
			resp, err := client.UpsertChannelAddress(context.Background(), connect.NewRequest(&v1.UpsertChannelAddressRequest{Id: ctx.Flag("id"), DeviceId: ctx.Flag("device-id"), Channel: ctx.Flag("channel"), Address: ctx.Flag("address"), ApprovedLabels: split(ctx.Flag("approved-labels"))}))
			if err != nil {
				return cliapp.WrapAPIError("upsert channel address", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Address %s saved.", resp.Msg.GetId())}})
		},
		"RecipientsService.RemoveChannelAddress": func(ctx cliapp.RunContext) error {
			resp, err := client.RemoveChannelAddress(context.Background(), connect.NewRequest(&v1.RemoveChannelAddressRequest{Id: ctx.Positional("id")}))
			if err != nil {
				return cliapp.WrapAPIError("remove channel address", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Channel address removed."}})
		},
		"RecipientsService.SetQuietWindow": func(ctx cliapp.RunContext) error {
			weekday, _ := strconv.ParseInt(ctx.Flag("weekday"), 10, 32)
			critical, _ := strconv.ParseBool(ctx.Flag("critical-override"))
			resp, err := client.SetQuietWindow(context.Background(), connect.NewRequest(&v1.SetQuietWindowRequest{Weekday: int32(weekday), Start: ctx.Flag("start"), End: ctx.Flag("end"), Timezone: ctx.Flag("timezone"), CriticalOverride: critical}))
			if err != nil {
				return cliapp.WrapAPIError("set quiet window", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Quiet window %s saved.", resp.Msg.GetId())}})
		},
		"RecipientsService.ListQuietWindows": func(ctx cliapp.RunContext) error {
			resp, err := client.ListQuietWindows(context.Background(), connect.NewRequest(&v1.ListQuietWindowsRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("list quiet windows", err, nil)
			}
			rows := make([]string, 0, len(resp.Msg.GetWindows()))
			for _, window := range resp.Msg.GetWindows() {
				rows = append(rows, fmt.Sprintf("%s weekday=%d %s-%s %s", window.GetId(), window.GetWeekday(), window.GetStart(), window.GetEnd(), window.GetTimezone()))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Quiet windows: %d", len(rows))}, ResultsHeading: "Quiet windows", Results: rows})
		},
		"RecipientsService.DeleteQuietWindow": func(ctx cliapp.RunContext) error {
			resp, err := client.DeleteQuietWindow(context.Background(), connect.NewRequest(&v1.DeleteQuietWindowRequest{Id: ctx.Positional("id")}))
			if err != nil {
				return cliapp.WrapAPIError("delete quiet window", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Quiet window deleted."}})
		},
		"RecipientsService.SetEscalationChain": func(ctx cliapp.RunContext) error {
			resp, err := client.SetEscalationChain(context.Background(), connect.NewRequest(&v1.SetEscalationChainRequest{Channels: split(ctx.Flag("channels"))}))
			if err != nil {
				return cliapp.WrapAPIError("set escalation chain", err, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Escalation steps: %d.", len(resp.Msg.GetSteps()))}})
		},
		"RecipientsService.GetEscalationChain": func(ctx cliapp.RunContext) error {
			resp, err := client.GetEscalationChain(context.Background(), connect.NewRequest(&v1.GetEscalationChainRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("get escalation chain", err, nil)
			}
			rows := make([]string, 0, len(resp.Msg.GetSteps()))
			for _, step := range resp.Msg.GetSteps() {
				rows = append(rows, fmt.Sprintf("%d: %s", step.GetOrdinal(), step.GetChannel()))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Escalation steps: %d", len(rows))}, ResultsHeading: "Escalation", Results: rows})
		},
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recipients: load manifest: %w", err)
	}
	return group, nil
}

func split(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
