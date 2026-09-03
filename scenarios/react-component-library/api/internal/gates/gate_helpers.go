// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"strings"
)

func fixtureStoryContracts(root string) ([]fixtureStoryContract, error) {
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return nil, err
	}
	contracts := make([]fixtureStoryContract, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var contract fixtureStoryContract
		if err := json.Unmarshal(data, &contract); err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}
	return contracts, nil
}

func fixtureConsumers(root, fixtureID string) (int, error) {
	contracts, err := fixtureStoryContracts(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, contract := range contracts {
		if contract.Composition.Fixture.Asset == fixtureID {
			count++
		}
	}
	return count, nil
}

func fixtureFailureAssertions(root, fixtureID string) (int, error) {
	contracts, err := fixtureStoryContracts(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, contract := range contracts {
		if contract.Composition.Fixture.Asset != fixtureID {
			continue
		}
		for _, story := range contract.Stories {
			for _, expectation := range story.Expect {
				if expectation.Role == "alert" ||
					(expectation.Attribute == "data-fixture-state" && expectation.Value == "failure") ||
					expectation.Selector == `[data-fixture-state="failure"]` ||
					strings.Contains(strings.ToLower(expectation.Value), "failure") ||
					strings.Contains(strings.ToLower(expectation.Value), "error") {
					count++
				}
			}
		}
	}
	return count, nil
}

// ValidateExamples checks that renderable assets have a public story contract
// beside their released source. Enum completeness is validated by the story
// contract parser in the registry; this gate owns the filesystem-level
// requirement so coverage never promotes a primitive with no specimen.
func basIdentityText(data []byte, path string) string {
	if filepath.Ext(path) != ".json" {
		return string(data)
	}
	var value any
	if json.Unmarshal(data, &value) != nil {
		return ""
	}
	var values []string
	var walk func(any, string)
	walk = func(node any, key string) {
		switch typed := node.(type) {
		case map[string]any:
			for childKey, child := range typed {
				lower := strings.ToLower(childKey)
				if strings.Contains(lower, "component") || strings.Contains(lower, "asset") || strings.Contains(lower, "library") || strings.Contains(lower, "story") || strings.Contains(lower, "package") {
					walk(child, lower)
				}
			}
		case []any:
			for _, child := range typed {
				walk(child, key)
			}
		case string:
			values = append(values, typed)
		}
	}
	walk(value, "")
	return strings.Join(values, "\n")
}

// defect is what a source check reports. Remediation is required alongside
// Message: a check that can describe a defect can describe its fix, and the
// pair is what makes the finding actionable without a second investigation.
type defect struct{ Message, Remediation, DocsRef string }

func ok() defect { return defect{} }

func validateActiveSources(scope Scope, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	return validateActiveSourcesWithPath(scope, gate, func(asset assetDoc, _ string, source string) defect {
		return check(asset, source)
	})
}

func validateActiveSourcesWithPath(scope Scope, gate string, check func(asset assetDoc, path, source string) defect) (Result, error) {
	root := scope.Root
	assets, err := loadLibraryAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	for _, asset := range assets {
		if !scope.IsFullCorpus() && !selected[asset.Asset.ID] {
			continue
		}
		if strings.HasPrefix(asset.Asset.ID, "supplemental.") {
			// Supplemental manifests are durable implementation inputs, but are
			// intentionally outside the active catalog population. They are not
			// runner failures and should not inflate the gate's skipped count.
			continue
		}
		sources, err := implementationSources(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if len(sources) == 0 {
			result.Skipped = append(result.Skipped, asset.Asset.ID)
			result.RunnerError = append(result.RunnerError, Finding{
				Code:        "catalog." + gate + "_asset_unresolved",
				AssetID:     asset.Asset.ID,
				Message:     fmt.Sprintf("no exported, non-deprecated implementation resolved for asset %q", asset.Asset.ID),
				Remediation: "Add a catalogId/libraryId-matching manifest and publish a non-deprecated version in the package exports map.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
			})
			continue
		}
		result.Inspected++
		result.InspectedAssets = append(result.InspectedAssets, asset.Asset.ID)
		for _, source := range sources {
			data, err := os.ReadFile(source.Path)
			if err != nil {
				return Result{}, err
			}
			result.InspectedVersions++
			if d := check(asset, source.Path, string(data)); d.Message != "" {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog." + gate, AssetID: asset.Asset.ID, File: repoRel(root, source.Path),
					Message: fmt.Sprintf("%s (version %s)", d.Message, source.Version), Remediation: d.Remediation, DocsRef: d.DocsRef,
				})
			}
		}
	}
	return nonEmpty(result, gate), nil
}

// validateActiveSourceFiles is used by gates whose contract applies to the
// complete implementation package, not only the component entrypoint. Style
// declarations commonly live beside that entrypoint in styles.ts, so looking
// at the first source file would make those declarations invisible.
func validateActiveSourceFiles(scope Scope, gate string, check func(asset assetDoc, source string) defect) (Result, error) {
	root := scope.Root
	assets, err := loadLibraryAssets(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	selected := make(map[string]bool, len(scope.Assets))
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	for _, asset := range assets {
		if !scope.IsFullCorpus() && !selected[asset.Asset.ID] {
			continue
		}
		versions, err := implementationSources(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if len(versions) == 0 {
			result.Skipped = append(result.Skipped, asset.Asset.ID)
			result.RunnerError = append(result.RunnerError, Finding{
				Code:        "catalog." + gate + "_asset_unresolved",
				AssetID:     asset.Asset.ID,
				Message:     fmt.Sprintf("no exported, non-deprecated implementation resolved for asset %q", asset.Asset.ID),
				Remediation: "Add a catalogId/libraryId-matching manifest and publish a non-deprecated version in the package exports map.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
			})
			continue
		}
		result.Inspected++
		result.InspectedAssets = append(result.InspectedAssets, asset.Asset.ID)
		for _, version := range versions {
			result.InspectedVersions++
			for _, path := range versionSources(filepath.Dir(version.Path)) {
				data, err := os.ReadFile(path)
				if err != nil {
					return Result{}, err
				}
				if d := check(asset, string(data)); d.Message != "" {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog." + gate, AssetID: asset.Asset.ID, File: repoRel(root, path),
						Message: fmt.Sprintf("%s (version %s)", d.Message, version.Version), Remediation: d.Remediation, DocsRef: d.DocsRef,
					})
				}
			}
		}
	}
	return nonEmpty(result, gate), nil
}

func nonEmpty(result Result, gate string) Result {
	if result.Inspected == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code:        "catalog." + gate + "_zero_inspected",
			AssetID:     "",
			Message:     "gate inspected zero inputs; runner configuration is stale or broken",
			Remediation: "Treat this as a runner fault, never as a pass. The most common cause is a source-glob that no longer matches the tree — check the path pattern this gate resolves against, and whether an asset kind or directory was renamed without updating it. A gate reporting no findings after inspecting nothing is indistinguishable from a clean corpus, which is exactly the failure this finding exists to make visible.",
			DocsRef:     "docs/internal/TESTING.md",
		})
	}
	return result
}

// UnmeasuredGate returns the built catalog asset set for a gate that has no
// runner. The set is only an attribution boundary; it is not an observation
// and therefore carries an explicit unmeasured status.
func UnmeasuredGate(root string) (Result, error) {
	result := Result{}
	kinds := []string{"foundations", "hooks", "services", "primitives", "components"}
	for _, kind := range kinds {
		manifests, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return Result{}, err
			}
			var doc struct {
				CatalogID string `json:"catalogId"`
				Latest    string `json:"latest"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return Result{}, fmt.Errorf("parse %s: %w", manifest, err)
			}
			if doc.CatalogID == "" || doc.Latest == "" {
				continue
			}
			result.Inspected++
			result.InspectedAssets = append(result.InspectedAssets, doc.CatalogID)
		}
	}
	result.Status = "unmeasured"
	return result, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
