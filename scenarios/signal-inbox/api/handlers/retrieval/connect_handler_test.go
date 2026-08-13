package retrieval

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	"signal-inbox/internal/categories"
	localdb "signal-inbox/internal/database"
	internal "signal-inbox/internal/retrieval"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/triage"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/retrieval"
)

func newHandler(t *testing.T) (*connectHandler, signals.Service) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(categories.Schema), apidb.SchemaProviderFunc(triage.Schema), apidb.SchemaProviderFunc(internal.Schema)))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	return NewConnectHandler(internal.NewService(internal.NewSQLiteRepository(database), clk)), signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
}

func TestRetrievalTransportSearchesAndReadsAmbient(t *testing.T) {
	handler, journal := newHandler(t)
	captured, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "durable capture substrate"})
	require.NoError(t, err)
	search, err := handler.Search(context.Background(), connect.NewRequest(&retrievalv1.SearchRequest{Filter: &retrievalv1.SearchFilter{Text: "durable"}}))
	require.NoError(t, err)
	require.Len(t, search.Msg.Results, 1)
	require.Equal(t, captured.Signal.ID, search.Msg.Results[0].Signal.Id)
	ambient, err := handler.Ambient(context.Background(), connect.NewRequest(&retrievalv1.AmbientRequest{Budget: 1}))
	require.NoError(t, err)
	require.Len(t, ambient.Msg.Results, 1)
}

func TestRetrievalTransportRejectsInvalidSourceKind(t *testing.T) {
	handler, _ := newHandler(t)
	_, err := handler.Search(context.Background(), connect.NewRequest(&retrievalv1.SearchRequest{Filter: &retrievalv1.SearchFilter{SourceKind: "rss"}}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
