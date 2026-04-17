package hostreq

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenario"
)

type ResolveOptions struct {
	Environment string
	When        string
	Resources   string
	Scenarios   string
	Platform    string
}

type Resolution struct {
	Tools      []ResolvedRequirement `json:"tools"`
	Safeguards []ResolvedRequirement `json:"safeguards"`
}

func Resolve(root, home string, opts ResolveOptions) (Resolution, error) {
	rootManifestPath := filepath.Join(root, ".vrooli", "service.json")
	rootManifest, err := scenario.ReadService(rootManifestPath)
	if err != nil {
		return Resolution{}, fmt.Errorf("load root manifest: %w", err)
	}

	platform := strings.ToLower(strings.TrimSpace(opts.Platform))
	if platform == "" {
		platform = CurrentPlatform()
	}

	state := resolverState{
		root:        root,
		environment: NormalizeEnvironment(opts.Environment),
		when:        strings.ToLower(strings.TrimSpace(opts.When)),
		platform:    platform,
		tools:       make(map[string]*ResolvedRequirement),
		safeguards:  make(map[string]*ResolvedRequirement),
	}

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

	if err := state.addResources(home, opts.Resources); err != nil {
		return Resolution{}, err
	}
	if err := state.addScenarios(opts.Scenarios); err != nil {
		return Resolution{}, err
	}

	return Resolution{
		Tools:      sortedRequirements(state.tools),
		Safeguards: sortedRequirements(state.safeguards),
	}, nil
}

type resolverState struct {
	root        string
	environment string
	when        string
	platform    string
	tools       map[string]*ResolvedRequirement
	safeguards  map[string]*ResolvedRequirement
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
	return nil
}

func (s resolverState) addAll(declarations []Declaration, kind Kind, provenance Provenance) {
	for _, declaration := range declarations {
		if !s.matches(declaration) {
			continue
		}
		s.add(declaration, kind, provenance)
	}
}

func (s resolverState) matches(declaration Declaration) bool {
	if len(declaration.Environments) > 0 && !containsFold(declaration.Environments, s.environment) {
		return false
	}
	if s.when != "" && len(declaration.When) > 0 && !containsFold(declaration.When, s.when) {
		return false
	}
	if len(declaration.Platforms) > 0 && !containsFold(declaration.Platforms, s.platform) {
		return false
	}
	return true
}

func (s resolverState) add(declaration Declaration, kind Kind, provenance Provenance) {
	target := s.tools
	if kind == KindSafeguard {
		target = s.safeguards
	}

	key := strings.TrimSpace(declaration.Name)
	resolved, exists := target[key]
	if !exists {
		target[key] = &ResolvedRequirement{
			Name:         key,
			Kind:         kind,
			Required:     declaration.Required,
			Manual:       declaration.Manual,
			Reasons:      uniqueStrings([]string{strings.TrimSpace(declaration.Reason)}),
			When:         uniqueStrings(declaration.When),
			Environments: uniqueStrings(declaration.Environments),
			Platforms:    uniqueStrings(declaration.Platforms),
			Notes:        uniqueStrings([]string{strings.TrimSpace(declaration.Notes)}),
			Provenance:   []Provenance{provenance},
		}
		return
	}

	resolved.Required = resolved.Required || declaration.Required
	resolved.Manual = resolved.Manual || declaration.Manual
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
