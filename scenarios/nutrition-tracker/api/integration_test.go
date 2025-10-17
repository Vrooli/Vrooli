// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestFullMealLifecycle tests creating, reading, updating, and deleting a meal
func TestFullMealLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	userID := "lifecycle-test-user"
	today := time.Now().Format("2006-01-02")

	t.Run("Create-Read-Update-Delete", func(t *testing.T) {
		// 1. Create a meal
		meal := Meal{
			UserID:          userID,
			MealType:        "breakfast",
			MealDate:        today,
			FoodDescription: "Scrambled Eggs",
			Calories:        300,
			Protein:         20,
			Carbs:           10,
			Fat:             20,
		}

		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		createRecorder, err := makeHTTPRequest(createReq, createMeal)
		if err != nil {
			t.Fatalf("Failed to create meal: %v", err)
		}

		var createdMeal Meal
		assertJSONResponse(t, createRecorder, http.StatusCreated, &createdMeal)
		mealID := createdMeal.ID

		if mealID == "" {
			t.Fatal("Created meal should have an ID")
		}

		// 2. Read the meal
		getReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
		}

		getRecorder, err := makeHTTPRequest(getReq, getMeal)
		if err != nil {
			t.Fatalf("Failed to get meal: %v", err)
		}

		var retrievedMeal Meal
		assertJSONResponse(t, getRecorder, http.StatusOK, &retrievedMeal)

		if retrievedMeal.ID != mealID {
			t.Errorf("Retrieved meal ID mismatch: expected %s, got %s", mealID, retrievedMeal.ID)
		}

		// 3. Update the meal
		updatedMeal := Meal{
			MealType: "lunch",
		}

		updateReq := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
			Body:    updatedMeal,
		}

		updateRecorder, err := makeHTTPRequest(updateReq, updateMeal)
		if err != nil {
			t.Fatalf("Failed to update meal: %v", err)
		}

		if updateRecorder.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 after update, got %d", updateRecorder.Code)
		}

		// 4. Delete the meal
		deleteReq := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
		}

		deleteRecorder, err := makeHTTPRequest(deleteReq, deleteMeal)
		if err != nil {
			t.Fatalf("Failed to delete meal: %v", err)
		}

		if deleteRecorder.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 after delete, got %d", deleteRecorder.Code)
		}

		// 5. Verify deletion
		verifyReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/meals/" + mealID,
			URLVars: map[string]string{"id": mealID},
		}

		verifyRecorder, err := makeHTTPRequest(verifyReq, getMeal)
		if err != nil {
			t.Fatalf("Failed to verify deletion: %v", err)
		}

		if verifyRecorder.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for deleted meal, got %d", verifyRecorder.Code)
		}
	})
}

// TestFullFoodLifecycle tests creating and reading foods
func TestFullFoodLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Create-Search-List", func(t *testing.T) {
		// 1. Create a food
		food := Food{
			Name:         "Integration Test Food",
			Brand:        "Test Brand",
			ServingSize:  "100",
			Calories:     200,
			Protein:      10,
			Carbs:        25,
			Fat:          8,
			Fiber:        5,
			Sugar:        3,
			Sodium:       150,
			FoodCategory: "test-category",
		}

		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/foods",
			Body:   food,
		}

		createRecorder, err := makeHTTPRequest(createReq, createFood)
		if err != nil {
			t.Fatalf("Failed to create food: %v", err)
		}

		var createdFood Food
		assertJSONResponse(t, createRecorder, http.StatusCreated, &createdFood)

		if createdFood.ID == "" {
			t.Fatal("Created food should have an ID")
		}

		// 2. Search for the food
		searchReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods/search",
			Query:  map[string]string{"q": "Integration"},
		}

		searchRecorder, err := makeHTTPRequest(searchReq, searchFoods)
		if err != nil {
			t.Fatalf("Failed to search foods: %v", err)
		}

		var searchResults []Food
		assertJSONResponse(t, searchRecorder, http.StatusOK, &searchResults)

		found := false
		for _, f := range searchResults {
			if f.ID == createdFood.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created food should appear in search results")
		}

		// 3. List all foods
		listReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foods",
		}

		listRecorder, err := makeHTTPRequest(listReq, getFoods)
		if err != nil {
			t.Fatalf("Failed to list foods: %v", err)
		}

		var allFoods []Food
		assertJSONResponse(t, listRecorder, http.StatusOK, &allFoods)

		if len(allFoods) == 0 {
			t.Error("Food list should not be empty")
		}
	})
}

// TestNutritionGoalsWorkflow tests the complete goals workflow
func TestNutritionGoalsWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	userID := "goals-workflow-user"

	t.Run("Set-Get-Update", func(t *testing.T) {
		// 1. Set initial goals
		initialGoals := NutritionGoals{
			UserID:        userID,
			DailyCalories: 1900,
			ProteinGrams:  70,
			CarbsGrams:    210,
			FatGrams:      60,
			FiberGrams:    25,
			SugarLimit:    45,
			SodiumLimit:   2100,
			WeightGoal:    "lose",
			ActivityLevel: "moderate",
		}

		setReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/goals",
			Body:   initialGoals,
		}

		setRecorder, err := makeHTTPRequest(setReq, updateGoals)
		if err != nil {
			t.Fatalf("Failed to set goals: %v", err)
		}

		var setGoals NutritionGoals
		assertJSONResponse(t, setRecorder, http.StatusOK, &setGoals)

		if setGoals.DailyCalories != 1900 {
			t.Errorf("Expected daily_calories 1900, got %d", setGoals.DailyCalories)
		}

		// 2. Get goals
		getReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/goals",
			Query:  map[string]string{"user_id": userID},
		}

		getRecorder, err := makeHTTPRequest(getReq, getGoals)
		if err != nil {
			t.Fatalf("Failed to get goals: %v", err)
		}

		var retrievedGoals NutritionGoals
		assertJSONResponse(t, getRecorder, http.StatusOK, &retrievedGoals)

		if retrievedGoals.DailyCalories != 1900 {
			t.Errorf("Retrieved goals mismatch: expected 1900 calories, got %d", retrievedGoals.DailyCalories)
		}

		// 3. Update goals
		updatedGoals := NutritionGoals{
			UserID:        userID,
			DailyCalories: 2100,
			ProteinGrams:  85,
			CarbsGrams:    240,
			FatGrams:      70,
			FiberGrams:    30,
			SugarLimit:    50,
			SodiumLimit:   2300,
			WeightGoal:    "gain",
			ActivityLevel: "very_active",
		}

		updateReq := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/goals",
			Body:   updatedGoals,
		}

		updateRecorder, err := makeHTTPRequest(updateReq, updateGoals)
		if err != nil {
			t.Fatalf("Failed to update goals: %v", err)
		}

		var finalGoals NutritionGoals
		assertJSONResponse(t, updateRecorder, http.StatusOK, &finalGoals)

		if finalGoals.DailyCalories != 2100 {
			t.Errorf("Updated goals mismatch: expected 2100 calories, got %d", finalGoals.DailyCalories)
		}

		if finalGoals.WeightGoal != "gain" {
			t.Errorf("Updated weight goal mismatch: expected 'gain', got '%s'", finalGoals.WeightGoal)
		}
	})
}

// TestDailyNutritionTracking tests tracking nutrition over a day
func TestDailyNutritionTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	userID := "daily-tracking-user"
	today := time.Now().Format("2006-01-02")

	t.Run("TrackMultipleMeals", func(t *testing.T) {
		// Create breakfast
		breakfast := Meal{
			UserID:          userID,
			MealType:        "breakfast",
			MealDate:        today,
			FoodDescription: "Oatmeal",
			Calories:        300,
			Protein:         10,
			Carbs:           55,
			Fat:             5,
		}

		breakfastReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   breakfast,
		}

		_, err := makeHTTPRequest(breakfastReq, createMeal)
		if err != nil {
			t.Fatalf("Failed to create breakfast: %v", err)
		}

		// Create lunch
		lunch := Meal{
			UserID:          userID,
			MealType:        "lunch",
			MealDate:        today,
			FoodDescription: "Chicken Salad",
			Calories:        450,
			Protein:         35,
			Carbs:           30,
			Fat:             18,
		}

		lunchReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   lunch,
		}

		_, err = makeHTTPRequest(lunchReq, createMeal)
		if err != nil {
			t.Fatalf("Failed to create lunch: %v", err)
		}

		// Create dinner
		dinner := Meal{
			UserID:          userID,
			MealType:        "dinner",
			MealDate:        today,
			FoodDescription: "Salmon and Vegetables",
			Calories:        550,
			Protein:         40,
			Carbs:           35,
			Fat:             25,
		}

		dinnerReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   dinner,
		}

		_, err = makeHTTPRequest(dinnerReq, createMeal)
		if err != nil {
			t.Fatalf("Failed to create dinner: %v", err)
		}

		// Get today's meals
		todayReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/meals/today",
			Query:  map[string]string{"user_id": userID},
		}

		todayRecorder, err := makeHTTPRequest(todayReq, getTodaysMeals)
		if err != nil {
			t.Fatalf("Failed to get today's meals: %v", err)
		}

		var todaysMeals []map[string]interface{}
		assertJSONResponse(t, todayRecorder, http.StatusOK, &todaysMeals)

		if len(todaysMeals) < 3 {
			t.Errorf("Expected at least 3 meals for today, got %d", len(todaysMeals))
		}

		// Get nutrition summary
		summaryReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nutrition",
			Query:  map[string]string{"user_id": userID, "date": today},
		}

		summaryRecorder, err := makeHTTPRequest(summaryReq, getNutritionSummary)
		if err != nil {
			t.Fatalf("Failed to get nutrition summary: %v", err)
		}

		var summary DailySummary
		assertJSONResponse(t, summaryRecorder, http.StatusOK, &summary)

		expectedCalories := 300.0 + 450.0 + 550.0
		if summary.TotalCalories < expectedCalories-10 || summary.TotalCalories > expectedCalories+10 {
			t.Errorf("Expected total calories around %.2f, got %.2f", expectedCalories, summary.TotalCalories)
		}

		if summary.MealCount < 3 {
			t.Errorf("Expected at least 3 meals in summary, got %d", summary.MealCount)
		}
	})
}

// TestMultipleUsersIsolation tests that user data is properly isolated
func TestMultipleUsersIsolation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	today := time.Now().Format("2006-01-02")

	t.Run("IsolatedUserData", func(t *testing.T) {
		// Create meals for user1
		user1Meal := Meal{
			UserID:          "user-1",
			MealType:        "breakfast",
			MealDate:        today,
			FoodDescription: "User 1 Breakfast",
			Calories:        400,
		}

		user1Req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   user1Meal,
		}

		_, err := makeHTTPRequest(user1Req, createMeal)
		if err != nil {
			t.Fatalf("Failed to create user1 meal: %v", err)
		}

		// Create meals for user2
		user2Meal := Meal{
			UserID:          "user-2",
			MealType:        "breakfast",
			MealDate:        today,
			FoodDescription: "User 2 Breakfast",
			Calories:        350,
		}

		user2Req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   user2Meal,
		}

		_, err = makeHTTPRequest(user2Req, createMeal)
		if err != nil {
			t.Fatalf("Failed to create user2 meal: %v", err)
		}

		// Get user1 summary
		user1SummaryReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nutrition",
			Query:  map[string]string{"user_id": "user-1", "date": today},
		}

		user1Recorder, err := makeHTTPRequest(user1SummaryReq, getNutritionSummary)
		if err != nil {
			t.Fatalf("Failed to get user1 summary: %v", err)
		}

		var user1Summary DailySummary
		json.Unmarshal(user1Recorder.Body.Bytes(), &user1Summary)

		// Get user2 summary
		user2SummaryReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nutrition",
			Query:  map[string]string{"user_id": "user-2", "date": today},
		}

		user2Recorder, err := makeHTTPRequest(user2SummaryReq, getNutritionSummary)
		if err != nil {
			t.Fatalf("Failed to get user2 summary: %v", err)
		}

		var user2Summary DailySummary
		json.Unmarshal(user2Recorder.Body.Bytes(), &user2Summary)

		// Verify isolation
		if user1Summary.TotalCalories < 390 || user1Summary.TotalCalories > 410 {
			t.Errorf("User1 should have ~400 calories, got %.2f", user1Summary.TotalCalories)
		}

		if user2Summary.TotalCalories < 340 || user2Summary.TotalCalories > 360 {
			t.Errorf("User2 should have ~350 calories, got %.2f", user2Summary.TotalCalories)
		}
	})
}
