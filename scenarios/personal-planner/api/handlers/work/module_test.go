package work_test

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"personal-planner/handlers/work"
	localdb "personal-planner/internal/database"
	internalwork "personal-planner/internal/work"
)

func TestModuleShape(t *testing.T) {
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(internalwork.Schema)))
	m := work.Module(database.NewFromPrimary(db), schedule.System(), log.New(io.Discard, "", 0))
	require.Equal(t, "work", m.Name)
	require.Len(t, m.Endpoints, 4)
}
