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
		var meta *types.DeploymentDependency
		if dependencyCatalog.Resources != nil {
			if resourceMeta, ok := dependencyCatalog.Resources[name]; ok {
				copyMeta := resourceMeta
				meta = &copyMeta
			}
		}
		node := buildResourceDependencyNode(name, meta)
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
			var meta *types.DeploymentDependency
			if dependencyCatalog.Scenarios != nil {
				if scenarioMeta, ok := dependencyCatalog.Scenarios[depName]; ok {
					copyMeta := scenarioMeta
					meta = &copyMeta
				}
			}
			node := buildScenarioDependencyNode(scenariosDir, depName, meta, visited)
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
func buildResourceDependencyNode(name string, meta *types.DeploymentDependency) types.DeploymentDependencyNode {
	node := types.DeploymentDependencyNode{
		Name: name,
		Type: "resource",
	}
	if meta == nil {
		// No metadata - infer default tier support based on resource type
		node.TierSupport = InferResourceTierSupport(name, nil)
		return node
	}
	node.ResourceType = meta.ResourceType
	node.Requirements = meta.Footprint

	// Convert explicit metadata to tier support
	tierSupport := convertTierSupportMap(meta.PlatformSupport)

	// If no tier support defined, infer from resource type
	if len(tierSupport) == 0 {
		tierSupport = InferResourceTierSupport(name, meta)
	}

	node.TierSupport = tierSupport
	node.Alternatives = collectDependencyAlternatives(meta)
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
		node.Requirements = cfg.Deployment.AggregateRequirements
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

// convertTierTierMap converts deployment tier definitions to tier support summary format.
// Uses InterpretTierStatus to decide how status strings map to supported flags.
func convertTierTierMap(tiers map[string]types.DeploymentTier) map[string]types.TierSupportSummary {
	if len(tiers) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(tiers))
	for tier, value := range tiers {
		var supported *bool
		status := InterpretTierStatus(strings.ToLower(value.Status))
		switch status {
		case TierStatusReady:
			flag := true
			supported = &flag
		case TierStatusLimited:
			flag := false
			supported = &flag
		// TierStatusUnknown leaves supported as nil
		}
		result[tier] = types.TierSupportSummary{
			Supported:    supported,
			FitnessScore: value.FitnessScore,
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

// InferResourceTierSupport generates intelligent default tier support based on resource type.
// This provides reasonable defaults when explicit metadata is missing.
//
// The function delegates to ClassifyResource and DecideTierFitness for all decision logic,
// keeping the inference logic centralized in the decisions module.
func InferResourceTierSupport(resourceName string, meta *types.DeploymentDependency) map[string]types.TierSupportSummary {
	// Classify the resource to understand its operational characteristics
	classification := ClassifyResource(resourceName)

	// Define standard tiers
	standardTiers := []string{"local", "desktop", "server", "mobile", "saas", "enterprise"}

	// Build tier support map using decision helpers
	support := make(map[string]types.TierSupportSummary, len(standardTiers))

	for _, tier := range standardTiers {
		// Delegate fitness decision to centralized decision logic
		decision := DecideTierFitness(tier, classification)

		summary := types.TierSupportSummary{
			Reason:       decision.Reason,
			Notes:        decision.Notes,
			Alternatives: decision.Alternatives,
		}
		supported := decision.Supported
		fitness := decision.FitnessScore

		summary.Supported = &supported
		summary.FitnessScore = &fitness
		support[tier] = summary
	}

	return support
}
