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

type producerTokenContextKey struct{}

// WithProducerToken carries the verified run's identity to Notification Hub.
// The token is passed only for the synchronous relay call and is never stored
// with approval or relay-attempt records.
func WithProducerToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, producerTokenContextKey{}, strings.TrimSpace(token))
}

type NotificationRelay struct {
	client   notificationsconnect.NotificationsServiceClient
	testMode bool
}

func NewNotificationRelay(baseURL string, client *http.Client) (*NotificationRelay, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || client == nil {
		return nil, errors.New("notification-hub base URL and HTTP client are required")
	}
	return &NotificationRelay{client: notificationsconnect.NewNotificationsServiceClient(client, baseURL)}, nil
}

// NewLeasedTestNotificationRelay directs owner rehearsals into Notification
// Hub's explicit loopback test database lease. Production callers must use
// NewNotificationRelay and cannot enable the routing header implicitly.
func NewLeasedTestNotificationRelay(baseURL string, client *http.Client) (*NotificationRelay, error) {
	parsed, err := url.ParseRequestURI(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" && parsed.Hostname() != "::1" || client == nil {
		return nil, errors.New("leased test notification relay requires a loopback HTTP endpoint and client")
	}
	relay, err := NewNotificationRelay(baseURL, client)
	if err != nil {
		return nil, err
	}
	relay.testMode = true
	return relay, nil
}

func (r *NotificationRelay) Relay(ctx context.Context, request Request) error {
	if r == nil || r.client == nil {
		return errors.New("notification relay unavailable")
	}
	token, _ := ctx.Value(producerTokenContextKey{}).(string)
	if token == "" {
		return errors.New("verified Notification Hub producer token is required")
	}
	message := connect.NewRequest(&notificationsv1.SendRequest{
		Title:            "Treasury approval required",
		Body:             fmt.Sprintf("Approve %d %s minor units for %s in Treasury approval %s", request.AmountMinor, request.Currency, request.Counterparty, request.ID),
		Urgency:          "high",
		SensitivityLabel: "financial",
		IdempotencyKey:   request.ID + ":relay",
		DedupeKey:        request.ID,
	})
	message.Header().Set("X-Agent-Identity-Token", token)
	if r.testMode {
		message.Header().Set("X-Vrooli-Test-Mode", "1")
	}
	_, err := r.client.Send(ctx, message)
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
