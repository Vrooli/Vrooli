// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestMealStructJSON tests Meal struct JSON marshaling/unmarshaling
func TestMealStructJSON(t *testing.T) {
	t.Run("MarshalMeal", func(t *testing.T) {
		meal := Meal{
			ID:              "test-id-123",
			UserID:          "user-456",
			MealType:        "breakfast",
			MealDate:        "2025-01-01",
			FoodDescription: "Oatmeal with berries",
			Calories:        350,
			Protein:         12,
			Carbs:           58,
			Fat:             8,
			Fiber:           10,
			Sugar:           12,
			Sodium:          150,
		}

		data, err := json.Marshal(meal)
		if err != nil {
			t.Fatalf("Failed to marshal meal: %v", err)
		}

		if !strings.Contains(string(data), "test-id-123") {
			t.Error("Marshaled JSON should contain meal ID")
		}

		if !strings.Contains(string(data), "breakfast") {
			t.Error("Marshaled JSON should contain meal type")
		}
	})

	t.Run("UnmarshalMeal", func(t *testing.T) {
		jsonData := `{
			"id": "meal-789",
			"user_id": "user-123",
			"meal_type": "lunch",
			"meal_date": "2025-01-02",
			"food_description": "Grilled chicken salad",
			"calories": 400,
			"protein": 35,
			"carbs": 20,
			"fat": 18
		}`

		var meal Meal
		err := json.Unmarshal([]byte(jsonData), &meal)
		if err != nil {
			t.Fatalf("Failed to unmarshal meal: %v", err)
		}

		if meal.ID != "meal-789" {
			t.Errorf("Expected ID 'meal-789', got '%s'", meal.ID)
		}

		if meal.MealType != "lunch" {
			t.Errorf("Expected meal_type 'lunch', got '%s'", meal.MealType)
		}

		if meal.Calories != 400 {
			t.Errorf("Expected calories 400, got %.2f", meal.Calories)
		}
	})
}

// TestFoodStructJSON tests Food struct JSON marshaling/unmarshaling
func TestFoodStructJSON(t *testing.T) {
	t.Run("MarshalFood", func(t *testing.T) {
		food := Food{
			ID:           "food-123",
			Name:         "Apple",
			Brand:        "Organic Valley",
			ServingSize:  "182",
			Calories:     95,
			Protein:      0.5,
			Carbs:        25,
			Fat:          0.3,
			Fiber:        4.4,
			Sugar:        19,
			Sodium:       2,
			FoodCategory: "fruits",
		}

		data, err := json.Marshal(food)
		if err != nil {
			t.Fatalf("Failed to marshal food: %v", err)
		}

		if !strings.Contains(string(data), "Apple") {
			t.Error("Marshaled JSON should contain food name")
		}

		if !strings.Contains(string(data), "fruits") {
			t.Error("Marshaled JSON should contain food category")
		}
	})

	t.Run("UnmarshalFood", func(t *testing.T) {
		jsonData := `{
			"id": "food-456",
			"name": "Banana",
			"brand": "Dole",
			"serving_size": "118",
			"calories": 105,
			"protein": 1.3,
			"carbs": 27,
			"fat": 0.4,
			"fiber": 3.1,
			"sugar": 14,
			"sodium": 1,
			"food_category": "fruits"
		}`

		var food Food
		err := json.Unmarshal([]byte(jsonData), &food)
		if err != nil {
			t.Fatalf("Failed to unmarshal food: %v", err)
		}

		if food.Name != "Banana" {
			t.Errorf("Expected name 'Banana', got '%s'", food.Name)
		}

		if food.Calories != 105 {
			t.Errorf("Expected calories 105, got %.2f", food.Calories)
		}
	})
}

// TestNutritionGoalsStructJSON tests NutritionGoals struct
func TestNutritionGoalsStructJSON(t *testing.T) {
	t.Run("MarshalGoals", func(t *testing.T) {
		goals := NutritionGoals{
			UserID:        "user-789",
			DailyCalories: 2000,
			ProteinGrams:  75,
			CarbsGrams:    225,
			FatGrams:      65,
			FiberGrams:    25,
			SugarLimit:    50,
			SodiumLimit:   2300,
			WeightGoal:    "maintain",
			ActivityLevel: "moderately_active",
		}

		data, err := json.Marshal(goals)
		if err != nil {
			t.Fatalf("Failed to marshal goals: %v", err)
		}

		if !strings.Contains(string(data), "user-789") {
			t.Error("Marshaled JSON should contain user ID")
		}

		if !strings.Contains(string(data), "maintain") {
			t.Error("Marshaled JSON should contain weight goal")
		}
	})

	t.Run("UnmarshalGoals", func(t *testing.T) {
		jsonData := `{
			"user_id": "user-999",
			"daily_calories": 1800,
			"protein_grams": 80,
			"carbs_grams": 200,
			"fat_grams": 60,
			"fiber_grams": 30,
			"sugar_limit": 40,
			"sodium_limit": 2000,
			"weight_goal": "lose",
			"activity_level": "active"
		}`

		var goals NutritionGoals
		err := json.Unmarshal([]byte(jsonData), &goals)
		if err != nil {
			t.Fatalf("Failed to unmarshal goals: %v", err)
		}

		if goals.DailyCalories != 1800 {
			t.Errorf("Expected daily_calories 1800, got %d", goals.DailyCalories)
		}

		if goals.WeightGoal != "lose" {
			t.Errorf("Expected weight_goal 'lose', got '%s'", goals.WeightGoal)
		}

		if goals.ActivityLevel != "active" {
			t.Errorf("Expected activity_level 'active', got '%s'", goals.ActivityLevel)
		}
	})
}

// TestDailySummaryStructJSON tests DailySummary struct
func TestDailySummaryStructJSON(t *testing.T) {
	t.Run("MarshalSummary", func(t *testing.T) {
		summary := DailySummary{
			Date:          "2025-01-03",
			TotalCalories: 1850,
			TotalProtein:  92,
			TotalCarbs:    210,
			TotalFat:      68,
			TotalFiber:    28,
			TotalSugar:    45,
			TotalSodium:   2100,
			MealCount:     4,
		}

		data, err := json.Marshal(summary)
		if err != nil {
			t.Fatalf("Failed to marshal summary: %v", err)
		}

		if !strings.Contains(string(data), "2025-01-03") {
			t.Error("Marshaled JSON should contain date")
		}
	})

	t.Run("UnmarshalSummary", func(t *testing.T) {
		jsonData := `{
			"date": "2025-01-04",
			"total_calories": 2100,
			"total_protein": 95,
			"total_carbs": 230,
			"total_fat": 72,
			"total_fiber": 26,
			"total_sugar": 48,
			"total_sodium": 2200,
			"meal_count": 5
		}`

		var summary DailySummary
		err := json.Unmarshal([]byte(jsonData), &summary)
		if err != nil {
			t.Fatalf("Failed to unmarshal summary: %v", err)
		}

		if summary.Date != "2025-01-04" {
			t.Errorf("Expected date '2025-01-04', got '%s'", summary.Date)
		}

		if summary.TotalCalories != 2100 {
			t.Errorf("Expected total_calories 2100, got %.2f", summary.TotalCalories)
		}

		if summary.MealCount != 5 {
			t.Errorf("Expected meal_count 5, got %d", summary.MealCount)
		}
	})
}

// TestHealthCheckWithoutDB tests health check without database dependency
func TestHealthCheckWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ReturnsCorrectFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		recorder, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var response map[string]string
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		requiredFields := []string{"status", "service", "version"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Response missing required field: %s", field)
			}
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}
	})

	t.Run("ContentTypeJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		recorder, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		contentType := recorder.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
		}
	})
}

// TestGetMealSuggestionsStructure tests meal suggestions without external dependencies
func TestGetMealSuggestionsStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ReturnsValidStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/suggestions",
		}

		recorder, err := makeHTTPRequest(req, getMealSuggestions)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var response map[string]interface{}
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		suggestions, ok := response["suggestions"].([]interface{})
		if !ok {
			t.Fatal("Expected 'suggestions' field to be an array")
		}

		if len(suggestions) == 0 {
			t.Error("Expected at least one suggestion")
		}

		// Validate first suggestion structure
		if len(suggestions) > 0 {
			suggestion, ok := suggestions[0].(map[string]interface{})
			if !ok {
				t.Fatal("Expected suggestion to be an object")
			}

			requiredFields := []string{"meal_name", "calories", "protein", "carbs", "fat", "recommendation_reason"}
			for _, field := range requiredFields {
				if _, exists := suggestion[field]; !exists {
					t.Errorf("Suggestion missing required field: %s", field)
				}
			}

			// Validate data types
			if _, ok := suggestion["meal_name"].(string); !ok {
				t.Error("meal_name should be a string")
			}

			if _, ok := suggestion["recommendation_reason"].(string); !ok {
				t.Error("recommendation_reason should be a string")
			}

			// Check that nutritional values are numbers
			numericFields := []string{"calories", "protein", "carbs", "fat"}
			for _, field := range numericFields {
				val := suggestion[field]
				switch val.(type) {
				case float64, int, int64:
					// Valid numeric type
				default:
					t.Errorf("Field %s should be numeric, got %T", field, val)
				}
			}
		}
	})

	t.Run("SuggestionsHaveReasonableValues", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/suggestions",
		}

		recorder, err := makeHTTPRequest(req, getMealSuggestions)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &response)

		suggestions := response["suggestions"].([]interface{})
		for i, s := range suggestions {
			suggestion := s.(map[string]interface{})

			calories := suggestion["calories"].(float64)
			if calories < 0 || calories > 10000 {
				t.Errorf("Suggestion %d has unreasonable calories: %.2f", i, calories)
			}

			protein := suggestion["protein"].(float64)
			if protein < 0 || protein > 500 {
				t.Errorf("Suggestion %d has unreasonable protein: %.2f", i, protein)
			}
		}
	})
}

// TestHTTPRequestHelpers tests the test helper functions
func TestHTTPRequestHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MakeHTTPRequestWithBody", func(t *testing.T) {
		body := map[string]string{"test": "value"}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   body,
		}

		recorder, err := makeHTTPRequest(req, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})

	t.Run("MakeHTTPRequestWithQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Query:  map[string]string{"param1": "value1", "param2": "value2"},
		}

		recorder, err := makeHTTPRequest(req, func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			if query.Get("param1") != "value1" {
				t.Error("Query param1 not set correctly")
			}
			if query.Get("param2") != "value2" {
				t.Error("Query param2 not set correctly")
			}
			w.WriteHeader(http.StatusOK)
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})

	t.Run("MakeHTTPRequestWithHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/test",
			Headers: map[string]string{"X-Custom-Header": "custom-value"},
		}

		recorder, err := makeHTTPRequest(req, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Custom-Header") != "custom-value" {
				t.Error("Custom header not set correctly")
			}
			w.WriteHeader(http.StatusOK)
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})
}

// TestAssertionHelpers tests the test assertion helper functions
func TestAssertionHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AssertJSONResponse_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		}

		recorder, _ := makeHTTPRequest(req, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"key": "value"})
		})

		var response map[string]string
		assertJSONResponse(t, recorder, http.StatusOK, &response)

		if response["key"] != "value" {
			t.Errorf("Expected key='value', got key='%s'", response["key"])
		}
	})

	t.Run("AssertErrorResponse_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		}

		recorder, _ := makeHTTPRequest(req, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Test error message", http.StatusBadRequest)
		})

		assertErrorResponse(t, recorder, http.StatusBadRequest, "Test error message")
	})
}

// TestContainsHelper tests the contains helper function
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"EmptyString", "", "", true},
		{"EmptySubstring", "hello", "", true},
		{"ExactMatch", "hello", "hello", true},
		{"Contains", "hello world", "world", true},
		{"NotContains", "hello world", "xyz", false},
		{"CaseSensitive", "Hello", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, expected %v", tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}
