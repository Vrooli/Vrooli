package walletops

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

type notificationTokenFixture struct {
	token string
	err   error
}

func (f notificationTokenFixture) Token(context.Context) (string, error) { return f.token, f.err }

type notificationServerFixture struct {
	notificationsconnect.UnimplementedNotificationsServiceHandler
	authorization string
}

func (f *notificationServerFixture) Send(_ context.Context, req *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
	f.authorization = req.Header().Get("Authorization")
	return connect.NewResponse(&notificationsv1.SendResponse{}), nil
}

func TestNewLeasedTestNotificationHubRequiresLoopbackHTTP(t *testing.T) {
	for _, endpoint := range []string{
		"https://127.0.0.1:8443",
		"http://example.com:3000",
		"file:///tmp/notification-hub",
	} {
		if _, err := NewLeasedTestNotificationHub(endpoint, "cf5-local-agent"); err == nil {
			t.Fatalf("NewLeasedTestNotificationHub(%q) unexpectedly accepted a non-local endpoint", endpoint)
		}
	}
	for _, endpoint := range []string{"http://127.0.0.1:3000", "http://[::1]:3000", "http://localhost:3000"} {
		if _, err := NewLeasedTestNotificationHub(endpoint, "cf5-local-agent"); err != nil {
			t.Fatalf("NewLeasedTestNotificationHub(%q): %v", endpoint, err)
		}
	}
	if _, err := NewLeasedTestNotificationHub("http://127.0.0.1:3000", " "); err == nil {
		t.Fatal("leased test hub must require an isolated scoped producer identity")
	}
}

func TestAuthenticatedNotificationHubUsesDedicatedServiceToken(t *testing.T) {
	service := &notificationServerFixture{}
	_, handler := notificationsconnect.NewNotificationsServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	hub, err := NewAuthenticatedNotificationHub(server.URL, server.Client(), notificationTokenFixture{token: "dedicated-notice-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hub.Received(context.Background(), "operator", "budget-1", "0xabc", "2500000", "receipt-1"); err != nil {
		t.Fatal(err)
	}
	if service.authorization != "Bearer dedicated-notice-token" {
		t.Fatalf("Authorization = %q", service.authorization)
	}
}

func TestAuthenticatedNotificationHubFailsClosedWhenServiceIdentityUnavailable(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	hub, err := NewAuthenticatedNotificationHub(server.URL, server.Client(), notificationTokenFixture{err: errors.New("credential absent")})
	if err != nil {
		t.Fatal(err)
	}
	if err := hub.Received(context.Background(), "operator", "budget-1", "0xabc", "2500000", "receipt-1"); err == nil {
		t.Fatal("missing identity unexpectedly sent a notification")
	}
	if requests != 0 {
		t.Fatalf("hub received %d unauthenticated requests", requests)
	}
	if _, err := NewAuthenticatedNotificationHub(server.URL, server.Client(), nil); err == nil {
		t.Fatal("nil service identity was accepted")
	}
}
