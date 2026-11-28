// Responsibility: the one-time background repair of rows derived from runs
// that no longer exist. Until 2026-10-29 deleting a run left its read-model
// projection and transcript registration behind, so run statistics kept
// counting deleted runs. It scans large tables, so it runs after startup.
package database

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/sirupsen/logrus"
)

const migrationOrphanRunProjections = "2026-10-29-drop-orphan-run-projections"

// orphanRunIDsSQL lists every run id that still has derived rows but no run.
func orphanRunIDsSQL() string {
	var parts []string
	for _, table := range append(append([]string(nil), runProjectionTables...), "transcript_files") {
		parts = append(parts, `SELECT run_id FROM `+table+` x WHERE run_id IS NOT NULL AND run_id <> '' AND NOT EXISTS (SELECT 1 FROM runs r WHERE r.id = x.run_id)`)
	}
	return strings.Join(parts, " UNION ")
}

// RepairOrphanRunProjections removes the derived rows of deleted runs, one run
// per short write transaction, then writes the marker. A run that stays locked
// is retried on the next pass, as the duplicate-event repair does.
func (db *DB) RepairOrphanRunProjections(ctx context.Context) error {
	for {
		done, err := db.duplicateRepairDone(ctx, migrationOrphanRunProjections)
		if err != nil || done {
			return err
		}
		var runIDs []string
		if err := db.retryBusy(ctx, func() error { return db.SelectContext(ctx, &runIDs, orphanRunIDsSQL()) }); err != nil {
			return &domain.DatabaseError{Operation: "orphan_projections_select", EntityType: "Run", Cause: err}
		}
		started, failed := time.Now(), 0
		for _, runID := range runIDs {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := db.retryBusy(ctx, func() error { return db.deleteOrphanRun(ctx, runID) }); err != nil {
				failed++
				if db.log != nil {
					db.log.WithError(err).WithField("run_id", runID).Warn("orphan run projection repair skipped a run")
				}
			}
		}
		if db.log != nil {
			db.log.WithFields(logrus.Fields{"migration": migrationOrphanRunProjections, "runs": len(runIDs), "runs_failed": failed, "elapsed": time.Since(started).String()}).Info("orphan run projection pass")
		}
		if failed == 0 {
			return db.retryBusy(ctx, func() error {
				_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations_done (name, completed_at) VALUES (?, ?)`, migrationOrphanRunProjections, time.Now().UTC().Format(time.RFC3339Nano))
				return err
			})
		}
		select {
		case <-time.After(duplicateRepairPassInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// deleteOrphanRun removes one deleted run's derived rows, re-checking inside
// the write transaction that the run is still absent.
func (db *DB) deleteOrphanRun(ctx context.Context, runID string) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var present int
	if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs WHERE id = ?`, runID).Scan(&present); err != nil {
		return err
	}
	if present == 0 {
		if err = deleteRunDerivedRowsTx(ctx, conn, runID); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}
