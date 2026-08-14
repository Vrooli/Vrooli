package library

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/programs"
)

func TestPromoteAndSetCurrentAreExplicitAndVersioned(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(Schema)))
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO programs (id,session_id,source,provenance,status,created_at) VALUES (?,?,?,?,?,?)`, "prog_ok", "sess", "print('ok')", "1", "succeeded", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	repo := NewRepository(db)
	first, err := repo.PromoteByID(context.Background(), "prog_ok", "probe", "A reusable probe", "operator", "validated", now)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.GetVersion())
	_, err = repo.SetCurrent(context.Background(), "probe", 1)
	require.NoError(t, err)
	current, err := repo.Get(context.Background(), "probe", 0)
	require.NoError(t, err)
	require.True(t, current.GetCurrent())
	_, err = repo.Promote(context.Background(), &programsv1.Program{Id: "failed", Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED}, "bad", "", "", "", now)
	require.ErrorIs(t, err, ErrSourceFailed)
}
