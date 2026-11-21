package deployment

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/packages/deployability"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// Note: config import retained for config.LoadServiceConfig and config.ResolvedResourceMap in AnalyzeGaps

// AnalyzeGaps crawls the dependency tree and identifies missing deployment metadata.
// It checks for:
// - Missing tier_feasibility blocks in service.json
// - Missing dependency catalogs
// - Missing tier definitions
// - Resource dependencies without metadata
// - Scenario dependencies without metadata
func AnalyzeGaps(scenarioName, scenarioPath, scenariosDir string, nodes []types.DeploymentDependencyNode, knownTiers []string) *types.DeploymentMetadataGaps {
	tierSet := buildTierSet(knownTiers)
	gapsByScenario := collectScenarioGaps(scenariosDir, nodes, tierSet)
	if rootGap, ok := buildScenarioGap(types.DeploymentDependencyNode{
		Name:   scenarioName,
		Type:   "scenario",
		Path:   scenarioPath,
		Source: "root",
	}, scenariosDir, tierSet); ok {
		gapsByScenario[scenarioName] = rootGap
	}

	totalGaps, scenariosMissingAll, missingTiersSet := summarizeGaps(gapsByScenario)
	secretRequirements := DetectSecretRequirements(nodes)
	resourceSwaps := SuggestResourceSwaps(nodes)
	recommendations := buildGapRecommendations(totalGaps, scenariosMissingAll, missingTiersSet, secretRequirements, resourceSwaps)

	return &types.DeploymentMetadataGaps{
		TotalGaps:               totalGaps,
		ScenariosMissingAll:     scenariosMissingAll,
		GapsByScenario:          gapsByScenario,
		MissingTiers:            MapKeys(missingTiersSet),
		SecretRequirements:      secretRequirements,
		ResourceSwapSuggestions: resourceSwaps,
		Recommendations:         recommendations,
	}
}

func buildTierSet(knownTiers []string) map[string]struct{} {
	tierSet := make(map[string]struct{}, len(knownTiers))
	for _, tier := range knownTiers {
		tierSet[tier] = struct{}{}
	}
	return tierSet
}

func collectScenarioGaps(scenariosDir string, nodes []types.DeploymentDependencyNode, tierSet map[string]struct{}) map[string]types.ScenarioGapInfo {
	gapsByScenario := make(map[string]types.ScenarioGapInfo)

	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		if node.Type == "scenario" {
			if gap, ok := buildScenarioGap(node, scenariosDir, tierSet); ok {
				gapsByScenario[node.Name] = gap
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, node := range nodes {
		walk(node)
	}

	return gapsByScenario
}

func buildScenarioGap(node types.DeploymentDependencyNode, scenariosDir string, tierSet map[string]struct{}) (types.ScenarioGapInfo, bool) {
	scenarioPath := node.Path
	if scenarioPath == "" {
		scenarioPath = filepath.Join(scenariosDir, node.Name)
	}

	cfg, err := config.LoadServiceConfig(scenarioPath)
	if err != nil {
		return types.ScenarioGapInfo{}, false
	}

	gap := types.ScenarioGapInfo{
		ScenarioName:            node.Name,
		ScenarioPath:            scenarioPath,
		HasTierFeasibility:      cfg.TierFeasibility != nil,
		MissingTierDefinitions:  []string{},
		MissingResourceMetadata: []string{},
		MissingScenarioMetadata: []string{},
		SuggestedActions:        []string{},
	}

	if cfg.TierFeasibility == nil {
		gap.SuggestedActions = append(gap.SuggestedActions, "Add tier_feasibility to .vrooli/service.json")
		return gap, gapHasFindings(gap)
	}

	hasResourceCatalog := len(cfg.TierFeasibility.Dependencies.Resources) > 0
	hasScenarioCatalog := len(cfg.TierFeasibility.Dependencies.Scenarios) > 0
	gap.MissingDependencyCatalog = !hasResourceCatalog && !hasScenarioCatalog
	if gap.MissingDependencyCatalog {
		gap.SuggestedActions = append(gap.SuggestedActions, "Add deployment.dependencies catalog for resources/scenarios")
	}

	if len(cfg.TierFeasibility.Tiers) == 0 {
		for tier := range tierSet {
			gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
		}
		if len(gap.MissingTierDefinitions) > 0 {
			gap.SuggestedActions = append(gap.SuggestedActions, "Define deployment.tiers with fitness scores")
		}
	} else {
		for tier := range tierSet {
			if _, exists := cfg.TierFeasibility.Tiers[tier]; !exists {
				gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
			}
		}
	}

	resources := config.ResolvedResourceMap(cfg)
	for resName, resource := range resources {
		if !(resource.Required || resource.Enabled) {
			continue
		}
		if cfg.TierFeasibility.Dependencies.Resources == nil {
			gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
		} else if _, exists := cfg.TierFeasibility.Dependencies.Resources[resName]; !exists {
			gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
		}
	}

	if cfg.Dependencies.Scenarios != nil {
		for scenName, dep := range cfg.Dependencies.Scenarios {
			if !(dep.Required || dep.Enabled) {
				continue
			}
			if cfg.TierFeasibility.Dependencies.Scenarios == nil {
				gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
			} else if _, exists := cfg.TierFeasibility.Dependencies.Scenarios[scenName]; !exists {
				gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
			}
		}
	}

	sort.Strings(gap.MissingTierDefinitions)
	sort.Strings(gap.MissingResourceMetadata)
	sort.Strings(gap.MissingScenarioMetadata)

	return gap, gapHasFindings(gap)
}

func gapHasFindings(gap types.ScenarioGapInfo) bool {
	return !gap.HasTierFeasibility || gap.MissingDependencyCatalog ||
		len(gap.MissingTierDefinitions) > 0 ||
		len(gap.MissingResourceMetadata) > 0 ||
		len(gap.MissingScenarioMetadata) > 0
}

func summarizeGaps(gapsByScenario map[string]types.ScenarioGapInfo) (int, int, map[string]struct{}) {
	totalGaps := 0
	scenariosMissingAll := 0
	missingTiersSet := make(map[string]struct{})

	for _, gap := range gapsByScenario {
		if !gap.HasTierFeasibility {
			totalGaps += 10 // Weight heavily
			scenariosMissingAll++
		} else {
			if gap.MissingDependencyCatalog {
				totalGaps++
			}
			totalGaps += len(gap.MissingTierDefinitions)
			totalGaps += len(gap.MissingResourceMetadata)
			totalGaps += len(gap.MissingScenarioMetadata)
		}

		for _, tier := range gap.MissingTierDefinitions {
			missingTiersSet[tier] = struct{}{}
		}
	}

	return totalGaps, scenariosMissingAll, missingTiersSet
}

func buildGapRecommendations(
	totalGaps int,
	scenariosMissingAll int,
	missingTiersSet map[string]struct{},
	secretRequirements []types.SecretRequirement,
	resourceSwaps []types.ResourceSwapSuggestion,
) []string {
	recommendations := []string{}

	if scenariosMissingAll > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d scenario(s) missing tier_feasibility entirely - author tier evidence before deployment analysis", scenariosMissingAll))
	}
	if len(missingTiersSet) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Add tier definitions for: %v", MapKeys(missingTiersSet)))
	}
	if len(secretRequirements) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Configure %d secret(s) for dependencies - see secrets-manager playbooks", len(secretRequirements)))
	}
	if len(resourceSwaps) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider %d resource swap(s) for deployment optimization", len(resourceSwaps)))
	}
	if totalGaps > 0 {
		recommendations = append(recommendations,
			"Review gaps_by_scenario for detailed per-scenario action items")
	}

	return recommendations
}

// DetectSecretRequirements analyzes dependencies using the credential declarations
// owned by each resource manifest. Missing manifests are reported as analysis gaps,
// never converted into guessed credentials.
func DetectSecretRequirements(nodes []types.DeploymentDependencyNode) []types.SecretRequirement {
	requirements := []types.SecretRequirement{}
	seen := make(map[string]bool)

	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		if node.Type == "resource" {
			key := node.Name
			if seen[key] {
				return
			}
			seen[key] = true

			declared, err := ReadSecretRequirements(node.Path, node.Name, node.Type)
			if err == nil {
				requirements = append(requirements, declared...)
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, node := range nodes {
		walk(node)
	}

	return requirements
}

// SuggestResourceSwaps analyzes dependencies using alternatives declared by
// deployment metadata. There is no name-based fallback catalog.
func SuggestResourceSwaps(nodes []types.DeploymentDependencyNode) []types.ResourceSwapSuggestion {
	sources := make([]deployability.SwapSource, 0)

	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		if node.Type == "resource" {
			alternatives := make([]deployability.SwapAlternative, 0, len(node.Alternatives))
			for _, alternative := range node.Alternatives {
				alternatives = append(alternatives, deployability.SwapAlternative{Name: alternative})
			}
			sources = append(sources, deployability.SwapSource{Original: node.Name, Alternatives: alternatives})
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, node := range nodes {
		walk(node)
	}

	shared := deployability.SuggestResourceSwaps(sources)
	suggestions := make([]types.ResourceSwapSuggestion, 0, len(shared))
	for _, suggestion := range shared {
		suggestions = append(suggestions, types.ResourceSwapSuggestion{
			OriginalResource:    suggestion.OriginalResource,
			AlternativeResource: suggestion.AlternativeResource,
			Reason:              suggestion.Reason,
			ApplicableTiers:     suggestion.ApplicableTiers,
			Relationship:        suggestion.Relationship,
			ImpactDescription:   suggestion.ImpactDescription,
		})
	}
	return suggestions
}
