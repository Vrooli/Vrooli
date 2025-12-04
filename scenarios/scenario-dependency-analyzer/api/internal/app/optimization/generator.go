package optimization

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
)

func GenerateRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, cfg *types.ServiceConfig) []types.OptimizationRecommendation {
	if analysis == nil {
		return nil
	}
	timestamp := time.Now().UTC()
	recommendations := []types.OptimizationRecommendation{}
	recommendations = append(recommendations, buildUnusedResourceRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildUnusedScenarioRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildTierBlockerRecommendations(scenarioName, analysis, timestamp)...)
	recommendations = append(recommendations, buildSecretStrategyRecommendations(scenarioName, cfg, timestamp)...)
	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Priority == recommendations[j].Priority {
			return recommendations[i].Title < recommendations[j].Title
		}
		priorityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return priorityOrder[strings.ToLower(recommendations[i].Priority)] < priorityOrder[strings.ToLower(recommendations[j].Priority)]
	})
	return recommendations
}

func BuildSummary(recs []types.OptimizationRecommendation) types.OptimizationSummary {
	summary := types.OptimizationSummary{
		RecommendationCount: len(recs),
		ByType:              map[string]int{},
		PotentialImpact:     map[string]int{},
	}
	for _, rec := range recs {
		summary.ByType[rec.RecommendationType]++
		if strings.EqualFold(rec.Priority, "high") || strings.EqualFold(rec.Priority, "critical") {
			summary.HighPriority++
		}
		if action, ok := rec.RecommendedState["action"].(string); ok {
			summary.PotentialImpact[action]++
		}
	}
	return summary
}

func buildUnusedResourceRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	if analysis == nil || len(analysis.ResourceDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]types.OptimizationRecommendation, 0, len(analysis.ResourceDiff.Extra))
	for _, drift := range analysis.ResourceDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, types.OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused resource '%s'", name),
			Description:        fmt.Sprintf("Resource '%s' is declared but not detected in the scenario codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
				"details":  drift.Details,
			},
			RecommendedState: map[string]interface{}{
				"action":        "remove_resource",
				"resource_name": name,
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused infrastructure dependency",
			},
			ConfidenceScore: 0.85,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildUnusedScenarioRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	if analysis == nil || len(analysis.ScenarioDiff.Extra) == 0 {
		return nil
	}
	recommendations := make([]types.OptimizationRecommendation, 0, len(analysis.ScenarioDiff.Extra))
	for _, drift := range analysis.ScenarioDiff.Extra {
		name := strings.TrimSpace(drift.Name)
		if name == "" {
			continue
		}
		recommendations = append(recommendations, types.OptimizationRecommendation{
			ID:                 uuid.New().String(),
			ScenarioName:       scenarioName,
			RecommendationType: "dependency_reduction",
			Title:              fmt.Sprintf("Remove unused scenario '%s'", name),
			Description:        fmt.Sprintf("Scenario dependency '%s' is declared but not referenced in the codebase.", name),
			CurrentState: map[string]interface{}{
				"declared": true,
				"detected": false,
			},
			RecommendedState: map[string]interface{}{
				"action":          "remove_scenario_dependency",
				"scenario_name":   name,
				"dependency_type": "scenario",
			},
			EstimatedImpact: map[string]interface{}{
				"complexity_score": -1,
				"description":      "Removes unused scenario coupling",
			},
			ConfidenceScore: 0.8,
			Priority:        "medium",
			Status:          "pending",
			CreatedAt:       timestamp,
		})
	}
	return recommendations
}

func buildTierBlockerRecommendations(scenarioName string, analysis *types.DependencyAnalysisResponse, timestamp time.Time) []types.OptimizationRecommendation {
	report := analysis.DeploymentReport
	if report == nil || len(report.Dependencies) == 0 {
		return nil
	}
	nodeIndex := buildDependencyNodeIndex(report.Dependencies)
	recommendations := []types.OptimizationRecommendation{}
	blockingByTier := map[string][]string{}
	for tier, aggregate := range report.Aggregates {
		for _, blocker := range aggregate.BlockingDependencies {
			key := strings.ToLower(blocker)
			blockingByTier[key] = append(blockingByTier[key], tier)
		}
	}
	for blockerKey, tiers := range blockingByTier {
		node, ok := nodeIndex[blockerKey]
		if !ok {
			continue
		}
		for _, tier := range tiers {
			recommendations = append(recommendations, buildBlockerRecommendation(scenarioName, node, tier, timestamp))
		}
	}
	return recommendations
}

func buildSecretStrategyRecommendations(scenarioName string, cfg *types.ServiceConfig, timestamp time.Time) []types.OptimizationRecommendation {
	if cfg == nil || cfg.Deployment == nil || len(cfg.Deployment.Tiers) == 0 {
		return nil
	}
	recommendations := []types.OptimizationRecommendation{}
	for tierName, tier := range cfg.Deployment.Tiers {
		for _, secret := range tier.Secrets {
			if strings.TrimSpace(secret.StrategyRef) != "" {
				continue
			}
			recommendations = append(recommendations, types.OptimizationRecommendation{
				ID:                 uuid.New().String(),
				ScenarioName:       scenarioName,
				RecommendationType: "performance_improvement",
				Title:              fmt.Sprintf("Annotate secret '%s' on %s", secret.SecretID, tierName),
				Description:        fmt.Sprintf("Secret '%s' on tier '%s' lacks a strategy reference. Link it to a secrets-manager playbook to unblock deployment readiness.", secret.SecretID, tierName),
				CurrentState: map[string]interface{}{
					"tier":           tierName,
					"secret_id":      secret.SecretID,
					"classification": secret.Classification,
				},
				RecommendedState: map[string]interface{}{
					"action":    "annotate_secret_strategy",
					"tier":      tierName,
					"secret_id": secret.SecretID,
					"guidance":  "Use secrets-manager playbooks to define strategy_ref",
				},
				EstimatedImpact: map[string]interface{}{
					"risk":        "secrets_gap",
					"description": "Missing secret strategies block deployment",
				},
				ConfidenceScore: 0.75,
				Priority:        "high",
				Status:          "pending",
				CreatedAt:       timestamp,
			})
		}
	}
	return recommendations
}

func buildBlockerRecommendation(scenarioName string, node types.DeploymentDependencyNode, tier string, timestamp time.Time) types.OptimizationRecommendation {
	alt := ""
	if len(node.Alternatives) > 0 {
		alt = node.Alternatives[0]
	}
	tierSupport := map[string]interface{}{}
	if node.TierSupport != nil {
		if support, ok := node.TierSupport[tier]; ok {
			tierSupport = map[string]interface{}{
				"supported":     support.Supported,
				"fitness_score": support.FitnessScore,
				"notes":         support.Notes,
				"alternatives":  support.Alternatives,
			}
		}
	}
	priority := "high"
	if node.Type != "resource" {
		priority = "medium"
	}
	estImpact := map[string]interface{}{
		"tier":       tier,
		"risk":       "deployment_blocker",
		"dependency": node.Name,
	}
	return types.OptimizationRecommendation{
		ID:                 uuid.New().String(),
		ScenarioName:       scenarioName,
		RecommendationType: "resource_swap",
		Title:              fmt.Sprintf("Resolve %s blocker on %s", node.Name, tier),
		Description:        fmt.Sprintf("Dependency '%s' blocks tier '%s'. Consider annotated alternatives or lighter swaps to restore fitness.", node.Name, tier),
		CurrentState: map[string]interface{}{
			"dependency":   node.Name,
			"tier":         tier,
			"tier_support": tierSupport,
		},
		RecommendedState: map[string]interface{}{
			"action":                "swap_dependency",
			"dependency":            node.Name,
			"tier":                  tier,
			"suggested_alternative": alt,
		},
		EstimatedImpact: estImpact,
		ConfidenceScore: 0.7,
		Priority:        priority,
		Status:          "pending",
		CreatedAt:       timestamp,
	}
}

func buildDependencyNodeIndex(nodes []types.DeploymentDependencyNode) map[string]types.DeploymentDependencyNode {
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
