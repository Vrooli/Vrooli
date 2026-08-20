package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git-control-tower/internal/dbschema"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

func ensureAuditSchema(db *sql.DB) error {
	return ensureDatabaseSchema(db)
}

func ensureRepoSchema(db *sql.DB) error {
	if err := ensureDatabaseSchema(db); err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ensurePrecommitHookColumns(ctx, db)
}

func ensureDatabaseSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(dbschema.Schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
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
	scenarioID := "git-control-tower"
	variant := strings.ToLower(strings.TrimSpace(os.Getenv(storage.EnvVariant)))
	if strings.TrimSpace(os.Getenv(storage.EnvScenario)) == scenarioID && variant != "" && variant != "live" {
		namespace, err := storage.ScenarioNamespace(scenarioID)
		if err != nil {
			return "", fmt.Errorf("resolve variant-aware storage namespace: %w", err)
		}
		scenarioID = namespace
	}
	return resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassData, "git-control-tower.db")
}
