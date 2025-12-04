package deployment

import (
	"fmt"
	"path/filepath"

	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// Note: config import retained for config.LoadServiceConfig and config.ResolvedResourceMap in AnalyzeGaps

// AnalyzeGaps crawls the dependency tree and identifies missing deployment metadata.
// It checks for:
// - Missing deployment blocks in service.json
// - Missing dependency catalogs
// - Missing tier definitions
// - Resource dependencies without metadata
// - Scenario dependencies without metadata
func AnalyzeGaps(scenarioName, scenariosDir string, nodes []types.DeploymentDependencyNode, knownTiers []string) *types.DeploymentMetadataGaps {
	gapsByScenario := make(map[string]types.ScenarioGapInfo)
	tierSet := make(map[string]struct{})
	for _, tier := range knownTiers {
		tierSet[tier] = struct{}{}
	}

	// Walk the tree recursively
	var walk func(node types.DeploymentDependencyNode, depth int)
	walk = func(node types.DeploymentDependencyNode, depth int) {
		if node.Type != "scenario" {
			return
		}

		// Load the scenario's service.json to check for deployment metadata
		scenarioPath := node.Path
		if scenarioPath == "" {
			scenarioPath = filepath.Join(scenariosDir, node.Name)
		}

		cfg, err := config.LoadServiceConfig(scenarioPath)
		if err != nil {
			// Can't analyze if we can't load the config
			return
		}

		gap := types.ScenarioGapInfo{
			ScenarioName:            node.Name,
			ScenarioPath:            scenarioPath,
			HasDeploymentBlock:      cfg.Deployment != nil,
			MissingTierDefinitions:  []string{},
			MissingResourceMetadata: []string{},
			MissingScenarioMetadata: []string{},
			SuggestedActions:        []string{},
		}

		if cfg.Deployment == nil {
			gap.SuggestedActions = append(gap.SuggestedActions, "Add deployment block to .vrooli/service.json")
		} else {
			// Check if dependency catalog exists
			hasResourceCatalog := cfg.Deployment.Dependencies.Resources != nil && len(cfg.Deployment.Dependencies.Resources) > 0
			hasScenarioCatalog := cfg.Deployment.Dependencies.Scenarios != nil && len(cfg.Deployment.Dependencies.Scenarios) > 0
			gap.MissingDependencyCatalog = !hasResourceCatalog && !hasScenarioCatalog

			if gap.MissingDependencyCatalog {
				gap.SuggestedActions = append(gap.SuggestedActions, "Add deployment.dependencies catalog for resources/scenarios")
			}

			// Check tier definitions
			if cfg.Deployment.Tiers == nil || len(cfg.Deployment.Tiers) == 0 {
				for tier := range tierSet {
					gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
				}
				gap.SuggestedActions = append(gap.SuggestedActions, "Define deployment.tiers with fitness scores")
			} else {
				for tier := range tierSet {
					if _, exists := cfg.Deployment.Tiers[tier]; !exists {
						gap.MissingTierDefinitions = append(gap.MissingTierDefinitions, tier)
					}
				}
			}

			// Check which declared resources are missing metadata
			resources := config.ResolvedResourceMap(cfg)
			for resName := range resources {
				if cfg.Deployment.Dependencies.Resources == nil {
					gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
				} else if _, exists := cfg.Deployment.Dependencies.Resources[resName]; !exists {
					gap.MissingResourceMetadata = append(gap.MissingResourceMetadata, resName)
				}
			}

			// Check which declared scenarios are missing metadata
			if cfg.Dependencies.Scenarios != nil {
				for scenName := range cfg.Dependencies.Scenarios {
					if cfg.Deployment.Dependencies.Scenarios == nil {
						gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
					} else if _, exists := cfg.Deployment.Dependencies.Scenarios[scenName]; !exists {
						gap.MissingScenarioMetadata = append(gap.MissingScenarioMetadata, scenName)
					}
				}
			}
		}

		// Only add if there are actual gaps
		if !gap.HasDeploymentBlock || gap.MissingDependencyCatalog ||
			len(gap.MissingTierDefinitions) > 0 ||
			len(gap.MissingResourceMetadata) > 0 ||
			len(gap.MissingScenarioMetadata) > 0 {
			gapsByScenario[node.Name] = gap
		}

		// Recurse into children
		for _, child := range node.Children {
			walk(child, depth+1)
		}
	}

	// Walk all top-level nodes
	for _, node := range nodes {
		walk(node, 0)
	}

	// Compute summary statistics
	totalGaps := 0
	scenariosMissingAll := 0
	missingTiersSet := make(map[string]struct{})
	recommendations := []string{}

	for _, gap := range gapsByScenario {
		if !gap.HasDeploymentBlock {
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

	// Generate recommendations
	if scenariosMissingAll > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d scenario(s) missing deployment blocks entirely - run scan --apply to initialize", scenariosMissingAll))
	}
	if len(missingTiersSet) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Add tier definitions for: %v", MapKeys(missingTiersSet)))
	}

	// Detect secret requirements from dependencies
	secretRequirements := DetectSecretRequirements(nodes)
	if len(secretRequirements) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Configure %d secret(s) for dependencies - see secrets-manager playbooks", len(secretRequirements)))
	}

	// Suggest resource swaps for deployment optimization
	resourceSwaps := SuggestResourceSwaps(nodes)
	if len(resourceSwaps) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider %d resource swap(s) for deployment optimization", len(resourceSwaps)))
	}

	if totalGaps > 0 {
		recommendations = append(recommendations,
			"Review gaps_by_scenario for detailed per-scenario action items")
	}

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

// DetectSecretRequirements analyzes dependencies and identifies which ones require secret configuration.
// Delegates to ClassifySecretRequirements for the actual decision logic.
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

			// Delegate secret classification to centralized decision logic
			classification := ClassifySecretRequirements(node.Name)
			if classification != nil {
				requirements = append(requirements, BuildSecretRequirement(node.Name, node.Type, classification))
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

// SuggestResourceSwaps analyzes dependencies and suggests lighter alternatives for specific deployment tiers.
// Delegates to DecideResourceSwaps for the actual decision logic.
func SuggestResourceSwaps(nodes []types.DeploymentDependencyNode) []types.ResourceSwapSuggestion {
	suggestions := []types.ResourceSwapSuggestion{}
	seen := make(map[string]bool)

	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		if node.Type == "resource" {
			key := node.Name
			if seen[key] {
				return
			}
			seen[key] = true

			// Delegate swap decisions to centralized decision logic
			recommendations := DecideResourceSwaps(node.Name)
			for _, rec := range recommendations {
				suggestions = append(suggestions, BuildSwapSuggestion(node.Name, rec))
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, node := range nodes {
		walk(node)
	}

	return suggestions
}
