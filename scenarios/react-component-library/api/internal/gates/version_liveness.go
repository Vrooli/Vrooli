package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/react-component-library/libspec"
	"react-component-library/internal/librarywalk"
)

func ValidateVersionLiveness(scope Scope) (Result, error) {
	root := scope.Root
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	var sources []string
	err := librarywalk.WalkContext(scope.Context, libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		// Behavioral fixtures are validated by the component-test gates. They
		// intentionally retain historical relative imports and must not be
		// mistaken for published library modules by release liveness.
		if strings.Contains(filepath.ToSlash(path), "/library/tests/") {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".ts" || ext == ".tsx" {
			if !sourceInScope(root, path, scope) {
				return nil
			}
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.Strings(sources)
	result := Result{Inspected: len(sources)}
	manifests, _ := librarywalk.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	for _, manifestPath := range manifests {
		raw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct {
			LibraryID       string   `json:"libraryId"`
			EvictedVersions []string `json:"evictedVersions"`
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return Result{}, err
		}
		for _, version := range manifest.EvictedVersions {
			versionDir := filepath.Join(filepath.Dir(manifestPath), "versions", version)
			if _, statErr := os.Stat(versionDir); statErr == nil {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.evicted_version_materialized", AssetID: "__corpus__.version-liveness",
					File:        repoRel(root, versionDir),
					Message:     fmt.Sprintf("evicted version remains materialized: %s@%s", manifest.LibraryID, version),
					Remediation: "Reconcile every dependency lock, then remove the exact evicted directory through the version lifecycle cleanup flow.",
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
	}
	for _, path := range sources {
		retired, retiredErr := isRetiredVersion(path)
		if retiredErr != nil {
			return Result{}, retiredErr
		}
		if retired {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		assetID := implementationName(path)
		for _, specifier := range libspec.ParseAll(text) {
			if specifier.Selector == "" {
				continue
			}
			name, version := specifier.Name, specifier.Selector
			if !versionEntryExists(libraryRoot, name, version) {
				line := lineAt(data, strings.Index(text, libspec.Prefix+name+"/"+version))
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.version_liveness", AssetID: assetID, File: repoRel(root, path), Line: line,
					Message:     fmt.Sprintf("imports retired or missing library version %s@%s", name, version),
					Remediation: fmt.Sprintf("Point this import at a surviving %s version and review `react-component-library versions reap` before retiring any further versions. Published package subpaths must resolve to a live version entry.", name),
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
		for _, match := range regexp.MustCompile(`(?:from\s*|import\s*)[\"']([^\"']+)[\"']`).FindAllStringSubmatchIndex(text, -1) {
			specifier := text[match[2]:match[3]]
			if strings.HasPrefix(specifier, ".") && strings.Contains(specifier, "/versions/") {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.version_liveness", AssetID: assetID, File: repoRel(root, path), Line: lineAt(data, match[0]),
					Message:     fmt.Sprintf("retains a relative import into a version directory: %s", specifier),
					Remediation: "Use the published @vrooli/react-component-library/<asset>/<version> entry for a dependency, or move a shared helper into the importing version's own closure before retiring versions.",
					DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
				})
			}
		}
	}
	return nonEmpty(result, "version-liveness"), nil
}
