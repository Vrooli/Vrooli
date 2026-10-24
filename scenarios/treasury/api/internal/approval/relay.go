package approval

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

type NotificationRelay struct {
	client notificationsconnect.NotificationsServiceClient
}

func NewNotificationRelay(baseURL string, client *http.Client) (*NotificationRelay, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || client == nil {
		return nil, errors.New("notification-hub base URL and HTTP client are required")
	}
	return &NotificationRelay{client: notificationsconnect.NewNotificationsServiceClient(client, baseURL)}, nil
}

func (r *NotificationRelay) Relay(ctx context.Context, request Request) error {
	if r == nil || r.client == nil {
		return errors.New("notification relay unavailable")
	}
	_, err := r.client.Send(ctx, connect.NewRequest(&notificationsv1.SendRequest{
		Title:            "Treasury approval required",
		Body:             fmt.Sprintf("Approve %d %s minor units for %s in Treasury approval %s", request.AmountMinor, request.Currency, request.Counterparty, request.ID),
		Urgency:          "high",
		SensitivityLabel: "financial",
		IdempotencyKey:   request.ID + ":relay",
		DedupeKey:        request.ID,
	}))
	return err
}

type UnavailableRelay struct{ Cause error }

func (r UnavailableRelay) Relay(context.Context, Request) error {
	if r.Cause == nil {
		return errors.New("notification relay unavailable")
	}
	return fmt.Errorf("notification relay unavailable: %w", r.Cause)
}

var (
	_ Relay = (*NotificationRelay)(nil)
	_ Relay = UnavailableRelay{}
)
