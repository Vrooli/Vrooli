package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestAgentManagerConnectGetRunEventsUsesSharedPaginationAndFilter(t *testing.T) {
	h, _, repos, eventStore := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "connect event page", ScopePath: ".", Status: domain.TaskStatusQueued}
	require.NoError(t, repos.Tasks.Create(ctx, task))
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	require.NoError(t, repos.Runs.Create(ctx, run))
	for _, event := range []*domain.RunEvent{
		domain.NewMessageEvent(run.ID, "assistant", "first"),
		domain.NewToolCallEvent(run.ID, "read_file", "call-1", map[string]interface{}{"path": "a.txt"}),
		domain.NewMessageEvent(run.ID, "assistant", "second"),
		domain.NewMessageEvent(run.ID, "assistant", "third"),
		domain.NewToolCallEvent(run.ID, "read_file", "call-2", map[string]interface{}{"path": "b.txt"}),
	} {
		require.NoError(t, eventStore.Append(ctx, run.ID, event))
	}

	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(h))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	limit := int32(2)
	filter := domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE
	first, err := client.GetRunEvents(ctx, connect.NewRequest(&apipb.GetRunEventsRequest{
		RunId: run.ID.String(), Limit: &limit, EventTypes: []domainpb.RunEventType{filter},
	}))
	require.NoError(t, err)
	require.Len(t, first.Msg.GetEvents(), 2)
	require.True(t, first.Msg.GetHasMore())
	require.Equal(t, "first", first.Msg.GetEvents()[0].GetMessage().GetContent())
	require.Equal(t, "second", first.Msg.GetEvents()[1].GetMessage().GetContent())

	after := first.Msg.GetEvents()[1].GetSequence()
	lastLimit := int32(2)
	last, err := client.GetRunEvents(ctx, connect.NewRequest(&apipb.GetRunEventsRequest{
		RunId: run.ID.String(), AfterSequence: &after, Limit: &lastLimit,
		EventTypes: []domainpb.RunEventType{filter},
	}))
	require.NoError(t, err)
	require.Len(t, last.Msg.GetEvents(), 1)
	require.False(t, last.Msg.GetHasMore())
	require.Equal(t, "third", last.Msg.GetEvents()[0].GetMessage().GetContent())
	require.Greater(t, last.Msg.GetEvents()[0].GetSequence(), after)
}

func TestAgentManagerConnectGetRunEventsRejectsInvalidLimits(t *testing.T) {
	h, _, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "connect event limit", ScopePath: ".", Status: domain.TaskStatusQueued}
	require.NoError(t, repos.Tasks.Create(ctx, task))
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	require.NoError(t, repos.Runs.Create(ctx, run))
	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(h))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	for _, invalid := range []int32{0, -1, 1001} {
		limit := invalid
		_, err := client.GetRunEvents(ctx, connect.NewRequest(&apipb.GetRunEventsRequest{RunId: run.ID.String(), Limit: &limit}))
		require.Error(t, err, "limit %d", invalid)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "limit %d", invalid)
	}
}
