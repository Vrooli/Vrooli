package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/modules"
	"audio-tools/internal/testutil/db"
)

func newTestDB(t *testing.T) *apidb.RoutedDB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, modules.AllSchemas()...))
	return apidb.NewFromPrimary(d)
}
