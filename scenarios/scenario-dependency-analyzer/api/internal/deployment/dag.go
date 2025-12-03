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

// convertTierTierMap converts deployment tier definitions to tier support summary format
func convertTierTierMap(tiers map[string]types.DeploymentTier) map[string]types.TierSupportSummary {
	if len(tiers) == 0 {
		return nil
	}
	result := make(map[string]types.TierSupportSummary, len(tiers))
	for tier, value := range tiers {
		var supported *bool
		switch strings.ToLower(value.Status) {
		case "ready", "supported":
			flag := true
			supported = &flag
		case "limited", "blocked":
			flag := false
			supported = &flag
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
func InferResourceTierSupport(resourceName string, meta *types.DeploymentDependency) map[string]types.TierSupportSummary {
	normalized := config.NormalizeName(resourceName)

	// Define standard tiers
	standardTiers := []string{"local", "desktop", "server", "mobile", "saas", "enterprise"}

	// Resource classification and fitness scoring
	var resourceClass string
	var heavyOps bool // Indicates resource-intensive operations

	switch normalized {
	case "postgres", "mysql", "mongodb", "redis", "qdrant":
		resourceClass = "database"
		heavyOps = (normalized == "postgres" || normalized == "mysql" || normalized == "mongodb")
	case "ollama", "claude-code", "openai", "anthropic":
		resourceClass = "ai"
		heavyOps = (normalized == "ollama") // Local LLM is heavy
	case "n8n", "huginn", "windmill":
		resourceClass = "automation"
		heavyOps = true
	case "minio", "s3":
		resourceClass = "storage"
		heavyOps = false
	case "browserless", "playwright":
		resourceClass = "browser"
		heavyOps = true
	case "judge0", "sandbox":
		resourceClass = "execution"
		heavyOps = true
	default:
		resourceClass = "service"
		heavyOps = false
	}

	// Build tier support map with intelligent defaults
	support := make(map[string]types.TierSupportSummary, len(standardTiers))

	for _, tier := range standardTiers {
		summary := types.TierSupportSummary{}
		supported := true
		var fitness float64

		switch tier {
		case "local":
			// Everything works locally (dev environment)
			fitness = 1.0
			supported = true

		case "desktop":
			// Desktop apps can run most things, but heavy ops reduce fitness
			if heavyOps {
				fitness = 0.6 // Can run but not ideal
			} else {
				fitness = 0.9
			}
			supported = true

		case "server":
			// Server environments are ideal for most resources
			fitness = 0.95
			supported = true

		case "mobile":
			// Mobile is very restrictive
			if resourceClass == "ai" && heavyOps {
				// Local LLMs don't work on mobile
				fitness = 0.0
				supported = false
				summary.Reason = "Resource-intensive AI not supported on mobile"
			} else if resourceClass == "database" || resourceClass == "storage" {
				// Databases generally don't run on mobile (use remote APIs instead)
				fitness = 0.2
				supported = false
				summary.Reason = "Database should be remote for mobile deployments"
				summary.Alternatives = []string{"saas-" + normalized, "cloud-" + normalized}
			} else if heavyOps {
				fitness = 0.1
				supported = false
				summary.Reason = "Heavy operations not suitable for mobile"
			} else {
				fitness = 0.4 // Lightweight services might work
				supported = true
			}

		case "saas":
			// SaaS deployments - databases and storage work well, heavy compute less so
			if resourceClass == "database" || resourceClass == "storage" {
				fitness = 0.95
				supported = true
			} else if resourceClass == "ai" && heavyOps {
				// Ollama -> use cloud APIs for SaaS
				fitness = 0.3
				supported = true
				summary.Alternatives = []string{"openai", "anthropic", "openrouter"}
				summary.Notes = "Consider using managed AI API for SaaS deployment"
			} else {
				fitness = 0.85
				supported = true
			}

		case "enterprise":
			// Enterprise deployments can handle everything
			fitness = 0.98
			supported = true
		}

		summary.Supported = &supported
		summary.FitnessScore = &fitness
		support[tier] = summary
	}

	return support
}
