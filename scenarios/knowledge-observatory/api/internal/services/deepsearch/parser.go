package deepsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ResultParser converts agent output into structured deep search results.
type ResultParser interface {
	Parse(ctx context.Context, raw string) ([]DeepSearchResult, error)
}

// JSONParser attempts to parse JSON output with optional fallback parsing.
type JSONParser struct {
	Fallback ResultParser
}

func (p *JSONParser) Parse(ctx context.Context, raw string) ([]DeepSearchResult, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, fmt.Errorf("empty output")
	}
	if parsed, ok := parseJSONResults(clean); ok {
		return parsed, nil
	}
	clean = stripCodeFence(clean)
	if parsed, ok := parseJSONResults(clean); ok {
		return parsed, nil
	}
	if extracted := extractJSONArray(clean); extracted != "" {
		if parsed, ok := parseJSONResults(extracted); ok {
			return parsed, nil
		}
	}
	if p.Fallback != nil {
		return p.Fallback.Parse(ctx, raw)
	}
	return nil, fmt.Errorf("unable to parse deep search output as JSON")
}

func parseJSONResults(raw string) ([]DeepSearchResult, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	var direct []DeepSearchResult
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &direct); err == nil {
			return direct, true
		}
	}
	var wrapped struct {
		Results []DeepSearchResult `json:"results"`
	}
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal([]byte(trimmed), &wrapped); err == nil {
			if len(wrapped.Results) > 0 || strings.Contains(trimmed, "\"results\"") {
				return wrapped.Results, true
			}
		}
	}
	return nil, false
}

func stripCodeFence(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "```") {
		return value
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= 1 {
		return value
	}
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func extractJSONArray(value string) string {
	start := strings.Index(value, "[")
	end := strings.LastIndex(value, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(value[start : end+1])
}
