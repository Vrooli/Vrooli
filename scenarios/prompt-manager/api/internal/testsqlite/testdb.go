package testsqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func Open(t *testing.T) *database.RoutedDB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prompt-manager.db")
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          buildDSN(path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func buildDSN(path string) string {
	return "file:" + path + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
}
