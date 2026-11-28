package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SnapshotDatabase writes a consistent copy of the SQLite database at dbPath to
// dest. SQLite's VACUUM INTO reads through one transaction, so a live writer
// cannot tear the copy; copying the file with the OS would.
//
// dest must not exist. The parent directory must already exist.
func SnapshotDatabase(ctx context.Context, dbPath, dest string) error {
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("snapshot source %s: %w", dbPath, err)
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("snapshot destination %s already exists", dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("snapshot destination %s: %w", dest, err)
	}
	if _, err := os.Stat(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("snapshot destination directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open %s for snapshot: %w", dbPath, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, dest); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("snapshot %s: %w", dbPath, err)
	}
	return nil
}
