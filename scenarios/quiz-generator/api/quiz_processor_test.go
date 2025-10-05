// +build testing

package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNewQuizProcessor(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		processor := NewQuizProcessor(
			env.DB,
			env.Redis,
			"http://localhost:11434",
			"http://localhost:6333",
		)

		if processor == nil {
			t.Fatal("Expected processor to be created")
		}
		if processor.db == nil {
			t.Error("Expected processor to have database connection")
		}
	})

	t.Run("WithoutRedis", func(t *testing.T) {
		processor := NewQuizProcessor(
			env.DB,
			nil, // No Redis
			"http://localhost:11434",
			"http://localhost:6333",
		)

		if processor == nil {
			t.Fatal("Expected processor to be created without Redis")
		}
	})
}

func TestGenerateQuizFromContent(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Success_WithDefaults", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content: "This is sample content about mathematics. The square root of 16 is 4.",
		}

		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		if err != nil {
			t.Fatalf("Failed to generate quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		if quiz.ID == "" {
			t.Error("Expected quiz to have ID")
		}
		if quiz.Title == "" {
			t.Error("Expected quiz to have title")
		}
		if len(quiz.Questions) != 5 { // Default is 5
			t.Errorf("Expected 5 questions (default), got %d", len(quiz.Questions))
		}
	})

	t.Run("Success_CustomQuestionCount", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample content for quiz generation",
			QuestionCount: 3,
			Difficulty:    "hard",
			QuestionTypes: []string{"mcq"},
		}

		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		if err != nil {
			t.Fatalf("Failed to generate quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		if len(quiz.Questions) != 3 {
			t.Errorf("Expected 3 questions, got %d", len(quiz.Questions))
		}
		if quiz.PassingScore != 70 {
			t.Errorf("Expected passing score 70, got %d", quiz.PassingScore)
		}
	})

	t.Run("EmptyContent", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content: "",
		}

		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		// Should still generate with fallback questions
		if err != nil {
			t.Logf("Empty content error (acceptable): %v", err)
		}
		if quiz != nil {
			defer cleanupTestQuiz(t, env, quiz.ID)
		}
	})

	t.Run("VeryLongContent", func(t *testing.T) {
		longContent := string(make([]byte, 50000))
		req := QuizGenerateRequest{
			Content:       longContent,
			QuestionCount: 2,
		}

		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		// Should handle gracefully
		if err != nil && quiz == nil {
			t.Logf("Very long content handled: %v", err)
		}
		if quiz != nil {
			defer cleanupTestQuiz(t, env, quiz.ID)
		}
	})
}

func TestGenerateFallbackQuestions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("MCQ_Questions", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample",
			QuestionCount: 3,
			Difficulty:    "easy",
			QuestionTypes: []string{"mcq"},
		}

		questions := env.Processor.generateFallbackQuestions(req)
		if len(questions) != 3 {
			t.Errorf("Expected 3 questions, got %d", len(questions))
		}

		for i, q := range questions {
			if q.Type != "mcq" {
				t.Errorf("Question %d: expected type mcq, got %s", i, q.Type)
			}
			if len(q.Options) != 4 {
				t.Errorf("Question %d: expected 4 options, got %d", i, len(q.Options))
			}
			if q.Difficulty != "easy" {
				t.Errorf("Question %d: expected difficulty easy, got %s", i, q.Difficulty)
			}
		}
	})

	t.Run("Mixed_QuestionTypes", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample",
			QuestionCount: 6,
			QuestionTypes: []string{"mcq", "true_false", "short_answer"},
		}

		questions := env.Processor.generateFallbackQuestions(req)
		if len(questions) != 6 {
			t.Errorf("Expected 6 questions, got %d", len(questions))
		}

		// Should cycle through question types
		typeCount := make(map[string]int)
		for _, q := range questions {
			typeCount[q.Type]++
		}

		if typeCount["mcq"] == 0 {
			t.Error("Expected at least one MCQ")
		}
		if typeCount["true_false"] == 0 {
			t.Error("Expected at least one true/false")
		}
	})
}

func TestStructureQuestions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		generated := []GeneratedQuestion{
			{
				Type:          "mcq",
				Question:      "Test question?",
				Options:       []string{"A) 1", "B) 2"},
				CorrectAnswer: "B",
				Explanation:   "Explanation",
				Difficulty:    "easy",
			},
			{
				Type:          "mcq",
				Question:      "Hard question?",
				CorrectAnswer: "A",
				Difficulty:    "hard",
			},
		}

		questions := env.Processor.structureQuestions(generated)
		if len(questions) != 2 {
			t.Errorf("Expected 2 questions, got %d", len(questions))
		}

		// Check point assignment
		if questions[0].Points != 1 { // easy = 1 point
			t.Errorf("Expected 1 point for easy question, got %d", questions[0].Points)
		}
		if questions[1].Points != 3 { // hard = 3 points
			t.Errorf("Expected 3 points for hard question, got %d", questions[1].Points)
		}

		// Check order indices
		if questions[0].OrderIndex != 1 {
			t.Errorf("Expected order index 1, got %d", questions[0].OrderIndex)
		}
		if questions[1].OrderIndex != 2 {
			t.Errorf("Expected order index 2, got %d", questions[1].OrderIndex)
		}

		// Check IDs are generated
		if questions[0].ID == "" {
			t.Error("Expected question to have ID")
		}
	})
}

func TestSaveQuizToDB(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		quiz := &Quiz{
			ID:           "test-save-quiz",
			Title:        "Test Quiz",
			Description:  "Test Description",
			TimeLimit:    600,
			PassingScore: 70,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Questions: []Question{
				{
					ID:            "q1",
					Type:          "mcq",
					QuestionText:  "Question 1",
					Options:       []string{"A", "B"},
					CorrectAnswer: "A",
					Difficulty:    "easy",
					Points:        1,
				},
			},
		}

		err := env.Processor.saveQuizToDB(ctx, quiz)
		if err != nil {
			t.Fatalf("Failed to save quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Verify quiz was saved
		var title string
		err = env.DB.QueryRow(ctx,
			"SELECT title FROM quiz_generator.quizzes WHERE id = $1",
			quiz.ID).Scan(&title)
		if err != nil {
			t.Errorf("Failed to retrieve saved quiz: %v", err)
		}
		if title != "Test Quiz" {
			t.Errorf("Expected title 'Test Quiz', got '%s'", title)
		}
	})

	t.Run("WithMultipleQuestions", func(t *testing.T) {
		quiz := &Quiz{
			ID:           "test-multi-q",
			Title:        "Multi Question Quiz",
			TimeLimit:    600,
			PassingScore: 70,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Questions: []Question{
				{ID: "q1", Type: "mcq", QuestionText: "Q1", Difficulty: "easy", Points: 1},
				{ID: "q2", Type: "true_false", QuestionText: "Q2", Difficulty: "medium", Points: 2},
				{ID: "q3", Type: "short_answer", QuestionText: "Q3", Difficulty: "hard", Points: 3},
			},
		}

		err := env.Processor.saveQuizToDB(ctx, quiz)
		if err != nil {
			t.Fatalf("Failed to save quiz with multiple questions: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Verify all questions were saved
		var count int
		err = env.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM quiz_generator.questions WHERE quiz_id = $1",
			quiz.ID).Scan(&count)
		if err != nil {
			t.Errorf("Failed to count questions: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 questions saved, got %d", count)
		}
	})
}

func TestCacheQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Success_WithRedis", func(t *testing.T) {
		if env.Redis == nil {
			t.Skip("Redis not available")
		}

		quiz := &Quiz{
			ID:    "test-cache",
			Title: "Cached Quiz",
		}

		err := env.Processor.cacheQuiz(ctx, quiz)
		if err != nil {
			t.Errorf("Failed to cache quiz: %v", err)
		}

		// Verify cache
		cached, err := env.Redis.Get(ctx, "quiz:test-cache").Result()
		if err != nil {
			t.Errorf("Failed to retrieve cached quiz: %v", err)
		}

		var cachedQuiz Quiz
		if err := json.Unmarshal([]byte(cached), &cachedQuiz); err != nil {
			t.Errorf("Failed to unmarshal cached quiz: %v", err)
		}

		if cachedQuiz.Title != "Cached Quiz" {
			t.Errorf("Expected cached title 'Cached Quiz', got '%s'", cachedQuiz.Title)
		}
	})

	t.Run("WithoutRedis", func(t *testing.T) {
		// Create processor without Redis
		processor := NewQuizProcessor(env.DB, nil, "", "")

		quiz := &Quiz{
			ID:    "test-no-redis",
			Title: "No Redis Quiz",
		}

		err := processor.cacheQuiz(ctx, quiz)
		if err != nil {
			t.Errorf("Expected no error without Redis, got: %v", err)
		}
	})
}

func TestProcessorGetQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Success_FromDatabase", func(t *testing.T) {
		// Create test quiz
		originalQuiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, originalQuiz.ID)

		// Retrieve quiz
		quiz, err := env.Processor.getQuiz(ctx, originalQuiz.ID)
		if err != nil {
			t.Fatalf("Failed to get quiz: %v", err)
		}

		if quiz.ID != originalQuiz.ID {
			t.Errorf("Expected ID %s, got %s", originalQuiz.ID, quiz.ID)
		}
		if quiz.Title != originalQuiz.Title {
			t.Errorf("Expected title %s, got %s", originalQuiz.Title, quiz.Title)
		}
		if len(quiz.Questions) != len(originalQuiz.Questions) {
			t.Errorf("Expected %d questions, got %d",
				len(originalQuiz.Questions), len(quiz.Questions))
		}
	})

	t.Run("Success_FromCache", func(t *testing.T) {
		if env.Redis == nil {
			t.Skip("Redis not available")
		}

		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Cache the quiz
		env.Processor.cacheQuiz(ctx, quiz)

		// Retrieve from cache
		retrieved, err := env.Processor.getQuiz(ctx, quiz.ID)
		if err != nil {
			t.Fatalf("Failed to get cached quiz: %v", err)
		}

		if retrieved.ID != quiz.ID {
			t.Errorf("Expected cached ID %s, got %s", quiz.ID, retrieved.ID)
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		_, err := env.Processor.getQuiz(ctx, "nonexistent-quiz-id")
		if err == nil {
			t.Error("Expected error for non-existent quiz")
		}
	})
}

func TestGradeQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("AllCorrect", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{
				{QuestionID: "q1", Answer: "B"},
				{QuestionID: "q2", Answer: "true"},
			},
			TimeTaken: 120,
		}

		result, err := env.Processor.GradeQuiz(ctx, quiz.ID, submission)
		if err != nil {
			t.Fatalf("Failed to grade quiz: %v", err)
		}

		if result.Score != 2 {
			t.Errorf("Expected score 2 (both correct), got %d", result.Score)
		}
		if result.Percentage != 100.0 {
			t.Errorf("Expected 100%%, got %.2f%%", result.Percentage)
		}
		if !result.Passed {
			t.Error("Expected quiz to be passed")
		}
	})

	t.Run("PartialCorrect", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{
				{QuestionID: "q1", Answer: "B"},      // Correct
				{QuestionID: "q2", Answer: "false"},  // Incorrect
			},
			TimeTaken: 120,
		}

		result, err := env.Processor.GradeQuiz(ctx, quiz.ID, submission)
		if err != nil {
			t.Fatalf("Failed to grade quiz: %v", err)
		}

		if result.Score != 1 {
			t.Errorf("Expected score 1 (one correct), got %d", result.Score)
		}
		if result.Percentage != 50.0 {
			t.Errorf("Expected 50%%, got %.2f%%", result.Percentage)
		}
	})

	t.Run("AllIncorrect", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{
				{QuestionID: "q1", Answer: "A"},      // Incorrect
				{QuestionID: "q2", Answer: "false"},  // Incorrect
			},
		}

		result, err := env.Processor.GradeQuiz(ctx, quiz.ID, submission)
		if err != nil {
			t.Fatalf("Failed to grade quiz: %v", err)
		}

		if result.Score != 0 {
			t.Errorf("Expected score 0, got %d", result.Score)
		}
		if !result.Passed && result.Percentage < 70 {
			// Expected behavior
		}
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{},
		}

		_, err := env.Processor.GradeQuiz(ctx, "nonexistent", submission)
		if err == nil {
			t.Error("Expected error for non-existent quiz")
		}
	})
}

func TestCheckAnswer(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ExactMatch", func(t *testing.T) {
		question := Question{
			CorrectAnswer: "B",
		}

		if !env.Processor.checkAnswer(question, "B") {
			t.Error("Expected exact match to be correct")
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		question := Question{
			CorrectAnswer: "True",
		}

		if !env.Processor.checkAnswer(question, "true") {
			t.Error("Expected case-insensitive match")
		}
		if !env.Processor.checkAnswer(question, "TRUE") {
			t.Error("Expected case-insensitive match")
		}
	})

	t.Run("Incorrect", func(t *testing.T) {
		question := Question{
			CorrectAnswer: "B",
		}

		if env.Processor.checkAnswer(question, "A") {
			t.Error("Expected incorrect answer to fail")
		}
	})

	t.Run("NumericAnswer", func(t *testing.T) {
		question := Question{
			CorrectAnswer: 42,
		}

		if !env.Processor.checkAnswer(question, 42) {
			t.Error("Expected numeric match")
		}
		if !env.Processor.checkAnswer(question, "42") {
			t.Error("Expected string-numeric match")
		}
	})
}

func TestGetAverageDifficulty(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AllEasy", func(t *testing.T) {
		questions := []Question{
			{Difficulty: "easy"},
			{Difficulty: "easy"},
			{Difficulty: "easy"},
		}

		avg := env.Processor.getAverageDifficulty(questions)
		if avg != "easy" {
			t.Errorf("Expected 'easy', got '%s'", avg)
		}
	})

	t.Run("Mixed", func(t *testing.T) {
		questions := []Question{
			{Difficulty: "easy"},
			{Difficulty: "medium"},
			{Difficulty: "medium"},
			{Difficulty: "hard"},
		}

		avg := env.Processor.getAverageDifficulty(questions)
		if avg != "medium" {
			t.Errorf("Expected 'medium' (most common), got '%s'", avg)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		questions := []Question{}

		avg := env.Processor.getAverageDifficulty(questions)
		if avg != "medium" {
			t.Errorf("Expected default 'medium', got '%s'", avg)
		}
	})
}

func TestGenerateEmbedding(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		embedding := env.Processor.generateEmbedding("test text")
		if len(embedding) != 384 {
			t.Errorf("Expected 384-dimensional vector, got %d", len(embedding))
		}
	})

	t.Run("EmptyText", func(t *testing.T) {
		embedding := env.Processor.generateEmbedding("")
		if len(embedding) != 384 {
			t.Errorf("Expected 384-dimensional vector for empty text, got %d", len(embedding))
		}
	})
}
