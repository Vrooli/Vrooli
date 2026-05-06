package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	TemplateBaseDirEnvVar = "SCENARIO_TEMPLATE_BASE_DIR"
	DesignBaseDirEnvVar   = "SCENARIO_DESIGN_BASE_DIR"
)

func HomeDir() (string, error) {
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}

func VrooliDir(home string) string {
	return filepath.Join(home, ".vrooli")
}

func RepoConfigDir(root string) string {
	return filepath.Join(root, ".vrooli")
}

func TemplateBaseDir(root string) string {
	if override := strings.TrimSpace(os.Getenv(TemplateBaseDirEnvVar)); override != "" {
		if filepath.IsAbs(override) {
			return filepath.Clean(override)
		}
		return filepath.Clean(filepath.Join(root, filepath.FromSlash(override)))
	}
	return filepath.Join(root, "templates", "scenarios")
}

func DesignBaseDir(root string) string {
	if override := strings.TrimSpace(os.Getenv(DesignBaseDirEnvVar)); override != "" {
		if filepath.IsAbs(override) {
			return filepath.Clean(override)
		}
		return filepath.Clean(filepath.Join(root, filepath.FromSlash(override)))
	}
	return filepath.Join(root, "templates", "design")
}
