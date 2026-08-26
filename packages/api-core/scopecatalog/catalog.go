// Package scopecatalog derives the authorization vocabulary from CLI
// governance metadata. It deliberately has no dependency on the authenticator
// or on any scenario package.
package scopecatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Effect is the ordered governance effect used to select a safe default for
// omitted RPCs. Higher values are more restrictive.
type Effect string

const (
	EffectRead        Effect = "read"
	EffectWrite       Effect = "write"
	EffectDestructive Effect = "destructive"
)

// Scope describes one command-derived authorization scope.
type Scope struct {
	Scenario    string   `json:"scenario"`
	Value       string   `json:"scope"`
	Effect      Effect   `json:"effect"`
	RunEligible bool     `json:"run_eligible"`
	Permissions []string `json:"permissions,omitempty"`
	Command     string   `json:"command"`
	Service     string   `json:"service,omitempty"`
	Method      string   `json:"method,omitempty"`
}

// Verb returns the argv vocabulary used by remote command admission. Project
// commands run through the vrooli CLI directly; scenario commands are prefixed
// by their owning scenario CLI name.
func (s Scope) Verb() string {
	command := strings.ReplaceAll(strings.TrimSpace(s.Command), "/", " ")
	if command == "" {
		return ""
	}
	if s.Scenario == ProjectManifestIdentity {
		return command
	}
	return strings.TrimSpace(s.Scenario + " " + command)
}

// OmittedResolution records how an intentionally unbound RPC gets a
// most-restrictive scenario scope.
type OmittedResolution struct {
	Scenario string `json:"scenario"`
	Service  string `json:"service"`
	Method   string `json:"method"`
	Scope    string `json:"scope"`
	Reason   string `json:"reason"`
}

// RPCMethod identifies a method enumerated from a proto service descriptor.
// Callers can use Reconcile to compare descriptor enumeration with this
// catalog without making the catalog depend on generated proto packages.
type RPCMethod struct {
	Scenario string
	Service  string
	Method   string
}

// CompletenessReport describes the reconciliation between proto enumeration
// and the derived catalog.
type CompletenessReport struct {
	Enumerated int
	Resolved   int
	Missing    []RPCMethod
}

// Catalog is the build-time artifact consumed by relying parties. It is
// derived from manifests; Build never writes beneath scenarios/.
type Catalog struct {
	ManifestCount               int                 `json:"manifest_count"`
	GovernedCommandCount        int                 `json:"governed_command_count"`
	RPCScopeCount               int                 `json:"rpc_scope_count"`
	OmittedCount                int                 `json:"omitted_count"`
	MostRestrictiveDefaultCount int                 `json:"most_restrictive_default_count"`
	Scopes                      []Scope             `json:"scopes"`
	OmittedResolutions          []OmittedResolution `json:"omitted_resolutions"`
	ScenariosWithoutManifest    []string            `json:"scenarios_without_manifest,omitempty"`
	InvalidManifests            []InvalidManifest   `json:"invalid_manifests,omitempty"`
}

// InvalidManifest records a scenario CLI manifest that was intentionally
// excluded from a resilient catalog. The path and reason are surfaced so a
// degraded control plane can remain usable without hiding the defect.
type InvalidManifest struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ProjectManifestIdentity is the stable scope namespace for the root CLI.
// Scenario manifests use their own names, so the project CLI remains a
// separate identity even when a scenario happens to expose similarly named
// commands.
const ProjectManifestIdentity = "vrooli"

type manifest struct {
	Name    string             `json:"name"`
	Groups  []manifestGroup    `json:"groups"`
	Omitted []manifestOmission `json:"omitted,omitempty"`
}

type manifestGroup struct {
	Name     string            `json:"name"`
	Flat     bool              `json:"flat,omitempty"`
	Commands []manifestCommand `json:"commands"`
	Groups   []manifestGroup   `json:"groups,omitempty"`
}

type manifestCommand struct {
	Name       string             `json:"name"`
	Binding    manifestBinding    `json:"binding"`
	Governance manifestGovernance `json:"governance"`
}

type manifestBinding struct {
	Kind    string `json:"kind"`
	Service string `json:"service"`
	Method  string `json:"method"`
}

type manifestGovernance struct {
	Effect      string   `json:"effect"`
	RunEligible bool     `json:"run_eligible"`
	Permissions []string `json:"permissions"`
}

type manifestOmission struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Reason  string `json:"reason"`
}

// Build reads and schema-validates the project CLI manifest and every scenario
// CLI manifest, deriving their scopes and omitted-RPC defaults. The returned
// catalog is deterministic.
func Build(repoRoot string) (Catalog, error) {
	return build(repoRoot, false)
}

// BuildResilient derives the catalog while excluding invalid scenario
// manifests individually. The project manifest and catalog infrastructure
// still fail closed; only a scenario-local manifest defect is quarantined.
func BuildResilient(repoRoot string) (Catalog, error) {
	return build(repoRoot, true)
}

func build(repoRoot string, resilient bool) (Catalog, error) {
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	paths, err := manifestPaths(repoRoot)
	if err != nil {
		return Catalog{}, err
	}
	return buildWithManifestPaths(repoRoot, scenariosRoot, paths, resilient)
}

func buildWithManifestPaths(repoRoot, scenariosRoot string, paths []string, resilient bool) (Catalog, error) {
	validate, err := manifestValidator(repoRoot)
	if err != nil {
		return Catalog{}, err
	}

	catalog := Catalog{ManifestCount: len(paths)}
	manifestScenarios := make(map[string]struct{}, len(paths))
	projectManifest := filepath.Join(repoRoot, "cli", "manifest.json")
	recordInvalid := func(path string, cause error) {
		catalog.InvalidManifests = append(catalog.InvalidManifests, InvalidManifest{Path: path, Reason: cause.Error()})
		// Keep the directory out of ScenariosWithoutManifest: it has a
		// manifest, but that manifest is quarantined and already reported.
		if filepath.Clean(path) != filepath.Clean(projectManifest) {
			scenarioDir := filepath.Base(filepath.Dir(filepath.Dir(path)))
			if scenarioDir != "" {
				manifestScenarios[scenarioDir] = struct{}{}
			}
		}
	}
	quarantine := func(path string, cause error) bool {
		if !resilient || filepath.Clean(path) == filepath.Clean(projectManifest) {
			return false
		}
		recordInvalid(path, cause)
		return true
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			if quarantine(path, fmt.Errorf("read %s: %w", path, err)) {
				continue
			}
			return Catalog{}, fmt.Errorf("read %s: %w", path, err)
		}
		var document any
		if err := json.Unmarshal(raw, &document); err != nil {
			if quarantine(path, fmt.Errorf("decode %s: %w", path, err)) {
				continue
			}
			return Catalog{}, fmt.Errorf("decode %s: %w", path, err)
		}
		if err := validate.Validate(document); err != nil {
			if quarantine(path, fmt.Errorf("validate %s: %w", path, err)) {
				continue
			}
			return Catalog{}, fmt.Errorf("validate %s: %w", path, err)
		}
		var m manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			if quarantine(path, fmt.Errorf("decode manifest %s: %w", path, err)) {
				continue
			}
			return Catalog{}, fmt.Errorf("decode manifest %s: %w", path, err)
		}
		if strings.TrimSpace(m.Name) == "" {
			cause := fmt.Errorf("manifest %s has an empty name", path)
			if quarantine(path, cause) {
				continue
			}
			return Catalog{}, fmt.Errorf("manifest %s has an empty name", path)
		}
		if filepath.Clean(path) == filepath.Clean(projectManifest) {
			// The root manifest's name is the public project CLI identity. Keep
			// this explicit instead of inferring it from a filesystem directory,
			// which prevents a scenario named "vrooli" from colliding with it.
			m.Name = ProjectManifestIdentity
		} else {
			if m.Name == ProjectManifestIdentity {
				cause := fmt.Errorf("scenario manifest %s collides with project scope identity %q", path, ProjectManifestIdentity)
				if quarantine(path, cause) {
					continue
				}
				return Catalog{}, fmt.Errorf("scenario manifest %s collides with project scope identity %q", path, ProjectManifestIdentity)
			}
			manifestScenarios[m.Name] = struct{}{}
		}
		deriveManifest(&catalog, m)
	}
	for _, scenario := range scenarioDirectories(scenariosRoot) {
		if _, ok := manifestScenarios[scenario]; !ok {
			catalog.ScenariosWithoutManifest = append(catalog.ScenariosWithoutManifest, scenario)
		}
	}
	sort.Strings(catalog.ScenariosWithoutManifest)
	sort.Slice(catalog.Scopes, func(i, j int) bool {
		return scopeKey(catalog.Scopes[i]) < scopeKey(catalog.Scopes[j])
	})
	sort.Slice(catalog.OmittedResolutions, func(i, j int) bool {
		a, b := catalog.OmittedResolutions[i], catalog.OmittedResolutions[j]
		return omittedKey(a) < omittedKey(b)
	})
	sort.Slice(catalog.InvalidManifests, func(i, j int) bool {
		return catalog.InvalidManifests[i].Path < catalog.InvalidManifests[j].Path
	})
	return catalog, nil
}

// Reconcile compares descriptor-derived RPC methods with the catalog. A
// method is resolved when a connect-rpc binding or an omitted entry names the
// same (scenario, service, method) tuple.
func (c Catalog) Reconcile(methods []RPCMethod) CompletenessReport {
	resolved := make(map[string]struct{}, c.RPCScopeCount+len(c.OmittedResolutions))
	for _, scope := range c.Scopes {
		if scope.Service != "" && scope.Method != "" {
			resolved[rpcKey(RPCMethod{Scenario: scope.Scenario, Service: scope.Service, Method: scope.Method})] = struct{}{}
		}
	}
	for _, omitted := range c.OmittedResolutions {
		resolved[rpcKey(RPCMethod{Scenario: omitted.Scenario, Service: omitted.Service, Method: omitted.Method})] = struct{}{}
	}
	report := CompletenessReport{Enumerated: len(methods)}
	for _, method := range methods {
		if _, ok := resolved[rpcKey(method)]; ok {
			report.Resolved++
		} else {
			report.Missing = append(report.Missing, method)
		}
	}
	sort.Slice(report.Missing, func(i, j int) bool { return rpcKey(report.Missing[i]) < rpcKey(report.Missing[j]) })
	return report
}

// HasScope reports whether the derived catalog contains the concrete scope
// value. It is intentionally a read-only lookup over an already-built catalog;
// relying parties use it to validate their own typed capability bindings
// without declaring a second scope vocabulary.
func (c Catalog) HasScope(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, scope := range c.Scopes {
		if scope.Value == value {
			return true
		}
	}
	for _, omitted := range c.OmittedResolutions {
		if omitted.Scope == value {
			return true
		}
	}
	return false
}

// LookupVerb resolves one run-eligible argv verb to its derived catalog scope.
// Ambiguous verbs fail closed unless every matching entry requires the same
// concrete scope value.
func (c Catalog) LookupVerb(verb string) (Scope, bool) {
	verb = strings.TrimSpace(verb)
	if verb == "" {
		return Scope{}, false
	}
	var found Scope
	matched := false
	for _, scope := range c.Scopes {
		if !scope.RunEligible || scope.Verb() != verb {
			continue
		}
		if matched && found.Value != scope.Value {
			return Scope{}, false
		}
		found = scope
		matched = true
	}
	return found, matched
}

func deriveManifest(catalog *Catalog, m manifest) {
	mostRestrictive := EffectRead
	for _, group := range m.Groups {
		deriveGroup(catalog, m.Name, group, nil, &mostRestrictive)
	}
	for _, omitted := range m.Omitted {
		catalog.OmittedCount++
		catalog.MostRestrictiveDefaultCount++
		catalog.OmittedResolutions = append(catalog.OmittedResolutions, OmittedResolution{
			Scenario: m.Name, Service: omitted.Service, Method: omitted.Method,
			Scope: m.Name + ":" + string(mostRestrictive), Reason: omitted.Reason,
		})
	}
}

func deriveGroup(catalog *Catalog, scenario string, group manifestGroup, parentPath []string, mostRestrictive *Effect) {
	path := parentPath
	if !group.Flat {
		path = append(append([]string(nil), parentPath...), group.Name)
	}
	for _, command := range group.Commands {
		if command.Governance.Effect == "" {
			continue
		}
		catalog.GovernedCommandCount++
		effect := Effect(command.Governance.Effect)
		if effectRank(effect) > effectRank(*mostRestrictive) {
			*mostRestrictive = effect
		}
		commandPath := append(append([]string(nil), path...), command.Name)
		derived := Scope{
			Scenario: scenario, Value: scenario + ":" + string(effect), Effect: effect,
			RunEligible: command.Governance.RunEligible,
			Permissions: append([]string(nil), command.Governance.Permissions...),
			Command:     strings.Join(commandPath, "/"),
		}
		if command.Binding.Kind == "connect-rpc" {
			derived.Service = command.Binding.Service
			derived.Method = command.Binding.Method
			catalog.RPCScopeCount++
		}
		catalog.Scopes = append(catalog.Scopes, derived)
	}
	for _, child := range group.Groups {
		deriveGroup(catalog, scenario, child, path, mostRestrictive)
	}
}

func manifestPaths(repoRoot string) ([]string, error) {
	// Keep the project manifest in the same sorted input set as scenario
	// manifests. Derivation order is therefore stable even though the final
	// catalog is sorted independently for consumers.
	root := filepath.Join(repoRoot, "scenarios")
	paths := []string{filepath.Join(repoRoot, "cli", "manifest.json")}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "manifest.json" && filepath.Base(filepath.Dir(path)) == "cli" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find CLI manifests: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func scenarioDirectories(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "" && !strings.HasPrefix(entry.Name(), ".") {
			scenarios = append(scenarios, entry.Name())
		}
	}
	return scenarios
}

func manifestValidator(repoRoot string) (*jsonschema.Schema, error) {
	path := filepath.Join(repoRoot, ".vrooli", "schemas", "cli-manifest.schema.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CLI manifest schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("cli-manifest.schema.json", bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("compile CLI manifest schema resource: %w", err)
	}
	schema, err := compiler.Compile("cli-manifest.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile CLI manifest schema: %w", err)
	}
	return schema, nil
}

func effectRank(effect Effect) int {
	switch effect {
	case EffectDestructive:
		return 3
	case EffectWrite:
		return 2
	default:
		return 1
	}
}

func scopeKey(scope Scope) string {
	return scope.Scenario + "\x00" + scope.Command + "\x00" + scope.Service + "\x00" + scope.Method
}

func omittedKey(omitted OmittedResolution) string {
	return omitted.Scenario + "\x00" + omitted.Service + "\x00" + omitted.Method
}

func rpcKey(method RPCMethod) string {
	return method.Scenario + "\x00" + method.Service + "\x00" + method.Method
}

// WriteJSON writes the deterministic catalog artifact to an explicit target.
// It refuses a target below scenarios/ so deriving the catalog cannot mutate
// scenario source data.
func (c Catalog) WriteJSON(repoRoot, target string) error {
	if filepath.IsAbs(target) {
		target = filepath.Clean(target)
	} else {
		target = filepath.Join(repoRoot, target)
	}
	scenariosRoot, err := filepath.Abs(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return err
	}
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if absoluteTarget == scenariosRoot || strings.HasPrefix(absoluteTarget, scenariosRoot+string(filepath.Separator)) {
		return fmt.Errorf("refusing to write scope catalog under scenarios/")
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scope catalog: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(absoluteTarget), 0o755); err != nil {
		return fmt.Errorf("create catalog directory: %w", err)
	}
	if err := os.WriteFile(absoluteTarget, raw, 0o644); err != nil {
		return fmt.Errorf("write scope catalog: %w", err)
	}
	return nil
}

// Resolve reports whether held contains the required scope. Wildcards may be
// held as *, scenario:* or *:effect. Matching is exact and case-sensitive;
// whitespace is not normalized so malformed scope strings cannot grant access.
func Resolve(held []string, required string) bool {
	if strings.TrimSpace(required) == "" || required != strings.TrimSpace(required) {
		return false
	}
	parts := strings.Split(required, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, candidate := range held {
		if candidate == "*" || candidate == required || candidate == parts[0]+":*" || candidate == "*:"+parts[1] {
			return true
		}
	}
	return false
}

// Materialize expands held wildcard grants against the concrete scope names
// supplied by the caller. Entries marked human-only are excluded when
// agentEligible is false. It performs no I/O, so callers can use it at token
// mint time after loading the catalog once.
func Materialize(held []string, concrete []Scope, agentEligible bool) []string {
	result := make([]string, 0, len(concrete))
	seen := map[string]struct{}{}
	for _, scope := range concrete {
		if !agentEligible && !scope.RunEligible {
			continue
		}
		if !Resolve(held, scope.Value) {
			continue
		}
		if _, ok := seen[scope.Value]; ok {
			continue
		}
		seen[scope.Value] = struct{}{}
		result = append(result, scope.Value)
	}
	return result
}
