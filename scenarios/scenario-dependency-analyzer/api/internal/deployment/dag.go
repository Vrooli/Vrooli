package deployment

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"scenario-dependency-analyzer/internal/config"

	types "scenario-dependency-analyzer/internal/types"
)

// BuildDependencyNodeList recursively builds a list of dependency nodes (resources + scenarios)
// from a scenario's service.json configuration. The visited map prevents infinite recursion
// when circular dependencies exist.
func BuildDependencyNodeList(scenariosDir, scenarioName string, cfg *types.ServiceConfig, visited map[string]struct{}) []types.DeploymentDependencyNode {
	nodes := []types.DeploymentDependencyNode{}
	if cfg == nil {
		return nodes
	}

	var dependencyCatalog types.DeploymentDependencyCatalog
	if cfg.Deployment != nil {
		dependencyCatalog = cfg.Deployment.Dependencies
	}

	resources := config.ResolvedResourceMap(cfg)
	resourceNames := make([]string, 0, len(resources))
	for name := range resources {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)
	for _, name := range resourceNames {
		resource := resources[name]
		var meta *types.DeploymentDependency
		if dependencyCatalog.Resources != nil {
			if resourceMeta, ok := dependencyCatalog.Resources[name]; ok {
				copyMeta := resourceMeta
				meta = &copyMeta
			}
		}
		node := buildResourceDependencyNode(filepath.Dir(scenariosDir), name, meta, resource.Required)
		required := resource.Required
		enabled := resource.Enabled
		node.Required = &required
		node.Enabled = &enabled
		node.Source = "declared"
		nodes = append(nodes, node)
	}

	if cfg.Dependencies.Scenarios != nil {
		scenarioNames := make([]string, 0, len(cfg.Dependencies.Scenarios))
		for name := range cfg.Dependencies.Scenarios {
			scenarioNames = append(scenarioNames, name)
		}
		sort.Strings(scenarioNames)
		for _, depName := range scenarioNames {
			depSpec := cfg.Dependencies.Scenarios[depName]
			var meta *types.DeploymentDependency
			if dependencyCatalog.Scenarios != nil {
				if scenarioMeta, ok := dependencyCatalog.Scenarios[depName]; ok {
					copyMeta := scenarioMeta
					meta = &copyMeta
				}
			}
			node := buildScenarioDependencyNode(scenariosDir, depName, meta, visited)
			required := depSpec.Required
			enabled := depSpec.Enabled
			node.Required = &required
			node.Enabled = &enabled
			node.Source = "declared"
			nodes = append(nodes, node)
		}
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].Type < nodes[j].Type
	})

	return nodes
}

// buildResourceDependencyNode creates a deployment node for a single resource dependency
func buildResourceDependencyNode(repoRoot, name string, meta *types.DeploymentDependency, required bool) types.DeploymentDependencyNode {
	node := types.DeploymentDependencyNode{
		Name: name,
		Type: "resource",
		Path: filepath.Join(repoRoot, "resources", name, "resource.json"),
	}

	// Resource facts are owned by the resource manifest. The deployment catalog
	// may still contribute authored swap hints, but it must never override the
	// live resource's requirements, bundling, or platform declarations.
	declaration, err := loadResourceDeclaration(repoRoot, name, required)
	if err != nil {
		node.TierSupport = unknownTierSupport(err.Error())
	} else {
		if declaration.Requirements != nil {
			node.ResourceType = declaration.Requirements.Class
			node.Requirements = &types.DeploymentRequirements{
				Class:      declaration.Requirements.Class,
				Weight:     ptr(declaration.Requirements.Weight),
				RAMMB:      ptr(declaration.Requirements.RAMMB),
				DiskMB:     ptr(declaration.Requirements.DiskMB),
				CPUCores:   ptr(declaration.Requirements.CPUCores),
				GPU:        ptr(declaration.Requirements.GPU),
				Network:    declaration.Requirements.Network,
				Source:     declaration.Requirements.Source,
				Confidence: declaration.Requirements.Confidence,
			}
		}
		node.TierSupport = resolveResourceTierSupportFromDeclaration(declaration)
	}
	if meta != nil {
		node.Alternatives = collectDependencyAlternatives(meta)
	}
	return node
}

// buildScenarioDependencyNode creates a deployment node for a scenario dependency,
// recursively loading the scenario's own dependencies to build a complete dependency tree.
func buildScenarioDependencyNode(scenariosDir, scenarioName string, parentMeta *types.DeploymentDependency, visited map[string]struct{}) types.DeploymentDependencyNode {
	node := types.DeploymentDependencyNode{
		Name: scenarioName,
		Type: "scenario",
	}

	normalized := config.NormalizeName(scenarioName)
	if _, exists := visited[normalized]; exists {
		node.Notes = "cycle detected"
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	if visited == nil {
		visited = map[string]struct{}{}
	}
	visited[normalized] = struct{}{}
	defer delete(visited, normalized)

	scenarioPath := filepath.Join(scenariosDir, scenarioName)
	node.Path = scenarioPath
	cfg, err := config.LoadServiceConfig(scenarioPath)
	if err != nil {
		node.Notes = fmt.Sprintf("unable to load scenario: %v", err)
		if parentMeta != nil {
			node.TierSupport = convertTierSupportMap(parentMeta.PlatformSupport)
			node.Requirements = parentMeta.Footprint
			node.Alternatives = collectDependencyAlternatives(parentMeta)
		}
		return node
	}

	var scenarioTierSupport map[string]types.TierSupportSummary
	if cfg.Deployment != nil {
		scenarioTierSupport = convertTierTierMap(cfg.Deployment.Tiers)
		node.Alternatives = append(node.Alternatives, collectAdaptationAlternatives(cfg.Deployment.Tiers)...)
	}

	if node.Requirements == nil && parentMeta != nil {
		node.Requirements = parentMeta.Footprint
	}
	fallbackSupport := convertTierSupportMap(nil)
	if parentMeta != nil {
		fallbackSupport = convertTierSupportMap(parentMeta.PlatformSupport)
		node.Alternatives = append(node.Alternatives, collectDependencyAlternatives(parentMeta)...)
	}
	node.TierSupport = mergeTierSupportMaps(scenarioTierSupport, fallbackSupport)
	node.Alternatives = dedupeStrings(node.Alternatives)
	node.Children = BuildDependencyNodeList(scenariosDir, scenarioName, cfg, visited)
	return node
}

// convertTierSupportMap converts deployment metadata tier support to summary format
func convertTierSupportMap(support map[string]types.DependencyTierSupport) map[string]types.TierSupportSummary {
	if len(support) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(support))
	for tier, value := range support {
		result[tier] = types.TierSupportSummary{
			Supported:    value.Supported,
			FitnessScore: value.FitnessScore,
			Reason:       value.Reason,
			Notes:        value.Notes,
			Requirements: value.Requirements,
			Alternatives: append([]string(nil), value.Alternatives...),
		}
	}
	return result
}

// convertTierTierMap converts authored deployment tier inputs to tier support
// summaries. Readiness is derived by the resolver and is intentionally absent
// from service.json.
func convertTierTierMap(tiers map[string]types.DeploymentTier) map[string]types.TierSupportSummary {
	if len(tiers) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(tiers))
	for tier, value := range tiers {
		result[tier] = types.TierSupportSummary{
			Notes:        value.Notes,
			Requirements: value.Requirements,
		}
	}
	return result
}

// mergeTierSupportMaps merges two tier support maps, with preferred taking precedence
func mergeTierSupportMaps(preferred, fallback map[string]types.TierSupportSummary) map[string]types.TierSupportSummary {
	if len(preferred) == 0 && len(fallback) == 0 {
		return nil
	}
	merged := make(map[string]types.TierSupportSummary)
	for tier, value := range fallback {
		merged[tier] = value
	}
	for tier, value := range preferred {
		merged[tier] = value
	}
	return merged
}

// collectDependencyAlternatives extracts all alternative dependency IDs from metadata
func collectDependencyAlternatives(meta *types.DeploymentDependency) []string {
	if meta == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, swap := range meta.SwappableWith {
		if swap.ID == "" {
			continue
		}
		set[swap.ID] = struct{}{}
	}
	for _, support := range meta.PlatformSupport {
		for _, alt := range support.Alternatives {
			if alt == "" {
				continue
			}
			set[alt] = struct{}{}
		}
	}
	return MapKeys(set)
}

// collectAdaptationAlternatives extracts alternative swaps from tier adaptations
func collectAdaptationAlternatives(tiers map[string]types.DeploymentTier) []string {
	set := map[string]struct{}{}
	for _, tier := range tiers {
		for _, adaptation := range tier.Adaptations {
			if adaptation.Swap == "" {
				continue
			}
			set[adaptation.Swap] = struct{}{}
		}
	}
	return MapKeys(set)
}

// BuildDependencyNodeIndex creates a lookup map of all nodes in the dependency tree
func BuildDependencyNodeIndex(nodes []types.DeploymentDependencyNode) map[string]types.DeploymentDependencyNode {
	index := map[string]types.DeploymentDependencyNode{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		key := strings.ToLower(node.Name)
		if key != "" {
			if _, exists := index[key]; !exists {
				index[key] = node
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return index
}

// MapKeys extracts sorted keys from a string set.
func MapKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// dedupeStrings removes duplicate strings from a slice
func dedupeStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return MapKeys(set)
}
