package onboard

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestDiagnosticsTail(t *testing.T) {
	t.Run("empty stays empty", func(t *testing.T) {
		require.Equal(t, "", diagnosticsTail(""))
		require.Equal(t, "", diagnosticsTail("\n\n"))
	})

	t.Run("short output passes through, trailing newline trimmed", func(t *testing.T) {
		require.Equal(t, "line one\nline two", diagnosticsTail("line one\nline two\n"))
	})

	t.Run("oversize output is tail-bounded and begins at a line boundary", func(t *testing.T) {
		// Build > diagnosticsTailMaxBytes of numbered lines; the tail must be the END.
		var b strings.Builder
		for b.Len() < diagnosticsTailMaxBytes*2 {
			b.WriteString("some node build output line filler ...........\n")
		}
		b.WriteString("FINAL ERROR: make setup failed\n")
		out := diagnosticsTail(b.String())

		require.LessOrEqual(t, len(out), diagnosticsTailMaxBytes)
		require.True(t, strings.HasSuffix(out, "FINAL ERROR: make setup failed"),
			"the tail must retain the final, most-relevant line")
		require.False(t, strings.HasPrefix(out, "output line filler"),
			"a partial leading line must be dropped so the tail starts clean")
		require.Equal(t, out, strings.TrimLeft(out, " ."),
			"the tail must begin at a whole-line boundary")
	})
}

func TestMigrate(t *testing.T) {
	newDB := func(t *testing.T) *sql.DB {
		t.Helper()
		d, err := sql.Open("sqlite", ":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { d.Close() })
		return d
	}
	ctx := context.Background()

	t.Run("skips a fresh DB with no table", func(t *testing.T) {
		d := newDB(t)
		require.NoError(t, Migrate(ctx, d), "a missing table is a no-op, not an error")
	})

	t.Run("adds failure_detail to a pre-existing table that lacks it", func(t *testing.T) {
		d := newDB(t)
		// Simulate an old DB: the table without the newer column.
		_, err := d.ExecContext(ctx, `CREATE TABLE onboarding_ops (id TEXT PRIMARY KEY, failure_reason TEXT NOT NULL DEFAULT '')`)
		require.NoError(t, err)

		has, err := columnExists(ctx, d, "onboarding_ops", "failure_detail")
		require.NoError(t, err)
		require.False(t, has, "precondition: column absent")

		require.NoError(t, Migrate(ctx, d))
		has, err = columnExists(ctx, d, "onboarding_ops", "failure_detail")
		require.NoError(t, err)
		require.True(t, has, "Migrate must add the column")

		// Idempotent: a second run over the now-current DB is a clean no-op.
		require.NoError(t, Migrate(ctx, d))
	})
}
