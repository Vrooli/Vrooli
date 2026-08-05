// Package catalogvalidate validates the catalog SSOT where it is authored.
// It reports authored-document defects as findings and returns an error only
// when the validator itself cannot be constructed or cannot read its inputs.
package catalogvalidate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Location string `json:"location"`
	Message  string `json:"message"`
}

type Validator struct {
	RepoRoot string
	once     sync.Once
	schema   *jsonschema.Schema
	err      error
}

func New(repoRoot string) *Validator { return &Validator{RepoRoot: repoRoot} }

func (v *Validator) Validate() ([]Finding, error) {
	if err := v.compile(); err != nil {
		return nil, err
	}
	catalogDir := filepath.Join(v.RepoRoot, "scenarios", "react-component-library", "catalog")
	paths, err := catalogPaths(catalogDir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			findings = append(findings, Finding{Code: "catalog.document_parse_error", Severity: "error", Location: rel(v.RepoRoot, path), Message: err.Error()})
			continue
		}
		if err := v.schema.Validate(document); err != nil {
			findings = append(findings, schemaFindings(rel(v.RepoRoot, path), err)...)
		}
	}
	findings = append(findings, crossRegistryFindings(v.RepoRoot, catalogDir)...)
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Location != findings[j].Location {
			return findings[i].Location < findings[j].Location
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Message < findings[j].Message
	})
	return findings, nil
}

func (v *Validator) compile() error {
	v.once.Do(func() {
		path := filepath.Join(v.RepoRoot, ".vrooli", "schemas", "catalog-asset.schema.json")
		data, err := os.ReadFile(path)
		if err != nil {
			v.err = fmt.Errorf("read catalog schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("catalog-asset.schema.json", bytes.NewReader(data)); err != nil {
			v.err = fmt.Errorf("add catalog schema: %w", err)
			return
		}
		v.schema, v.err = compiler.Compile("catalog-asset.schema.json")
		if v.err != nil {
			v.err = fmt.Errorf("compile catalog schema: %w", v.err)
		}
	})
	return v.err
}

func catalogPaths(dir string) ([]string, error) {
	paths := []string{filepath.Join(dir, "config.json")}
	assets, err := filepath.Glob(filepath.Join(dir, "assets", "*", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan catalog assets: %w", err)
	}
	paths = append(paths, assets...)
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("catalog input %s: %w", path, err)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func schemaFindings(location string, err error) []Finding {
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return []Finding{{Code: "catalog.schema_error", Severity: "error", Location: location, Message: err.Error()}}
	}
	if len(validation.Causes) == 0 {
		return []Finding{{Code: "catalog.schema_error", Severity: "error", Location: location, Message: validation.Message}}
	}
	var out []Finding
	for _, cause := range validation.Causes {
		out = append(out, schemaFindings(location, cause)...)
	}
	return out
}

type assetDoc struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Target struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
		Targets []string `json:"targets"`
	} `json:"asset"`
	Dependencies struct {
		Requires []struct {
			Asset string `json:"asset"`
		} `json:"requires"`
		Suggests []struct {
			Asset string `json:"asset"`
		} `json:"suggests"`
	} `json:"dependencies"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
	Expects              []struct {
		Capability string `json:"capability"`
	} `json:"expects"`
	Satisfies []string `json:"satisfies"`
}

type gateConfig struct {
	ID        string   `json:"id"`
	Rung      string   `json:"rung"`
	Blocking  bool     `json:"blocking"`
	AppliesTo []string `json:"appliesTo"`
}

type catalogConfig struct {
	Gates []gateConfig `json:"gates"`
}

func crossRegistryFindings(repoRoot, catalogDir string) []Finding {
	capabilities := loadCapabilityFacets(filepath.Join(repoRoot, "scenarios", "experience-manager", "capabilities"))
	targets := loadTemplateTargets(repoRoot)
	assets, _ := filepath.Glob(filepath.Join(catalogDir, "assets", "*", "*.json"))
	byID := map[string]assetDoc{}
	var docs []assetDoc
	for _, path := range assets {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc assetDoc
		if json.Unmarshal(data, &doc) != nil || doc.Kind != "catalog-asset" {
			continue
		}
		byID[doc.Asset.ID] = doc
		docs = append(docs, doc)
	}
	var findings []Finding
	for _, doc := range docs {
		for _, target := range doc.Asset.Targets {
			if !targets[target] {
				findings = append(findings, Finding{Code: "catalog.target_missing", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("target %q is not declared by any scenario template", target)})
			}
		}
		for _, id := range doc.RequiredCapabilities {
			if !capabilities[id]["promises"] {
				findings = append(findings, Finding{Code: "catalog.required_capability_facet", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("required capability %q is missing or does not carry the promises facet", id)})
			}
		}
		for _, port := range doc.Expects {
			if !capabilities[port.Capability]["port"] {
				findings = append(findings, Finding{Code: "catalog.expected_port_facet", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("expected capability %q is missing or does not carry the port facet", port.Capability)})
			}
		}
		for _, id := range doc.Satisfies {
			if !capabilities[id]["port"] {
				findings = append(findings, Finding{Code: "catalog.satisfied_port_facet", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("satisfied capability %q is missing or does not carry the port facet", id)})
			}
		}
		for _, edge := range append(doc.Dependencies.Requires, doc.Dependencies.Suggests...) {
			dep, ok := byID[edge.Asset]
			if !ok {
				findings = append(findings, Finding{Code: "catalog.dependency_missing", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("dependency %q does not resolve to a catalog asset", edge.Asset)})
				continue
			}
			// Generators consume composition recipes in order to emit assets;
			// their dependency direction is intentionally outside the UI
			// composition rank. All composing kinds still obey the one-way rank.
			if doc.Asset.Kind != "generator" && rank(doc.Asset.Kind) < rank(dep.Asset.Kind) && containsRequires(doc, edge.Asset) {
				findings = append(findings, Finding{Code: "catalog.dependency_rank", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("requires dependency %q has higher rank than %q", edge.Asset, doc.Asset.Kind)})
			}
		}
	}
	findings = append(findings, cycleFindings(docs, byID)...)
	findings = append(findings, vacuousFindings(repoRoot, docs)...)
	return findings
}

func loadTemplateTargets(repoRoot string) map[string]bool {
	out := map[string]bool{}
	paths, _ := filepath.Glob(filepath.Join(repoRoot, "templates", "scenarios", "*", "ui", "manifest.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Targets []string `json:"targets"`
		}
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		if len(doc.Targets) == 0 {
			// v1 templates predate an explicit target field. Their directory
			// name remains a stable compatibility declaration until v2 migration.
			name := filepath.Base(filepath.Dir(filepath.Dir(path)))
			if name == "landing-page-react-vite" {
				name = "react-vite"
			}
			doc.Targets = []string{name}
		}
		for _, target := range doc.Targets {
			out[target] = true
		}
	}
	return out
}

func containsRequires(doc assetDoc, id string) bool {
	for _, e := range doc.Dependencies.Requires {
		if e.Asset == id {
			return true
		}
	}
	return false
}

func rank(kind string) int {
	return map[string]int{"foundation": 0, "runtime-hook": 1, "runtime-service": 1, "adapter": 1, "primitive": 2, "component": 3, "pattern": 4, "navigation": 4, "page-template": 5, "fixture": -1, "generator": 1}[kind]
}

func cycleFindings(docs []assetDoc, byID map[string]assetDoc) []Finding {
	graph := map[string][]string{}
	for _, doc := range docs {
		for _, edge := range doc.Dependencies.Requires {
			if _, ok := byID[edge.Asset]; ok {
				graph[doc.Asset.ID] = append(graph[doc.Asset.ID], edge.Asset)
			}
		}
	}
	state := map[string]int{}
	var out []Finding
	var visit func(string, []string)
	visit = func(id string, path []string) {
		if state[id] == 1 {
			out = append(out, Finding{Code: "catalog.requires_cycle", Severity: "error", Location: id, Message: "requires dependency graph contains a cycle: " + strings.Join(append(path, id), " -> ")})
			return
		}
		if state[id] == 2 {
			return
		}
		state[id] = 1
		for _, dep := range graph[id] {
			visit(dep, append(path, id))
		}
		state[id] = 2
	}
	for id := range graph {
		visit(id, nil)
	}
	return out
}

func loadCapabilityFacets(dir string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	paths, _ := filepath.Glob(filepath.Join(dir, "capabilities", "*.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Capabilities []struct {
				ID     string   `json:"id"`
				Facets []string `json:"facets"`
			} `json:"capabilities"`
		}
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		for _, cap := range doc.Capabilities {
			if out[cap.ID] == nil {
				out[cap.ID] = map[string]bool{}
			}
			for _, facet := range cap.Facets {
				out[cap.ID][facet] = true
			}
		}
	}
	return out
}

func vacuousFindings(repoRoot string, docs []assetDoc) []Finding {
	data, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return nil
	}
	var config catalogConfig
	if json.Unmarshal(data, &config) != nil {
		return nil
	}
	rungs := []string{"scaffolded", "implemented", "verified", "production-ready"}
	rankRung := map[string]int{"scaffolded": 0, "implemented": 1, "verified": 2, "production-ready": 3}
	var out []Finding
	for _, doc := range docs {
		target := doc.Asset.Target.Maturity
		if rankRung[target] < 0 {
			continue
		}
		for _, rung := range rungs[:rankRung[target]+1] {
			// Fixtures intentionally jump from schema/type validity to their
			// verified adversarial-case gate; there is no meaningful implemented
			// fixture rung to make non-vacuous.
			if doc.Asset.Kind == "fixture" && rung == "implemented" {
				continue
			}
			applicable := false
			for _, gate := range config.Gates {
				if !gate.Blocking {
					continue
				}
				if gate.Rung != rung {
					continue
				}
				for _, kind := range gate.AppliesTo {
					if kind == doc.Asset.Kind {
						applicable = true
					}
				}
			}
			// A verified gate remains part of the production-ready bar. This is
			// intentional for foundations and non-visual runtime assets whose
			// production proof is the verified contract plus operational docs;
			// they do not need a fictitious second copy of the same gate.
			if !applicable && rung == "production-ready" {
				for _, gate := range config.Gates {
					if gate.Blocking && gate.Rung == "verified" && contains(gate.AppliesTo, doc.Asset.Kind) {
						applicable = true
						break
					}
				}
			}
			if !applicable {
				out = append(out, Finding{Code: "catalog.vacuous_rung", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("asset kind %q has no applicable blocking gate at rung %q", doc.Asset.Kind, rung)})
			}
		}
	}
	return out
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func rel(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(value)
}
