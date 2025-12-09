package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) migrationsDir() string {
	return filepath.Join(s.Config.DatabasePath, "migrations")
}

func (s *Service) InitMigrations(name string) error {
	path, err := s.databasePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("database not found: %s", name)
	}
	ctx := context.Background()
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  checksum TEXT,
  description TEXT
);
CREATE INDEX IF NOT EXISTS idx_migrations_applied_at ON schema_migrations(applied_at);
`)
	return err
}

func (s *Service) CreateMigration(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("migration name required")
	}
	dir := s.migrationsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	version := time.Now().Format("20060102150405")
	safeName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	file := filepath.Join(dir, fmt.Sprintf("%s_%s.sql", version, safeName))
	content := fmt.Sprintf(`-- Migration: %s_%s
-- Created: %s
-- Description: %s

BEGIN TRANSACTION;
-- up migration here
COMMIT;

-- Down Migration (manual)
-- BEGIN TRANSACTION;
-- ROLLBACK;
`, version, safeName, time.Now().Format(time.RFC3339), name)
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		return "", err
	}
	return file, nil
}

func (s *Service) ApplyMigrations(ctx context.Context, dbName string, targetVersion string) (int, error) {
	dbPath, err := s.databasePath(dbName)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return 0, fmt.Errorf("database not found: %s", dbName)
	}
	if err := s.InitMigrations(dbName); err != nil {
		return 0, err
	}
	db, err := s.openDatabase(ctx, dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	applied, err := s.appliedVersions(ctx, db)
	if err != nil {
		return 0, err
	}
	dir := s.migrationsDir()
	files, _ := os.ReadDir(dir)
	count := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		filename := f.Name()
		version := strings.SplitN(filename, "_", 2)[0]
		if applied[version] {
			continue
		}
		if targetVersion != "" && version > targetVersion {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			return count, err
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return count, fmt.Errorf("failed applying %s: %w", filename, err)
		}
		count++
		_, _ = db.ExecContext(ctx, "INSERT INTO schema_migrations(version, description) VALUES(?, ?)", version, filename)
	}
	return count, nil
}

func (s *Service) MigrationStatus(ctx context.Context, dbName string) (applied []string, pending []string, err error) {
	dbPath, err := s.databasePath(dbName)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, nil, fmt.Errorf("database not found: %s", dbName)
	}
	db, err := s.openDatabase(ctx, dbPath)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	appliedMap, err := s.appliedVersions(ctx, db)
	if err != nil {
		return nil, nil, err
	}
	for v := range appliedMap {
		applied = append(applied, v)
	}
	dir := s.migrationsDir()
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		version := strings.SplitN(f.Name(), "_", 2)[0]
		if !appliedMap[version] {
			pending = append(pending, f.Name())
		}
	}
	return applied, pending, nil
}

func (s *Service) appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return map[string]bool{}, nil // not initialized yet
	}
	defer rows.Close()
	result := map[string]bool{}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		result[v] = true
	}
	return result, nil
}
