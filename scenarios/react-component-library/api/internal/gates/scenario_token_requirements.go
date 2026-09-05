package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"react-component-library/internal/themes"
)

func ValidateScenarioTokenRequirements(scope Scope) (Result, error) {
	root := scope.Root
	baseTokens, err := themes.ReadTokenFile(filepath.Join(root, "templates", "design", "_base", "tokens.css"))
	if err != nil {
		return Result{}, err
	}
	reference := make(map[string]themes.DesignToken, len(baseTokens))
	for _, token := range baseTokens {
		if token.Tier == "" {
			return Result{}, fmt.Errorf("canonical token %s has no parseable tier annotation", token.Name)
		}
		reference[token.Name] = token
	}
	assets, err := readGateLibraryAssets(root)
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	scenariosRoot := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		return Result{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "react-component-library" {
			continue
		}
		scenario := entry.Name()
		specifiers, declared, scanErr := scanScenarioLibraryImports(filepath.Join(scenariosRoot, scenario, "ui", "src"))
		if scanErr != nil {
			return Result{}, scanErr
		}
		if len(specifiers) == 0 {
			continue
		}
		result.Inspected++
		mainPath := filepath.Join(scenariosRoot, scenario, "ui", "src", "main.tsx")
		mainSource, mainErr := os.ReadFile(mainPath)
		if mainErr != nil && !os.IsNotExist(mainErr) {
			return Result{}, mainErr
		}
		mainText := string(mainSource)
		if !containsCanonicalLayerMount(mainText) {
			first := specifiers[0]
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.scenario_canonical_layer_unmounted",
				AssetID:     "__corpus__.scenario-token-requirements",
				File:        repoRel(root, mainPath),
				Message:     fmt.Sprintf("scenario %s imports %s but does not mount the canonical BaseStyles provider", scenario, first.Name),
				Remediation: fmt.Sprintf("Add `import { BaseStyles } from \"@vrooli/react-component-library/BaseStyles/1\";` to %s and render `<BaseStyles />` above the application root.", repoRel(root, mainPath)),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
			})
		}
		required := map[string]bool{}
		for _, specifier := range specifiers {
			asset, exists := assets[specifier.Name]
			if !exists {
				result.Findings = append(result.Findings, unresolvedScenarioImportFinding(root, scenario, specifier, fmt.Sprintf("asset %s is absent from the library", specifier.Name)))
				continue
			}
			version, selectErr := selectGateAssetVersion(asset, specifier.RequestedVersion)
			if selectErr != nil {
				result.Findings = append(result.Findings, unresolvedScenarioImportFinding(root, scenario, specifier, selectErr.Error()))
				continue
			}
			if collectErr := collectGateVersionTokens(assets, asset, version, reference, required, map[string]bool{}); collectErr != nil {
				return Result{}, collectErr
			}
		}
		rampPath := filepath.Join(scenariosRoot, scenario, scenarioTokenRampPath(root, scenario))
		_, rampErr := readManagedRampProperties(rampPath)
		if rampErr != nil {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.scenario_token_ramp_unavailable",
				AssetID:     "__corpus__.scenario-token-requirements",
				File:        repoRel(root, rampPath),
				Message:     fmt.Sprintf("scenario %s has no readable managed design-token ramp: %v", scenario, rampErr),
				Remediation: fmt.Sprintf("Create or restore %s with rcl:tokens:begin and rcl:tokens:end markers, then run the governed token-sync workflow. A missing ramp cannot prove that the imported release has the tokens it requires.", repoRel(root, rampPath)),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
			})
			continue
		}
		missing := make([]string, 0)
		for property := range required {
			token, exists := reference[property]
			if !exists || token.Tier == themes.TokenTierContract {
				continue
			}
			if !declared[property] {
				missing = append(missing, property)
			}
		}
		sort.Strings(missing)
		for _, property := range missing {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.scenario_token_requirements",
				AssetID:     "__corpus__.scenario-token-requirements",
				File:        repoRel(root, rampPath),
				Message:     fmt.Sprintf("scenario %s imports a library release requiring %s, but its UI does not declare it", scenario, property),
				Remediation: fmt.Sprintf("Run `react-component-library adoptions tokens-sync %s`, then review the managed region in %s. Do not add Contract-tier properties or edit declarations outside the managed markers.", scenario, repoRel(root, rampPath)),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
			})
		}
	}
	return nonEmpty(result, "scenario-token-requirements"), nil
}
