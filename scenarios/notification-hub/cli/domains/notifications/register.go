package notifications

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

const GroupName = "notifications"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := &handlers{client: newClient(core)}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"NotificationsService.Send":  h.send,
		"NotificationsService.Relay": h.relay,
		"NotificationsService.Get":   h.get,
		"NotificationsService.List":  h.list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("notifications: load manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) relay(ctx cliapp.RunContext) error {
	resp, err := h.client.Relay(context.Background(), connect.NewRequest(&v1.RelayRequest{PayloadBase64: ctx.Flag("payload-base64")}))
	if err != nil {
		return cliapp.WrapAPIError("relay notification", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Accepted relayed notification %s.", resp.Msg.GetNotification().GetId())}})
}

type handlers struct {
	client connectv1.NotificationsServiceClient
}

func newClient(core *cliapp.ScenarioApp) connectv1.NotificationsServiceClient {
	h, base := cliapp.NewConnectHTTPClient(core)
	return connectv1.NewNotificationsServiceClient(h, base)
}

func (h *handlers) send(ctx cliapp.RunContext) error {
	digestWindow, _ := strconv.ParseInt(ctx.Flag("digest-window-seconds"), 10, 32)
	if scheduled := strings.TrimSpace(ctx.Flag("scheduled-at")); scheduled != "" {
		if _, err := time.Parse(time.RFC3339, scheduled); err != nil {
			return fmt.Errorf("scheduled-at must be RFC3339: %w", err)
		}
	}
	resp, err := h.client.Send(context.Background(), connect.NewRequest(&v1.SendRequest{Title: ctx.Flag("title"), Body: ctx.Flag("body"), Urgency: ctx.Flag("urgency"), SensitivityLabel: ctx.Flag("sensitivity-label"), IdempotencyKey: ctx.Flag("idempotency-key"), DedupeKey: ctx.Flag("dedupe-key"), ScheduledAt: ctx.Flag("scheduled-at"), DigestWindowSeconds: int32(digestWindow)}))
	if err != nil {
		return cliapp.WrapAPIError("send notification", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Accepted notification %s.", resp.Msg.GetNotification().GetId())}})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&v1.GetRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get notification", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Notification " + resp.Msg.GetNotification().GetId()}, ResultsHeading: "State", Results: []string{resp.Msg.GetNotification().GetState().String() + ": " + resp.Msg.GetNotification().GetReason()}})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	limit, _ := strconv.ParseInt(strings.TrimSpace(ctx.Flag("limit")), 10, 32)
	resp, err := h.client.List(context.Background(), connect.NewRequest(&v1.ListRequest{Limit: int32(limit)}))
	if err != nil {
		return cliapp.WrapAPIError("list notifications", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetNotifications()))
	for _, n := range resp.Msg.GetNotifications() {
		rows = append(rows, fmt.Sprintf("%s %s — %s", n.GetId(), n.GetState().String(), n.GetReason()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Notifications: %d", len(rows))}, ResultsHeading: "Timeline", Results: rows})
}
