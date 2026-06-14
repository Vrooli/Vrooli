package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/integrations/ollama"
	"scenario-dependency-analyzer/internal/integrations/qdrant"
)

// Integrate with Qdrant for semantic similarity matching
func findSimilarScenariosQdrant(description string, existingScenarios []string) ([]map[string]interface{}, error) {
	if strings.TrimSpace(os.Getenv("USE_RESOURCE_QDRANT_CLI")) == "true" {
		return findSimilarScenariosQdrantViaCLI(description, existingScenarios)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ollamaClient := ollama.NewEmbedderFromEnv()
	qdrantClient := qdrant.NewClientFromEnv(nil)

	collection := strings.TrimSpace(os.Getenv("SCENARIO_EMBEDDINGS_COLLECTION"))
	if collection == "" {
		collection = "scenario_embeddings"
	}

	embedding, err := ollamaClient.Embed(ctx, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if err := qdrantClient.EnsureCollection(ctx, collection, len(embedding)); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	results, err := qdrantClient.Search(ctx, collection, embedding, 5)
	if err != nil {
		// Qdrant search failed - return empty matches rather than error
		// This allows the analysis to continue with other methods
		return []map[string]interface{}{}, nil
	}

	var matches []map[string]interface{}

	for _, result := range results {
		if result.Score <= 0.7 {
			continue
		}

		scenarioName, _ := result.Payload["scenario_name"].(string)
		desc, _ := result.Payload["description"].(string)
		resources := coerceStringSlice(result.Payload["resources"])

		matches = append(matches, map[string]interface{}{
			"scenario_name": scenarioName,
			"similarity":    result.Score,
			"resources":     resources,
			"description":   desc,
		})
	}

	return matches, nil
}

func findSimilarScenariosQdrantViaCLI(description string, existingScenarios []string) ([]map[string]interface{}, error) {
	var matches []map[string]interface{}

	embeddingCmd := exec.Command("resource-qdrant", "embed", description) // #nosec G204 -- executable is fixed; description is passed as an argument, not through a shell.
	embeddingOutput, err := embeddingCmd.Output()
	if err != nil {
		return matches, fmt.Errorf("failed to create embedding: %w", err)
	}

	searchCmd := exec.Command("resource-qdrant", "search", // #nosec G204 -- executable and flags are fixed; vector content is passed as an argument, not through a shell.
		"--collection", "scenario_embeddings",
		"--vector", string(embeddingOutput),
		"--limit", "5",
		"--output", "json")

	searchOutput, err := searchCmd.Output()
	if err != nil {
		return matches, nil
	}

	var searchResults QdrantSearchResults
	if err := json.Unmarshal(searchOutput, &searchResults); err != nil {
		return matches, fmt.Errorf("failed to parse qdrant results: %w", err)
	}

	for _, result := range searchResults.Matches {
		if result.Score > 0.7 {
			matches = append(matches, map[string]interface{}{
				"scenario_name": result.ScenarioName,
				"similarity":    result.Score,
				"resources":     result.Resources,
				"description":   result.Description,
			})
		}
	}

	return matches, nil
}

func coerceStringSlice(v interface{}) []string {
	switch typed := v.(type) {
	case []string:
		return typed
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, raw := range typed {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Helper functions for analysis
func getHeuristicPredictions(description string) []map[string]interface{} {
	var predictions []map[string]interface{}

	heuristics := map[string][]string{
		"postgres": {"data", "database", "store", "persist", "sql", "table"},
		"redis":    {"cache", "session", "temporary", "fast", "memory"},
		"ollama":   {"ai", "llm", "language model", "chat", "generate", "intelligent"},
		"n8n":      {"workflow", "automation", "process", "trigger", "orchestrate"},
		"qdrant":   {"vector", "semantic", "search", "similarity", "embedding"},
		"minio":    {"file", "upload", "storage", "document", "asset", "image"},
	}

	for resource, keywords := range heuristics {
		confidence := 0.0
		matches := 0

		for _, keyword := range keywords {
			if strings.Contains(description, keyword) {
				matches++
				confidence += 0.1
			}
		}

		if confidence > 0 {
			// Normalize confidence based on number of matches
			confidence = math.Min(confidence, 0.8)

			predictions = append(predictions, map[string]interface{}{
				"resource_name": resource,
				"confidence":    confidence,
				"reasoning":     fmt.Sprintf("Heuristic match: %d keywords detected", matches),
			})
		}
	}

	return predictions
}

func deduplicateResources(resources []map[string]interface{}) []map[string]interface{} {
	seen := make(map[string]float64)
	var deduplicated []map[string]interface{}

	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if existingConfidence, exists := seen[name]; !exists || confidence > existingConfidence {
			seen[name] = confidence
		}
	}

	// Rebuild array with highest confidence for each resource
	for _, resource := range resources {
		name := resource["resource_name"].(string)
		confidence := resource["confidence"].(float64)

		if seen[name] == confidence {
			deduplicated = append(deduplicated, resource)
			delete(seen, name) // Prevent duplicates
		}
	}

	return deduplicated
}

func calculateResourceConfidence(predictions []map[string]interface{}) float64 {
	if len(predictions) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, pred := range predictions {
		totalConfidence += pred["confidence"].(float64)
	}

	return math.Min(totalConfidence/float64(len(predictions)), 1.0)
}

func calculateScenarioConfidence(patterns []map[string]interface{}) float64 {
	if len(patterns) == 0 {
		return 0.0
	}

	totalSimilarity := 0.0
	for _, pattern := range patterns {
		if sim, ok := pattern["similarity"].(float64); ok {
			totalSimilarity += sim
		}
	}

	return math.Min(totalSimilarity/float64(len(patterns)), 1.0)
}

type QdrantSearchResults struct {
	Matches []QdrantMatch `json:"matches"`
}

type QdrantMatch struct {
	ScenarioName string                 `json:"scenario_name"`
	Score        float64                `json:"score"`
	Resources    []string               `json:"resources"`
	Description  string                 `json:"description"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Run boots the HTTP API using the provided configuration and database connection.
// NOTE: Run is defined in server.go to keep this file focused on analysis logic.
