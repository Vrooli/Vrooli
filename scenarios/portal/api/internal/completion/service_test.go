package completion_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
	internalchat "portal/internal/chat"
	"portal/internal/completion"
	localdb "portal/internal/database"
	"portal/internal/integrations/openrouter"

	"github.com/vrooli/api-core/scheduletest"
)

type fakeSkills struct {
	skills []completion.Skill
}

func (f fakeSkills) ResolveSkills(context.Context, []string) ([]completion.Skill, error) {
	return f.skills, nil
}

type fakeOpenRouter struct {
	request openrouter.CompletionRequest
}

type fakeSearchContext struct {
	block string
}

func (f fakeSearchContext) RecentContextBlock(context.Context, string) string {
	return f.block
}

func (f *fakeOpenRouter) StreamCompletion(ctx context.Context, req openrouter.CompletionRequest, emit func(openrouter.StreamEvent) error) error {
	f.request = req
	if err := emit(openrouter.StreamEvent{Token: "Portal "}); err != nil {
		return err
	}
	if err := emit(openrouter.StreamEvent{Token: "ready", Usage: openrouter.Usage{PromptTokens: 11, CompletionTokens: 2, TotalTokens: 13}}); err != nil {
		return err
	}
	return ctx.Err()
}

func newCompletionService(t *testing.T, streamer completion.OpenRouterStreamer, resolver completion.SkillResolver) (*completion.Service, *internalchat.Service, *sql.DB) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalchat.Schema),
	))
	repo := internalchat.NewSQLiteRepository(d, scheduletest.New(time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)))
	chatService := internalchat.NewService(repo)
	return completion.NewService(completion.Config{
		Chat:          chatService,
		OpenRouter:    streamer,
		SkillResolver: resolver,
	}), chatService, d
}

func TestBuildOpenRouterRequestInjectsSelectedSkillsAsSystemPrompt(t *testing.T) {
	svc, chats, _ := newCompletionService(t, nil, fakeSkills{skills: []completion.Skill{
		{ID: "scientific-debugging", Content: "Use hypothesis-driven debugging."},
	}})
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal", Model: "test/model", WebSearchEnabled: true})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Find the bug"})
	require.NoError(t, err)

	req, parentID, err := svc.BuildOpenRouterRequest(ctx, completion.StreamInput{
		ChatID:           chat.ID,
		FromMessageID:    user.ID,
		SelectedSkillIDs: []string{"scientific-debugging"},
	})

	require.NoError(t, err)
	require.Equal(t, user.ID, parentID)
	require.Equal(t, "test/model", req.Model)
	require.Len(t, req.Messages, 2)
	require.Equal(t, "system", req.Messages[0].Role)
	require.Contains(t, req.Messages[0].Content, "Use hypothesis-driven debugging.")
	require.Contains(t, req.Messages[0].Content, "not a tool call")
	require.Equal(t, "user", req.Messages[1].Role)
	require.Len(t, req.Plugins, 1)
	require.Equal(t, "web", req.Plugins[0].ID)
}

func TestBuildOpenRouterRequestInjectsRecentSearchContext(t *testing.T) {
	_, chats, _ := newCompletionService(t, nil, nil)
	svc := completion.NewService(completion.Config{
		Chat:          chats,
		SearchContext: fakeSearchContext{block: "Recent Vrooli ecosystem search context.\n- [search-hub/doc] Portal README"},
	})
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal", Model: "test/model"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Use the latest context"})
	require.NoError(t, err)

	req, _, err := svc.BuildOpenRouterRequest(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID})

	require.NoError(t, err)
	require.Contains(t, req.Messages[0].Content, "Recent Vrooli ecosystem search context")
	require.Contains(t, req.Messages[0].Content, "Portal README")
}

func TestStreamPersistsAssistantMessageAndUsage(t *testing.T) {
	streamer := &fakeOpenRouter{}
	svc, chats, _ := newCompletionService(t, streamer, nil)
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal", Model: "test/model"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Status?"})
	require.NoError(t, err)

	var tokens []string
	result, err := svc.Stream(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(ev openrouter.StreamEvent) error {
		if ev.Token != "" {
			tokens = append(tokens, ev.Token)
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"Portal ", "ready"}, tokens)
	require.Equal(t, "Portal ready", result.AssistantMessage.Content)
	require.Equal(t, user.ID, result.AssistantMessage.ParentMessageID)
	require.EqualValues(t, 11, result.Usage.PromptTokens)
	require.EqualValues(t, 2, result.Usage.CompletionTokens)
	require.Equal(t, "test/model", streamer.request.Model)

	messages, leaf, err := chats.GetTree(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, result.AssistantMessage.ID, leaf)
}
