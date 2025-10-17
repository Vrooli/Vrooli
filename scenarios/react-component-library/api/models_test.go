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
// Component Model Tests
// ==============================================================================

func TestComponentModel(t *testing.T) {
	t.Run("CreateValidComponent", func(t *testing.T) {
		component := models.Component{
			ID:          uuid.New(),
			Name:        "TestButton",
			Category:    "form",
			Description: "A test button component",
			Code:        "const TestButton = () => <button>Test</button>;",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Version:     "1.0.0",
			Author:      "test-user",
			IsActive:    true,
		}

		assert.NotEqual(t, uuid.Nil, component.ID)
		assert.Equal(t, "TestButton", component.Name)
		assert.True(t, component.IsActive)
	})

	t.Run("ComponentJSONSerialization", func(t *testing.T) {
		component := models.Component{
			ID:          uuid.New(),
			Name:        "JSONTest",
			Category:    "test",
			Description: "Test JSON serialization",
			Code:        "const JSONTest = () => <div />;",
			Version:     "1.0.0",
			Author:      "test",
			Tags:        []string{"json", "test"},
			IsActive:    true,
		}

		// Serialize to JSON
		jsonData, err := json.Marshal(component)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Deserialize from JSON
		var decoded models.Component
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, component.ID, decoded.ID)
		assert.Equal(t, component.Name, decoded.Name)
		assert.Equal(t, component.Tags, decoded.Tags)
	})

	t.Run("ComponentWithComplexData", func(t *testing.T) {
		propsSchema := json.RawMessage(`{"type": "object", "properties": {"label": {"type": "string"}}}`)
		perfMetrics := json.RawMessage(`{"renderTime": 15, "bundleSize": 1024}`)

		score := 0.95
		component := models.Component{
			ID:                 uuid.New(),
			Name:               "ComplexButton",
			Category:           "form",
			Description:        "Complex button with schema",
			Code:               "const ComplexButton = ({label}) => <button>{label}</button>;",
			PropsSchema:        propsSchema,
			PerformanceMetrics: perfMetrics,
			AccessibilityScore: &score,
			Tags:               []string{"complex", "button", "accessible"},
			Dependencies:       []string{"react", "react-dom"},
			Screenshots:        []string{"screenshot1.png", "screenshot2.png"},
			ExampleUsage:       `<ComplexButton label="Click me" />`,
			Version:            "1.0.0",
			Author:             "test",
			IsActive:           true,
		}

		assert.Equal(t, propsSchema, component.PropsSchema)
		assert.Equal(t, perfMetrics, component.PerformanceMetrics)
		assert.NotNil(t, component.AccessibilityScore)
		assert.Equal(t, 0.95, *component.AccessibilityScore)
	})
}

// ==============================================================================
// ComponentVersion Model Tests
// ==============================================================================

func TestComponentVersionModel(t *testing.T) {
	t.Run("CreateComponentVersion", func(t *testing.T) {
		version := models.ComponentVersion{
			ID:              uuid.New(),
			ComponentID:     uuid.New(),
			Version:         "2.0.0",
			Code:            "const UpdatedComponent = () => <div>Updated</div>;",
			Changelog:       "Major update with breaking changes",
			BreakingChanges: []string{"Removed prop X", "Changed prop Y type"},
			Deprecated:      false,
			CreatedAt:       time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, version.ID)
		assert.Equal(t, "2.0.0", version.Version)
		assert.Len(t, version.BreakingChanges, 2)
		assert.False(t, version.Deprecated)
	})

	t.Run("DeprecatedVersion", func(t *testing.T) {
		version := models.ComponentVersion{
			ID:          uuid.New(),
			ComponentID: uuid.New(),
			Version:     "1.0.0",
			Code:        "const OldComponent = () => <div>Old</div>;",
			Deprecated:  true,
			CreatedAt:   time.Now(),
		}

		assert.True(t, version.Deprecated)
	})
}

// ==============================================================================
// TestResult Model Tests
// ==============================================================================

func TestTestResultModel(t *testing.T) {
	t.Run("CreateAccessibilityTestResult", func(t *testing.T) {
		results := json.RawMessage(`{"violations": [], "passes": 15, "incomplete": 0}`)

		testResult := models.TestResult{
			ID:           uuid.New(),
			ComponentID:  uuid.New(),
			TestType:     models.TestTypeAccessibility,
			Results:      results,
			Passed:       true,
			Score:        0.95,
			TestedAt:     time.Now(),
			TestDuration: 150,
		}

		assert.Equal(t, models.TestTypeAccessibility, testResult.TestType)
		assert.True(t, testResult.Passed)
		assert.Equal(t, 0.95, testResult.Score)
	})

	t.Run("CreatePerformanceTestResult", func(t *testing.T) {
		results := json.RawMessage(`{"renderTime": 14.5, "memoryUsage": 512, "bundleSize": 2048}`)

		testResult := models.TestResult{
			ID:           uuid.New(),
			ComponentID:  uuid.New(),
			TestType:     models.TestTypePerformance,
			Results:      results,
			Passed:       true,
			Score:        0.88,
			TestedAt:     time.Now(),
			TestDuration: 500,
		}

		assert.Equal(t, models.TestTypePerformance, testResult.TestType)
		assert.Equal(t, 0.88, testResult.Score)
		assert.Equal(t, 500, testResult.TestDuration)
	})

	t.Run("FailedTestResult", func(t *testing.T) {
		results := json.RawMessage(`{"violations": [{"rule": "aria-required", "severity": "critical"}]}`)

		testResult := models.TestResult{
			ID:          uuid.New(),
			ComponentID: uuid.New(),
			TestType:    models.TestTypeAccessibility,
			Results:     results,
			Passed:      false,
			Score:       0.45,
			TestedAt:    time.Now(),
		}

		assert.False(t, testResult.Passed)
		assert.Less(t, testResult.Score, 0.5)
	})
}

// ==============================================================================
// Request/Response Model Tests
// ==============================================================================

func TestComponentSearchRequest(t *testing.T) {
	t.Run("ValidSearchRequest", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query:                 "button component",
			Category:              "form",
			Tags:                  []string{"accessible", "interactive"},
			MinAccessibilityScore: 0.8,
			Limit:                 20,
			Offset:                0,
		}

		assert.Equal(t, "button component", req.Query)
		assert.Equal(t, "form", req.Category)
		assert.Len(t, req.Tags, 2)
		assert.Equal(t, 0.8, req.MinAccessibilityScore)
	})

	t.Run("SearchRequestSerialization", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query:  "modal",
			Limit:  10,
			Offset: 0,
		}

		jsonData, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded models.ComponentSearchRequest
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, req.Query, decoded.Query)
	})
}

func TestComponentGenerationRequest(t *testing.T) {
	t.Run("ValidGenerationRequest", func(t *testing.T) {
		req := models.ComponentGenerationRequest{
			Description:        "A simple modal dialog component",
			Requirements:       []string{"accessible", "animated", "responsive"},
			StylePreferences:   map[string]string{"theme": "minimal", "color": "blue"},
			AccessibilityLevel: models.AccessibilityLevelAA,
			Category:           "feedback",
			Dependencies:       []string{"react", "framer-motion"},
		}

		assert.NotEmpty(t, req.Description)
		assert.Len(t, req.Requirements, 3)
		assert.Equal(t, models.AccessibilityLevelAA, req.AccessibilityLevel)
		assert.Equal(t, "minimal", req.StylePreferences["theme"])
	})

	t.Run("MinimalGenerationRequest", func(t *testing.T) {
		req := models.ComponentGenerationRequest{
			Description: "A simple button",
		}

		assert.NotEmpty(t, req.Description)
		assert.Nil(t, req.Requirements)
	})
}

func TestComponentTestRequest(t *testing.T) {
	t.Run("SingleTestType", func(t *testing.T) {
		req := models.ComponentTestRequest{
			TestTypes: []models.TestType{models.TestTypeAccessibility},
		}

		assert.Len(t, req.TestTypes, 1)
		assert.Equal(t, models.TestTypeAccessibility, req.TestTypes[0])
	})

	t.Run("MultipleTestTypes", func(t *testing.T) {
		req := models.ComponentTestRequest{
			TestTypes: []models.TestType{
				models.TestTypeAccessibility,
				models.TestTypePerformance,
				models.TestTypeVisual,
			},
			TestConfig: map[string]string{
				"viewport": "1920x1080",
				"browser":  "chrome",
			},
		}

		assert.Len(t, req.TestTypes, 3)
		assert.NotNil(t, req.TestConfig)
		assert.Equal(t, "1920x1080", req.TestConfig["viewport"])
	})
}

func TestComponentExportRequest(t *testing.T) {
	t.Run("NPMPackageExport", func(t *testing.T) {
		req := models.ComponentExportRequest{
			Format:      models.ExportFormatNPMPackage,
			IncludeDeps: true,
			Output:      "/tmp/component-export",
		}

		assert.Equal(t, models.ExportFormatNPMPackage, req.Format)
		assert.True(t, req.IncludeDeps)
	})

	t.Run("RawCodeExport", func(t *testing.T) {
		req := models.ComponentExportRequest{
			Format:      models.ExportFormatRawCode,
			IncludeDeps: false,
		}

		assert.Equal(t, models.ExportFormatRawCode, req.Format)
		assert.False(t, req.IncludeDeps)
	})
}

// ==============================================================================
// Enum and Constant Tests
// ==============================================================================

func TestTestTypes(t *testing.T) {
	t.Run("AllTestTypesAreDefined", func(t *testing.T) {
		testTypes := []models.TestType{
			models.TestTypeAccessibility,
			models.TestTypePerformance,
			models.TestTypeVisual,
			models.TestTypeUnitTest,
			models.TestTypeLinting,
		}

		assert.Len(t, testTypes, 5)

		for _, testType := range testTypes {
			assert.NotEmpty(t, string(testType))
		}
	})
}

func TestAccessibilityLevels(t *testing.T) {
	t.Run("AllAccessibilityLevelsAreDefined", func(t *testing.T) {
		levels := []models.AccessibilityLevel{
			models.AccessibilityLevelA,
			models.AccessibilityLevelAA,
			models.AccessibilityLevelAAA,
		}

		assert.Len(t, levels, 3)
		assert.Equal(t, "A", string(models.AccessibilityLevelA))
		assert.Equal(t, "AA", string(models.AccessibilityLevelAA))
		assert.Equal(t, "AAA", string(models.AccessibilityLevelAAA))
	})
}

func TestExportFormats(t *testing.T) {
	t.Run("AllExportFormatsAreDefined", func(t *testing.T) {
		formats := []models.ExportFormat{
			models.ExportFormatNPMPackage,
			models.ExportFormatCDN,
			models.ExportFormatRawCode,
			models.ExportFormatZip,
		}

		assert.Len(t, formats, 4)
		for _, format := range formats {
			assert.NotEmpty(t, string(format))
		}
	})
}

func TestComponentCategories(t *testing.T) {
	t.Run("GetValidCategories", func(t *testing.T) {
		categories := models.GetValidCategories()

		assert.GreaterOrEqual(t, len(categories), 8)

		// Check specific categories exist
		categoryStrings := make([]string, len(categories))
		for i, cat := range categories {
			categoryStrings[i] = string(cat)
		}

		assert.Contains(t, categoryStrings, "form")
		assert.Contains(t, categoryStrings, "layout")
		assert.Contains(t, categoryStrings, "display")
	})

	t.Run("CategoryForm", func(t *testing.T) {
		assert.Equal(t, "form", string(models.CategoryForm))
	})

	t.Run("CategoryLayout", func(t *testing.T) {
		assert.Equal(t, "layout", string(models.CategoryLayout))
	})
}

func TestImprovementFocus(t *testing.T) {
	t.Run("AllImprovementFocusAreasAreDefined", func(t *testing.T) {
		focuses := []models.ImprovementFocus{
			models.ImprovementFocusAccessibility,
			models.ImprovementFocusPerformance,
			models.ImprovementFocusCodeQuality,
			models.ImprovementFocusSecurity,
		}

		assert.Len(t, focuses, 4)
		for _, focus := range focuses {
			assert.NotEmpty(t, string(focus))
		}
	})
}

// ==============================================================================
// Edge Cases Tests
// ==============================================================================

func TestModelEdgeCases(t *testing.T) {
	t.Run("ComponentWithNilAccessibilityScore", func(t *testing.T) {
		component := models.Component{
			ID:                 uuid.New(),
			Name:               "NoScoreComponent",
			AccessibilityScore: nil,
		}

		assert.Nil(t, component.AccessibilityScore)

		// Should serialize correctly
		jsonData, err := json.Marshal(component)
		assert.NoError(t, err)

		var decoded models.Component
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Nil(t, decoded.AccessibilityScore)
	})

	t.Run("ComponentWithEmptyArrays", func(t *testing.T) {
		component := models.Component{
			ID:           uuid.New(),
			Name:         "EmptyArraysComponent",
			Tags:         []string{},
			Dependencies: []string{},
			Screenshots:  []string{},
		}

		jsonData, err := json.Marshal(component)
		assert.NoError(t, err)

		var decoded models.Component
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		// Arrays should be empty, not nil
		assert.NotNil(t, decoded.Tags)
		assert.Len(t, decoded.Tags, 0)
	})

	t.Run("ComponentWithLargeCode", func(t *testing.T) {
		largeCode := string(make([]byte, 100000)) // 100KB of code

		component := models.Component{
			ID:   uuid.New(),
			Name: "LargeCodeComponent",
			Code: largeCode,
		}

		assert.Len(t, component.Code, 100000)

		// Should serialize without error
		_, err := json.Marshal(component)
		assert.NoError(t, err)
	})
}
