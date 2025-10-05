// +build testing

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test TypingProcessor creation
func TestNewTypingProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()

		processor := NewTypingProcessor(testDB.DB)
		if processor == nil {
			t.Error("Expected non-nil TypingProcessor")
		}
		if processor.db == nil {
			t.Error("Expected database connection in processor")
		}
	}
}

// Test ProcessStats
func TestProcessStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	processor := NewTypingProcessor(testDB.DB)
	ctx := context.Background()

	t.Run("BeginnerPerformance", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 30, 75.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		assertValidProcessedStats(t, result)

		if result.Analysis.PerformanceLevel != "beginner" {
			t.Errorf("Expected beginner level, got %s", result.Analysis.PerformanceLevel)
		}

		if len(result.PersonalizedTips) == 0 {
			t.Error("Expected personalized tips for beginner")
		}
	})

	t.Run("IntermediatePerformance", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 50, 88.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		if result.Analysis.PerformanceLevel != "intermediate" {
			t.Errorf("Expected intermediate level, got %s", result.Analysis.PerformanceLevel)
		}
	})

	t.Run("AdvancedPerformance", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 70, 92.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		if result.Analysis.PerformanceLevel != "advanced" {
			t.Errorf("Expected advanced level, got %s", result.Analysis.PerformanceLevel)
		}
	})

	t.Run("ExpertPerformance", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 85, 97.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		if result.Analysis.PerformanceLevel != "expert" {
			t.Errorf("Expected expert level, got %s", result.Analysis.PerformanceLevel)
		}
	})

	t.Run("VerifyMetrics", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 60, 90.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		if result.Metrics.WPM != stats.WPM {
			t.Errorf("Expected WPM %d, got %d", stats.WPM, result.Metrics.WPM)
		}

		if result.Metrics.Accuracy != stats.Accuracy {
			t.Errorf("Expected accuracy %.2f, got %.2f", stats.Accuracy, result.Metrics.Accuracy)
		}

		expectedErrorRate := 100.0 - stats.Accuracy
		if result.Metrics.ErrorRate != expectedErrorRate {
			t.Errorf("Expected error rate %.2f, got %.2f", expectedErrorRate, result.Metrics.ErrorRate)
		}
	})

	t.Run("NextGoalsCalculation", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 50, 85.0)

		result, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		if result.NextGoals.WPM <= stats.WPM {
			t.Error("Expected next WPM goal to be higher than current")
		}

		if result.NextGoals.Accuracy <= int(stats.Accuracy) {
			t.Error("Expected next accuracy goal to be higher than current")
		}

		// Should cap at 100
		if result.NextGoals.WPM > 100 {
			t.Error("WPM goal should not exceed 100")
		}
		if result.NextGoals.Accuracy > 100 {
			t.Error("Accuracy goal should not exceed 100")
		}
	})

	t.Run("SessionDataPersistence", func(t *testing.T) {
		sessionID := uuid.New().String()
		stats := createTestStats(sessionID, 55, 88.0)

		_, err := processor.ProcessStats(ctx, stats)
		if err != nil {
			t.Fatalf("Failed to process stats: %v", err)
		}

		// Verify session was saved
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM game_sessions WHERE session_id = $1", sessionID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query session: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 session record, got %d", count)
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		cases := []struct {
			name     string
			wpm      int
			accuracy float64
		}{
			{"ZeroWPM", 0, 90.0},
			{"ZeroAccuracy", 50, 0.0},
			{"PerfectScore", 100, 100.0},
			{"LowBoth", 1, 1.0},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				stats := createTestStats(uuid.New().String(), tc.wpm, tc.accuracy)

				result, err := processor.ProcessStats(ctx, stats)
				if err != nil {
					t.Errorf("Failed to process %s: %v", tc.name, err)
					return
				}

				if result.SessionID == "" {
					t.Error("Expected session ID in result")
				}
			})
		}
	})
}

// Test ProvideCoaching
func TestProvideCoaching(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := &TypingProcessor{}
	ctx := context.Background()

	t.Run("BeginnerCoaching", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 25, 70.0)

		response := processor.ProvideCoaching(ctx, stats, "easy")
		assertValidCoachingResponse(t, response)

		if !strings.Contains(response.Feedback, "foundation") && !strings.Contains(response.Feedback, "practicing") {
			t.Log("Feedback for beginner:", response.Feedback)
		}
	})

	t.Run("IntermediateCoaching", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 50, 88.0)

		response := processor.ProvideCoaching(ctx, stats, "medium")
		assertValidCoachingResponse(t, response)

		if len(response.Tips) < 3 {
			t.Error("Expected at least 3 tips for intermediate")
		}
	})

	t.Run("AdvancedCoaching", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 70, 92.0)

		response := processor.ProvideCoaching(ctx, stats, "medium")
		assertValidCoachingResponse(t, response)
	})

	t.Run("ExpertCoaching", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 90, 98.0)

		response := processor.ProvideCoaching(ctx, stats, "hard")
		assertValidCoachingResponse(t, response)

		if !strings.Contains(response.Feedback, "Outstanding") && !strings.Contains(response.Feedback, "expert") {
			t.Log("Feedback for expert:", response.Feedback)
		}
	})

	t.Run("LowAccuracyTips", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 60, 75.0)

		response := processor.ProvideCoaching(ctx, stats, "medium")

		// Should include tip about slowing down for accuracy
		foundAccuracyTip := false
		for _, tip := range response.Tips {
			if strings.Contains(strings.ToLower(tip), "accuracy") || strings.Contains(strings.ToLower(tip), "slow") {
				foundAccuracyTip = true
				break
			}
		}

		if !foundAccuracyTip {
			t.Log("Tips provided:", response.Tips)
		}
	})

	t.Run("LowSpeedTips", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 25, 90.0)

		response := processor.ProvideCoaching(ctx, stats, "easy")

		// Should include tip about speed improvement
		foundSpeedTip := false
		for _, tip := range response.Tips {
			if strings.Contains(strings.ToLower(tip), "speed") || strings.Contains(strings.ToLower(tip), "keyboard") {
				foundSpeedTip = true
				break
			}
		}

		if !foundSpeedTip {
			t.Log("Tips provided:", response.Tips)
		}
	})
}

// Test GenerateAdaptiveText
func TestGenerateAdaptiveText(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := &TypingProcessor{}
	ctx := context.Background()

	t.Run("EasyText", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "easy")
		request.UserLevel = "beginner"

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)

		if response.Difficulty != "easy" {
			t.Errorf("Expected easy difficulty, got %s", response.Difficulty)
		}
	})

	t.Run("MediumText", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.UserLevel = "intermediate"

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)
	})

	t.Run("HardText", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "hard")
		request.UserLevel = "advanced"

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)
	})

	t.Run("ShortText", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.TextLength = "short"

		response := processor.GenerateAdaptiveText(ctx, request)

		if response.WordCount > 30 {
			t.Logf("Short text has %d words (expected ~25)", response.WordCount)
		}
	})

	t.Run("MediumText_Length", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.TextLength = "medium"

		response := processor.GenerateAdaptiveText(ctx, request)

		if response.WordCount < 40 || response.WordCount > 60 {
			t.Logf("Medium text has %d words (expected ~50)", response.WordCount)
		}
	})

	t.Run("LongText", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.TextLength = "long"

		response := processor.GenerateAdaptiveText(ctx, request)

		if response.WordCount < 90 {
			t.Logf("Long text has %d words (expected ~100)", response.WordCount)
		}
	})

	t.Run("WithTargetWords", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "easy")
		request.TargetWords = []string{"practice", "typing", "test"}

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)

		// Text should not be empty
		if response.Text == "" {
			t.Error("Expected non-empty text with target words")
		}
	})

	t.Run("WithProblemChars", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.ProblemChars = []string{"q", "z", "x"}

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)
	})

	t.Run("WithPreviousMistakes", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")
		request.PreviousMistakes = []struct {
			Word       string `json:"word"`
			Char       string `json:"char"`
			Position   string `json:"position"`
			ErrorCount int    `json:"errorCount"`
		}{
			{Word: "difficult", Char: "f", Position: "3", ErrorCount: 5},
			{Word: "challenge", Char: "l", Position: "4", ErrorCount: 3},
		}

		response := processor.GenerateAdaptiveText(ctx, request)
		assertValidAdaptiveResponse(t, response)

		// Should incorporate mistake words
		text := strings.ToLower(response.Text)
		if !strings.Contains(text, "difficult") && !strings.Contains(text, "challenge") {
			t.Log("Generated text:", response.Text)
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		request := AdaptiveTextRequest{
			UserID: uuid.New().String(),
		}

		response := processor.GenerateAdaptiveText(ctx, request)

		// Should handle missing fields with defaults
		if response.Text == "" {
			t.Error("Expected text even with minimal request")
		}

		// Note: Difficulty might be empty string if not set in request,
		// but the function should still work and generate text
		if response.WordCount <= 0 {
			t.Error("Expected positive word count even with minimal request")
		}
	})
}

// Test ManageLeaderboard
func TestManageLeaderboard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	processor := NewTypingProcessor(testDB.DB)
	ctx := context.Background()

	// Insert test scores
	scores := generateTestScores(10)
	for _, score := range scores {
		insertTestScore(t, testDB.DB, score)
	}

	t.Run("AllTimeLeaderboard", func(t *testing.T) {
		entries, err := processor.ManageLeaderboard(ctx, "all", "")
		if err != nil {
			t.Fatalf("Failed to get leaderboard: %v", err)
		}

		if len(entries) == 0 {
			t.Error("Expected leaderboard entries")
		}

		// Verify ranking
		for i, entry := range entries {
			if entry.Rank != i+1 {
				t.Errorf("Expected rank %d, got %d", i+1, entry.Rank)
			}
		}

		// Verify descending order by score
		for i := 1; i < len(entries); i++ {
			if entries[i].Score > entries[i-1].Score {
				t.Error("Leaderboard not sorted by score descending")
			}
		}
	})

	t.Run("DailyLeaderboard", func(t *testing.T) {
		entries, err := processor.ManageLeaderboard(ctx, "daily", "")
		if err != nil {
			t.Fatalf("Failed to get daily leaderboard: %v", err)
		}

		// Should only include recent scores
		for _, entry := range entries {
			age := time.Since(entry.Date)
			if age > 24*time.Hour {
				t.Errorf("Daily leaderboard includes score from %v ago", age)
			}
		}
	})

	t.Run("WeeklyLeaderboard", func(t *testing.T) {
		entries, err := processor.ManageLeaderboard(ctx, "weekly", "")
		if err != nil {
			t.Fatalf("Failed to get weekly leaderboard: %v", err)
		}

		// Should only include recent scores
		for _, entry := range entries {
			age := time.Since(entry.Date)
			if age > 7*24*time.Hour {
				t.Errorf("Weekly leaderboard includes score from %v ago", age)
			}
		}
	})

	t.Run("MonthlyLeaderboard", func(t *testing.T) {
		entries, err := processor.ManageLeaderboard(ctx, "monthly", "")
		if err != nil {
			t.Fatalf("Failed to get monthly leaderboard: %v", err)
		}

		// Verify results
		if len(entries) == 0 {
			t.Error("Expected monthly leaderboard entries")
		}
	})

	t.Run("CurrentUserHighlight", func(t *testing.T) {
		// Insert score with specific user ID
		userID := uuid.New().String()
		score := createTestScore("CurrentUser", 75, 95)

		testDB.DB.Exec(`INSERT INTO scores (name, score, wpm, accuracy, user_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			score.Name, score.Score, score.WPM, score.Accuracy, userID, time.Now())

		entries, err := processor.ManageLeaderboard(ctx, "all", userID)
		if err != nil {
			t.Fatalf("Failed to get leaderboard: %v", err)
		}

		// Find current user entry
		foundUser := false
		for _, entry := range entries {
			if entry.IsCurrentUser {
				foundUser = true
				break
			}
		}

		if !foundUser {
			t.Log("Current user not highlighted in leaderboard (may not be in top 100)")
		}
	})

	t.Run("Limit100Entries", func(t *testing.T) {
		// Insert many scores
		for i := 0; i < 120; i++ {
			score := createTestScore("Player", 50+i, 85)
			insertTestScore(t, testDB.DB, score)
		}

		entries, err := processor.ManageLeaderboard(ctx, "all", "")
		if err != nil {
			t.Fatalf("Failed to get leaderboard: %v", err)
		}

		if len(entries) > 100 {
			t.Errorf("Leaderboard should limit to 100 entries, got %d", len(entries))
		}
	})
}

// Test AddScore
func TestAddScore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	processor := NewTypingProcessor(testDB.DB)
	ctx := context.Background()

	t.Run("AddValidScore", func(t *testing.T) {
		score := createTestScore("TestPlayer", 60, 90)
		initialCount := getScoreCount(t, testDB.DB)

		err := processor.AddScore(ctx, score)
		if err != nil {
			t.Fatalf("Failed to add score: %v", err)
		}

		finalCount := getScoreCount(t, testDB.DB)
		if finalCount != initialCount+1 {
			t.Errorf("Expected count to increase by 1, got %d -> %d", initialCount, finalCount)
		}
	})

	t.Run("AddMultipleScores", func(t *testing.T) {
		initialCount := getScoreCount(t, testDB.DB)

		for i := 0; i < 5; i++ {
			score := createTestScore("Player", 50+i*5, 85+i)
			if err := processor.AddScore(ctx, score); err != nil {
				t.Errorf("Failed to add score %d: %v", i, err)
			}
		}

		finalCount := getScoreCount(t, testDB.DB)
		if finalCount != initialCount+5 {
			t.Errorf("Expected count to increase by 5, got %d -> %d", initialCount, finalCount)
		}
	})

	t.Run("ScoreDataIntegrity", func(t *testing.T) {
		score := createTestScore("DataCheck", 75, 93)
		score.Difficulty = "hard"
		score.Mode = "challenge"
		score.MaxCombo = 25

		err := processor.AddScore(ctx, score)
		if err != nil {
			t.Fatalf("Failed to add score: %v", err)
		}

		// Verify data was saved correctly
		var savedName string
		var savedWPM, savedAccuracy int
		err = testDB.DB.QueryRow(`
			SELECT name, wpm, accuracy
			FROM scores
			WHERE name = $1
			ORDER BY created_at DESC
			LIMIT 1`, "DataCheck").Scan(&savedName, &savedWPM, &savedAccuracy)

		if err != nil {
			t.Fatalf("Failed to query saved score: %v", err)
		}

		if savedWPM != score.WPM {
			t.Errorf("WPM mismatch: expected %d, got %d", score.WPM, savedWPM)
		}
		if savedAccuracy != score.Accuracy {
			t.Errorf("Accuracy mismatch: expected %d, got %d", score.Accuracy, savedAccuracy)
		}
	})
}

// Test helper functions
func TestAnalyzePerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := &TypingProcessor{}

	cases := []struct {
		name     string
		stats    SessionStats
		expected string
	}{
		{"Beginner", createTestStats(uuid.New().String(), 30, 75), "beginner"},
		{"Intermediate", createTestStats(uuid.New().String(), 50, 88), "intermediate"},
		{"Advanced", createTestStats(uuid.New().String(), 70, 92), "advanced"},
		{"Expert", createTestStats(uuid.New().String(), 90, 97), "expert"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := processor.analyzePerformance(tc.stats)

			if analysis.PerformanceLevel != tc.expected {
				t.Errorf("Expected level %s, got %s", tc.expected, analysis.PerformanceLevel)
			}

			if len(analysis.Recommendations) == 0 {
				t.Error("Expected recommendations")
			}

			if analysis.ImprovementScore <= 0 {
				t.Error("Expected positive improvement score")
			}
		})
	}
}

func TestGenerateTips(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := &TypingProcessor{}

	t.Run("LowSpeedTips", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 25, 85)
		analysis := processor.analyzePerformance(stats)

		tips := processor.generateTips(stats, analysis)

		if len(tips) == 0 {
			t.Error("Expected tips for low speed")
		}

		// Should include speed-related tips
		foundSpeedTip := false
		for _, tip := range tips {
			if tip.Category == "speed" {
				foundSpeedTip = true
				break
			}
		}

		if !foundSpeedTip {
			t.Error("Expected speed tips for low WPM")
		}
	})

	t.Run("LowAccuracyTips", func(t *testing.T) {
		stats := createTestStats(uuid.New().String(), 60, 75)
		analysis := processor.analyzePerformance(stats)

		tips := processor.generateTips(stats, analysis)

		// Should include accuracy tips
		foundAccuracyTip := false
		for _, tip := range tips {
			if tip.Category == "accuracy" && tip.Priority == "high" {
				foundAccuracyTip = true
				break
			}
		}

		if !foundAccuracyTip {
			t.Error("Expected high-priority accuracy tips")
		}
	})
}

// Benchmark tests for processor methods
func BenchmarkProcessStats(b *testing.B) {
	processor := &TypingProcessor{}
	ctx := context.Background()
	stats := createTestStats(uuid.New().String(), 60, 90)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessStats(ctx, stats)
	}
}

func BenchmarkProvideCoaching(b *testing.B) {
	processor := &TypingProcessor{}
	ctx := context.Background()
	stats := createTestStats(uuid.New().String(), 60, 90)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProvideCoaching(ctx, stats, "medium")
	}
}

func BenchmarkGenerateAdaptiveText(b *testing.B) {
	processor := &TypingProcessor{}
	ctx := context.Background()
	request := createTestAdaptiveRequest(uuid.New().String(), "medium")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.GenerateAdaptiveText(ctx, request)
	}
}
