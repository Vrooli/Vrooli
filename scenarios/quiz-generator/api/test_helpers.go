// +build testing

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB            *pgxpool.Pool
	Redis         *redis.Client
	Router        *gin.Engine
	Processor     *QuizProcessor
	Cleanup       func()
	OriginalDB    *pgxpool.Pool
	OriginalRedis *redis.Client
	OriginalLogger *logrus.Logger
}

// setupTestLogger initializes a test logger that suppresses output during tests
func setupTestLogger() func() {
	originalLogger := logger
	logger = logrus.New()
	logger.SetOutput(io.Discard) // Suppress test output
	logger.SetLevel(logrus.ErrorLevel)

	return func() { logger = originalLogger }
}

// setupTestEnvironment creates a complete test environment with database and router
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Setup test logger
	cleanupLogger := setupTestLogger()

	// Save originals
	originalDB := db
	originalRedis := redisClient
	originalProcessor := quizProcessor

	// Create test database connection
	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// Use default test database
		dbHost := os.Getenv("POSTGRES_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("POSTGRES_PORT")
		if dbPort == "" {
			dbPort = "5433"
		}
		dbURL = "postgres://vrooli:vrooli@" + dbHost + ":" + dbPort + "/vrooli?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("Failed to parse database URL: %v", err)
	}

	config.MaxConns = 5
	config.MinConns = 1

	testDB, err := pgxpool.New(ctx, config.ConnString())
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ping database with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := testDB.Ping(pingCtx); err != nil {
		testDB.Close()
		t.Skipf("Database not available (skipping test): %v", err)
	}

	// Setup test schema
	_, err = testDB.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS quiz_generator;

		CREATE TABLE IF NOT EXISTS quiz_generator.quizzes (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			difficulty TEXT,
			time_limit INTEGER,
			passing_score INTEGER,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS quiz_generator.questions (
			id TEXT PRIMARY KEY,
			quiz_id TEXT REFERENCES quiz_generator.quizzes(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			question_text TEXT NOT NULL,
			options JSONB,
			correct_answer JSONB,
			explanation TEXT,
			difficulty TEXT,
			points INTEGER,
			order_index INTEGER
		);

		CREATE TABLE IF NOT EXISTS quiz_generator.quiz_results (
			id TEXT PRIMARY KEY,
			quiz_id TEXT REFERENCES quiz_generator.quizzes(id) ON DELETE CASCADE,
			score INTEGER,
			percentage DOUBLE PRECISION,
			passed BOOLEAN,
			responses JSONB,
			time_taken INTEGER,
			submitted_at TIMESTAMP
		);
	`)
	if err != nil {
		testDB.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Setup Redis (optional - can be nil)
	var testRedis *redis.Client
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	testRedis = redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           1, // Use DB 1 for tests
		MaxRetries:   1,
		DialTimeout:  2 * time.Second,
	})

	// Test Redis connection (but don't fail if unavailable)
	if err := testRedis.Ping(ctx).Err(); err != nil {
		testRedis.Close()
		testRedis = nil
	}

	// Set global variables for handlers
	db = testDB
	redisClient = testRedis

	// Create test processor
	testProcessor := NewQuizProcessor(
		testDB,
		testRedis,
		"http://localhost:11434", // Ollama URL
		"http://localhost:6333",  // Qdrant URL
	)
	quizProcessor = testProcessor

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	env := &TestEnvironment{
		DB:            testDB,
		Redis:         testRedis,
		Router:        router,
		Processor:     testProcessor,
		OriginalDB:    originalDB,
		OriginalRedis: originalRedis,
		OriginalLogger: logger,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec(context.Background(), "DROP SCHEMA IF EXISTS quiz_generator CASCADE")
			testDB.Close()
			if testRedis != nil {
				testRedis.FlushDB(context.Background())
				testRedis.Close()
			}

			// Restore originals
			db = originalDB
			redisClient = originalRedis
			quizProcessor = originalProcessor
			cleanupLogger()
		},
	}

	return env
}

// setupRouter creates a test router with all routes
func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthCheck)
	router.GET("/ready", healthCheck)
	router.GET("/api/health", healthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/quiz/generate", generateQuiz)
		v1.POST("/quiz", createQuiz)
		v1.PUT("/quiz/:id", updateQuiz)
		v1.DELETE("/quiz/:id", deleteQuiz)
		v1.GET("/quiz/:id", getQuiz)
		v1.GET("/quizzes", listQuizzes)
		v1.POST("/quiz/:id/submit", submitQuiz)
		v1.GET("/quiz/:id/export", exportQuiz)
		v1.POST("/question-bank/search", searchQuestions)
		v1.GET("/stats", getStats)
		v1.GET("/quiz/:id/take", getQuizForTaking)
		v1.POST("/quiz/:id/answer/:questionId", submitSingleAnswer)
	}

	return router
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(method, path string, body interface{}, router *gin.Engine) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// assertJSONResponse validates a JSON response matches expected structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateBody func(map[string]interface{})) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if validateBody != nil {
		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
			return
		}
		validateBody(body)
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Errorf("Failed to parse error JSON: %v", err)
		return
	}

	if _, hasError := body["error"]; !hasError {
		t.Errorf("Expected error field in response, got: %v", body)
	}
}

// createTestQuiz creates a quiz in the database for testing
func createTestQuiz(t *testing.T, env *TestEnvironment) *Quiz {
	quiz := &Quiz{
		ID:           "test-quiz-" + time.Now().Format("20060102150405"),
		Title:        "Test Quiz",
		Description:  "A test quiz for unit testing",
		TimeLimit:    600,
		PassingScore: 70,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Questions: []Question{
			{
				ID:            "q1",
				Type:          "mcq",
				QuestionText:  "What is 2+2?",
				Options:       []string{"A) 3", "B) 4", "C) 5", "D) 6"},
				CorrectAnswer: "B",
				Explanation:   "2+2 equals 4",
				Difficulty:    "easy",
				Points:        1,
				OrderIndex:    1,
			},
			{
				ID:            "q2",
				Type:          "true_false",
				QuestionText:  "The sky is blue",
				CorrectAnswer: "true",
				Explanation:   "The sky appears blue due to Rayleigh scattering",
				Difficulty:    "easy",
				Points:        1,
				OrderIndex:    2,
			},
		},
	}

	if err := env.Processor.saveQuizToDB(context.Background(), quiz); err != nil {
		t.Fatalf("Failed to create test quiz: %v", err)
	}

	return quiz
}

// cleanupTestQuiz removes a quiz from the database
func cleanupTestQuiz(t *testing.T, env *TestEnvironment, quizID string) {
	_, err := env.DB.Exec(context.Background(),
		"DELETE FROM quiz_generator.quizzes WHERE id = $1", quizID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test quiz: %v", err)
	}
}
