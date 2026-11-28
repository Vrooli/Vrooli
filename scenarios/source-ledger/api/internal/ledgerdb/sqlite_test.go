package ledgerdb

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"source-ledger/internal/files"
	"source-ledger/internal/journal"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention/sqlitemaintenance"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
)

func isolatedConfig(t *testing.T, cfg storage.SQLiteConfig) storage.SQLiteConfig {
	t.Helper()
	cfg.RootOverride = t.TempDir()
	cfg.EnvGet = func(string) string { return "" }
	return cfg
}

func TestNewStoresInitializeBeforeTheirSchemas(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		cfg  storage.SQLiteConfig
		init func(*database.RoutedDB) error
	}{
		{"journal", LedgerConfig(), func(db *database.RoutedDB) error {
			return database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(journal.Schema))
		}},
		{"research index", FilesIndexConfig(), func(db *database.RoutedDB) error {
			_, err := files.NewService(ctx, db, nil, nil, nil)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := isolatedConfig(t, tc.cfg)
			db, err := Open(ctx, cfg)
			require.NoError(t, err)
			require.NoError(t, tc.init(db))
			require.NoError(t, db.Close())
			db, err = Open(ctx, cfg)
			require.NoError(t, err)
			defer db.Close()
			stats, err := sqlitemaintenance.Measure(ctx, db)
			require.NoError(t, err)
			require.Equal(t, "incremental", stats.AutoVacuum)
			var journalMode string
			require.NoError(t, db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
			require.Equal(t, "wal", journalMode)
			var synchronous int
			require.NoError(t, db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous))
			require.Equal(t, 1, synchronous)
		})
	}
}

func TestLegacyLedgerPreservesCursorRowIDs(t *testing.T) {
	ctx := context.Background()
	cfg := isolatedConfig(t, LedgerConfig())
	db := legacyDatabase(t, cfg)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(journal.Schema)))
	exec(t, db, `DROP TRIGGER entries_append_position; DROP TABLE journal_entry_positions;
		INSERT INTO entries(rowid,id,body,facet_id,kind,created_at)
		VALUES(2,'first','alpha','facet','note','2026-11-26'),(90,'last','bravo','facet','note','2026-11-26');
		UPDATE journal_high_water_mark SET max_rowid=90 WHERE id=1;
		CREATE TABLE pending_cleanup(body BLOB); INSERT INTO pending_cleanup VALUES(zeroblob(1048576));
		DROP TABLE pending_cleanup; PRAGMA wal_checkpoint(TRUNCATE);`)
	require.NoError(t, db.Close())

	db, err := Open(ctx, cfg)
	require.NoError(t, err)
	defer db.Close()
	stats, err := sqlitemaintenance.Measure(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "none", stats.AutoVacuum)
	require.Positive(t, stats.FreePages)
	result, err := sqlitemaintenance.Maintain(ctx, db, sqlitemaintenance.Options{Policy: sqlitemaintenance.Policy{
		FreeTriggerBytes: 4096, BatchPages: 128, TickBudget: time.Second,
	}})
	require.NoError(t, err)
	require.Equal(t, "none", result.After.AutoVacuum)
	require.NotEmpty(t, result.Deferred)
	require.NoError(t, journal.EnsureMigrations(ctx, db.Primary()))
	var rowID, marked int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT rowid FROM entries WHERE id='last'`).Scan(&rowID))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT max_rowid FROM journal_high_water_mark WHERE id=1`).Scan(&marked))
	require.Equal(t, 90, rowID)
	require.Equal(t, rowID, marked)
}

func TestLegacyFilesIndexConversionHoldsLifecycleLockAndPreservesFTS(t *testing.T) {
	ctx := context.Background()
	cfg := isolatedConfig(t, FilesIndexConfig())
	db := legacyDatabase(t, cfg)
	_, err := files.NewService(ctx, db, nil, nil, nil)
	require.NoError(t, err)
	exec(t, db, `INSERT INTO chunks VALUES (43,'research','topic/page.md','topic','feed',1,1,'Heading','live');
		INSERT INTO chunks_fts(rowid,path,heading,text) VALUES (43,'topic/page.md','Heading','stable canary');
		INSERT INTO vectors VALUES ('live',X'0000803f');`)
	require.NoError(t, db.Close())

	lock := filepath.Join(t.TempDir(), "scenario-source-ledger.lock")
	_, err = convertFilesIndex(ctx, cfg, lock, func() string { return "running" })
	require.ErrorContains(t, err, "must be stopped")
	_, err = convertFilesIndex(ctx, cfg, lock, func() string { return "" })
	require.ErrorContains(t, err, "must be stopped")
	status := func() string {
		contender, err := platform.OpenLockFile(lock, 0o644)
		require.NoError(t, err)
		defer contender.Close()
		release, err := platform.LockFile(contender, true)
		if release != nil {
			release()
		}
		require.ErrorIs(t, err, platform.ErrLockUnavailable)
		return "stopped"
	}
	result, err := convertFilesIndex(ctx, cfg, lock, status)
	require.NoError(t, err)
	require.True(t, result.Converted)
	require.Equal(t, "none", result.Before.AutoVacuum)
	require.Zero(t, result.Before.FreePages)
	require.Equal(t, "incremental", result.After.AutoVacuum)
	result, err = convertFilesIndex(ctx, cfg, lock, status)
	require.NoError(t, err)
	require.False(t, result.Converted)

	db, err = Open(ctx, cfg)
	require.NoError(t, err)
	defer db.Close()
	var chunkID int
	var vector string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT c.id,hex(v.vec) FROM chunks_fts
		JOIN chunks c ON c.id=chunks_fts.rowid JOIN vectors v ON v.hash=c.hash
		WHERE chunks_fts MATCH 'canary'`).Scan(&chunkID, &vector))
	require.Equal(t, 43, chunkID)
	require.Equal(t, "0000803F", vector)
}

func TestConversionRejectsWrongSchemaAndMissingFiles(t *testing.T) {
	ctx := context.Background()
	cfg := isolatedConfig(t, FilesIndexConfig())
	lock := filepath.Join(t.TempDir(), "scenario-source-ledger.lock")
	stopped := func() string { return "stopped" }
	_, err := convertFilesIndex(ctx, cfg, lock, stopped)
	require.Error(t, err)
	db := legacyDatabase(t, cfg)
	exec(t, db, `CREATE TABLE entries(id TEXT PRIMARY KEY); INSERT INTO entries VALUES ('ledger')`)
	require.NoError(t, db.Close())
	_, err = convertFilesIndex(ctx, cfg, lock, stopped)
	require.ErrorContains(t, err, "does not have the Source Ledger file-index schema")
	db = legacyDatabase(t, cfg)
	defer db.Close()
	stats, err := sqlitemaintenance.Measure(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "none", stats.AutoVacuum)
}

func TestScheduledMaintenanceReclaimsMeasuredBytes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ledger, err := Open(ctx, isolatedConfig(t, LedgerConfig()))
	require.NoError(t, err)
	defer ledger.Close()
	index, err := Open(ctx, isolatedConfig(t, FilesIndexConfig()))
	require.NoError(t, err)
	defer index.Close()
	_, err = files.NewService(ctx, index, nil, nil, nil)
	require.NoError(t, err)
	exec(t, index, `INSERT INTO vectors(hash,vec) VALUES ('retained', X'0000803f');
		WITH RECURSIVE sequence(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM sequence WHERE n<512)
		INSERT INTO vectors(hash,vec) SELECT 'expired-'||n, zeroblob(4096) FROM sequence;
		PRAGMA wal_checkpoint(TRUNCATE);
		DELETE FROM vectors WHERE hash LIKE 'expired-%'; PRAGMA wal_checkpoint(TRUNCATE);`)
	before, err := sqlitemaintenance.Measure(ctx, index)
	require.NoError(t, err)
	require.Greater(t, before.FreeBytes, int64(1<<20))

	done := make(chan struct{})
	go func() {
		defer close(done)
		runMaintenance(ctx, schedule.System(), slog.New(slog.NewTextHandler(io.Discard, nil)), 10*time.Millisecond,
			sqlitemaintenance.Policy{FreeTriggerBytes: 8192, FreeFloorBytes: 4096, BatchPages: 128, TickBudget: time.Second}, ledger, index)
	}()
	defer func() { cancel(); <-done }()
	var after sqlitemaintenance.Stats
	require.Eventually(t, func() bool {
		var err error
		after, err = sqlitemaintenance.Measure(ctx, index)
		return err == nil && after.DatabaseBytes < before.DatabaseBytes/2 && after.WALBytes == 0
	}, 5*time.Second, 20*time.Millisecond)
	t.Logf("scheduled index reclamation returned %d bytes and %d pages (WAL %d → %d bytes)",
		before.DatabaseBytes-after.DatabaseBytes, before.PageCount-after.PageCount, before.WALBytes, after.WALBytes)
	var vector string
	require.NoError(t, index.QueryRowContext(ctx, `SELECT hex(vec) FROM vectors WHERE hash='retained'`).Scan(&vector))
	require.Equal(t, "0000803F", vector)
}

func exec(t *testing.T, db *database.RoutedDB, statement string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), statement)
	require.NoError(t, err)
}

func legacyDatabase(t *testing.T, cfg storage.SQLiteConfig) *database.RoutedDB {
	t.Helper()
	dsn, err := storage.SQLiteDSN(cfg)
	require.NoError(t, err)
	db, err := open(context.Background(), dsn)
	require.NoError(t, err)
	return db
}
