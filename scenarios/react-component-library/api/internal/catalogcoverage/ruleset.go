package catalogcoverage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/gates"
)

// RuleSetDigest is the cache fingerprint for an asset's resolved rules.
func RuleSetDigest(root, assetID string) (string, error) {
	bindings, err := ResolveRuleSet(root, assetID)
	if err != nil {
		return "", err
	}
	inputs := make(map[string][]string)
	for _, definition := range gates.Definitions() {
		inputs[definition.ID] = append([]string(nil), definition.DeterminismInputs...)
	}
	config, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		Bindings []RuleBinding
		Inputs   map[string][]string
	}{bindings, inputs})
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write(payload)
	_, _ = hash.Write(config)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// AnnotateFindings applies the resolved rule binding at the engine boundary.
// Runners report defects; they do not choose or invent rule provenance.
func AnnotateFindings(root, gate string, result *gates.Result) error {
	if result == nil {
		return nil
	}
	for i := range result.Findings {
		if err := annotateFinding(root, gate, &result.Findings[i]); err != nil {
			return err
		}
	}
	for i := range result.InformationalFindings {
		if err := annotateFinding(root, gate, &result.InformationalFindings[i]); err != nil {
			return err
		}
	}
	for i := range result.RunnerError {
		if err := annotateFinding(root, gate, &result.RunnerError[i]); err != nil {
			return err
		}
	}
	return nil
}

func annotateFinding(root, gate string, finding *gates.Finding) error {
	if finding.AssetID == "" || isNonCatalogObservation(finding.AssetID) {
		finding.RuleSource = gates.RuleSourceCorpus
		finding.RuleDeclaredIn = filepath.ToSlash(filepath.Join("scenarios", "react-component-library", "catalog", "config.json"))
		return nil
	}
	// Corpus-scoped runners may include the most useful source asset in their
	// diagnostic, but that asset is context rather than the rule's attribution
	// boundary. Keep the finding corpus-scoped instead of requiring an
	// applicability binding that the corpus gate intentionally does not have.
	if definition, ok := gates.Lookup(gate); ok && definition.CorpusScoped {
		finding.RuleSource = gates.RuleSourceCorpus
		finding.RuleDeclaredIn = filepath.ToSlash(filepath.Join("scenarios", "react-component-library", "catalog", "config.json"))
		return nil
	}
	// Several corpus validators naturally report the stable library id while
	// rule applicability is declared against the catalog id. Normalize that
	// identity at the annotation boundary instead of turning a valid finding
	// into a runner error.
	if strings.HasPrefix(finding.AssetID, "react-component-library:") {
		implementations, loadErr := LoadImplementations(filepath.Join(root, "scenarios", "react-component-library", "library"))
		if loadErr != nil {
			return loadErr
		}
		for _, implementation := range implementations {
			if implementation.LibraryID != finding.AssetID {
				continue
			}
			if implementation.CatalogID != "" && implementation.CatalogID != finding.AssetID {
				finding.AssetID = implementation.CatalogID
				break
			}
			catalogAssets, catalogErr := LoadCatalog(filepath.Join(root, "scenarios", "react-component-library", "catalog"))
			if catalogErr != nil {
				return catalogErr
			}
			name := strings.TrimPrefix(finding.AssetID, "react-component-library:")
			for _, asset := range catalogAssets {
				if asset.Name == name {
					finding.AssetID = asset.ID
					break
				}
			}
			break
		}
	}
	bindings, err := ResolveRuleSet(root, finding.AssetID)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.GateID == gate {
			finding.RuleSource = gates.RuleSource(binding.Source)
			finding.RuleDeclaredIn = binding.DeclaredIn
			return nil
		}
	}
	// A legacy/support identity can be a useful diagnostic without being an
	// applicable catalog asset. Keep the observation corpus-scoped rather than
	// aborting the entire gate matrix on annotation bookkeeping.
	finding.AssetID = "__corpus__.unresolved-" + gate
	finding.RuleSource = gates.RuleSourceCorpus
	finding.RuleDeclaredIn = filepath.ToSlash(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	return nil
}

// RuleSource identifies the declaration layer that selected a gate.
type RuleSource string

const (
	RuleSourceUniversal RuleSource = "universal"
	RuleSourceKind      RuleSource = "kind"
	RuleSourceAsset     RuleSource = "asset"
	RuleSourceCorpus    RuleSource = "corpus"
)

// RuleBinding is the resolved, provenance-carrying form of one applicable gate.
type RuleBinding struct {
	GateID     string
	Source     RuleSource
	DeclaredIn string
}

// ResolveRuleSet resolves applicability from catalog declarations. Gate
// runners do not get to repeat this decision. An asset-level qualityGates
// entry is additive; qualityGateOptOuts remove a kind default only when the
// declaration supplies the required reason.
func ResolveRuleSet(root, assetID string) ([]RuleBinding, error) {
	catalogRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	assets, err := LoadCatalog(catalogRoot)
	if err != nil {
		return nil, err
	}
	var asset Asset
	found := false
	for _, candidate := range assets {
		if candidate.ID == assetID {
			asset = candidate
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("catalog asset %q not found", assetID)
	}
	definitions, err := LoadGateDefinitions(filepath.Join(catalogRoot, "config.json"))
	if err != nil {
		return nil, err
	}
	manifest, err := loadAssetRuleOverrides(catalogRoot, assetID)
	if err != nil {
		return nil, err
	}
	optOut := make(map[string]bool, len(manifest.OptOuts))
	for _, gate := range manifest.OptOuts {
		optOut[gate.Gate] = true
	}
	added := make(map[string]bool, len(manifest.Gates))
	for _, gate := range manifest.Gates {
		added[gate] = true
	}

	bindings := make([]RuleBinding, 0)
	for _, definition := range definitions {
		if optOut[definition.ID] {
			continue
		}
		if added[definition.ID] {
			bindings = append(bindings, RuleBinding{GateID: definition.ID, Source: RuleSourceAsset, DeclaredIn: assetDeclarationPath(catalogRoot, assetID)})
			continue
		}
		if !containsKind(definition.AppliesTo, asset.Kind) {
			continue
		}
		source := RuleSourceKind
		if appliesToAllKinds(definition.AppliesTo) {
			source = RuleSourceUniversal
		}
		if definition.Attribution == "corpus" {
			source = RuleSourceCorpus
		}
		bindings = append(bindings, RuleBinding{GateID: definition.ID, Source: source, DeclaredIn: filepath.Join("scenarios", "react-component-library", "catalog", "config.json")})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].GateID < bindings[j].GateID })
	return bindings, nil
}

type assetRuleOverrides struct {
	Gates   []string `json:"qualityGates"`
	OptOuts []struct {
		Gate string `json:"gate"`
	} `json:"qualityGateOptOuts"`
}

func loadAssetRuleOverrides(catalogRoot, assetID string) (assetRuleOverrides, error) {
	path := ""
	matches, _ := filepath.Glob(filepath.Join(catalogRoot, "assets", "*", "*.json"))
	for _, candidate := range matches {
		data, readErr := os.ReadFile(candidate)
		if readErr != nil {
			return assetRuleOverrides{}, readErr
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.Asset.ID == assetID {
			path = candidate
			break
		}
	}
	if path == "" {
		return assetRuleOverrides{}, fmt.Errorf("catalog declaration for %q not found", assetID)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return assetRuleOverrides{}, err
	}
	var doc assetRuleOverrides
	if err := json.Unmarshal(data, &doc); err != nil {
		return assetRuleOverrides{}, fmt.Errorf("parse asset rule overrides: %w", err)
	}
	return doc, nil
}

func assetDeclarationPath(catalogRoot, assetID string) string {
	matches, _ := filepath.Glob(filepath.Join(catalogRoot, "assets", "*", "*.json"))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.Asset.ID == assetID {
			return filepath.ToSlash(filepath.Join("scenarios", "react-component-library", "catalog", "assets", filepath.Base(filepath.Dir(path)), filepath.Base(path)))
		}
	}
	return ""
}

func appliesToAllKinds(kinds []string) bool {
	canonical := []string{"foundation", "runtime-hook", "runtime-service", "adapter", "primitive", "component", "pattern", "page-template", "navigation", "fixture", "generator"}
	for _, kind := range canonical {
		if !containsKind(kinds, kind) {
			return false
		}
	}
	return true
}
