package autofix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeAPIFile lays out root/api/<rel> with content, creating parent dirs.
func writeAPIFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, "api", filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

const brokenRowsSource = `package repo

import "database/sql"

func list(db *sql.DB) error {
	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		return err
	}
	for rows.Next() {
	}
	return rows.Err()
}
`

func TestRowsClose_PreviewProposesDefer(t *testing.T) {
	root := t.TempDir()
	path := writeAPIFile(t, root, "internal/repo/list.go", brokenRowsSource)

	candidates, err := Preview(root, []string{RuleDBRowsNotClosed})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	c := candidates[0]
	require.Equal(t, RuleDBRowsNotClosed, c.RuleID)
	require.Equal(t, path, c.FilePath)
	require.False(t, c.Applied)
	require.Equal(t, brokenRowsSource, c.Before)
	require.Contains(t, c.After, "defer rows.Close()")
	// The defer is inserted after the err guard, before the for-loop.
	deferIdx := strings.Index(c.After, "defer rows.Close()")
	forIdx := strings.Index(c.After, "for rows.Next()")
	require.Less(t, deferIdx, forIdx, "defer must precede the rows iteration")
}

func TestRowsClose_ApplyIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path := writeAPIFile(t, root, "internal/repo/list.go", brokenRowsSource)

	applied, err := Apply(root, []string{RuleDBRowsNotClosed})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)

	first, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(first), "defer rows.Close()")
	// The file must still parse as valid Go (the edit is well-formed).
	requireParses(t, string(first))

	// Second apply: the rows are now closed, so detection finds nothing and the
	// file is left byte-for-byte unchanged.
	second, err := Apply(root, []string{RuleDBRowsNotClosed})
	require.NoError(t, err)
	require.Empty(t, second, "second apply must be a no-op")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(first), string(after), "second apply must not change the file")
	require.Equal(t, 1, strings.Count(string(after), "defer rows.Close()"),
		"the defer must be inserted exactly once across two applies")
}

func TestRowsClose_SkipsAlreadyClosed(t *testing.T) {
	root := t.TempDir()
	closed := `package repo

import "database/sql"

func list(db *sql.DB) error {
	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
	}
	return rows.Err()
}
`
	writeAPIFile(t, root, "internal/repo/ok.go", closed)
	candidates, err := Preview(root, []string{RuleDBRowsNotClosed})
	require.NoError(t, err)
	require.Empty(t, candidates, "a closed result set must not be flagged for fixing")
}

func TestRowsClose_SkipsTestAndMigrationFiles(t *testing.T) {
	root := t.TempDir()
	writeAPIFile(t, root, "internal/repo/list_test.go", brokenRowsSource)
	writeAPIFile(t, root, "migrations/seed.go", brokenRowsSource)
	candidates, err := Preview(root, []string{RuleDBRowsNotClosed})
	require.NoError(t, err)
	require.Empty(t, candidates, "exempt paths (test/migrations) must not be fixed")
}

func TestRowsClose_CanFixResolvesLocation(t *testing.T) {
	root := t.TempDir()
	writeAPIFile(t, root, "internal/repo/list.go", brokenRowsSource)
	require.True(t, CanFix(root, RuleDBRowsNotClosed, "api/internal/repo/list.go:6"))
	require.True(t, CanFix(root, RuleDBRowsNotClosed, ""))
	require.False(t, CanFix(root, RuleDBRowsNotClosed, "api/internal/repo/missing.go"))
}
