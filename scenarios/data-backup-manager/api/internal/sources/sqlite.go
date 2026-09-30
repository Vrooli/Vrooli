package sources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

// sqliteSnapshotFile is the fixed filename the VACUUM INTO copy is written to
// inside the artifact directory. Keeping it constant lets Restore find the file
// regardless of the source database's name.
const sqliteSnapshotFile = "snapshot.db"

// sqliteCapturer captures a SQLite database file by opening it read-only and
// running VACUUM INTO, which produces a consistent point-in-time copy even
// when concurrent writers are active. No external resource CLI is needed.
type sqliteCapturer struct{}

// Compile-time guarantee.
var _ Capturer = (*sqliteCapturer)(nil)

func newSQLiteCapturer() *sqliteCapturer { return &sqliteCapturer{} }

func (c *sqliteCapturer) Kind() SourceKind { return KindSQLite }

// Capture opens the SQLite database at spec.Locator and runs
// `VACUUM INTO '<StageDir>/sqlite/snapshot.db'`. The VACUUM INTO statement writes
// a consistent, fully compacted copy of the database — it is safe to run against
// a live database with concurrent writers.
//
// The artifact is a DIRECTORY (containing snapshot.db), not the bare file. The
// engine snapshots Artifact.Path with kopia and later restores it into a fresh
// directory; a single-file snapshot root cannot be restored into a directory
// target (kopia fails with "is a directory"), so the file is wrapped in a dir to
// match the filesystem capturer's shape.
func (c *sqliteCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	dir := filepath.Join(spec.StageDir, "sqlite")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return Artifact{}, fmt.Errorf("sqlite capture: mkdir stage: %w", err)
	}
	dst := filepath.Join(dir, sqliteSnapshotFile)

	// Open arbitrary target DBs through api-core/database so this external
	// capture path gets the same retry/backoff behavior as scenario DB opens.
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          spec.Locator,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return Artifact{}, fmt.Errorf("sqlite capture: open %q: %w", spec.Locator, err)
	}
	defer db.Close()

	// VACUUM INTO writes a consistent, compacted copy at the given path.
	// The destination must not exist (SQLite refuses to overwrite).
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dst)); err != nil {
		return Artifact{}, fmt.Errorf("sqlite capture: VACUUM INTO %q: %w", dst, err)
	}

	var bytes int64
	if info, statErr := os.Stat(dst); statErr == nil {
		bytes = info.Size()
	}
	return Artifact{Path: dir, Bytes: bytes}, nil
}

// Restore copies the captured snapshot.db out of the artifact directory at
// spec.ArtifactPath to spec.Target, replacing whatever was there. This is a
// plain file copy — the caller is responsible for stopping writers before
// invoking Restore.
func (c *sqliteCapturer) Restore(_ context.Context, spec RestoreSpec) error {
	src := filepath.Join(spec.ArtifactPath, sqliteSnapshotFile)
	if _, err := copyFile(src, spec.Target); err != nil {
		return fmt.Errorf("sqlite restore: %w", err)
	}
	return nil
}
