package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/storage/sqlitedb"

	repocontract "github.com/vrooli/repo-contract-go"
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

// resolveDatabaseConfig returns the location of Test Genie's own run ledger.
//
// There is deliberately no fallback here. Resolution used to try a chain of
// environment variables, then a second hand-rolled storage-resolver path, then
// a routine that MOVED an existing database to wherever that second path
// landed. Every extra branch was another way for the ledger to end up somewhere
// unexpected — and the move could relocate a live database out from under a
// running process. sqlitedb.Resolve is deterministic from this scenario's own
// identity, so one call is the whole answer.
func resolveDatabaseConfig() (sqlitedb.Config, error) {
	return sqlitedb.Resolve()
}

func resolveScenariosRoot() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("SCENARIOS_ROOT")); raw != "" {
		return filepath.Abs(raw)
	}

	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}

	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", fmt.Errorf("load repo contract: %w", err)
	}

	scenariosRoot, err := contract.TopLevelDir(root, "scenarios")
	if err != nil {
		return "", fmt.Errorf("resolve scenarios dir: %w", err)
	}
	return scenariosRoot, nil
}
