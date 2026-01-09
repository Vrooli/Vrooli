package app

import (
	"fmt"
	"sort"
	"strings"

	types "scenario-dependency-analyzer/internal/types"
)

// analyzeDependencyImpact performs impact analysis for removing a dependency using the provided catalog snapshot.
func analyzeDependencyImpact(dependencyName string, deps []types.ScenarioDependency, knownScenarios map[string]struct{}) (*types.DependencyImpactReport, error) {
	if dependencyName == "" {
		return nil, fmt.Errorf("dependency name is required")
	}

	directDeps := directDependentsFrom(dependencyName, deps)
	indirectDeps := indirectDependentsFrom(directDeps, deps)

	depType := determineDependencyType(dependencyName, directDeps, knownScenarios)
	severity, criticalImpact := calculateImpactSeverity(directDeps, indirectDeps)
	recommendations := generateImpactRecommendations(dependencyName, depType, directDeps, severity)
	summary := generateImpactSummary(dependencyName, depType, len(directDeps), len(indirectDeps), severity)

	return &types.DependencyImpactReport{
		DependencyName:     dependencyName,
		DependencyType:     depType,
		DirectDependents:   directDeps,
		IndirectDependents: indirectDeps,
		TotalAffected:      len(directDeps) + len(indirectDeps),
		CriticalImpact:     criticalImpact,
		Severity:           severity,
		ImpactSummary:      summary,
		Recommendations:    recommendations,
	}, nil
}

// directDependentsFrom collects the scenarios that directly depend on the provided dependency.
func directDependentsFrom(dependencyName string, deps []types.ScenarioDependency) []types.DependentScenario {
	if len(deps) == 0 {
		return []types.DependentScenario{}
	}

	normalized := normalizeDependencyName(dependencyName)
	result := make([]types.DependentScenario, 0)

	for _, dep := range deps {
		if normalizeDependencyName(dep.DependencyName) != normalized {
			continue
		}

		metadata := map[string]interface{}{}
		for k, v := range dep.Configuration {
			metadata[k] = v
		}
		metadata["dependency_type"] = dep.DependencyType

		dependent := types.DependentScenario{
			ScenarioName: dep.ScenarioName,
			Required:     dep.Required,
			Purpose:      dep.Purpose,
			AccessMethod: dep.AccessMethod,
			Metadata:     metadata,
		}

		if alternatives := suggestAlternatives(dep.DependencyName); len(alternatives) > 0 {
			dependent.Alternatives = alternatives
		}

		result = append(result, dependent)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].ScenarioName < result[j].ScenarioName })
	return result
}

// indirectDependentsFrom finds scenarios that depend on direct dependents (second-degree impact).
func indirectDependentsFrom(directDeps []types.DependentScenario, deps []types.ScenarioDependency) []types.DependentScenario {
	if len(directDeps) == 0 || len(deps) == 0 {
		return []types.DependentScenario{}
	}

	directSet := make(map[string]struct{}, len(directDeps))
	for _, dep := range directDeps {
		directSet[normalizeDependencyName(dep.ScenarioName)] = struct{}{}
	}

	seen := make(map[string]struct{})
	indirect := make([]types.DependentScenario, 0)

	for _, dep := range deps {
		if dep.DependencyType != "scenario" {
			continue
		}
		// Does this dependency point at one of the direct dependents?
		if _, ok := directSet[normalizeDependencyName(dep.DependencyName)]; !ok {
			continue
		}
		normalizedScenario := normalizeDependencyName(dep.ScenarioName)
		if _, already := seen[normalizedScenario]; already {
			continue
		}
		seen[normalizedScenario] = struct{}{}

		indirect = append(indirect, types.DependentScenario{
			ScenarioName: dep.ScenarioName,
			Required:     dep.Required,
			Purpose:      dep.Purpose,
			AccessMethod: dep.AccessMethod,
			Metadata: map[string]interface{}{
				"dependency_chain": fmt.Sprintf("%s → %s", dep.DependencyName, dep.ScenarioName),
			},
		})
	}

	sort.Slice(indirect, func(i, j int) bool { return indirect[i].ScenarioName < indirect[j].ScenarioName })
	return indirect
}

// determineDependencyType identifies if this is a resource or scenario dependency.
func determineDependencyType(dependencyName string, directDeps []types.DependentScenario, knownScenarios map[string]struct{}) string {
	if scenarioIsKnown(dependencyName, knownScenarios) {
		return "scenario"
	}

	for _, dep := range directDeps {
		if dep.Metadata != nil {
			if dt, ok := dep.Metadata["dependency_type"].(string); ok && dt != "" {
				if dt == "resource" || dt == "scenario" {
					return dt
				}
			}
		}
	}

	if isLikelyResource(dependencyName) {
		return "resource"
	}

	return "unknown"
}

// isLikelyResource checks if the name follows resource naming patterns.
func isLikelyResource(name string) bool {
	resourcePatterns := []string{"postgres", "redis", "qdrant", "claude-code", "ollama", "n8n", "minio", "browserless", "judge0", "vault"}
	for _, pattern := range resourcePatterns {
		if name == pattern {
			return true
		}
	}
	return false
}

// calculateImpactSeverity determines the severity level of removing this dependency.
func calculateImpactSeverity(directDeps, indirectDeps []types.DependentScenario) (string, bool) {
	totalAffected := len(directDeps) + len(indirectDeps)

	requiredCount := 0
	for _, dep := range directDeps {
		if dep.Required {
			requiredCount++
		}
	}

	criticalImpact := requiredCount > 0

	if requiredCount >= 5 || (requiredCount > 0 && totalAffected >= 10) {
		return "critical", criticalImpact
	} else if requiredCount >= 3 || totalAffected >= 8 {
		return "high", criticalImpact
	} else if requiredCount >= 1 || totalAffected >= 5 {
		return "medium", criticalImpact
	} else if totalAffected >= 2 {
		return "low", criticalImpact
	}

	return "none", criticalImpact
}

// generateImpactRecommendations creates actionable recommendations.
func generateImpactRecommendations(dependencyName, depType string, directDeps []types.DependentScenario, severity string) []string {
	var recommendations []string

	if len(directDeps) == 0 {
		recommendations = append(recommendations, fmt.Sprintf("No scenarios currently depend on '%s'. Safe to remove if unused.", dependencyName))
		return recommendations
	}

	if severity == "critical" || severity == "high" {
		recommendations = append(recommendations, fmt.Sprintf("⚠️ HIGH IMPACT: Removing '%s' will break %d scenario(s).", dependencyName, len(directDeps)))
		recommendations = append(recommendations, "Consider migrating dependent scenarios before removal.")
	}

	hasAlternatives := false
	for _, dep := range directDeps {
		if len(dep.Alternatives) > 0 {
			hasAlternatives = true
			break
		}
	}

	if hasAlternatives {
		recommendations = append(recommendations, "Alternative dependencies are available for some scenarios. Review 'alternatives' field for swap options.")
	}

	requiredCount := 0
	for _, dep := range directDeps {
		if dep.Required {
			requiredCount++
		}
	}

	if requiredCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d scenario(s) mark this as a REQUIRED dependency. Update service.json before removal.", requiredCount))
	}

	if depType == "resource" {
		recommendations = append(recommendations, "If removing a resource, ensure all dependent scenarios have migrated to alternatives or been decommissioned.")
	} else if depType == "scenario" {
		recommendations = append(recommendations, "If deprecating this scenario, coordinate with dependent scenarios to prevent breakage.")
	}

	return recommendations
}

// generateImpactSummary creates a concise summary message.
func generateImpactSummary(dependencyName, depType string, directCount, indirectCount int, severity string) string {
	if directCount == 0 && indirectCount == 0 {
		return fmt.Sprintf("No scenarios depend on '%s'. Safe to remove.", dependencyName)
	}

	totalCount := directCount + indirectCount
	depTypeLabel := depType
	if depTypeLabel == "unknown" {
		depTypeLabel = "dependency"
	}

	return fmt.Sprintf("Removing %s '%s' would affect %d scenario(s): %d direct, %d indirect. Impact severity: %s",
		depTypeLabel, dependencyName, totalCount, directCount, indirectCount, severity)
}

// suggestAlternatives returns potential alternatives for common dependencies.
func suggestAlternatives(dependencyName string) []string {
	alternatives := map[string][]string{
		"postgres":    {"sqlite", "mysql"},
		"redis":       {"memcached", "in-memory-cache"},
		"ollama":      {"openrouter", "claude-code", "openai"},
		"qdrant":      {"pinecone", "weaviate", "chromadb"},
		"n8n":         {"zapier", "make", "custom-workflows"},
		"minio":       {"s3", "local-filesystem"},
		"browserless": {"puppeteer", "playwright"},
	}

	if alts, ok := alternatives[dependencyName]; ok {
		return alts
	}

	return []string{}
}

func normalizeDependencyName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func scenarioIsKnown(name string, known map[string]struct{}) bool {
	if len(known) == 0 {
		return true
	}
	_, ok := known[normalizeDependencyName(name)]
	return ok
}
