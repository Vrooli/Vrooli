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

	"react-component-library/internal/components"
	"react-component-library/internal/themes"
)

var tokenDeclarationRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)

type CompatibilityVerdict = components.KitCompatibilityVerdict

const (
	CompatibilityUniversal           = components.KitCompatibilityUniversal
	CompatibilityRestricted          = components.KitCompatibilityRestricted
	CompatibilityUnsatisfiable       = components.KitCompatibilityUnsatisfiable
	CompatibilityUndefinedVocabulary = components.KitCompatibilityUndefinedVocabulary
)

type ComponentTokenCensus struct {
	LibraryID      string               `json:"libraryId"`
	Version        string               `json:"version"`
	RequiredTokens []string             `json:"requiredTokens"`
	Verdict        CompatibilityVerdict `json:"verdict"`
	CompatibleKits []string             `json:"compatibleKits,omitempty"`
}

type AffinityOverclaim struct {
	LibraryID string `json:"libraryId"`
	StyleID   string `json:"styleId"`
}

type ScenarioRampCensus struct {
	Scenario       string `json:"scenario"`
	ManagedTokens  int    `json:"managedTokens"`
	HasManagedRamp bool   `json:"hasManagedRamp"`
}

type Census struct {
	ComponentsScanned      int                    `json:"componentsScanned"`
	ReferencedProperties   []string               `json:"referencedProperties"`
	RequiredProperties     []string               `json:"requiredProperties"`
	BaseStylesProperties   []string               `json:"baseStylesProperties"`
	KitPublishedProperties map[string][]string    `json:"kitPublishedProperties"`
	UndefinedProperties    []string               `json:"undefinedProperties"`
	VerdictCounts          map[string]int         `json:"verdictCounts"`
	Components             []ComponentTokenCensus `json:"components"`
	AffinityOverclaims     []AffinityOverclaim    `json:"affinityOverclaims"`
	ScenarioRamps          []ScenarioRampCensus   `json:"scenarioRamps"`
}

type componentManifest struct {
	LibraryID    string `json:"libraryId"`
	Latest       string `json:"latest"`
	DesignStyles []struct {
		StyleID string `json:"styleId"`
	} `json:"designStyles"`
}

func TokenCensus(root string) (Census, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Census{}, err
	}

	baseProperties, err := activeAssetDeclarations(filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "BaseStyles"))
	if err != nil {
		return Census{}, fmt.Errorf("read BaseStyles vocabulary: %w", err)
	}
	kits, err := readKitProperties(root)
	if err != nil {
		return Census{}, err
	}

	manifestPaths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "*", "component.json"))
	if err != nil {
		return Census{}, err
	}
	sort.Strings(manifestPaths)

	referenced := map[string]struct{}{}
	required := map[string]struct{}{}
	undefined := map[string]struct{}{}
	census := Census{
		BaseStylesProperties:   sortedSet(baseProperties),
		KitPublishedProperties: kits,
		VerdictCounts: map[string]int{
			string(CompatibilityUniversal):           0,
			string(CompatibilityRestricted):          0,
			string(CompatibilityUnsatisfiable):       0,
			string(CompatibilityUndefinedVocabulary): 0,
		},
	}

	for _, manifestPath := range manifestPaths {
		manifest, err := readComponentManifest(manifestPath)
		if err != nil {
			return Census{}, err
		}
		if manifest.Latest == "" {
			return Census{}, fmt.Errorf("component manifest %s has no latest version", manifestPath)
		}
		files, err := readVersionFiles(filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest))
		if err != nil {
			return Census{}, fmt.Errorf("read %s@%s: %w", manifest.LibraryID, manifest.Latest, err)
		}
		external := components.ExtractRequiredTokens(files)
		for _, property := range external {
			referenced[property] = struct{}{}
		}
		componentRequired := make([]string, 0, len(external))
		for _, property := range external {
			if strings.HasPrefix(property, "--rcl-") {
				continue
			}
			if _, supplied := baseProperties[property]; supplied {
				continue
			}
			required[property] = struct{}{}
			componentRequired = append(componentRequired, property)
		}

		compatibility := components.DeriveKitCompatibility(componentRequired, kits)
		compatible := compatibility.CompatibleKitIDs
		verdict := compatibility.Verdict
		for _, property := range compatibility.UnsatisfiedProperties {
			undefined[property] = struct{}{}
		}
		census.VerdictCounts[string(verdict)]++
		census.Components = append(census.Components, ComponentTokenCensus{
			LibraryID: manifest.LibraryID, Version: manifest.Latest,
			RequiredTokens: componentRequired, Verdict: verdict, CompatibleKits: compatible,
		})

		// Vocabulary/composition errors have their own blocking verdict and no
		// trustworthy compatibility set yet. Affinity breadth is evaluated only
		// once compatibility itself is defined, so one defect never fabricates a
		// second class of remediation findings.
		if verdict == CompatibilityUniversal || verdict == CompatibilityRestricted {
			compatibleSet := stringSet(compatible)
			for _, affinity := range manifest.DesignStyles {
				if _, allowed := compatibleSet[affinity.StyleID]; allowed {
					continue
				}
				census.AffinityOverclaims = append(census.AffinityOverclaims, AffinityOverclaim{
					LibraryID: manifest.LibraryID, StyleID: affinity.StyleID,
				})
			}
		}
	}

	census.ComponentsScanned = len(census.Components)
	census.ReferencedProperties = sortedSet(referenced)
	census.RequiredProperties = sortedSet(required)
	census.UndefinedProperties = sortedSet(undefined)
	census.ScenarioRamps, err = readScenarioRamps(root)
	if err != nil {
		return Census{}, err
	}
	return census, nil
}

func readComponentManifest(path string) (componentManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return componentManifest{}, err
	}
	var manifest componentManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return componentManifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return manifest, nil
}

func readVersionFiles(dir string) ([]components.ComponentVersionFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]components.ComponentVersionFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".ts" && ext != ".tsx" && ext != ".css" && ext != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, components.ComponentVersionFile{Path: entry.Name(), Content: string(raw)})
	}
	return files, nil
}

func activeAssetDeclarations(assetDir string) (map[string]struct{}, error) {
	manifest, err := readComponentManifest(filepath.Join(assetDir, "component.json"))
	if err != nil {
		return nil, err
	}
	files, err := readVersionFiles(filepath.Join(assetDir, "versions", manifest.Latest))
	if err != nil {
		return nil, err
	}
	result := map[string]struct{}{}
	for _, file := range files {
		for _, match := range tokenDeclarationRE.FindAllStringSubmatch(file.Content, -1) {
			result[match[1]] = struct{}{}
		}
	}
	return result, nil
}

func readKitProperties(root string) (map[string][]string, error) {
	metadataPaths, err := librarywalk.Glob(filepath.Join(root, "templates", "design", "*", "metadata.json"))
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(metadataPaths))
	_, baseErr := os.Stat(filepath.Join(root, "templates", "design", "_base", "tokens.css"))
	for _, metadataPath := range metadataPaths {
		kitID := filepath.Base(filepath.Dir(metadataPath))
		var tokens []themes.DesignToken
		var err error
		if baseErr == nil {
			tokens, err = themes.ResolveKitTokens(root, kitID)
		} else {
			tokens, err = themes.ReadTokenFile(filepath.Join(filepath.Dir(metadataPath), "adapters", "react-vite-tailwind", "tokens.css"))
		}
		if err != nil {
			return nil, err
		}
		properties := map[string]struct{}{}
		for _, token := range tokens {
			properties[token.Name] = struct{}{}
		}
		result[kitID] = sortedSet(properties)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no design kits found under templates/design")
	}
	return result, nil
}

func readScenarioRamps(root string) ([]ScenarioRampCensus, error) {
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "*", "ui", "src", "design-tokens.css"))
	if err != nil {
		return nil, err
	}
	result := make([]ScenarioRampCensus, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(raw)
		begin := strings.Index(text, "/* rcl:tokens:begin */")
		end := strings.Index(text, "/* rcl:tokens:end */")
		row := ScenarioRampCensus{Scenario: filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))}
		if begin >= 0 && end > begin {
			row.HasManagedRamp = true
			row.ManagedTokens = len(tokenDeclarationRE.FindAllStringSubmatch(text[begin:end], -1))
		}
		result = append(result, row)
	}
	return result, nil
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
