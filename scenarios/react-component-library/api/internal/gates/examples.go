package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"react-component-library/internal/librarywalk"
)

func ValidateExamples(scope Scope) (Result, error) {
	root := scope.Root
	result := Result{}
	for _, kind := range []string{"components", "primitives"} {
		manifests, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		sort.Strings(manifests)
		for _, manifestPath := range manifests {
			if len(scope.Assets) > 0 && !scopeReportsAsset(scope, implementationName(manifestPath)) {
				continue
			}
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return Result{}, err
			}
			var manifest struct {
				Latest string `json:"latest"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Result{}, err
			}
			result.Inspected++
			storyPath := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest, "story.json")
			if _, err := os.Stat(storyPath); err != nil {
				if os.IsNotExist(err) {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.examples_missing", AssetID: filepath.Base(filepath.Dir(manifestPath)), File: repoRel(root, storyPath),
						Message:     fmt.Sprintf("released version %s has no story.json specimen", manifest.Latest),
						Remediation: fmt.Sprintf("Author %s describing at least one rendering of this asset. The specimen is what the preview surface, the visual gate, and every adopting scenario's picker all read; a released renderable asset without one is invisible in the catalog UI and is skipped by every gate that needs something to render.", repoRel(root, storyPath)),
						DocsRef:     "docs/internal/TESTING.md",
					})
					continue
				}
				return Result{}, err
			}
		}
	}
	return nonEmpty(result, "examples"), nil
}

// ValidateStress requires every active renderable implementation to have an
// indexed story contract. The story contract is the stress fixture boundary:
// it is where long, empty, disabled, and large-value specimens are declared
// and version-pinned for the browser runner.
