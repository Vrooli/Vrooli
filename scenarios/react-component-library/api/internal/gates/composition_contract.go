package gates

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ValidateCompositionContract reconciles the catalog's shared context
// declarations with the source that is expected to honor them. The catalog is
// the relationship oracle; source is only used to catch a child that silently
// replaces a declared context value with a hard-coded treatment.
func ValidateCompositionContract(scope Scope) (Result, error) {
	root := scope.Root
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	providers := map[string][]string{}
	consumers := map[string][]string{}
	result := Result{}

	for _, asset := range assets {
		if len(asset.Provides) == 0 && len(asset.Consumes) == 0 {
			continue
		}
		result.Inspected++
		result.InspectedAssets = appendUnique(result.InspectedAssets, asset.Asset.ID)
		for key := range asset.Provides {
			providers[key] = append(providers[key], asset.Asset.ID)
		}
		for key := range asset.Consumes {
			consumers[key] = append(consumers[key], asset.Asset.ID)
		}

		_, sourcePath, implemented, sourceErr := implementationSource(root, asset.Asset.ID)
		if sourceErr != nil {
			return Result{}, sourceErr
		}
		if !implemented {
			continue
		}
		sourceBytes, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			return Result{}, readErr
		}
		source := string(sourceBytes)
		for key := range asset.Provides {
			if !strings.Contains(source, "SurfaceProvider") {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.composition_contract_provider_not_shared",
					AssetID:     asset.Asset.ID,
					File:        repoRel(root, sourcePath),
					Line:        lineOf(sourceBytes, key),
					Message:     fmt.Sprintf("provider declares %q without using the shared SurfaceProvider", key),
					Remediation: "Establish declared context values through SurfaceProvider from the Contracts foundation so consumers can reconcile the runtime relationship.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
		for key := range asset.Consumes {
			if !strings.Contains(source, "useSurfaceContext") {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.composition_contract_context_missing",
					AssetID:     asset.Asset.ID,
					File:        repoRel(root, sourcePath),
					Line:        lineOf(sourceBytes, key),
					Message:     fmt.Sprintf("consumer declares %q but does not read the shared SurfaceContext", key),
					Remediation: "Read the declared key through useSurfaceContext and retain the catalog fallback only for standalone rendering.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
			if key == "elevation" && hardCodedElevation(source) {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.composition_contract_contexts_differ",
					AssetID:     asset.Asset.ID,
					File:        repoRel(root, sourcePath),
					Line:        hardCodedElevationLine(sourceBytes),
					Message:     "elevation consumer hard-codes a shadow value, so standalone and provided contexts cannot differ",
					Remediation: "Use the shared SurfaceContext elevation and keep the declared fallback only as the no-provider standalone value. The differential claim requires contexts-differ.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
	}

	for key, ids := range providers {
		if len(consumers[key]) == 0 {
			for _, id := range ids {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.composition_contract_no_consumer",
					AssetID:     id,
					Message:     fmt.Sprintf("provider declares %q but no catalog consumer declares that key", key),
					Remediation: "Add a consuming asset for the provided context key, or remove the provider declaration until a real composition relationship exists.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
	}
	for key, ids := range consumers {
		if len(providers[key]) == 0 {
			for _, id := range ids {
				result.Findings = append(result.Findings, Finding{
					Code:        "catalog.composition_contract_orphan_consumer",
					AssetID:     id,
					Message:     fmt.Sprintf("consumer declares %q but no catalog provider declares that key", key),
					Remediation: "Declare the provider relationship on the container asset or remove the consumer declaration.",
					DocsRef:     "docs/internal/TESTING.md",
				})
			}
		}
	}

	return nonEmpty(result, "composition-contract"), nil
}

var elevationLiteralPattern = regexp.MustCompile(`var\(--elev-(?:flat|raised|floating|overlay)\)`)

func hardCodedElevation(source string) bool {
	return elevationLiteralPattern.MatchString(source)
}

func hardCodedElevationLine(source []byte) int {
	for _, literal := range []string{"var(--elev-flat)", "var(--elev-raised)", "var(--elev-floating)", "var(--elev-overlay)"} {
		if line := lineOf(source, literal); line > 0 {
			return line
		}
	}
	return 0
}
