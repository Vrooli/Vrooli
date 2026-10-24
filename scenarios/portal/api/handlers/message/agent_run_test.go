package message

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"
	"portal/internal/agentchat"
	"portal/internal/chat"
	localdb "portal/internal/database"
	am "portal/internal/integrations/agentmanager"
)

type controlledAgent struct{ stops int }

func (f *controlledAgent) Start(context.Context, am.StartInput) (am.Session, error) {
	panic("run lookup must not start work")
}

func (f *controlledAgent) FindAdmission(context.Context, string) (am.Session, error) {
	panic("bound lookup must not search")
}

func (f *controlledAgent) StreamRunEvents(context.Context, string, func(am.ActivityEvent) error) error {
	panic("lookup must not stream")
}

func (f *controlledAgent) Run(_ context.Context, id string) (am.RunState, error) {
	status := "running"
	if f.stops > 0 {
		status = "cancelled"
	}
	return am.RunState{RunID: id, Status: status, Terminal: f.stops > 0}, nil
}

func (f *controlledAgent) Stop(context.Context, string) (am.RunState, error) {
	f.stops++
	return am.RunState{}, am.ErrStopUnconfirmed
}

func TestAgentRunRPCScopesStopAndReconcilesUncertainReply(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(chat.Schema), apidb.SchemaProviderFunc(agentchat.Schema)))
	chats := chat.NewService(chat.NewSQLiteRepository(db, nil))
	c, err := chats.CreateChat(ctx, chat.CreateChatInput{Title: "RPC fixture"})
	require.NoError(t, err)
	m, err := chats.SendUserMessage(ctx, chat.SendMessageInput{ChatID: c.ID, Content: "fixture"})
	require.NoError(t, err)
	repo := agentchat.NewSQLiteRepository(db)
	b, err := repo.Reserve(ctx, c.ID, m.ID)
	require.NoError(t, err)
	require.NoError(t, repo.Bind(ctx, b.ID, am.Session{TaskID: "task", RunID: "owned-run"}))
	agent := &controlledAgent{}
	handler := NewHandler(chats, nil, agentchat.NewService(agentchat.Config{Chat: chats, Runs: repo, AgentManager: agent}), nil)
	_, httpHandler := rpc.NewMessageServiceHandler(handler)
	server := httptest.NewServer(httpHandler)
	defer server.Close()
	client := rpc.NewMessageServiceClient(server.Client(), server.URL)

	// Recovery lists unknown launch outcomes without invoking any owner operation.
	unknown, err := chats.SendUserMessage(ctx, chat.SendMessageInput{ChatID: c.ID, Content: "unknown launch"})
	require.NoError(t, err)
	_, err = repo.Reserve(ctx, c.ID, unknown.ID)
	require.NoError(t, err)
	page, err := client.ListAgentAdmissions(ctx, connect.NewRequest(&pb.ListAgentAdmissionsRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Admissions, 1)
	require.Equal(t, m.ID, page.Msg.Admissions[0].MessageId)
	require.NotEmpty(t, page.Msg.NextPageToken)
	next, err := client.ListAgentAdmissions(ctx, connect.NewRequest(&pb.ListAgentAdmissionsRequest{PageSize: 1, PageToken: page.Msg.NextPageToken}))
	require.NoError(t, err)
	require.Len(t, next.Msg.Admissions, 1)
	require.Equal(t, unknown.ID, next.Msg.Admissions[0].MessageId)
	require.Empty(t, next.Msg.NextPageToken)
	_, err = client.ListAgentAdmissions(ctx, connect.NewRequest(&pb.ListAgentAdmissionsRequest{PageSize: 101}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Zero(t, agent.stops)
	request := func(chatID, messageID string) *connect.Request[pb.AgentRunRequest] {
		return connect.NewRequest(&pb.AgentRunRequest{ChatId: chatID, MessageId: messageID})
	}
	_, err = client.StopAgentRun(ctx, request(c.ID, "other-message"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Zero(t, agent.stops)
	_, err = client.StopAgentRun(ctx, request("", m.ID))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Zero(t, agent.stops)
	_, err = client.StopAgentRun(ctx, request(c.ID, m.ID))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.Equal(t, 1, agent.stops)
	state, err := client.GetAgentRun(ctx, request(c.ID, m.ID))
	require.NoError(t, err)
	require.True(t, state.Msg.Terminal)
	require.Equal(t, "owned-run", state.Msg.RunId)
	require.Equal(t, 1, agent.stops)
}
