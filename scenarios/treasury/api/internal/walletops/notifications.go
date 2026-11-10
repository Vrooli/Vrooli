package walletops

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

// NotificationHub sends durable, idempotent receipt notices to the operator's
// own Notification Hub recipient. It never includes a fiat valuation.
type NotificationHub struct {
	client notificationsconnect.NotificationsServiceClient
}

func NewNotificationHub(baseURL string, client *http.Client) (*NotificationHub, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || client == nil {
		return nil, errors.New("notification-hub URL and HTTP client are required")
	}
	return &NotificationHub{client: notificationsconnect.NewNotificationsServiceClient(client, baseURL)}, nil
}

// NewLeasedTestNotificationHub opts a local rehearsal into Notification Hub's
// routed test-owner boundary. The hub independently requires test mode and an
// active routed lease before it accepts the fixed fixture identity.
func NewLeasedTestNotificationHub(baseURL string) (*NotificationHub, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return nil, errors.New("leased test notification-hub URL must be local HTTP")
	}
	host := parsed.Hostname()
	if host != "localhost" && net.ParseIP(host) == nil || (net.ParseIP(host) != nil && !net.ParseIP(host).IsLoopback()) {
		return nil, errors.New("leased test notification-hub URL must use a loopback host")
	}
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		request = request.Clone(request.Context())
		request.Header.Set("X-Vrooli-Test-Mode", "1")
		return http.DefaultTransport.RoundTrip(request)
	})
	return NewNotificationHub(baseURL, &http.Client{Timeout: 5 * time.Second, Transport: transport})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (n *NotificationHub) Received(ctx context.Context, operator, budgetID, address, atomicAmount, receiptID string) error {
	if n == nil || n.client == nil || strings.TrimSpace(operator) == "" || strings.TrimSpace(budgetID) == "" || strings.TrimSpace(address) == "" || strings.TrimSpace(receiptID) == "" || !positiveAtoms.MatchString(atomicAmount) {
		return ErrInvalid
	}
	amount, ok := new(big.Int).SetString(atomicAmount, 10)
	if !ok {
		return ErrInvalid
	}
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(amount, big.NewInt(1_000_000), fraction)
	body := fmt.Sprintf("Received %s.%06d USDC at %s. The accepted receipt was credited to budget %s.", whole.String(), fraction.Int64(), address, budgetID)
	_, err := n.client.Send(ctx, connect.NewRequest(&notificationsv1.SendRequest{
		Title:            "Treasury funds received",
		Body:             body,
		Urgency:          "normal",
		SensitivityLabel: "financial",
		IdempotencyKey:   receiptID + ":funding-notice",
		DedupeKey:        receiptID,
	}))
	return err
}

var _ Notification = (*NotificationHub)(nil)
