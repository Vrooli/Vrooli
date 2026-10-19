package notifications_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"

	notificationsH "notification-hub/handlers/notifications"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func TestConfiguredRecipientCanReadNotificationsWithoutIdentity(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	insert := func(id, recipient, contextURL string) {
		_, err := db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, context_url, created_at, updated_at) VALUES (?, ?, ?, 'body', 'normal', 'public', ?, '', 300, ?, ?, ?)`, id, recipient, id, id, contextURL, createdAt, createdAt)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'accepted', ?)`, id, createdAt)
		require.NoError(t, err)
	}
	insert("alice-notification", "alice", "https://agent-manager.example/runs/run-42")
	insert("bob-notification", "bob", "javascript:alert(1)")
	_, err := db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, machine_id, provider_id, delivered_at) VALUES (?, ?, 'web_push', 'local', ?, ?)`, "receipt-1", "alice-notification", "https://push.example.invalid/subscription?secret=sentinel", createdAt)
	require.NoError(t, err)

	router := mux.NewRouter()
	notificationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL)

	list, err := client.List(ctx, connect.NewRequest(&v1.ListRequest{Limit: 50}))
	require.NoError(t, err, "ordinary notification reads must not require an owner login")
	require.Len(t, list.Msg.GetNotifications(), 1)
	require.Equal(t, "alice-notification", list.Msg.GetNotifications()[0].GetId())
	require.Equal(t, "https://agent-manager.example/runs/run-42", list.Msg.GetNotifications()[0].GetContextUrl())

	got, err := client.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "alice-notification"}))
	require.NoError(t, err)
	require.Equal(t, "alice", got.Msg.GetNotification().GetRequestedBy())
	require.Equal(t, "https://agent-manager.example/runs/run-42", got.Msg.GetNotification().GetContextUrl())
	require.Len(t, got.Msg.GetReceipts(), 1)
	require.Empty(t, got.Msg.GetReceipts()[0].GetProviderId(), "public reads must not expose push endpoint URLs")
	_, internalReceipts, err := service.GetForRecipient(ctx, "alice-notification", "alice")
	require.NoError(t, err)
	require.Equal(t, "https://push.example.invalid/subscription?secret=sentinel", internalReceipts[0].ProviderID, "redaction is projection-only; stored provider data remains available to the delivery owner")
	_, err = client.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "bob-notification"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	ownerClient := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL, connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			request.Header().Set("X-Vrooli-Identity-Subject", "alice")
			return next(ctx, request)
		}
	})))
	owned, err := ownerClient.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "alice-notification"}))
	require.NoError(t, err)
	require.Empty(t, owned.Msg.GetReceipts()[0].GetProviderId(), "configured-recipient reads never expose endpoint URLs, even when an identity header is present")
	unrelatedClient := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL, connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			request.Header().Set("X-Vrooli-Identity-Subject", "bob")
			return next(ctx, request)
		}
	})))
	unrelated, err := unrelatedClient.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "alice-notification"}))
	require.NoError(t, err)
	require.Empty(t, unrelated.Msg.GetReceipts()[0].GetProviderId(), "a different verified identity must not receive the configured recipient's endpoint URL")
}
