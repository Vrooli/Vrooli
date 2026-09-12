package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

// [REQ:SWBD-P0-007] [REQ:SWBD-P0-008]
func TestStoreCreatesAndListsBinding(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(func() string { return Schema() })))
	store := NewStore(database)
	record, err := store.Create(context.Background(), Binding{AgentID: "agent-1", ChannelID: "telegram", Address: "chat-1", ThreadKey: "thread-1"})
	require.NoError(t, err)
	require.NotEmpty(t, record.ID)
	rows, err := store.List(context.Background(), "agent-1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, record.ID, rows[0].ID)
}
