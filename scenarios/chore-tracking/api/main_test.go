package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	w := testHandlerWithRequest(t, healthCheck, req)

	response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "chore-tracking-api",
	})

	if response != nil {
		if _, exists := response["timestamp"]; !exists {
			t.Error("Expected timestamp in health check response")
		}
	}
}

// TestGetChores tests the chore listing endpoint
func TestGetChores(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	// Create test chores
	chore1 := setupTestChore(t, env, "TestChore1", 10)
	defer chore1.Cleanup()

	chore2 := setupTestChore(t, env, "TestChore2", 20)
	defer chore2.Cleanup()

	t.Run("Success_AllChores", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/chores",
		}

		w := testHandlerWithRequest(t, getChores, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected array response")
		}

		if len(array) < 2 {
			t.Errorf("Expected at least 2 chores, got %d", len(array))
		}
	})

	t.Run("Success_FilterByStatus", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/chores",
			QueryParams: map[string]string{"status": "pending"},
		}

		w := testHandlerWithRequest(t, getChores, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array != nil {
			for _, item := range array {
				choreMap := item.(map[string]interface{})
				if status, ok := choreMap["status"].(string); ok {
					if status != "pending" {
						t.Errorf("Expected status 'pending', got '%s'", status)
					}
				}
			}
		}
	})

	t.Run("Success_FilterByUser", func(t *testing.T) {
		user := setupTestUser(t, env, "FilterTestUser")
		defer user.Cleanup()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/chores",
			QueryParams: map[string]string{"user_id": strconv.Itoa(user.User.ID)},
		}

		w := testHandlerWithRequest(t, getChores, req)
		assertJSONArray(t, w, http.StatusOK)
	})
}

// TestCreateChore tests chore creation endpoint
func TestCreateChore(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success_CreateChore", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/chores",
			Body: map[string]interface{}{
				"title":       "New Test Chore",
				"description": "This is a new test chore",
				"points":      15,
				"difficulty":  "easy",
				"category":    "cleaning",
				"frequency":   "daily",
			},
		}

		w := testHandlerWithRequest(t, createChore, req)
		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"title":  "New Test Chore",
			"points": float64(15),
			"status": "pending",
		})

		if response != nil {
			// Cleanup created chore
			if id, ok := response["id"].(float64); ok {
				env.DB.Exec("DELETE FROM chores WHERE id = $1", int(id))
			}
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/chores",
			Body:   `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, createChore, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/chores",
			Body:   "",
		}

		w := testHandlerWithRequest(t, createChore, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestCompleteChore tests chore completion endpoint
func TestCompleteChore(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	user := setupTestUser(t, env, "CompleteTestUser")
	defer user.Cleanup()

	chore := setupTestChore(t, env, "CompleteTestChore", 50)
	defer chore.Cleanup()

	t.Run("Success_CompleteChore", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/chores/%d/complete", chore.Chore.ID),
			URLVars: map[string]string{"id": strconv.Itoa(chore.Chore.ID)},
			Body:    map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, completeChore, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if points, ok := response["points_earned"].(float64); !ok || points <= 0 {
				t.Error("Expected positive points_earned in response")
			}
		}
	})

	t.Run("Error_InvalidChoreID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/chores/invalid/complete",
			URLVars: map[string]string{"id": "invalid"},
			Body:    map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, completeChore, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_MalformedBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/chores/%d/complete", chore.Chore.ID),
			URLVars: map[string]string{"id": strconv.Itoa(chore.Chore.ID)},
			Body:    `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, completeChore, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetUsers tests user listing endpoint
func TestGetUsers(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	user1 := setupTestUser(t, env, "TestUser1")
	defer user1.Cleanup()

	user2 := setupTestUser(t, env, "TestUser2")
	defer user2.Cleanup()

	t.Run("Success_GetUsers", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/users",
		}

		w := testHandlerWithRequest(t, getUsers, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected array response")
		}

		if len(array) < 2 {
			t.Errorf("Expected at least 2 users, got %d", len(array))
		}
	})
}

// TestGetUser tests individual user endpoint
func TestGetUser(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	user := setupTestUser(t, env, "IndividualTestUser")
	defer user.Cleanup()

	t.Run("Success_GetUser", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/users/%d", user.User.ID),
			URLVars: map[string]string{"id": strconv.Itoa(user.User.ID)},
		}

		w := testHandlerWithRequest(t, getUser, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   float64(user.User.ID),
			"name": user.User.Name,
		})

		if response != nil {
			if _, exists := response["total_points"]; !exists {
				t.Error("Expected total_points in user response")
			}
			if _, exists := response["current_streak"]; !exists {
				t.Error("Expected current_streak in user response")
			}
		}
	})

	t.Run("Error_InvalidUserID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/users/invalid",
			URLVars: map[string]string{"id": "invalid"},
		}

		w := testHandlerWithRequest(t, getUser, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_NonExistentUser", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/users/999999",
			URLVars: map[string]string{"id": "999999"},
		}

		w := testHandlerWithRequest(t, getUser, req)
		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

// TestGetAchievements tests achievement listing endpoint
func TestGetAchievements(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success_GetAchievements", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/achievements",
		}

		w := testHandlerWithRequest(t, getAchievements, req)
		assertJSONArray(t, w, http.StatusOK)
	})

	t.Run("Success_GetAchievementsWithFilter", func(t *testing.T) {
		user := setupTestUser(t, env, "AchievementTestUser")
		defer user.Cleanup()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/achievements",
			QueryParams: map[string]string{"user_id": strconv.Itoa(user.User.ID)},
		}

		w := testHandlerWithRequest(t, getAchievements, req)
		assertJSONArray(t, w, http.StatusOK)
	})
}

// TestGetRewards tests reward listing endpoint
func TestGetRewards(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	reward := setupTestReward(t, env, "TestReward", 100)
	defer reward.Cleanup()

	t.Run("Success_GetRewards", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/rewards",
		}

		w := testHandlerWithRequest(t, getRewards, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array != nil && len(array) < 1 {
			t.Errorf("Expected at least 1 reward, got %d", len(array))
		}
	})
}

// TestRedeemReward tests reward redemption endpoint
func TestRedeemReward(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	user := setupTestUser(t, env, "RewardTestUser")
	defer user.Cleanup()

	// Give user enough points
	env.DB.Exec("UPDATE users SET total_points = $1 WHERE id = $2", 500, user.User.ID)

	reward := setupTestReward(t, env, "AffordableReward", 100)
	defer reward.Cleanup()

	t.Run("Success_RedeemReward", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/rewards/%d/redeem", reward.Reward.ID),
			URLVars: map[string]string{"id": strconv.Itoa(reward.Reward.ID)},
			Body:    map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, redeemReward, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if remaining, ok := response["points_remaining"].(float64); !ok || remaining != 400 {
				t.Errorf("Expected points_remaining to be 400, got %v", remaining)
			}
		}
	})

	t.Run("Error_InvalidRewardID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/rewards/invalid/redeem",
			URLVars: map[string]string{"id": "invalid"},
			Body:    map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, redeemReward, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_InsufficientPoints", func(t *testing.T) {
		poorUser := setupTestUser(t, env, "PoorUser")
		defer poorUser.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/rewards/%d/redeem", reward.Reward.ID),
			URLVars: map[string]string{"id": strconv.Itoa(reward.Reward.ID)},
			Body:    map[string]int{"user_id": poorUser.User.ID},
		}

		w := testHandlerWithRequest(t, redeemReward, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_NonExistentReward", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/rewards/999999/redeem",
			URLVars: map[string]string{"id": "999999"},
			Body:    map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, redeemReward, req)
		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

// TestGenerateScheduleHandler tests schedule generation endpoint
func TestGenerateScheduleHandler(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	// Set up test environment - need choreProcessor
	choreProcessor = NewChoreProcessor(env.DB)

	user := setupTestUser(t, env, "ScheduleTestUser")
	defer user.Cleanup()

	// Create test chore with active status for schedule generation
	env.DB.Exec(`UPDATE chores SET status = 'active' WHERE id >= 9000`)

	t.Run("Success_GenerateSchedule", func(t *testing.T) {
		weekStart := time.Now().Truncate(24 * time.Hour)
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedule/generate",
			Body: map[string]interface{}{
				"user_id":    user.User.ID,
				"week_start": weekStart,
				"preferences": map[string]interface{}{
					"max_daily_chores": 3,
					"preferred_time":   "morning",
					"skip_days":        []int{0, 6}, // Skip Sunday and Saturday
				},
			},
		}

		w := testHandlerWithRequest(t, generateScheduleHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, exists := response["assignments"]; !exists {
				t.Error("Expected assignments in schedule response")
			}
		}
	})

	t.Run("Error_InvalidBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedule/generate",
			Body:   `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, generateScheduleHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestCalculatePointsHandler tests points calculation endpoint
func TestCalculatePointsHandler(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	choreProcessor = NewChoreProcessor(env.DB)

	user := setupTestUser(t, env, "PointsTestUser")
	defer user.Cleanup()

	chore := setupTestChore(t, env, "PointsTestChore", 25)
	defer chore.Cleanup()

	t.Run("Success_CalculatePoints", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/points/calculate",
			Body: map[string]int{
				"user_id":  user.User.ID,
				"chore_id": chore.Chore.ID,
			},
		}

		w := testHandlerWithRequest(t, calculatePointsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if points, ok := response["points_awarded"].(float64); !ok || points <= 0 {
				t.Error("Expected positive points_awarded in response")
			}
		}
	})

	t.Run("Error_InvalidBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/points/calculate",
			Body:   map[string]string{"invalid": "data"},
		}

		w := testHandlerWithRequest(t, calculatePointsHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestProcessAchievementsHandler tests achievement processing endpoint
func TestProcessAchievementsHandler(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	choreProcessor = NewChoreProcessor(env.DB)

	user := setupTestUser(t, env, "AchievementProcessTestUser")
	defer user.Cleanup()

	t.Run("Success_ProcessAchievements", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/achievements/process",
			Body:   map[string]int{"user_id": user.User.ID},
		}

		w := testHandlerWithRequest(t, processAchievementsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, exists := response["new_achievements"]; !exists {
				t.Error("Expected new_achievements in response")
			}
		}
	})

	t.Run("Error_InvalidBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/achievements/process",
			Body:   `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, processAchievementsHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestMain validates lifecycle management requirement
func TestMain(m *testing.M) {
	// Set lifecycle environment variable for tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	os.Exit(code)
}
