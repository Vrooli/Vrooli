package retention

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// autoVacuumIncremental is the value SQLite reports for PRAGMA auto_vacuum when
// incremental mode is active.
const autoVacuumIncremental = 2

// Execer is the narrow database surface the builtin SQLite pruner needs.
// *sql.DB satisfies it.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// EnsureIncrementalAutoVacuum switches a database to incremental auto-vacuum so
// retention can return freed pages to the filesystem without an offline rebuild.
//
// The subtlety this exists for: on an already-created database,
// `PRAGMA auto_vacuum = INCREMENTAL` is recorded but does not take effect until
// a full VACUUM rewrites the file. Setting the pragma alone and assuming it
// worked is how a retention job silently frees nothing — the state a prior
// incident database was in, holding 73 GB of file for 3.26 GB of live payload,
// and the state autoheal.sqlite is in today at auto_vacuum = 0.
//
// The one-time VACUUM is expensive and needs free space for a complete copy of
// the result, so it runs only when the mode is actually wrong and only after
// pruning has reduced what has to be copied. On an already-migrated database
// this is a single pragma read, which makes it safe to call on every startup and
// safe to re-run.
//
// It is deliberately not fatal to the caller's startup. A database that cannot
// be migrated still works; it just keeps its freed pages. Refusing to start over
// a space optimization would be the worse outcome.
func EnsureIncrementalAutoVacuum(ctx context.Context, db Execer, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}

	mode, err := currentAutoVacuum(ctx, db)
	if err != nil {
		return err
	}
	if mode == autoVacuumIncremental {
		return nil
	}

	if _, err := db.ExecContext(ctx, `PRAGMA auto_vacuum = INCREMENTAL`); err != nil {
		return fmt.Errorf("set auto_vacuum incremental: %w", err)
	}

	// The rewrite is what makes the setting real on an existing file.
	log.InfoContext(ctx, "rewriting database to enable incremental auto-vacuum; this is a one-time cost",
		"previous_auto_vacuum", mode)
	if _, err := db.ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("vacuum to apply auto_vacuum change: %w", err)
	}

	applied, err := currentAutoVacuum(ctx, db)
	if err != nil {
		return err
	}
	if applied != autoVacuumIncremental {
		return fmt.Errorf("auto_vacuum is %d after migration, want %d", applied, autoVacuumIncremental)
	}

	log.InfoContext(ctx, "incremental auto-vacuum enabled; retention can now return pages to the filesystem")
	return nil
}

func currentAutoVacuum(ctx context.Context, db Execer) (int, error) {
	var mode int
	if err := db.QueryRowContext(ctx, `PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		return 0, fmt.Errorf("read auto_vacuum: %w", err)
	}
	return mode, nil
}
