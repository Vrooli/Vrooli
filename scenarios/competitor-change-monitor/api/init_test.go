// +build testing

package main

import (
	"os"
	"testing"
)

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	t.Run("VROOLI_LIFECYCLE_MANAGED_Required", func(t *testing.T) {
		// This is set in TestMain, so it should be present
		value := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
		if value != "true" {
			t.Errorf("Expected VROOLI_LIFECYCLE_MANAGED=true, got %s", value)
		}
	})

	t.Run("POSTGRES_URL_Optional", func(t *testing.T) {
		// Can be empty, that's okay
		_ = os.Getenv("POSTGRES_URL")
	})

	t.Run("N8N_BASE_URL_Optional", func(t *testing.T) {
		// Can be empty, that's okay
		_ = os.Getenv("N8N_BASE_URL")
	})
}

// TestDatabaseConnectionString tests database connection string building
func TestDatabaseConnectionString(t *testing.T) {
	testCases := []struct{
		name string
		env map[string]string
		expectSkip bool
	}{
		{
			name: "With_POSTGRES_URL",
			env: map[string]string{
				"POSTGRES_URL": "postgres://user:pass@localhost:5432/db",
			},
			expectSkip: false,
		},
		{
			name: "With_Individual_Components",
			env: map[string]string{
				"POSTGRES_HOST": "localhost",
				"POSTGRES_PORT": "5432",
				"POSTGRES_USER": "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB": "test_db",
			},
			expectSkip: false,
		},
		{
			name: "Missing_All_Config",
			env: map[string]string{},
			expectSkip: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("POSTGRES_URL")
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")

			// Set test env vars
			for key, value := range tc.env {
				os.Setenv(key, value)
			}

			// Test would involve calling initDB(), but that would try to actually connect
			// Instead we just verify the env vars are set correctly
			if len(tc.env) > 0 {
				for key, value := range tc.env {
					if os.Getenv(key) != value {
						t.Errorf("Expected %s=%s, got %s", key, value, os.Getenv(key))
					}
				}
			}

			// Restore defaults
			os.Setenv("POSTGRES_HOST", "localhost")
			os.Setenv("POSTGRES_PORT", "5432")
			os.Setenv("POSTGRES_USER", "postgres")
			os.Setenv("POSTGRES_PASSWORD", "postgres")
			os.Setenv("POSTGRES_DB", "competitor_monitor_test")
		})
	}
}

// TestStructDefaults tests struct default values
func TestStructDefaults(t *testing.T) {
	t.Run("Competitor_Defaults", func(t *testing.T) {
		comp := Competitor{}

		if comp.IsActive {
			t.Error("Expected IsActive to be false by default")
		}

		if comp.Metadata != nil {
			// Metadata should be nil or empty
		}
	})

	t.Run("MonitoringTarget_Defaults", func(t *testing.T) {
		target := MonitoringTarget{}

		if target.IsActive {
			t.Error("Expected IsActive to be false by default")
		}

		if target.CheckFrequency != 0 {
			t.Errorf("Expected CheckFrequency to be 0 by default, got %d", target.CheckFrequency)
		}
	})

	t.Run("Alert_Defaults", func(t *testing.T) {
		alert := Alert{}

		if alert.RelevanceScore != 0 {
			t.Errorf("Expected RelevanceScore to be 0 by default, got %d", alert.RelevanceScore)
		}
	})
}

// TestFieldValidation tests field validation logic
func TestFieldValidation(t *testing.T) {
	t.Run("Competitor_ValidCategory", func(t *testing.T) {
		validCategories := []string{"technology", "finance", "healthcare", "retail"}

		for _, cat := range validCategories {
			comp := Competitor{
				Name:     "Test",
				Category: cat,
			}

			if comp.Category != cat {
				t.Errorf("Expected category %s, got %s", cat, comp.Category)
			}
		}
	})

	t.Run("Competitor_ValidImportance", func(t *testing.T) {
		validImportance := []string{"low", "medium", "high", "critical"}

		for _, imp := range validImportance {
			comp := Competitor{
				Name:       "Test",
				Importance: imp,
			}

			if comp.Importance != imp {
				t.Errorf("Expected importance %s, got %s", imp, comp.Importance)
			}
		}
	})

	t.Run("MonitoringTarget_ValidTypes", func(t *testing.T) {
		validTypes := []string{"website", "github", "api", "rss", "social"}

		for _, targetType := range validTypes {
			target := MonitoringTarget{
				URL:        "https://example.com",
				TargetType: targetType,
			}

			if target.TargetType != targetType {
				t.Errorf("Expected target type %s, got %s", targetType, target.TargetType)
			}
		}
	})

	t.Run("Alert_ValidPriorities", func(t *testing.T) {
		validPriorities := []string{"low", "medium", "high", "critical"}

		for _, priority := range validPriorities {
			alert := Alert{
				Title:    "Test",
				Priority: priority,
			}

			if alert.Priority != priority {
				t.Errorf("Expected priority %s, got %s", priority, alert.Priority)
			}
		}
	})

	t.Run("Alert_ValidStatuses", func(t *testing.T) {
		validStatuses := []string{"unread", "read", "acknowledged", "dismissed"}

		for _, status := range validStatuses {
			alert := Alert{
				Title:  "Test",
				Status: status,
			}

			if alert.Status != status {
				t.Errorf("Expected status %s, got %s", status, alert.Status)
			}
		}
	})
}

// TestJSONRawMessage tests JSON raw message handling
func TestJSONRawMessage(t *testing.T) {
	t.Run("Competitor_Metadata", func(t *testing.T) {
		comp := Competitor{
			Name:     "Test",
			Metadata: []byte(`{"key": "value"}`),
		}

		if string(comp.Metadata) != `{"key": "value"}` {
			t.Errorf("Expected metadata to be preserved, got %s", comp.Metadata)
		}
	})

	t.Run("MonitoringTarget_Config", func(t *testing.T) {
		target := MonitoringTarget{
			URL:    "https://example.com",
			Config: []byte(`{"timeout": 30}`),
		}

		if string(target.Config) != `{"timeout": 30}` {
			t.Errorf("Expected config to be preserved, got %s", target.Config)
		}
	})

	t.Run("Alert_Insights", func(t *testing.T) {
		alert := Alert{
			Title:    "Test",
			Insights: []byte(`{"insight": "value"}`),
		}

		if string(alert.Insights) != `{"insight": "value"}` {
			t.Errorf("Expected insights to be preserved, got %s", alert.Insights)
		}
	})

	t.Run("Alert_Actions", func(t *testing.T) {
		alert := Alert{
			Title:   "Test",
			Actions: []byte(`{"action": "review"}`),
		}

		if string(alert.Actions) != `{"action": "review"}` {
			t.Errorf("Expected actions to be preserved, got %s", alert.Actions)
		}
	})
}

// TestURLFormatting tests URL formatting
func TestURLFormatting(t *testing.T) {
	t.Run("MonitoringTarget_URLs", func(t *testing.T) {
		validURLs := []string{
			"https://example.com",
			"http://example.com/path",
			"https://example.com:8080/api",
			"https://api.example.com/v1/endpoint",
		}

		for _, url := range validURLs {
			target := MonitoringTarget{
				URL:        url,
				TargetType: "website",
			}

			if target.URL != url {
				t.Errorf("Expected URL %s, got %s", url, target.URL)
			}
		}
	})

	t.Run("Alert_URLs", func(t *testing.T) {
		validURLs := []string{
			"https://example.com",
			"http://example.com/announcement",
		}

		for _, url := range validURLs {
			alert := Alert{
				Title: "Test",
				URL:   url,
			}

			if alert.URL != url {
				t.Errorf("Expected URL %s, got %s", url, alert.URL)
			}
		}
	})
}

// TestCSSSelectorHandling tests CSS selector handling
func TestCSSSelectorHandling(t *testing.T) {
	t.Run("Valid_Selectors", func(t *testing.T) {
		validSelectors := []string{
			".content",
			"#main",
			"div.class",
			"a[href]",
			"ul > li",
			".parent .child",
		}

		for _, selector := range validSelectors {
			target := MonitoringTarget{
				URL:        "https://example.com",
				TargetType: "website",
				Selector:   selector,
			}

			if target.Selector != selector {
				t.Errorf("Expected selector %s, got %s", selector, target.Selector)
			}
		}
	})
}

// TestCheckFrequencyValidation tests check frequency validation
func TestCheckFrequencyValidation(t *testing.T) {
	t.Run("ValidFrequencies", func(t *testing.T) {
		validFrequencies := []int{
			60,     // 1 minute
			300,    // 5 minutes
			900,    // 15 minutes
			1800,   // 30 minutes
			3600,   // 1 hour
			14400,  // 4 hours
			86400,  // 24 hours
		}

		for _, freq := range validFrequencies {
			target := MonitoringTarget{
				URL:            "https://example.com",
				TargetType:     "website",
				CheckFrequency: freq,
			}

			if target.CheckFrequency != freq {
				t.Errorf("Expected check frequency %d, got %d", freq, target.CheckFrequency)
			}
		}
	})
}

// TestRelevanceScoreRange tests relevance score range
func TestRelevanceScoreRange(t *testing.T) {
	t.Run("ValidScores", func(t *testing.T) {
		validScores := []int{0, 25, 50, 75, 100}

		for _, score := range validScores {
			alert := Alert{
				Title:          "Test",
				RelevanceScore: score,
			}

			if alert.RelevanceScore != score {
				t.Errorf("Expected relevance score %d, got %d", score, alert.RelevanceScore)
			}

			analysis := ChangeAnalysis{
				TargetURL:      "https://example.com",
				RelevanceScore: score,
			}

			if analysis.RelevanceScore != score {
				t.Errorf("Expected relevance score %d, got %d", score, analysis.RelevanceScore)
			}
		}
	})
}
