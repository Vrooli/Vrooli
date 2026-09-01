package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/librarywalk"
)

func ValidateBASGenericity(scope Scope) (Result, error) {
	root := scope.Root
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	componentNames := map[string]bool{}
	manifests, err := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifestPath := range manifests {
		data, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct{ DisplayName, LibraryID string }
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		name := filepath.Base(filepath.Dir(manifestPath))
		if name != "" {
			componentNames[strings.ToLower(name)] = true
		}
		if manifest.DisplayName != "" {
			componentNames[strings.ToLower(manifest.DisplayName)] = true
		}
	}
	result := Result{}
	for _, directory := range []string{"cases", "calibration"} {
		base := filepath.Join(root, "scenarios", "react-component-library", "bas", directory)
		err := librarywalk.Walk(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() || (filepath.Ext(path) != ".json" && filepath.Ext(path) != ".js") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			result.Inspected++
			text := strings.ToLower(string(data))
			if strings.Contains(text, "version=") {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.bas_version_pin", AssetID: "__corpus__.bas-genericity", File: repoRel(root, path),
					Message: "BAS workflow embeds a version query parameter", Remediation: "Pass the version through the generic runner input contract instead of encoding it in the workflow.", DocsRef: "docs/reference/composition-validation.md",
				})
			}
			// Descriptions and labels are allowed to name a capability (for
			// example, "overlay" or "surface"). Only structured asset-selection
			// fields and package imports are identity-bearing BAS knowledge.
			identityText := basIdentityText(data, path)
			for name := range componentNames {
				if regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(name) + `([^a-z0-9]|$)`).MatchString(strings.ToLower(identityText)) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.bas_component_knowledge", AssetID: "__corpus__.bas-genericity", File: repoRel(root, path),
						Message: fmt.Sprintf("BAS workflow contains component-specific name %q", name), Remediation: "Move asset selection into story capabilities and runner parameters; keep the workflow universal.", DocsRef: "docs/reference/composition-validation.md",
					})
					break
				}
			}
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	return nonEmpty(result, "bas-genericity"), nil
}
