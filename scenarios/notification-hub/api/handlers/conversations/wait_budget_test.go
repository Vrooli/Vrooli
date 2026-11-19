package conversations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/schedule"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

// Maintenance yields an admitted long poll with the same resumable open result
// as a normal wait slice, allowing the provider to drain without losing an ask.
func TestWait_YieldsToMaintenanceAndCanResume(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	service.SetClockForTest(schedule.System())
	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	entered := make(chan struct{}, 1)
	application := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == conversationsconnect.ConversationsServiceWaitProcedure {
			entered <- struct{}{}
		}
		router.ServeHTTP(w, r)
	})
	provider, err := apilifecycle.NewProvider("notification-hub", "test-instance", apilifecycle.Hooks{})
	require.NoError(t, err)
	server := httptest.NewServer(provider.Mount(application))
	t.Cleanup(server.Close)
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	asked, err := client.Ask(ctx, as("alice", formAskRequest("maintenance-wait")))
	require.NoError(t, err)
	type result struct {
		response *connect.Response[v1.WaitResponse]
		err      error
	}
	done := make(chan result, 1)
	go func() {
		response, err := client.Wait(ctx, as("alice", &v1.WaitRequest{AskId: asked.Msg.GetAskId(), Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}))
		done <- result{response, err}
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("wait was not admitted")
	}
	prepared, err := provider.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "restart", Reason: "test"}))
	require.NoError(t, err)
	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.Equal(t, "pending", got.response.Msg.GetState())
		require.Equal(t, hub.WaitYieldReason, got.response.Msg.GetReason())
	case <-ctx.Done():
		t.Fatal("wait did not yield to maintenance")
	}
	fence := prepared.Msg.GetStanding()
	drained, err := provider.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "restart", FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision()}))
	require.NoError(t, err)
	require.True(t, drained.Msg.GetStanding().GetDrained())
	_, err = provider.Resume(ctx, connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "restart", FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision()}))
	require.NoError(t, err)
	service.SetWaitSliceForTest(50 * time.Millisecond)
	again, err := client.Wait(ctx, as("alice", &v1.WaitRequest{AskId: asked.Msg.GetAskId(), Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}))
	require.NoError(t, err)
	require.Equal(t, "pending", again.Msg.GetState())
	require.Equal(t, hub.WaitSliceReason, again.Msg.GetReason())
}
