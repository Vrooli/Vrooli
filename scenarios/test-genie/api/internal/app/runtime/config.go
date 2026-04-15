package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apistorage "github.com/vrooli/api-core/storage"

	"test-genie/internal/storage/sqlitedb"
)

// Config captures runtime parameters that should not be hard-coded inside HTTP handlers.
type Config struct {
	Port          string
	DatabasePath  string
	DatabaseDSN   string
	ScenariosRoot string
}

// LoadConfig gathers lifecycle-provided environment variables and resolves derived paths.
func LoadConfig() (*Config, error) {
	sqliteCfg, err := resolveDatabaseConfig()
	if err != nil {
		return nil, err
	}

	scenariosRoot, err := resolveScenariosRoot()
	if err != nil {
		return nil, err
	}

	port, err := requireEnv("API_PORT")
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:          port,
		DatabasePath:  sqliteCfg.Path,
		DatabaseDSN:   sqliteCfg.DSN,
		ScenariosRoot: scenariosRoot,
	}, nil
}

func requireEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required; run the scenario via 'vrooli scenario run <name>' so lifecycle exports it", key)
	}
	return value, nil
}

func resolveDatabaseConfig() (sqlitedb.Config, error) {
	cfg, err := sqlitedb.Resolve()
	if err == nil {
		return cfg, nil
	}

	fallbackPath, fallbackErr := resolveFallbackDatabasePath()
	if fallbackErr == nil {
		fallbackCfg, fallbackResolveErr := sqlitedb.ResolveExplicit(fallbackPath)
		if fallbackResolveErr == nil {
			return fallbackCfg, nil
		}
		return sqlitedb.Config{}, fmt.Errorf("sqlite configuration failed: %w (fallback path %s also failed: %v)", err, fallbackPath, fallbackResolveErr)
	}

	return sqlitedb.Config{}, fmt.Errorf("sqlite configuration failed: %w (storage fallback error: %v)", err, fallbackErr)
}

func resolveScenariosRoot() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("SCENARIOS_ROOT")); raw != "" {
		return filepath.Abs(raw)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	scenarioDir := filepath.Dir(wd)
	root := filepath.Dir(scenarioDir)
	return root, nil
}

func resolveFallbackDatabasePath() (string, error) {
	resolver, err := apistorage.NewResolver(apistorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: apistorage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	dbPath, err := resolver.Path(
		apistorage.Options{ScenarioID: "test-genie"},
		apistorage.ClassData,
		"test-genie.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve storage-backed database path: %w", err)
	}
	if err := migrateLegacyDatabase(dbPath); err != nil {
		return "", err
	}
	return dbPath, nil
}

func migrateLegacyDatabase(dst string) error {
	root, err := scenarioRoot()
	if err != nil {
		return nil
	}
	legacy := filepath.Join(root, "data", "test-genie.db")
	if _, err := os.Stat(legacy); err != nil {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("ensure fallback database dir: %w", err)
	}
	if err := os.Rename(legacy, dst); err != nil {
		return fmt.Errorf("migrate legacy test-genie database: %w", err)
	}
	return nil
}
