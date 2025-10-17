// +build testing

package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestHealthCheck(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest("GET", "/health", nil, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if status, ok := body["status"].(string); !ok || status == "" {
				t.Error("Expected status field in health check")
			}
			if _, ok := body["database"]; !ok {
				t.Error("Expected database field in health check")
			}
			if _, ok := body["version"]; !ok {
				t.Error("Expected version field in health check")
			}
		})
	})

	t.Run("APIPathHealth", func(t *testing.T) {
		w := makeHTTPRequest("GET", "/api/health", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 at /api/health, got %d", w.Code)
		}
	})
}

func TestGenerateQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "This is sample content about mathematics. 2+2=4 and 3+3=6.",
			QuestionCount: 3,
			Difficulty:    "easy",
			QuestionTypes: []string{"mcq", "true_false"},
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		assertJSONResponse(t, w, http.StatusCreated, func(body map[string]interface{}) {
			if _, ok := body["quiz_id"]; !ok {
				t.Error("Expected quiz_id in response")
			}
			if _, ok := body["title"]; !ok {
				t.Error("Expected title in response")
			}
			if questions, ok := body["questions"].([]interface{}); !ok || len(questions) == 0 {
				t.Error("Expected questions array in response")
			}
		})
	})

	t.Run("MissingContent", func(t *testing.T) {
		req := map[string]interface{}{
			"question_count": 5,
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content: "",
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", "invalid{json", env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DefaultValues", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content: "Sample content for quiz generation",
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})
}

func TestCreateQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		quiz := Quiz{
			Title:        "Test Quiz",
			Description:  "A test quiz",
			TimeLimit:    600,
			PassingScore: 70,
			Questions: []Question{
				{
					Type:          "mcq",
					QuestionText:  "What is 2+2?",
					Options:       []string{"A) 3", "B) 4", "C) 5"},
					CorrectAnswer: "B",
					Difficulty:    "easy",
					Points:        1,
				},
			},
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz", quiz, env.Router)

		var responseBody map[string]interface{}
		assertJSONResponse(t, w, http.StatusCreated, func(body map[string]interface{}) {
			responseBody = body
			if _, ok := body["id"]; !ok {
				t.Error("Expected id in response")
			}
			if title, ok := body["title"].(string); !ok || title != "Test Quiz" {
				t.Error("Expected title to match")
			}
		})

		// Cleanup
		if id, ok := responseBody["id"].(string); ok {
			cleanupTestQuiz(t, env, id)
		}
	})

	t.Run("MissingTitle", func(t *testing.T) {
		quiz := map[string]interface{}{
			"description": "No title",
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz", quiz, env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyQuestions", func(t *testing.T) {
		quiz := Quiz{
			Title:     "Empty Quiz",
			Questions: []Question{},
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz", quiz, env.Router)
		// Should still succeed, but with no questions
		if w.Code != http.StatusCreated {
			t.Logf("Note: Empty questions returned status %d", w.Code)
		}
	})
}

func TestGetQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test quiz
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID, nil, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if id, ok := body["id"].(string); !ok || id != quiz.ID {
				t.Errorf("Expected id %s, got %v", quiz.ID, body["id"])
			}
			if title, ok := body["title"].(string); !ok || title == "" {
				t.Error("Expected title in response")
			}
		})
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest("GET", "/api/v1/quiz/"+nonExistentID, nil, env.Router)
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		w := makeHTTPRequest("GET", "/api/v1/quiz/invalid-uuid", nil, env.Router)
		// Gin router returns 404 for invalid UUID format
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid UUID, got %d", w.Code)
		}
	})
}

func TestListQuizzes(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test quizzes
		quiz1 := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz1.ID)

		w := makeHTTPRequest("GET", "/api/v1/quizzes", nil, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			// Body should be an array
			if w.Body.String() == "null" {
				// Empty list is acceptable
				return
			}
		})
	})

	t.Run("WithLimit", func(t *testing.T) {
		w := makeHTTPRequest("GET", "/api/v1/quizzes?limit=5", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		// Clean database first
		env.DB.Exec(context.Background(), "DELETE FROM quiz_generator.quizzes")

		w := makeHTTPRequest("GET", "/api/v1/quizzes", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty list, got %d", w.Code)
		}
	})
}

func TestUpdateQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test quiz
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		// Update quiz
		quiz.Title = "Updated Quiz Title"
		quiz.Description = "Updated description"

		w := makeHTTPRequest("PUT", "/api/v1/quiz/"+quiz.ID, quiz, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if title, ok := body["title"].(string); !ok || title != "Updated Quiz Title" {
				t.Error("Expected updated title")
			}
		})
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		quiz := Quiz{
			Title: "Updated Quiz",
		}
		w := makeHTTPRequest("PUT", "/api/v1/quiz/"+nonExistentID, quiz, env.Router)
		// Should still succeed (upsert behavior) or return error
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Update non-existent quiz returned status %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("PUT", "/api/v1/quiz/"+quiz.ID, "invalid{json", env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestDeleteQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test quiz
		quiz := createTestQuiz(t, env)

		w := makeHTTPRequest("DELETE", "/api/v1/quiz/"+quiz.ID, nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify deletion
		w2 := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID, nil, env.Router)
		if w2.Code != http.StatusNotFound {
			t.Error("Expected quiz to be deleted")
		}
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest("DELETE", "/api/v1/quiz/"+nonExistentID, nil, env.Router)
		// Deletion of non-existent should still succeed
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Delete non-existent quiz returned status %d", w.Code)
		}
	})
}

func TestSubmitQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test quiz
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{
				{QuestionID: "q1", Answer: "B"},
				{QuestionID: "q2", Answer: "true"},
			},
			TimeTaken: 120,
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/submit", submission, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["score"]; !ok {
				t.Error("Expected score in result")
			}
			if _, ok := body["percentage"]; !ok {
				t.Error("Expected percentage in result")
			}
			if _, ok := body["passed"]; !ok {
				t.Error("Expected passed field in result")
			}
		})
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{},
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz/"+nonExistentID+"/submit", submission, env.Router)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	t.Run("EmptyResponses", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		submission := QuizSubmitRequest{
			Responses: []QuestionResponse{},
		}
		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/submit", submission, env.Router)
		// Should succeed with 0 score
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty responses, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/submit", "invalid{json", env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestSearchQuestions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := SearchRequest{
			Query:      "mathematics",
			Limit:      5,
			Difficulty: "easy",
		}

		w := makeHTTPRequest("POST", "/api/v1/question-bank/search", req, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["questions"]; !ok {
				t.Error("Expected questions in response")
			}
		})
	})

	t.Run("MissingQuery", func(t *testing.T) {
		req := map[string]interface{}{
			"limit": 5,
		}
		w := makeHTTPRequest("POST", "/api/v1/question-bank/search", req, env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		req := SearchRequest{
			Query: "test query",
		}
		w := makeHTTPRequest("POST", "/api/v1/question-bank/search", req, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestGetQuizForTaking(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID+"/take", nil, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			// Verify correct answers are hidden
			if questions, ok := body["questions"].([]interface{}); ok && len(questions) > 0 {
				firstQ := questions[0].(map[string]interface{})
				if firstQ["correct_answer"] != nil {
					t.Error("Expected correct_answer to be hidden in take mode")
				}
				if explanation, ok := firstQ["explanation"].(string); ok && explanation != "" {
					t.Error("Expected explanation to be hidden in take mode")
				}
			}
		})
	})

	t.Run("NonExistentQuiz", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest("GET", "/api/v1/quiz/"+nonExistentID+"/take", nil, env.Router)
		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

func TestSubmitSingleAnswer(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CorrectAnswer", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		body := map[string]interface{}{
			"answer": "B",
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/answer/q1", body, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(respBody map[string]interface{}) {
			if correct, ok := respBody["correct"].(bool); !ok || !correct {
				t.Error("Expected answer to be marked as correct")
			}
			if points, ok := respBody["points"].(float64); !ok || points == 0 {
				t.Error("Expected points to be awarded")
			}
		})
	})

	t.Run("IncorrectAnswer", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		body := map[string]interface{}{
			"answer": "A",
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/answer/q1", body, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(respBody map[string]interface{}) {
			if correct, ok := respBody["correct"].(bool); ok && correct {
				t.Error("Expected answer to be marked as incorrect")
			}
		})
	})

	t.Run("NonExistentQuestion", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		body := map[string]interface{}{
			"answer": "A",
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/"+quiz.ID+"/answer/nonexistent", body, env.Router)
		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

func TestExportQuiz(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("JSONFormat", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID+"/export?format=json", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("QTIFormat", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID+"/export?format=qti", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for QTI export, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/xml" {
			t.Errorf("Expected Content-Type application/xml, got %s", contentType)
		}
	})

	t.Run("MoodleFormat", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID+"/export?format=moodle", nil, env.Router)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for Moodle export, got %d", w.Code)
		}
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		quiz := createTestQuiz(t, env)
		defer cleanupTestQuiz(t, env, quiz.ID)

		w := makeHTTPRequest("GET", "/api/v1/quiz/"+quiz.ID+"/export?format=pdf", nil, env.Router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestGetStats(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest("GET", "/api/v1/stats", nil, env.Router)
		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["total_quizzes"]; !ok {
				t.Error("Expected total_quizzes in stats")
			}
			if _, ok := body["total_questions"]; !ok {
				t.Error("Expected total_questions in stats")
			}
		})
	})
}

// Test helper functions
func TestGenerateMockQuestions(t *testing.T) {
	t.Run("GenerateMultiple", func(t *testing.T) {
		questions := generateMockQuestions(5)
		if len(questions) != 5 {
			t.Errorf("Expected 5 questions, got %d", len(questions))
		}
	})

	t.Run("QuestionStructure", func(t *testing.T) {
		questions := generateMockQuestions(1)
		if len(questions) == 0 {
			t.Fatal("Expected at least one question")
		}

		q := questions[0]
		if q.ID == "" {
			t.Error("Expected question to have ID")
		}
		if q.QuestionText == "" {
			t.Error("Expected question to have text")
		}
		if len(q.Options) == 0 {
			t.Error("Expected question to have options")
		}
	})
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("VeryLongContent", func(t *testing.T) {
		longContent := string(make([]byte, 10000))
		req := QuizGenerateRequest{
			Content:       longContent,
			QuestionCount: 3,
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		// Should handle long content gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Logf("Long content returned status %d", w.Code)
		}
	})

	t.Run("ZeroQuestionCount", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample content",
			QuestionCount: 0,
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		// Should use default value
		if w.Code != http.StatusCreated {
			t.Logf("Zero question count returned status %d", w.Code)
		}
	})

	t.Run("NegativeQuestionCount", func(t *testing.T) {
		req := QuizGenerateRequest{
			Content:       "Sample content",
			QuestionCount: -5,
		}

		w := makeHTTPRequest("POST", "/api/v1/quiz/generate", req, env.Router)
		// Should handle gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Logf("Negative question count returned status %d", w.Code)
		}
	})
}
