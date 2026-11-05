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

func TestMessageContextDocumentsPersistInOrderAndStayScoped(t *testing.T) {
	repo, d, _ := newRepo(t)
	alice, err := internalchat.WithOwner(context.Background(), "alice")
	require.NoError(t, err)
	chat, err := repo.CreateChat(alice, internalchat.CreateChatInput{Title: "Context"})
	require.NoError(t, err)
	ids := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"}
	user, err := repo.SendMessage(alice, internalchat.SendMessageInput{ChatID: chat.ID, Role: internalchat.RoleUser, Content: "inspect", ContextDocumentIDs: ids})
	require.NoError(t, err)
	require.Equal(t, ids, user.ContextDocumentIDs)
	messages, _, err := repo.ListMessages(alice, chat.ID)
	require.NoError(t, err)
	require.Equal(t, ids, messages[0].ContextDocumentIDs)
	got, err := repo.ListMessageContextDocumentIDs(alice, chat.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, ids, got)
	_, err = repo.SendMessage(alice, internalchat.SendMessageInput{ChatID: chat.ID, Role: internalchat.RoleUser, Content: "duplicate", ContextDocumentIDs: []string{ids[0], ids[0]}})
	require.ErrorIs(t, err, internalchat.ErrInvalidInput)
	_, err = repo.SendMessage(alice, internalchat.SendMessageInput{ChatID: chat.ID, Role: internalchat.RoleUser, Content: "bad", ContextDocumentIDs: []string{"not-a-uuid"}})
	require.ErrorIs(t, err, internalchat.ErrInvalidInput)
	bob, err := internalchat.WithOwner(context.Background(), "bob")
	require.NoError(t, err)
	_, err = repo.ListMessageContextDocumentIDs(bob, chat.ID, user.ID)
	require.Error(t, err)
	var count int
	require.NoError(t, d.QueryRow("SELECT COUNT(*) FROM message_context_documents WHERE message_id = ?", user.ID).Scan(&count))
	require.Equal(t, 2, count)
}

func TestAccountOwnershipIsolatesChatsGroupsAndAllMessageOperations(t *testing.T) {
	repo, _, _ := newRepo(t)
	legacy := context.Background()
	alice, err := internalchat.WithOwner(legacy, "alice")
	require.NoError(t, err)
	bob, err := internalchat.WithOwner(legacy, "bob")
	require.NoError(t, err)
	old, err := repo.CreateChat(legacy, internalchat.CreateChatInput{Title: "Legacy"})
	require.NoError(t, err)
	group, err := repo.CreateGroup(alice, internalchat.CreateGroupInput{Name: "Private group"})
	require.NoError(t, err)
	private, err := repo.CreateChat(alice, internalchat.CreateChatInput{Title: "Private conversation", GroupID: group.ID})
	require.NoError(t, err)
	user, err := repo.SendMessage(alice, internalchat.SendMessageInput{ChatID: private.ID, Content: "Private request", Role: internalchat.RoleUser})
	require.NoError(t, err)
	assistant, err := repo.AppendMessage(alice, internalchat.SendMessageInput{ChatID: private.ID, ParentMessageID: user.ID, Content: "Private answer", Role: internalchat.RoleAssistant})
	require.NoError(t, err)
	for _, ctx := range []context.Context{legacy, bob} {
		_, err = repo.GetChat(ctx, private.ID)
		require.Error(t, err)
		_, _, err = repo.ListMessages(ctx, private.ID)
		require.Error(t, err)
		_, err = repo.SendMessage(ctx, internalchat.SendMessageInput{ChatID: private.ID, Content: "Injection"})
		require.Error(t, err)
		_, err = repo.BranchMessage(ctx, internalchat.BranchMessageInput{MessageID: assistant.ID, Content: "Stolen branch"})
		require.Error(t, err)
		_, err = repo.CreateUsageRecord(ctx, internalchat.CreateUsageInput{ChatID: private.ID, MessageID: assistant.ID, Model: "model"})
		require.Error(t, err)
		_, err = repo.CreateSearchAttachment(ctx, internalchat.CreateSearchAttachmentInput{ChatID: private.ID, MessageID: user.ID, Query: "private"})
		require.Error(t, err)
		_, err = repo.ListSearchAttachments(ctx, private.ID, 5)
		require.Error(t, err)
		deleted, err := repo.DeleteChat(ctx, private.ID)
		require.NoError(t, err)
		require.False(t, deleted)
		deleted, err = repo.DeleteGroup(ctx, group.ID)
		require.NoError(t, err)
		require.False(t, deleted)
		_, err = repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Foreign group", GroupID: group.ID})
		require.Error(t, err)
		groups, err := repo.ListGroups(ctx)
		require.NoError(t, err)
		require.Empty(t, groups)
	}
	chats, groups, err := repo.ListChats(alice, internalchat.SearchInput{})
	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, private.ID, chats[0].ID)
	require.Len(t, groups, 1)
	chats, _, err = repo.ListChats(legacy, internalchat.SearchInput{})
	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, old.ID, chats[0].ID)
	_, err = repo.GetChat(alice, old.ID)
	require.Error(t, err)
	foreignGroup, err := repo.CreateGroup(bob, internalchat.CreateGroupInput{Name: "Bob"})
	require.NoError(t, err)
	_, err = repo.UpdateChat(alice, internalchat.UpdateChatInput{ID: private.ID, GroupID: &foreignGroup.ID})
	require.Error(t, err)
	messages, _, err := repo.ListMessages(alice, private.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
}

func TestOwnerAssignmentFailureCannotPublishLegacyRecords(t *testing.T) {
	repo, d, _ := newRepo(t)
	ctx, err := internalchat.WithOwner(context.Background(), "alice")
	require.NoError(t, err)
	_, err = d.Exec(`CREATE TRIGGER fail_chat_owner BEFORE INSERT ON chat_owners BEGIN SELECT RAISE(ABORT,'injected owner failure'); END`)
	require.NoError(t, err)
	_, err = repo.CreateChat(ctx, internalchat.CreateChatInput{Title: "Must not become legacy"})
	require.Error(t, err)
	var count int
	require.NoError(t, d.QueryRow("SELECT COUNT(*) FROM chats").Scan(&count))
	require.Zero(t, count)
	_, err = d.Exec(`CREATE TRIGGER fail_group_owner BEFORE INSERT ON chat_group_owners BEGIN SELECT RAISE(ABORT,'injected owner failure'); END`)
	require.NoError(t, err)
	_, err = repo.CreateGroup(ctx, internalchat.CreateGroupInput{Name: "Must not become legacy"})
	require.Error(t, err)
	require.NoError(t, d.QueryRow("SELECT COUNT(*) FROM chat_groups").Scan(&count))
	require.Zero(t, count)
}
