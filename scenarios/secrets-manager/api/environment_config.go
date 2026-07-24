package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"secrets-manager-api/internal/envx"
)

// startupEnvironment is the typed subset of process configuration that affects
// API boot. It rejects misspelled values instead of silently selecting a
// storage or lifecycle mode.
type startupEnvironment struct {
	desktopMode bool
	skipDB      bool
}

func loadStartupEnvironment(reader envx.Reader) (startupEnvironment, error) {
	desktopMode, err := optionalBoolEnvironment(reader, "VROOLI_DESKTOP_MODE")
	if err != nil {
		return startupEnvironment{}, err
	}
	skipDB, err := optionalBoolEnvironment(reader, "SECRETS_MANAGER_SKIP_DB")
	if err != nil {
		return startupEnvironment{}, err
	}
	return startupEnvironment{desktopMode: desktopMode, skipDB: skipDB}, nil
}

func testModeEnabled(reader envx.Reader) (bool, error) {
	return optionalBoolEnvironment(reader, "SECRETS_MANAGER_TEST_MODE")
}

func optionalScenarioDirectory(reader envx.Reader) (string, error) {
	directory := strings.TrimSpace(reader.Getenv("VROOLI_SCENARIO_DIR"))
	if directory == "" {
		return "", nil
	}
	if !filepath.IsAbs(directory) {
		return "", fmt.Errorf("VROOLI_SCENARIO_DIR must be an absolute path")
	}
	return filepath.Clean(directory), nil
}

func optionalBoolEnvironment(reader envx.Reader, key string) (bool, error) {
	value := strings.TrimSpace(reader.Getenv(key))
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}
