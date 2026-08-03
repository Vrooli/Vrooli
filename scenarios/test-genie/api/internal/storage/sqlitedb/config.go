package sqlitedb

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	// PrimaryPathEnvVar is the scenario-specific override for Test Genie's
	// embedded SQLite database.
	PrimaryPathEnvVar = "TEST_GENIE_SQLITE_PATH"

	defaultDatabaseFile = "test-genie.db"
)

// Config captures the resolved SQLite location for runtime consumers.
type Config struct {
	Path string
	DSN  string
}

// Resolve returns the SQLite file path and DSN for Test Genie runtime storage.
func Resolve() (Config, error) {
	for _, key := range []string{PrimaryPathEnvVar, "SQLITE_PATH", "SQLITE_DB"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			expanded, ok := expandConfiguredPath(raw)
			if !ok {
				continue
			}
			return ResolveExplicit(expanded)
		}
	}

	root := defaultDataRoot()
	if root == "" {
		return Config{}, fmt.Errorf("%s, SCENARIO_DATA_DIR, or SQLITE_DATABASE_PATH must be set", PrimaryPathEnvVar)
	}
	return ResolveExplicit(filepath.Join(root, defaultDatabaseFile))
}

func defaultDataRoot() string {
	// storage-manager:allow-no-writer data — the lifecycle supplies this class
	// root through environment variables; the owner does not hard-code a host
	// directory to create.
	if dir := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR")); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH")); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("VROOLI_DATA")); dir != "" {
		return filepath.Join(dir, "sqlite", "databases")
	}
	return ""
}

// ResolveExplicit resolves a sqlite path or DSN supplied directly by a caller.
func ResolveExplicit(raw string) (Config, error) {
	if strings.HasPrefix(raw, "file:") {
		path, err := extractPathFromDSN(raw)
		if err != nil {
			return Config{}, err
		}
		if err := ensureDir(path); err != nil {
			return Config{}, err
		}
		return Config{Path: path, DSN: raw}, nil
	}

	path, err := filepath.Abs(raw)
	if err != nil {
		return Config{}, fmt.Errorf("resolve sqlite path: %w", err)
	}
	if err := ensureDir(path); err != nil {
		return Config{}, err
	}

	return Config{
		Path: path,
		DSN:  BuildDSN(path),
	}, nil
}

// BuildDSN applies the default pragmas used for portable SQLite-backed scenarios.
func BuildDSN(path string) string {
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		path,
	)
}

func extractPathFromDSN(dsn string) (string, error) {
	trimmed := strings.TrimPrefix(dsn, "file:")
	parsed, err := url.Parse("file:" + trimmed)
	if err == nil && parsed.Path != "" {
		path, decodeErr := url.PathUnescape(parsed.Path)
		if decodeErr == nil {
			if abs, absErr := filepath.Abs(path); absErr == nil {
				return abs, nil
			}
			return path, nil
		}
	}

	path := trimmed
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return "", fmt.Errorf("sqlite DSN must include a file path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve sqlite path: %w", err)
	}
	return abs, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare sqlite directory %s: %w", dir, err)
	}
	return nil
}

func expandConfiguredPath(raw string) (string, bool) {
	expanded := raw
	for _, key := range []string{"SCENARIO_DATA_DIR", "SQLITE_DATABASE_PATH", "VROOLI_DATA"} {
		token := "${" + key + "}"
		if !strings.Contains(expanded, token) {
			continue
		}

		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			return "", false
		}
		expanded = strings.ReplaceAll(expanded, token, value)
	}
	return expanded, true
}
