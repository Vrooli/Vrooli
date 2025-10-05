package main

import (
	"testing"
)

// TestScanScenariosDetailed provides comprehensive coverage of scanScenarios
func TestScanScenariosDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Note: scanScenarios uses a hardcoded path "../../../scenarios" which makes it difficult to test
	// in isolation. These tests verify the behavior we can control.

	t.Run("VerifyKnownScenarioLogic", func(t *testing.T) {
		// Test the logic that would be applied to known scenarios
		config := ServiceConfig{
			Name:        "retro-game-launcher",
			Description: "Generic description",
			Category:    []string{"games"},
			Deployment: struct {
				Port int `json:"port"`
			}{Port: 3301},
		}

		// Verify it's classified as kid-friendly
		if !isKidFriendly(config) {
			t.Error("Expected retro-game-launcher to be kid-friendly")
		}
	})

	t.Run("VerifyUnknownScenarioLogic", func(t *testing.T) {
		// Test logic for unknown kid-friendly scenarios
		config := ServiceConfig{
			Name:        "unknown-game",
			Description: "A new game",
			Category:    []string{"kid-friendly"},
			Deployment: struct {
				Port int `json:"port"`
			}{Port: 3600},
		}

		if !isKidFriendly(config) {
			t.Error("Expected unknown-game to be kid-friendly")
		}
	})

	t.Run("VerifyBlacklistLogic", func(t *testing.T) {
		config := ServiceConfig{
			Name:        "admin-panel",
			Description: "Admin tools",
			Category:    []string{"admin"},
		}

		if isKidFriendly(config) {
			t.Error("Expected admin-panel to not be kid-friendly")
		}
	})

	t.Run("ScanScenariosDoesNotPanic", func(t *testing.T) {
		// Verify scanScenarios can run without panicking
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("scanScenarios panicked: %v", r)
			}
		}()

		originalScenarios := kidScenarios
		defer func() { kidScenarios = originalScenarios }()

		scanScenarios()
	})
}

// TestIsKidFriendlyComprehensive tests all branches of isKidFriendly
func TestIsKidFriendlyComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AllKnownKidFriendlyScenarios", func(t *testing.T) {
		knownScenarios := []string{"retro-game-launcher", "picker-wheel", "word-games", "study-buddy"}

		for _, name := range knownScenarios {
			config := ServiceConfig{Name: name}
			if !isKidFriendly(config) {
				t.Errorf("Expected %s to be kid-friendly", name)
			}
		}
	})

	t.Run("AllBlacklistedCategories", func(t *testing.T) {
		blacklist := []string{"system", "development", "admin", "financial", "debug", "infrastructure"}

		for _, category := range blacklist {
			config := ServiceConfig{Category: []string{category}}
			if isKidFriendly(config) {
				t.Errorf("Expected category '%s' to be blacklisted", category)
			}
		}
	})

	t.Run("KidFriendlyMetadataTag", func(t *testing.T) {
		config := ServiceConfig{
			Metadata: struct {
				Tags           []string `json:"tags"`
				TargetAudience struct {
					AgeRange string `json:"ageRange"`
				} `json:"targetAudience"`
			}{
				Tags: []string{"kids", "educational"},
			},
		}

		if !isKidFriendly(config) {
			t.Error("Expected scenario with 'kids' tag to be kid-friendly")
		}
	})

	t.Run("ChildrenMetadataTag", func(t *testing.T) {
		config := ServiceConfig{
			Metadata: struct {
				Tags           []string `json:"tags"`
				TargetAudience struct {
					AgeRange string `json:"ageRange"`
				} `json:"targetAudience"`
			}{
				Tags: []string{"children"},
			},
		}

		if !isKidFriendly(config) {
			t.Error("Expected scenario with 'children' tag to be kid-friendly")
		}
	})
}

// TestFilterScenariosComprehensive tests all edge cases of filterScenarios
func TestFilterScenariosComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyScenarioList", func(t *testing.T) {
		result := filterScenarios([]Scenario{}, "5-12", "games")
		if len(result) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(result))
		}
	})

	t.Run("NilScenarioList", func(t *testing.T) {
		result := filterScenarios(nil, "5-12", "games")
		if len(result) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(result))
		}
	})

	t.Run("FilterExcludesNonMatchingAgeRange", func(t *testing.T) {
		scenarios := []Scenario{
			{ID: "s1", AgeRange: "13-17", Category: "games"},
			{ID: "s2", AgeRange: "9-12", Category: "games"},
		}

		result := filterScenarios(scenarios, "5-8", "")

		// Should only include scenarios with matching or compatible age ranges
		hasTeenScenario := false
		for _, s := range result {
			if s.ID == "s1" && s.AgeRange == "13-17" {
				hasTeenScenario = true
			}
		}

		if hasTeenScenario {
			t.Error("Expected to exclude 13-17 scenario when filtering for 5-8")
		}
	})

	t.Run("FilterIncludesUniversalAgeRange", func(t *testing.T) {
		scenarios := []Scenario{
			{ID: "s1", AgeRange: "5-12", Category: "games"},
		}

		// 5-12 should be included for any age range filter
		result := filterScenarios(scenarios, "7-9", "")

		if len(result) == 0 {
			t.Error("Expected universal age range 5-12 to be included")
		}
	})
}
