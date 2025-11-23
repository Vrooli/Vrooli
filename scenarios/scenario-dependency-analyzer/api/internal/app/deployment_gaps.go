package app

import (
	"fmt"
	"path/filepath"

	types "scenario-dependency-analyzer/internal/types"
)

// analyzeDeploymentGaps crawls the dependency tree and identifies missing deployment metadata.
// It checks for:
// - Missing deployment blocks in service.json
// - Missing dependency catalogs
// - Missing tier definitions
// - Resource dependencies without metadata
// - Scenario dependencies without metadata
func analyzeDeploymentGaps(scenarioName, scenariosDir string, nodes []types.DeploymentDependencyNode, knownTiers []string) *types.DeploymentMetadataGaps {
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

		cfg, err := loadServiceConfigFromFile(scenarioPath)
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
			resources := resolvedResourceMap(cfg)
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
			fmt.Sprintf("Add tier definitions for: %v", mapKeys(missingTiersSet)))
	}

	// Detect secret requirements from dependencies
	secretRequirements := detectSecretRequirements(nodes)
	if len(secretRequirements) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Configure %d secret(s) for dependencies - see secrets-manager playbooks", len(secretRequirements)))
	}

	// Suggest resource swaps for deployment optimization
	resourceSwaps := suggestResourceSwaps(nodes)
	if len(resourceSwaps) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider %d resource swap(s) for deployment optimization", len(resourceSwaps)))
	}

	if totalGaps > 0 {
		recommendations = append(recommendations,
			"Review gaps_by_scenario for detailed per-scenario action items")
	}

	return &types.DeploymentMetadataGaps{
		TotalGaps:              totalGaps,
		ScenariosMissingAll:    scenariosMissingAll,
		GapsByScenario:         gapsByScenario,
		MissingTiers:           mapKeys(missingTiersSet),
		SecretRequirements:     secretRequirements,
		ResourceSwapSuggestions: resourceSwaps,
		Recommendations:        recommendations,
	}
}

// detectSecretRequirements analyzes dependencies and identifies which ones require secret configuration
func detectSecretRequirements(nodes []types.DeploymentDependencyNode) []types.SecretRequirement {
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

			// Determine if this resource needs secrets
			secretType := ""
			var secretNames []string
			playbookRef := ""

			normalized := normalizeName(node.Name)
			switch normalized {
			case "postgres", "mysql", "mongodb":
				secretType = "database_credentials"
				secretNames = []string{normalized + "_password", normalized + "_user"}
				playbookRef = "secrets-manager/playbooks/database-credentials.md"

			case "redis":
				secretType = "cache_credentials"
				secretNames = []string{"redis_password"}
				playbookRef = "secrets-manager/playbooks/cache-credentials.md"

			case "minio", "s3":
				secretType = "object_storage_credentials"
				secretNames = []string{normalized + "_access_key", normalized + "_secret_key"}
				playbookRef = "secrets-manager/playbooks/object-storage.md"

			case "n8n", "huginn", "windmill":
				secretType = "automation_credentials"
				secretNames = []string{normalized + "_api_key", normalized + "_webhook_secret"}
				playbookRef = "secrets-manager/playbooks/automation-platform.md"

			case "ollama":
				// Ollama typically doesn't need secrets for local use
				secretType = ""

			case "claude-code", "anthropic", "openai":
				secretType = "ai_api_key"
				secretNames = []string{normalized + "_api_key"}
				playbookRef = "secrets-manager/playbooks/ai-api-keys.md"

			case "qdrant":
				secretType = "vector_db_credentials"
				secretNames = []string{"qdrant_api_key"}
				playbookRef = "secrets-manager/playbooks/vector-db.md"

			case "browserless", "playwright":
				secretType = "browser_automation_token"
				secretNames = []string{normalized + "_token"}
				playbookRef = "secrets-manager/playbooks/browser-automation.md"
			}

			if secretType != "" {
				requirements = append(requirements, types.SecretRequirement{
					DependencyName:   node.Name,
					DependencyType:   node.Type,
					SecretType:       secretType,
					RequiredSecrets:  secretNames,
					PlaybookReference: playbookRef,
					Priority:         "required",
				})
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

// suggestResourceSwaps analyzes dependencies and suggests lighter alternatives for specific deployment tiers
func suggestResourceSwaps(nodes []types.DeploymentDependencyNode) []types.ResourceSwapSuggestion {
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

			normalized := normalizeName(node.Name)

			// Suggest swaps based on resource type
			switch normalized {
			case "ollama":
				// For non-local tiers, suggest cloud AI APIs
				suggestions = append(suggestions, types.ResourceSwapSuggestion{
					OriginalResource:   node.Name,
					AlternativeResource: "openrouter",
					Reason:             "Lighter alternative for SaaS/mobile deployments",
					ApplicableTiers:    []string{"mobile", "saas"},
					Relationship:       "api_alternative",
					ImpactDescription:  "Reduces deployment size and resource requirements",
				})
				suggestions = append(suggestions, types.ResourceSwapSuggestion{
					OriginalResource:   node.Name,
					AlternativeResource: "anthropic",
					Reason:             "Managed AI API for production deployments",
					ApplicableTiers:    []string{"saas", "enterprise"},
					Relationship:       "api_alternative",
					ImpactDescription:  "Better scalability and reliability",
				})

			case "postgres":
				// For mobile/saas, suggest managed database
				suggestions = append(suggestions, types.ResourceSwapSuggestion{
					OriginalResource:   node.Name,
					AlternativeResource: "supabase",
					Reason:             "Managed PostgreSQL with built-in APIs",
					ApplicableTiers:    []string{"mobile", "saas"},
					Relationship:       "managed_service",
					ImpactDescription:  "Eliminates database management overhead",
				})

			case "minio":
				// Suggest cloud storage for production
				suggestions = append(suggestions, types.ResourceSwapSuggestion{
					OriginalResource:   node.Name,
					AlternativeResource: "s3",
					Reason:             "Cloud object storage for production deployments",
					ApplicableTiers:    []string{"saas", "enterprise"},
					Relationship:       "cloud_alternative",
					ImpactDescription:  "Better reliability and global availability",
				})

			case "redis":
				// For simple caching, suggest lighter alternatives
				suggestions = append(suggestions, types.ResourceSwapSuggestion{
					OriginalResource:   node.Name,
					AlternativeResource: "in-memory-cache",
					Reason:             "Embedded caching for desktop deployments",
					ApplicableTiers:    []string{"desktop"},
					Relationship:       "embedded_alternative",
					ImpactDescription:  "Simplifies deployment, no separate service needed",
				})
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
