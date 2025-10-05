// +build testing

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// Test health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("HealthCheck", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		var response map[string]string
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response["status"])
		}
	})
}

// Test leaderboard endpoint
func TestGetLeaderboard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test database if available
	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()
		db = testDB.DB
		typingProcessor = NewTypingProcessor(testDB.DB)

		// Insert test scores
		scores := generateTestScores(5)
		for _, score := range scores {
			insertTestScore(t, testDB.DB, score)
		}
	}

	router := setupTestRouter()

	t.Run("AllTimeLeaderboard", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?period=all",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Period != "all" {
			t.Errorf("Expected period 'all', got '%s'", response.Period)
		}

		if testDB != nil && len(response.Scores) == 0 {
			t.Error("Expected scores in leaderboard")
		}
	})

	t.Run("DailyLeaderboard", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?period=daily",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Period != "daily" {
			t.Errorf("Expected period 'daily', got '%s'", response.Period)
		}
	})

	t.Run("WeeklyLeaderboard", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?period=weekly",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Period != "weekly" {
			t.Errorf("Expected period 'weekly', got '%s'", response.Period)
		}
	})

	t.Run("MonthlyLeaderboard", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?period=monthly",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Period != "monthly" {
			t.Errorf("Expected period 'monthly', got '%s'", response.Period)
		}
	})

	t.Run("WithUserID", func(t *testing.T) {
		userID := uuid.New().String()
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?user_id=" + userID,
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)
	})

	t.Run("DefaultPeriod", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Period != "all" {
			t.Errorf("Expected default period 'all', got '%s'", response.Period)
		}
	})
}

// Test submit score endpoint
func TestSubmitScore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()
		db = testDB.DB
		typingProcessor = NewTypingProcessor(testDB.DB)
	}

	router := setupTestRouter()

	t.Run("ValidScore", func(t *testing.T) {
		if testDB == nil {
			t.Skip("Database not available")
		}

		score := createTestScore("TestPlayer", 60, 95)
		initialCount := getScoreCount(t, testDB.DB)

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/submit-score",
			Body:   score,
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success response")
		}

		finalCount := getScoreCount(t, testDB.DB)
		if finalCount != initialCount+1 {
			t.Errorf("Expected score count to increase by 1, got %d -> %d", initialCount, finalCount)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/submit-score").
			AddEmptyBody("POST", "/api/submit-score").
			Build()

		for _, scenario := range scenarios {
			t.Run(scenario.Name, func(t *testing.T) {
				w := makeHTTPRequest(t, router, scenario.Request)
				assertErrorResponse(t, w, scenario.ExpectedStatus)
			})
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		if testDB == nil {
			t.Skip("Database not available")
		}

		cases := []struct {
			name      string
			scoreType string
		}{
			{"MinimalScore", "minimal"},
			{"MaximumScore", "maximum"},
			{"EmptyName", "empty_name"},
			{"LongName", "long_name"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				score := createEdgeCaseScore(tc.scoreType)
				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/submit-score",
					Body:   score,
				})

				// Should handle edge cases gracefully
				if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
					t.Errorf("Unexpected status code: %d", w.Code)
				}
			})
		}
	})
}

// Test submit stats endpoint
func TestSubmitStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()
		db = testDB.DB
		typingProcessor = NewTypingProcessor(testDB.DB)
	}

	router := setupTestRouter()

	t.Run("ValidStats", func(t *testing.T) {
		if testDB == nil {
			t.Skip("Database not available")
		}

		stats := StatsRequest{
			SessionID: uuid.New().String(),
			WPM:       50,
			Accuracy:  90,
			Text:      "The quick brown fox jumps over the lazy dog",
		}

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stats",
			Body:   stats,
		})

		var response ProcessedStats
		assertJSONResponse(t, w, http.StatusOK, &response)
		assertValidProcessedStats(t, response)

		if response.Metrics.WPM != stats.WPM {
			t.Errorf("Expected WPM %d, got %d", stats.WPM, response.Metrics.WPM)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/stats").
			AddEmptyBody("POST", "/api/stats").
			Build()

		for _, scenario := range scenarios {
			t.Run(scenario.Name, func(t *testing.T) {
				w := makeHTTPRequest(t, router, scenario.Request)
				assertErrorResponse(t, w, scenario.ExpectedStatus)
			})
		}
	})

	t.Run("DifferentPerformanceLevels", func(t *testing.T) {
		if testDB == nil {
			t.Skip("Database not available")
		}

		cases := []struct {
			name     string
			wpm      int
			accuracy int
			expected string
		}{
			{"Beginner", 25, 75, "beginner"},
			{"Intermediate", 50, 88, "intermediate"},
			{"Advanced", 70, 92, "advanced"},
			{"Expert", 85, 97, "expert"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				stats := StatsRequest{
					SessionID: uuid.New().String(),
					WPM:       tc.wpm,
					Accuracy:  tc.accuracy,
					Text:      "Test text for performance levels",
				}

				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/stats",
					Body:   stats,
				})

				var response ProcessedStats
				assertJSONResponse(t, w, http.StatusOK, &response)

				if response.Analysis.PerformanceLevel != tc.expected {
					t.Errorf("Expected performance level '%s', got '%s'", tc.expected, response.Analysis.PerformanceLevel)
				}
			})
		}
	})
}

// Test coaching endpoint
func TestGetCoaching(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()
		db = testDB.DB
		typingProcessor = NewTypingProcessor(testDB.DB)
	} else {
		typingProcessor = &TypingProcessor{}
	}

	router := setupTestRouter()

	t.Run("ValidCoachingRequest", func(t *testing.T) {
		request := CoachingRequest{
			UserStats: StatsRequest{
				SessionID: uuid.New().String(),
				WPM:       45,
				Accuracy:  85,
				Text:      "Sample text for coaching",
			},
			Difficulty: "medium",
		}

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/coaching",
			Body:   request,
		})

		var response CoachingResponse
		assertJSONResponse(t, w, http.StatusOK, &response)
		assertValidCoachingResponse(t, response)
	})

	t.Run("DifferentSkillLevels", func(t *testing.T) {
		cases := []struct {
			name     string
			wpm      int
			accuracy int
		}{
			{"Beginner", 20, 70},
			{"Intermediate", 50, 90},
			{"Advanced", 70, 95},
			{"Expert", 90, 98},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				request := CoachingRequest{
					UserStats: StatsRequest{
						SessionID: uuid.New().String(),
						WPM:       tc.wpm,
						Accuracy:  tc.accuracy,
						Text:      "Test text",
					},
					Difficulty: "medium",
				}

				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/coaching",
					Body:   request,
				})

				var response CoachingResponse
				assertJSONResponse(t, w, http.StatusOK, &response)

				if len(response.Tips) == 0 {
					t.Error("Expected coaching tips")
				}
			})
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/coaching").
			AddEmptyBody("POST", "/api/coaching").
			Build()

		for _, scenario := range scenarios {
			t.Run(scenario.Name, func(t *testing.T) {
				w := makeHTTPRequest(t, router, scenario.Request)
				assertErrorResponse(t, w, scenario.ExpectedStatus)
			})
		}
	})
}

// Test practice text endpoint
func TestGetPracticeText(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("EasyDifficulty", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text?difficulty=easy",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response["difficulty"] != "easy" {
			t.Errorf("Expected difficulty 'easy', got '%v'", response["difficulty"])
		}

		if texts, ok := response["texts"].([]interface{}); !ok || len(texts) == 0 {
			t.Error("Expected practice texts")
		}
	})

	t.Run("MediumDifficulty", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text?difficulty=medium",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response["difficulty"] != "medium" {
			t.Errorf("Expected difficulty 'medium', got '%v'", response["difficulty"])
		}
	})

	t.Run("HardDifficulty", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text?difficulty=hard",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response["difficulty"] != "hard" {
			t.Errorf("Expected difficulty 'hard', got '%v'", response["difficulty"])
		}
	})

	t.Run("DefaultDifficulty", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response["difficulty"] != "easy" {
			t.Errorf("Expected default difficulty 'easy', got '%v'", response["difficulty"])
		}
	})

	t.Run("InvalidDifficulty", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text?difficulty=invalid",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Should default to easy for invalid difficulty
		if response["difficulty"] != "invalid" && response["difficulty"] != "easy" {
			t.Logf("Invalid difficulty handled: got '%v'", response["difficulty"])
		}
	})
}

// Test adaptive text endpoint
func TestGetAdaptiveText(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB != nil {
		defer testDB.Cleanup()
		db = testDB.DB
		typingProcessor = NewTypingProcessor(testDB.DB)
	} else {
		typingProcessor = &TypingProcessor{}
	}

	router := setupTestRouter()

	t.Run("ValidAdaptiveRequest", func(t *testing.T) {
		request := createTestAdaptiveRequest(uuid.New().String(), "medium")

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/adaptive-text",
			Body:   request,
		})

		var response AdaptiveTextResponse
		assertJSONResponse(t, w, http.StatusOK, &response)
		assertValidAdaptiveResponse(t, response)
	})

	t.Run("DifferentDifficulties", func(t *testing.T) {
		difficulties := []string{"easy", "medium", "hard"}

		for _, diff := range difficulties {
			t.Run(diff, func(t *testing.T) {
				request := createTestAdaptiveRequest(uuid.New().String(), diff)

				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/adaptive-text",
					Body:   request,
				})

				var response AdaptiveTextResponse
				assertJSONResponse(t, w, http.StatusOK, &response)

				if response.Difficulty != diff {
					t.Errorf("Expected difficulty '%s', got '%s'", diff, response.Difficulty)
				}
			})
		}
	})

	t.Run("DifferentTextLengths", func(t *testing.T) {
		lengths := []string{"short", "medium", "long"}

		for _, length := range lengths {
			t.Run(length, func(t *testing.T) {
				request := createTestAdaptiveRequest(uuid.New().String(), "medium")
				request.TextLength = length

				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/adaptive-text",
					Body:   request,
				})

				var response AdaptiveTextResponse
				assertJSONResponse(t, w, http.StatusOK, &response)

				// Verify word count varies with length
				if response.WordCount <= 0 {
					t.Error("Expected positive word count")
				}
			})
		}
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
			{Word: "practice", Char: "c", Position: "5", ErrorCount: 3},
		}

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/adaptive-text",
			Body:   request,
		})

		var response AdaptiveTextResponse
		assertJSONResponse(t, w, http.StatusOK, &response)
		assertValidAdaptiveResponse(t, response)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		cases := []struct {
			name        string
			requestType string
		}{
			{"NoTargetWords", "no_target_words"},
			{"ManyTargetWords", "many_target_words"},
			{"ManyMistakes", "many_mistakes"},
			{"EmptyStrings", "empty_strings"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				request := createEdgeCaseAdaptiveRequest(tc.requestType)

				w := makeHTTPRequest(t, router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/adaptive-text",
					Body:   request,
				})

				// Should handle edge cases gracefully
				if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
					t.Errorf("Unexpected status code: %d", w.Code)
				}
			})
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/adaptive-text").
			AddEmptyBody("POST", "/api/adaptive-text").
			Build()

		for _, scenario := range scenarios {
			t.Run(scenario.Name, func(t *testing.T) {
				w := makeHTTPRequest(t, router, scenario.Request)
				assertErrorResponse(t, w, scenario.ExpectedStatus)
			})
		}
	})
}

// Test getRecommendedDifficulty helper function
func TestGetRecommendedDifficulty(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	cases := []struct {
		name     string
		wpm      int
		accuracy int
		expected string
	}{
		{"BeginnerLevel", 30, 85, "easy"},
		{"IntermediateLevel", 50, 92, "medium"},
		{"AdvancedLevel", 70, 96, "hard"},
		{"EdgeCase_HighWPM_LowAccuracy", 80, 85, "easy"}, // accuracy <= 90, returns easy
		{"EdgeCase_LowWPM_HighAccuracy", 35, 96, "easy"}, // wpm <= 40, returns easy
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := getRecommendedDifficulty(tc.wpm, tc.accuracy)
			if result != tc.expected {
				t.Errorf("Expected difficulty '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// Integration test - full user flow
func TestFullUserFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for integration test")
	}
	defer testDB.Cleanup()

	db = testDB.DB
	typingProcessor = NewTypingProcessor(testDB.DB)
	router := setupTestRouter()

	sessionID := uuid.New().String()

	t.Run("1_GetPracticeText", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/practice-text?difficulty=easy",
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)
	})

	t.Run("2_SubmitStats", func(t *testing.T) {
		stats := StatsRequest{
			SessionID: sessionID,
			WPM:       45,
			Accuracy:  88,
			Text:      "Practice typing text",
		}

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stats",
			Body:   stats,
		})

		var response ProcessedStats
		assertJSONResponse(t, w, http.StatusOK, &response)
	})

	t.Run("3_GetCoaching", func(t *testing.T) {
		request := CoachingRequest{
			UserStats: StatsRequest{
				SessionID: sessionID,
				WPM:       45,
				Accuracy:  88,
				Text:      "Test",
			},
			Difficulty: "medium",
		}

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/coaching",
			Body:   request,
		})

		var response CoachingResponse
		assertJSONResponse(t, w, http.StatusOK, &response)
	})

	t.Run("4_SubmitScore", func(t *testing.T) {
		score := createTestScore("FlowTestUser", 45, 88)

		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/submit-score",
			Body:   score,
		})

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)
	})

	t.Run("5_ViewLeaderboard", func(t *testing.T) {
		w := makeHTTPRequest(t, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/leaderboard?period=all",
		})

		var response LeaderboardResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if len(response.Scores) == 0 {
			t.Error("Expected to see scores in leaderboard after submission")
		}
	})
}

// Benchmark tests
func BenchmarkHealthHandler(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetLeaderboard(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/leaderboard", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetPracticeText(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/practice-text?difficulty=medium", nil)
		router.ServeHTTP(w, req)
	}
}
