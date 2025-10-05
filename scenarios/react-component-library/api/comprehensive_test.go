package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vrooli/scenarios/react-component-library/models"
)

// ==============================================================================
// Comprehensive Coverage Tests - No Database Required
// ==============================================================================

func TestTestPatterns(t *testing.T) {
	t.Run("TestScenarioBuilder_Build", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		scenarios := builder.
			AddInvalidUUID("/api/v1/components/%s").
			AddNonExistentComponent("/api/v1/components/%s").
			AddInvalidJSON("/api/v1/components", "POST").
			AddMissingRequiredFields("/api/v1/components", "POST").
			Build()

		assert.Len(t, scenarios, 4)
		assert.Equal(t, "InvalidUUID", scenarios[0].Name)
		assert.Equal(t, "NonExistentComponent", scenarios[1].Name)
		assert.Equal(t, "InvalidJSON", scenarios[2].Name)
		assert.Equal(t, "MissingRequiredFields", scenarios[3].Name)
	})

	t.Run("PerformanceTestBuilder_Build", func(t *testing.T) {
		builder := NewPerformanceTestBuilder()
		tests := builder.
			AddListComponentsPerformanceTest().
			AddComponentCreationPerformanceTest().
			Build()

		assert.Len(t, tests, 2)
		assert.Equal(t, "ListComponents", tests[0].Name)
		assert.Equal(t, "CreateComponent", tests[1].Name)
		assert.Equal(t, 100, tests[0].RequestCount)
		assert.Equal(t, 50, tests[1].RequestCount)
	})

	t.Run("IntegrationTestBuilder_Build", func(t *testing.T) {
		builder := NewIntegrationTestBuilder()
		tests := builder.
			AddFullComponentLifecycleTest().
			Build()

		assert.Len(t, tests, 1)
		assert.Equal(t, "FullComponentLifecycle", tests[0].Name)
		assert.Contains(t, tests[0].Description, "create, read, update, delete")
	})
}

func TestTestHelpers_Utility(t *testing.T) {
	t.Run("setupTestLogger", func(t *testing.T) {
		cleanup := setupTestLogger()
		assert.NotNil(t, cleanup)
		cleanup()
	})

	t.Run("getValidComponentData", func(t *testing.T) {
		data := getValidComponentData()
		assert.Equal(t, "TestButton", data.Name)
		assert.Equal(t, "form", data.Category)
		assert.NotEmpty(t, data.Code)
		assert.NotEmpty(t, data.PropsSchema)
		assert.Contains(t, data.Tags, "button")
		assert.Contains(t, data.Dependencies, "react")
	})

	t.Run("getInvalidComponentData", func(t *testing.T) {
		invalidData := getInvalidComponentData()
		assert.Greater(t, len(invalidData), 0)

		// Check that we have various error scenarios
		hasEmptyName := false
		hasEmptyCode := false
		hasInvalidCode := false

		for _, scenario := range invalidData {
			if scenario.Name == "EmptyName" {
				hasEmptyName = true
				assert.Empty(t, scenario.Data.Name)
			}
			if scenario.Name == "EmptyCode" {
				hasEmptyCode = true
				assert.Empty(t, scenario.Data.Code)
			}
			if scenario.Name == "InvalidCode" {
				hasInvalidCode = true
				assert.NotEmpty(t, scenario.Data.Code)
			}
		}

		assert.True(t, hasEmptyName, "Should have EmptyName scenario")
		assert.True(t, hasEmptyCode, "Should have EmptyCode scenario")
		assert.True(t, hasInvalidCode, "Should have InvalidCode scenario")
	})
}

func TestModels(t *testing.T) {
	t.Run("Component_JSONMarshal", func(t *testing.T) {
		component := &models.Component{
			ID:           uuid.New(),
			Name:         "TestComponent",
			Category:     "form",
			Description:  "Test component description",
			Code:         "const Test = () => <div>Test</div>;",
			PropsSchema:  json.RawMessage(`{"type": "object"}`),
			Version:      "1.0.0",
			Author:       "test-user",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Tags:         []string{"test"},
			Dependencies: []string{"react"},
			IsActive:     true,
		}

		data, err := json.Marshal(component)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled models.Component
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, component.Name, unmarshaled.Name)
		assert.Equal(t, component.Category, unmarshaled.Category)
	})

	t.Run("TestType_Constants", func(t *testing.T) {
		assert.Equal(t, models.TestType("accessibility"), models.TestTypeAccessibility)
		assert.Equal(t, models.TestType("performance"), models.TestTypePerformance)
		assert.Equal(t, models.TestType("visual"), models.TestTypeVisual)
		assert.Equal(t, models.TestType("unit_test"), models.TestTypeUnitTest)
		assert.Equal(t, models.TestType("linting"), models.TestTypeLinting)
	})

	t.Run("AccessibilityLevel_Constants", func(t *testing.T) {
		assert.Equal(t, models.AccessibilityLevel("A"), models.AccessibilityLevelA)
		assert.Equal(t, models.AccessibilityLevel("AA"), models.AccessibilityLevelAA)
		assert.Equal(t, models.AccessibilityLevel("AAA"), models.AccessibilityLevelAAA)
	})

	t.Run("ExportFormat_Constants", func(t *testing.T) {
		assert.Equal(t, models.ExportFormat("npm-package"), models.ExportFormatNPMPackage)
		assert.Equal(t, models.ExportFormat("cdn"), models.ExportFormatCDN)
		assert.Equal(t, models.ExportFormat("raw-code"), models.ExportFormatRawCode)
		assert.Equal(t, models.ExportFormat("zip"), models.ExportFormatZip)
	})

	t.Run("ImprovementFocus_Constants", func(t *testing.T) {
		assert.Equal(t, models.ImprovementFocus("accessibility"), models.ImprovementFocusAccessibility)
		assert.Equal(t, models.ImprovementFocus("performance"), models.ImprovementFocusPerformance)
		assert.Equal(t, models.ImprovementFocus("code-quality"), models.ImprovementFocusCodeQuality)
		assert.Equal(t, models.ImprovementFocus("security"), models.ImprovementFocusSecurity)
	})

	t.Run("ComponentCategory_GetValidCategories", func(t *testing.T) {
		categories := models.GetValidCategories()
		assert.Len(t, categories, 8)
		assert.Contains(t, categories, models.CategoryForm)
		assert.Contains(t, categories, models.CategoryLayout)
		assert.Contains(t, categories, models.CategoryDisplay)
		assert.Contains(t, categories, models.CategoryNavigation)
		assert.Contains(t, categories, models.CategoryFeedback)
		assert.Contains(t, categories, models.CategoryDataViz)
		assert.Contains(t, categories, models.CategoryMedia)
		assert.Contains(t, categories, models.CategoryUtility)
	})

	t.Run("ComponentSearchRequest_Validation", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query:                 "button",
			Category:              "form",
			Tags:                  []string{"interactive"},
			MinAccessibilityScore: 0.8,
			Limit:                 20,
			Offset:                0,
		}

		assert.Equal(t, "button", req.Query)
		assert.Equal(t, "form", req.Category)
		assert.Equal(t, 20, req.Limit)
		assert.Equal(t, 0, req.Offset)
	})

	t.Run("ComponentGenerationRequest_Validation", func(t *testing.T) {
		req := models.ComponentGenerationRequest{
			Description:        "A button component",
			Requirements:       []string{"accessible", "responsive"},
			StylePreferences:   map[string]string{"theme": "dark"},
			AccessibilityLevel: models.AccessibilityLevelAA,
			Category:           "form",
			Dependencies:       []string{"react"},
		}

		assert.Equal(t, "A button component", req.Description)
		assert.Len(t, req.Requirements, 2)
		assert.Equal(t, "dark", req.StylePreferences["theme"])
	})

	t.Run("ComponentTestRequest_Validation", func(t *testing.T) {
		req := models.ComponentTestRequest{
			TestTypes: []models.TestType{
				models.TestTypeAccessibility,
				models.TestTypePerformance,
			},
			TestConfig: map[string]string{
				"standard": "WCAG2AA",
			},
		}

		assert.Len(t, req.TestTypes, 2)
		assert.Contains(t, req.TestTypes, models.TestTypeAccessibility)
		assert.Contains(t, req.TestTypes, models.TestTypePerformance)
	})
}

func TestHTTPTestRequest_Creation(t *testing.T) {
	t.Run("CreateBasicHTTPRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
		}

		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "/api/v1/components", req.Path)
		assert.Nil(t, req.Body)
		assert.Nil(t, req.Headers)
		assert.Nil(t, req.QueryParams)
	})

	t.Run("CreateHTTPRequestWithBody", func(t *testing.T) {
		body := map[string]interface{}{
			"name":     "TestComponent",
			"category": "form",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   body,
		}

		assert.Equal(t, "POST", req.Method)
		assert.NotNil(t, req.Body)
	})

	t.Run("CreateHTTPRequestWithHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
			Headers: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
		}

		assert.Equal(t, "Bearer token", req.Headers["Authorization"])
		assert.Equal(t, "application/json", req.Headers["Content-Type"])
	})

	t.Run("CreateHTTPRequestWithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
			QueryParams: map[string]string{
				"limit":  "20",
				"offset": "0",
			},
		}

		assert.Equal(t, "20", req.QueryParams["limit"])
		assert.Equal(t, "0", req.QueryParams["offset"])
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("getEnvWithDefault", func(t *testing.T) {
		// Test with non-existent env var
		result := getEnvWithDefault("NON_EXISTENT_VAR_12345", "default_value")
		assert.Equal(t, "default_value", result)
	})
}
