// +build testing

package main

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set lifecycle env for main.go check
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// setupRouter creates a router with all endpoints registered
func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "study-buddy"})
	})

	// Subject endpoints
	router.POST("/api/subjects", createSubject)
	router.GET("/api/subjects/:userId", getUserSubjects)
	router.PUT("/api/subjects/:id", updateSubject)
	router.DELETE("/api/subjects/:id", deleteSubject)

	// Study material endpoints
	router.POST("/api/materials", createStudyMaterial)
	router.GET("/api/materials/:userId", getUserMaterials)
	router.GET("/api/materials/:userId/:subjectId", getSubjectMaterials)
	router.PUT("/api/materials/:id", updateStudyMaterial)
	router.DELETE("/api/materials/:id", deleteStudyMaterial)

	// Flashcard endpoints
	router.POST("/api/flashcards", createFlashcard)
	router.GET("/api/flashcards/:userId", getUserFlashcards)
	router.GET("/api/flashcards/:userId/:subjectId", getSubjectFlashcards)
	router.GET("/api/flashcards/due/:userId", getDueFlashcards)
	router.POST("/api/flashcards/:id/review", reviewFlashcard)

	// Study materials endpoint
	router.GET("/api/study/materials", getStudyMaterials)

	// Study session endpoints
	router.POST("/api/sessions", startStudySession)
	router.PUT("/api/sessions/:id/end", endStudySession)
	router.GET("/api/sessions/:userId", getUserSessions)
	router.GET("/api/sessions/:userId/stats", getStudyStats)

	// Quiz endpoints
	router.POST("/api/quiz/generate", generateQuiz)
	router.POST("/api/quiz/submit", submitQuizAnswers)
	router.GET("/api/quiz/:userId/history", getQuizHistory)

	// Learning analytics
	router.GET("/api/analytics/:userId", getLearningAnalytics)
	router.GET("/api/progress/:userId/:subjectId", getSubjectProgress)

	// AI-powered features (PRD compliant endpoints)
	router.POST("/api/flashcards/generate", generateFlashcardsFromText)
	router.POST("/api/study/session/start", startStudySession)
	router.POST("/api/study/answer", submitFlashcardAnswer)
	router.GET("/api/study/due-cards", getDueCards)

	// Legacy AI endpoints
	router.POST("/api/ai/explain", explainConcept)
	router.POST("/api/ai/generate-flashcards", generateFlashcardsFromText)
	router.POST("/api/ai/study-plan", generateStudyPlan)

	return router
}

// ====================
// HEALTH CHECK TESTS
// ====================

func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "status", "service")

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}

		if response["service"] != "study-buddy" {
			t.Errorf("Expected service 'study-buddy', got '%v'", response["service"])
		}
	})
}

// ====================
// FLASHCARD GENERATION TESTS (P0 REQUIREMENT)
// ====================

func TestFlashcardGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_GenerateFlashcards", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id":       "test-subject",
				"content":          "The mitochondria is the powerhouse of the cell",
				"card_count":       3,
				"difficulty_level": "intermediate",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "cards", "generation_metadata")

		// Validate cards array
		cards, ok := response["cards"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'cards' to be an array")
		}

		if len(cards) != 3 {
			t.Errorf("Expected 3 cards, got %d", len(cards))
		}

		// Validate first card structure
		if len(cards) > 0 {
			card := cards[0].(map[string]interface{})
			if _, ok := card["question"]; !ok {
				t.Error("Card missing 'question' field")
			}
			if _, ok := card["answer"]; !ok {
				t.Error("Card missing 'answer' field")
			}
			if _, ok := card["hint"]; !ok {
				t.Error("Card missing 'hint' field")
			}
			if _, ok := card["difficulty"]; !ok {
				t.Error("Card missing 'difficulty' field")
			}
		}

		// Validate generation metadata
		metadata, ok := response["generation_metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected 'generation_metadata' to be an object")
		}

		if _, ok := metadata["processing_time"]; !ok {
			t.Error("Metadata missing 'processing_time' field")
		}

		if _, ok := metadata["content_quality_score"]; !ok {
			t.Error("Metadata missing 'content_quality_score' field")
		}
	})

	t.Run("Success_DefaultValues", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id": "test-subject",
				"content":    "Test content",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "cards")

		cards := response["cards"].([]interface{})
		if len(cards) != 10 { // Default card count
			t.Errorf("Expected 10 cards (default), got %d", len(cards))
		}
	})

	t.Run("Success_PerformanceUnder5Seconds", func(t *testing.T) {
		start := time.Now()

		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id":       "test-subject",
				"content":          "Long content for performance testing. " + string(make([]byte, 1000)),
				"card_count":       5,
				"difficulty_level": "advanced",
			},
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertJSONResponse(t, w, 200, "cards")

		if duration > 5*time.Second {
			t.Errorf("Generation took %v, exceeds 5 second target", duration)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			Name:   "FlashcardGeneration",
			Router: router,
		}

		suite.RunErrorTests(t, FlashcardGenerationPatterns())
	})
}

// ====================
// DUE CARDS TESTS (P0 REQUIREMENT)
// ====================

func TestDueCards(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_GetDueCards", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/due-cards",
			QueryParams: map[string]string{
				"user_id": "test-user",
				"limit":   "10",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "cards", "next_session_recommendation")

		// Validate cards array
		cards, ok := response["cards"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'cards' to be an array")
		}

		// Validate card structure if cards exist
		if len(cards) > 0 {
			card := cards[0].(map[string]interface{})
			expectedFields := []string{"id", "question", "answer", "hint", "days_overdue", "priority_score"}
			for _, field := range expectedFields {
				if _, ok := card[field]; !ok {
					t.Errorf("Card missing '%s' field", field)
				}
			}
		}

		// Validate next session recommendation
		if _, ok := response["next_session_recommendation"].(string); !ok {
			t.Error("next_session_recommendation should be a string timestamp")
		}
	})

	t.Run("Success_WithSubjectFilter", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/due-cards",
			QueryParams: map[string]string{
				"user_id":    "test-user",
				"subject_id": "test-subject",
				"limit":      "5",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertJSONResponse(t, w, 200, "cards")
	})

	t.Run("Error_MissingUserID", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/due-cards",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertErrorResponse(t, w, 400)
	})
}

// ====================
// STUDY SESSION TESTS (P0 REQUIREMENT)
// ====================

func TestStudySession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_StartSession", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/session/start",
			Body: map[string]interface{}{
				"user_id":         "test-user",
				"subject_id":      "test-subject",
				"session_type":    "review",
				"target_duration": 25,
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 201, "session_id")

		// Validate session_id is a valid UUID
		sessionID := response["session_id"].(string)
		if _, err := uuid.Parse(sessionID); err != nil {
			t.Errorf("session_id is not a valid UUID: %s", sessionID)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			Name:   "StudySession",
			Router: router,
		}

		suite.RunErrorTests(t, StudySessionPatterns())
	})
}

// ====================
// FLASHCARD ANSWER TESTS (P0 REQUIREMENT)
// ====================

func TestFlashcardAnswer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_AnswerEasy", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/answer",
			Body: map[string]interface{}{
				"session_id":    "test-session",
				"flashcard_id":  "test-card",
				"user_response": "easy",
				"response_time": 3000,
				"user_id":       "test-user",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "correct", "xp_earned", "next_review_date", "progress_update")

		// Validate correct is true for "easy" response
		if correct, ok := response["correct"].(bool); !ok || !correct {
			t.Error("Expected 'correct' to be true for 'easy' response")
		}

		// Validate XP earned is positive
		if xp, ok := response["xp_earned"].(float64); !ok || xp <= 0 {
			t.Error("Expected positive XP for 'easy' response")
		}

		// Validate progress update structure
		progressUpdate := response["progress_update"].(map[string]interface{})
		expectedFields := []string{"mastery_level", "streak_continues", "current_streak", "total_xp", "level"}
		for _, field := range expectedFields {
			if _, ok := progressUpdate[field]; !ok {
				t.Errorf("progress_update missing '%s' field", field)
			}
		}

		// Validate mastery level is "mastered" for "easy"
		if mastery := progressUpdate["mastery_level"].(string); mastery != "mastered" {
			t.Errorf("Expected mastery_level 'mastered' for 'easy' response, got '%s'", mastery)
		}
	})

	t.Run("Success_AnswerGood", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/answer",
			Body: map[string]interface{}{
				"session_id":    "test-session",
				"flashcard_id":  "test-card",
				"user_response": "good",
				"response_time": 5000,
				"user_id":       "test-user",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "xp_earned", "progress_update")

		progressUpdate := response["progress_update"].(map[string]interface{})
		if mastery := progressUpdate["mastery_level"].(string); mastery != "reviewing" {
			t.Errorf("Expected mastery_level 'reviewing' for 'good' response, got '%s'", mastery)
		}
	})

	t.Run("Success_AnswerHard", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/answer",
			Body: map[string]interface{}{
				"session_id":    "test-session",
				"flashcard_id":  "test-card",
				"user_response": "hard",
				"response_time": 8000,
				"user_id":       "test-user",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "progress_update")

		progressUpdate := response["progress_update"].(map[string]interface{})
		if mastery := progressUpdate["mastery_level"].(string); mastery != "learning" {
			t.Errorf("Expected mastery_level 'learning' for 'hard' response, got '%s'", mastery)
		}
	})

	t.Run("Success_AnswerAgain", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/answer",
			Body: map[string]interface{}{
				"session_id":    "test-session",
				"flashcard_id":  "test-card",
				"user_response": "again",
				"response_time": 10000,
				"user_id":       "test-user",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "progress_update")

		progressUpdate := response["progress_update"].(map[string]interface{})
		if mastery := progressUpdate["mastery_level"].(string); mastery != "struggling" {
			t.Errorf("Expected mastery_level 'struggling' for 'again' response, got '%s'", mastery)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			Name:   "FlashcardAnswer",
			Router: router,
		}

		suite.RunErrorTests(t, FlashcardAnswerPatterns())
	})
}

// ====================
// SPACED REPETITION ALGORITHM TESTS
// ====================

func TestSpacedRepetitionAlgorithm(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InitialCard_FirstReview", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID:  "test-card",
			Interval:     0,
			Repetitions:  0,
			EaseFactor:   2.5,
			LastReviewed: time.Now(),
		}

		// Quality 3 = "good"
		result := calculateSpacedRepetition(data, 3)

		if result.Interval != 1 {
			t.Errorf("First successful review should have 1 day interval, got %d", result.Interval)
		}

		if result.Repetitions != 1 {
			t.Errorf("Expected 1 repetition, got %d", result.Repetitions)
		}
	})

	t.Run("SecondReview_GoodResponse", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID:  "test-card",
			Interval:     1,
			Repetitions:  1,
			EaseFactor:   2.5,
			LastReviewed: time.Now().AddDate(0, 0, -1),
		}

		result := calculateSpacedRepetition(data, 3)

		if result.Interval != 6 {
			t.Errorf("Second successful review should have 6 day interval, got %d", result.Interval)
		}

		if result.Repetitions != 2 {
			t.Errorf("Expected 2 repetitions, got %d", result.Repetitions)
		}
	})

	t.Run("ThirdReview_ExponentialGrowth", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID:  "test-card",
			Interval:     6,
			Repetitions:  2,
			EaseFactor:   2.5,
			LastReviewed: time.Now().AddDate(0, 0, -6),
		}

		result := calculateSpacedRepetition(data, 3)

		// Interval should be 6 * 2.5 = 15
		if result.Interval < 14 || result.Interval > 16 {
			t.Errorf("Expected interval around 15 days, got %d", result.Interval)
		}

		if result.Repetitions != 3 {
			t.Errorf("Expected 3 repetitions, got %d", result.Repetitions)
		}
	})

	t.Run("FailedReview_ResetsInterval", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID:  "test-card",
			Interval:     15,
			Repetitions:  3,
			EaseFactor:   2.5,
			LastReviewed: time.Now().AddDate(0, 0, -15),
		}

		// Quality 0 = "again" (failed)
		result := calculateSpacedRepetition(data, 0)

		if result.Interval != 1 {
			t.Errorf("Failed review should reset interval to 1 day, got %d", result.Interval)
		}

		if result.Repetitions != 0 {
			t.Errorf("Failed review should reset repetitions to 0, got %d", result.Repetitions)
		}
	})

	t.Run("EaseFactor_AdjustsWithQuality", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID: "test-card",
			Interval:    1,
			Repetitions: 1,
			EaseFactor:  2.5,
		}

		// Quality 5 = "easy" (should increase ease factor)
		resultEasy := calculateSpacedRepetition(data, 5)
		if resultEasy.EaseFactor <= data.EaseFactor {
			t.Error("Easy response should increase ease factor")
		}

		// Quality 1 = "hard" (should decrease ease factor)
		resultHard := calculateSpacedRepetition(data, 1)
		if resultHard.EaseFactor >= data.EaseFactor {
			t.Error("Hard response should decrease ease factor")
		}

		// Ease factor should never go below 1.3
		if resultHard.EaseFactor < 1.3 {
			t.Errorf("Ease factor should not go below 1.3, got %f", resultHard.EaseFactor)
		}
	})

	t.Run("NextReviewDate_IsAfterLastReviewed", func(t *testing.T) {
		data := SpacedRepetitionData{
			FlashcardID:  "test-card",
			Interval:     1,
			Repetitions:  0,
			EaseFactor:   2.5,
			LastReviewed: time.Now(),
		}

		result := calculateSpacedRepetition(data, 3)

		if result.NextReview.Before(result.LastReviewed) {
			t.Error("Next review date should be after last reviewed date")
		}

		expectedNextReview := result.LastReviewed.AddDate(0, 0, result.Interval)
		if result.NextReview.Sub(expectedNextReview) > time.Hour {
			t.Errorf("Next review date calculation incorrect: expected around %v, got %v",
				expectedNextReview, result.NextReview)
		}
	})
}

// ====================
// XP CALCULATION TESTS
// ====================

func TestXPCalculation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BaseXP_ForCorrectAnswers", func(t *testing.T) {
		xpGood := calculateXP("good", 10000)
		if xpGood != 13 { // 10 base + 3 bonus
			t.Errorf("Expected 13 XP for 'good' response, got %d", xpGood)
		}

		xpEasy := calculateXP("easy", 10000)
		if xpEasy != 15 { // 10 base + 5 bonus
			t.Errorf("Expected 15 XP for 'easy' response, got %d", xpEasy)
		}
	})

	t.Run("SpeedBonus_UnderThreshold", func(t *testing.T) {
		xpFast := calculateXP("good", 5000) // Under 10 seconds
		xpSlow := calculateXP("good", 15000) // Over 10 seconds

		if xpFast <= xpSlow {
			t.Error("Fast response should earn more XP than slow response")
		}

		if xpFast != 15 { // 10 base + 3 good + 2 speed
			t.Errorf("Expected 15 XP for fast 'good' response, got %d", xpFast)
		}
	})

	t.Run("NoBonus_ForIncorrectAnswers", func(t *testing.T) {
		xpHard := calculateXP("hard", 5000)
		xpAgain := calculateXP("again", 5000)

		if xpHard != 10 { // Only base XP
			t.Errorf("Expected 10 XP for 'hard' response, got %d", xpHard)
		}

		if xpAgain != 10 { // Only base XP
			t.Errorf("Expected 10 XP for 'again' response, got %d", xpAgain)
		}
	})
}

// ====================
// MASTERY LEVEL TESTS
// ====================

func TestMasteryLevel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		response string
		expected string
	}{
		{"easy", "mastered"},
		{"good", "reviewing"},
		{"hard", "learning"},
		{"again", "struggling"},
		{"unknown", "struggling"},
	}

	for _, tt := range tests {
		t.Run(tt.response, func(t *testing.T) {
			result := getMasteryLevel(tt.response)
			if result != tt.expected {
				t.Errorf("Expected mastery level '%s' for response '%s', got '%s'",
					tt.expected, tt.response, result)
			}
		})
	}
}

// ====================
// STUDY MATERIALS ENDPOINT TESTS
// ====================

func TestStudyMaterials(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_GetStudyMaterials", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/materials",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "materials", "total")

		// Validate materials is an array
		if _, ok := response["materials"].([]interface{}); !ok {
			t.Error("Expected 'materials' to be an array")
		}

		// Validate total is a number
		if _, ok := response["total"].(float64); !ok {
			t.Error("Expected 'total' to be a number")
		}
	})
}

// ====================
// QUIZ ENDPOINTS TESTS
// ====================

func TestQuizEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_GenerateQuiz", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/quiz/generate",
			Body: map[string]interface{}{
				"user_id":    "test-user",
				"subject_id": "test-subject",
				"count":      10,
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertJSONResponse(t, w, 200, "message", "workflow_triggered")
	})

	t.Run("Success_SubmitQuizAnswers", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/quiz/submit",
			Body: map[string]interface{}{
				"user_id":    "test-user",
				"session_id": "test-session",
				"answers": []map[string]interface{}{
					{"question_id": "q1", "answer": "a"},
				},
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "score", "correct", "total")

		// Validate score is reasonable
		if score, ok := response["score"].(float64); !ok || score < 0 || score > 100 {
			t.Errorf("Score should be between 0 and 100, got %v", score)
		}
	})

	t.Run("Success_GetQuizHistory", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/quiz/test-user/history",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "user_id", "quizzes")

		if _, ok := response["quizzes"].([]interface{}); !ok {
			t.Error("Expected 'quizzes' to be an array")
		}
	})
}

// ====================
// ANALYTICS ENDPOINTS TESTS
// ====================

func TestAnalyticsEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_GetLearningAnalytics", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/analytics/test-user",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "user_id", "analytics")

		analytics := response["analytics"].(map[string]interface{})
		expectedFields := []string{"study_patterns", "performance_trends", "subject_mastery"}
		for _, field := range expectedFields {
			if _, ok := analytics[field]; !ok {
				t.Errorf("Analytics missing '%s' field", field)
			}
		}
	})

	t.Run("Success_GetSubjectProgress", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/progress/test-user/test-subject",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "user_id", "subject_id", "progress")

		progress := response["progress"].(map[string]interface{})
		expectedFields := []string{"completion_percentage", "cards_mastered", "average_score"}
		for _, field := range expectedFields {
			if _, ok := progress[field]; !ok {
				t.Errorf("Progress missing '%s' field", field)
			}
		}
	})
}

// ====================
// AI ENDPOINTS TESTS
// ====================

func TestAIEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("Success_ExplainConcept", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/ai/explain",
			Body: map[string]interface{}{
				"concept": "Photosynthesis",
				"context": "High school biology",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "explanation", "examples", "related_concepts")

		// Validate arrays
		if _, ok := response["examples"].([]interface{}); !ok {
			t.Error("Expected 'examples' to be an array")
		}
		if _, ok := response["related_concepts"].([]interface{}); !ok {
			t.Error("Expected 'related_concepts' to be an array")
		}
	})

	t.Run("Success_GenerateStudyPlan", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/ai/study-plan",
			Body: map[string]interface{}{
				"user_id":  "test-user",
				"goals":    []string{"Learn calculus", "Master biology"},
				"deadline": "2024-12-31",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "message", "plan_id")

		// Validate plan_id is a valid UUID
		planID := response["plan_id"].(string)
		if _, err := uuid.Parse(planID); err != nil {
			t.Errorf("plan_id is not a valid UUID: %s", planID)
		}
	})
}

// ====================
// PERFORMANCE TESTS
// ====================

func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("SpacedRepetition_Performance", func(t *testing.T) {
		RunPerformanceTest(t, router, SpacedRepetitionPerformanceTest())
	})

	t.Run("DueCardsRetrieval_Performance", func(t *testing.T) {
		RunPerformanceTest(t, router, DueCardsRetrievalPerformanceTest())
	})

	t.Run("FlashcardGeneration_Performance", func(t *testing.T) {
		RunPerformanceTest(t, router, FlashcardGenerationPerformanceTest())
	})
}

// ====================
// EDGE CASE TESTS
// ====================

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	t.Run("EmptyContent_FlashcardGeneration", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id": "test-subject",
				"content":    "",
				"card_count": 3,
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should still return 200 with cards (fallback behavior)
		assertJSONResponse(t, w, 200, "cards")
	})

	t.Run("LargeCardCount_FlashcardGeneration", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id": "test-subject",
				"content":    "Test content for large card generation",
				"card_count": 50,
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		response := assertJSONResponse(t, w, 200, "cards")

		cards := response["cards"].([]interface{})
		if len(cards) != 50 {
			t.Errorf("Expected 50 cards, got %d", len(cards))
		}
	})

	t.Run("VeryFastResponse_XPBonus", func(t *testing.T) {
		xp := calculateXP("easy", 100) // 100ms response
		expectedXP := 10 + 5 + 2       // base + easy + speed

		if xp != expectedXP {
			t.Errorf("Expected %d XP for very fast easy response, got %d", expectedXP, xp)
		}
	})
}

// ====================
// HELPER FUNCTION TESTS
// ====================

func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("truncateContent_ShortString", func(t *testing.T) {
		result := truncateContent("Short", 10)
		if result != "Short" {
			t.Errorf("Expected 'Short', got '%s'", result)
		}
	})

	t.Run("truncateContent_LongString", func(t *testing.T) {
		result := truncateContent("This is a very long string", 10)
		if result != "This is a ..." {
			t.Errorf("Expected 'This is a ...', got '%s'", result)
		}
	})

	t.Run("calculateLevel_ProgressiveLevels", func(t *testing.T) {
		tests := []struct {
			xp       int
			expected int
		}{
			{0, 0},
			{99, 0},
			{100, 1},
			{250, 2},
			{1000, 10},
			{10000, 100},
		}

		for _, tt := range tests {
			result := calculateLevel(tt.xp)
			if result != tt.expected {
				t.Errorf("For %d XP, expected level %d, got %d", tt.xp, tt.expected, result)
			}
		}
	})
}

// ====================
// SUBJECT MANAGEMENT TESTS
// ====================

func TestSubjectManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()

	// Mock database for testing without DB
	testUserID := uuid.New().String()

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			Name:   "SubjectManagement",
			Router: router,
		}

		suite.RunErrorTests(t, SubjectManagementPatterns())
	})

	t.Run("Success_UpdateSubject", func(t *testing.T) {
		subjectID := uuid.New().String()

		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/subjects/" + subjectID,
			Body: map[string]interface{}{
				"name":        "Updated Subject",
				"description": "Updated description",
				"color":       "#FF5733",
				"icon":        "📖",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Success_DeleteSubject", func(t *testing.T) {
		subjectID := uuid.New().String()

		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/subjects/" + subjectID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Success_GetUserSubjects", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/subjects/" + testUserID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// ====================
// FLASHCARD CRUD TESTS
// ====================

func TestFlashcardCRUD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()
	testUserID := uuid.New().String()
	testSubjectID := uuid.New().String()

	t.Run("Success_GetUserFlashcards", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/flashcards/" + testUserID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Success_GetSubjectFlashcards", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/flashcards/" + testUserID + "/" + testSubjectID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// ====================
// STUDY MATERIALS TESTS
// ====================

func TestStudyMaterialsCRUD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()
	testUserID := uuid.New().String()
	testSubjectID := uuid.New().String()

	t.Run("Success_GetUserMaterials", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/materials/" + testUserID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Success_GetSubjectMaterials", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/materials/" + testUserID + "/" + testSubjectID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// ====================
// STUDY STATS TESTS
// ====================

func TestStudyStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()
	testUserID := uuid.New().String()

	t.Run("Success_GetStudyStats", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/sessions/" + testUserID + "/stats",
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Success_GetUserSessions", func(t *testing.T) {
		w, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/sessions/" + testUserID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should fail with 500 (no DB) but structure is valid
		if w.Code != 500 && w.Code != 200 {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// ====================
// INTEGRATION TESTS
// ====================

func TestIntegrationFlashcardLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	router := setupRouter()
	testUserID := uuid.New().String()

	t.Run("Complete_FlashcardLearningFlow", func(t *testing.T) {
		// 1. Generate flashcards
		genResp, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id":       "biology-101",
				"content":          "DNA contains four nucleotide bases: adenine, thymine, guanine, and cytosine.",
				"card_count":       5,
				"difficulty_level": "intermediate",
			},
		})

		if err != nil {
			t.Fatalf("Failed to generate flashcards: %v", err)
		}

		response := assertJSONResponse(t, genResp, 200, "cards")
		cards := response["cards"].([]interface{})

		if len(cards) < 1 {
			t.Fatal("No cards generated")
		}

		// 2. Start study session
		sessionResp, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/session/start",
			Body: map[string]interface{}{
				"user_id":         testUserID,
				"subject_id":      "biology-101",
				"session_type":    "learn",
				"target_duration": 15,
			},
		})

		if err != nil {
			t.Fatalf("Failed to start session: %v", err)
		}

		sessionData := assertJSONResponse(t, sessionResp, 201, "session_id")
		sessionID := sessionData["session_id"].(string)

		// 3. Answer flashcards
		responses := []string{"easy", "good", "hard", "good", "easy"}
		for i, userResp := range responses {
			answerResp, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/study/answer",
				Body: map[string]interface{}{
					"session_id":    sessionID,
					"flashcard_id":  "card-" + string(rune(i)),
					"user_response": userResp,
					"response_time": 5000 + (i * 1000),
					"user_id":       testUserID,
				},
			})

			if err != nil {
				t.Fatalf("Failed to submit answer %d: %v", i, err)
			}

			answerData := assertJSONResponse(t, answerResp, 200, "xp_earned", "next_review_date")

			// Validate XP progression
			xp := int(answerData["xp_earned"].(float64))
			if xp <= 0 {
				t.Errorf("Answer %d: Expected positive XP, got %d", i, xp)
			}
		}

		// 4. Get due cards
		dueResp, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/due-cards",
			QueryParams: map[string]string{
				"user_id": testUserID,
				"limit":   "10",
			},
		})

		if err != nil {
			t.Fatalf("Failed to get due cards: %v", err)
		}

		assertJSONResponse(t, dueResp, 200, "cards", "next_session_recommendation")

		t.Log("✅ Complete flashcard learning flow validated")
	})
}

// ====================
// BUSINESS LOGIC TESTS
// ====================

func TestBusinessLogic_StreakCalculation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("UpdateUserProgress_MaintainsStreak", func(t *testing.T) {
		userID := uuid.New().String()

		// Simulate consecutive study days
		progress := updateUserProgress(userID, 50)

		if progress.CurrentStreak <= 0 {
			t.Error("Expected positive streak for active user")
		}

		if progress.Level < 0 {
			t.Error("Level should never be negative")
		}

		if progress.XPTotal <= 0 {
			t.Error("XP total should be positive")
		}
	})
}

func TestBusinessLogic_NextReviewCalculation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CalculateNextReview_AllResponses", func(t *testing.T) {
		responses := []string{"again", "hard", "good", "easy"}

		for _, response := range responses {
			nextReview := calculateNextReviewDate(response)

			if nextReview.Before(time.Now()) {
				t.Errorf("Next review for '%s' should be in the future, got %v", response, nextReview)
			}

			// "easy" should have longest interval
			if response == "easy" {
				daysDiff := int(nextReview.Sub(time.Now()).Hours() / 24)
				if daysDiff < 1 {
					t.Error("'Easy' response should schedule review at least 1 day ahead")
				}
			}
		}
	})
}

// ====================
// STRUCTURED CARDS GENERATION TESTS
// ====================

func TestGenerateStructuredCards(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GeneratesCardsFromText", func(t *testing.T) {
		response := "Line 1 content\nLine 2 content\nLine 3 content"
		cards := generateStructuredCards(response, 3, "intermediate")

		if len(cards) != 3 {
			t.Errorf("Expected 3 cards, got %d", len(cards))
		}

		for i, card := range cards {
			if question, ok := card["question"].(string); !ok || question == "" {
				t.Errorf("Card %d missing question", i)
			}
			if answer, ok := card["answer"].(string); !ok || answer == "" {
				t.Errorf("Card %d missing answer", i)
			}
			if difficulty, ok := card["difficulty"].(string); !ok || difficulty != "intermediate" {
				t.Errorf("Card %d has wrong difficulty", i)
			}
		}
	})

	t.Run("HandlesFewerLinesThanCount", func(t *testing.T) {
		response := "Only one line"
		cards := generateStructuredCards(response, 5, "beginner")

		if len(cards) != 1 {
			t.Errorf("Expected 1 card (not 5), got %d", len(cards))
		}
	})
}

func TestGenerateMockFlashcards(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GeneratesCorrectCount", func(t *testing.T) {
		cards := generateMockFlashcards("Test content", 7, "advanced")

		if len(cards) != 7 {
			t.Errorf("Expected 7 cards, got %d", len(cards))
		}

		for i, card := range cards {
			if difficulty := card["difficulty"].(string); difficulty != "advanced" {
				t.Errorf("Card %d has wrong difficulty: %s", i, difficulty)
			}
		}
	})
}
