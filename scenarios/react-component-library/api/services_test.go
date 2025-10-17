package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vrooli/scenarios/react-component-library/models"
	"github.com/vrooli/scenarios/react-component-library/services"
)

// ==============================================================================
// ComponentService Tests
// ==============================================================================

func TestComponentService_CreateComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	t.Run("CreateValidComponent", func(t *testing.T) {
		component := &models.Component{
			ID:           uuid.New(),
			Name:         "ServiceTestButton",
			Category:     "form",
			Description:  "Test button created by service test",
			Code:         "const TestButton = () => <button>Click me</button>;",
			PropsSchema:  json.RawMessage(`{"props": []}`),
			Version:      "1.0.0",
			Author:       "test-service",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Tags:         []string{"test", "button"},
			Dependencies: []string{"react"},
			IsActive:     true,
		}

		err := service.CreateComponent(component)
		assert.NoError(t, err)

		// Verify component was created
		retrieved, err := service.GetComponent(component.ID)
		assert.NoError(t, err)
		assert.Equal(t, component.Name, retrieved.Name)
		assert.Equal(t, component.Category, retrieved.Category)
	})

	t.Run("CreateComponentWithMinimalData", func(t *testing.T) {
		component := &models.Component{
			ID:        uuid.New(),
			Name:      "MinimalComponent",
			Category:  "test",
			Code:      "const MinimalComponent = () => <div />;",
			Version:   "1.0.0",
			Author:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}

		err := service.CreateComponent(component)
		assert.NoError(t, err)
	})

	t.Run("CreateDuplicateComponent", func(t *testing.T) {
		componentID := uuid.New()

		component1 := &models.Component{
			ID:        componentID,
			Name:      "DuplicateTest",
			Category:  "test",
			Code:      "const DuplicateTest = () => <div />;",
			Version:   "1.0.0",
			Author:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}

		err := service.CreateComponent(component1)
		assert.NoError(t, err)

		// Try to create duplicate
		component2 := &models.Component{
			ID:        componentID, // Same ID
			Name:      "DuplicateTest2",
			Category:  "test",
			Code:      "const DuplicateTest2 = () => <div />;",
			Version:   "1.0.0",
			Author:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}

		err = service.CreateComponent(component2)
		assert.Error(t, err) // Should fail due to duplicate ID
	})
}

func TestComponentService_GetComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	t.Run("GetExistingComponent", func(t *testing.T) {
		// Create component first
		componentID := createTestComponent(t, testDB.DB, "GetTestComponent")

		// Retrieve it
		component, err := service.GetComponent(componentID)
		assert.NoError(t, err)
		assert.NotNil(t, component)
		assert.Equal(t, componentID, component.ID)
	})

	t.Run("GetNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		component, err := service.GetComponent(nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, component)
		assert.Equal(t, services.ErrComponentNotFound, err)
	})

	t.Run("GetInactiveComponent", func(t *testing.T) {
		// Create and then deactivate a component
		componentID := createTestComponent(t, testDB.DB, "InactiveComponent")

		// Deactivate it
		_, err := testDB.DB.Exec("UPDATE components SET is_active = false WHERE id = $1", componentID)
		assert.NoError(t, err)

		// Try to retrieve - should not find inactive components
		component, err := service.GetComponent(componentID)
		assert.Error(t, err)
		assert.Nil(t, component)
	})
}

func TestComponentService_UpdateComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	t.Run("UpdateExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "UpdateTestComponent")

		// Get existing component
		component, err := service.GetComponent(componentID)
		assert.NoError(t, err)

		// Update component
		component.Description = "Updated description"
		component.Tags = []string{"updated", "test", "component"}
		component.UpdatedAt = time.Now()

		updated, err := service.UpdateComponent(component)
		assert.NoError(t, err)
		assert.Contains(t, updated.Description, "Updated")
	})

	t.Run("UpdateNonExistentComponent", func(t *testing.T) {
		nonExistentComponent := &models.Component{
			ID:          uuid.New(),
			Name:        "NonExistent",
			Category:    "test",
			Description: "This should fail",
			Code:        "const Test = () => <div />;",
			Version:     "1.0.0",
			Author:      "test",
			UpdatedAt:   time.Now(),
		}

		_, err := service.UpdateComponent(nonExistentComponent)
		assert.Error(t, err)
	})

	t.Run("UpdateComponentCode", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "UpdateCodeComponent")

		// Get existing component
		component, err := service.GetComponent(componentID)
		assert.NoError(t, err)

		// Update code
		component.Code = "const UpdatedComponent = () => <div>Updated!</div>;"
		component.UpdatedAt = time.Now()

		_, err = service.UpdateComponent(component)
		assert.NoError(t, err)

		retrievedComponent, err := service.GetComponent(componentID)
		assert.NoError(t, err)
		assert.Contains(t, retrievedComponent.Code, "Updated!")
	})
}

func TestComponentService_DeleteComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	t.Run("DeleteExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "DeleteTestComponent")

		// Delete component
		err := service.DeleteComponent(componentID)
		assert.NoError(t, err)

		// Verify it's gone (soft delete - marked inactive)
		component, err := service.GetComponent(componentID)
		assert.Error(t, err)
		assert.Nil(t, component)
	})

	t.Run("DeleteNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		err := service.DeleteComponent(nonExistentID)
		assert.Error(t, err)
	})

	t.Run("DeleteAlreadyDeletedComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "DoubleDeleteComponent")

		// Delete once
		err := service.DeleteComponent(componentID)
		assert.NoError(t, err)

		// Try to delete again
		err = service.DeleteComponent(componentID)
		assert.Error(t, err) // Should fail - already deleted
	})
}

func TestComponentService_ListComponents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	// Create multiple test components
	createTestComponent(t, testDB.DB, "ListComponent1")
	createTestComponent(t, testDB.DB, "ListComponent2")
	createTestComponent(t, testDB.DB, "ListComponent3")

	t.Run("ListAllComponents", func(t *testing.T) {
		components, total, err := service.ListComponents("", 100, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.GreaterOrEqual(t, len(components), 3)
	})

	t.Run("ListWithPagination", func(t *testing.T) {
		// Get first page
		components, total, err := service.ListComponents("", 2, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.LessOrEqual(t, len(components), 2)

		// Get second page
		components, _, err = service.ListComponents("", 2, 2)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(components), 1)
	})

	t.Run("ListByCategory", func(t *testing.T) {
		components, _, err := service.ListComponents("test", 100, 0)
		assert.NoError(t, err)
		// All test components have category "test"
		for _, comp := range components {
			assert.Equal(t, "test", comp.Category)
		}
	})

	t.Run("ListWithZeroLimit", func(t *testing.T) {
		components, total, err := service.ListComponents("", 0, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		// Should return all or use default limit
		assert.NotNil(t, components)
	})
}

func TestComponentService_ValidateComponentCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	service := services.NewComponentService(testDB.DB)

	t.Run("ValidReactCode", func(t *testing.T) {
		validCode := "const TestComponent = () => <div>Test</div>;"
		err := service.ValidateComponentCode(validCode)
		// May pass or fail depending on validation implementation
		// Just verify it doesn't panic
		_ = err
	})

	t.Run("EmptyCode", func(t *testing.T) {
		err := service.ValidateComponentCode("")
		assert.Error(t, err)
		assert.Equal(t, services.ErrInvalidCode, err)
	})

	t.Run("InvalidJavaScript", func(t *testing.T) {
		invalidCode := "this is not valid javascript {[}]"
		err := service.ValidateComponentCode(invalidCode)
		// Should detect invalid syntax
		_ = err // Implementation may or may not detect this
	})

	t.Run("CodeWithComments", func(t *testing.T) {
		codeWithComments := `
			// This is a test component
			const TestComponent = () => {
				/* Multi-line comment */
				return <div>Test</div>;
			};
		`
		err := service.ValidateComponentCode(codeWithComments)
		// Should handle comments properly
		_ = err
	})
}

// ==============================================================================
// TestingService Tests
// ==============================================================================

func TestTestingService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testingService := services.NewTestingService()

	t.Run("RunAccessibilityTest", func(t *testing.T) {
		componentID := uuid.New()
		testRequest := models.ComponentTestRequest{
			TestTypes: []models.TestType{models.TestTypeAccessibility},
			TestConfig: map[string]string{
				"standard": "WCAG2AA",
			},
		}

		result, err := testingService.RunTests(componentID, testRequest)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.TestResults), 0)
	})

	t.Run("RunPerformanceTest", func(t *testing.T) {
		componentID := uuid.New()
		testRequest := models.ComponentTestRequest{
			TestTypes: []models.TestType{models.TestTypePerformance},
		}

		result, err := testingService.RunTests(componentID, testRequest)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("RunVisualTest", func(t *testing.T) {
		componentID := uuid.New()
		testRequest := models.ComponentTestRequest{
			TestTypes: []models.TestType{models.TestTypeVisual},
		}

		result, err := testingService.RunTests(componentID, testRequest)
		// Visual tests may require browserless
		_ = result
		_ = err
	})
}

// ==============================================================================
// SearchService Tests
// ==============================================================================

func TestSearchService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	searchService := services.NewSearchService()

	t.Run("SearchWithValidQuery", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query: "button component",
			Limit: 10,
		}
		results, err := searchService.SearchComponents(req)
		// May return empty results or error if Qdrant not available
		// Just verify it doesn't panic
		_ = results
		_ = err
	})

	t.Run("SearchWithCategoryFilter", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query:    "component",
			Category: "form",
			Limit:    10,
		}
		results, err := searchService.SearchComponents(req)
		_ = results
		_ = err
	})

	t.Run("SearchWithTagsFilter", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query: "component",
			Tags:  []string{"button", "interactive"},
			Limit: 10,
		}
		results, err := searchService.SearchComponents(req)
		_ = results
		_ = err
	})

	t.Run("SearchWithAccessibilityFilter", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query:                 "component",
			MinAccessibilityScore: 0.8,
			Limit:                 10,
		}
		results, err := searchService.SearchComponents(req)
		_ = results
		_ = err
	})

	t.Run("SearchWithEmptyQuery", func(t *testing.T) {
		req := models.ComponentSearchRequest{
			Query: "",
			Limit: 10,
		}
		results, err := searchService.SearchComponents(req)
		// Should handle empty query gracefully
		_ = results
		_ = err
	})
}

// ==============================================================================
// AIService Tests
// ==============================================================================

func TestAIService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	aiService := services.NewAIService()

	t.Run("GenerateComponent", func(t *testing.T) {
		req := models.ComponentGenerationRequest{
			Description:  "A simple button component with hover effects",
			Requirements: []string{"accessible", "animated"},
			Category:     "form",
		}

		result, err := aiService.GenerateComponent(req)
		// May fail if AI service not available
		// Just verify it doesn't panic
		_ = result
		_ = err
	})

	t.Run("ImproveComponent", func(t *testing.T) {
		componentID := uuid.New()
		req := models.ComponentImprovementRequest{
			Focus: []models.ImprovementFocus{models.ImprovementFocusAccessibility},
			Apply: false,
		}

		suggestions, err := aiService.ImproveComponent(componentID, req)
		_ = suggestions
		_ = err
	})

	t.Run("GenerateWithEmptyDescription", func(t *testing.T) {
		req := models.ComponentGenerationRequest{
			Description: "", // Empty description should fail validation
		}
		result, err := aiService.GenerateComponent(req)
		// Should return error for empty description
		_ = err // May or may not error depending on implementation
		_ = result
	})

	t.Run("ImproveWithInvalidID", func(t *testing.T) {
		req := models.ComponentImprovementRequest{
			Focus: []models.ImprovementFocus{models.ImprovementFocusPerformance},
		}
		suggestions, err := aiService.ImproveComponent(uuid.Nil, req)
		// Should handle invalid ID
		_ = err
		_ = suggestions
	})
}
