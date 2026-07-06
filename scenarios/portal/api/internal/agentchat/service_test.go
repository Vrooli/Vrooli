package agentchat_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"portal/internal/agentchat"
	internalchat "portal/internal/chat"
	localdb "portal/internal/database"
	"portal/internal/integrations/agentmanager"
	"portal/internal/testutil/db"
	"portal/internal/testutil/mocks"
)

type fakeAgentManager struct {
	startInput agentmanager.StartInput
	events     []agentmanager.ActivityEvent
}

func (f *fakeAgentManager) Start(_ context.Context, input agentmanager.StartInput) (agentmanager.Session, error) {
	f.startInput = input
	return agentmanager.Session{TaskID: "task-1", RunID: "run-1"}, nil
}

func (f *fakeAgentManager) StreamRunEvents(_ context.Context, runID string, emit func(agentmanager.ActivityEvent) error) error {
	for _, ev := range f.events {
		ev.RunID = runID
		if err := emit(ev); err != nil {
			return err
		}
	}
	return nil
}

func newChatService(t *testing.T) (*internalchat.Service, *sql.DB) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalchat.Schema),
	))
	repo := internalchat.NewSQLiteRepository(d, mocks.NewFakeClock(time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)))
	return internalchat.NewService(repo), d
}

func TestStreamStartsAgentRunAndPersistsAgentMessage(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{
		Title:        "Agent chat",
		Mode:         internalchat.ChatModeAgent,
		AgentHarness: internalchat.AgentHarnessCodex,
	})
	require.NoError(t, err)
	user, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{
		ChatID:  chat.ID,
		Content: "Update the portal tests",
	})
	require.NoError(t, err)
	fake := &fakeAgentManager{events: []agentmanager.ActivityEvent{
		{Kind: agentmanager.EventKindProgress, Text: "Agent progress 50%"},
		{Kind: agentmanager.EventKindDone, Text: "Agent run complete", Done: true},
	}}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake})
	var streamed []agentmanager.ActivityEvent

	result, err := svc.Stream(ctx, agentchat.StreamInput{
		ChatID:        chat.ID,
		FromMessageID: user.ID,
	}, func(ev agentmanager.ActivityEvent) error {
		streamed = append(streamed, ev)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "Update the portal tests", fake.startInput.Prompt)
	require.Equal(t, internalchat.AgentHarnessCodex, fake.startInput.Harness)
	require.Equal(t, internalchat.RoleAgent, result.Message.Role)
	require.Equal(t, user.ID, result.Message.ParentMessageID)
	require.Contains(t, result.Message.Content, "Agent progress 50%")
	require.Len(t, streamed, 3)

	messages, leaf, err := chatSvc.GetTree(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, result.Message.ID, leaf)
	require.Len(t, messages, 2)
	require.Equal(t, internalchat.RoleAgent, messages[1].Role)
}
