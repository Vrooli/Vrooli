package app

import (
	"log"
	"strings"

	types "scenario-dependency-analyzer/internal/types"
)

func analyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	predictedResources := []map[string]interface{}{}
	similarPatterns := []map[string]interface{}{}
	recommendations := []map[string]interface{}{}

	for _, reqResource := range req.Requirements {
		predictedResources = append(predictedResources, map[string]interface{}{
			"resource_name": reqResource,
			"confidence":    0.9,
			"reasoning":     "Explicitly mentioned in requirements",
		})
	}

	claudeAnalysis, err := analyzeWithClaudeCode(req.Name, req.Description)
	if err != nil {
		log.Printf("Claude Code analysis failed: %v", err)
	} else {
		predictedResources = append(predictedResources, claudeAnalysis.PredictedResources...)
		recommendations = append(recommendations, claudeAnalysis.Recommendations...)
	}

	qdrantMatches, err := findSimilarScenariosQdrant(req.Description, req.SimilarScenarios)
	if err != nil {
		log.Printf("Qdrant similarity matching failed: %v", err)
	} else {
		similarPatterns = qdrantMatches
	}

	description := strings.ToLower(req.Description)
	heuristicResources := getHeuristicPredictions(description)
	predictedResources = append(predictedResources, heuristicResources...)

	resourceConfidence := calculateResourceConfidence(predictedResources)
	scenarioConfidence := calculateScenarioConfidence(similarPatterns)

	return map[string]interface{}{
		"predicted_resources": deduplicateResources(predictedResources),
		"similar_patterns":    similarPatterns,
		"recommendations":     recommendations,
		"confidence_scores": map[string]float64{
			"resource": resourceConfidence,
			"scenario": scenarioConfidence,
		},
	}, nil
}
