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
func ValidateAPI(root string) ([]Finding, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, asset := range assets {
		if asset.API == nil {
			continue
		}
		manifest, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		text := string(data)
		for group, values := range asset.API.Variants {
			for _, value := range values {
				if !strings.Contains(text, value) {
					findings = append(findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared %s variant %q is absent from %s", group, value, manifest)})
				}
			}
		}
		for _, value := range asset.API.Modes {
			if !strings.Contains(text, value) {
				findings = append(findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared mode %q is absent from %s", value, manifest)})
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
				findings = append(findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared part %q is absent from %s", partID, manifest)})
			}
		}
	}
	return findings, nil
}

func implementationSource(root, catalogID string) (string, string, bool, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "*", "component.json"))
	if err != nil {
		return "", "", false, err
	}
	paths2, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "hooks", "*", "component.json"))
	if err != nil {
		return "", "", false, err
	}
	paths = append(paths, paths2...)
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
		source := filepath.Join(rootDir, doc.Latest, filepath.Base(rootDir)+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches, _ := filepath.Glob(filepath.Join(rootDir, doc.Latest, "*.tsx"))
			if len(matches) == 0 {
				return manifest, "", false, nil
			}
			source = matches[0]
		}
		return manifest, source, true, nil
	}
	return "", "", false, nil
}

var pxValue = regexp.MustCompile(`--space-[a-z0-9-]+\s*:\s*([0-9.]+)px`)
var literalDimension = regexp.MustCompile(`(?:\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap|w|h)-[0-9]+(?:\.[0-9]+)?\b|\[[0-9.]+px\])`)

// ValidateTokens checks the shared ramp contract in every design kit and
// rejects non-grid spacing declarations.
func ValidateTokens(root string) ([]Finding, error) {
	paths, err := filepath.Glob(filepath.Join(root, "templates", "design", "*", "adapters", "react-vite-tailwind", "tokens.css"))
	if err != nil {
		return nil, err
	}
	shared := []string{"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl", "text-display", "text-title", "text-heading", "text-body", "elev-flat", "elev-raised", "layer-base", "layer-modal", "dur-instant"}
	var findings []Finding
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(data)
		kit := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
		for _, token := range shared {
			if !strings.Contains(text, "--"+token) {
				findings = append(findings, Finding{"catalog.tokens_missing", "foundations.tokens", fmt.Sprintf("%s does not declare shared token --%s", kit, token)})
			}
		}
		for _, match := range pxValue.FindAllStringSubmatch(text, -1) {
			value, _ := strconv.ParseFloat(match[1], 64)
			if int(value)%4 != 0 {
				findings = append(findings, Finding{"catalog.tokens_grid", "foundations.tokens", fmt.Sprintf("%s spacing token is not on the 4px grid: %spx", kit, match[1])})
			}
		}
	}
	for _, kind := range []string{"components", "hooks"} {
		sources, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "*.tsx"))
		if err != nil {
			return nil, err
		}
		for _, path := range sources {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			if match := literalDimension.FindString(string(data)); match != "" {
				findings = append(findings, Finding{"catalog.tokens_literal", filepath.Base(filepath.Dir(filepath.Dir(path))), fmt.Sprintf("implementation contains literal dimension %q; use a declared semantic token", match)})
			}
		}
	}
	return findings, nil
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func ValidateLifecycle(root string) ([]Finding, error) {
	var findings []Finding
	for _, kind := range []string{"hooks", "components"} {
		paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "*.tsx"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			text := string(data)
			if strings.Contains(text, "addEventListener") && !strings.Contains(text, "removeEventListener") {
				findings = append(findings, Finding{"catalog.lifecycle_cleanup", filepath.Base(filepath.Dir(filepath.Dir(path))), "adds an event listener without a matching removal"})
			}
			if strings.Contains(text, "new MutationObserver") && !strings.Contains(text, ".disconnect(") {
				findings = append(findings, Finding{"catalog.lifecycle_cleanup", filepath.Base(filepath.Dir(filepath.Dir(path))), "creates an observer without disconnect cleanup"})
			}
			if strings.Contains(text, "window.") && strings.Contains(text, "export") && strings.Contains(text, "typeof window") == false {
				findings = append(findings, Finding{"catalog.lifecycle_ssr", filepath.Base(filepath.Dir(filepath.Dir(path))), "accesses window without an SSR guard"})
			}
		}
	}
	return findings, nil
}

func ValidateFixtures(root string) ([]Finding, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, asset := range assets {
		if asset.Asset.Kind != "fixture" || asset.Fixture == nil {
			continue
		}
		if len(asset.Fixture.DataShapes) == 0 {
			findings = append(findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture declares no adversarial data shapes"})
		}
		if !contains(asset.Fixture.DataShapes, "failure") && !contains(asset.Fixture.DataShapes, "overflow") {
			findings = append(findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture must include failure or overflow data"})
		}
	}
	return findings, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
