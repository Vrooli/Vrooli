package components

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// ValidateVersionDependencyLocks enforces the generated-lock corpus contract
// without relying on the registry database. This keeps a clean checkout
// indexable and makes missing or dead pins fail before partial indexing.
func ValidateVersionDependencyLocks(fsys fs.FS) error {
	versionDirs := map[string]struct{}{}
	if err := fs.WalkDir(fsys, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isDependencyBearingSource(entry.Name()) {
			return nil
		}
		parts := strings.Split(filePath, "/")
		if len(parts) >= 5 && dependencyAssetRoot(parts[0]) && parts[2] == "versions" && !strings.Contains(parts[3], "-") {
			versionDirs[path.Dir(filePath)] = struct{}{}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk version dependency locks: %w", err)
	}

	directories := make([]string, 0, len(versionDirs))
	for directory := range versionDirs {
		directories = append(directories, directory)
	}
	sort.Strings(directories)
	var findings []string
	for _, directory := range directories {
		lockPath := path.Join(directory, "dependencies.json")
		raw, err := fs.ReadFile(fsys, lockPath)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: missing dependencies.json", directory))
			continue
		}
		var lock versionDependencyLock
		if err := json.Unmarshal(raw, &lock); err != nil {
			findings = append(findings, fmt.Sprintf("%s: invalid dependencies.json: %v", directory, err))
			continue
		}
		for _, dependency := range lock.Dependencies {
			name := strings.TrimPrefix(strings.TrimSpace(dependency.LibraryID), "react-component-library:")
			if name == "" || name == dependency.LibraryID || !dependencyTargetExists(fsys, name, strings.TrimSpace(dependency.Version)) {
				findings = append(findings, fmt.Sprintf("%s: dependency %s@%s has no materialized version directory", directory, dependency.LibraryID, dependency.Version))
			}
		}
	}
	if len(findings) > 0 {
		return fmt.Errorf("version dependency lock validation failed:\n  - %s", strings.Join(findings, "\n  - "))
	}
	return nil
}

func dependencyAssetRoot(root string) bool {
	switch root {
	case "foundations", "hooks", "services", "adapters", "primitives", "components", "patterns", "navigation", "page-templates":
		return true
	default:
		return false
	}
}

func isDependencyBearingSource(name string) bool {
	lower := strings.ToLower(name)
	if lower == "story.tsx" || strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
		return false
	}
	return strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx")
}

func dependencyTargetExists(fsys fs.FS, name, version string) bool {
	for _, root := range []string{"foundations", "hooks", "services", "adapters", "primitives", "components", "patterns", "navigation", "page-templates"} {
		info, err := fs.Stat(fsys, path.Join(root, name, "versions", version))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
