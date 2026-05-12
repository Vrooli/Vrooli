package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// DefaultDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// Shared by api/main.go and CLI subcommands that need direct read access
// to the same database file (e.g. `flow-verifier runs list`).
func DefaultDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return FileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return FileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "flow-verifier"},
		storage.ClassData,
		"flow-verifier.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve flow-verifier db path: %w", err)
	}
	return FileDSN(path)
}

// FileDSN wraps a filesystem path in the canonical SQLite DSN used by
// this scenario. Pragmas mirror agent-inbox; keep in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func FileDSN(path string) (string, error) {
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
