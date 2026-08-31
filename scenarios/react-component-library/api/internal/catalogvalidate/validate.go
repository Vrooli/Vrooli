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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"react-component-library/internal/assetrung"
	"react-component-library/internal/gates"

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
	findings = append(findings, blockingRunnerFindings(v.RepoRoot, catalogDir)...)
	findings = append(findings, implementationIdentityFindings(v.RepoRoot)...)
	findings = append(findings, vacuousAllowlistFindings(v.RepoRoot)...)
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

// blockingRunnerFindings keeps the catalog's blocking contract honest at the
// boundary where declarations are loaded. A declaration-only gate may remain
// advisory, but it cannot claim to block a release when the executable gate
// registry has no in-process runner for it.
func blockingRunnerFindings(repoRoot, catalogDir string) []Finding {
	data, err := os.ReadFile(filepath.Join(catalogDir, "config.json"))
	if err != nil {
		return nil
	}
	var config struct {
		Gates []struct {
			ID       string `json:"id"`
			Blocking bool   `json:"blocking"`
		} `json:"gates"`
	}
	if json.Unmarshal(data, &config) != nil {
		return nil
	}
	var findings []Finding
	for _, gate := range config.Gates {
		definition, ok := gates.Lookup(gate.ID)
		if gate.Blocking && (!ok || definition.Run == nil) {
			findings = append(findings, Finding{
				Code: "catalog.blocking_gate_without_runner", Severity: "error",
				Location: rel(repoRoot, filepath.Join(catalogDir, "config.json")),
				Message:  fmt.Sprintf("blocking gate %q has no in-process runner; add one or set blocking to false and document the advisory gate", gate.ID),
			})
		}
	}
	return findings
}

// implementationIdentityFindings closes the join between authored library
// manifests and the desired-state catalog. An implementation may be outside
// the catalog, but only when that choice is explicit in its manifest.
func implementationIdentityFindings(repoRoot string) []Finding {
	libraryRoot := filepath.Join(repoRoot, "scenarios", "react-component-library", "library")
	paths, _ := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	var findings []Finding
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Code: "catalog.implementation_manifest_unreadable", Severity: "error", Location: rel(repoRoot, path), Message: err.Error()})
			continue
		}
		var manifest struct {
			CatalogID    *string `json:"catalogId"`
			Supplemental bool    `json:"supplemental"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		if manifest.CatalogID != nil && strings.TrimSpace(*manifest.CatalogID) != "" {
			continue
		}
		if manifest.Supplemental {
			continue
		}
		findings = append(findings, Finding{
			Code:     "catalog.implementation_identity_missing",
			Severity: "error",
			Location: rel(repoRoot, path),
			Message:  "implementation must declare a catalogId or supplemental=true",
		})
	}
	return findings
}

type vacuousAllowlistEntry struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type vacuousAllowlistFile struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Entries       []vacuousAllowlistEntry `json:"entries"`
}

const vacuousAllowlistRelativePath = "scenarios/react-component-library/library/vacuous-allowlist.json"

// vacuousAllowlistFindings enforces the two-way ratchet around legacy
// contracts. The committed file is the comparison authority; a missing HEAD
// copy is allowed only while the allowlist is first being introduced.
func vacuousAllowlistFindings(repoRoot string) []Finding {
	path := filepath.Join(repoRoot, vacuousAllowlistRelativePath)
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: "catalog.vacuous_allowlist_unreadable", Severity: "error", Location: vacuousAllowlistRelativePath, Message: err.Error()}}
	}
	current, findings := parseVacuousAllowlist(vacuousAllowlistRelativePath, data)
	if len(findings) > 0 {
		return findings
	}
	for _, entry := range current.Entries {
		contractPath := filepath.Join(scenarioRoot, filepath.FromSlash(entry.Path))
		if _, err := os.Stat(contractPath); err != nil {
			if isEvictedContract(scenarioRoot, entry.Path) {
				// Eviction removes the working-tree projection but preserves
				// the version identity and contract mirror in the registry.
				// The allowlist entry is still valid; it is not an unknown
				// contract merely because its bytes are cold.
				continue
			}
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_unknown_entry", Severity: "error", Location: entry.Path, Message: "allowlisted contract does not exist"})
		}
	}
	baselineData, err := gitShow(repoRoot, "HEAD:"+vacuousAllowlistRelativePath)
	if err != nil {
		return findings
	}
	baseline, baselineFindings := parseVacuousAllowlist(vacuousAllowlistRelativePath, baselineData)
	findings = append(findings, baselineFindings...)
	if len(baselineFindings) > 0 {
		return findings
	}
	for _, path := range allowlistGrowth(current.Entries, baseline.Entries) {
		findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_growth", Severity: "error", Location: path, Message: "legacy vacuous allowlist may shrink but may not gain entries"})
	}
	for _, entry := range current.Entries {
		if !allowlistPaths(baseline.Entries)[entry.Path] {
			continue
		}
		if allowlistSourceChanged(repoRoot, filepath.ToSlash(filepath.Join("scenarios/react-component-library", entry.Path))) {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_source_changed", Severity: "error", Location: entry.Path, Message: "source changed while its vacuous contract remains allowlisted; remove the entry and add an implemented machine claim"})
		}
	}
	return findings
}

func isEvictedContract(scenarioRoot, relativePath string) bool {
	clean := filepath.Clean(filepath.FromSlash(relativePath))
	if filepath.Base(clean) != "experience-contract.json" {
		return false
	}
	versionDir := filepath.Dir(clean)
	version := filepath.Base(versionDir)
	componentDir := filepath.Dir(filepath.Dir(versionDir))
	manifestPath := filepath.Join(scenarioRoot, componentDir, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var manifest struct {
		EvictedVersions []string `json:"evictedVersions"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	for _, evicted := range manifest.EvictedVersions {
		if strings.TrimSpace(evicted) == version {
			return true
		}
	}
	return false
}

func parseVacuousAllowlist(location string, data []byte) (vacuousAllowlistFile, []Finding) {
	var document vacuousAllowlistFile
	if err := json.Unmarshal(data, &document); err != nil {
		return document, []Finding{{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: err.Error()}}
	}
	if document.SchemaVersion != 1 {
		return document, []Finding{{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: "schemaVersion must be 1"}}
	}
	var findings []Finding
	previous := ""
	seen := map[string]bool{}
	for index := range document.Entries {
		entry := &document.Entries[index]
		entry.Path = filepath.ToSlash(strings.TrimSpace(entry.Path))
		entry.Reason = strings.TrimSpace(entry.Reason)
		if entry.Path == "" || entry.Reason == "" {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: "every entry requires a path and written reason"})
		}
		if seen[entry.Path] {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_duplicate", Severity: "error", Location: entry.Path, Message: "allowlist entry is duplicated"})
		}
		seen[entry.Path] = true
		if previous != "" && entry.Path < previous {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_unsorted", Severity: "error", Location: location, Message: "allowlist entries must be sorted by path"})
		}
		previous = entry.Path
	}
	return document, findings
}

func allowlistGrowth(current, baseline []vacuousAllowlistEntry) []string {
	baselinePaths := allowlistPaths(baseline)
	var growth []string
	for _, entry := range current {
		if !baselinePaths[entry.Path] {
			growth = append(growth, entry.Path)
		}
	}
	sort.Strings(growth)
	return growth
}

func allowlistPaths(entries []vacuousAllowlistEntry) map[string]bool {
	out := make(map[string]bool, len(entries))
	for _, entry := range entries {
		out[filepath.ToSlash(strings.TrimSpace(entry.Path))] = true
	}
	return out
}

func gitShow(repoRoot, ref string) ([]byte, error) {
	command := exec.Command("git", "-C", repoRoot, "show", ref)
	return command.Output()
}

func allowlistSourceChanged(repoRoot, contractPath string) bool {
	versionDir := filepath.Dir(filepath.Join(repoRoot, filepath.FromSlash(contractPath)))
	paths, _ := filepath.Glob(filepath.Join(versionDir, "*.tsx"))
	paths2, _ := filepath.Glob(filepath.Join(versionDir, "*.ts"))
	paths = append(paths, paths2...)
	for _, path := range paths {
		relative := rel(repoRoot, path)
		command := exec.Command("git", "-C", repoRoot, "diff", "--quiet", "HEAD", "--", relative)
		if err := command.Run(); err != nil {
			return true
		}
	}
	return false
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
		Slot   string `json:"slot"`
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
	Domains []domainConfig `json:"domains"`
	Gates   []gateConfig   `json:"gates"`
}

type domainConfig struct {
	ID          string `json:"id"`
	Order       int    `json:"order"`
	Description string `json:"description"`
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
	findings = append(findings, domainOrderFindings(repoRoot, catalogDir)...)
	findings = append(findings, slotVocabularyFindings(repoRoot, docs)...)
	for _, doc := range docs {
		if _, err := assetrung.Of(doc.Asset.Kind); err != nil {
			findings = append(findings, Finding{Code: "catalog.unknown_kind", Severity: "error", Location: doc.Asset.ID, Message: fmt.Sprintf("asset %q has unrecognized kind %q", doc.Asset.ID, doc.Asset.Kind)})
			continue
		}
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

// slotVocabularyFindings keeps catalog placement declarative. Catalog assets
// target the template's published UI-manifest vocabulary; a new slot must be
// added to that manifest before an asset can claim it. This is intentionally a
// set-containment check, so adding a template slot never requires changing a
// catalog validator allow-list.
func slotVocabularyFindings(repoRoot string, docs []assetDoc) []Finding {
	path := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "ui", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return []Finding{{Code: "catalog.slot_manifest_unreadable", Severity: "error", Location: rel(repoRoot, path), Message: err.Error()}}
	}
	var manifest struct {
		Slots map[string]json.RawMessage `json:"slots"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return []Finding{{Code: "catalog.slot_manifest_invalid", Severity: "error", Location: rel(repoRoot, path), Message: err.Error()}}
	}
	findings := make([]Finding, 0)
	for _, doc := range docs {
		slot := strings.TrimSpace(doc.Asset.Slot)
		if slot == "" {
			continue
		}
		if _, ok := manifest.Slots[slot]; ok {
			continue
		}
		findings = append(findings, Finding{
			Code:     "catalog.slot_not_in_ui_manifest",
			Severity: "error",
			Location: doc.Asset.ID,
			Message:  fmt.Sprintf("catalog slot %q is not declared by templates/scenarios/react-vite/ui/manifest.json", slot),
		})
	}
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
	rung, err := assetrung.Of(kind)
	if err != nil {
		return -1
	}
	return int(rung)
}

func domainOrderFindings(repoRoot, catalogDir string) []Finding {
	data, err := os.ReadFile(filepath.Join(catalogDir, "config.json"))
	if err != nil {
		return nil
	}
	var config catalogConfig
	if json.Unmarshal(data, &config) != nil {
		return nil
	}
	byOrder := map[int][]domainConfig{}
	var findings []Finding
	for _, domain := range config.Domains {
		if domain.Order <= 0 {
			findings = append(findings, Finding{Code: "catalog.domain_order_invalid", Severity: "error", Location: rel(repoRoot, filepath.Join(catalogDir, "config.json")), Message: fmt.Sprintf("domain %q must declare an order greater than zero", domain.ID)})
		}
		byOrder[domain.Order] = append(byOrder[domain.Order], domain)
	}
	location := rel(repoRoot, filepath.Join(catalogDir, "config.json"))
	for order, domains := range byOrder {
		if order <= 0 || len(domains) < 2 {
			continue
		}
		for _, domain := range domains {
			findings = append(findings, Finding{Code: "catalog.domain_order_duplicate", Severity: "error", Location: location, Message: fmt.Sprintf("domain %q shares order %d with another domain", domain.ID, order)})
		}
	}
	return findings
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
