package chat_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	internalchat "portal/internal/chat"
	localdb "portal/internal/database"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newRepo(t *testing.T) (internalchat.Repository, *sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalchat.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC))
	return internalchat.NewSQLiteRepository(d, clk), d, clk
}

func TestSchemaOmitsLegacyTables(t *testing.T) {
	_, d, _ := newRepo(t)
	for _, table := range []string{"tool_calls", "labels", "chat_labels", "attachments", "async_operations"} {
		var count int
		err := d.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
		require.NoError(t, err)
		require.Zero(t, count, "legacy table %s must not exist", table)
	}
}

func TestGroupsCRUD(t *testing.T) {
	repo, _, clk := newRepo(t)
	ctx := context.Background()

	group, err := repo.CreateGroup(ctx, internalchat.CreateGroupInput{Name: "Build", Color: "#0f766e"})
	require.NoError(t, err)
	require.Equal(t, "Build", group.Name)
	require.False(t, group.Collapsed)

	clk.Advance(time.Minute)
	name := "Research"
	collapsed := true
	order := int32(4)
	updated, err := repo.UpdateGroup(ctx, internalchat.UpdateGroupInput{
		ID: group.ID, Name: &name, Collapsed: &collapsed, SortOrder: &order,
	})
	require.NoError(t, err)
	require.Equal(t, "Research", updated.Name)
	require.True(t, updated.Collapsed)
	require.Equal(t, int32(4), updated.SortOrder)
	require.True(t, updated.UpdatedAt.After(group.UpdatedAt))

	groups, err := repo.ListGroups(ctx)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, updated.ID, groups[0].ID)

	deleted, err := repo.DeleteGroup(ctx, group.ID)
	require.NoError(t, err)
	require.True(t, deleted)
}

func TestMessageBranchingTracksSiblingsAndActiveLeaf(t *testing.T) {
	repo, _, _ := newRepo(t)
	ctx := context.Background()

	chat, err := repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal design"})
	require.NoError(t, err)

	user, err := repo.SendMessage(ctx, internalchat.SendMessageInput{
		ChatID: chat.ID, Role: internalchat.RoleUser, Content: "Sketch the portal shell",
	})
	require.NoError(t, err)
	require.Equal(t, int32(0), user.SiblingIndex)

	assistant, err := repo.SendMessage(ctx, internalchat.SendMessageInput{
		ChatID: chat.ID, ParentMessageID: user.ID, Role: internalchat.RoleAssistant, Content: "First answer",
	})
	require.NoError(t, err)
	require.Equal(t, int32(0), assistant.SiblingIndex)

	branch, err := repo.BranchMessage(ctx, internalchat.BranchMessageInput{
		MessageID: assistant.ID,
		Content:   "Second answer",
	})
	require.NoError(t, err)
	require.Equal(t, user.ID, branch.ParentMessageID)
	require.Equal(t, internalchat.RoleAssistant, branch.Role)
	require.Equal(t, int32(1), branch.SiblingIndex)

	messages, leaf, err := repo.ListMessages(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, branch.ID, leaf)
}

func TestChatFTSFiltersList(t *testing.T) {
	repo, _, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal readiness ladder"})
	require.NoError(t, err)
	_, err = repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Unrelated"})
	require.NoError(t, err)

	chats, _, err := repo.ListChats(ctx, internalchat.SearchInput{Query: "readiness"})
	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, "Portal readiness ladder", chats[0].Title)
}

func TestSearchAttachmentsHydrateWithMessageTree(t *testing.T) {
	repo, _, _ := newRepo(t)
	ctx := context.Background()

	chat, err := repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Portal search"})
	require.NoError(t, err)
	user, err := repo.SendMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Role: internalchat.RoleUser, Content: "Find scenario docs"})
	require.NoError(t, err)

	attachment, err := repo.CreateSearchAttachment(ctx, internalchat.CreateSearchAttachmentInput{
		ChatID:    chat.ID,
		MessageID: user.ID,
		Query:     "Find scenario docs",
		Hits: []internalchat.SearchHit{{
			ProviderID:  "knowledge-observatory",
			Type:        "doc",
			Title:       "Portal README",
			Snippet:     "Chat-first front door",
			Path:        "scenarios/portal/README.md",
			Score:       0.72,
			RerankScore: 0.91,
			Locations:   []string{"scenarios/portal/README.md:1"},
		}},
		LatencyMS: 37,
	})
	require.NoError(t, err)

	messages, _, err := repo.ListMessages(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Len(t, messages[0].SearchAttachments, 1)
	require.Equal(t, attachment.ID, messages[0].SearchAttachments[0].ID)
	require.Equal(t, "Portal README", messages[0].SearchAttachments[0].Hits[0].Title)

	recent, err := repo.ListSearchAttachments(ctx, chat.ID, 1)
	require.NoError(t, err)
	require.Len(t, recent, 1)
	require.Equal(t, attachment.ID, recent[0].ID)
}
