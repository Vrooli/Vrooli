package sources

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

// sqliteCapturer captures a SQLite database file by opening it read-only and
// running VACUUM INTO, which produces a consistent point-in-time copy even
// when concurrent writers are active. No external resource CLI is needed.
type sqliteCapturer struct{}

// Compile-time guarantee.
var _ Capturer = (*sqliteCapturer)(nil)

func newSQLiteCapturer() *sqliteCapturer { return &sqliteCapturer{} }

func (c *sqliteCapturer) Kind() SourceKind { return KindSQLite }

// Capture opens the SQLite database at spec.Locator and runs
// `VACUUM INTO '<StageDir>/snapshot.db'`. The VACUUM INTO statement writes a
// consistent, fully compacted copy of the database — it is safe to run
// against a live database with concurrent writers.
func (c *sqliteCapturer) Capture(_ context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, "snapshot.db")

	// Open with the modernc sqlite driver (driver name "sqlite").
	db, err := sql.Open("sqlite", spec.Locator)
	if err != nil {
		return Artifact{}, fmt.Errorf("sqlite capture: open %q: %w", spec.Locator, err)
	}
	defer db.Close()

	// VACUUM INTO writes a consistent, compacted copy at the given path.
	// The destination must not exist (SQLite refuses to overwrite).
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dst)); err != nil {
		return Artifact{}, fmt.Errorf("sqlite capture: VACUUM INTO %q: %w", dst, err)
	}

	return Artifact{Path: dst}, nil
}

// Restore copies the snapshot file at spec.ArtifactPath to spec.Target,
// replacing whatever was there. This is a plain file copy — the caller is
// responsible for stopping writers before invoking Restore.
func (c *sqliteCapturer) Restore(_ context.Context, spec RestoreSpec) error {
	if _, err := copyFile(spec.ArtifactPath, spec.Target); err != nil {
		return fmt.Errorf("sqlite restore: %w", err)
	}
	return nil
}
