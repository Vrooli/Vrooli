package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"test-genie/internal/dbexec"
	"test-genie/internal/persistence"

	repocontract "github.com/vrooli/repo-contract-go"
)

const initializationDialectDir = "sqlite"

// ApplySchema remains the runtime-facing entry point; persistence owns the
// schema authority so offline cutover uses exactly the startup contract.
func ApplySchema(db dbexec.Executor, includeSeed bool) error {
	return persistence.ApplySchema(db, includeSeed)
}

func ensureDatabaseSchema(db dbexec.Executor) error {
	return persistence.ApplySchema(db, true)
}

// These runtime path helpers also resolve configuration-relative files. Schema
// construction itself is intentionally delegated to persistence.ApplySchema.
func resolveInitializationFile(name string) (string, error) {
	scenarioDir, err := scenarioRoot()
	if err != nil {
		return "", err
	}
	target := filepath.Join(scenarioDir, "initialization", initializationDialectDir, name)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("initialization file not accessible (%s): %w", target, err)
	}
	return target, nil
}

func scenarioRoot() (string, error) {
	_, currentFile, _, ok := goruntime.Caller(0)
	if ok {
		root, err := repocontract.FindRepoRoot(currentFile)
		if err == nil {
			if path, resolveErr := repocontract.ResolveScenarioPath(root, "test-genie"); resolveErr == nil {
				return path, nil
			}
		}
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("determine test-genie root: %w", err)
	}
	return repocontract.ResolveScenarioPath(root, "test-genie")
}
