package hostreq

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenario"
)

type ResolveOptions struct {
	// ExcludeRoot limits resolution to the requested scenario/resource scope.
	// Desktop packaging must not inherit Vrooli's development-machine setup
	// requirements (for example, Docker or host hardening safeguards).
	ExcludeRoot   bool
	Environment   string
	When          string
	Resources     string
	Scenarios     string
	ScenarioPaths []string
	Platform      string
}

type Resolution struct {
	Tools      []ResolvedRequirement `json:"tools"`
	Safeguards []ResolvedRequirement `json:"safeguards"`
}

// ResolveSafeguard resolves one focused safeguard through the same manifest
// and operator-state path as project setup. It keeps `vrooli host safeguard`
// from bypassing typed config validation merely because it targets one item.
func ResolveSafeguard(root, name, platform string) (ResolvedRequirement, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResolvedRequirement{}, fmt.Errorf("safeguard name is required")
	}
	catalog, err := loadRequirementCatalog()
	if err != nil {
		return ResolvedRequirement{}, err
	}
	if _, ok := catalog.safeguards[name]; !ok {
		return ResolvedRequirement{}, fmt.Errorf("unknown safeguard %q", name)
	}
	operatorState, err := LoadOperatorState(root)
	if err != nil {
		return ResolvedRequirement{}, err
	}
	platform = hostreqspec.NormalizePlatform(platform)
	state := resolverState{
		root:          root,
		platform:      platform,
		operatorState: operatorState,
		catalog:       catalog,
		tools:         make(map[string]*ResolvedRequirement),
		safeguards:    make(map[string]*ResolvedRequirement),
	}
	declaration := Declaration{Name: name, Required: true, Reason: "focused host safeguard repair"}
	if !state.matches(declaration, KindSafeguard) {
		return ResolvedRequirement{}, fmt.Errorf("safeguard %q is not supported on platform %q", name, platform)
	}
	state.add(declaration, KindSafeguard, Provenance{Kind: "focused", Name: name, Source: "host safeguard command"})
	resolved, ok := state.safeguards[name]
	if !ok {
		return ResolvedRequirement{}, fmt.Errorf("safeguard %q could not be resolved", name)
	}
	return *resolved, nil
}

func Resolve(root, home string, opts ResolveOptions) (Resolution, error) {
	rootManifestPath := filepath.Join(root, ".vrooli", "service.json")
	rootManifest, err := scenario.ReadService(rootManifestPath)
	if err != nil {
		return Resolution{}, fmt.Errorf("load root manifest: %w", err)
	}
	operatorState, err := LoadOperatorState(root)
	if err != nil {
		return Resolution{}, err
	}

	platform := hostreqspec.NormalizePlatform(opts.Platform)
	if platform == "" {
		platform = CurrentPlatform()
	}

	catalog, err := loadRequirementCatalog()
	if err != nil {
		return Resolution{}, err
	}
	state := resolverState{
		root:          root,
		environment:   NormalizeEnvironment(opts.Environment),
		when:          strings.ToLower(strings.TrimSpace(opts.When)),
		platform:      platform,
		operatorState: operatorState,
		catalog:       catalog,
		tools:         make(map[string]*ResolvedRequirement),
		safeguards:    make(map[string]*ResolvedRequirement),
	}

	if !opts.ExcludeRoot {
		state.addAll(rootManifest.HostTools, KindTool, Provenance{
			Kind:   "root",
			Name:   "vrooli",
			Path:   rootManifestPath,
			Source: manifestSourcePath(root, rootManifestPath),
		})
		state.addAll(rootManifest.HostSafeguards, KindSafeguard, Provenance{
			Kind:   "root",
			Name:   "vrooli",
			Path:   rootManifestPath,
			Source: manifestSourcePath(root, rootManifestPath),
		})
	}

	if err := state.addResources(home, opts.Resources); err != nil {
		return Resolution{}, err
	}
	if err := state.addScenarios(opts.Scenarios); err != nil {
		return Resolution{}, err
	}
	if err := state.addScenarioPaths(opts.ScenarioPaths); err != nil {
		return Resolution{}, err
	}

	return Resolution{
		Tools:      sortedRequirements(state.tools),
		Safeguards: sortedRequirements(state.safeguards),
	}, nil
}

// ResolveHostRequirements is the read-only control-plane accessor for callers
// that need the validated host contract without taking part in setup or
// mutating operator state. Keeping this wrapper explicit gives scenario
// consumers a stable boundary while Resolve remains the implementation used by
// the setup and explain commands.
func ResolveHostRequirements(root, home string, opts ResolveOptions) (Resolution, error) {
	return Resolve(root, home, opts)
}

func (s resolverState) addScenarioPaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	items := make([]scenario.Scenario, 0, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		servicePath := filepath.Join(path, ".vrooli", "service.json")
		manifest, err := scenario.ReadService(servicePath)
		if err != nil {
			return fmt.Errorf("load scenario at %q: %w", path, err)
		}
		slug := strings.TrimSpace(manifest.Service.Name)
		if slug == "" {
			slug = filepath.Base(path)
		}
		items = append(items, scenario.Scenario{
			Slug:        slug,
			Path:        path,
			ServicePath: servicePath,
			Manifest:    manifest,
		})
	}
	s.addScenarioItems(items)
	return nil
}

func (s resolverState) addScenarioItems(items []scenario.Scenario) {
	for _, item := range items {
		provenance := Provenance{
			Kind:   "scenario",
			Name:   item.Slug,
			Path:   item.ServicePath,
			Source: manifestSourcePath(s.root, item.ServicePath),
		}
		s.addAll(item.Manifest.HostTools, KindTool, provenance)
		s.addAll(item.Manifest.HostSafeguards, KindSafeguard, provenance)
	}
}

type resolverState struct {
	root          string
	environment   string
	when          string
	platform      string
	operatorState OperatorState
	catalog       requirementCatalog
	tools         map[string]*ResolvedRequirement
	safeguards    map[string]*ResolvedRequirement
}

func (s resolverState) addResources(home, selector string) error {
	selector = normalizeSelector(selector, "enabled")
	if selector == "none" {
		return nil
	}

	controller := resources.NewController(s.root, home)
	report, err := controller.DiscoverReport()
	if err != nil {
		return fmt.Errorf("discover resources: %w", err)
	}
	items := report.Items

	selected := make([]resources.Resource, 0, len(items))
	switch selector {
	case "enabled":
		for _, item := range items {
			if item.Enabled {
				selected = append(selected, item)
			}
		}
	default:
		index := make(map[string]resources.Resource, len(items))
		for _, item := range items {
			index[item.Name] = item
		}
		for _, name := range normalizeCSV(selector) {
			item, ok := index[name]
			if !ok {
				return fmt.Errorf("resource %q not found", name)
			}
			selected = append(selected, item)
		}
	}

	for _, item := range selected {
		if strings.TrimSpace(item.ManifestPath) == "" {
			continue
		}
		manifest, err := controller.LoadManifest(item.ManifestPath)
		if err != nil {
			return fmt.Errorf("load resource manifest %s: %w", item.Name, err)
		}
		provenance := Provenance{
			Kind:   "resource",
			Name:   item.Name,
			Path:   item.ManifestPath,
			Source: manifestSourcePath(s.root, item.ManifestPath),
		}
		s.addAll(manifest.HostTools, KindTool, provenance)
		s.addAll(manifest.HostSafeguards, KindSafeguard, provenance)
		// A resource's target profile is part of its deployment contract, not
		// advisory text. Promote registered tool/safeguard requirements into the
		// same typed resolution as hostTools/hostSafeguards so callers receive
		// provenance and eligibility rather than having to parse profile strings.
		if target, found := manifest.Deployment.Target("desktop", s.platform, ""); found {
			for _, name := range target.Requires {
				name = strings.TrimSpace(name)
				reason := fmt.Sprintf("resource %s desktop target requires %s", item.Name, name)
				if _, ok := s.catalog.tools[name]; ok {
					s.add(Declaration{Name: name, Required: true, Reason: reason}, KindTool, provenance)
					continue
				}
				if _, ok := s.catalog.safeguards[name]; ok {
					s.add(Declaration{Name: name, Required: true, Reason: reason}, KindSafeguard, provenance)
				}
			}
		}
	}

	return nil
}

func (s resolverState) addScenarios(selector string) error {
	selector = normalizeSelector(selector, "none")
	if selector == "none" {
		return nil
	}

	var items []scenario.Scenario
	switch selector {
	case "all":
		report, err := scenario.DiscoverReport(s.root, scenario.SandboxEnv{})
		if err != nil {
			return fmt.Errorf("discover scenarios: %w", err)
		}
		items = report.Items
	default:
		names := normalizeCSV(selector)
		items = make([]scenario.Scenario, 0, len(names))
		for _, name := range names {
			item, err := scenario.Load(s.root, name, scenario.SandboxEnv{})
			if err != nil {
				return fmt.Errorf("load scenario %q: %w", name, err)
			}
			items = append(items, item)
		}
	}

	s.addScenarioItems(items)
	return nil
}

func (s resolverState) addAll(declarations []Declaration, kind Kind, provenance Provenance) {
	for _, declaration := range declarations {
		if !s.matches(declaration, kind) {
			continue
		}
		s.add(declaration, kind, provenance)
	}
}

func (s resolverState) matches(declaration Declaration, kind Kind) bool {
	if len(declaration.Environments) > 0 && !containsFold(declaration.Environments, s.environment) {
		return false
	}
	if s.when != "" && len(declaration.When) > 0 && !containsFold(declaration.When, s.when) {
		return false
	}
	// Platform mismatches remain in the resolution so the runtime can report a
	// durable NotApplicable result with the declared platform as its reason.
	// Filtering them here made macOS/Linux reports look as if declarations had
	// disappeared and encouraged callers to infer support from absence.
	return true
}

func containsPlatform(values []string, target string) bool {
	target = hostreqspec.NormalizePlatform(target)
	for _, value := range values {
		if hostreqspec.NormalizePlatform(value) == target {
			return true
		}
	}
	return false
}

func (s resolverState) add(declaration Declaration, kind Kind, provenance Provenance) {
	target := s.tools
	if kind == KindSafeguard {
		target = s.safeguards
	}

	key := strings.TrimSpace(declaration.Name)
	privilege, bundling, err := s.catalog.details(kind, key, s.platform)
	if err != nil {
		// Resolver fixtures and third-party extension manifests can name objects
		// outside the embedded catalog. Keep them visible and conservative; the
		// conformance validator remains responsible for rejecting an undeclared
		// production registry entry.
		privilege = declaration.DerivePrivilege(s.platform)
		bundling = declaration.Bundling
		if bundling == "" {
			bundling = hostreqspec.BundlingHostRequired
		}
	}
	var config map[string]any
	var configError string
	platforms := append([]string(nil), declaration.Platforms...)
	if kind == KindSafeguard {
		manifest := s.catalog.safeguards[key]
		platforms = mergeUnique(platforms, manifest.Platforms)
		config, configError = resolveSafeguardConfig(key, manifest, s.operatorState.config(kind, key))
	}
	resolved, exists := target[key]
	if !exists {
		target[key] = &ResolvedRequirement{
			Name:           key,
			Kind:           kind,
			Required:       declaration.Required,
			Manual:         declaration.Manual,
			Privilege:      privilege,
			Bundling:       bundling,
			Reasons:        uniqueStrings([]string{strings.TrimSpace(declaration.Reason)}),
			When:           uniqueStrings(declaration.When),
			Environments:   uniqueStrings(declaration.Environments),
			Platforms:      uniqueStrings(platforms),
			Notes:          uniqueStrings([]string{strings.TrimSpace(declaration.Notes)}),
			Provenance:     []Provenance{provenance},
			Requires:       declaration.Requires,
			OperatorChoice: s.operatorState.choice(kind, key),
			Config:         config,
			ConfigError:    configError,
		}
		return
	}

	resolved.Required = resolved.Required || declaration.Required
	if resolved.OperatorChoice == hostreqspec.OperatorChoiceNotRecorded {
		resolved.OperatorChoice = s.operatorState.choice(kind, key)
	}
	resolved.Manual = resolved.Manual || declaration.Manual
	if resolved.Requires.IsZero() && !declaration.Requires.IsZero() {
		// First non-zero capability gate across merged declarations wins; the
		// platform manifest's own `requires` still takes precedence at the
		// handler (see runtime.effectiveCapability).
		resolved.Requires = declaration.Requires
	}
	resolved.Reasons = mergeUnique(resolved.Reasons, []string{strings.TrimSpace(declaration.Reason)})
	resolved.When = mergeUnique(resolved.When, declaration.When)
	resolved.Environments = mergeUnique(resolved.Environments, declaration.Environments)
	resolved.Platforms = mergeUnique(resolved.Platforms, declaration.Platforms)
	resolved.Notes = mergeUnique(resolved.Notes, []string{strings.TrimSpace(declaration.Notes)})
	resolved.Provenance = append(resolved.Provenance, provenance)
	sort.Slice(resolved.Provenance, func(i, j int) bool {
		if resolved.Provenance[i].Kind == resolved.Provenance[j].Kind {
			if resolved.Provenance[i].Name == resolved.Provenance[j].Name {
				return resolved.Provenance[i].Source < resolved.Provenance[j].Source
			}
			return resolved.Provenance[i].Name < resolved.Provenance[j].Name
		}
		return resolved.Provenance[i].Kind < resolved.Provenance[j].Kind
	})
}

func sortedRequirements(items map[string]*ResolvedRequirement) []ResolvedRequirement {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]ResolvedRequirement, 0, len(names))
	for _, name := range names {
		item := *items[name]
		item.Reasons = uniqueStrings(item.Reasons)
		item.When = uniqueStrings(item.When)
		item.Environments = uniqueStrings(item.Environments)
		item.Platforms = uniqueStrings(item.Platforms)
		item.Notes = uniqueStrings(item.Notes)
		result = append(result, item)
	}
	return result
}

func uniqueStrings(values []string) []string {
	return mergeUnique(nil, values)
}

func mergeUnique(existing, incoming []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	result := make([]string, 0, len(existing)+len(incoming))
	for _, value := range append(append([]string{}, existing...), incoming...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsFold(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}
