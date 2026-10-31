package notifications_test

import (
	"context"
	"encoding/json"
	"net/http"
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
	const rawDeliveryError = "https://provider.example/secret?token=raw-machine-secret"
	_, err := db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'failed', ?, ?)`, "alice-notification", rawDeliveryError, createdAt)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, machine_id, provider_id, delivered_at) VALUES (?, ?, 'web_push', 'local', ?, ?)`, "receipt-1", "alice-notification", "https://push.example.invalid/subscription?secret=sentinel", createdAt)
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
	require.Empty(t, got.Msg.GetNotification().GetReason(), "recipient reads must not expose raw delivery errors")
	require.Len(t, got.Msg.GetReceipts(), 1)
	require.Empty(t, got.Msg.GetReceipts()[0].GetProviderId(), "public reads must not expose push endpoint URLs")
	require.Empty(t, got.Msg.GetReceipts()[0].GetMachineId(), "recipient-facing reads must not expose machine identifiers")
	internalNotification, internalReceipts, err := service.GetForRecipient(ctx, "alice-notification", "alice")
	require.NoError(t, err)
	require.Equal(t, rawDeliveryError, internalNotification.Reason, "redaction is projection-only")
	require.Equal(t, "https://push.example.invalid/subscription?secret=sentinel", internalReceipts[0].ProviderID, "redaction is projection-only; stored provider data remains available to the delivery owner")
	listed, err := client.List(ctx, connect.NewRequest(&v1.ListRequest{Limit: 50}))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetNotifications()[0].GetReason(), "recipient timeline reads must not expose raw delivery errors")
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
	require.Empty(t, owned.Msg.GetReceipts()[0].GetMachineId(), "configured-recipient reads never expose machine identifiers")
	unrelatedClient := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL, connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			request.Header().Set("X-Vrooli-Identity-Subject", "bob")
			return next(ctx, request)
		}
	})))
	unrelated, err := unrelatedClient.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "alice-notification"}))
	require.NoError(t, err)
	require.Empty(t, unrelated.Msg.GetReceipts()[0].GetProviderId(), "a different verified identity must not receive the configured recipient's endpoint URL")
	require.Empty(t, unrelated.Msg.GetReceipts()[0].GetMachineId(), "recipient-facing reads must not expose machine identifiers")
	service.SetDefaultRecipient("")
	_, err = client.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: "alice-notification"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code(), "an empty configured recipient must fail closed")
}

func TestConfiguredRecipientCanReadBoundedRedactedAttempts(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, id := range []string{"mine", "empty", "theirs"} {
		recipient := "alice"
		if id == "theirs" {
			recipient = "bob"
		}
		_, err := db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_window_seconds, created_at, updated_at) VALUES (?, ?, 'private title', 'private body', 'normal', 'private', ?, 300, ?, ?)`, id, recipient, id, now, now)
		require.NoError(t, err)
	}
	for i := 1; i <= 3; i++ {
		_, err := db.ExecContext(ctx, `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, created_at) VALUES ('mine', 'push', 'secret-machine', ?, 'retrying', 'https://provider.example/secret?content=protected-body', ?)`, i, now)
		require.NoError(t, err)
	}
	router := mux.NewRouter()
	notificationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	response, err := http.Get(server.URL + "/api/v1/notifications/mine/attempts?limit=2")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var payload struct {
		Attempts  []hub.RecipientDeliveryAttempt `json:"attempts"`
		Truncated bool                           `json:"truncated"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	require.Len(t, payload.Attempts, 2)
	require.True(t, payload.Truncated)
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "secret-machine")
	require.NotContains(t, string(encoded), "provider_id")
	require.NotContains(t, string(encoded), "provider.example")
	require.NotContains(t, string(encoded), "protected-body")
	require.NotContains(t, string(encoded), "reason")
	require.NotContains(t, string(encoded), "private body")
	emptyResponse, err := http.Get(server.URL + "/api/v1/notifications/empty/attempts")
	require.NoError(t, err)
	var emptyPayload struct {
		Attempts  []hub.RecipientDeliveryAttempt `json:"attempts"`
		Truncated bool                           `json:"truncated"`
	}
	require.NoError(t, json.NewDecoder(emptyResponse.Body).Decode(&emptyPayload))
	emptyResponse.Body.Close()
	require.NotNil(t, emptyPayload.Attempts, "notifications without recorded attempts return an empty list")
	require.Empty(t, emptyPayload.Attempts)
	require.False(t, emptyPayload.Truncated)
	for _, id := range []string{"theirs", "missing"} {
		response, err := http.Get(server.URL + "/api/v1/notifications/" + id + "/attempts")
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	}
	service.SetDefaultRecipient("")
	response, err = http.Get(server.URL + "/api/v1/notifications/mine/attempts")
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, http.StatusNotFound, response.StatusCode, "an empty configured recipient must fail closed")
}
