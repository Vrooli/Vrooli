package library

import (
	"context"
	"testing"
	"time"

	"program-runtime/internal/programs"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func TestPromoteAndSetCurrentAreExplicitAndVersioned(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(Schema)))
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

func TestAddCandidateIsAutomaticTierAndIdempotent(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(Schema)))
	repo := NewRepository(db)
	now := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	p := &programsv1.Program{Id: "prog_candidate", Source: "print('candidate')", Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED}
	require.NoError(t, repo.AddCandidate(context.Background(), p, []string{"program-runtime/bindings/list"}, now))
	require.NoError(t, repo.AddCandidate(context.Background(), p, []string{"program-runtime/bindings/list"}, now))

	got, err := repo.Get(context.Background(), "candidate-prog_candidate", 1)
	require.NoError(t, err)
	require.Equal(t, "candidate", got.GetTier())
	require.Equal(t, "agent-authored", got.GetOrigin())
	require.Equal(t, []string{"session_id"}, got.GetDeclaredInputs())
	require.Equal(t, []string{"bounded projection"}, got.GetDeclaredOutputs())
	require.Equal(t, "successful governed program", got.GetCoverage())
	require.Equal(t, now.Format(time.RFC3339Nano), got.GetValidatedAt())
	require.Equal(t, []string{"program-runtime/bindings/list"}, got.GetCalledBindingIds())
	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
}
