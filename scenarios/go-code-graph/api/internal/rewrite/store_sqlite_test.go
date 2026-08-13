package rewrite_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	intrewrite "go-code-graph/internal/rewrite"
	"go-code-graph/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"
)

// newSchemaDB returns a connected SQLite handle with the rewrite
// domain's tables created via the same EnsureSchemas path production
// uses. Pattern matches the worked example in testutil/db/sqlite.go.
func newSchemaDB(t *testing.T) *intrewrite.SQLiteStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(intrewrite.Schema),
	))
	return intrewrite.NewSQLiteStore(d, scheduletest.New(time.Unix(1717900000, 0).UTC()))
}

func TestSQLiteStore_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newSchemaDB(t)
	ctx := context.Background()

	plan := intrewrite.Plan{
		ID:         "abc",
		ModulePath: "/tmp/x",
		Operations: []intrewrite.Operation{
			intrewrite.FileMove{From: "old/a.go", To: "new/a.go"},
			intrewrite.ImportRewrite{Old: "example.com/x", New: "example.com/y"},
		},
	}
	require.NoError(t, s.Save(ctx, plan))

	got, err := s.Load(ctx, "abc")
	require.NoError(t, err)
	require.Equal(t, plan.ID, got.ID)
	require.Equal(t, plan.ModulePath, got.ModulePath)
	require.Len(t, got.Operations, 2)

	fm, ok := got.Operations[0].(intrewrite.FileMove)
	require.True(t, ok)
	require.Equal(t, "old/a.go", fm.From)
	require.Equal(t, "new/a.go", fm.To)

	ir, ok := got.Operations[1].(intrewrite.ImportRewrite)
	require.True(t, ok)
	require.Equal(t, "example.com/x", ir.Old)
	require.Equal(t, "example.com/y", ir.New)
}

func TestSQLiteStore_LoadMissingIsPlanNotFound(t *testing.T) {
	t.Parallel()
	s := newSchemaDB(t)

	_, err := s.Load(context.Background(), "nope")
	require.Error(t, err)

	rerr, ok := err.(intrewrite.RewriteError)
	require.True(t, ok, "want RewriteError, got %T", err)
	require.Equal(t, intrewrite.RewriteErrorPlanNotFound, rerr.Kind)
}

func TestSQLiteStore_SaveIsIdempotent(t *testing.T) {
	t.Parallel()
	s := newSchemaDB(t)
	ctx := context.Background()
	plan := intrewrite.Plan{
		ID:         "same",
		ModulePath: "/tmp/x",
		Operations: []intrewrite.Operation{intrewrite.FileMove{From: "a.go", To: "b.go"}},
	}
	require.NoError(t, s.Save(ctx, plan))
	require.NoError(t, s.Save(ctx, plan))
	_, err := s.Load(ctx, "same")
	require.NoError(t, err)
}

func TestSQLiteStore_RecordApplyAppendsRows(t *testing.T) {
	t.Parallel()
	s := newSchemaDB(t)
	ctx := context.Background()

	results := []intrewrite.OperationResult{
		{Operation: intrewrite.FileMove{From: "a.go", To: "b.go"}, Status: intrewrite.OperationStatusOK},
		{Operation: intrewrite.ImportRewrite{Old: "x", New: "y"}, Status: intrewrite.OperationStatusFailed, Message: "boom"},
	}
	require.NoError(t, s.RecordApply(ctx, "plan-1", results))

	// Empty results: no-op.
	require.NoError(t, s.RecordApply(ctx, "plan-2", nil))
}
