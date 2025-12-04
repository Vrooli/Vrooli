package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
)

// normalizeName lowercases and trims whitespace for consistent name comparisons.
func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

// Core analysis functions

// Integrate with Claude Code resource for intelligent analysis
func analyzeWithClaudeCode(scenarioName, description string) (*ClaudeCodeAnalysis, error) {
	// Create a temporary analysis file
	analysisPrompt := fmt.Sprintf(`
Analyze the following proposed Vrooli scenario and predict its likely dependencies:

Scenario Name: %s
Description: %s

Please identify:
1. Likely resource dependencies (postgres, redis, ollama, n8n, etc.)
2. Similar existing scenarios it might depend on
3. Recommended architecture patterns
4. Potential optimization opportunities

Format your response as structured analysis focusing on technical implementation needs.
`, scenarioName, description)

	// Write prompt to temporary file
	tempFile := "/tmp/claude-analysis-" + uuid.New().String() + ".txt"
	if err := os.WriteFile(tempFile, []byte(analysisPrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to create analysis prompt file: %w", err)
	}
	defer os.Remove(tempFile)

	// Execute Claude Code analysis
	cmd := exec.Command("resource-claude-code", "analyze", tempFile, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude-code command failed: %w", err)
	}

	// Parse Claude Code response and extract dependency predictions
	analysis := parseClaudeCodeResponse(string(output), description)
	return analysis, nil
}

// Integrate with Qdrant for semantic similarity matching
func findSimilarScenariosQdrant(description string, existingScenarios []string) ([]map[string]interface{}, error) {
	var matches []map[string]interface{}

	// Create embedding for the proposed scenario description
	embeddingCmd := exec.Command("resource-qdrant", "embed", description)
	embeddingOutput, err := embeddingCmd.Output()
	if err != nil {
		return matches, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Search for similar scenarios in the vector database
	searchCmd := exec.Command("resource-qdrant", "search",
		"--collection", "scenario_embeddings",
		"--vector", string(embeddingOutput),
		"--limit", "5",
		"--output", "json")

	searchOutput, err := searchCmd.Output()
	if err != nil {
		// Qdrant search failed - return empty matches rather than error
		// This allows the analysis to continue with other methods
		return matches, nil
	}

	// Parse Qdrant search results
	var searchResults QdrantSearchResults
	if err := json.Unmarshal(searchOutput, &searchResults); err != nil {
		return matches, fmt.Errorf("failed to parse qdrant results: %w", err)
	}

	// Convert to our format
	for _, result := range searchResults.Matches {
		if result.Score > 0.7 { // Only include high-confidence matches
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

// Data structures for external integrations
type ClaudeCodeAnalysis struct {
	PredictedResources []map[string]interface{} `json:"predicted_resources"`
	Recommendations    []map[string]interface{} `json:"recommendations"`
	ArchitectureNotes  string                   `json:"architecture_notes"`
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

func parseClaudeCodeResponse(response, originalDescription string) *ClaudeCodeAnalysis {
	// Parse Claude Code response and extract structured dependency predictions
	// This is a simplified implementation - in practice, you'd parse the AI response more sophisticatedly

	analysis := &ClaudeCodeAnalysis{
		PredictedResources: []map[string]interface{}{},
		Recommendations:    []map[string]interface{}{},
		ArchitectureNotes:  response,
	}

	// Extract resource mentions from Claude's response
	responseText := strings.ToLower(response)

	resourcePatterns := map[string]float64{
		"postgres":   0.8,
		"postgresql": 0.8,
		"database":   0.7,
		"redis":      0.8,
		"cache":      0.6,
		"ollama":     0.9,
		"llm":        0.7,
		"n8n":        0.8,
		"workflow":   0.6,
		"qdrant":     0.9,
		"vector":     0.7,
		"minio":      0.8,
		"storage":    0.5,
	}

	for pattern, confidence := range resourcePatterns {
		if strings.Contains(responseText, pattern) {
			// Map patterns to actual resource names
			resourceName := mapPatternToResource(pattern)
			if resourceName != "" {
				analysis.PredictedResources = append(analysis.PredictedResources, map[string]interface{}{
					"resource_name": resourceName,
					"confidence":    confidence,
					"reasoning":     fmt.Sprintf("Claude Code analysis mentioned '%s'", pattern),
				})
			}
		}
	}

	return analysis
}

func mapPatternToResource(pattern string) string {
	resourceMap := map[string]string{
		"postgres":   "postgres",
		"postgresql": "postgres",
		"database":   "postgres",
		"redis":      "redis",
		"cache":      "redis",
		"ollama":     "ollama",
		"llm":        "ollama",
		"n8n":        "n8n",
		"workflow":   "n8n",
		"qdrant":     "qdrant",
		"vector":     "qdrant",
		"minio":      "minio",
		"storage":    "minio",
	}

	return resourceMap[pattern]
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
