package search_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	internalchat "portal/internal/chat"
	localdb "portal/internal/database"
	internalsearch "portal/internal/search"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

type fakeHub struct {
	result internalsearch.QueryResult
	err    error
	input  internalsearch.QueryInput
}

func (f *fakeHub) Query(_ context.Context, input internalsearch.QueryInput) (internalsearch.QueryResult, error) {
	f.input = input
	return f.result, f.err
}

func newSearchFixture(t *testing.T, hub internalsearch.HubClient) (*internalsearch.Service, *internalchat.Service, *sql.DB) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalchat.Schema),
	))
	repo := internalchat.NewSQLiteRepository(d, scheduletest.New(time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)))
	chatService := internalchat.NewService(repo)
	return internalsearch.NewService(internalsearch.Config{Chat: chatService, Hub: hub}), chatService, d
}

func TestSuggestBoundsLimitAndReturnsHits(t *testing.T) {
	hub := &fakeHub{result: internalsearch.QueryResult{Hits: []internalchat.SearchHit{{
		ProviderID: "prompt-manager",
		Type:       "skill",
		Title:      "Scientific Debugging",
		Score:      0.8,
	}}}}
	svc, _, _ := newSearchFixture(t, hub)

	result, err := svc.Suggest(context.Background(), internalsearch.QueryInput{Query: "debug", Limit: 1})

	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
	require.Equal(t, int32(1), hub.input.Limit)
	require.Equal(t, "debug", hub.input.Query)
}

func TestSuggestDegradesOnHubError(t *testing.T) {
	hub := &fakeHub{err: errors.New("search-hub down")}
	svc, _, _ := newSearchFixture(t, hub)

	result, err := svc.Suggest(context.Background(), internalsearch.QueryInput{Query: "portal"})

	require.NoError(t, err)
	require.True(t, result.Degraded)
	require.Contains(t, result.Reason, "search-hub down")
}

func TestAttachForMessagePersistsSearchResults(t *testing.T) {
	hub := &fakeHub{result: internalsearch.QueryResult{Hits: []internalchat.SearchHit{{
		ProviderID: "knowledge-observatory",
		Type:       "doc",
		Title:      "Portal PRD",
		Path:       "scenarios/portal/PRD.md",
	}}}}
	svc, chats, _ := newSearchFixture(t, hub)
	ctx := context.Background()
	chat, err := chats.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal"})
	require.NoError(t, err)
	user, err := chats.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "portal requirements"})
	require.NoError(t, err)

	attachment, err := svc.AttachForMessage(ctx, chat.ID, user.ID)

	require.NoError(t, err)
	require.Equal(t, "portal requirements", attachment.Query)
	require.Len(t, attachment.Hits, 1)
	require.Equal(t, "Portal PRD", attachment.Hits[0].Title)

	messages, _, err := chats.GetTree(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, messages[0].SearchAttachments, 1)
}
