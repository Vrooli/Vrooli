package sources_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"data-backup-manager/internal/sources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// TestSqliteSource_VacuumIntoConsistent seeds a SQLite database, optionally
// spawns a concurrent writer (using WAL so VACUUM INTO can proceed), captures
// via VACUUM INTO, restores, and then:
//   - asserts `PRAGMA integrity_check` == "ok"
//   - asserts the originally seeded rows are present in the restored copy.
func TestSqliteSource_VacuumIntoConsistent(t *testing.T) {
	t.Parallel()

	// --- create and seed source db ---
	tmpDir := t.TempDir()
	srcDB := filepath.Join(tmpDir, "source.db")

	db, err := sql.Open("sqlite", srcDB)
	require.NoError(t, err)
	defer db.Close()

	// WAL mode allows VACUUM INTO to proceed even with concurrent writers.
	_, err = db.Exec(`PRAGMA journal_mode=WAL`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	require.NoError(t, err)

	seeded := []struct {
		id   int
		name string
	}{
		{1, "alpha"},
		{2, "beta"},
		{3, "gamma"},
	}
	for _, row := range seeded {
		_, err = db.Exec(`INSERT INTO items (id, name) VALUES (?, ?)`, row.id, row.name)
		require.NoError(t, err)
	}

	// --- concurrent writer: writes rows while Capture runs ---
	// Uses WAL mode so VACUUM INTO can get a consistent read without blocking.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 100
		for {
			select {
			case <-stop:
				return
			default:
				// Best-effort insert; ignore errors (db may be briefly busy).
				_, _ = db.Exec(`INSERT OR IGNORE INTO items (id, name) VALUES (?, ?)`, i, "concurrent")
				i++
			}
		}
	}()

	// --- Capture ---
	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})
	cap, err := reg.Capturer(sources.KindSQLite)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  srcDB,
		StageDir: stageDir,
	})
	close(stop)
	wg.Wait()
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)
	// The artifact is a directory containing snapshot.db (so the kopia snapshot
	// root is a directory, restorable into a directory target).
	artInfo, statErr := os.Stat(art.Path)
	require.NoError(t, statErr)
	assert.True(t, artInfo.IsDir(), "sqlite artifact must be a directory")
	assert.FileExists(t, filepath.Join(art.Path, "snapshot.db"))
	assert.Positive(t, art.Bytes, "sqlite artifact must report copied bytes")

	// --- Restore ---
	restorePath := filepath.Join(t.TempDir(), "restored.db")
	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      srcDB,
		ArtifactPath: art.Path,
		Target:       restorePath,
	})
	require.NoError(t, err)

	// --- Open restored db and verify ---
	rdb, err := sql.Open("sqlite", restorePath)
	require.NoError(t, err)
	defer rdb.Close()

	// integrity_check must return exactly "ok"
	var ic string
	require.NoError(t, rdb.QueryRow(`PRAGMA integrity_check`).Scan(&ic))
	assert.Equal(t, "ok", ic, "restored database must pass integrity_check")

	// All originally seeded rows must still be present.
	for _, row := range seeded {
		var name string
		err = rdb.QueryRow(`SELECT name FROM items WHERE id = ?`, row.id).Scan(&name)
		assert.NoError(t, err, "seeded row id=%d must exist", row.id)
		assert.Equal(t, row.name, name, "seeded row id=%d name must match", row.id)
	}

	// The artifact's snapshot.db must exist on disk.
	_, dbStatErr := os.Stat(filepath.Join(art.Path, "snapshot.db"))
	assert.NoError(t, dbStatErr, "artifact snapshot.db must exist on disk")
}
