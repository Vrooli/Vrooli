package harness

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/facets"
	"vrooli-memory/internal/journal"
	"vrooli-memory/internal/testutil/mocks"
)

func TestClaudeImportIsContentAddressedAndReadOnly(t *testing.T) { // [REQ:VMEM-P0-011]
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.md"), []byte("first memory"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "two.md"), []byte("second memory"), 0o600))
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:harness-import?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(Schema)))
	fr := facets.NewSQLiteRepository(db.Primary())
	require.NoError(t, fr.Seed(context.Background()))
	svc := journal.NewService(journal.NewSQLiteRepository(db.Primary()), &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, facets.NewService(fr))
	importer := NewImporter(svc, dir)
	first, err := importer.Import(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Equal(t, 2, first.Imported)
	entries, err := svc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		require.Equal(t, "claude-code", entry.Import.Harness)
		require.NotEmpty(t, entry.Import.Path)
		require.False(t, entry.Import.ImportedAt.IsZero())
	}
	second, err := importer.Import(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Zero(t, second.Imported)
	require.Equal(t, 2, second.Existing)
	_, err = os.ReadFile(filepath.Join(dir, "one.md"))
	require.NoError(t, err)
}

func TestStartPersistsProgressAndJoinsRepeatedRequest(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.md"), []byte("first memory"), 0o600))
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:harness-runs?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(Schema)))
	fr := facets.NewSQLiteRepository(db.Primary())
	require.NoError(t, fr.Seed(context.Background()))
	svc := journal.NewService(journal.NewSQLiteRepository(db.Primary()), &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, facets.NewService(fr))
	importer := NewImporter(svc, dir, db.Primary())
	first, joined, err := importer.Start(context.Background(), "claude-code")
	require.NoError(t, err)
	require.False(t, joined)
	second, joined, err := importer.Start(context.Background(), "claude-code")
	require.NoError(t, err)
	require.True(t, joined)
	require.Equal(t, first.ID, second.ID)
	require.Eventually(t, func() bool {
		run, err := importer.Status(context.Background(), first.ID, "")
		return err == nil && run.Status == ImportRunCompleted && run.ProcessedSources == 1 && run.ImportedCount == 1
	}, time.Second, 10*time.Millisecond)
	run, err := importer.Status(context.Background(), "", "claude-code")
	require.NoError(t, err)
	require.Equal(t, first.ID, run.ID)
}

func TestCaptureAndLaterImportConvergeOnOneEntry(t *testing.T) { // [REQ:VMEM-P1-008]
	dir := t.TempDir()
	path := filepath.Join(dir, "native.md")
	const body = "native durable memory"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:harness-capture-dedup?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(Schema)))
	fr := facets.NewSQLiteRepository(db.Primary())
	require.NoError(t, fr.Seed(context.Background()))
	svc := journal.NewService(journal.NewSQLiteRepository(db.Primary()), &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, facets.NewService(fr))
	importer := NewImporter(svc, dir)
	captured, err := importer.Capture(context.Background(), "claude-code", path, body)
	require.NoError(t, err)
	require.NotEmpty(t, captured.ID)
	result, err := importer.Import(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Zero(t, result.Imported)
	require.Equal(t, 1, result.Existing)
	entries, err := svc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestSwarmManagerRecordsImportIsWorkRecordAndIdempotent(t *testing.T) { // [REQ:VMEM-P1-001]
	dir := t.TempDir()
	record := `{"id":"rec-1","trigger":"need a durable memory","approach":"implemented the import","evidence":"focused tests pass","outcome":"ready for validation"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rec-1.json"), []byte(record), 0o600))
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:swarm-record-import?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(Schema)))
	fr := facets.NewSQLiteRepository(db.Primary())
	require.NoError(t, fr.Seed(context.Background()))
	svc := journal.NewService(journal.NewSQLiteRepository(db.Primary()), &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, facets.NewService(fr))
	importer := NewImporter(svc, t.TempDir())
	importer.adapters["swarm-manager-records"] = AdapterDescriptor{HarnessID: "swarm-manager-records", Locations: []string{dir}, Format: JSONL, Extract: swarmRecord, Provenance: Provenance{SourceRuntime: "swarm-manager"}}

	first, err := importer.Import(context.Background(), "swarm-manager-records", false)
	require.NoError(t, err)
	require.Equal(t, 1, first.Imported)
	entries, err := svc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "work-record", entries[0].Kind)

	second, err := importer.Import(context.Background(), "swarm-manager-records", false)
	require.NoError(t, err)
	require.Zero(t, second.Imported)
	require.Equal(t, 1, second.Existing)
}

func TestImportWorkerCountIsBoundedAndConfigurable(t *testing.T) {
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "8")
	require.Equal(t, 8, importWorkerCount())
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "99")
	require.Equal(t, 16, importWorkerCount())
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "0")
	require.Equal(t, 4, importWorkerCount())
}
