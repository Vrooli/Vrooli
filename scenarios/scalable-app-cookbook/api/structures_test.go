// +build testing

package main

import (
	"encoding/json"
	"testing"
)

// TestPatternStructure tests the Pattern struct marshaling
func TestPatternStructure(t *testing.T) {
	t.Run("Success_JSONMarshaling", func(t *testing.T) {
		pattern := Pattern{
			ID:            "test-1",
			Title:         "Test Pattern",
			Chapter:       "Chapter 1",
			Section:       "Section 1",
			MaturityLevel: "proven",
			Tags:          []string{"tag1", "tag2"},
			WhatAndWhy:    "Description",
			WhenToUse:     "Usage",
			Tradeoffs:     "Tradeoffs",
			RefPatterns:   []string{"ref1"},
			FailureModes:  "Failures",
			CostLevers:    "Costs",
			RecipeCount:   5,
			ImplCount:     10,
			Languages:     []string{"go", "python"},
		}

		data, err := json.Marshal(pattern)
		if err != nil {
			t.Fatalf("Failed to marshal pattern: %v", err)
		}

		var unmarshaled Pattern
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal pattern: %v", err)
		}

		if unmarshaled.ID != pattern.ID {
			t.Errorf("Expected ID %s, got %s", pattern.ID, unmarshaled.ID)
		}

		if unmarshaled.Title != pattern.Title {
			t.Errorf("Expected Title %s, got %s", pattern.Title, unmarshaled.Title)
		}

		if len(unmarshaled.Tags) != len(pattern.Tags) {
			t.Errorf("Expected %d tags, got %d", len(pattern.Tags), len(unmarshaled.Tags))
		}
	})
}

// TestRecipeStructure tests the Recipe struct marshaling
func TestRecipeStructure(t *testing.T) {
	t.Run("Success_JSONMarshaling", func(t *testing.T) {
		recipe := Recipe{
			ID:               "recipe-1",
			PatternID:        "pattern-1",
			Title:            "Test Recipe",
			Type:             "greenfield",
			Prerequisites:    []string{"prereq1"},
			Steps:            []map[string]interface{}{{"step": 1}},
			ConfigSnippets:   map[string]interface{}{"config": "value"},
			ValidationChecks: []string{"check1"},
			Artifacts:        []string{"artifact1"},
			Metrics:          []string{"metric1"},
			Rollbacks:        []string{"rollback1"},
			Prompts:          []string{"prompt1"},
			TimeoutSec:       300,
		}

		data, err := json.Marshal(recipe)
		if err != nil {
			t.Fatalf("Failed to marshal recipe: %v", err)
		}

		var unmarshaled Recipe
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal recipe: %v", err)
		}

		if unmarshaled.ID != recipe.ID {
			t.Errorf("Expected ID %s, got %s", recipe.ID, unmarshaled.ID)
		}

		if unmarshaled.Type != recipe.Type {
			t.Errorf("Expected Type %s, got %s", recipe.Type, unmarshaled.Type)
		}

		if unmarshaled.TimeoutSec != recipe.TimeoutSec {
			t.Errorf("Expected TimeoutSec %d, got %d", recipe.TimeoutSec, unmarshaled.TimeoutSec)
		}
	})
}

// TestImplementationStructure tests the Implementation struct marshaling
func TestImplementationStructure(t *testing.T) {
	t.Run("Success_JSONMarshaling", func(t *testing.T) {
		impl := Implementation{
			ID:           "impl-1",
			RecipeID:     "recipe-1",
			Language:     "go",
			Code:         "package main",
			FilePath:     "/main.go",
			Description:  "Test implementation",
			Dependencies: []string{"dep1", "dep2"},
			TestCode:     "package main_test",
		}

		data, err := json.Marshal(impl)
		if err != nil {
			t.Fatalf("Failed to marshal implementation: %v", err)
		}

		var unmarshaled Implementation
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal implementation: %v", err)
		}

		if unmarshaled.Language != impl.Language {
			t.Errorf("Expected Language %s, got %s", impl.Language, unmarshaled.Language)
		}

		if len(unmarshaled.Dependencies) != len(impl.Dependencies) {
			t.Errorf("Expected %d dependencies, got %d", len(impl.Dependencies), len(unmarshaled.Dependencies))
		}
	})
}

// TestGenerationRequestStructure tests the GenerationRequest struct
func TestGenerationRequestStructure(t *testing.T) {
	t.Run("Success_JSONUnmarshaling", func(t *testing.T) {
		jsonData := `{
			"recipe_id": "recipe-1",
			"language": "go",
			"parameters": {"param1": "value1"},
			"target_platform": "linux"
		}`

		var req GenerationRequest
		if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
			t.Fatalf("Failed to unmarshal generation request: %v", err)
		}

		if req.RecipeID != "recipe-1" {
			t.Errorf("Expected RecipeID 'recipe-1', got %s", req.RecipeID)
		}

		if req.Language != "go" {
			t.Errorf("Expected Language 'go', got %s", req.Language)
		}

		if req.TargetPlatform != "linux" {
			t.Errorf("Expected TargetPlatform 'linux', got %s", req.TargetPlatform)
		}
	})

	t.Run("Success_WithEmptyParameters", func(t *testing.T) {
		jsonData := `{
			"recipe_id": "recipe-1",
			"language": "python"
		}`

		var req GenerationRequest
		if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
			t.Fatalf("Failed to unmarshal generation request: %v", err)
		}

		if req.RecipeID != "recipe-1" {
			t.Errorf("Expected RecipeID 'recipe-1', got %s", req.RecipeID)
		}
	})
}

// TestGenerationResultStructure tests the GenerationResult struct
func TestGenerationResultStructure(t *testing.T) {
	t.Run("Success_JSONMarshaling", func(t *testing.T) {
		result := GenerationResult{
			GeneratedCode: "package main\n\nfunc main() {}",
			FileStructure: map[string]interface{}{
				"main_file": "main.go",
				"test_file": "main_test.go",
			},
			Dependencies: []string{"github.com/pkg/errors"},
			SetupInstructions: []string{
				"go mod init",
				"go mod tidy",
			},
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal generation result: %v", err)
		}

		var unmarshaled GenerationResult
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal generation result: %v", err)
		}

		if unmarshaled.GeneratedCode != result.GeneratedCode {
			t.Errorf("Generated code mismatch")
		}

		if len(unmarshaled.Dependencies) != len(result.Dependencies) {
			t.Errorf("Expected %d dependencies, got %d",
				len(result.Dependencies), len(unmarshaled.Dependencies))
		}

		if len(unmarshaled.SetupInstructions) != len(result.SetupInstructions) {
			t.Errorf("Expected %d setup instructions, got %d",
				len(result.SetupInstructions), len(unmarshaled.SetupInstructions))
		}
	})
}

// TestRecipeTypes validates recipe type constants
func TestRecipeTypes(t *testing.T) {
	validTypes := []string{"greenfield", "brownfield", "migration"}

	for _, recipeType := range validTypes {
		t.Run("ValidType_"+recipeType, func(t *testing.T) {
			recipe := Recipe{
				Type: recipeType,
			}

			if recipe.Type != recipeType {
				t.Errorf("Expected type %s, got %s", recipeType, recipe.Type)
			}
		})
	}
}

// TestMaturityLevels validates maturity level constants
func TestMaturityLevels(t *testing.T) {
	validLevels := []string{"experimental", "emerging", "proven", "deprecated"}

	for _, level := range validLevels {
		t.Run("ValidLevel_"+level, func(t *testing.T) {
			pattern := Pattern{
				MaturityLevel: level,
			}

			if pattern.MaturityLevel != level {
				t.Errorf("Expected maturity level %s, got %s", level, pattern.MaturityLevel)
			}
		})
	}
}
