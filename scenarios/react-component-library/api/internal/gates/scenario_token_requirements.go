package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/themes"
)

const (
	scenarioRampPath  = "ui/src/design-tokens.css"
	scenarioRampBegin = "/* rcl:tokens:begin */"
	scenarioRampEnd   = "/* rcl:tokens:end */"
)

type gateLibraryAsset struct {
	name        string
	dir         string
	unavailable map[string]bool
}

// ValidateScenarioTokenRequirements proves that each scenario's managed token
// region covers the active package releases it actually imports. Contract-tier
// properties are host behavior and are intentionally never copied into ramps.
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
		rampPath := filepath.Join(scenariosRoot, scenario, scenarioRampPath)
		_, rampErr := readManagedRampProperties(rampPath)
		if rampErr != nil {
			return Result{}, rampErr
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

func unresolvedScenarioImportFinding(root, scenario string, specifier components.LibraryPackageSpecifier, reason string) Finding {
	requested := specifier.RequestedVersion
	if requested == "" {
		requested = "newest active release"
	}
	return Finding{
		Code:        "catalog.scenario_token_import_unresolved",
		AssetID:     "__corpus__.scenario-token-requirements",
		File:        repoRel(root, filepath.Join(root, "scenarios", scenario, "ui", "src")),
		Message:     fmt.Sprintf("scenario %s imports %s at %s, but the export-map policy cannot resolve it: %s", scenario, specifier.Name, requested, reason),
		Remediation: fmt.Sprintf("Update %s to an active exported version of %s, then rerun token sync. Requirements cannot be proven for an import that the package export map no longer serves.", scenario, specifier.Name),
		DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
	}
}

func readGateLibraryAssets(root string) (map[string]gateLibraryAsset, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "component.json"))
	if err != nil {
		return nil, err
	}
	assets := make(map[string]gateLibraryAsset, len(paths))
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			LibraryID          string   `json:"libraryId"`
			DeprecatedVersions []string `json:"deprecatedVersions"`
			EvictedVersions    []string `json:"evictedVersions"`
		}
		if decodeErr := json.Unmarshal(raw, &manifest); decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", path, decodeErr)
		}
		name := strings.TrimPrefix(manifest.LibraryID, "react-component-library:")
		if name == manifest.LibraryID || name == "" {
			continue
		}
		unavailable := map[string]bool{}
		for _, version := range append(manifest.DeprecatedVersions, manifest.EvictedVersions...) {
			unavailable[version] = true
		}
		assets[name] = gateLibraryAsset{name: name, dir: filepath.Dir(path), unavailable: unavailable}
	}
	return assets, nil
}

func selectGateAssetVersion(asset gateLibraryAsset, requested string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(asset.dir, "versions"))
	if err != nil {
		return "", err
	}
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !asset.unavailable[entry.Name()] {
			candidates = append(candidates, entry.Name())
		}
	}
	selected, ok := components.SelectActivePackageVersion(candidates, requested)
	if !ok {
		return "", fmt.Errorf("asset %s has no active release matching %q", asset.name, requested)
	}
	return selected, nil
}

func scanScenarioLibraryImports(sourceRoot string) ([]components.LibraryPackageSpecifier, map[string]bool, error) {
	if _, err := os.Stat(sourceRoot); os.IsNotExist(err) {
		return nil, map[string]bool{}, nil
	}
	seen := map[components.LibraryPackageSpecifier]bool{}
	declared := map[string]bool{}
	result := []components.LibraryPackageSpecifier{}
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".css" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(text, -1) {
			declared[match[1]] = true
		}
		for _, specifier := range components.LibraryPackageSpecifiers(text) {
			if !seen[specifier] {
				seen[specifier] = true
				result = append(result, specifier)
			}
		}
		return nil
	})
	return result, declared, err
}

func collectGateVersionTokens(assets map[string]gateLibraryAsset, asset gateLibraryAsset, version string, reference map[string]themes.DesignToken, required map[string]bool, seen map[string]bool) error {
	key := asset.name + "@" + version
	if seen[key] {
		return nil
	}
	seen[key] = true
	versionDir := filepath.Join(asset.dir, "versions", version)
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return err
	}
	files := []components.ComponentVersionFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".ts" && ext != ".tsx" && ext != ".css" {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(versionDir, entry.Name()))
		if readErr != nil {
			return readErr
		}
		files = append(files, components.ComponentVersionFile{Path: entry.Name(), Content: string(raw)})
	}
	for _, property := range components.ExtractRequiredTokens(files) {
		required[property] = true
	}
	for _, pattern := range components.ExtractRequiredTokenPatterns(files) {
		prefix := strings.TrimSuffix(pattern, "*")
		for property := range reference {
			if strings.HasPrefix(property, prefix) {
				required[property] = true
			}
		}
	}
	lockRaw, err := os.ReadFile(filepath.Join(versionDir, "dependencies.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var lock struct {
		Dependencies []struct {
			LibraryID string `json:"libraryId"`
			Version   string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(lockRaw, &lock); err != nil {
		return fmt.Errorf("decode %s dependencies: %w", key, err)
	}
	for _, dependency := range lock.Dependencies {
		name := strings.TrimPrefix(dependency.LibraryID, "react-component-library:")
		child, exists := assets[name]
		if !exists {
			return fmt.Errorf("%s depends on unknown library asset %s", key, dependency.LibraryID)
		}
		if err := collectGateVersionTokens(assets, child, dependency.Version, reference, required, seen); err != nil {
			return err
		}
	}
	return nil
}

func readManagedRampProperties(path string) (map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(raw)
	begin := strings.Index(text, scenarioRampBegin)
	end := strings.Index(text, scenarioRampEnd)
	if begin < 0 || end < begin {
		return nil, fmt.Errorf("scenario token ramp %s has no valid managed region", path)
	}
	managed := text[begin+len(scenarioRampBegin) : end]
	properties := map[string]bool{}
	for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(managed, -1) {
		properties[match[1]] = true
	}
	return properties, nil
}
