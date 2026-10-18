package delivery_test

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
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery"
	deliveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery/delivery_v1connect"

	deliveryH "notification-hub/handlers/delivery"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func TestConfiguredRecipientCanReadDeliveryViewsWithoutIdentity(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	insert := func(id, recipient string) {
		t.Helper()
		_, err := db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, created_at, updated_at) VALUES (?, ?, ?, 'body', 'normal', 'public', ?, '', 300, ?, ?)`, id, recipient, id, id, createdAt, createdAt)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'accepted', ?)`, id, createdAt)
		require.NoError(t, err)
	}
	insert("alice-delivery", "alice")
	insert("bob-delivery", "bob")

	router := mux.NewRouter()
	deliveryH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := deliveryconnect.NewDeliveryServiceClient(server.Client(), server.URL)

	timeline, err := client.GetTimeline(ctx, connect.NewRequest(&v1.GetTimelineRequest{Limit: 50}))
	require.NoError(t, err, "ordinary delivery reads must not require an owner login")
	require.Len(t, timeline.Msg.GetNotifications(), 1)
	require.Equal(t, "alice-delivery", timeline.Msg.GetNotifications()[0].GetId())

	analytics, err := client.GetAnalytics(ctx, connect.NewRequest(&v1.GetAnalyticsRequest{}))
	require.NoError(t, err)
	require.EqualValues(t, 1, analytics.Msg.GetTotalNotifications())
}
