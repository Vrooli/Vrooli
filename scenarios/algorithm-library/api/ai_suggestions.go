package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OllamaSuggestionClient handles AI-powered algorithm suggestions via the
// resource-ollama gateway CLI. All daemon traffic goes through the CLI so the
// host-wide semaphore can bound fleet-wide parallelism — never call Ollama
// HTTP directly.
type OllamaSuggestionClient struct {
	model string
}

// SuggestionRequest represents a request for algorithm suggestions
type SuggestionRequest struct {
	ProblemDescription string `json:"problem_description"`
	MaxSuggestions     int    `json:"max_suggestions,omitempty"`
}

// SuggestionResponse represents algorithm suggestions
type SuggestionResponse struct {
	Suggestions []AlgorithmSuggestion `json:"suggestions"`
	Reasoning   string                `json:"reasoning"`
}

// AlgorithmSuggestion represents a single algorithm suggestion
type AlgorithmSuggestion struct {
	AlgorithmID   string  `json:"algorithm_id"`
	AlgorithmName string  `json:"algorithm_name"`
	Category      string  `json:"category"`
	Confidence    float64 `json:"confidence"`
	Explanation   string  `json:"explanation"`
}

// NewOllamaSuggestionClient creates a new Ollama client for suggestions.
func NewOllamaSuggestionClient() *OllamaSuggestionClient {
	role := os.Getenv("OLLAMA_ROLE")
	if role == "" {
		role = "chat.small"
	}
	return &OllamaSuggestionClient{model: role}
}

// IsAvailable checks if the resource-ollama daemon is reachable.
func (c *OllamaSuggestionClient) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "resource-ollama", "status")
	return cmd.Run() == nil
}

// GetSuggestions gets algorithm suggestions based on problem description
func (c *OllamaSuggestionClient) GetSuggestions(problemDesc string, maxSuggestions int) (*SuggestionResponse, error) {
	if maxSuggestions == 0 {
		maxSuggestions = 3
	}

	// Get available algorithms from database
	algorithms, err := c.getAvailableAlgorithms()
	if err != nil {
		return nil, fmt.Errorf("failed to get available algorithms: %v", err)
	}

	// Create prompt for Ollama
	prompt := c.createPrompt(problemDesc, algorithms, maxSuggestions)

	// Call resource-ollama gateway generate
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "resource-ollama", "gateway", "generate",
		"--role", c.model, "--json", "--prompt-stdin")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("resource-ollama gateway generate failed: %v: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("resource-ollama gateway generate failed: %v", err)
	}

	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway generate response: %w", err)
	}

	// Parse Ollama response and match with database algorithms
	return c.parseOllamaResponse(decoded.Response, algorithms)
}

func (c *OllamaSuggestionClient) getAvailableAlgorithms() ([]Algorithm, error) {
	rows, err := db.Query(`
		SELECT id, name, display_name, category, description, 
		       complexity_time, complexity_space, difficulty, tags
		FROM algorithms
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var algorithms []Algorithm
	for rows.Next() {
		var algo Algorithm
		var tags string
		err := rows.Scan(
			&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
			&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
			&algo.Difficulty, &tags,
		)
		if err != nil {
			continue
		}
		if tags != "" {
			algo.Tags = strings.Split(tags, ",")
		}
		algorithms = append(algorithms, algo)
	}

	return algorithms, nil
}

func (c *OllamaSuggestionClient) createPrompt(problemDesc string, algorithms []Algorithm, maxSuggestions int) string {
	// Create algorithm list for context
	algoList := []string{}
	for _, algo := range algorithms {
		algoList = append(algoList, fmt.Sprintf(
			"- %s (%s): %s [Time: %s, Space: %s]",
			algo.Name, algo.Category, algo.Description,
			algo.ComplexityTime, algo.ComplexitySpace,
		))
	}

	prompt := fmt.Sprintf(`You are an algorithm expert. Given a problem description, suggest the most appropriate algorithms from the available list.

Problem Description:
%s

Available Algorithms:
%s

Please suggest the top %d most relevant algorithms for solving this problem. For each suggestion, provide:
1. The algorithm name (exactly as it appears in the list)
2. Why this algorithm is suitable
3. A confidence score (0.0-1.0)

Format your response as a JSON array with objects containing: algorithm_name, explanation, and confidence.
Example:
[
  {"algorithm_name": "quicksort", "explanation": "Efficient for sorting large datasets", "confidence": 0.9},
  {"algorithm_name": "mergesort", "explanation": "Stable sorting with guaranteed O(n log n)", "confidence": 0.8}
]

Respond ONLY with the JSON array, no additional text.`,
		problemDesc,
		strings.Join(algoList, "\n"),
		maxSuggestions,
	)

	return prompt
}

func (c *OllamaSuggestionClient) parseOllamaResponse(response string, algorithms []Algorithm) (*SuggestionResponse, error) {
	// Try to extract JSON from the response
	response = strings.TrimSpace(response)

	// Find JSON array in response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// Fallback: provide generic suggestions based on keywords
		return c.fallbackSuggestions(response, algorithms), nil
	}

	jsonStr := response[startIdx : endIdx+1]

	var rawSuggestions []struct {
		AlgorithmName string  `json:"algorithm_name"`
		Explanation   string  `json:"explanation"`
		Confidence    float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		// Fallback on parse error
		return c.fallbackSuggestions(response, algorithms), nil
	}

	// Map to actual algorithms
	suggestions := []AlgorithmSuggestion{}
	for _, raw := range rawSuggestions {
		for _, algo := range algorithms {
			if strings.EqualFold(algo.Name, raw.AlgorithmName) ||
				strings.EqualFold(algo.DisplayName, raw.AlgorithmName) {
				suggestions = append(suggestions, AlgorithmSuggestion{
					AlgorithmID:   algo.ID,
					AlgorithmName: algo.DisplayName,
					Category:      algo.Category,
					Confidence:    raw.Confidence,
					Explanation:   raw.Explanation,
				})
				break
			}
		}
	}

	return &SuggestionResponse{
		Suggestions: suggestions,
		Reasoning:   "AI-powered suggestions based on problem description analysis",
	}, nil
}

func (c *OllamaSuggestionClient) fallbackSuggestions(response string, algorithms []Algorithm) *SuggestionResponse {
	// Simple keyword matching fallback
	lowerResponse := strings.ToLower(response)
	suggestions := []AlgorithmSuggestion{}

	// Keywords to algorithm mapping
	keywordMap := map[string][]string{
		"sort":    {"sorting"},
		"search":  {"searching"},
		"graph":   {"graph"},
		"tree":    {"tree"},
		"dynamic": {"dynamic_programming"},
		"greedy":  {"greedy"},
		"divide":  {"divide_conquer"},
	}

	for keyword, categories := range keywordMap {
		if strings.Contains(lowerResponse, keyword) {
			for _, algo := range algorithms {
				for _, cat := range categories {
					if algo.Category == cat && len(suggestions) < 3 {
						suggestions = append(suggestions, AlgorithmSuggestion{
							AlgorithmID:   algo.ID,
							AlgorithmName: algo.DisplayName,
							Category:      algo.Category,
							Confidence:    0.5,
							Explanation:   fmt.Sprintf("Keyword match for '%s'", keyword),
						})
						break
					}
				}
			}
		}
	}

	// If no matches, suggest popular algorithms
	if len(suggestions) == 0 {
		popularAlgos := []string{"quicksort", "binarysearch", "dfs"}
		for _, name := range popularAlgos {
			for _, algo := range algorithms {
				if algo.Name == name {
					suggestions = append(suggestions, AlgorithmSuggestion{
						AlgorithmID:   algo.ID,
						AlgorithmName: algo.DisplayName,
						Category:      algo.Category,
						Confidence:    0.3,
						Explanation:   "Popular algorithm for general problems",
					})
					break
				}
			}
		}
	}

	return &SuggestionResponse{
		Suggestions: suggestions,
		Reasoning:   "Keyword-based fallback suggestions",
	}
}

// HTTP handler for algorithm suggestions
func algorithmSuggestHandler(w http.ResponseWriter, r *http.Request) {
	var req SuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.ProblemDescription == "" {
		http.Error(w, "Problem description is required", http.StatusBadRequest)
		return
	}

	// Create Ollama client
	client := NewOllamaSuggestionClient()

	// Check if Ollama is available
	if !client.IsAvailable() {
		// Provide fallback suggestions without AI
		suggestions := provideFallbackSuggestions(req.ProblemDescription)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(suggestions)
		return
	}

	// Get AI-powered suggestions
	suggestions, err := client.GetSuggestions(req.ProblemDescription, req.MaxSuggestions)
	if err != nil {
		log.Printf("Error getting AI suggestions: %v", err)
		// Provide fallback suggestions
		suggestions = provideFallbackSuggestions(req.ProblemDescription)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

func provideFallbackSuggestions(problemDesc string) *SuggestionResponse {
	lowerDesc := strings.ToLower(problemDesc)
	suggestions := []AlgorithmSuggestion{}

	// Simple keyword-based suggestions
	if strings.Contains(lowerDesc, "sort") {
		suggestions = append(suggestions, AlgorithmSuggestion{
			AlgorithmName: "QuickSort",
			Category:      "sorting",
			Confidence:    0.7,
			Explanation:   "Efficient general-purpose sorting algorithm",
		})
	}

	if strings.Contains(lowerDesc, "search") || strings.Contains(lowerDesc, "find") {
		suggestions = append(suggestions, AlgorithmSuggestion{
			AlgorithmName: "Binary Search",
			Category:      "searching",
			Confidence:    0.7,
			Explanation:   "Efficient search in sorted data",
		})
	}

	if strings.Contains(lowerDesc, "graph") || strings.Contains(lowerDesc, "path") {
		suggestions = append(suggestions, AlgorithmSuggestion{
			AlgorithmName: "Dijkstra",
			Category:      "graph",
			Confidence:    0.6,
			Explanation:   "Shortest path in weighted graphs",
		})
	}

	if strings.Contains(lowerDesc, "tree") {
		suggestions = append(suggestions, AlgorithmSuggestion{
			AlgorithmName: "DFS",
			Category:      "tree",
			Confidence:    0.6,
			Explanation:   "Tree traversal and exploration",
		})
	}

	return &SuggestionResponse{
		Suggestions: suggestions,
		Reasoning:   "Keyword-based suggestions (AI service unavailable)",
	}
}
