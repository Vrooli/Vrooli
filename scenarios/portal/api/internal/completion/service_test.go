package completion_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	agentbrief "github.com/vrooli/agentbrief-go"
	apidb "github.com/vrooli/api-core/database"

	internalbrief "portal/internal/brief"
	internalchat "portal/internal/chat"
	"portal/internal/completion"
	localdb "portal/internal/database"
	"portal/internal/integrations/openrouter"

	db "github.com/vrooli/api-core/databasetest"

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
	tokens  []string
}

func (f *fakeOpenRouter) StreamCompletion(ctx context.Context, req openrouter.CompletionRequest, emit func(openrouter.StreamEvent) error) error {
	f.request = req
	tokens := f.tokens
	if len(tokens) == 0 {
		tokens = []string{"Portal ", "ready"}
	}
	for index, token := range tokens {
		ev := openrouter.StreamEvent{Token: token}
		if index == len(tokens)-1 {
			ev.Usage = openrouter.Usage{PromptTokens: 11, CompletionTokens: 2, TotalTokens: 13}
		}
		if err := emit(ev); err != nil {
			return err
		}
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

type fakeBriefs struct {
	records []internalbrief.Record
	inputs  []internalbrief.BuildInput
	uses    []internalbrief.UseInput
}

func (f *fakeBriefs) Build(_ context.Context, input internalbrief.BuildInput) (internalbrief.Record, error) {
	f.inputs = append(f.inputs, input)
	if len(f.records) == 0 {
		return internalbrief.Record{}, nil
	}
	return f.records[min(len(f.inputs)-1, len(f.records)-1)], nil
}

func (f *fakeBriefs) Get(_ context.Context, id string) (internalbrief.Record, error) {
	for _, record := range f.records {
		if record.ID == id {
			return record, nil
		}
	}
	return internalbrief.Record{}, sql.ErrNoRows
}

func (f *fakeBriefs) RecordUse(_ context.Context, input internalbrief.UseInput) (bool, error) {
	for _, use := range f.uses {
		if use == input {
			return false, nil
		}
	}
	f.uses = append(f.uses, input)
	return true, nil
}

func TestBuildOpenRouterRequestUsesCurrentUserPromptForBrief(t *testing.T) {
	_, chats, _ := newCompletionService(t, nil, nil)
	briefs := &fakeBriefs{records: []internalbrief.Record{{ID: "brief-second", Rendered: "<vrooli-context-brief>second</vrooli-context-brief>"}}}
	svc := completion.NewService(completion.Config{Chat: chats, Briefs: briefs})
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal", Model: "test/model"})
	require.NoError(t, err)
	first, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "First request with enough detail"})
	require.NoError(t, err)
	_, err = chats.AppendAssistantMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, ParentMessageID: first.ID, Content: "first answer", Model: "test/model"})
	require.NoError(t, err)
	second, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Second request with different detail"})
	require.NoError(t, err)

	req, parentID, err := svc.BuildOpenRouterRequest(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: second.ID})

	require.NoError(t, err)
	require.Equal(t, second.ID, parentID)
	require.Len(t, briefs.inputs, 1)
	require.Equal(t, "Second request with different detail", briefs.inputs[0].Prompt)
	require.Contains(t, req.Messages[0].Content, "second")
	require.NotContains(t, req.Messages[0].Content, "first</vrooli-context-brief>")
}

func TestStreamRecordsBriefReferenceOnlyForExactPathOrCommand(t *testing.T) {
	ctx := context.Background()
	briefs := &fakeBriefs{records: []internalbrief.Record{{
		ID:    "brief-reference",
		Items: []agentbrief.Item{{Title: "Portal status", Path: "docs/portal/status.md", SuggestedCommand: "vrooli scenario status portal"}},
	}}}
	streamer := &fakeOpenRouter{tokens: []string{"See docs/portal/status.md"}}
	svc, chats, _ := newCompletionService(t, streamer, nil)
	// The service is constructed separately so the test can exercise the optional
	// use-recording seam without changing the Build-only provider contract.
	svc = completion.NewService(completion.Config{Chat: chats, OpenRouter: streamer, Briefs: briefs, BriefStore: briefs})
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "References", Model: "test/model"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Find the portal status documentation"})
	require.NoError(t, err)
	_, err = svc.Stream(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(openrouter.StreamEvent) error { return nil })
	require.NoError(t, err)
	require.Equal(t, []internalbrief.UseInput{{BriefID: "brief-reference", ItemIndex: 0, Kind: "REFERENCED"}}, briefs.uses)

	briefs.uses = nil
	streamer.tokens = []string{"Portal status is healthy"}
	_, err = svc.Stream(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(openrouter.StreamEvent) error { return nil })
	require.NoError(t, err)
	require.Empty(t, briefs.uses)
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

type messageImages struct {
	calls []string
	data  []byte
	err   error
}

func (f *messageImages) ResolveMessageImages(_ context.Context, chatID, messageID string) ([][]byte, error) {
	f.calls = append(f.calls, chatID+"/"+messageID)
	if f.err != nil {
		return nil, f.err
	}
	return [][]byte{f.data}, nil
}

func TestCompletionResolvesOnlySelectedBranchImagesAndCopiesThem(t *testing.T) {
	_, chats, _ := newCompletionService(t, nil, nil)
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Image review", Model: "vision-fixture"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Review selected crop"})
	require.NoError(t, err)
	_, err = chats.EditMessage(ctx, internalchat.BranchMessageInput{MessageID: user.ID, Content: "Sibling request"})
	require.NoError(t, err)
	images := &messageImages{data: []byte{1, 2, 3}}
	svc := completion.NewService(completion.Config{Chat: chats, Images: images})
	req, from, err := svc.BuildOpenRouterRequest(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID})
	require.NoError(t, err)
	require.Equal(t, user.ID, from)
	require.Equal(t, []string{chat.ID + "/" + user.ID}, images.calls)
	require.Equal(t, "Review selected crop", req.Messages[1].Content)
	require.Equal(t, [][]byte{{1, 2, 3}}, req.Messages[1].Images)
	images.data[0] = 9
	require.Equal(t, byte(1), req.Messages[1].Images[0][0])
}

func TestUnavailableMessageImagePreventsProviderCallAndAssistantWrite(t *testing.T) {
	streamer := &fakeOpenRouter{}
	_, chats, _ := newCompletionService(t, streamer, nil)
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Private image", Model: "vision-fixture"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Review"})
	require.NoError(t, err)
	images := &messageImages{err: sql.ErrNoRows}
	svc := completion.NewService(completion.Config{Chat: chats, OpenRouter: streamer, Images: images})
	_, err = svc.Stream(ctx, completion.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(openrouter.StreamEvent) error { return nil })
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Empty(t, streamer.request.Messages)
	tree, _, err := chats.GetTree(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, tree, 1)
}
