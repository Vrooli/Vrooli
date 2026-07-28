package sources

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"
	internal "signal-inbox/internal/sources"
	"signal-inbox/internal/testutil/db"
	"signal-inbox/internal/testutil/mocks"
)

func newHandler(t *testing.T) (*connectHandler, signals.Service) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(internal.Schema)))
	clk := mocks.NewFakeClock(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	journal := signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
	service, err := internal.NewService(internal.NewSQLiteRepository(database), journal, clk, internal.ChromeBookmarksAdapter{}, internal.RedditSavedArchiveAdapter{})
	require.NoError(t, err)
	return NewConnectHandler(service), journal
}

func TestSourceTransportImportsMeasuredChromeArchive(t *testing.T) {
	handler, journal := newHandler(t)
	listed, err := handler.ListAdapters(context.Background(), connect.NewRequest(&sourcesv1.ListAdaptersRequest{}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Adapters, 2)
	require.True(t, listed.Msg.Adapters[0].Enabled)
	require.True(t, listed.Msg.Adapters[1].Enabled)

	export := []byte(`<!DOCTYPE NETSCAPE-Bookmark-file-1><DL><p><DT><A HREF="https://example.test/a">A</A></DL><p>`)
	imported, err := handler.ImportArchive(context.Background(), connect.NewRequest(&sourcesv1.ImportArchiveRequest{AdapterId: internal.ChromeBookmarksAdapterID, Content: export}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), imported.Msg.Result.Created)
	all, err := journal.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func TestSourceTransportRejectsEmptyAndUnknownImports(t *testing.T) {
	handler, _ := newHandler(t)
	_, err := handler.ImportArchive(context.Background(), connect.NewRequest(&sourcesv1.ImportArchiveRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = handler.ImportArchive(context.Background(), connect.NewRequest(&sourcesv1.ImportArchiveRequest{AdapterId: "unknown", Content: []byte("x")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
