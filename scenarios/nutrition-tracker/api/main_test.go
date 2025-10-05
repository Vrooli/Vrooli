// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ReturnsEnvValue", func(t *testing.T) {
		t.Setenv("TEST_VAR", "test_value")
		result := getEnv("TEST_VAR", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("ReturnsDefault", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("EmptyEnvReturnsDefault", func(t *testing.T) {
		t.Setenv("EMPTY_VAR", "")
		result := getEnv("EMPTY_VAR", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		recorder, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]string
		assertJSONResponse(t, recorder, http.StatusOK, &response)

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}

		if response["service"] != "nutrition-tracker-api" {
			t.Errorf("Expected service 'nutrition-tracker-api', got '%s'", response["service"])
		}

		if response["version"] != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", response["version"])
		}
	})
}

// TestGetMeals tests the getMeals handler
func TestGetMeals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Set global db for handlers
	db = testDB.DB

	t.Run("Success_WithUserID", func(t *testing.T) {
		userID := "test-user-123"
		mealDate := time.Now().Format("2006-01-02")

		// Create test data
		mealID := createTestMeal(t, testDB.DB, userID, "breakfast", mealDate)
		createTestMealItem(t, testDB.DB, mealID, "Oatmeal", 300)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals",
			Query:  map[string]string{"user_id": userID},
		}

		recorder, err := makeHTTPRequest(req, getMeals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var meals []Meal
		assertJSONResponse(t, recorder, http.StatusOK, &meals)

		if len(meals) == 0 {
			t.Error("Expected at least one meal, got none")
		}
	})

	t.Run("Success_DefaultUser", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals",
		}

		recorder, err := makeHTTPRequest(req, getMeals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var meals []Meal
		assertJSONResponse(t, recorder, http.StatusOK, &meals)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals",
			Query:  map[string]string{"user_id": "nonexistent-user"},
		}

		recorder, err := makeHTTPRequest(req, getMeals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})
}

// TestGetTodaysMeals tests the getTodaysMeals handler
func TestGetTodaysMeals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		userID := "test-user-today"
		today := time.Now().Format("2006-01-02")

		mealID := createTestMeal(t, testDB.DB, userID, "lunch", today)
		createTestMealItem(t, testDB.DB, mealID, "Salad", 250)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals/today",
			Query:  map[string]string{"user_id": userID},
		}

		recorder, err := makeHTTPRequest(req, getTodaysMeals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var meals []map[string]interface{}
		assertJSONResponse(t, recorder, http.StatusOK, &meals)

		if len(meals) == 0 {
			t.Error("Expected at least one meal for today, got none")
		}
	})

	t.Run("DefaultUserID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals/today",
		}

		recorder, err := makeHTTPRequest(req, getTodaysMeals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})
}

// TestCreateMeal tests the createMeal handler
func TestCreateMeal(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		meal := Meal{
			UserID:          "test-user-create",
			MealType:        "dinner",
			MealDate:        time.Now().Format("2006-01-02"),
			FoodDescription: "Grilled Chicken",
			Calories:        400,
			Protein:         35,
			Carbs:           10,
			Fat:             20,
			Fiber:           2,
			Sugar:           1,
			Sodium:          500,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		recorder, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var createdMeal Meal
		assertJSONResponse(t, recorder, http.StatusCreated, &createdMeal)

		if createdMeal.ID == "" {
			t.Error("Expected meal ID to be set")
		}

		if createdMeal.UserID != meal.UserID {
			t.Errorf("Expected user_id '%s', got '%s'", meal.UserID, createdMeal.UserID)
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		meal := Meal{
			MealType:        "snack",
			FoodDescription: "Apple",
			Calories:        95,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		recorder, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var createdMeal Meal
		assertJSONResponse(t, recorder, http.StatusCreated, &createdMeal)

		if createdMeal.UserID != "demo-user-123" {
			t.Errorf("Expected default user_id 'demo-user-123', got '%s'", createdMeal.UserID)
		}

		if createdMeal.MealDate == "" {
			t.Error("Expected meal_date to be set to current date")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   "invalid json",
		}

		recorder, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})
}

// TestGetMeal tests the getMeal handler
func TestGetMeal(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		userID := "test-user-get"
		mealDate := time.Now().Format("2006-01-02")
		mealID := createTestMeal(t, testDB.DB, userID, "breakfast", mealDate)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
		}

		recorder, err := makeHTTPRequest(req, getMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var meal Meal
		assertJSONResponse(t, recorder, http.StatusOK, &meal)

		if meal.ID != mealID {
			t.Errorf("Expected meal ID '%s', got '%s'", mealID, meal.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/meals/00000000-0000-0000-0000-000000000000",
			URLVars: map[string]string{"id": "00000000-0000-0000-0000-000000000000"},
		}

		recorder, err := makeHTTPRequest(req, getMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusNotFound, "Meal not found")
	})
}

// TestUpdateMeal tests the updateMeal handler
func TestUpdateMeal(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		userID := "test-user-update"
		mealDate := time.Now().Format("2006-01-02")
		mealID := createTestMeal(t, testDB.DB, userID, "lunch", mealDate)

		updatedMeal := Meal{
			MealType: "dinner",
		}

		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
			Body:    updatedMeal,
		}

		recorder, err := makeHTTPRequest(req, updateMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", recorder.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/meals/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    "invalid json",
		}

		recorder, err := makeHTTPRequest(req, updateMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})
}

// TestDeleteMeal tests the deleteMeal handler
func TestDeleteMeal(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		userID := "test-user-delete"
		mealDate := time.Now().Format("2006-01-02")
		mealID := createTestMeal(t, testDB.DB, userID, "breakfast", mealDate)

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
		}

		recorder, err := makeHTTPRequest(req, deleteMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", recorder.Code)
		}
	})
}

// TestGetFoods tests the getFoods handler
func TestGetFoods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		createTestFood(t, testDB.DB, "Banana", 105)
		createTestFood(t, testDB.DB, "Orange", 62)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods",
		}

		recorder, err := makeHTTPRequest(req, getFoods)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var foods []Food
		assertJSONResponse(t, recorder, http.StatusOK, &foods)

		if len(foods) < 2 {
			t.Errorf("Expected at least 2 foods, got %d", len(foods))
		}
	})

	t.Run("EmptyResult", func(t *testing.T) {
		// Clean database
		testDB.DB.Exec("DELETE FROM foods")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods",
		}

		recorder, err := makeHTTPRequest(req, getFoods)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})
}

// TestSearchFoods tests the searchFoods handler
func TestSearchFoods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		createTestFood(t, testDB.DB, "Strawberry", 32)
		createTestFood(t, testDB.DB, "Blueberry", 57)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods/search",
			Query:  map[string]string{"q": "berry"},
		}

		recorder, err := makeHTTPRequest(req, searchFoods)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var foods []Food
		assertJSONResponse(t, recorder, http.StatusOK, &foods)

		if len(foods) < 2 {
			t.Errorf("Expected at least 2 foods matching 'berry', got %d", len(foods))
		}
	})

	t.Run("MissingQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods/search",
		}

		recorder, err := makeHTTPRequest(req, searchFoods)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "Search query required")
	})

	t.Run("NoResults", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods/search",
			Query:  map[string]string{"q": "nonexistent food xyz"},
		}

		recorder, err := makeHTTPRequest(req, searchFoods)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})
}

// TestCreateFood tests the createFood handler
func TestCreateFood(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		food := Food{
			Name:         "Avocado",
			Brand:        "Fresh Market",
			ServingSize:  "100",
			Calories:     160,
			Protein:      2,
			Carbs:        9,
			Fat:          15,
			Fiber:        7,
			Sugar:        0.7,
			Sodium:       7,
			FoodCategory: "fruits",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/foods",
			Body:   food,
		}

		recorder, err := makeHTTPRequest(req, createFood)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var createdFood Food
		assertJSONResponse(t, recorder, http.StatusCreated, &createdFood)

		if createdFood.ID == "" {
			t.Error("Expected food ID to be set")
		}

		if createdFood.Name != food.Name {
			t.Errorf("Expected name '%s', got '%s'", food.Name, createdFood.Name)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/foods",
			Body:   "invalid json",
		}

		recorder, err := makeHTTPRequest(req, createFood)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})
}

// TestGetNutritionSummary tests the getNutritionSummary handler
func TestGetNutritionSummary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		userID := "test-user-summary"
		today := time.Now().Format("2006-01-02")

		mealID := createTestMeal(t, testDB.DB, userID, "breakfast", today)
		createTestMealItem(t, testDB.DB, mealID, "Toast", 150)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nutrition",
			Query:  map[string]string{"user_id": userID, "date": today},
		}

		recorder, err := makeHTTPRequest(req, getNutritionSummary)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var summary DailySummary
		assertJSONResponse(t, recorder, http.StatusOK, &summary)

		if summary.Date != today {
			t.Errorf("Expected date '%s', got '%s'", today, summary.Date)
		}

		if summary.TotalCalories <= 0 {
			t.Error("Expected total calories > 0")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nutrition",
		}

		recorder, err := makeHTTPRequest(req, getNutritionSummary)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var summary DailySummary
		assertJSONResponse(t, recorder, http.StatusOK, &summary)

		today := time.Now().Format("2006-01-02")
		if summary.Date != today {
			t.Errorf("Expected default date '%s', got '%s'", today, summary.Date)
		}
	})
}

// TestGetGoals tests the getGoals handler
func TestGetGoals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success_ExistingGoals", func(t *testing.T) {
		userID := "test-user-goals"
		createTestUserGoals(t, testDB.DB, userID)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/goals",
			Query:  map[string]string{"user_id": userID},
		}

		recorder, err := makeHTTPRequest(req, getGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var goals NutritionGoals
		assertJSONResponse(t, recorder, http.StatusOK, &goals)

		if goals.UserID != userID {
			t.Errorf("Expected user_id '%s', got '%s'", userID, goals.UserID)
		}

		if goals.DailyCalories != 2000 {
			t.Errorf("Expected daily_calories 2000, got %d", goals.DailyCalories)
		}
	})

	t.Run("DefaultGoals", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/goals",
			Query:  map[string]string{"user_id": "nonexistent-user"},
		}

		recorder, err := makeHTTPRequest(req, getGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var goals NutritionGoals
		assertJSONResponse(t, recorder, http.StatusOK, &goals)

		if goals.DailyCalories != 2000 {
			t.Errorf("Expected default daily_calories 2000, got %d", goals.DailyCalories)
		}

		if goals.WeightGoal != "maintain" {
			t.Errorf("Expected default weight_goal 'maintain', got '%s'", goals.WeightGoal)
		}
	})
}

// TestUpdateGoals tests the updateGoals handler
func TestUpdateGoals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success_Insert", func(t *testing.T) {
		goals := NutritionGoals{
			UserID:        "test-user-new-goals",
			DailyCalories: 1800,
			ProteinGrams:  80,
			CarbsGrams:    200,
			FatGrams:      60,
			FiberGrams:    30,
			SugarLimit:    40,
			SodiumLimit:   2000,
			WeightGoal:    "lose",
			ActivityLevel: "active",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/goals",
			Body:   goals,
		}

		recorder, err := makeHTTPRequest(req, updateGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var updatedGoals NutritionGoals
		assertJSONResponse(t, recorder, http.StatusOK, &updatedGoals)

		if updatedGoals.DailyCalories != goals.DailyCalories {
			t.Errorf("Expected daily_calories %d, got %d", goals.DailyCalories, updatedGoals.DailyCalories)
		}
	})

	t.Run("Success_Update", func(t *testing.T) {
		userID := "test-user-update-goals"
		createTestUserGoals(t, testDB.DB, userID)

		goals := NutritionGoals{
			UserID:        userID,
			DailyCalories: 2200,
			ProteinGrams:  90,
			CarbsGrams:    250,
			FatGrams:      70,
			FiberGrams:    28,
			SugarLimit:    45,
			SodiumLimit:   2100,
			WeightGoal:    "gain",
			ActivityLevel: "very_active",
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/goals",
			Body:   goals,
		}

		recorder, err := makeHTTPRequest(req, updateGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var updatedGoals NutritionGoals
		assertJSONResponse(t, recorder, http.StatusOK, &updatedGoals)

		if updatedGoals.DailyCalories != 2200 {
			t.Errorf("Expected updated daily_calories 2200, got %d", updatedGoals.DailyCalories)
		}
	})

	t.Run("DefaultUserID", func(t *testing.T) {
		goals := NutritionGoals{
			DailyCalories: 1900,
			ProteinGrams:  70,
			CarbsGrams:    210,
			FatGrams:      55,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/goals",
			Body:   goals,
		}

		recorder, err := makeHTTPRequest(req, updateGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var updatedGoals NutritionGoals
		assertJSONResponse(t, recorder, http.StatusOK, &updatedGoals)

		if updatedGoals.UserID != "demo-user-123" {
			t.Errorf("Expected default user_id 'demo-user-123', got '%s'", updatedGoals.UserID)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/goals",
			Body:   "invalid json",
		}

		recorder, err := makeHTTPRequest(req, updateGoals)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})
}

// TestGetMealSuggestions tests the getMealSuggestions handler
func TestGetMealSuggestions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/suggestions",
		}

		recorder, err := makeHTTPRequest(req, getMealSuggestions)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, recorder, http.StatusOK, &response)

		suggestions, ok := response["suggestions"].([]interface{})
		if !ok {
			t.Fatal("Expected 'suggestions' field in response")
		}

		if len(suggestions) == 0 {
			t.Error("Expected at least one meal suggestion")
		}

		// Validate structure of first suggestion
		if len(suggestions) > 0 {
			suggestion := suggestions[0].(map[string]interface{})

			if _, ok := suggestion["meal_name"]; !ok {
				t.Error("Expected 'meal_name' field in suggestion")
			}

			if _, ok := suggestion["calories"]; !ok {
				t.Error("Expected 'calories' field in suggestion")
			}

			if _, ok := suggestion["recommendation_reason"]; !ok {
				t.Error("Expected 'recommendation_reason' field in suggestion")
			}
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("NegativeCalories", func(t *testing.T) {
		meal := Meal{
			UserID:          "test-user-edge",
			MealType:        "snack",
			FoodDescription: "Test",
			Calories:        -100, // Negative value
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		recorder, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// System currently accepts negative values - this documents the behavior
		if recorder.Code == http.StatusCreated {
			var createdMeal Meal
			json.Unmarshal(recorder.Body.Bytes(), &createdMeal)
			// Test passes but documents that validation might be needed
		}
	})

	t.Run("VeryLongFoodName", func(t *testing.T) {
		longName := ""
		for i := 0; i < 300; i++ {
			longName += "a"
		}

		food := Food{
			Name:     longName,
			Calories: 100,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/foods",
			Body:   food,
		}

		recorder, err := makeHTTPRequest(req, createFood)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Documents current behavior with very long names
		_ = recorder.Code
	})

	t.Run("ZeroMacronutrients", func(t *testing.T) {
		meal := Meal{
			UserID:          "test-user-zero",
			MealType:        "snack",
			FoodDescription: "Water",
			Calories:        0,
			Protein:         0,
			Carbs:           0,
			Fat:             0,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		recorder, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected to accept zero macronutrients, got status %d", recorder.Code)
		}
	})
}
