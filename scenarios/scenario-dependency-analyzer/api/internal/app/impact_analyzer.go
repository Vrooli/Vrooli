package app

import (
	"fmt"
	"sort"
)

// DependencyImpactReport represents the analysis of removing a dependency
type DependencyImpactReport struct {
	DependencyName     string              `json:"dependency_name"`
	DependencyType     string              `json:"dependency_type"` // "resource", "scenario"
	DirectDependents   []DependentScenario `json:"direct_dependents"`
	IndirectDependents []DependentScenario `json:"indirect_dependents"`
	TotalAffected      int                 `json:"total_affected"`
	CriticalImpact     bool                `json:"critical_impact"` // true if any required dependency
	Severity           string              `json:"severity"`        // "none", "low", "medium", "high", "critical"
	ImpactSummary      string              `json:"impact_summary"`
	Recommendations    []string            `json:"recommendations"`
}

// DependentScenario represents a scenario that depends on the analyzed dependency
type DependentScenario struct {
	ScenarioName string                 `json:"scenario_name"`
	Required     bool                   `json:"required"`
	Purpose      string                 `json:"purpose"`
	AccessMethod string                 `json:"access_method"`
	Alternatives []string               `json:"alternatives,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// analyzeDependencyImpact performs impact analysis for removing a dependency
func analyzeDependencyImpact(dependencyName string) (*DependencyImpactReport, error) {
	if dependencyName == "" {
		return nil, fmt.Errorf("dependency name is required")
	}

	// Query database for direct dependents
	directDeps, err := findDirectDependents(dependencyName)
	if err != nil {
		return nil, fmt.Errorf("failed to find direct dependents: %w", err)
	}

	// Find indirect dependents (scenarios that depend on direct dependents)
	indirectDeps := findIndirectDependents(directDeps)

	// Determine dependency type
	depType := determineDependencyType(dependencyName, directDeps)

	// Calculate severity
	severity, criticalImpact := calculateImpactSeverity(directDeps, indirectDeps)

	// Generate recommendations
	recommendations := generateImpactRecommendations(dependencyName, depType, directDeps, severity)

	// Create summary
	summary := generateImpactSummary(dependencyName, depType, len(directDeps), len(indirectDeps), severity)

	return &DependencyImpactReport{
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

// findDirectDependents queries the database for scenarios that directly depend on the given dependency
func findDirectDependents(dependencyName string) ([]DependentScenario, error) {
	dbConn := currentDB()
	if dbConn == nil {
		return []DependentScenario{}, nil
	}

	rows, err := dbConn.Query(`
		SELECT scenario_name, required, purpose, access_method, configuration
		FROM scenario_dependencies
		WHERE dependency_name = $1
		ORDER BY scenario_name`, dependencyName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependents []DependentScenario
	for rows.Next() {
		var scenarioName, purpose, accessMethod string
		var required bool
		var configJSON []byte

		if err := rows.Scan(&scenarioName, &required, &purpose, &accessMethod, &configJSON); err != nil {
			continue
		}

		dependent := DependentScenario{
			ScenarioName: scenarioName,
			Required:     required,
			Purpose:      purpose,
			AccessMethod: accessMethod,
			Metadata:     make(map[string]interface{}),
		}

		// Add alternatives if this is a resource that could be swapped
		if alternatives := suggestAlternatives(dependencyName); len(alternatives) > 0 {
			dependent.Alternatives = alternatives
		}

		dependents = append(dependents, dependent)
	}

	return dependents, nil
}

// findIndirectDependents finds scenarios that depend on direct dependents
func findIndirectDependents(directDeps []DependentScenario) []DependentScenario {
	dbConn := currentDB()
	if dbConn == nil || len(directDeps) == 0 {
		return []DependentScenario{}
	}

	// Build list of scenario names to check
	scenarioNames := make([]string, 0, len(directDeps))
	for _, dep := range directDeps {
		scenarioNames = append(scenarioNames, dep.ScenarioName)
	}

	var indirectDeps []DependentScenario
	seen := make(map[string]bool)

	// Mark direct dependents as seen to avoid duplication
	for _, dep := range directDeps {
		seen[dep.ScenarioName] = true
	}

	// For each direct dependent, find what depends on it
	for _, scenarioName := range scenarioNames {
		rows, err := dbConn.Query(`
			SELECT scenario_name, required, purpose, access_method
			FROM scenario_dependencies
			WHERE dependency_name = $1 AND dependency_type = 'scenario'
			ORDER BY scenario_name`, scenarioName)
		if err != nil {
			continue
		}

		for rows.Next() {
			var depScenarioName, purpose, accessMethod string
			var required bool

			if err := rows.Scan(&depScenarioName, &required, &purpose, &accessMethod); err != nil {
				continue
			}

			// Skip if already seen
			if seen[depScenarioName] {
				continue
			}
			seen[depScenarioName] = true

			indirectDeps = append(indirectDeps, DependentScenario{
				ScenarioName: depScenarioName,
				Required:     required,
				Purpose:      purpose,
				AccessMethod: accessMethod,
				Metadata: map[string]interface{}{
					"dependency_chain": fmt.Sprintf("%s → %s", scenarioName, depScenarioName),
				},
			})
		}
		rows.Close()
	}

	// Sort for consistent output
	sort.Slice(indirectDeps, func(i, j int) bool {
		return indirectDeps[i].ScenarioName < indirectDeps[j].ScenarioName
	})

	return indirectDeps
}

// determineDependencyType identifies if this is a resource or scenario dependency
func determineDependencyType(dependencyName string, directDeps []DependentScenario) string {
	// Check if it's a known scenario
	if isKnownScenario(dependencyName) {
		return "scenario"
	}

	// If any direct dependent lists it as a resource, it's a resource
	for _, dep := range directDeps {
		if dep.Metadata != nil {
			if dt, ok := dep.Metadata["dependency_type"].(string); ok && dt == "resource" {
				return "resource"
			}
		}
	}

	// Check known resource patterns
	if isLikelyResource(dependencyName) {
		return "resource"
	}

	return "unknown"
}

// isLikelyResource checks if the name follows resource naming patterns
func isLikelyResource(name string) bool {
	resourcePatterns := []string{"postgres", "redis", "qdrant", "claude-code", "ollama", "n8n", "minio", "browserless", "judge0", "vault"}
	for _, pattern := range resourcePatterns {
		if name == pattern {
			return true
		}
	}
	return false
}

// calculateImpactSeverity determines the severity level of removing this dependency
func calculateImpactSeverity(directDeps, indirectDeps []DependentScenario) (string, bool) {
	totalAffected := len(directDeps) + len(indirectDeps)

	// Count required vs optional dependencies
	requiredCount := 0
	for _, dep := range directDeps {
		if dep.Required {
			requiredCount++
		}
	}

	// Determine criticality
	criticalImpact := requiredCount > 0

	// Calculate severity
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

// generateImpactRecommendations creates actionable recommendations
func generateImpactRecommendations(dependencyName, depType string, directDeps []DependentScenario, severity string) []string {
	var recommendations []string

	if len(directDeps) == 0 {
		recommendations = append(recommendations, fmt.Sprintf("No scenarios currently depend on '%s'. Safe to remove if unused.", dependencyName))
		return recommendations
	}

	// Severity-based recommendations
	if severity == "critical" || severity == "high" {
		recommendations = append(recommendations, fmt.Sprintf("⚠️ HIGH IMPACT: Removing '%s' will break %d scenario(s).", dependencyName, len(directDeps)))
		recommendations = append(recommendations, "Consider migrating dependent scenarios before removal.")
	}

	// Check for alternatives
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

	// Required dependency recommendations
	requiredCount := 0
	for _, dep := range directDeps {
		if dep.Required {
			requiredCount++
		}
	}

	if requiredCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d scenario(s) mark this as a REQUIRED dependency. Update service.json before removal.", requiredCount))
	}

	// Type-specific recommendations
	if depType == "resource" {
		recommendations = append(recommendations, "If removing a resource, ensure all dependent scenarios have migrated to alternatives or been decommissioned.")
	} else if depType == "scenario" {
		recommendations = append(recommendations, "If deprecating this scenario, coordinate with dependent scenarios to prevent breakage.")
	}

	return recommendations
}

// generateImpactSummary creates a concise summary message
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

// suggestAlternatives returns potential alternatives for common dependencies
func suggestAlternatives(dependencyName string) []string {
	// Common resource alternatives
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
