package redemption

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

const notificationHubScenario = "notification-hub"

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type NotificationRelay struct {
	resolver URLResolver
	client   connect.HTTPClient
}

func NewNotificationRelay(resolver URLResolver, client connect.HTTPClient) *NotificationRelay {
	return &NotificationRelay{resolver: resolver, client: client}
}

func (r *NotificationRelay) Pending(ctx context.Context, value Redemption) error {
	if r == nil || r.resolver == nil || r.client == nil {
		return errorsUnavailable("notification relay")
	}
	baseURL, err := r.resolver.ResolveScenarioURLDefault(ctx, notificationHubScenario)
	if err != nil {
		return fmt.Errorf("resolve notification-hub: %w", err)
	}
	client := notificationsconnect.NewNotificationsServiceClient(r.client, baseURL)
	_, err = client.Send(ctx, connect.NewRequest(&notificationsv1.SendRequest{
		Title:            "Token redemption approval required",
		Body:             fmt.Sprintf("Review redemption %s for %d %s tokens.", value.ID, value.Amount, value.TokenTypeID),
		Urgency:          "normal",
		SensitivityLabel: "household",
		IdempotencyKey:   value.ID + ":approval-relay",
		DedupeKey:        value.ID,
	}))
	if err != nil {
		return fmt.Errorf("relay redemption approval: %w", err)
	}
	return nil
}

func errorsUnavailable(name string) error { return fmt.Errorf("%s unavailable", name) }

var _ Relay = (*NotificationRelay)(nil)
