package content

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SkipDirs defines directories to skip during JSON validation.
var SkipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	"artifacts":    {},
}

// jsonValidator is the default implementation of JSONValidator.
type jsonValidator struct {
	scenarioDir string
	logWriter   io.Writer
}

// NewJSONValidator creates a new JSON syntax validator.
func NewJSONValidator(scenarioDir string, logWriter io.Writer) JSONValidator {
	return &jsonValidator{
		scenarioDir: scenarioDir,
		logWriter:   logWriter,
	}
}

// Validate implements JSONValidator.
func (v *jsonValidator) Validate() Result {
	return ValidateJSON(v.scenarioDir, v.logWriter)
}

// ValidateJSON scans the scenario directory for JSON files and validates their syntax.
func ValidateJSON(scenarioDir string, logWriter io.Writer) Result {
	count, invalidFiles, err := scanJSONFiles(scenarioDir)
	if err != nil {
		logError(logWriter, "JSON scan failed: %v", err)
		return FailSystem(
			err,
			"Ensure JSON files under the scenario tree are readable.",
		)
	}

	if len(invalidFiles) > 0 {
		logError(logWriter, "Invalid JSON files found: %s", strings.Join(invalidFiles, ", "))
		return FailMisconfiguration(
			fmt.Errorf("invalid JSON detected: %s", strings.Join(invalidFiles, ", ")),
			"Fix the malformed JSON files listed in the error message.",
		)
	}

	logSuccess(logWriter, "All JSON files valid (%d)", count)
	return OKWithCount(count)
}

// scanJSONFiles walks the scenario directory and validates all JSON files.
// Returns the count of valid files, a list of invalid file paths, and any error.
func scanJSONFiles(root string) (int, []string, error) {
	var invalid []string
	count := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if _, skip := SkipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return nil
		}

		count++
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		if !json.Valid(data) {
			if rel, relErr := filepath.Rel(root, path); relErr == nil {
				invalid = append(invalid, filepath.ToSlash(rel))
			} else {
				invalid = append(invalid, path)
			}
		}
		return nil
	})

	return count, invalid, err
}

// JSONValidationResult contains detailed results from JSON validation.
type JSONValidationResult struct {
	// TotalFiles is the number of JSON files found.
	TotalFiles int

	// ValidFiles is the number of syntactically valid JSON files.
	ValidFiles int

	// InvalidFiles lists paths to files with invalid JSON.
	InvalidFiles []string
}

// ValidateJSONDetailed performs JSON validation and returns detailed results.
func ValidateJSONDetailed(scenarioDir string) (*JSONValidationResult, error) {
	count, invalid, err := scanJSONFiles(scenarioDir)
	if err != nil {
		return nil, err
	}

	return &JSONValidationResult{
		TotalFiles:   count,
		ValidFiles:   count - len(invalid),
		InvalidFiles: invalid,
	}, nil
}
