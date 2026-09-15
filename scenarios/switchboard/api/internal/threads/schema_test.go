package threads

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P0-006] [REQ:SWBD-P1-005] [REQ:SWBD-P1-006]
func TestSchemaAndDuplicateAppend(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(mustSchema)))
	s := NewStore(d)
	e := channels.Envelope{ChannelID: "a", ThreadKey: "t", RemoteMessageID: "1", AuthorKind: channels.AuthorHuman, ReceivedAt: time.Now()}
	th, err := s.Upsert(context.Background(), e, false)
	require.NoError(t, err)
	ok, err := s.Append(context.Background(), th, e)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = s.Append(context.Background(), th, e)
	require.NoError(t, err)
	require.False(t, ok)
}
func mustSchema() string { return schemaSQL }
