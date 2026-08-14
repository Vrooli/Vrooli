package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestImportTreeReportsBrokenReferencesWithoutCopyingNarrative(t *testing.T) { // [REQ:MIG-001] [REQ:MIG-002] [REQ:MIG-003]
	root := filepath.Join(t.TempDir(), "testdata")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "offer.md"), []byte("# Fixture offer\n\n**SKU ID:** `fixture-offer`\n**Status:** `candidate`\n\nThis narrative is intentionally excluded from the imported node.\n\nSee [missing deliverable](missing-deliverable.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "notes.md"), []byte("# Notes\n\nNarrative-only fixture.\n"), 0o644))
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	s := NewStore(db, nil)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return s.Schema() })))
	report, err := s.ImportTree(context.Background(), root, "operator")
	require.NoError(t, err)
	require.Len(t, report.Files, 2)
	require.Equal(t, 1, report.Files[0].Read)
	require.Equal(t, 1, report.Files[0].Written)
	require.Equal(t, 1, report.Findings)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM migration_findings`).Scan(&count))
	require.Equal(t, 1, count)
	var narrative string
	require.Error(t, db.QueryRowContext(context.Background(), `SELECT name FROM nodes WHERE name LIKE '%narrative%'`).Scan(&narrative))
}

func TestImportTreeRejectsLiveOrUnscopedRoots(t *testing.T) {
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	s := NewStore(db, nil)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return s.Schema() })))
	_, err := s.ImportTree(context.Background(), t.TempDir(), "operator")
	require.ErrorContains(t, err, "fixture root")
}
