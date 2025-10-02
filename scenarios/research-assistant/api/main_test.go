package main

import (
	"testing"
	"time"
)

// Test depth validation
func TestValidateDepth(t *testing.T) {
	tests := []struct {
		name     string
		depth    string
		expected bool
	}{
		{"Valid quick", "quick", true},
		{"Valid standard", "standard", true},
		{"Valid deep", "deep", true},
		{"Invalid empty", "", false},
		{"Invalid wrong value", "super-deep", false},
		{"Invalid case", "Quick", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateDepth(tt.depth)
			if result != tt.expected {
				t.Errorf("validateDepth(%q) = %v, want %v", tt.depth, result, tt.expected)
			}
		})
	}
}

// Test depth configuration retrieval
func TestGetDepthConfig(t *testing.T) {
	tests := []struct {
		name           string
		depth          string
		expectedSources int
		expectedEngines int
		expectedRounds  int
		expectedTimeout int
	}{
		{
			name:            "Quick depth",
			depth:           "quick",
			expectedSources: 5,
			expectedEngines: 3,
			expectedRounds:  1,
			expectedTimeout: 2,
		},
		{
			name:            "Standard depth",
			depth:           "standard",
			expectedSources: 15,
			expectedEngines: 7,
			expectedRounds:  2,
			expectedTimeout: 5,
		},
		{
			name:            "Deep depth",
			depth:           "deep",
			expectedSources: 30,
			expectedEngines: 15,
			expectedRounds:  3,
			expectedTimeout: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getDepthConfig(tt.depth)
			if config.MaxSources != tt.expectedSources {
				t.Errorf("MaxSources = %d, want %d", config.MaxSources, tt.expectedSources)
			}
			if config.SearchEngines != tt.expectedEngines {
				t.Errorf("SearchEngines = %d, want %d", config.SearchEngines, tt.expectedEngines)
			}
			if config.AnalysisRounds != tt.expectedRounds {
				t.Errorf("AnalysisRounds = %d, want %d", config.AnalysisRounds, tt.expectedRounds)
			}
			if config.TimeoutMinutes != tt.expectedTimeout {
				t.Errorf("TimeoutMinutes = %d, want %d", config.TimeoutMinutes, tt.expectedTimeout)
			}
		})
	}
}

// Test report templates
func TestGetReportTemplates(t *testing.T) {
	templates := getReportTemplates()

	expectedTemplates := []string{"general", "academic", "market", "technical", "quick-brief"}

	if len(templates) != len(expectedTemplates) {
		t.Errorf("Expected %d templates, got %d", len(expectedTemplates), len(templates))
	}

	for _, name := range expectedTemplates {
		if _, exists := templates[name]; !exists {
			t.Errorf("Template %q not found", name)
		}
	}

	// Validate academic template structure
	academic := templates["academic"]
	if academic.Name != "Academic Research" {
		t.Errorf("Academic template name = %q, want %q", academic.Name, "Academic Research")
	}
	if academic.DefaultDepth != "deep" {
		t.Errorf("Academic template depth = %q, want %q", academic.DefaultDepth, "deep")
	}
	if len(academic.RequiredSections) < 5 {
		t.Errorf("Academic template should have at least 5 required sections, got %d", len(academic.RequiredSections))
	}
}

// Test domain authority calculation
func TestCalculateDomainAuthority(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		minScore float64
		maxScore float64
	}{
		{
			name:     "Academic domain (.edu)",
			url:      "https://stanford.edu/research/ai",
			minScore: 0.95,
			maxScore: 1.0,
		},
		{
			name:     "Government domain (.gov)",
			url:      "https://nih.gov/health/covid",
			minScore: 0.95,
			maxScore: 1.0,
		},
		{
			name:     "News source (Reuters)",
			url:      "https://reuters.com/article/tech",
			minScore: 0.85,
			maxScore: 0.95,
		},
		{
			name:     "Wikipedia",
			url:      "https://en.wikipedia.org/wiki/AI",
			minScore: 0.70,
			maxScore: 0.80,
		},
		{
			name:     "Social media (Reddit)",
			url:      "https://reddit.com/r/science",
			minScore: 0.60,
			maxScore: 0.70,
		},
		{
			name:     "Unknown domain",
			url:      "https://random-blog.com/post",
			minScore: 0.40,
			maxScore: 0.60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateDomainAuthority(tt.url)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateDomainAuthority(%q) = %f, want between %f and %f",
					tt.url, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Test recency score calculation
func TestCalculateRecencyScore(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		date     interface{}
		minScore float64
		maxScore float64
	}{
		{
			name:     "Recent (1 day ago)",
			date:     now.AddDate(0, 0, -1).Format("2006-01-02"),
			minScore: 0.95,
			maxScore: 1.0,
		},
		{
			name:     "Recent (15 days ago)",
			date:     now.AddDate(0, 0, -15).Format("2006-01-02"),
			minScore: 0.90,
			maxScore: 1.0,
		},
		{
			name:     "Medium age (60 days ago)",
			date:     now.AddDate(0, 0, -60).Format("2006-01-02"),
			minScore: 0.70,
			maxScore: 0.90,
		},
		{
			name:     "Old (180 days ago)",
			date:     now.AddDate(0, 0, -180).Format("2006-01-02"),
			minScore: 0.50,
			maxScore: 0.70,
		},
		{
			name:     "Very old (2 years ago)",
			date:     now.AddDate(-2, 0, 0).Format("2006-01-02"),
			minScore: 0.30,
			maxScore: 0.50,
		},
		{
			name:     "No date provided",
			date:     nil,
			minScore: 0.50,
			maxScore: 0.50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateRecencyScore(tt.date)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateRecencyScore(%v) = %f, want between %f and %f",
					tt.date, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Test content depth calculation
func TestCalculateContentDepth(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]interface{}
		minScore float64
		maxScore float64
	}{
		{
			name: "High quality content",
			result: map[string]interface{}{
				"title":   "Comprehensive Analysis of AI Safety: Methods, Challenges, and Future Directions",
				"content": string(make([]byte, 1500)), // 1500 chars
				"url":     "https://research.example.com/papers/ai-safety-2024",
				"author":  "Dr. Jane Smith",
			},
			minScore: 0.80,
			maxScore: 1.0,
		},
		{
			name: "Medium quality content",
			result: map[string]interface{}{
				"title":   "AI Safety Overview",
				"content": string(make([]byte, 300)), // 300 chars
				"url":     "https://blog.example.com/ai-safety",
			},
			minScore: 0.50,
			maxScore: 0.80,
		},
		{
			name: "Low quality content",
			result: map[string]interface{}{
				"title":   "AI",
				"content": "Short content",
				"url":     "https://example.com/p=123",
			},
			minScore: 0.20,
			maxScore: 0.50,
		},
		{
			name: "Minimal content",
			result: map[string]interface{}{
				"title": "",
				"url":   "example.com",
			},
			minScore: 0.40, // Adjusted - empty fields get default scores
			maxScore: 0.60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateContentDepth(tt.result)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateContentDepth() = %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Test source quality calculation (integration test)
func TestCalculateSourceQuality(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		result         map[string]interface{}
		minOverall     float64
		maxOverall     float64
	}{
		{
			name: "High quality academic source",
			result: map[string]interface{}{
				"url":            "https://nature.com/articles/ai-research-2024",
				"title":          "Breakthrough in AI Safety Research: Novel Alignment Techniques",
				"content":        string(make([]byte, 2000)),
				"publishedDate":  now.AddDate(0, 0, -5).Format("2006-01-02"),
				"author":         "Dr. Smith",
			},
			minOverall: 0.85,
			maxOverall: 1.0,
		},
		{
			name: "Medium quality news source",
			result: map[string]interface{}{
				"url":            "https://reuters.com/tech/ai-news",
				"title":          "AI Companies Release New Safety Guidelines",
				"content":        string(make([]byte, 800)),
				"publishedDate":  now.AddDate(0, 0, -30).Format("2006-01-02"),
			},
			minOverall: 0.80, // Reuters has high domain authority (0.90)
			maxOverall: 0.95,
		},
		{
			name: "Low quality source",
			result: map[string]interface{}{
				"url":     "https://random-blog.com/ai-post",
				"title":   "AI stuff",
				"content": "Not much content here",
			},
			minOverall: 0.30,
			maxOverall: 0.60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := calculateSourceQuality(tt.result)

			// Verify all metrics are in valid range [0, 1]
			if metrics.DomainAuthority < 0 || metrics.DomainAuthority > 1 {
				t.Errorf("DomainAuthority %f out of range [0, 1]", metrics.DomainAuthority)
			}
			if metrics.RecencyScore < 0 || metrics.RecencyScore > 1 {
				t.Errorf("RecencyScore %f out of range [0, 1]", metrics.RecencyScore)
			}
			if metrics.ContentDepth < 0 || metrics.ContentDepth > 1 {
				t.Errorf("ContentDepth %f out of range [0, 1]", metrics.ContentDepth)
			}

			// Verify overall quality is in expected range
			if metrics.OverallQuality < tt.minOverall || metrics.OverallQuality > tt.maxOverall {
				t.Errorf("OverallQuality = %f, want between %f and %f",
					metrics.OverallQuality, tt.minOverall, tt.maxOverall)
			}

			// Verify composite formula: domain(50%) + content(30%) + recency(20%)
			expected := metrics.DomainAuthority*0.5 + metrics.ContentDepth*0.3 + metrics.RecencyScore*0.2
			if metrics.OverallQuality != expected {
				t.Errorf("OverallQuality calculation incorrect: got %f, want %f",
					metrics.OverallQuality, expected)
			}
		})
	}
}

// Test result enhancement with quality metrics
func TestEnhanceResultsWithQuality(t *testing.T) {
	results := []interface{}{
		map[string]interface{}{
			"url":     "https://nature.com/article",
			"title":   "Research Paper",
			"content": string(make([]byte, 1000)),
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

		// Verify quality_metrics was added
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

		// Verify overall_quality exists and is valid
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
}

// Test result sorting by quality
func TestSortResultsByQuality(t *testing.T) {
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

	// Verify results are sorted in descending order
	for i := 0; i < len(results)-1; i++ {
		currentMetrics := results[i].(map[string]interface{})["quality_metrics"].(map[string]interface{})
		nextMetrics := results[i+1].(map[string]interface{})["quality_metrics"].(map[string]interface{})

		currentQuality := currentMetrics["overall_quality"].(float64)
		nextQuality := nextMetrics["overall_quality"].(float64)

		if currentQuality < nextQuality {
			t.Errorf("Results not sorted correctly: result[%d] (%f) < result[%d] (%f)",
				i, currentQuality, i+1, nextQuality)
		}
	}

	// Verify first result has highest quality
	firstMetrics := results[0].(map[string]interface{})["quality_metrics"].(map[string]interface{})
	firstQuality := firstMetrics["overall_quality"].(float64)
	if firstQuality != 0.9 {
		t.Errorf("First result should have quality 0.9, got %f", firstQuality)
	}
}
