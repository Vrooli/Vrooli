// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func libraryAssetForPath(libraryRoot, file string) string {
	relative, err := filepath.Rel(libraryRoot, filepath.Clean(file))
	if err != nil || strings.HasPrefix(relative, "..") {
		return ""
	}
	segments := strings.Split(filepath.ToSlash(relative), "/")
	if len(segments) < 2 {
		return ""
	}
	return segments[1]
}

func countCatalogSources(scope Scope) int {
	sources, _ := activeLibrarySources(scope)
	return len(sources)
}

// activeLibrarySources returns the files represented by each manifest's
// latest and draft pointers. Historical versions remain available to callers
// that pin them explicitly, but corpus-wide quality gates should measure the
// active catalog surface consistently with indexing, coverage, and the type
// gate rather than double-counting retired implementations.
func activeLibrarySources(scope Scope) ([]string, error) {
	root := scope.Root
	var sources []string
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return nil, err
			}
			var doc struct {
				Latest string `json:"latest"`
				Draft  string `json:"draft"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, err
			}
			versions := []string{doc.Latest}
			if doc.Draft != "" && doc.Draft != doc.Latest {
				versions = append(versions, doc.Draft)
			}
			for _, version := range versions {
				if version == "" {
					continue
				}
				for _, extension := range []string{"*.ts", "*.tsx"} {
					matches, err := filepath.Glob(filepath.Join(filepath.Dir(manifest), "versions", version, extension))
					if err != nil {
						return nil, err
					}
					for _, path := range matches {
						if sourceInScope(root, path, scope) {
							sources = append(sources, path)
						}
					}
				}
			}
		}
	}
	if len(sources) > 0 {
		sort.Strings(sources)
		return sources, nil
	}

	// Keep the unit-level gate contract useful for isolated fixtures that do
	// not need a full component manifest. Real repositories always take the
	// manifest-backed path above.
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		for _, extension := range []string{"*.ts", "*.tsx"} {
			matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*", extension))
			if err != nil {
				return nil, err
			}
			for _, path := range matches {
				if sourceInScope(root, path, scope) {
					sources = append(sources, path)
				}
			}
		}
	}
	sort.Strings(sources)
	return sources, nil
}

func sourceInScope(root, sourcePath string, scope Scope) bool {
	if scope.IsFullCorpus() {
		return true
	}
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	versionDir := filepath.Dir(sourcePath)
	assetDir := filepath.Dir(filepath.Dir(versionDir))
	manifestPath := filepath.Join(assetDir, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var manifest struct {
		CatalogID    string `json:"catalogId"`
		LibraryID    string `json:"libraryId"`
		Supplemental bool   `json:"supplemental"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	assetID := manifest.CatalogID
	if assetID == "" && manifest.Supplemental {
		assetID = "supplemental." + strings.ReplaceAll(manifest.LibraryID, ":", ".")
	}
	return selected[assetID]
}

func implementationSource(root, catalogID string) (string, string, bool, error) {
	sources, err := implementationSources(root, catalogID)
	if err != nil || len(sources) == 0 {
		return "", "", false, err
	}
	for _, source := range sources {
		if source.Version == source.Latest {
			return source.Manifest, source.Path, true, nil
		}
	}
	source := sources[len(sources)-1]
	return source.Manifest, source.Path, true, nil
}

type implementationSourceEntry struct {
	Manifest string
	Path     string
	Version  string
	Latest   string
}

// implementationSources returns every released version that consumers can
// reach through the published package exports map. Keeping this lookup next to
// implementationSource prevents a gate from silently regressing to latest-only
// coverage when a package publishes an older pinned entry.
func implementationSources(root, catalogID string) ([]implementationSourceEntry, error) {
	paths := make([]string, 0)
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return nil, err
		}
		var doc struct {
			CatalogID  string   `json:"catalogId"`
			LibraryID  string   `json:"libraryId"`
			Latest     string   `json:"latest"`
			Deprecated []string `json:"deprecatedVersions"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		if doc.CatalogID != catalogID && doc.LibraryID != catalogID {
			continue
		}
		return exportedImplementationSources(root, manifest, doc.Latest, doc.Deprecated), nil
	}
	return nil, nil
}

func exportedImplementationSources(root, manifest, latest string, deprecated []string) []implementationSourceEntry {
	rootDir := filepath.Dir(manifest)
	name := filepath.Base(rootDir)
	exported := exportedVersions(root, name)
	var versions []string
	entries, _ := os.ReadDir(filepath.Join(rootDir, "versions"))
	for _, entry := range entries {
		if !entry.IsDir() || containsString(deprecated, entry.Name()) {
			continue
		}
		if len(exported) > 0 && !exported[entry.Name()] {
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Slice(versions, func(i, j int) bool { return semverLikeLess(versions[i], versions[j]) })
	result := make([]implementationSourceEntry, 0, len(versions))
	for _, version := range versions {
		versionDir := filepath.Join(rootDir, "versions", version)
		source := filepath.Join(versionDir, name+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches := versionSources(versionDir)
			if len(matches) == 0 {
				continue
			}
			source = matches[0]
		}
		result = append(result, implementationSourceEntry{Manifest: manifest, Path: source, Version: version, Latest: latest})
	}
	return result
}

func exportedVersions(root, name string) map[string]bool {
	path := filepath.Join(root, "packages", "react-component-library", "dist", "exports", "resolution.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return exportedVersionsFromPackage(root, name)
	}
	var resolutions map[string]struct {
		Source string `json:"source"`
	}
	if json.Unmarshal(data, &resolutions) != nil {
		return nil
	}
	result := map[string]bool{}
	prefix := "./" + name + "/"
	for key, resolution := range resolutions {
		if strings.HasPrefix(key, prefix) {
			version := strings.TrimPrefix(key, prefix)
			if isConcreteVersion(version) && resolution.Source != "" {
				result[version] = true
			}
		}
	}
	return result
}

func exportedVersionsFromPackage(root, name string) map[string]bool {
	path := filepath.Join(root, "packages", "react-component-library", "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Exports map[string]json.RawMessage `json:"exports"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	result := map[string]bool{}
	prefix := "./" + name + "/"
	for key := range doc.Exports {
		version := strings.TrimPrefix(key, prefix)
		if strings.HasPrefix(key, prefix) && isConcreteVersion(version) {
			result[version] = true
		}
	}
	return result
}

func isConcreteVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}
