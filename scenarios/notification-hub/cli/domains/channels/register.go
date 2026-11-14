package channels

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing/routing_v1connect"
)

const GroupName = "channels"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h, base := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewRoutingServiceClient(h, base)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"RoutingService.ChannelsStatus": func(ctx cliapp.RunContext) error {
			resp, callErr := client.ChannelsStatus(context.Background(), connect.NewRequest(&v1.ChannelsStatusRequest{MachineId: ctx.Flag("machine-id")}))
			if callErr != nil {
				return cliapp.WrapAPIError("read channel status", callErr, nil)
			}
			rows := make([]string, 0, len(resp.Msg.GetChannels()))
			for _, channel := range resp.Msg.GetChannels() {
				rows = append(rows, fmt.Sprintf("%s: %s (%s)", channel.GetChannel(), channel.GetDisposition().String(), channel.GetReason()))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Channel dispositions"}, ResultsHeading: "Channels", Results: rows})
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("channels: load manifest: %w", err)
	}
	return group, nil
}
