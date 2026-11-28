package ledgerdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention/sqlitemaintenance"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ConvertFilesIndex enables incremental auto-vacuum on the existing live cache.
// The lifecycle lock spans the stopped observation and the database close.
// SQLite still arbitrates access by clients outside the lifecycle.
func ConvertFilesIndex(ctx context.Context) (sqlitemaintenance.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return sqlitemaintenance.Result{}, err
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return sqlitemaintenance.Result{}, err
	}
	cfg := FilesIndexConfig()
	cfg.EnvGet = func(key string) string {
		switch key {
		case storage.EnvStorageNamespace:
			return Scenario
		case storage.EnvVariant:
			return "live"
		default:
			return os.Getenv(key)
		}
	}
	lock := filepath.Join(state, "locks", "scenario-"+Scenario+".lock")
	return convertFilesIndex(ctx, cfg, lock, cliutil.DetectScenarioRuntimeStatus(Scenario+"@live"))
}

func convertFilesIndex(ctx context.Context, cfg storage.SQLiteConfig, lockPath string, status func() string) (result sqlitemaintenance.Result, err error) {
	lock, err := platform.OpenLockFile(lockPath, 0o644)
	if err != nil {
		return result, fmt.Errorf("open Source Ledger lifecycle lock: %w", err)
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, true)
	if err != nil {
		return result, fmt.Errorf("acquire Source Ledger lifecycle lock: %w", err)
	}
	defer release()
	if err := platform.VerifyLockFile(lock, lockPath); err != nil {
		return result, err
	}
	if seen := status(); seen != "stopped" {
		return result, fmt.Errorf("Source Ledger must be stopped for file-index conversion (observed state %q)", seen)
	}

	path, err := storage.SQLitePath(cfg)
	if err != nil {
		return result, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("file index path %q is not a regular file", path)
	}
	probe, cancelProbe := context.WithTimeout(ctx, 5*time.Second)
	defer cancelProbe()
	inspection, err := OpenExistingReadOnly(probe, path)
	if err != nil {
		return result, err
	}
	if err := errors.Join(verifyFilesIndex(probe, inspection, path), inspection.Close()); err != nil {
		return result, err
	}
	cfg.Tuning.BusyTimeout = 100 * time.Millisecond
	dsn, err := storage.SQLiteDSNAt(path, cfg.Tuning)
	if err != nil {
		return result, err
	}
	db, err := open(probe, dsn+"&mode=rw")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	if latest, err := os.Lstat(path); err != nil || !os.SameFile(info, latest) {
		return result, fmt.Errorf("file index %q changed during conversion admission", path)
	}
	if err := verifyFilesIndex(probe, db, path); err != nil {
		return result, err
	}

	result, err = sqlitemaintenance.Maintain(ctx, db, sqlitemaintenance.Options{
		AllowConversion: true, ForceConversion: true, ConversionBudget: 20 * time.Minute,
	})
	if err == nil && (result.Deferred != "" || result.After.AutoVacuum != "incremental") {
		err = fmt.Errorf("file-index conversion incomplete: mode %q, deferred reason %q", result.After.AutoVacuum, result.Deferred)
	}
	return result, err
}

func verifyFilesIndex(ctx context.Context, db *database.RoutedDB, path string) error {
	// Require the dedicated FTS cache shape and an explicit INTEGER PRIMARY KEY
	// for the chunk identifiers shared with the FTS rowids. In particular a
	// renamed ledger or a SQLite file with a different shape cannot be rewritten.
	var known bool
	if err := db.QueryRowContext(ctx, `SELECT
		EXISTS (SELECT 1 FROM sqlite_schema WHERE type='table' AND name='chunks_fts' AND instr(lower(sql), 'using fts5(') > 0)
		AND EXISTS (SELECT 1 FROM pragma_table_info('chunks') WHERE name='id' AND upper(type)='INTEGER' AND pk=1)
		AND NOT EXISTS (SELECT 1 FROM pragma_index_list('chunks') WHERE origin='pk')
		AND NOT EXISTS (SELECT 1 FROM sqlite_schema WHERE type='table' AND name='entries')`).Scan(&known); err != nil {
		return fmt.Errorf("verify file-index schema: %w", err)
	}
	if !known {
		return fmt.Errorf("%q does not have the Source Ledger file-index schema", path)
	}
	return nil
}
