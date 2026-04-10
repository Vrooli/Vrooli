package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Lifecycle should normally provide SCENARIO_DATA_DIR, but some execution
	// paths only guarantee the scenario working directory. Default to
	// <scenario>/data/test-genie.db so embedded storage still works portably.
	root, rootErr := scenarioRoot()
	if rootErr == nil {
		fallbackPath := filepath.Join(root, "data", "test-genie.db")
		fallbackCfg, fallbackResolveErr := sqlitedb.ResolveExplicit(fallbackPath)
		if fallbackResolveErr == nil {
			return fallbackCfg, nil
		}
		return sqlitedb.Config{}, fmt.Errorf("sqlite configuration failed: %w (fallback path %s also failed: %v)", err, fallbackPath, fallbackResolveErr)
	}

	return sqlitedb.Config{}, fmt.Errorf("sqlite configuration failed: %w", err)
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
