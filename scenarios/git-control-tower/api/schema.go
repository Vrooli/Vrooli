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

CREATE TABLE IF NOT EXISTS git_repo_precommit (
    repo_path TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    command TEXT NOT NULL DEFAULT '',
    working_directory TEXT NOT NULL DEFAULT '',
    timeout_seconds INTEGER NOT NULL DEFAULT 300,
    run_before_commit INTEGER NOT NULL DEFAULT 1,
    allow_override INTEGER NOT NULL DEFAULT 1,
    last_status TEXT,
    last_exit_code INTEGER,
    last_summary TEXT,
    last_stdout TEXT,
    last_stderr TEXT,
    last_duration_ms INTEGER,
    last_timestamp TEXT,
    hook_install_status TEXT,
    hook_install_reason TEXT,
    hook_existing_kind TEXT,
    hook_installed_at TEXT,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS git_commit_check_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_path TEXT NOT NULL,
    commit_hash TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    command TEXT NOT NULL DEFAULT '',
    exit_code INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    stdout TEXT,
    stderr TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_commit_check_runs_repo_hash ON git_commit_check_runs(repo_path, commit_hash);
CREATE INDEX IF NOT EXISTS idx_commit_check_runs_repo_created ON git_commit_check_runs(repo_path, created_at DESC);
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

	if err := ensurePrecommitHookColumns(ctx, db); err != nil {
		return err
	}

	return nil
}

func ensurePrecommitHookColumns(ctx context.Context, db *sql.DB) error {
	cols, err := tableColumns(ctx, db, "git_repo_precommit")
	if err != nil {
		return err
	}
	additions := []struct {
		name string
		ddl  string
	}{
		{"hook_install_status", "ALTER TABLE git_repo_precommit ADD COLUMN hook_install_status TEXT"},
		{"hook_install_reason", "ALTER TABLE git_repo_precommit ADD COLUMN hook_install_reason TEXT"},
		{"hook_existing_kind", "ALTER TABLE git_repo_precommit ADD COLUMN hook_existing_kind TEXT"},
		{"hook_installed_at", "ALTER TABLE git_repo_precommit ADD COLUMN hook_installed_at TEXT"},
	}
	for _, addition := range additions {
		if cols[addition.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, addition.ddl); err != nil {
			return fmt.Errorf("add column %s: %w", addition.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info(%s): %w", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info(%s): %w", table, err)
	}
	return cols, nil
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

	path, err := scenarioDBPath()
	if err != nil {
		return "", fmt.Errorf("resolve git-control-tower db path: %w", err)
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
