package library

import (
	"context"
	"testing"
	"time"

	internalbindings "program-runtime/internal/bindings"
	"program-runtime/internal/programs"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func TestPromoteAndSetCurrentAreExplicitAndVersioned(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(internalbindings.Schema), apidb.SchemaProviderFunc(Schema)))
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO programs (id,session_id,source,provenance,status,created_at) VALUES (?,?,?,?,?,?)`, "prog_ok", "sess", "print('ok')", "1", "succeeded", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	repo := NewRepository(db)
	first, err := repo.PromoteByID(context.Background(), "prog_ok", "probe", "A reusable probe", "operator", "validated", "probe coverage", []string{"session_id"}, []string{"bounded projection"}, now)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.GetVersion())
	_, err = repo.SetCurrent(context.Background(), "probe", 1)
	require.NoError(t, err)
	current, err := repo.Get(context.Background(), "probe", 0)
	require.NoError(t, err)
	require.True(t, current.GetCurrent())
	_, err = repo.Promote(context.Background(), &programsv1.Program{Id: "failed", Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED}, "bad", "", "", "", "", nil, nil, now)
	require.ErrorIs(t, err, ErrSourceFailed)
}

func TestPromoteDerivesOnlySuccessfulInvocations(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(internalbindings.Schema), apidb.SchemaProviderFunc(Schema)))
	now := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO programs (id,session_id,source,provenance,status,created_at) VALUES (?,?,?,?,?,?)`, "prog_invocations", "sess", "print('ok')", "1", "succeeded", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO binding_invocations (invocation_id,binding_id,target_scenario,session_id,program_id,provenance,outcome,reason,latency_ms,occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?), (?,?,?,?,?,?,?,?,?,?)`, "inv_success", "demo/ops/read", "demo", "sess", "prog_invocations", "agent", "success", "", 1, now.Format(time.RFC3339Nano), "inv_refused", "demo/ops/delete", "demo", "sess", "prog_invocations", "agent", "refused", "denied", 1, now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	repo := NewRepository(db)
	got, err := repo.PromoteByID(context.Background(), "prog_invocations", "invocation-proof", "", "operator", "verified", "", nil, nil, now)
	require.NoError(t, err)
	require.Equal(t, []string{"demo/ops/read"}, got.GetCalledBindingIds())
}

func TestListCallableExcludesUnselectedVersionsAndListPaginates(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(internalbindings.Schema), apidb.SchemaProviderFunc(Schema)))
	for _, row := range []struct{ id, name string }{{"lib_a", "a"}, {"lib_b", "b"}, {"lib_c", "c"}} {
		_, err := db.Exec(`INSERT INTO library_programs (id,name,version,source,description,origin,created_at,tier) VALUES (?,?,?,?,?,?,?,?)`, row.id, row.name, 1, "print(1)", row.name, "operator", "2026-09-03T00:00:00Z", "promoted")
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO library_current(name,version) VALUES ('a',1),('c',1)`)
	require.NoError(t, err)
	repo := NewRepository(db)
	callable, err := repo.ListCallable(context.Background())
	require.NoError(t, err)
	require.Len(t, callable, 2)
	page, err := repo.List(context.Background(), 2, 1)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.NotEqual(t, page[0].GetName(), page[1].GetName())
}
