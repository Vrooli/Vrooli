package services

import (
	"os"
	"path/filepath"
)

// ResolveConfigBasePath returns the base path for configuration files.
func ResolveConfigBasePath() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root"
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}
	return filepath.Join(vrooliRoot, "scenarios", "system-monitor", "initialization", "configuration")
}

// ResolvePromptBasePath returns the base path for prompt templates.
func ResolvePromptBasePath() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root"
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}
	return filepath.Join(vrooliRoot, "scenarios", "system-monitor", "initialization", "claude-code")
}

// ResolveScriptsDir finds the investigations/active directory on disk.
func ResolveScriptsDir() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			vrooliRoot = filepath.Join(homeDir, "Vrooli")
		}
	}
	if vrooliRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			vrooliRoot = cwd
		} else {
			return "."
		}
	}

	scriptsPath := filepath.Join(vrooliRoot, "scenarios", "system-monitor", "investigations", "active")
	if info, err := os.Stat(scriptsPath); err == nil && info.IsDir() {
		return scriptsPath
	}

	return filepath.Join(vrooliRoot, "investigations", "active")
}
