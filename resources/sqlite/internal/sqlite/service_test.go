package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/resources/sqlite/internal/config"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "test"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	dbPath, _ := svc.databasePath("test")

	if svc.IsEncrypted(dbPath) {
		t.Fatalf("fresh database incorrectly reported as encrypted")
	}

	if err := svc.EncryptDatabase("test", "password"); err != nil {
		t.Fatalf("EncryptDatabase: %v", err)
	}
	if !svc.IsEncrypted(dbPath) {
		t.Fatalf("encrypted database not detected")
	}

	if err := svc.DecryptDatabase("test", "password"); err != nil {
		t.Fatalf("DecryptDatabase: %v", err)
	}
	if svc.IsEncrypted(dbPath) {
		t.Fatalf("database still marked encrypted after decrypt")
	}

	if _, err := svc.Execute(ctx, "test", "PRAGMA integrity_check;"); err != nil {
		t.Fatalf("post-decrypt integrity check failed: %v", err)
	}
}

func TestIsEncryptedDetectionForPrefixedFile(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	path := filepath.Join(cfg.DatabasePath, "custom.db")
	if err := os.MkdirAll(cfg.DatabasePath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(encMagic+"12345"), cfg.FilePermissions); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !svc.IsEncrypted(path) {
		t.Fatalf("encMagic-prefixed file should be treated as encrypted")
	}
}

func TestContentLifecycleAndBackupRestore(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "app"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "app", "CREATE TABLE items(id INTEGER PRIMARY KEY, name TEXT);"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := svc.Execute(ctx, "app", "INSERT INTO items(name) VALUES ('one'), ('two');"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	dbs, err := svc.ListDatabases()
	if err != nil {
		t.Fatalf("ListDatabases: %v", err)
	}
	if len(dbs) != 1 {
		t.Fatalf("expected 1 database, got %d", len(dbs))
	}

	backup, err := svc.BackupDatabase(ctx, "app")
	if err != nil {
		t.Fatalf("BackupDatabase: %v", err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	restorePath, err := svc.RestoreDatabase(ctx, "restored", backup, true)
	if err != nil {
		t.Fatalf("RestoreDatabase: %v", err)
	}
	if _, err := os.Stat(restorePath); err != nil {
		t.Fatalf("restored db missing: %v", err)
	}
	if _, err := svc.Execute(ctx, "restored", "SELECT COUNT(*) FROM items;"); err != nil {
		t.Fatalf("restored db query failed: %v", err)
	}
}

func TestCSVImportExportAndBatch(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "csv"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "csv", "CREATE TABLE entries(id INTEGER, name TEXT);"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	csvPath := filepath.Join(cfg.DataRoot, "sample.csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,alpha\n2,beta\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	count, err := svc.ImportCSV(ctx, "csv", "entries", csvPath, true, nil)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows imported, got %d", count)
	}

	exportPath := filepath.Join(cfg.DataRoot, "out.csv")
	exportCount, err := svc.ExportCSV(ctx, "csv", "entries", exportPath)
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if exportCount != 2 {
		t.Fatalf("expected 2 rows exported, got %d", exportCount)
	}
	data, _ := os.ReadFile(exportPath)
	if !strings.Contains(string(data), "alpha") {
		t.Fatalf("export content unexpected: %s", string(data))
	}

	sqlBatch := filepath.Join(cfg.DataRoot, "batch.sql")
	if err := os.WriteFile(sqlBatch, []byte("INSERT INTO entries(id, name) VALUES (3, 'gamma');"), 0o644); err != nil {
		t.Fatalf("write batch: %v", err)
	}
	if _, err := svc.Batch(ctx, "csv", mustOpen(t, sqlBatch)); err != nil {
		t.Fatalf("Batch: %v", err)
	}
	res, err := svc.Execute(ctx, "csv", "SELECT COUNT(*) FROM entries;")
	if err != nil {
		t.Fatalf("post-batch query: %v", err)
	}
	if res.Rows[0][0] != "3" {
		t.Fatalf("expected 3 rows after batch, got %s", res.Rows[0][0])
	}
}

func TestReplicationAndVerify(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "rep"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "rep", "CREATE TABLE items(id INTEGER PRIMARY KEY, name TEXT);"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := svc.Execute(ctx, "rep", "INSERT INTO items(name) VALUES ('item');"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	target := filepath.Join(cfg.ReplicaPath, "replica.db")
	if err := svc.AddReplica("rep", target, time.Second); err != nil {
		t.Fatalf("AddReplica: %v", err)
	}

	ok, failed, err := svc.SyncReplica(ctx, "rep", true)
	if err != nil {
		t.Fatalf("SyncReplica: %v", err)
	}
	if ok != 1 || failed != 0 {
		t.Fatalf("unexpected sync counts ok=%d failed=%d", ok, failed)
	}
	issues, err := svc.VerifyReplicas(ctx, "rep")
	if err != nil {
		t.Fatalf("VerifyReplicas: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestMigrationsApplyAndStatus(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "mig"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	path, err := svc.CreateMigration("create_things")
	if err != nil {
		t.Fatalf("CreateMigration: %v", err)
	}
	if err := os.WriteFile(path, []byte("BEGIN; CREATE TABLE things(id INTEGER PRIMARY KEY); COMMIT;"), 0o644); err != nil {
		t.Fatalf("overwrite migration: %v", err)
	}

	applied, err := svc.ApplyMigrations(ctx, "mig", "")
	if err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 migration applied, got %d", applied)
	}

	appliedVersions, pending, err := svc.MigrationStatus(ctx, "mig")
	if err != nil {
		t.Fatalf("MigrationStatus: %v", err)
	}
	if len(appliedVersions) == 0 {
		t.Fatalf("expected applied versions, got none")
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending migrations, got %v", pending)
	}
}

func TestBackupRetentionPrunesOldBackups(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRetentionDays = 1
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "keep"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if err := os.MkdirAll(cfg.BackupPath, 0o755); err != nil {
		t.Fatalf("mkdir backup: %v", err)
	}
	oldPath := filepath.Join(cfg.BackupPath, "keep_20200101_000000.db")
	if err := os.WriteFile(oldPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("write old backup: %v", err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	if _, err := svc.BackupDatabase(ctx, "keep"); err != nil {
		t.Fatalf("BackupDatabase: %v", err)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old backup pruned, stat err=%v", err)
	}
}

func TestEncryptDecryptRemovesJournals(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "secure"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	path, _ := svc.databasePath("secure")
	wal := path + "-wal"
	shm := path + "-shm"
	if err := os.WriteFile(wal, []byte("wal"), 0o600); err != nil {
		t.Fatalf("write wal: %v", err)
	}
	if err := os.WriteFile(shm, []byte("shm"), 0o600); err != nil {
		t.Fatalf("write shm: %v", err)
	}

	if err := svc.EncryptDatabase("secure", "pass"); err != nil {
		t.Fatalf("EncryptDatabase: %v", err)
	}
	if _, err := os.Stat(wal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wal file should be removed after encrypt")
	}
	if _, err := os.Stat(shm); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shm file should be removed after encrypt")
	}

	if err := svc.DecryptDatabase("secure", "pass"); err != nil {
		t.Fatalf("DecryptDatabase: %v", err)
	}
	if _, err := os.Stat(wal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wal file should be removed after decrypt")
	}
	if _, err := os.Stat(shm); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shm file should be removed after decrypt")
	}
}

func TestImportCSVHeaderMappingAndValidation(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "hdr"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "hdr", "CREATE TABLE people(id INTEGER, name TEXT, email TEXT);"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	csvPath := filepath.Join(cfg.DataRoot, "header.csv")
	content := "name,email,id\nAlice,alice@example.com,1\nBob,bob@example.com,2\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if _, err := svc.ImportCSV(ctx, "hdr", "people", csvPath, true, nil); err != nil {
		t.Fatalf("ImportCSV header mapping: %v", err)
	}
	res, err := svc.Execute(ctx, "hdr", "SELECT id, name, email FROM people ORDER BY id;")
	if err != nil {
		t.Fatalf("query inserted rows: %v", err)
	}
	if len(res.Rows) != 2 || res.Rows[0][1] != "Alice" || res.Rows[1][2] != "bob@example.com" {
		t.Fatalf("unexpected rows: %+v", res.Rows)
	}

	// Row count validation
	badPath := filepath.Join(cfg.DataRoot, "bad.csv")
	if err := os.WriteFile(badPath, []byte("name,email,id\nOnlyTwo,missing\n"), 0o644); err != nil {
		t.Fatalf("write bad csv: %v", err)
	}
	if _, err := svc.ImportCSV(ctx, "hdr", "people", badPath, true, nil); err == nil {
		t.Fatalf("expected error for mismatched column count")
	}
}

func TestSyncDueReplicasUsesIntervals(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "primary"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "primary", "CREATE TABLE items(id INTEGER); INSERT INTO items(id) VALUES (1);"); err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	target := filepath.Join(cfg.ReplicaPath, "primary_copy.db")
	if err := svc.AddReplica("primary", target, time.Hour); err != nil {
		t.Fatalf("AddReplica: %v", err)
	}

	ok, fail, err := svc.SyncDueReplicas(ctx, false)
	if err != nil {
		t.Fatalf("SyncDueReplicas first run: %v", err)
	}
	if ok != 1 || fail != 0 {
		t.Fatalf("unexpected counts ok=%d fail=%d", ok, fail)
	}
	info, _ := os.Stat(target)
	if info == nil {
		t.Fatalf("replica file not created")
	}

	ok, fail, err = svc.SyncDueReplicas(ctx, false)
	if err != nil {
		t.Fatalf("SyncDueReplicas second run: %v", err)
	}
	if ok != 0 || fail != 0 {
		t.Fatalf("expected no-op when interval not elapsed, got ok=%d fail=%d", ok, fail)
	}
}

func TestShowStatsGracefulWhenDbstatMissing(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "stats"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "stats", "CREATE TABLE t(id INTEGER);"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err := svc.ShowStats(ctx, "stats")
	if err != nil && !errors.Is(err, ErrDbstatUnavailable) {
		t.Fatalf("unexpected stats error: %v", err)
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	return config.Config{
		DataRoot:            root,
		DatabasePath:        filepath.Join(root, "databases"),
		BackupPath:          filepath.Join(root, "backups"),
		ReplicaPath:         filepath.Join(root, "replicas"),
		MigrationPath:       filepath.Join(root, "migrations"),
		JournalMode:         "WAL",
		BusyTimeout:         10 * time.Second,
		CacheSize:           2000,
		PageSize:            4096,
		Synchronous:         "NORMAL",
		TempStore:           "MEMORY",
		MmapSize:            268435456,
		FilePermissions:     0o600,
		CLITimeout:          5 * time.Second,
		BackupRetentionDays: 7,
	}
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
