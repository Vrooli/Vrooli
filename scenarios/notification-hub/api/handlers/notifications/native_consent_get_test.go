package notifications_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"

	notificationsH "notification-hub/handlers/notifications"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

// The delivery record leads with a review button only when the Hub itself says the
// notification is a native permission request. The page never infers it.
func TestGet_NamesTheReviewPageOnlyForAQualifyingNativePermissionRequest(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	router := mux.NewRouter()
	notificationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL)

	id := uuid.NewString()
	page := "https://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"
	send := func(key string, in hub.SendInput) string {
		in.RequestedBy, in.Title, in.Body, in.SensitivityLabel, in.IdempotencyKey = "alice", "Permission", "Exact scope", "private", key
		n, err := service.Send(ctx, in)
		require.NoError(t, err)
		return n.ID
	}
	get := func(notificationID string) string {
		got, err := client.Get(ctx, connect.NewRequest(&v1.GetRequest{Id: notificationID}))
		require.NoError(t, err)
		return got.Msg.GetNativeConsentReviewUrl()
	}

	qualifying := send("native-ok", hub.SendInput{DedupeKey: id, SourceScenario: "agent-manager", Category: "authorization-requested", ContextURL: page})
	require.Equal(t, page, get(qualifying))

	for name, in := range map[string]hub.SendInput{
		"wrong source":               {DedupeKey: id, SourceScenario: "reports", Category: "authorization-requested", ContextURL: page},
		"wrong category":             {DedupeKey: id, SourceScenario: "agent-manager", Category: "completed", ContextURL: page},
		"dedupe key not in the link": {DedupeKey: uuid.NewString(), SourceScenario: "agent-manager", Category: "authorization-requested", ContextURL: page},
		"no source":                  {DedupeKey: id, ContextURL: page},
	} {
		require.Empty(t, get(send("refused-"+name, in)), name)
	}
}
