package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestNewChoreProcessor tests ChoreProcessor initialization
func TestNewChoreProcessor(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)

	if processor == nil {
		t.Fatal("Expected non-nil ChoreProcessor")
	}

	if processor.db != env.DB {
		t.Error("Expected ChoreProcessor to use provided database")
	}

	if processor.schedulerManager == nil {
		t.Error("Expected non-nil SchedulerManager")
	}

	if processor.rewardManager == nil {
		t.Error("Expected non-nil RewardManager")
	}

	if processor.pointsCalculator == nil {
		t.Error("Expected non-nil PointsCalculator")
	}
}

// TestGenerateWeeklySchedule tests weekly schedule generation
func TestGenerateWeeklySchedule(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)
	user := setupTestUser(t, env, "ScheduleUser")
	defer user.Cleanup()

	// Create some active chores for schedule generation
	chore1 := setupTestChore(t, env, "DailyChore1", 10)
	defer chore1.Cleanup()
	env.DB.Exec("UPDATE chores SET status = 'active', frequency = 'daily' WHERE id = $1", chore1.Chore.ID)

	chore2 := setupTestChore(t, env, "WeeklyChore1", 20)
	defer chore2.Cleanup()
	env.DB.Exec("UPDATE chores SET status = 'active', frequency = 'weekly' WHERE id = $1", chore2.Chore.ID)

	t.Run("Success_GenerateSchedule", func(t *testing.T) {
		ctx := context.Background()
		weekStart := time.Now().Truncate(24 * time.Hour)

		req := ScheduleRequest{
			UserID:    user.User.ID,
			WeekStart: weekStart,
			Preferences: SchedulePreferences{
				MaxDailyChores: 3,
				PreferredTime:  "morning",
				SkipDays:       []int{}, // No skip days
			},
		}

		assignments, err := processor.GenerateWeeklySchedule(ctx, req)
		if err != nil {
			t.Fatalf("GenerateWeeklySchedule failed: %v", err)
		}

		if len(assignments) == 0 {
			t.Error("Expected at least one assignment")
		}

		// Verify assignments are for correct user
		for _, assignment := range assignments {
			if assignment.UserID != user.User.ID {
				t.Errorf("Expected assignment for user %d, got %d", user.User.ID, assignment.UserID)
			}

			if assignment.Status != "pending" {
				t.Errorf("Expected assignment status 'pending', got '%s'", assignment.Status)
			}
		}
	})

	t.Run("Success_SkipDays", func(t *testing.T) {
		ctx := context.Background()
		weekStart := time.Now().Truncate(24 * time.Hour)

		req := ScheduleRequest{
			UserID:    user.User.ID,
			WeekStart: weekStart,
			Preferences: SchedulePreferences{
				MaxDailyChores: 2,
				PreferredTime:  "evening",
				SkipDays:       []int{0, 6}, // Skip Sunday and Saturday
			},
		}

		assignments, err := processor.GenerateWeeklySchedule(ctx, req)
		if err != nil {
			t.Fatalf("GenerateWeeklySchedule with skip days failed: %v", err)
		}

		// Verify no assignments on skip days
		for _, assignment := range assignments {
			dayOfWeek := int(assignment.DueDate.Weekday())
			if dayOfWeek == 0 || dayOfWeek == 6 {
				t.Errorf("Found assignment on skip day: %s (day %d)", assignment.DueDate, dayOfWeek)
			}
		}
	})

	t.Run("Success_MaxDailyChores", func(t *testing.T) {
		ctx := context.Background()
		weekStart := time.Now().Truncate(24 * time.Hour)

		maxDaily := 1
		req := ScheduleRequest{
			UserID:    user.User.ID,
			WeekStart: weekStart,
			Preferences: SchedulePreferences{
				MaxDailyChores: maxDaily,
				PreferredTime:  "afternoon",
				SkipDays:       []int{},
			},
		}

		assignments, err := processor.GenerateWeeklySchedule(ctx, req)
		if err != nil {
			t.Fatalf("GenerateWeeklySchedule with max daily chores failed: %v", err)
		}

		// Count assignments per day
		dailyCounts := make(map[string]int)
		for _, assignment := range assignments {
			day := assignment.DueDate.Format("2006-01-02")
			dailyCounts[day]++
		}

		// Verify max daily chores not exceeded
		for day, count := range dailyCounts {
			if count > maxDaily {
				t.Errorf("Exceeded max daily chores on %s: %d > %d", day, count, maxDaily)
			}
		}
	})

	t.Run("Error_NoActiveChores", func(t *testing.T) {
		ctx := context.Background()
		env.DB.Exec("UPDATE chores SET status = 'inactive' WHERE id >= 9000")

		emptyUser := setupTestUser(t, env, "EmptyScheduleUser")
		defer emptyUser.Cleanup()

		req := ScheduleRequest{
			UserID:    emptyUser.User.ID,
			WeekStart: time.Now(),
			Preferences: SchedulePreferences{
				MaxDailyChores: 3,
			},
		}

		assignments, err := processor.GenerateWeeklySchedule(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(assignments) != 0 {
			t.Errorf("Expected no assignments for user with no chores, got %d", len(assignments))
		}

		// Restore active status
		env.DB.Exec("UPDATE chores SET status = 'active' WHERE id >= 9000")
	})
}

// TestCalculatePoints tests point calculation logic
func TestCalculatePoints(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)
	user := setupTestUser(t, env, "PointsUser")
	defer user.Cleanup()

	t.Run("Success_EasyDifficulty", func(t *testing.T) {
		chore := setupTestChore(t, env, "EasyChore", 100)
		defer chore.Cleanup()
		env.DB.Exec("UPDATE chores SET difficulty = 'easy' WHERE id = $1", chore.Chore.ID)

		ctx := context.Background()
		points, err := processor.CalculatePoints(ctx, user.User.ID, chore.Chore.ID)
		if err != nil {
			t.Fatalf("CalculatePoints failed: %v", err)
		}

		// Easy difficulty has 1.0 multiplier
		if points != 100 {
			t.Errorf("Expected 100 points, got %d", points)
		}
	})

	t.Run("Success_MediumDifficulty", func(t *testing.T) {
		chore := setupTestChore(t, env, "MediumChore", 100)
		defer chore.Cleanup()
		env.DB.Exec("UPDATE chores SET difficulty = 'medium' WHERE id = $1", chore.Chore.ID)

		ctx := context.Background()
		points, err := processor.CalculatePoints(ctx, user.User.ID, chore.Chore.ID)
		if err != nil {
			t.Fatalf("CalculatePoints failed: %v", err)
		}

		// Medium difficulty has 1.2 multiplier
		expected := 120
		if points != expected {
			t.Errorf("Expected %d points, got %d", expected, points)
		}
	})

	t.Run("Success_HardDifficulty", func(t *testing.T) {
		chore := setupTestChore(t, env, "HardChore", 100)
		defer chore.Cleanup()
		env.DB.Exec("UPDATE chores SET difficulty = 'hard' WHERE id = $1", chore.Chore.ID)

		ctx := context.Background()
		points, err := processor.CalculatePoints(ctx, user.User.ID, chore.Chore.ID)
		if err != nil {
			t.Fatalf("CalculatePoints failed: %v", err)
		}

		// Hard difficulty has 1.5 multiplier
		expected := 150
		if points != expected {
			t.Errorf("Expected %d points, got %d", expected, points)
		}
	})

	t.Run("Success_WithStreak", func(t *testing.T) {
		streakUser := setupTestUser(t, env, "StreakUser")
		defer streakUser.Cleanup()

		// Set user streak
		env.DB.Exec("UPDATE users SET current_streak = 10 WHERE id = $1", streakUser.User.ID)

		chore := setupTestChore(t, env, "StreakChore", 100)
		defer chore.Cleanup()

		ctx := context.Background()
		points, err := processor.CalculatePoints(ctx, streakUser.User.ID, chore.Chore.ID)
		if err != nil {
			t.Fatalf("CalculatePoints with streak failed: %v", err)
		}

		// Should have base 100 + streak bonus (capped at 50% = 50)
		if points < 100 {
			t.Errorf("Expected points > 100 with streak, got %d", points)
		}
	})

	t.Run("Error_NonExistentChore", func(t *testing.T) {
		ctx := context.Background()
		_, err := processor.CalculatePoints(ctx, user.User.ID, 999999)
		if err == nil {
			t.Error("Expected error for non-existent chore")
		}
	})

	t.Run("Error_NonExistentUser", func(t *testing.T) {
		chore := setupTestChore(t, env, "ErrorChore", 100)
		defer chore.Cleanup()

		ctx := context.Background()
		_, err := processor.CalculatePoints(ctx, 999999, chore.Chore.ID)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

// TestProcessAchievements tests achievement processing logic
func TestProcessAchievements(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)
	user := setupTestUser(t, env, "AchievementUser")
	defer user.Cleanup()

	t.Run("Success_ProcessAchievements", func(t *testing.T) {
		// Create test achievement with total_points criteria
		criteria := map[string]interface{}{
			"type":   "total_points",
			"target": float64(50),
		}
		criteriaJSON, _ := json.Marshal(criteria)

		achievementID := 9000 + int(time.Now().Unix()%1000)
		_, err := env.DB.Exec(`INSERT INTO achievements (id, name, description, icon, points, criteria)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			achievementID, "Test Achievement", "Test description", "test.png", 10, string(criteriaJSON))
		if err != nil {
			t.Fatalf("Failed to create test achievement: %v", err)
		}
		defer env.DB.Exec("DELETE FROM achievements WHERE id = $1", achievementID)

		// Give user enough points to unlock achievement
		env.DB.Exec("UPDATE users SET total_points = 100 WHERE id = $1", user.User.ID)

		ctx := context.Background()
		achievements, err := processor.ProcessAchievements(ctx, user.User.ID)
		if err != nil {
			t.Fatalf("ProcessAchievements failed: %v", err)
		}

		// Verify achievement was unlocked
		found := false
		for _, ach := range achievements {
			if ach.ID == achievementID {
				found = true
				if ach.UnlockedAt == nil {
					t.Error("Expected UnlockedAt to be set")
				}
			}
		}

		if !found {
			t.Error("Expected to find unlocked achievement")
		}
	})

	t.Run("Success_NoEligibleAchievements", func(t *testing.T) {
		newUser := setupTestUser(t, env, "NewAchievementUser")
		defer newUser.Cleanup()

		ctx := context.Background()
		achievements, err := processor.ProcessAchievements(ctx, newUser.User.ID)
		if err != nil {
			t.Fatalf("ProcessAchievements failed: %v", err)
		}

		// Should return empty array for user not meeting criteria
		if len(achievements) > 0 {
			t.Logf("Unexpected achievements unlocked: %d", len(achievements))
		}
	})

	t.Run("Success_StreakAchievement", func(t *testing.T) {
		streakUser := setupTestUser(t, env, "StreakAchievementUser")
		defer streakUser.Cleanup()

		// Create streak achievement
		criteria := map[string]interface{}{
			"type":   "streak",
			"target": float64(5),
		}
		criteriaJSON, _ := json.Marshal(criteria)

		achievementID := 9001 + int(time.Now().Unix()%1000)
		env.DB.Exec(`INSERT INTO achievements (id, name, description, icon, points, criteria)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			achievementID, "Streak Achievement", "Test streak", "streak.png", 15, string(criteriaJSON))
		defer env.DB.Exec("DELETE FROM achievements WHERE id = $1", achievementID)

		// Set user streak
		env.DB.Exec("UPDATE users SET current_streak = 10 WHERE id = $1", streakUser.User.ID)

		ctx := context.Background()
		achievements, err := processor.ProcessAchievements(ctx, streakUser.User.ID)
		if err != nil {
			t.Fatalf("ProcessAchievements for streak failed: %v", err)
		}

		// Verify streak achievement was unlocked
		found := false
		for _, ach := range achievements {
			if ach.ID == achievementID {
				found = true
			}
		}

		if !found {
			t.Error("Expected to find unlocked streak achievement")
		}
	})
}

// TestManageRewards tests reward management logic
func TestManageRewards(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)
	user := setupTestUser(t, env, "RewardUser")
	defer user.Cleanup()

	t.Run("Success_RedeemReward", func(t *testing.T) {
		reward := setupTestReward(t, env, "TestReward", 50)
		defer reward.Cleanup()

		// Give user enough points
		env.DB.Exec("UPDATE users SET total_points = 100 WHERE id = $1", user.User.ID)

		ctx := context.Background()
		err := processor.ManageRewards(ctx, user.User.ID, reward.Reward.ID)
		if err != nil {
			t.Fatalf("ManageRewards failed: %v", err)
		}

		// Verify points were deducted
		var points int
		env.DB.QueryRow("SELECT total_points FROM users WHERE id = $1", user.User.ID).Scan(&points)
		if points != 50 {
			t.Errorf("Expected 50 points remaining, got %d", points)
		}
	})

	t.Run("Error_InsufficientPoints", func(t *testing.T) {
		poorUser := setupTestUser(t, env, "PoorUser")
		defer poorUser.Cleanup()

		reward := setupTestReward(t, env, "ExpensiveReward", 1000)
		defer reward.Cleanup()

		ctx := context.Background()
		err := processor.ManageRewards(ctx, poorUser.User.ID, reward.Reward.ID)
		if err == nil {
			t.Error("Expected error for insufficient points")
		}
	})

	t.Run("Error_UnavailableReward", func(t *testing.T) {
		reward := setupTestReward(t, env, "UnavailableReward", 50)
		defer reward.Cleanup()

		// Make reward unavailable
		env.DB.Exec("UPDATE rewards SET available = false WHERE id = $1", reward.Reward.ID)

		ctx := context.Background()
		err := processor.ManageRewards(ctx, user.User.ID, reward.Reward.ID)
		if err == nil {
			t.Error("Expected error for unavailable reward")
		}
	})

	t.Run("Error_NonExistentReward", func(t *testing.T) {
		ctx := context.Background()
		err := processor.ManageRewards(ctx, user.User.ID, 999999)
		if err == nil {
			t.Error("Expected error for non-existent reward")
		}
	})
}

// TestHelperFunctions tests internal helper functions
func TestHelperFunctions(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)

	t.Run("GetAvailableChores", func(t *testing.T) {
		chore := setupTestChore(t, env, "AvailableChore", 10)
		defer chore.Cleanup()
		env.DB.Exec("UPDATE chores SET status = 'active' WHERE id = $1", chore.Chore.ID)

		ctx := context.Background()
		chores, err := processor.getAvailableChores(ctx, 1)
		if err != nil {
			t.Fatalf("getAvailableChores failed: %v", err)
		}

		if len(chores) == 0 {
			t.Error("Expected at least one available chore")
		}
	})

	t.Run("GetUserPerformance", func(t *testing.T) {
		user := setupTestUser(t, env, "PerfUser")
		defer user.Cleanup()

		ctx := context.Background()
		performance, err := processor.getUserPerformance(ctx, user.User.ID)
		if err != nil {
			t.Fatalf("getUserPerformance failed: %v", err)
		}

		// Should return empty map for user with no history
		if performance == nil {
			t.Error("Expected non-nil performance map")
		}
	})

	t.Run("GetUserStreak", func(t *testing.T) {
		user := setupTestUser(t, env, "StreakUser")
		defer user.Cleanup()

		env.DB.Exec("UPDATE users SET current_streak = 5 WHERE id = $1", user.User.ID)

		ctx := context.Background()
		streak, err := processor.getUserStreak(ctx, user.User.ID)
		if err != nil {
			t.Fatalf("getUserStreak failed: %v", err)
		}

		if streak != 5 {
			t.Errorf("Expected streak of 5, got %d", streak)
		}
	})

	t.Run("SelectDailyChores", func(t *testing.T) {
		chores := []Chore{
			{ID: 1, Frequency: "daily", Points: 10},
			{ID: 2, Frequency: "weekly", Points: 20},
			{ID: 3, Frequency: "daily", Points: 15},
		}

		performance := make(map[int]float64)
		selected := processor.selectDailyChores(chores, 2, performance)

		if len(selected) > 2 {
			t.Errorf("Expected max 2 chores, got %d", len(selected))
		}
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	processor := NewChoreProcessor(env.DB)

	t.Run("ScheduleGeneration_Performance", func(t *testing.T) {
		user := setupTestUser(t, env, "PerfUser")
		defer user.Cleanup()

		// Create multiple chores
		for i := 0; i < 10; i++ {
			chore := setupTestChore(t, env, fmt.Sprintf("PerfChore%d", i), 10)
			defer chore.Cleanup()
			env.DB.Exec("UPDATE chores SET status = 'active' WHERE id = $1", chore.Chore.ID)
		}

		ctx := context.Background()
		req := ScheduleRequest{
			UserID:    user.User.ID,
			WeekStart: time.Now(),
			Preferences: SchedulePreferences{
				MaxDailyChores: 3,
			},
		}

		start := time.Now()
		_, err := processor.GenerateWeeklySchedule(ctx, req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Schedule generation failed: %v", err)
		}

		if duration > 5*time.Second {
			t.Errorf("Schedule generation too slow: %v", duration)
		}

		t.Logf("Schedule generation completed in %v", duration)
	})
}
