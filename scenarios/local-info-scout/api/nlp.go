package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ParsedQuery represents a parsed natural language query
type ParsedQuery struct {
	Category string   `json:"category"`
	Radius   float64  `json:"radius"`
	Keywords []string `json:"keywords"`
}

// parseNaturalLanguageQuery parses a natural language query into structured data.
// Uses the resource-ollama gateway CLI for intelligent parsing, with fallback
// to keyword matching when the daemon is unavailable.
func parseNaturalLanguageQuery(query string) ParsedQuery {
	parsed := ParsedQuery{
		Category: "",
		Radius:   1.0, // Default 1 mile
		Keywords: []string{},
	}

	prompt := fmt.Sprintf(`Parse this location search query and extract structured information.
Query: "%s"

Extract:
1. Category (restaurant, grocery, pharmacy, parks, shopping, entertainment, services, fitness, healthcare, or empty if not specified)
2. Radius in miles (default 1 if not specified)
3. Keywords (relevant search terms)

Respond in JSON format like:
{"category": "restaurant", "radius": 2.0, "keywords": ["vegan", "organic"]}`, query)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "resource-ollama", "gateway", "generate",
		"--model", "llama3.2:latest", "--json", "--prompt-stdin")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.Output()
	if err == nil {
		var decoded struct {
			Response string `json:"response"`
		}
		if json.Unmarshal(out, &decoded) == nil {
			// Try to parse the JSON from gateway's response
			json.Unmarshal([]byte(decoded.Response), &parsed)
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
