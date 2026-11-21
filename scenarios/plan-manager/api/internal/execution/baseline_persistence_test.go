package execution

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiDatabase "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
)

func TestBaselineSetPersistenceRejectsUnknownStatus(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	require.NoError(t, apiDatabase.EnsureSchemas(ctx, database, apiDatabase.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(database, scheduletest.New(time.Now().UTC()))

	valid := Execution{
		ID: "baseline-valid", PlanID: "plan-1", LifecycleState: ExecutionLifecycleActive,
		BaselineSet: BaselineSetState{Version: BaselineSetStateSchemaVersion, Name: "before", Status: BaselineSetStatusRequired},
	}
	require.NoError(t, repo.SaveExecution(ctx, valid))

	invalid := valid
	invalid.ID = "baseline-invalid-write"
	invalid.BaselineSet.Status = "future"
	require.Error(t, repo.SaveExecution(ctx, invalid), "unknown durable state must not be written")

	_, err := database.ExecContext(ctx, upsertBaselineSetSQL, valid.ID, `{"version":2,"name":"before","status":"future"}`, "now")
	require.NoError(t, err)
	_, found, err := repo.GetExecution(ctx, valid.ID)
	require.Error(t, err, "unknown durable state must fail closed on read")
	require.False(t, found)
}
