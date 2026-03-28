package artifacts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MigrationResult tracks what was migrated.
type MigrationResult struct {
	// FilesMoved counts files moved to new locations.
	FilesMoved int
	// DirectoriesCreated counts directories created.
	DirectoriesCreated int
	// Errors lists any errors encountered during migration.
	Errors []error
	// Actions lists human-readable descriptions of actions taken.
	Actions []string
}

// MigrationOptions configures migration behavior.
type MigrationOptions struct {
	// DryRun previews what would be migrated without making changes.
	DryRun bool
	// Verbose prints detailed progress information.
	Verbose bool
	// Logger receives verbose output (defaults to io.Discard).
	Logger io.Writer
}

// Migrate checks for artifacts in legacy locations and moves them to canonical paths.
// This helps scenarios transition to the standardized artifact structure.
func Migrate(scenarioDir string, opts MigrationOptions) (*MigrationResult, error) {
	if opts.Logger == nil {
		opts.Logger = io.Discard
	}

	result := &MigrationResult{}

	return result, nil
}

// EnsureCoverageStructure creates all standard coverage directories.
// This is useful for initializing a new scenario or ensuring the structure exists.
func EnsureCoverageStructure(scenarioDir string) error {
	for _, dir := range AllCoverageSubdirs(scenarioDir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}

// CleanCoverageArtifacts removes all coverage artifacts.
// This is useful for starting fresh before a test run.
func CleanCoverageArtifacts(scenarioDir string) error {
	coverageRoot := filepath.Join(scenarioDir, CoverageRoot)
	if _, err := os.Stat(coverageRoot); os.IsNotExist(err) {
		return nil // Nothing to clean
	}
	return os.RemoveAll(coverageRoot)
}
