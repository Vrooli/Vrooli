package versionledger

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"

	"react-component-library/internal/components"
)

func archiveDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(components.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	return database
}

func TestArchiveRoundTripPreservesLedgerAndMirror(t *testing.T) {
	ctx := context.Background()
	source := archiveDatabase(t)
	_, err := source.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence)
VALUES ('v1', 'button', 'rcl:Button', '1.0.0', 'released', 'components/Button/versions/1.0.0/Button.tsx', 'export const Button = 1', 'hash', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', 'evicted');
INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry, slot)
VALUES ('v1', 'Button.tsx', 'export const Button = 1', 'hash', 1, '');
INSERT INTO version_ledger (library_id, version, lifecycle_state, created_at)
VALUES ('rcl:Button', '1.0.0', 'archived', '2026-01-01T00:00:00Z');
`)
	require.NoError(t, err)

	repository := NewRepository(source, t.TempDir())
	archivePath := filepath.Join(t.TempDir(), "ledger.json")
	exported, err := repository.ExportArchive(ctx, archivePath)
	require.NoError(t, err)
	require.Equal(t, archiveSchemaVersion, exported.SchemaVersion)
	require.Equal(t, 1, exported.RowCounts["component_versions"])
	require.Equal(t, 1, exported.RowCounts["component_version_files"])
	require.Equal(t, 1, exported.RowCounts["version_ledger"])
	require.FileExists(t, archivePath)

	destination := archiveDatabase(t)
	destinationRepository := NewRepository(destination, t.TempDir())
	imported, err := destinationRepository.ImportArchive(ctx, archivePath, false)
	require.NoError(t, err)
	require.Equal(t, exported.Checksum, imported.Checksum)
	var presence, content, lifecycle string
	require.NoError(t, destination.QueryRowContext(ctx, `SELECT presence FROM component_versions WHERE id='v1'`).Scan(&presence))
	require.NoError(t, destination.QueryRowContext(ctx, `SELECT content FROM component_version_files WHERE version_id='v1'`).Scan(&content))
	require.NoError(t, destination.QueryRowContext(ctx, `SELECT lifecycle_state FROM version_ledger WHERE library_id='rcl:Button' AND version='1.0.0'`).Scan(&lifecycle))
	require.Equal(t, "evicted", presence)
	require.Equal(t, "export const Button = 1", content)
	require.Equal(t, "archived", lifecycle)

	_, err = destinationRepository.ImportArchive(ctx, archivePath, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty table")
	_, err = destinationRepository.ImportArchive(ctx, archivePath, true)
	require.NoError(t, err)
}

func TestDoctorReportsEvictedMirrorIntegrity(t *testing.T) {
	ctx := context.Background()
	database := archiveDatabase(t)
	_, err := database.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence)
VALUES ('v1', 'button', 'rcl:Button', '1.0.0', 'archived', 'components/Button/versions/1.0.0/Button.tsx', '', '', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '', 'evicted');
INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry, slot)
VALUES ('v1', 'Button.tsx', 'actual', 'expected', 1, '');
`)
	require.NoError(t, err)

	issues, err := NewRepository(database, t.TempDir()).Doctor(ctx)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, "rcl:Button", issues[0].LibraryID)
	require.Equal(t, "Button.tsx", issues[0].Path)
	require.Equal(t, "mirror content hash mismatch", issues[0].Reason)
}

func TestDoctorReportsMissingMirrorForEvictedVersion(t *testing.T) {
	ctx := context.Background()
	database := archiveDatabase(t)
	_, err := database.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content_sha256, indexed_at, presence)
VALUES ('v-missing', 'button', 'rcl:Button', '0.9.0', 'archived', 'components/Button/versions/0.9.0/Button.tsx', '', '2026-01-01T00:00:00Z', 'evicted');
`)
	require.NoError(t, err)

	issues, err := NewRepository(database, t.TempDir()).Doctor(ctx)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, "rcl:Button", issues[0].LibraryID)
	require.Equal(t, "evicted version has no file mirror", issues[0].Reason)
}
