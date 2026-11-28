package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention/sqlitemaintenance"
	"github.com/vrooli/api-core/storage"
)

func openConsoleDatabase(ctx context.Context, path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, fmt.Errorf("build SQLite writer DSN: %w", err)
	}
	readDSN, err := storage.SQLiteReadPoolDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, fmt.Errorf("build SQLite reader DSN: %w", err)
	}
	// A single writer owns SQLite changes. WAL lets query-only reads run
	// concurrently, and RoutedDB isolates requests with active test leases.
	db, err := database.Open(ctx, database.Config{
		Driver:       "sqlite",
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		ReadDSN:      readDSN,
	})
	if err != nil {
		return nil, err
	}
	if err := sqlitemaintenance.Initialize(ctx, db.Primary()); err != nil {
		return nil, errors.Join(fmt.Errorf("initialize Web Console SQLite: %w", err), db.Close())
	}
	return db, nil
}

// newConsoleSQLiteMaintenance returns a task owned by the conversation
// retention loop. It records changes in deferral without repeating the same
// diagnostic every time the loop runs.
func newConsoleSQLiteMaintenance(db *sql.DB) func(context.Context) {
	policy := sqlitemaintenance.DefaultPolicy()
	policy.TickBudget = 2 * time.Second
	// A legacy full VACUUM can change the implicit conversation_events
	// rowids used by the external-content FTS index and saved backfill cursor.
	options := sqlitemaintenance.Options{Policy: policy}
	var lastDeferred string
	return func(ctx context.Context) {
		result, err := sqlitemaintenance.Maintain(ctx, db, options)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("sqlite retention: %v", err)
			}
			return
		}
		if result.ReturnedPages > 0 || result.ReclaimedBytes > 0 || (result.Deferred != "" && result.Deferred != lastDeferred) {
			log.Printf("sqlite retention: returned_pages=%d reclaimed_bytes=%d deferred=%q", result.ReturnedPages, result.ReclaimedBytes, result.Deferred)
		}
		lastDeferred = result.Deferred
	}
}
