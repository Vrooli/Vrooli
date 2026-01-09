package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// --- Environment Variable Parsing Utilities ---
// These provide consistent parsing of environment variables across all config packages.
// Each function documents the decision criteria for parsing.

// ParseBool interprets a string as a boolean value.
// This is the central decision point for boolean environment variable parsing.
//
// Decision criteria:
//   - "true", "1", "yes", "on" (case-insensitive) → true
//   - "false", "0", "no", "off" (case-insensitive) → false
//   - Any other value → defaultVal
func ParseBool(s string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// EnvBool reads a boolean environment variable with a default.
// Returns defaultVal if the variable is not set or cannot be parsed.
func EnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return ParseBool(val, defaultVal)
}

// EnvInt reads an integer environment variable with a default.
// Returns defaultVal if the variable is not set or cannot be parsed.
func EnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// EnvInt64 reads an int64 environment variable with a default.
// Returns defaultVal if the variable is not set or cannot be parsed.
func EnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return i
}

// EnvString reads a string environment variable with a default.
// Returns defaultVal if the variable is not set.
func EnvString(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// ClampInt constrains an integer value to be within [min, max].
// This is the decision point for "is this value within acceptable bounds?"
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampInt64 constrains an int64 value to be within [min, max].
func ClampInt64(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// TestingConfigPath returns the path to the testing.json config file.
func TestingConfigPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, ".vrooli", "testing.json")
}

// LoadTestingConfig loads and parses the testing.json config file.
// If the file doesn't exist, returns nil data with no error.
// If the file exists but is invalid JSON, returns an error.
func LoadTestingConfig(scenarioDir string) ([]byte, error) {
	configPath := TestingConfigPath(scenarioDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	return data, nil
}

// LoadPhaseConfig loads phase-specific configuration from testing.json.
// T should be a struct type that can unmarshal the phase-specific section.
// phaseName is the JSON key for the phase (e.g., "structure", "business").
// defaultConfig is returned if the file doesn't exist or the phase section is missing.
func LoadPhaseConfig[T any](scenarioDir, phaseName string, defaultConfig T) (T, error) {
	data, err := LoadTestingConfig(scenarioDir)
	if err != nil {
		return defaultConfig, err
	}
	if data == nil {
		return defaultConfig, nil
	}

	// Parse into a generic map to extract just the phase section
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return defaultConfig, fmt.Errorf("failed to parse testing.json: %w", err)
	}

	phaseData, exists := doc[phaseName]
	if !exists {
		return defaultConfig, nil
	}

	var config T
	if err := json.Unmarshal(phaseData, &config); err != nil {
		return defaultConfig, fmt.Errorf("failed to parse %s section in testing.json: %w", phaseName, err)
	}

	return config, nil
}

// MergePhaseConfig loads phase config and merges non-zero values into the provided config.
// This is useful when you want to start with defaults and override specific fields.
// phaseName is the JSON key for the phase (e.g., "structure", "business").
// config is a pointer to the config struct to populate.
func MergePhaseConfig[T any](scenarioDir, phaseName string, config *T) error {
	data, err := LoadTestingConfig(scenarioDir)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	// Parse into a generic map to extract just the phase section
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse testing.json: %w", err)
	}

	phaseData, exists := doc[phaseName]
	if !exists {
		return nil
	}

	// Unmarshal directly into the config pointer, which will only override
	// fields that are present in the JSON
	if err := json.Unmarshal(phaseData, config); err != nil {
		return fmt.Errorf("failed to parse %s section in testing.json: %w", phaseName, err)
	}

	return nil
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists at the given path.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir checks that a directory exists.
func EnsureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return fmt.Errorf("failed to stat directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}
	return nil
}

// EnsureFile checks that a file exists.
func EnsureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("failed to stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path exists but is a directory: %s", path)
	}
	return nil
}
