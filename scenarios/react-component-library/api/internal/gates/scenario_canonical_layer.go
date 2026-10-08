package gates

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateScenarioCanonicalLayer checks provider mounting separately from the
// wider token-ramp and release requirements corpus.
func ValidateScenarioCanonicalLayer(scope Scope) (Result, error) {
	result := Result{}
	entries, err := os.ReadDir(filepath.Join(scope.Root, "scenarios"))
	if err != nil {
		return Result{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "react-component-library" {
			continue
		}
		scenario := entry.Name()
		sourceRoot := filepath.Join(scope.Root, "scenarios", scenario, "ui", "src")
		specifiers, _, scanErr := scanScenarioLibraryImports(sourceRoot)
		if scanErr != nil {
			return Result{}, scanErr
		}
		if len(specifiers) == 0 {
			continue
		}
		result.Inspected++
		mainPath := filepath.Join(sourceRoot, "main.tsx")
		mainSource, readErr := os.ReadFile(mainPath)
		if readErr != nil && !os.IsNotExist(readErr) {
			return Result{}, readErr
		}
		if containsCanonicalLayerMount(string(mainSource)) {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.scenario_canonical_layer_unmounted",
			AssetID:     "__corpus__.scenario-canonical-layer",
			File:        repoRel(scope.Root, mainPath),
			Message:     fmt.Sprintf("scenario %s imports %s but does not mount the canonical BaseStyles provider", scenario, specifiers[0].Name),
			Remediation: fmt.Sprintf("Add `import { BaseStyles } from \"@vrooli/react-component-library/BaseStyles/1\";` to %s and render `<BaseStyles />` above the application root.", repoRel(scope.Root, mainPath)),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	return nonEmpty(result, "scenario-canonical-layer"), nil
}
