package gates

import (
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"sort"
	"strings"

	"react-component-library/internal/components"
)

// ValidateVersionShape enforces the declared shape across every live released
// version. Historical folders are now canonicalized, so the migration cutoff
// is retained only as provenance in the policy file.
func ValidateVersionShape(scope Scope) (Result, error) {
	root := scope.Root
	var dirs []string
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*"))
		if err != nil {
			return Result{}, err
		}
		dirs = append(dirs, matches...)
	}
	sort.Strings(dirs)
	result := Result{}
	catalogIDs := catalogAssetIDs(root)
	for _, dir := range dirs {
		if strings.HasSuffix(dir, ".retired") {
			continue
		}
		retired, err := isRetiredVersion(dir)
		if err != nil {
			return Result{}, err
		}
		if retired {
			continue
		}
		if len(scope.Assets) > 0 && !scopeReportsAsset(scope, implementationName(dir)) {
			continue
		}
		result.Inspected++
		assetName := filepath.Base(filepath.Dir(filepath.Dir(dir)))
		problems, err := components.ValidateVersionShape(root, dir, assetName, false)
		if err != nil {
			return Result{}, err
		}
		if len(problems) == 0 {
			continue
		}
		for _, problem := range problems {
			assetID := implementationName(filepath.Join(dir, "component.json"))
			if !catalogIDs[assetID] {
				// A library identity without a catalog projection is itself a
				// measurable gap, but cannot be attributed to a catalog asset.
				// Keep it in the corpus bucket so annotation does not turn the
				// finding into a runner failure.
				assetID = "__corpus__.version-shape"
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.version_shape", AssetID: assetID,
				File:    repoRel(root, filepath.Join(dir, strings.TrimPrefix(strings.SplitN(problem, ":", 2)[0], "./"))),
				Message: problem, Remediation: "Author only the declared version files, then run catalog build to regenerate story.json and dependencies.json.", DocsRef: "docs/concepts/ASSET-DERIVATION.md",
			})
		}
	}
	return nonEmpty(result, "version-shape"), nil
}
