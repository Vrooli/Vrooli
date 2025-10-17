package main

import (
	"testing"
	"time"
)

// TestPureFunctions tests pure functions that don't depend on database or HTTP
func TestPureFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidateDepth", func(t *testing.T) {
		testCases := []struct {
			depth    string
			expected bool
		}{
			{"quick", true},
			{"standard", true},
			{"deep", true},
			{"", false},
			{"invalid", false},
			{"Quick", false}, // Case sensitive
		}

		for _, tc := range testCases {
			result := validateDepth(tc.depth)
			if result != tc.expected {
				t.Errorf("validateDepth(%q) = %v, want %v", tc.depth, result, tc.expected)
			}
		}
	})

	t.Run("GetDepthConfig", func(t *testing.T) {
		quick := getDepthConfig("quick")
		if quick.MaxSources != 5 {
			t.Errorf("Quick depth MaxSources = %d, want 5", quick.MaxSources)
		}
		if quick.SearchEngines != 3 {
			t.Errorf("Quick depth SearchEngines = %d, want 3", quick.SearchEngines)
		}

		standard := getDepthConfig("standard")
		if standard.MaxSources != 15 {
			t.Errorf("Standard depth MaxSources = %d, want 15", standard.MaxSources)
		}

		deep := getDepthConfig("deep")
		if deep.MaxSources != 30 {
			t.Errorf("Deep depth MaxSources = %d, want 30", deep.MaxSources)
		}

		// Invalid depth should default to standard
		invalid := getDepthConfig("invalid")
		if invalid.MaxSources != 15 {
			t.Errorf("Invalid depth should default to standard, got MaxSources = %d", invalid.MaxSources)
		}
	})

	t.Run("GetReportTemplates", func(t *testing.T) {
		templates := getReportTemplates()

		expectedCount := 5
		if len(templates) != expectedCount {
			t.Errorf("Expected %d templates, got %d", expectedCount, len(templates))
		}

		// Verify each template exists
		requiredTemplates := []string{"general", "academic", "market", "technical", "quick-brief"}
		for _, name := range requiredTemplates {
			if _, exists := templates[name]; !exists {
				t.Errorf("Template %q not found", name)
			}
		}

		// Verify academic template structure
		academic := templates["academic"]
		if academic.Name != "Academic Research" {
			t.Errorf("Academic name = %q, want %q", academic.Name, "Academic Research")
		}
		if academic.DefaultDepth != "deep" {
			t.Errorf("Academic depth = %q, want %q", academic.DefaultDepth, "deep")
		}
		if len(academic.RequiredSections) < 5 {
			t.Errorf("Academic should have >= 5 required sections, got %d", len(academic.RequiredSections))
		}
	})

	t.Run("CalculateDomainAuthority", func(t *testing.T) {
		testCases := []struct {
			url      string
			minScore float64
			maxScore float64
		}{
			{"https://arxiv.org/paper", 0.95, 1.0},
			{"https://stanford.edu/research", 0.95, 1.0},
			{"https://nih.gov/study", 0.95, 1.0},
			{"https://reuters.com/news", 0.85, 0.95},
			{"https://wikipedia.org/wiki", 0.75, 0.85},
			{"https://reddit.com/r/science", 0.60, 0.70},
			{"https://unknown-site.com", 0.40, 0.60},
		}

		for _, tc := range testCases {
			score := calculateDomainAuthority(tc.url)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("calculateDomainAuthority(%q) = %f, want between %f and %f",
					tc.url, score, tc.minScore, tc.maxScore)
			}
		}
	})

	t.Run("CalculateRecencyScore", func(t *testing.T) {
		now := time.Now()

		testCases := []struct {
			name     string
			date     interface{}
			minScore float64
			maxScore float64
		}{
			{"Recent (5 days)", now.AddDate(0, 0, -5).Format("2006-01-02"), 0.90, 1.0},
			{"Medium (60 days)", now.AddDate(0, 0, -60).Format("2006-01-02"), 0.70, 0.90},
			{"Old (180 days)", now.AddDate(0, 0, -180).Format("2006-01-02"), 0.50, 0.70},
			{"Very old (730 days)", now.AddDate(-2, 0, 0).Format("2006-01-02"), 0.30, 0.50},
			{"No date", nil, 0.50, 0.50},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				score := calculateRecencyScore(tc.date)
				if score < tc.minScore || score > tc.maxScore {
					t.Errorf("Score = %f, want between %f and %f", score, tc.minScore, tc.maxScore)
				}
			})
		}
	})

	t.Run("CalculateContentDepth", func(t *testing.T) {
		testCases := []struct {
			name     string
			result   map[string]interface{}
			minScore float64
			maxScore float64
		}{
			{
				name: "High quality content",
				result: map[string]interface{}{
					"title":   "Comprehensive Analysis of AI Safety Methods",
					"content": string(make([]byte, 500)),
					"url":     "https://research.example.com/detailed-paper",
					"author":  "Dr. Smith",
				},
				minScore: 0.75,
				maxScore: 1.0,
			},
			{
				name: "Medium quality",
				result: map[string]interface{}{
					"title":   "AI Overview",
					"content": string(make([]byte, 150)),
					"url":     "https://blog.com/ai",
				},
				minScore: 0.50,
				maxScore: 0.75,
			},
			{
				name: "Low quality",
				result: map[string]interface{}{
					"title":   "AI",
					"content": "Short",
					"url":     "example.com",
				},
				minScore: 0.40,
				maxScore: 0.60,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				score := calculateContentDepth(tc.result)
				if score < tc.minScore || score > tc.maxScore {
					t.Errorf("Score = %f, want between %f and %f", score, tc.minScore, tc.maxScore)
				}
			})
		}
	})

	t.Run("CalculateSourceQuality", func(t *testing.T) {
		result := map[string]interface{}{
			"url":           "https://nature.com/article",
			"title":         "Research Paper on AI",
			"content":       string(make([]byte, 1000)),
			"publishedDate": time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
			"author":        "Dr. Smith",
		}

		metrics := calculateSourceQuality(result)

		// Verify all metrics are in valid range
		if metrics.DomainAuthority < 0 || metrics.DomainAuthority > 1 {
			t.Errorf("DomainAuthority %f out of range [0, 1]", metrics.DomainAuthority)
		}
		if metrics.RecencyScore < 0 || metrics.RecencyScore > 1 {
			t.Errorf("RecencyScore %f out of range [0, 1]", metrics.RecencyScore)
		}
		if metrics.ContentDepth < 0 || metrics.ContentDepth > 1 {
			t.Errorf("ContentDepth %f out of range [0, 1]", metrics.ContentDepth)
		}
		if metrics.OverallQuality < 0 || metrics.OverallQuality > 1 {
			t.Errorf("OverallQuality %f out of range [0, 1]", metrics.OverallQuality)
		}

		// Verify composite formula
		expected := metrics.DomainAuthority*0.5 + metrics.ContentDepth*0.3 + metrics.RecencyScore*0.2
		if metrics.OverallQuality != expected {
			t.Errorf("OverallQuality formula incorrect: got %f, want %f", metrics.OverallQuality, expected)
		}
	})

	t.Run("EnhanceResultsWithQuality", func(t *testing.T) {
		results := []interface{}{
			map[string]interface{}{
				"url":     "https://arxiv.org/paper",
				"title":   "Research Paper",
				"content": string(make([]byte, 500)),
			},
			map[string]interface{}{
				"url":     "https://blog.com/post",
				"title":   "Blog Post",
				"content": "Short content",
			},
		}

		enhanced := enhanceResultsWithQuality(results)

		if len(enhanced) != len(results) {
			t.Errorf("Expected %d results, got %d", len(results), len(enhanced))
		}

		for i, result := range enhanced {
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Errorf("Result %d is not a map", i)
				continue
			}

			metrics, exists := resultMap["quality_metrics"]
			if !exists {
				t.Errorf("Result %d missing quality_metrics", i)
				continue
			}

			metricsMap, ok := metrics.(map[string]interface{})
			if !ok {
				t.Errorf("Result %d quality_metrics is not a map", i)
				continue
			}

			overallQuality, exists := metricsMap["overall_quality"]
			if !exists {
				t.Errorf("Result %d missing overall_quality", i)
				continue
			}

			qualityValue, ok := overallQuality.(float64)
			if !ok {
				t.Errorf("Result %d overall_quality is not float64", i)
				continue
			}

			if qualityValue < 0 || qualityValue > 1 {
				t.Errorf("Result %d has invalid overall_quality: %f", i, qualityValue)
			}
		}
	})

	t.Run("SortResultsByQuality", func(t *testing.T) {
		results := []interface{}{
			map[string]interface{}{
				"title": "Low quality",
				"quality_metrics": map[string]interface{}{
					"overall_quality": 0.3,
				},
			},
			map[string]interface{}{
				"title": "High quality",
				"quality_metrics": map[string]interface{}{
					"overall_quality": 0.9,
				},
			},
			map[string]interface{}{
				"title": "Medium quality",
				"quality_metrics": map[string]interface{}{
					"overall_quality": 0.6,
				},
			},
		}

		sortResultsByQuality(results)

		// Verify sorted in descending order
		for i := 0; i < len(results)-1; i++ {
			currentMetrics := results[i].(map[string]interface{})["quality_metrics"].(map[string]interface{})
			nextMetrics := results[i+1].(map[string]interface{})["quality_metrics"].(map[string]interface{})

			currentQuality := currentMetrics["overall_quality"].(float64)
			nextQuality := nextMetrics["overall_quality"].(float64)

			if currentQuality < nextQuality {
				t.Errorf("Results not sorted: result[%d] (%f) < result[%d] (%f)",
					i, currentQuality, i+1, nextQuality)
			}
		}

		// Verify first result has highest quality
		firstMetrics := results[0].(map[string]interface{})["quality_metrics"].(map[string]interface{})
		firstQuality := firstMetrics["overall_quality"].(float64)
		if firstQuality != 0.9 {
			t.Errorf("First result should have quality 0.9, got %f", firstQuality)
		}
	})
}
