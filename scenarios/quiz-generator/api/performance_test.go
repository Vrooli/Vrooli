// +build testing

package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestConcurrentQuizCreation tests creating multiple quizzes concurrently
func TestConcurrentQuizCreation(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("10_Concurrent_Creates", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		quizIDs := make(chan string, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				quiz := Quiz{
					Title:        "Concurrent Quiz",
					Description:  "Test concurrent creation",
					TimeLimit:    600,
					PassingScore: 70,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
					Questions: []Question{
						{
							ID:            "q1",
							Type:          "mcq",
							QuestionText:  "Test question",
							Difficulty:    "easy",
							Points:        1,
						},
					},
				}

				w := makeHTTPRequest("POST", "/api/v1/quiz", quiz, env.Router)
				if w.Code != 201 {
					errors <- nil
					return
				}

				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
					if id, ok := resp["id"].(string); ok {
						quizIDs <- id
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(quizIDs)

		elapsed := time.Since(start)

		// Cleanup
		for id := range quizIDs {
			cleanupTestQuiz(t, env, id)
		}

		t.Logf("Created %d quizzes concurrently in %v", concurrency, elapsed)

		// Performance expectation: should complete in reasonable time
		if elapsed > 10*time.Second {
			t.Logf("Warning: Concurrent creation took %v (>10s)", elapsed)
		}
	})
}

// TestConcurrentQuizRetrieval tests reading quizzes concurrently
func TestConcurrentQuizRetrieval(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test quiz
	quiz := createTestQuiz(t, env)
	defer cleanupTestQuiz(t, env, quiz.ID)

	t.Run("100_Concurrent_Reads", func(t *testing.T) {
		concurrency := 100
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID, nil, env.Router)
				if w.Code == 200 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Completed %d concurrent reads in %v (%d successful)",
			concurrency, elapsed, successCount)

		if successCount < concurrency {
			t.Errorf("Expected all %d reads to succeed, got %d", concurrency, successCount)
		}

		// Performance expectation
		avgTime := elapsed / time.Duration(concurrency)
		if avgTime > 100*time.Millisecond {
			t.Logf("Warning: Average read time %v (>100ms)", avgTime)
		}
	})
}

// TestQuizGenerationPerformance tests quiz generation performance
func TestQuizGenerationPerformance(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Small_Quiz_Generation", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "This is sample content for quiz generation testing.",
			QuestionCount: 3,
			Difficulty:    "easy",
		}

		start := time.Now()
		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to generate quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		t.Logf("Generated 3-question quiz in %v", elapsed)

		// Should be fast with fallback
		if elapsed > 5*time.Second {
			t.Logf("Warning: Quiz generation took %v (>5s)", elapsed)
		}
	})

	t.Run("Large_Quiz_Generation", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample content with more detail for generating a larger quiz with multiple questions.",
			QuestionCount: 10,
			Difficulty:    "medium",
		}

		start := time.Now()
		quiz, err := env.Processor.GenerateQuizFromContent(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to generate large quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		t.Logf("Generated 10-question quiz in %v", elapsed)

		if elapsed > 15*time.Second {
			t.Logf("Warning: Large quiz generation took %v (>15s)", elapsed)
		}
	})
}

// TestQuizSubmissionPerformance tests grading performance
func TestQuizSubmissionPerformance(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("Grade_Large_Quiz", func(t *testing.T) {
		// Create a quiz with many questions
		quiz := &Quiz{
			ID:           "perf-test-quiz",
			Title:        "Performance Test Quiz",
			TimeLimit:    1200,
			PassingScore: 70,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Add 50 questions
		for i := 1; i <= 50; i++ {
			quiz.Questions = append(quiz.Questions, Question{
				ID:            string(rune('a' + i)),
				Type:          "mcq",
				QuestionText:  "Question",
				CorrectAnswer: "A",
				Difficulty:    "medium",
				Points:        2,
				OrderIndex:    i,
			})
		}

		err := env.Processor.saveQuizToDB(ctx, quiz)
		if err != nil {
			t.Fatalf("Failed to create performance test quiz: %v", err)
		}
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Submit answers
		submission := QuizSubmitRequest{
			Responses:  []QuestionResponse{},
			TimeTaken: 600,
		}

		for i := 1; i <= 50; i++ {
			submission.Responses = append(submission.Responses, QuestionResponse{
				QuestionID: string(rune('a' + i)),
				Answer:     "A",
			})
		}

		start := time.Now()
		result, err := env.Processor.GradeQuiz(ctx, quiz.ID, submission)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to grade quiz: %v", err)
		}

		t.Logf("Graded 50-question quiz in %v (score: %d)", elapsed, result.Score)

		if elapsed > 2*time.Second {
			t.Logf("Warning: Grading took %v (>2s)", elapsed)
		}
	})
}

// TestDatabaseQueryPerformance tests database operation performance
func TestDatabaseQueryPerformance(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("List_Many_Quizzes", func(t *testing.T) {
		// Create multiple quizzes
		quizIDs := []string{}
		for i := 0; i < 20; i++ {
			quiz := createTestQuiz(t, env)
			quizIDs = append(quizIDs, quiz.ID)
		}
		defer func() {
			for _, id := range quizIDs {
				cleanupTestQuiz(t, env, id)
			}
		}()

		start := time.Now()
		w := makeHTTPRequest("GET", "/api/v1/quizzes?limit=100", nil, env.Router)
		elapsed := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Listed quizzes in %v", elapsed)

		if elapsed > 500*time.Millisecond {
			t.Logf("Warning: Listing took %v (>500ms)", elapsed)
		}
	})
}

// TestCachePerformance tests Redis cache performance
func TestCachePerformance(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.Redis == nil {
		t.Skip("Redis not available")
	}

	t.Run("Cache_Hit_vs_Database", func(t *testing.T) {
		ctx := context.Background()
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Cache the quiz
		_ = env.Processor.cacheQuiz(ctx, quiz)

		// Measure cache hit
		start := time.Now()
		_, err := env.Processor.getQuiz(ctx, quiz.ID)
		cacheTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get cached quiz: %v", err)
		}

		// Clear cache
		env.Redis.Del(ctx, "quiz:"+quiz.ID)

		// Measure database hit
		start = time.Now()
		_, err = env.Processor.getQuiz(ctx, quiz.ID)
		dbTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get quiz from DB: %v", err)
		}

		t.Logf("Cache hit: %v, DB hit: %v, Speedup: %.2fx",
			cacheTime, dbTime, float64(dbTime)/float64(cacheTime))

		// Cache should be faster
		if cacheTime > dbTime {
			t.Logf("Note: Cache was slower than DB (%v vs %v)", cacheTime, dbTime)
		}
	})
}

// BenchmarkQuizCreation benchmarks quiz creation
func BenchmarkQuizCreation(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	quiz := Quiz{
		Title:        "Benchmark Quiz",
		TimeLimit:    600,
		PassingScore: 70,
		Questions: []Question{
			{
				Type:          "mcq",
				QuestionText:  "Test",
				Difficulty:    "easy",
				Points:        1,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest("POST", "/api/v1/quiz", quiz, env.Router)
		if w.Code == 201 {
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if id, ok := resp["id"].(string); ok {
				// Cleanup after benchmark
				cleanupTestQuiz(&testing.T{}, env, id)
			}
		}
	}
}

// BenchmarkQuizRetrieval benchmarks quiz retrieval
func BenchmarkQuizRetrieval(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	quiz := createTestQuiz(&testing.T{}, env)
	defer cleanupTestQuiz(&testing.T{}, env, quiz.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID, nil, env.Router)
	}
}
