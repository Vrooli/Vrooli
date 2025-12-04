package optimization

import (
	"testing"
	"time"

	types "scenario-dependency-analyzer/internal/types"
)

// TestGenerateRecommendations tests the main recommendation generation function.
func TestGenerateRecommendations(t *testing.T) {
	t.Run("NilAnalysis", func(t *testing.T) {
		recs := GenerateRecommendations("test", nil, nil)
		if recs != nil {
			t.Errorf("expected nil for nil analysis, got %v", recs)
		}
	})

	t.Run("EmptyAnalysis", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{}
		recs := GenerateRecommendations("test", analysis, nil)
		if len(recs) != 0 {
			t.Errorf("expected 0 recommendations for empty analysis, got %d", len(recs))
		}
	})

	t.Run("UnusedResources", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			ResourceDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{
					{Name: "redis"},
					{Name: "minio"},
				},
			},
		}
		recs := GenerateRecommendations("test", analysis, nil)
		if len(recs) != 2 {
			t.Fatalf("expected 2 recommendations, got %d", len(recs))
		}

		for _, rec := range recs {
			if rec.RecommendationType != "dependency_reduction" {
				t.Errorf("expected dependency_reduction type, got %s", rec.RecommendationType)
			}
			if rec.ScenarioName != "test" {
				t.Errorf("expected scenario name 'test', got %s", rec.ScenarioName)
			}
			if rec.Status != "pending" {
				t.Errorf("expected pending status, got %s", rec.Status)
			}
			if rec.ID == "" {
				t.Error("expected non-empty ID")
			}
		}
	})

	t.Run("UnusedScenarios", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			ScenarioDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{{Name: "legacy-service"}},
			},
		}
		recs := GenerateRecommendations("test", analysis, nil)
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}

		rec := recs[0]
		if rec.RecommendedState["scenario_name"] != "legacy-service" {
			t.Errorf("expected scenario_name 'legacy-service', got %v", rec.RecommendedState["scenario_name"])
		}
	})

	t.Run("TierBlockers", func(t *testing.T) {
		fitness := 0.3
		analysis := &types.DependencyAnalysisResponse{
			DeploymentReport: &types.DeploymentAnalysisReport{
				Aggregates: map[string]types.DeploymentTierAggregate{
					"desktop": {BlockingDependencies: []string{"postgres"}},
				},
				Dependencies: []types.DeploymentDependencyNode{
					{
						Name:         "postgres",
						Type:         "resource",
						Alternatives: []string{"sqlite"},
						TierSupport: map[string]types.TierSupportSummary{
							"desktop": {
								Supported:    boolPtr(false),
								FitnessScore: &fitness,
							},
						},
					},
				},
			},
		}
		recs := GenerateRecommendations("test", analysis, nil)
		if len(recs) == 0 {
			t.Fatal("expected at least 1 recommendation for blocker")
		}

		found := false
		for _, rec := range recs {
			if rec.RecommendationType == "resource_swap" {
				found = true
				if rec.RecommendedState["suggested_alternative"] != "sqlite" {
					t.Errorf("expected sqlite alternative, got %v", rec.RecommendedState["suggested_alternative"])
				}
				break
			}
		}
		if !found {
			t.Error("expected resource_swap recommendation")
		}
	})

	t.Run("SecretStrategies", func(t *testing.T) {
		cfg := &types.ServiceConfig{
			Deployment: &types.ServiceDeployment{
				Tiers: map[string]types.DeploymentTier{
					"desktop": {
						Secrets: []types.DeploymentTierSecret{
							{SecretID: "api-key", StrategyRef: ""},
							{SecretID: "db-password", StrategyRef: "secrets-manager://auto"},
						},
					},
				},
			},
		}
		analysis := &types.DependencyAnalysisResponse{}
		recs := GenerateRecommendations("test", analysis, cfg)

		found := false
		for _, rec := range recs {
			if rec.RecommendedState["action"] == "annotate_secret_strategy" {
				found = true
				if rec.RecommendedState["secret_id"] != "api-key" {
					t.Errorf("expected api-key secret, got %v", rec.RecommendedState["secret_id"])
				}
				break
			}
		}
		if !found {
			t.Error("expected annotate_secret_strategy recommendation")
		}
	})

	t.Run("PrioritySorting", func(t *testing.T) {
		// Create analysis that produces multiple recommendation types
		fitness := 0.3
		analysis := &types.DependencyAnalysisResponse{
			ResourceDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{{Name: "redis"}},
			},
			DeploymentReport: &types.DeploymentAnalysisReport{
				Aggregates: map[string]types.DeploymentTierAggregate{
					"desktop": {BlockingDependencies: []string{"postgres"}},
				},
				Dependencies: []types.DeploymentDependencyNode{
					{Name: "postgres", Type: "resource", Alternatives: []string{"sqlite"},
						TierSupport: map[string]types.TierSupportSummary{
							"desktop": {Supported: boolPtr(false), FitnessScore: &fitness},
						}},
				},
			},
		}
		cfg := &types.ServiceConfig{
			Deployment: &types.ServiceDeployment{
				Tiers: map[string]types.DeploymentTier{
					"desktop": {
						Secrets: []types.DeploymentTierSecret{{SecretID: "api-key"}},
					},
				},
			},
		}

		recs := GenerateRecommendations("test", analysis, cfg)
		if len(recs) < 2 {
			t.Fatalf("expected at least 2 recommendations, got %d", len(recs))
		}

		// Verify high priority comes before medium
		priorityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		for i := 0; i < len(recs)-1; i++ {
			curr := priorityOrder[recs[i].Priority]
			next := priorityOrder[recs[i+1].Priority]
			if curr > next {
				t.Errorf("recommendations not sorted by priority: %s before %s",
					recs[i].Priority, recs[i+1].Priority)
			}
		}
	})

	t.Run("EmptyDriftNamesFiltered", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			ResourceDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{
					{Name: ""},
					{Name: "  "},
					{Name: "valid-resource"},
				},
			},
		}
		recs := GenerateRecommendations("test", analysis, nil)
		if len(recs) != 1 {
			t.Errorf("expected 1 recommendation (empty names filtered), got %d", len(recs))
		}
	})
}

// TestBuildSummary tests the summary builder.
func TestBuildSummary(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		summary := BuildSummary(nil)
		if summary.RecommendationCount != 0 {
			t.Errorf("expected 0 count, got %d", summary.RecommendationCount)
		}
	})

	t.Run("WithRecommendations", func(t *testing.T) {
		recs := []types.OptimizationRecommendation{
			{RecommendationType: "dependency_reduction", Priority: "medium",
				RecommendedState: map[string]interface{}{"action": "remove_resource"}},
			{RecommendationType: "dependency_reduction", Priority: "high",
				RecommendedState: map[string]interface{}{"action": "remove_resource"}},
			{RecommendationType: "resource_swap", Priority: "critical",
				RecommendedState: map[string]interface{}{"action": "swap_dependency"}},
		}

		summary := BuildSummary(recs)
		if summary.RecommendationCount != 3 {
			t.Errorf("expected 3 count, got %d", summary.RecommendationCount)
		}
		if summary.HighPriority != 2 {
			t.Errorf("expected 2 high priority, got %d", summary.HighPriority)
		}
		if summary.ByType["dependency_reduction"] != 2 {
			t.Errorf("expected 2 dependency_reduction, got %d", summary.ByType["dependency_reduction"])
		}
		if summary.ByType["resource_swap"] != 1 {
			t.Errorf("expected 1 resource_swap, got %d", summary.ByType["resource_swap"])
		}
		if summary.PotentialImpact["remove_resource"] != 2 {
			t.Errorf("expected 2 remove_resource actions, got %d", summary.PotentialImpact["remove_resource"])
		}
	})
}

// TestBuildUnusedResourceRecommendations tests unused resource recommendation generation.
func TestBuildUnusedResourceRecommendations(t *testing.T) {
	timestamp := time.Now()

	t.Run("NilAnalysis", func(t *testing.T) {
		recs := buildUnusedResourceRecommendations("test", nil, timestamp)
		if recs != nil {
			t.Errorf("expected nil for nil analysis, got %v", recs)
		}
	})

	t.Run("NoExtraResources", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{}
		recs := buildUnusedResourceRecommendations("test", analysis, timestamp)
		if recs != nil {
			t.Errorf("expected nil for no extra resources, got %v", recs)
		}
	})

	t.Run("WithExtraResources", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			ResourceDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{
					{Name: "redis", Details: map[string]interface{}{"type": "cache"}},
				},
			},
		}
		recs := buildUnusedResourceRecommendations("test-scenario", analysis, timestamp)
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}

		rec := recs[0]
		if rec.ScenarioName != "test-scenario" {
			t.Errorf("expected scenario name 'test-scenario', got %s", rec.ScenarioName)
		}
		if rec.RecommendationType != "dependency_reduction" {
			t.Errorf("expected dependency_reduction, got %s", rec.RecommendationType)
		}
		if rec.RecommendedState["resource_name"] != "redis" {
			t.Errorf("expected resource_name 'redis', got %v", rec.RecommendedState["resource_name"])
		}
		if rec.Priority != "medium" {
			t.Errorf("expected medium priority, got %s", rec.Priority)
		}
		if rec.ConfidenceScore != 0.85 {
			t.Errorf("expected confidence 0.85, got %f", rec.ConfidenceScore)
		}
	})
}

// TestBuildUnusedScenarioRecommendations tests unused scenario recommendation generation.
func TestBuildUnusedScenarioRecommendations(t *testing.T) {
	timestamp := time.Now()

	t.Run("WithExtraScenarios", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			ScenarioDiff: types.DependencyDiff{
				Extra: []types.DependencyDrift{
					{Name: "old-helper"},
				},
			},
		}
		recs := buildUnusedScenarioRecommendations("main-scenario", analysis, timestamp)
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}

		rec := recs[0]
		if rec.RecommendedState["scenario_name"] != "old-helper" {
			t.Errorf("expected scenario_name 'old-helper', got %v", rec.RecommendedState["scenario_name"])
		}
		if rec.RecommendedState["dependency_type"] != "scenario" {
			t.Errorf("expected dependency_type 'scenario', got %v", rec.RecommendedState["dependency_type"])
		}
	})
}

// TestBuildTierBlockerRecommendations tests tier blocker recommendation generation.
func TestBuildTierBlockerRecommendations(t *testing.T) {
	timestamp := time.Now()
	fitness := 0.4

	t.Run("NilDeploymentReport", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{}
		recs := buildTierBlockerRecommendations("test", analysis, timestamp)
		if recs != nil {
			t.Errorf("expected nil for nil deployment report, got %v", recs)
		}
	})

	t.Run("EmptyDependencies", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			DeploymentReport: &types.DeploymentAnalysisReport{},
		}
		recs := buildTierBlockerRecommendations("test", analysis, timestamp)
		if recs != nil {
			t.Errorf("expected nil for empty dependencies, got %v", recs)
		}
	})

	t.Run("WithBlocker", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			DeploymentReport: &types.DeploymentAnalysisReport{
				Aggregates: map[string]types.DeploymentTierAggregate{
					"mobile": {BlockingDependencies: []string{"ollama"}},
				},
				Dependencies: []types.DeploymentDependencyNode{
					{
						Name:         "ollama",
						Type:         "resource",
						Alternatives: []string{"openai", "anthropic"},
						TierSupport: map[string]types.TierSupportSummary{
							"mobile": {
								Supported:    boolPtr(false),
								FitnessScore: &fitness,
								Notes:        "Too heavy for mobile",
							},
						},
					},
				},
			},
		}

		recs := buildTierBlockerRecommendations("ai-app", analysis, timestamp)
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}

		rec := recs[0]
		if rec.RecommendationType != "resource_swap" {
			t.Errorf("expected resource_swap, got %s", rec.RecommendationType)
		}
		if rec.RecommendedState["suggested_alternative"] != "openai" {
			t.Errorf("expected first alternative 'openai', got %v", rec.RecommendedState["suggested_alternative"])
		}
		if rec.Priority != "high" {
			t.Errorf("expected high priority for resource blocker, got %s", rec.Priority)
		}
	})

	t.Run("ScenarioBlockerMediumPriority", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			DeploymentReport: &types.DeploymentAnalysisReport{
				Aggregates: map[string]types.DeploymentTierAggregate{
					"desktop": {BlockingDependencies: []string{"heavy-service"}},
				},
				Dependencies: []types.DeploymentDependencyNode{
					{
						Name: "heavy-service",
						Type: "scenario",
						TierSupport: map[string]types.TierSupportSummary{
							"desktop": {Supported: boolPtr(false), FitnessScore: &fitness},
						},
					},
				},
			},
		}

		recs := buildTierBlockerRecommendations("main-app", analysis, timestamp)
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}

		rec := recs[0]
		if rec.Priority != "medium" {
			t.Errorf("expected medium priority for scenario blocker, got %s", rec.Priority)
		}
	})

	t.Run("MultipleTiers", func(t *testing.T) {
		analysis := &types.DependencyAnalysisResponse{
			DeploymentReport: &types.DeploymentAnalysisReport{
				Aggregates: map[string]types.DeploymentTierAggregate{
					"mobile":  {BlockingDependencies: []string{"postgres"}},
					"desktop": {BlockingDependencies: []string{"postgres"}},
				},
				Dependencies: []types.DeploymentDependencyNode{
					{
						Name:         "postgres",
						Type:         "resource",
						Alternatives: []string{"sqlite"},
						TierSupport: map[string]types.TierSupportSummary{
							"mobile":  {Supported: boolPtr(false), FitnessScore: &fitness},
							"desktop": {Supported: boolPtr(false), FitnessScore: &fitness},
						},
					},
				},
			},
		}

		recs := buildTierBlockerRecommendations("multi-tier-app", analysis, timestamp)
		if len(recs) != 2 {
			t.Errorf("expected 2 recommendations (one per tier), got %d", len(recs))
		}
	})
}

// TestBuildSecretStrategyRecommendations tests secret strategy recommendation generation.
func TestBuildSecretStrategyRecommendations(t *testing.T) {
	timestamp := time.Now()

	t.Run("NilConfig", func(t *testing.T) {
		recs := buildSecretStrategyRecommendations("test", nil, timestamp)
		if recs != nil {
			t.Errorf("expected nil for nil config, got %v", recs)
		}
	})

	t.Run("NilDeployment", func(t *testing.T) {
		cfg := &types.ServiceConfig{}
		recs := buildSecretStrategyRecommendations("test", cfg, timestamp)
		if recs != nil {
			t.Errorf("expected nil for nil deployment, got %v", recs)
		}
	})

	t.Run("NoTiers", func(t *testing.T) {
		cfg := &types.ServiceConfig{Deployment: &types.ServiceDeployment{}}
		recs := buildSecretStrategyRecommendations("test", cfg, timestamp)
		if recs != nil {
			t.Errorf("expected nil for no tiers, got %v", recs)
		}
	})

	t.Run("SecretWithStrategy", func(t *testing.T) {
		cfg := &types.ServiceConfig{
			Deployment: &types.ServiceDeployment{
				Tiers: map[string]types.DeploymentTier{
					"desktop": {
						Secrets: []types.DeploymentTierSecret{
							{SecretID: "api-key", StrategyRef: "secrets-manager://playbook/auto-rotate"},
						},
					},
				},
			},
		}
		recs := buildSecretStrategyRecommendations("test", cfg, timestamp)
		if len(recs) != 0 {
			t.Errorf("expected 0 recommendations for secret with strategy, got %d", len(recs))
		}
	})

	t.Run("SecretWithoutStrategy", func(t *testing.T) {
		cfg := &types.ServiceConfig{
			Deployment: &types.ServiceDeployment{
				Tiers: map[string]types.DeploymentTier{
					"desktop": {
						Secrets: []types.DeploymentTierSecret{
							{SecretID: "api-key", StrategyRef: ""},
							{SecretID: "db-password", Classification: "infrastructure"},
						},
					},
				},
			},
		}
		recs := buildSecretStrategyRecommendations("my-app", cfg, timestamp)
		if len(recs) != 2 {
			t.Fatalf("expected 2 recommendations, got %d", len(recs))
		}

		for _, rec := range recs {
			if rec.RecommendationType != "performance_improvement" {
				t.Errorf("expected performance_improvement, got %s", rec.RecommendationType)
			}
			if rec.Priority != "high" {
				t.Errorf("expected high priority, got %s", rec.Priority)
			}
			if rec.RecommendedState["action"] != "annotate_secret_strategy" {
				t.Errorf("expected annotate_secret_strategy action, got %v", rec.RecommendedState["action"])
			}
		}
	})

	t.Run("MultipleTiersMultipleSecrets", func(t *testing.T) {
		cfg := &types.ServiceConfig{
			Deployment: &types.ServiceDeployment{
				Tiers: map[string]types.DeploymentTier{
					"desktop": {
						Secrets: []types.DeploymentTierSecret{
							{SecretID: "desktop-key"},
						},
					},
					"server": {
						Secrets: []types.DeploymentTierSecret{
							{SecretID: "server-key"},
						},
					},
				},
			},
		}
		recs := buildSecretStrategyRecommendations("multi-tier", cfg, timestamp)
		if len(recs) != 2 {
			t.Errorf("expected 2 recommendations (one per tier secret), got %d", len(recs))
		}
	})
}

// TestBuildDependencyNodeIndex tests the node index builder.
func TestBuildDependencyNodeIndex(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		index := buildDependencyNodeIndex(nil)
		if len(index) != 0 {
			t.Errorf("expected empty index, got %d entries", len(index))
		}
	})

	t.Run("FlatList", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{Name: "postgres", Type: "resource"},
			{Name: "redis", Type: "resource"},
			{Name: "auth-service", Type: "scenario"},
		}
		index := buildDependencyNodeIndex(nodes)
		if len(index) != 3 {
			t.Errorf("expected 3 entries, got %d", len(index))
		}
		if _, ok := index["postgres"]; !ok {
			t.Error("expected postgres in index")
		}
	})

	t.Run("NestedNodes", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{
				Name: "auth-service",
				Type: "scenario",
				Children: []types.DeploymentDependencyNode{
					{Name: "postgres", Type: "resource"},
					{
						Name: "secrets-manager",
						Type: "scenario",
						Children: []types.DeploymentDependencyNode{
							{Name: "vault", Type: "resource"},
						},
					},
				},
			},
		}
		index := buildDependencyNodeIndex(nodes)
		if len(index) != 4 {
			t.Errorf("expected 4 entries (all nested), got %d", len(index))
		}
		if _, ok := index["vault"]; !ok {
			t.Error("expected vault in index (deeply nested)")
		}
	})

	t.Run("KeysAreLowercase", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{Name: "PostgreSQL", Type: "resource"},
			{Name: "REDIS", Type: "resource"},
		}
		index := buildDependencyNodeIndex(nodes)
		if _, ok := index["postgresql"]; !ok {
			t.Error("expected lowercase postgresql in index")
		}
		if _, ok := index["redis"]; !ok {
			t.Error("expected lowercase redis in index")
		}
	})

	t.Run("EmptyNamesIgnored", func(t *testing.T) {
		nodes := []types.DeploymentDependencyNode{
			{Name: "", Type: "resource"},
			{Name: "postgres", Type: "resource"},
		}
		index := buildDependencyNodeIndex(nodes)
		if len(index) != 1 {
			t.Errorf("expected 1 entry (empty name ignored), got %d", len(index))
		}
	})

	t.Run("FirstWins", func(t *testing.T) {
		// When duplicate names exist, first one wins
		nodes := []types.DeploymentDependencyNode{
			{Name: "postgres", Type: "resource", Notes: "first"},
			{Name: "postgres", Type: "resource", Notes: "second"},
		}
		index := buildDependencyNodeIndex(nodes)
		if index["postgres"].Notes != "first" {
			t.Errorf("expected first node to win, got notes=%s", index["postgres"].Notes)
		}
	})
}

func boolPtr(v bool) *bool { return &v }
