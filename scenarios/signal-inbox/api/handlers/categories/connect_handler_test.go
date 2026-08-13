package categories

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	internal "signal-inbox/internal/categories"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
	categoriesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/categories"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(internal.Schema),
	))
	clock := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	service := internal.NewService(internal.NewSQLiteRepository(database), clock, &mocks.FakeInference{})
	_, err := service.Bootstrap(context.Background())
	require.NoError(t, err)
	return NewConnectHandler(service)
}

func TestCreateListAndRenameCategoryTransport(t *testing.T) {
	handler := newHandler(t)
	created, err := handler.CreateCategory(context.Background(), connect.NewRequest(&categoriesv1.CreateCategoryRequest{Name: "Research", Description: "Items to investigate"}))
	require.NoError(t, err)
	require.Equal(t, "Research", created.Msg.Category.Name)

	renamed, err := handler.RenameCategory(context.Background(), connect.NewRequest(&categoriesv1.RenameCategoryRequest{Id: created.Msg.Category.Id, Name: "Investigations", Description: "Reviewed by the operator"}))
	require.NoError(t, err)
	require.Equal(t, "Investigations", renamed.Msg.Category.Name)

	listed, err := handler.ListCategories(context.Background(), connect.NewRequest(&categoriesv1.ListCategoriesRequest{}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Categories, 2, "the runtime category plus the reserved fallback are exposed")
}

func TestCategoryTransportReturnsActionableErrors(t *testing.T) {
	handler := newHandler(t)
	_, err := handler.CreateCategory(context.Background(), connect.NewRequest(&categoriesv1.CreateCategoryRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = handler.RenameCategory(context.Background(), connect.NewRequest(&categoriesv1.RenameCategoryRequest{Id: "missing", Name: "Anything"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
