package cliutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// APIConfig stores common CLI connection settings.
type APIConfig struct {
	APIBase string `json:"api_base"`
	Token   string `json:"token,omitempty"`
}

// ResolveConfigDir determines where to store CLI config, preferring explicit env
// overrides, then os.UserConfigDir, falling back to HOME. The directory is
// created if it does not exist.
func ResolveConfigDir(appName string, envVars ...string) (string, error) {
	for _, env := range envVars {
		if val := strings.TrimSpace(os.Getenv(env)); val != "" {
			dir := filepath.Clean(val)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return "", fmt.Errorf("create config dir from %s: %w", env, err)
			}
			return dir, nil
		}
	}

	ensureDir := func(path string) (string, error) {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		return path, nil
	}

	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		namespaced := filepath.Join(dir, "vrooli", appName)
		legacy := filepath.Join(dir, appName)
		if info, err := os.Stat(legacy); err == nil && info.IsDir() {
			return ensureDir(legacy)
		}
		return ensureDir(namespaced)
	}

	home := os.Getenv("HOME")
	if home == "" {
		if dir, err := os.UserHomeDir(); err == nil {
			home = dir
		}
	}
	if home == "" {
		return "", fmt.Errorf("unable to resolve config directory")
	}

	namespaced := filepath.Join(home, ".vrooli", "config", appName)
	legacy := filepath.Join(home, "."+appName)
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return ensureDir(legacy)
	}
	return ensureDir(namespaced)
}

// LoadAPIConfig returns a ConfigFile at the standard location and loads any
// existing APIConfig into memory.
func LoadAPIConfig(appName string, envVars ...string) (*ConfigFile, APIConfig, error) {
	dir, err := ResolveConfigDir(appName, envVars...)
	if err != nil {
		return nil, APIConfig{}, err
	}
	cfgFile, err := NewConfigFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, APIConfig{}, err
	}

	cfg := APIConfig{}
	if err := cfgFile.Load(&cfg); err != nil {
		return nil, APIConfig{}, err
	}
	return cfgFile, cfg, nil
}
