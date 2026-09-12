package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const previewRuntimeStoreEnv = "RCL_PREVIEW_RUNTIME_STORE"

// previewRuntimeSurfaceName is deliberately deterministic: one governed SDA
// surface owns one package request, while package.json records the exact
// version that was resolved by the package manager.
func previewRuntimeSurfaceName(name, versionRange string) string {
	return "preview-runtime"
}

func previewDependencyPopulateCommand(name, versionRange string) string {
	return fmt.Sprintf("scenario-dependency-analyzer deps install npm/%s@%s --scenario react-component-library --surface tools/%s --apply", name, versionRange, previewRuntimeSurfaceName(name, versionRange))
}

func previewRuntimeStoreRoot(repoRoot string) string {
	if root := strings.TrimSpace(os.Getenv(previewRuntimeStoreEnv)); root != "" {
		return root
	}
	return filepath.Join(repoRoot, "scenarios", "react-component-library", "tools", "preview-runtime")
}

func previewRuntimeSurfaceRoots(repoRoot string) []string {
	base := previewRuntimeStoreRoot(repoRoot)
	var roots []string
	if info, err := os.Stat(filepath.Join(base, "node_modules")); err == nil && info.IsDir() {
		roots = append(roots, filepath.Join(base, "node_modules"))
	}
	return roots
}

func installedPackageVersionCandidates(name string) []string {
	return installedPackageVersionCandidatesAtRoot(discoverRepoRoot(""), name)
}

func installedPackageVersionCandidatesAtRoot(repoRoot, name string) []string {
	name = strings.TrimSpace(name)
	if !safePackageName(name) {
		return nil
	}
	var versions []string
	for _, root := range previewRuntimeSurfaceRoots(repoRoot) {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name), "package.json"))
		if err != nil {
			continue
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(raw, &pkg); err != nil {
			continue
		}
		version := strings.TrimSpace(pkg.Version)
		if version != "" && !containsString(versions, version) {
			versions = append(versions, version)
		}
	}
	return versions
}

func previewRuntimePackageRoot(repoRoot, name, version string) (string, error) {
	name = strings.TrimSpace(name)
	if !safePackageName(name) {
		return "", fmt.Errorf("invalid preview dependency package %q", name)
	}
	for _, root := range previewRuntimeSurfaceRoots(repoRoot) {
		packageJSON := filepath.Join(root, filepath.FromSlash(name), "package.json")
		raw, err := os.ReadFile(packageJSON)
		if err != nil {
			continue
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(raw, &pkg) != nil {
			continue
		}
		if version == "" || strings.TrimSpace(pkg.Version) == strings.TrimSpace(version) {
			return root, nil
		}
	}
	return "", fmt.Errorf("preview dependency %s@%s is absent from governed store", name, version)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
