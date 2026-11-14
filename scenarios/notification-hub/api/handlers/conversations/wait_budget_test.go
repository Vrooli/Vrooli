package conversations_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"

	conversationsH "notification-hub/handlers/conversations"
	hub "notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

// A wait that outlasts the server's ordinary write deadline must still be
// answered, not cut with an EOF. The server here has the deadline the real one
// has, scaled down; the wait is longer than it.
func TestWait_OutlastsTheServersOrdinaryWriteDeadline(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	service.SetClockForTest(schedule.System())
	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewUnstartedServer(router)
	server.Config.WriteTimeout = 200 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)

	ctx := context.Background()
	request := formAskRequest("wait-budget-1")
	request.SubjectKey = "wait-budget"
	asked, err := client.Ask(ctx, as("alice", request))
	require.NoError(t, err)
	service.SetWaitSliceForTest(700 * time.Millisecond)

	waited, err := client.Wait(ctx, as("alice", &v1.WaitRequest{
		AskId: asked.Msg.GetAskId(), Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	}))
	require.NoError(t, err, "the response of a long wait must be written after the ordinary write deadline")
	require.Equal(t, "pending", waited.Msg.GetState())
	require.Equal(t, hub.WaitSliceReason, waited.Msg.GetReason())
}
