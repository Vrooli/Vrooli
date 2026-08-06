// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Finding struct{ Code, AssetID, Message string }

// Result makes runner coverage observable. A gate that reports no findings
// after inspecting zero inputs is not a passing gate; it is a broken runner.
type Result struct {
	Findings  []Finding
	Inspected int
}

type assetDoc struct {
	Asset struct {
		ID, Kind, Name string
		Target         struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
	} `json:"asset"`
	API *struct {
		Variants map[string][]string `json:"variants"`
		Modes    []string            `json:"modes"`
		Parts    []json.RawMessage   `json:"parts"`
	} `json:"api"`
	Fixture *struct {
		DataShapes []string `json:"dataShapes"`
		Satisfies  *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
}

func loadAssets(root string) ([]assetDoc, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []assetDoc
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc assetDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		out = append(out, doc)
	}
	return out, nil
}

// ValidateAPI checks declared API vocabulary against the implementation
// source selected by catalogId. Missing implementations are not failures of
// this runner; coverage keeps those assets at missing/scaffolded.
func ValidateAPI(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.API == nil {
			continue
		}
		manifest, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		for group, values := range asset.API.Variants {
			for _, value := range values {
				if !strings.Contains(text, value) {
					result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared %s variant %q is absent from %s", group, value, manifest)})
				}
			}
		}
		for _, value := range asset.API.Modes {
			if !strings.Contains(text, value) {
				result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared mode %q is absent from %s", value, manifest)})
			}
		}
		for _, rawPart := range asset.API.Parts {
			partID := ""
			if json.Unmarshal(rawPart, &partID) != nil {
				var part struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(rawPart, &part)
				partID = part.ID
			}
			if partID != "" && !strings.Contains(text, partID) {
				result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared part %q is absent from %s", partID, manifest)})
			}
		}
	}
	return nonEmpty(result, "api"), nil
}

func implementationSource(root, catalogID string) (string, string, bool, error) {
	paths := []string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return "", "", false, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return "", "", false, err
		}
		var doc struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.CatalogID != catalogID {
			continue
		}
		if doc.Latest == "" {
			return manifest, "", false, nil
		}
		rootDir := filepath.Dir(manifest)
		versionDir := filepath.Join(rootDir, "versions", doc.Latest)
		source := filepath.Join(versionDir, filepath.Base(rootDir)+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches := versionSources(versionDir)
			if len(matches) == 0 {
				versionDir = filepath.Join(rootDir, doc.Latest)
				matches = versionSources(versionDir)
			}
			if len(matches) == 0 {
				return manifest, "", false, nil
			}
			source = matches[0]
		}
		return manifest, source, true, nil
	}
	return "", "", false, nil
}

func versionSources(versionDir string) []string {
	var matches []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		found, _ := filepath.Glob(filepath.Join(versionDir, extension))
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	return matches
}

var (
	pxValue          = regexp.MustCompile(`--space-[a-z0-9-]+\s*:\s*([0-9.]+)px`)
	literalDimension = regexp.MustCompile(`(?:\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap|w|h)-[0-9]+(?:\.[0-9]+)?\b|\[[0-9.]+px\])`)
)

// ValidateTokens checks the shared ramp contract in every design kit and
// rejects non-grid spacing declarations.
func ValidateTokens(root string) (Result, error) {
	paths, err := filepath.Glob(filepath.Join(root, "templates", "design", "*", "adapters", "react-vite-tailwind", "tokens.css"))
	if err != nil {
		return Result{}, err
	}
	shared := []string{"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl", "text-display", "text-title", "text-heading", "text-body", "elev-flat", "elev-raised", "layer-base", "layer-modal", "dur-instant"}
	result := Result{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		kit := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
		for _, token := range shared {
			if !strings.Contains(text, "--"+token) {
				result.Findings = append(result.Findings, Finding{"catalog.tokens_missing", "foundations.tokens", fmt.Sprintf("%s does not declare shared token --%s", kit, token)})
			}
		}
		for _, match := range pxValue.FindAllStringSubmatch(text, -1) {
			value, _ := strconv.ParseFloat(match[1], 64)
			if int(value)%4 != 0 {
				result.Findings = append(result.Findings, Finding{"catalog.tokens_grid", "foundations.tokens", fmt.Sprintf("%s spacing token is not on the 4px grid: %spx", kit, match[1])})
			}
		}
	}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		sources := []string{}
		for _, extension := range []string{"*.tsx", "*.ts"} {
			matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*", extension))
			if err != nil {
				return Result{}, err
			}
			sources = append(sources, matches...)
		}
		sort.Strings(sources)
		for _, path := range sources {
			data, err := os.ReadFile(path)
			if err != nil {
				return Result{}, err
			}
			result.Inspected++
			for _, match := range literalDimension.FindAllString(string(data), -1) {
				result.Findings = append(result.Findings, Finding{"catalog.tokens_literal", implementationName(path), fmt.Sprintf("implementation contains literal dimension %q; use a declared semantic token", match)})
			}
		}
	}
	return nonEmpty(result, "tokens"), nil
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func ValidateLifecycle(root string) (Result, error) {
	result := Result{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths := []string{}
		for _, extension := range []string{"*.tsx", "*.ts"} {
			matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*", extension))
			if err != nil {
				return Result{}, err
			}
			paths = append(paths, matches...)
		}
		sort.Strings(paths)
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return Result{}, err
			}
			result.Inspected++
			text := string(data)
			if strings.Contains(text, "addEventListener") && !strings.Contains(text, "removeEventListener") {
				result.Findings = append(result.Findings, Finding{"catalog.lifecycle_cleanup", implementationName(path), "adds an event listener without a matching removal"})
			}
			if strings.Contains(text, "new MutationObserver") && !strings.Contains(text, ".disconnect(") {
				result.Findings = append(result.Findings, Finding{"catalog.lifecycle_cleanup", implementationName(path), "creates an observer without disconnect cleanup"})
			}
			if strings.Contains(text, "window.") && strings.Contains(text, "export") && !strings.Contains(text, "typeof window") {
				result.Findings = append(result.Findings, Finding{"catalog.lifecycle_ssr", implementationName(path), "accesses window without an SSR guard"})
			}
		}
	}
	return nonEmpty(result, "lifecycle"), nil
}

func implementationName(path string) string {
	return filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
}

func ValidateFixtures(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "fixture" || asset.Fixture == nil {
			continue
		}
		result.Inspected++
		if len(asset.Fixture.DataShapes) == 0 {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture declares no adversarial data shapes"})
		}
		if !contains(asset.Fixture.DataShapes, "failure") && !contains(asset.Fixture.DataShapes, "overflow") {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture must include failure or overflow data"})
		}
		if asset.Fixture.Satisfies != nil && asset.Fixture.Satisfies.Capability == "data-source" && len(asset.Fixture.Satisfies.TypeArguments) == 0 {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_data_source", asset.Asset.ID, "data-source fixture must declare a type argument"})
		}
	}
	return nonEmpty(result, "fixture_adversarial"), nil
}

// ValidateExamples checks that renderable assets have a public story contract
// beside their released source. Enum completeness is validated by the story
// contract parser in the registry; this gate owns the filesystem-level
// requirement so coverage never promotes a primitive with no specimen.
func ValidateExamples(root string) (Result, error) {
	result := Result{}
	for _, kind := range []string{"components", "primitives"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		sort.Strings(manifests)
		for _, manifestPath := range manifests {
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
					result.Findings = append(result.Findings, Finding{"catalog.examples_missing", filepath.Base(filepath.Dir(manifestPath)), "released renderable asset has no story.json specimen"})
					continue
				}
				return Result{}, err
			}
		}
	}
	return nonEmpty(result, "examples"), nil
}

func nonEmpty(result Result, gate string) Result {
	if result.Inspected == 0 {
		result.Findings = append(result.Findings, Finding{
			Code:    "catalog." + gate + "_zero_inspected",
			AssetID: "catalog.runner",
			Message: "gate inspected zero inputs; runner configuration is stale or broken",
		})
	}
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
