package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Manifest represents the relevant fields from package.json.
type Manifest struct {
	// Scripts contains the npm scripts defined in package.json.
	Scripts map[string]string `json:"scripts"`

	// PackageManager specifies the package manager to use (e.g., "pnpm@8.0.0").
	PackageManager string `json:"packageManager"`
}

// LoadManifest reads and parses a package.json file.
// Returns nil, nil if the file doesn't exist.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	if manifest.Scripts == nil {
		manifest.Scripts = make(map[string]string)
	}

	// Normalize the test script - treat placeholder as empty
	testScript := strings.TrimSpace(manifest.Scripts["test"])
	if testScript == "" || testScript == `echo "Error: no test specified" && exit 1` {
		manifest.Scripts["test"] = ""
	} else {
		manifest.Scripts["test"] = testScript
	}

	return &manifest, nil
}

// HasTestScript returns true if the manifest has a valid test script.
func (m *Manifest) HasTestScript() bool {
	return m != nil && m.Scripts["test"] != ""
}

// DetectPackageManager determines which package manager to use.
// Priority: packageManager field > lockfile detection > npm default.
func DetectPackageManager(manifest *Manifest, dir string) string {
	// Check packageManager field first
	if manifest != nil && manifest.PackageManager != "" {
		if mgr := parsePackageManagerField(manifest.PackageManager); mgr != "" {
			return mgr
		}
	}

	// Check for lockfiles
	switch {
	case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(dir, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

// parsePackageManagerField extracts the package manager name from the field.
// Examples: "pnpm@8.0.0" -> "pnpm", "yarn@3.5.0" -> "yarn"
func parsePackageManagerField(raw string) string {
	if raw == "" {
		return ""
	}
	lowered := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lowered, "pnpm"):
		return "pnpm"
	case strings.HasPrefix(lowered, "yarn"):
		return "yarn"
	case strings.HasPrefix(lowered, "npm"):
		return "npm"
	default:
		return ""
	}
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
