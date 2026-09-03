package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"
)

var distImportRE = regexp.MustCompile(`(?:from|import\s+)["']([^"']+)["']`)

// ValidateDistResolution checks the bundle's actual import edges against the
// package manifest and exports map. TypeScript path aliases are intentionally
// not considered: consumers resolve the published dist, not the workbench.
func ValidateDistResolution(scope Scope) (Result, error) {
	packageRoot := filepath.Join(scope.Root, "packages", "react-component-library")
	manifestPath := filepath.Join(packageRoot, "package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Result{}, err
	}
	var manifest struct {
		Dependencies         map[string]string          `json:"dependencies"`
		DevDependencies      map[string]string          `json:"devDependencies"`
		PeerDependencies     map[string]string          `json:"peerDependencies"`
		OptionalDependencies map[string]string          `json:"optionalDependencies"`
		Exports              map[string]json.RawMessage `json:"exports"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Result{}, err
	}
	declared := map[string]bool{}
	for _, group := range []map[string]string{manifest.Dependencies, manifest.DevDependencies, manifest.PeerDependencies, manifest.OptionalDependencies} {
		for name := range group {
			declared[name] = true
		}
	}
	files := []string{}
	err = librarywalk.Walk(filepath.Join(packageRoot, "dist"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.Strings(files)
	result := Result{}
	for _, path := range files {
		if len(scope.Assets) > 0 && !distPathMatchesAsset(path, scope.Assets) {
			continue
		}
		result.Inspected++
		contents, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		for _, match := range distImportRE.FindAllStringSubmatch(string(contents), -1) {
			if len(match) < 2 || distImportAllowed(packageRoot, path, match[1], declared, manifest.Exports) {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.dist_resolution", AssetID: "__corpus__.dist-resolution",
				File: repoRel(scope.Root, path), Line: lineOf(contents, match[0]),
				Message:     fmt.Sprintf("bundle import %q is not declared or exported", match[1]),
				Remediation: "Declare the external package or add the internal subpath to package.json exports.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#package-resolution",
			})
		}
	}
	return nonEmpty(result, "dist-resolution"), nil
}

func distPathMatchesAsset(path string, assets []string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, asset := range assets {
		name := strings.ToLower(asset[strings.LastIndex(asset, ".")+1:])
		if strings.Contains(base, name) {
			return true
		}
	}
	return false
}

func distImportAllowed(packageRoot, importer, specifier string, declared map[string]bool, exports map[string]json.RawMessage) bool {
	if strings.HasPrefix(specifier, ".") {
		candidate := filepath.Clean(filepath.Join(filepath.Dir(importer), specifier))
		for _, suffix := range []string{"", ".js", ".d.ts"} {
			if _, err := os.Stat(candidate + suffix); err == nil {
				return true
			}
		}
		return false
	}
	if strings.HasPrefix(specifier, "node:") {
		return true
	}
	if strings.HasPrefix(specifier, "@vrooli/react-component-library") {
		subpath := strings.TrimPrefix(specifier, "@vrooli/react-component-library")
		if subpath == "" {
			subpath = "."
		} else if !strings.HasPrefix(subpath, ".") {
			subpath = "." + subpath
		}
		_, ok := exports[subpath]
		return ok
	}
	if strings.HasPrefix(specifier, "@vrooli/") {
		workspacePackage := strings.TrimPrefix(specifier, "@vrooli/")
		if slash := strings.IndexByte(workspacePackage, '/'); slash >= 0 {
			workspacePackage = workspacePackage[:slash]
		}
		if _, err := os.Stat(filepath.Join(packageRoot, "..", workspacePackage, "package.json")); err == nil {
			return true
		}
	}
	packageName := specifier
	if strings.HasPrefix(packageName, "@") {
		parts := strings.Split(packageName, "/")
		if len(parts) >= 2 {
			packageName = strings.Join(parts[:2], "/")
		}
	} else if index := strings.IndexByte(packageName, '/'); index >= 0 {
		packageName = packageName[:index]
	}
	return declared[packageName] || packageRoot == ""
}
