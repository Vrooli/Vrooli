package programs

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/testutil/db"
)

func newProgramsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	return d
}

func TestSQLiteRepositoryRoundTripAfterRepositoryRestart(t *testing.T) { // [REQ:PRT-P1-006]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	want := &programsv1.Program{Id: "prog_persisted", SessionId: "sess_1", Source: "raise ValueError()", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: "failed", Stdout: "partial", FailureDetail: "field title: invalid", FailureShape: "field title", ContextBytes: 128, CreatedAt: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), OutputLimitBytes: 4096}
	require.NoError(t, NewRepository(d).Save(ctx, want))

	got, err := NewRepository(d).Get(ctx, want.Id)
	require.NoError(t, err)
	require.Equal(t, want.Id, got.Id)
	require.Equal(t, want.Source, got.Source)
	require.Equal(t, want.FailureDetail, got.FailureDetail)
	require.Equal(t, want.Provenance, got.Provenance)
	require.Equal(t, want.ContextBytes, got.ContextBytes)
}

func TestSQLiteRepositoryMineFailuresExcludesOperatorByDefault(t *testing.T) { // [REQ:PRT-P1-008]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	for i, provenance := range []programsv1.Provenance{programsv1.Provenance_PROVENANCE_OPERATOR, programsv1.Provenance_PROVENANCE_AGENT, programsv1.Provenance_PROVENANCE_AGENT} {
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "prog_" + string(rune('a'+i)), SessionId: "sess_1", Source: "x", Provenance: provenance, Status: "failed", FailureShape: "same failure", CreatedAt: time.Date(2026, 8, 11, 14, i, 0, 0, time.UTC).Format(time.RFC3339Nano)}))
	}
	shapes, err := repo.MineFailures(ctx, false, time.Time{})
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(2), shapes[0].Count)
	shapes, err = repo.MineFailures(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, int64(3), shapes[0].Count)
}

func TestSQLiteRepositoryMineFailuresHonorsTimeWindow(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	for id, created := range map[string]time.Time{"old": old, "new": now} {
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: id, SessionId: "sess_1", Source: "x", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: "failed", FailureShape: "window", CreatedAt: created.Format(time.RFC3339Nano)}))
	}
	shapes, err := repo.MineFailures(ctx, false, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(1), shapes[0].Count)
}
