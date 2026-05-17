package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
	"audio-tools/internal/testutil/db"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return d
}
