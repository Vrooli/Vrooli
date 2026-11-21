package message

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"portal/internal/chat"
	"portal/internal/contextcapture"
	localdb "portal/internal/database"
)

type fakeContextRenderer struct {
	owner string
	ids   []string
}

func (f *fakeContextRenderer) Render(_ context.Context, owner, id string) (contextcapture.Document, []byte, string, error) {
	f.owner = owner
	f.ids = append(f.ids, id)
	return contextcapture.Document{ID: id, Owner: owner}, []byte(id), "digest", nil
}

func newMessageImageChat(t *testing.T) (*chat.Service, *sql.DB, context.Context, context.Context) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(chat.Schema)))
	service := chat.NewService(chat.NewSQLiteRepository(db, nil))
	alice, err := chat.WithOwner(context.Background(), "alice")
	require.NoError(t, err)
	bob, err := chat.WithOwner(context.Background(), "bob")
	require.NoError(t, err)
	return service, db, alice, bob
}

func TestContextImageResolverUsesOwnedOpaqueReferencesAndFreshRender(t *testing.T) {
	service, db, alice, bob := newMessageImageChat(t)
	chatRecord, err := service.CreateChat(alice, chat.CreateChatInput{Title: "images"})
	require.NoError(t, err)
	ids := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"}
	message, err := service.SendUserMessage(alice, chat.SendMessageInput{ChatID: chatRecord.ID, Content: "look", ContextDocumentIDs: ids})
	require.NoError(t, err)
	renderer := &fakeContextRenderer{}
	resolver := newContextImageResolver(service, renderer)
	images, err := resolver.ResolveMessageImages(alice, chatRecord.ID, message.ID)
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte(ids[0]), []byte(ids[1])}, images)
	require.Equal(t, "alice", renderer.owner)
	require.Equal(t, ids, renderer.ids)
	_, err = resolver.ResolveMessageImages(bob, chatRecord.ID, message.ID)
	require.Error(t, err)
	require.Equal(t, ids, renderer.ids)
	_, err = resolver.ResolveMessageImages(context.Background(), chatRecord.ID, message.ID)
	require.Error(t, err)
	require.NoError(t, db.Close())
}
