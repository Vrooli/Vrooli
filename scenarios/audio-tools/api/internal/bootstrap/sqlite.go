package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"audio-tools/internal/modules"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// OpenDB resolves the on-disk SQLite path, opens it via api-core's
// database.Connect, ensures the audio-tools schemas, and returns the
// live handle plus the resolved DSN for diagnostics.
func OpenDB(ctx context.Context, env Env) (*sql.DB, string, error) {
	dsn, err := resolveDSN(env)
	if err != nil {
		return nil, "", err
	}
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, dsn, fmt.Errorf("connect sqlite: %w", err)
	}
	if err := database.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		_ = db.Close()
		return nil, dsn, fmt.Errorf("ensure schemas: %w", err)
	}
	return db, dsn, nil
}

func resolveDSN(env Env) (string, error) {
	if env.SqlitePath != "" {
		return sqliteFileDSN(env.SqlitePath)
	}
	if env.SqliteDB != "" {
		return sqliteFileDSN(env.SqliteDB)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "audio-tools"},
		storage.ClassData,
		"audio-tools.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve audio-tools db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}
