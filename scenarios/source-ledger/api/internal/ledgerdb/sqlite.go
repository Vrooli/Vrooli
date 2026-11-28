// Package ledgerdb wires the owner policy for Source Ledger's two SQLite files.
package ledgerdb

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/retention/sqlitemaintenance"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

const Scenario = "source-ledger"

func LedgerConfig() storage.SQLiteConfig {
	return storage.SQLiteConfig{Scenario: Scenario}
}

func FilesIndexConfig() storage.SQLiteConfig {
	return storage.SQLiteConfig{Scenario: Scenario, Filename: "files-index.db", Class: storage.ClassCache}
}

// Open initializes a new file before its domain creates any schema. An existing
// ledger can still serve with its prior auto-vacuum mode.
func Open(ctx context.Context, cfg storage.SQLiteConfig) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSN(cfg)
	if err != nil {
		return nil, err
	}
	var readDSN string
	if cfg.Filename == "" {
		path, err := storage.SQLitePath(cfg)
		if err != nil {
			return nil, err
		}
		readDSN, err = storage.SQLiteReadPoolDSNAt(path, cfg.Tuning)
		if err != nil {
			return nil, err
		}
	}
	db, err := openWithReadPool(ctx, dsn, readDSN)
	if err != nil {
		return nil, err
	}
	if err := sqlitemaintenance.Initialize(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func open(ctx context.Context, dsn string) (*database.RoutedDB, error) {
	return openWithReadPool(ctx, dsn, "")
}

func openWithReadPool(ctx context.Context, dsn, readDSN string) (*database.RoutedDB, error) {
	return database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: dsn, ReadDSN: readDSN, MaxOpenConns: 1, MaxIdleConns: 1})
}

// OpenExistingReadOnly participates in SQLite's normal locking and reads
// committed WAL frames. The source file must already exist.
func OpenExistingReadOnly(ctx context.Context, path string) (*database.RoutedDB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absolute); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(absolute)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	uri := (&url.URL{Scheme: "file", Path: uriPath, RawQuery: "mode=ro"}).String()
	return open(ctx, uri)
}

// RunMaintenance services the two primary files serially. The worker can
// reclaim already-free pages and checkpoint WAL; domain records belong to
// their own retention policy. Legacy conversion is never part of this loop.
func RunMaintenance(ctx context.Context, clock schedule.Clock, log *slog.Logger, ledger, filesIndex sqlitemaintenance.Handle) {
	runMaintenance(ctx, clock, log, retention.DefaultInterval, sqlitemaintenance.DefaultPolicy(), ledger, filesIndex)
}

func runMaintenance(ctx context.Context, clock schedule.Clock, log *slog.Logger, interval time.Duration, policy sqlitemaintenance.Policy, ledger, filesIndex sqlitemaintenance.Handle) {
	stores := []struct {
		name string
		db   sqlitemaintenance.Handle
	}{{"ledger", ledger}, {"files-index", filesIndex}}
	timer := clock.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C():
			for _, store := range stores {
				if ctx.Err() != nil {
					return
				}
				maintain(ctx, log.With("store", store.name), store.db, policy)
			}
			timer.Reset(interval)
		}
	}
}

func maintain(ctx context.Context, logger *slog.Logger, db sqlitemaintenance.Handle, policy sqlitemaintenance.Policy) {
	result, err := sqlitemaintenance.Maintain(ctx, db, sqlitemaintenance.Options{Policy: policy, Logger: logger})
	if err != nil {
		if ctx.Err() == nil {
			logger.WarnContext(ctx, "SQLite maintenance failed", "error", err)
		}
		return
	}
	level := slog.LevelDebug
	if result.ReturnedPages > 0 || result.ReclaimedBytes > 0 || result.Deferred != "" {
		level = slog.LevelInfo
	}
	logger.Log(ctx, level, "SQLite maintenance cycle", "auto_vacuum", result.After.AutoVacuum,
		"database_bytes_before", result.Before.DatabaseBytes, "database_bytes_after", result.After.DatabaseBytes,
		"wal_bytes_before", result.Before.WALBytes, "wal_bytes_after", result.After.WALBytes,
		"free_pages_before", result.Before.FreePages, "free_pages_after", result.After.FreePages,
		"returned_pages", result.ReturnedPages, "reclaimed_bytes", result.ReclaimedBytes, "deferred", result.Deferred)
}
