package existence

import (
	"fmt"
	"io"
	"os"
)

// ValidateFiles checks that all required files exist in the scenario.
func ValidateFiles(scenarioDir string, files []string, logWriter io.Writer) Result {
	for _, rel := range files {
		abs := resolvePath(scenarioDir, rel)
		if err := ensureFile(abs); err != nil {
			logError(logWriter, "Missing file: %s", rel)
			return FailMisconfiguration(
				err,
				fmt.Sprintf("Create the file '%s'. See docs/phases/structure/README.md for required files and customization options.", rel),
			)
		}
		logStep(logWriter, "  ✓ %s", rel)
	}
	return OKWithCount(len(files))
}

// ensureFile verifies that a path exists and is a file (not a directory).
func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

// FileExists checks whether a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
