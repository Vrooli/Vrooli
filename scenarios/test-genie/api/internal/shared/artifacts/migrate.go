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

// migrateDirectory moves files from legacy to canonical location.
func migrateDirectory(legacy, canon, desc string, opts MigrationOptions, result *MigrationResult) error {
	// Check if legacy location exists
	info, err := os.Stat(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to migrate
		}
		return fmt.Errorf("failed to check legacy path %s: %w", legacy, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("legacy path is not a directory: %s", legacy)
	}

	// Check if canonical location already has content
	if _, err := os.Stat(canon); err == nil {
		fmt.Fprintf(opts.Logger, "SKIP: %s - canonical location already exists\n", desc)
		return nil
	}

	// Migrate
	action := fmt.Sprintf("MIGRATE: %s\n  from: %s\n  to:   %s", desc, legacy, canon)
	result.Actions = append(result.Actions, action)
	fmt.Fprintln(opts.Logger, action)

	if opts.DryRun {
		fmt.Fprintln(opts.Logger, "  (dry run - no changes made)")
		return nil
	}

	// Create parent directory of canonical path
	if err := os.MkdirAll(filepath.Dir(canon), 0o755); err != nil {
		return fmt.Errorf("failed to create parent dir: %w", err)
	}
	result.DirectoriesCreated++

	// Move directory
	if err := os.Rename(legacy, canon); err != nil {
		// If rename fails (cross-device), fall back to copy+delete
		if err := copyDir(legacy, canon); err != nil {
			return fmt.Errorf("failed to copy directory: %w", err)
		}
		if err := os.RemoveAll(legacy); err != nil {
			return fmt.Errorf("failed to remove legacy directory: %w", err)
		}
	}

	// Count files
	err = filepath.Walk(canon, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			result.FilesMoved++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to count migrated files: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
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
