package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

// Keep this in sync with initialization/sqlite/schema.sql.
const auditSchemaSQL = `
CREATE TABLE IF NOT EXISTS git_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,
    repo_dir TEXT NOT NULL,
    branch TEXT,
    paths TEXT,
    commit_hash TEXT,
    commit_message TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON git_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON git_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_branch ON git_audit_log(branch);
CREATE INDEX IF NOT EXISTS idx_audit_log_op_created ON git_audit_log(operation, created_at DESC);
`

const repoSchemaSQL = `
CREATE TABLE IF NOT EXISTS git_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    remote_url TEXT,
    added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_opened_at TEXT,
    is_favorite INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_git_repos_last_opened ON git_repos(last_opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_git_repos_added_at ON git_repos(added_at DESC);

CREATE TABLE IF NOT EXISTS git_repo_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func ensureAuditSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, statement := range strings.Split(auditSchemaSQL, ";") {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}

func ensureRepoSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, statement := range strings.Split(repoSchemaSQL, ";") {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("GCT_SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(dsn, "file:") {
		return dsn, nil
	}

	dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
	if dataRoot == "" {
		dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
	}
	if dataRoot == "" {
		if path, err := scenarioDBPath(); err == nil {
			if migrateErr := migrateLegacySQLite(path); migrateErr != nil {
				return "", migrateErr
			}
			return sqliteFileDSN(path)
		}
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
	}

	return sqliteFileDSN(filepath.Join(dataRoot, "git-control-tower.db"))
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		path,
	), nil
}

func scenarioDBPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "git-control-tower"}, storage.ClassData, "git-control-tower.db")
}

func migrateLegacySQLite(dst string) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	src := filepath.Join(home, ".vrooli", "data", "sqlite", "databases", "git-control-tower.db")
	if src == dst {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}
