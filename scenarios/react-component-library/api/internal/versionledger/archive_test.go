package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
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
	_, err = destination.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, lifecycle_state, created_at) VALUES ('rcl:Other', '9.9.9', 'released', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = destinationRepository.ImportArchive(ctx, archivePath, true)
	require.NoError(t, err)
	var replacedRows int
	require.NoError(t, destination.QueryRowContext(ctx, `SELECT COUNT(*) FROM version_ledger WHERE library_id='rcl:Other'`).Scan(&replacedRows))
	require.Zero(t, replacedRows, "overwrite=true replaces the complete ledger archive")
}

func recoveryFixture() (PublishedReleaseBundle, releaseAttestation) {
	entry := "export const Button = 'verified fixture'"
	story := "export default { title: 'Button' }"
	digest := func(value string) string { sum := sha256.Sum256([]byte(value)); return hex.EncodeToString(sum[:]) }
	bundle := PublishedReleaseBundle{
		SchemaVersion: 1, LibraryID: button224LibraryID, Version: button224Version, Status: "released",
		SourceRevision: button224Revision, PublishedAt: button224PublishedAt,
		Files: []PublishedReleaseBundleFile{{Path: "Button.tsx", Content: entry, SHA256: digest(entry), Entry: true}, {Path: "story.tsx", Content: story, SHA256: digest(story)}},
	}
	attestation := releaseAttestation{libraryID: button224LibraryID, version: button224Version, revision: button224Revision, publishedAt: button224PublishedAt, hashes: map[string]string{"Button.tsx": digest(entry), "story.tsx": digest(story)}}
	return bundle, attestation
}

func writeRecoveryFixture(t *testing.T, bundle PublishedReleaseBundle) (string, []byte) {
	t.Helper()
	data, err := json.Marshal(bundle)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "release-bundle.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path, data
}

func seedRecoveryComponent(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO components (id, library_id, slug, source_path, version, latest_version, indexed_at, updated_at) VALUES ('button-id', ?, 'Button', 'components/Button', '2.2.12', '2.2.12', '2026-10-20T00:00:00Z', '2026-10-20T00:00:00Z')`, button224LibraryID)
	require.NoError(t, err)
}

func recoveryCounts(t *testing.T, database *sql.DB) [3]int {
	t.Helper()
	var counts [3]int
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM component_versions`).Scan(&counts[0]))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM component_version_files`).Scan(&counts[1]))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM version_ledger`).Scan(&counts[2]))
	return counts
}

func recoveryTableSnapshot(t *testing.T, database *sql.DB) string {
	t.Helper()
	var snapshot []byte
	for _, table := range []string{"component_versions", "component_version_files", "version_ledger"} {
		rows, err := database.Query(`SELECT * FROM ` + table + ` ORDER BY rowid`)
		require.NoError(t, err)
		columns, err := rows.Columns()
		require.NoError(t, err)
		var tableRows [][]any
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for i := range values {
				destinations[i] = &values[i]
			}
			require.NoError(t, rows.Scan(destinations...))
			tableRows = append(tableRows, values)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		entry, err := json.Marshal(struct {
			Table string  `json:"table"`
			Rows  [][]any `json:"rows"`
		}{table, tableRows})
		require.NoError(t, err)
		snapshot = append(snapshot, entry...)
	}
	return string(snapshot)
}

func seedOrphanRecoveryLedger(t *testing.T, database *sql.DB, lifecycle, createdAt, releasedAt, retiredAt string, fileCount int) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO version_ledger (library_id, version, created_at, released_at, retired_at, lifecycle_state, gate_pass_count, gate_fail_count, test_runs, test_pass_rate, adoption_current, adoption_peak, file_count, lines_of_code, dependency_count)
VALUES (?, ?, ?, ?, ?, ?, 7, 2, 9, 0.75, 11, 19, ?, 83, 4)`, button224LibraryID, button224Version, createdAt, releasedAt, retiredAt, lifecycle, fileCount)
	require.NoError(t, err)
}

func TestRecoverButton224PublishedReleaseValidatesAndAddsOnlyOneAttestedRelease(t *testing.T) {
	ctx := context.Background()
	database := archiveDatabase(t)
	seedRecoveryComponent(t, database)
	_, err := database.Exec(`
INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, created_at, released_at, presence)
VALUES ('existing-version', 'button-id', ?, '2.2.12', 'released', 'components/Button/versions/2.2.12/Button.tsx', 'keep-existing', 'existing-hash', '2026-10-20T00:00:00Z', '2026-09-10T00:00:00Z', '2026-09-10T00:00:00Z', 'materialized');
INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry)
VALUES ('existing-version', 'Button.tsx', 'keep-existing', 'existing-hash', 1);
INSERT INTO version_ledger (library_id, version, created_at, released_at, lifecycle_state, file_count)
VALUES (?, '2.2.12', '2026-09-10T00:00:00Z', '2026-09-10T00:00:00Z', 'released', 1);`, button224LibraryID, button224LibraryID)
	require.NoError(t, err)
	bundle, attestation := recoveryFixture()
	path, data := writeRecoveryFixture(t, bundle)
	repository := NewRepository(database, t.TempDir())

	files, err := validatePublishedReleaseBundle(bundle, attestation)
	require.NoError(t, err)
	require.Len(t, files, 2)
	before := recoveryCounts(t, database)
	summary, componentID, err := repository.restoreValidatedRelease(ctx, path, data, bundle, attestation)
	require.NoError(t, err)
	require.Equal(t, "button-id", componentID)
	require.Equal(t, map[string]int{"component_versions": 1, "component_version_files": 2, "version_ledger": 1}, summary.RowCounts)
	require.Equal(t, [3]int{1, 1, 1}, before)
	require.Equal(t, [3]int{2, 3, 2}, recoveryCounts(t, database))
	var latest, state, presence string
	require.NoError(t, database.QueryRow(`SELECT latest_version FROM components WHERE id='button-id'`).Scan(&latest))
	require.NoError(t, database.QueryRow(`SELECT lifecycle_state FROM version_ledger WHERE library_id=? AND version=?`, button224LibraryID, button224Version).Scan(&state))
	require.NoError(t, database.QueryRow(`SELECT presence FROM component_versions WHERE component_id='button-id' AND version=?`, button224Version).Scan(&presence))
	require.Equal(t, "2.2.12", latest)
	require.Equal(t, "released", state)
	require.Equal(t, "evicted", presence)
	var existingContent, existingState string
	require.NoError(t, database.QueryRow(`SELECT content FROM component_version_files WHERE version_id='existing-version'`).Scan(&existingContent))
	require.NoError(t, database.QueryRow(`SELECT lifecycle_state FROM version_ledger WHERE library_id=? AND version='2.2.12'`, button224LibraryID).Scan(&existingState))
	require.Equal(t, "keep-existing", existingContent)
	require.Equal(t, "released", existingState)
}

func TestRecoverButton224PublishedReleaseRejectsInvalidBundlesWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PublishedReleaseBundle)
	}{
		{name: "hash", mutate: func(b *PublishedReleaseBundle) { b.Files[0].SHA256 = "bad" }},
		{name: "identity", mutate: func(b *PublishedReleaseBundle) { b.LibraryID = "react-component-library:Other" }},
		{name: "version", mutate: func(b *PublishedReleaseBundle) { b.Version = "2.2.5" }},
		{name: "published status", mutate: func(b *PublishedReleaseBundle) { b.Status = "draft" }},
		{name: "source revision", mutate: func(b *PublishedReleaseBundle) { b.SourceRevision = "0000000000000000000000000000000000000000" }},
		{name: "publication time", mutate: func(b *PublishedReleaseBundle) { b.PublishedAt = "2026-08-31T05:19:39Z" }},
		{name: "file count", mutate: func(b *PublishedReleaseBundle) { b.Files = b.Files[:1] }},
		{name: "path traversal", mutate: func(b *PublishedReleaseBundle) { b.Files[0].Path = "../Button.tsx" }},
		{name: "unexpected path", mutate: func(b *PublishedReleaseBundle) { b.Files[0].Path = "Button/2.2.4.tsx" }},
		{name: "duplicate path", mutate: func(b *PublishedReleaseBundle) { b.Files[1].Path = "Button.tsx" }},
		{name: "entry marker", mutate: func(b *PublishedReleaseBundle) { b.Files[0].Entry = false }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := archiveDatabase(t)
			seedRecoveryComponent(t, database)
			bundle, attestation := recoveryFixture()
			test.mutate(&bundle)
			path, data := writeRecoveryFixture(t, bundle)
			before := recoveryCounts(t, database)
			_, _, err := NewRepository(database, t.TempDir()).restoreValidatedRelease(context.Background(), path, data, bundle, attestation)
			require.Error(t, err)
			require.Equal(t, before, recoveryCounts(t, database))
		})
	}
}

func TestRecoverButton224PublishedReleaseRejectsMalformedBundleWithoutMutation(t *testing.T) {
	database := archiveDatabase(t)
	seedRecoveryComponent(t, database)
	path := filepath.Join(t.TempDir(), "malformed-bundle.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":`), 0o600))
	before := recoveryCounts(t, database)
	_, _, err := NewRepository(database, t.TempDir()).RecoverButton224PublishedRelease(context.Background(), path)
	require.Error(t, err)
	require.Equal(t, before, recoveryCounts(t, database))
}

func TestRecoverButton224PublishedReleaseDuplicateLeavesLedgerUnchanged(t *testing.T) {
	database := archiveDatabase(t)
	seedRecoveryComponent(t, database)
	bundle, attestation := recoveryFixture()
	path, data := writeRecoveryFixture(t, bundle)
	_, err := database.Exec(`
INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, created_at, released_at, presence)
VALUES ('existing-button-224', 'button-id', ?, ?, 'released', 'components/Button/versions/2.2.4/Button.tsx', 'existing', 'existing-hash', '2026-10-20T00:00:00Z', ?, ?, 'materialized');
INSERT INTO version_ledger (library_id, version, created_at, released_at, lifecycle_state, file_count)
VALUES (?, ?, ?, ?, 'released', 1);`, button224LibraryID, button224Version, button224PublishedAt, button224PublishedAt, button224LibraryID, button224Version, button224PublishedAt, button224PublishedAt)
	require.NoError(t, err)
	root := t.TempDir()
	repository := NewRepository(database, root)
	before := recoveryTableSnapshot(t, database)
	_, _, err = repository.restoreValidatedRelease(context.Background(), path, data, bundle, attestation)
	require.ErrorContains(t, err, "already present")
	require.Equal(t, before, recoveryTableSnapshot(t, database))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestRecoverButton224PublishedReleasePartialMaterializedDuplicateLeavesTablesUnchanged(t *testing.T) {
	database := archiveDatabase(t)
	seedRecoveryComponent(t, database)
	_, err := database.Exec(`INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, created_at, released_at, presence)
VALUES ('partial-button-224', 'unexpected-component', ?, ?, 'released', 'components/Button/versions/2.2.4/Button.tsx', 'partial', 'partial-hash', '2026-10-20T00:00:00Z', ?, ?, 'evicted')`, button224LibraryID, button224Version, button224PublishedAt, button224PublishedAt)
	require.NoError(t, err)
	bundle, attestation := recoveryFixture()
	path, data := writeRecoveryFixture(t, bundle)
	before := recoveryTableSnapshot(t, database)
	_, _, err = NewRepository(database, t.TempDir()).restoreValidatedRelease(context.Background(), path, data, bundle, attestation)
	require.ErrorContains(t, err, "already present")
	require.Equal(t, before, recoveryTableSnapshot(t, database))
}

func TestRecoverButton224PublishedReleaseCompletesMatchingOrphanLedgerWithoutRewritingIt(t *testing.T) {
	database := archiveDatabase(t)
	seedRecoveryComponent(t, database)
	seedOrphanRecoveryLedger(t, database, "released", button224PublishedAt, button224PublishedAt, "", 2)
	ledgerBefore, err := database.Query(`SELECT * FROM version_ledger WHERE library_id=? AND version=?`, button224LibraryID, button224Version)
	require.NoError(t, err)
	var ledgerBytesBefore [][]byte
	columns, err := ledgerBefore.Columns()
	require.NoError(t, err)
	values := make([]any, len(columns))
	destinations := make([]any, len(columns))
	for i := range values {
		destinations[i] = &values[i]
	}
	require.True(t, ledgerBefore.Next())
	require.NoError(t, ledgerBefore.Scan(destinations...))
	for _, value := range values {
		ledgerBytesBefore = append(ledgerBytesBefore, []byte(fmt.Sprint(value)))
	}
	require.NoError(t, ledgerBefore.Close())
	bundle, attestation := recoveryFixture()
	path, data := writeRecoveryFixture(t, bundle)
	summary, componentID, err := NewRepository(database, t.TempDir()).restoreValidatedRelease(context.Background(), path, data, bundle, attestation)
	require.NoError(t, err)
	require.Equal(t, "button-id", componentID)
	require.Equal(t, map[string]int{"component_versions": 1, "component_version_files": 2, "version_ledger": 0}, summary.RowCounts)
	require.Equal(t, [3]int{1, 2, 1}, recoveryCounts(t, database))
	var gotContent, gotHash string
	require.NoError(t, database.QueryRow(`SELECT content, content_sha256 FROM component_versions WHERE library_id=? AND version=?`, button224LibraryID, button224Version).Scan(&gotContent, &gotHash))
	require.Equal(t, "export const Button = 'verified fixture'", gotContent)
	require.Equal(t, bundle.Files[0].SHA256, gotHash)
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM component_version_files WHERE version_id=(SELECT id FROM component_versions WHERE library_id=? AND version=?)`, button224LibraryID, button224Version).Scan(&gotHash))
	require.Equal(t, "2", gotHash)
	ledgerAfter, err := database.Query(`SELECT * FROM version_ledger WHERE library_id=? AND version=?`, button224LibraryID, button224Version)
	require.NoError(t, err)
	valuesAfter := make([]any, len(columns))
	destinationsAfter := make([]any, len(columns))
	for i := range valuesAfter {
		destinationsAfter[i] = &valuesAfter[i]
	}
	require.True(t, ledgerAfter.Next())
	require.NoError(t, ledgerAfter.Scan(destinationsAfter...))
	require.NoError(t, ledgerAfter.Close())
	for i, value := range valuesAfter {
		require.Equal(t, ledgerBytesBefore[i], []byte(fmt.Sprint(value)), "ledger field %s remains byte-for-byte identical", columns[i])
	}
}

func TestRecoverButton224PublishedReleaseRejectsMismatchedOrphanMetadataWithoutMutation(t *testing.T) {
	tests := []struct {
		name                                        string
		lifecycle, createdAt, releasedAt, retiredAt string
		fileCount                                   int
	}{
		{name: "lifecycle", lifecycle: "archived", createdAt: button224PublishedAt, releasedAt: button224PublishedAt, fileCount: 2},
		{name: "created timestamp", lifecycle: "released", createdAt: "2026-08-31T05:19:39Z", releasedAt: button224PublishedAt, fileCount: 2},
		{name: "released timestamp", lifecycle: "released", createdAt: button224PublishedAt, releasedAt: "2026-08-31T05:19:39Z", fileCount: 2},
		{name: "retired timestamp", lifecycle: "released", createdAt: button224PublishedAt, releasedAt: button224PublishedAt, retiredAt: "2026-09-01T00:00:00Z", fileCount: 2},
		{name: "file count", lifecycle: "released", createdAt: button224PublishedAt, releasedAt: button224PublishedAt, fileCount: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := archiveDatabase(t)
			seedRecoveryComponent(t, database)
			seedOrphanRecoveryLedger(t, database, test.lifecycle, test.createdAt, test.releasedAt, test.retiredAt, test.fileCount)
			bundle, attestation := recoveryFixture()
			path, data := writeRecoveryFixture(t, bundle)
			before := recoveryTableSnapshot(t, database)
			_, _, err := NewRepository(database, t.TempDir()).restoreValidatedRelease(context.Background(), path, data, bundle, attestation)
			require.ErrorContains(t, err, "orphan ledger metadata does not match")
			require.Equal(t, before, recoveryTableSnapshot(t, database))
		})
	}
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

func TestDoctorReportsVersionLedgerRowWithoutComponentVersion(t *testing.T) {
	ctx := context.Background()
	database := archiveDatabase(t)
	_, err := database.ExecContext(ctx, `
INSERT INTO version_ledger (library_id, version, created_at, released_at, retired_at, lifecycle_state, file_count)
VALUES ('rcl:Button', '2.2.4', '2026-08-30T05:19:38Z', '2026-08-31T05:19:38Z', '', 'released', 2);
`)
	require.NoError(t, err)

	issues, err := NewRepository(database, t.TempDir()).Doctor(ctx)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, "rcl:Button", issues[0].LibraryID)
	require.Equal(t, "2.2.4", issues[0].Version)
	require.Equal(t, "version ledger row has no component version", issues[0].Reason)
	require.Equal(t, &DoctorLedgerMetadata{
		CreatedAt:      "2026-08-30T05:19:38Z",
		ReleasedAt:     "2026-08-31T05:19:38Z",
		RetiredAt:      "",
		LifecycleState: "released",
		FileCount:      2,
	}, issues[0].LedgerMetadata)

	var after DoctorLedgerMetadata
	err = database.QueryRowContext(ctx, `
SELECT created_at, released_at, retired_at, lifecycle_state, file_count
FROM version_ledger WHERE library_id = 'rcl:Button' AND version = '2.2.4'
`).Scan(&after.CreatedAt, &after.ReleasedAt, &after.RetiredAt, &after.LifecycleState, &after.FileCount)
	require.NoError(t, err)
	require.Equal(t, *issues[0].LedgerMetadata, after, "Doctor must not mutate the orphan ledger row")
}

func TestDoctorDoesNotReportVersionLedgerRowWithComponentVersion(t *testing.T) {
	ctx := context.Background()
	database := archiveDatabase(t)
	_, err := database.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, presence)
VALUES ('v-button-224', 'button', 'rcl:Button', '2.2.4', 'released', 'components/Button/versions/2.2.4/Button.tsx', '', '', '2026-08-31T05:19:38Z', 'materialized');
INSERT INTO version_ledger (library_id, version, lifecycle_state, created_at)
VALUES ('rcl:Button', '2.2.4', 'released', '2026-08-31T05:19:38Z');
`)
	require.NoError(t, err)

	issues, err := NewRepository(database, t.TempDir()).Doctor(ctx)
	require.NoError(t, err)
	require.Empty(t, issues)
}
