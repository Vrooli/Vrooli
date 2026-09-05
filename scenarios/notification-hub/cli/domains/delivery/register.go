package delivery

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery/delivery_v1connect"
)

const GroupName = "delivery"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h, base := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewDeliveryServiceClient(h, base)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"DeliveryService.Deliver": func(ctx cliapp.RunContext) error {
			resp, callErr := client.Deliver(context.Background(), connect.NewRequest(&v1.DeliverRequest{Id: ctx.Positional("id")}))
			if callErr != nil {
				return cliapp.WrapAPIError("deliver notification", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Delivery receipts: %d.", len(resp.Msg.GetReceipts()))}})
		},
		"DeliveryService.GetTimeline": func(ctx cliapp.RunContext) error {
			resp, callErr := client.GetTimeline(context.Background(), connect.NewRequest(&v1.GetTimelineRequest{Limit: 50}))
			if callErr != nil {
				return cliapp.WrapAPIError("read delivery timeline", callErr, nil)
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Timeline notifications: %d", len(resp.Msg.GetNotifications()))}, ResultsHeading: "Notifications"})
		},
		"DeliveryService.GetAnalytics": func(ctx cliapp.RunContext) error {
			resp, callErr := client.GetAnalytics(context.Background(), connect.NewRequest(&v1.GetAnalyticsRequest{Since: ctx.Flag("since"), Until: ctx.Flag("until")}))
			if callErr != nil {
				return cliapp.WrapAPIError("read delivery analytics", callErr, nil)
			}
			rows := make([]string, 0, len(resp.Msg.GetChannels()))
			for _, channel := range resp.Msg.GetChannels() {
				rows = append(rows, fmt.Sprintf("%s delivered=%d failed=%d attempts=%d failure_rate=%s latency_ms=%s", channel.GetChannel(), channel.GetDelivered(), channel.GetFailed(), channel.GetAttempts(), strconv.FormatFloat(channel.GetFailureRate(), 'f', 3, 64), strconv.FormatFloat(channel.GetAverageLatencyMs(), 'f', 1, 64)))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Notifications: %d", resp.Msg.GetTotalNotifications())}, ResultsHeading: "Analytics", Results: rows})
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("delivery: load manifest: %w", err)
	}
	return group, nil
}
