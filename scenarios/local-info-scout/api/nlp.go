package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
}

// ParsedQuery represents a parsed natural language query
type ParsedQuery struct {
	Category string   `json:"category"`
	Radius   float64  `json:"radius"`
	Keywords []string `json:"keywords"`
}

// parseNaturalLanguageQuery parses a natural language query into structured data
// It attempts to use Ollama for intelligent parsing, with fallback to keyword matching
func parseNaturalLanguageQuery(query string) ParsedQuery {
	parsed := ParsedQuery{
		Category: "",
		Radius:   1.0, // Default 1 mile
		Keywords: []string{},
	}

	// Try to use Ollama for parsing if available
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	prompt := fmt.Sprintf(`Parse this location search query and extract structured information.
Query: "%s"

Extract:
1. Category (restaurant, grocery, pharmacy, parks, shopping, entertainment, services, fitness, healthcare, or empty if not specified)
2. Radius in miles (default 1 if not specified)
3. Keywords (relevant search terms)

Respond in JSON format like:
{"category": "restaurant", "radius": 2.0, "keywords": ["vegan", "organic"]}`, query)

	reqBody, _ := json.Marshal(OllamaRequest{
		Model:  "llama3.2:latest",
		Prompt: prompt,
		Stream: false,
	})

	resp, err := http.Post(ollamaHost+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var ollamaResp OllamaResponse
		if json.Unmarshal(body, &ollamaResp) == nil {
			// Try to parse the JSON from Ollama's response
			json.Unmarshal([]byte(ollamaResp.Response), &parsed)
		}
	}

	// Fallback parsing if Ollama is not available or fails
	if len(parsed.Keywords) == 0 {
		parsed = parseFallbackQuery(query)
	}

	return parsed
}

// parseFallbackQuery provides keyword-based parsing when Ollama is unavailable
func parseFallbackQuery(query string) ParsedQuery {
	parsed := ParsedQuery{
		Category: "",
		Radius:   1.0,
		Keywords: []string{},
	}

	lowerQuery := strings.ToLower(query)

	// Parse radius
	radiusPatterns := []string{" mile", " mi", " km", " kilometer"}
	for _, pattern := range radiusPatterns {
		if idx := strings.Index(lowerQuery, pattern); idx > 0 {
			// Try to extract number before the pattern
			parts := strings.Fields(lowerQuery[:idx])
			if len(parts) > 0 {
				if radius, err := strconv.ParseFloat(parts[len(parts)-1], 64); err == nil {
					parsed.Radius = radius
				}
			}
		}
	}

	// Parse category
	categories := []string{"restaurant", "grocery", "pharmacy", "park", "shopping", "entertainment", "service", "fitness", "healthcare"}
	for _, cat := range categories {
		if strings.Contains(lowerQuery, cat) {
			parsed.Category = cat
			if cat == "park" {
				parsed.Category = "parks"
			} else if cat == "service" {
				parsed.Category = "services"
			}
			break
		}
	}

	// Extract keywords
	keywords := []string{"vegan", "organic", "healthy", "fast", "cheap", "luxury", "local", "new", "24 hour", "open late"}
	for _, keyword := range keywords {
		if strings.Contains(lowerQuery, keyword) {
			parsed.Keywords = append(parsed.Keywords, keyword)
		}
	}

	return parsed
}
