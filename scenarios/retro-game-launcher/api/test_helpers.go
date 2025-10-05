// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for tests with proper cleanup
func setupTestLogger() func() {
	// Redirect log output to discard during tests to reduce noise
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)

	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment and database
type TestEnvironment struct {
	Server    *APIServer
	DB        *sql.DB
	Router    *mux.Router
	Cleanup   func()
}

// setupTestServer creates a test server with in-memory or test database
func setupTestServer(t *testing.T) *TestEnvironment {
	// Use test database connection
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Build from components for testing
		dbHost := os.Getenv("POSTGRES_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("POSTGRES_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("POSTGRES_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = "postgres"
		}
		dbName := os.Getenv("POSTGRES_DB")
		if dbName == "" {
			dbName = "retro_game_launcher_test"
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Connect to test database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: database not reachable: %v", err)
		return nil
	}

	// Set up test schema (ensure tables exist)
	setupTestSchema(t, db)

	// Mock Ollama URL for testing
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	server := &APIServer{
		db:        db,
		ollamaURL: ollamaURL,
	}

	router := mux.NewRouter()
	setupRoutes(server, router)

	cleanup := func() {
		cleanupTestData(db)
		db.Close()
	}

	return &TestEnvironment{
		Server:  server,
		DB:      db,
		Router:  router,
		Cleanup: cleanup,
	}
}

// setupRoutes configures all API routes (mirrors main.go setup)
func setupRoutes(server *APIServer, router *mux.Router) {
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	api := router.PathPrefix("/api").Subrouter()

	// Games endpoints
	api.HandleFunc("/games", server.getGames).Methods("GET")
	api.HandleFunc("/games", server.createGame).Methods("POST")
	api.HandleFunc("/games/{id}", server.getGame).Methods("GET")
	api.HandleFunc("/games/{id}", server.updateGame).Methods("PUT")
	api.HandleFunc("/games/{id}", server.deleteGame).Methods("DELETE")
	api.HandleFunc("/games/{id}/play", server.recordPlay).Methods("POST")
	api.HandleFunc("/games/{id}/remix", server.createRemix).Methods("POST")

	// Game generation
	api.HandleFunc("/generate", server.generateGame).Methods("POST")
	api.HandleFunc("/generate/status/{id}", server.getGenerationStatus).Methods("GET")

	// High scores
	api.HandleFunc("/games/{id}/scores", server.getHighScores).Methods("GET")
	api.HandleFunc("/games/{id}/scores", server.submitScore).Methods("POST")

	// Search and discovery
	api.HandleFunc("/search/games", server.searchGames).Methods("GET")
	api.HandleFunc("/featured", server.getFeaturedGames).Methods("GET")
	api.HandleFunc("/trending", server.getTrendingGames).Methods("GET")

	// Template and prompt management
	api.HandleFunc("/templates", server.getPromptTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", server.getPromptTemplate).Methods("GET")

	// Users
	api.HandleFunc("/users", server.createUser).Methods("POST")
	api.HandleFunc("/users/{id}", server.getUser).Methods("GET")
}

// setupTestSchema ensures required database tables exist
func setupTestSchema(t *testing.T, db *sql.DB) {
	// Create games table
	createGamesTable := `
	CREATE TABLE IF NOT EXISTS games (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		prompt TEXT NOT NULL,
		description TEXT,
		code TEXT NOT NULL,
		engine VARCHAR(50) NOT NULL,
		author_id UUID,
		parent_game_id UUID,
		is_remix BOOLEAN DEFAULT false,
		thumbnail_url TEXT,
		play_count INTEGER DEFAULT 0,
		remix_count INTEGER DEFAULT 0,
		rating DECIMAL(3, 2),
		tags JSONB DEFAULT '[]'::jsonb,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		published BOOLEAN DEFAULT false
	);`

	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(100) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		subscription_tier VARCHAR(50) DEFAULT 'free',
		games_created_this_month INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login TIMESTAMP
	);`

	createScoresTable := `
	CREATE TABLE IF NOT EXISTS high_scores (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
		user_id UUID,
		score INTEGER NOT NULL,
		play_time_seconds INTEGER,
		achieved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT
	);`

	if _, err := db.Exec(createGamesTable); err != nil {
		t.Logf("Warning: Could not create games table: %v", err)
	}
	if _, err := db.Exec(createUsersTable); err != nil {
		t.Logf("Warning: Could not create users table: %v", err)
	}
	if _, err := db.Exec(createScoresTable); err != nil {
		t.Logf("Warning: Could not create high_scores table: %v", err)
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(db *sql.DB) {
	// Delete test data created during tests
	db.Exec("DELETE FROM high_scores WHERE game_id IN (SELECT id FROM games WHERE title LIKE 'Test%')")
	db.Exec("DELETE FROM games WHERE title LIKE 'Test%' OR prompt LIKE 'Test%'")
	db.Exec("DELETE FROM users WHERE username LIKE 'test%'")
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(router *mux.Router, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonBody, _ := json.Marshal(v)
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}
	}

	// Default content type for POST/PUT
	if (req.Method == "POST" || req.Method == "PUT") && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL vars if using gorilla/mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httpReq)

	return rr
}

// assertJSONResponse validates JSON response structure and status
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}

	body := strings.TrimSpace(rr.Body.String())
	if expectedMessage != "" && !strings.Contains(body, expectedMessage) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMessage, body)
	}
}

// createTestGame creates a test game in the database
func createTestGame(t *testing.T, db *sql.DB, game *Game) *Game {
	t.Helper()

	if game.ID == "" {
		game.ID = uuid.New().String()
	}
	if game.CreatedAt.IsZero() {
		game.CreatedAt = time.Now()
	}
	if game.UpdatedAt.IsZero() {
		game.UpdatedAt = time.Now()
	}

	tagsJSON, _ := json.Marshal(game.Tags)

	query := `
		INSERT INTO games (id, title, prompt, description, code, engine,
		                  author_id, parent_game_id, is_remix, thumbnail_url,
		                  tags, created_at, updated_at, published, play_count, remix_count, rating)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := db.Exec(query,
		game.ID, game.Title, game.Prompt, game.Description, game.Code,
		game.Engine, game.AuthorID, game.ParentGameID, game.IsRemix,
		game.ThumbnailURL, tagsJSON, game.CreatedAt, game.UpdatedAt,
		game.Published, game.PlayCount, game.RemixCount, game.Rating,
	)

	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	return game
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *sql.DB, username, email string) *User {
	t.Helper()

	user := &User{
		ID:                    uuid.New().String(),
		Username:              username,
		Email:                 email,
		SubscriptionTier:      "free",
		GamesCreatedThisMonth: 0,
		CreatedAt:             time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, subscription_tier, games_created_this_month, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.Exec(query,
		user.ID, user.Username, user.Email, user.SubscriptionTier,
		user.GamesCreatedThisMonth, user.CreatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// mockOllamaServer creates a mock Ollama server for testing
func mockOllamaServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/tags") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []string{"codellama", "llama3.2"},
			})
			return
		}

		if strings.HasSuffix(r.URL.Path, "/api/generate") {
			response := OllamaResponse{
				Model:     "codellama",
				CreatedAt: time.Now(),
				Response: "```javascript\nconst canvas = document.getElementById('game');\nconst ctx = canvas.getContext('2d');\nfunction gameLoop() {\n  requestAnimationFrame(gameLoop);\n}\ngameLoop();\n```",
				Done:      true,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(handler)
}

// createTestHighScore creates a test high score in the database
func createTestHighScore(t *testing.T, db *sql.DB, gameID string, userID *string, score int) *HighScore {
	t.Helper()

	highScore := &HighScore{
		ID:         uuid.New().String(),
		GameID:     gameID,
		UserID:     userID,
		Score:      score,
		AchievedAt: time.Now(),
	}

	query := `
		INSERT INTO high_scores (id, game_id, user_id, score, achieved_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query,
		highScore.ID, highScore.GameID, highScore.UserID,
		highScore.Score, highScore.AchievedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create test high score: %v", err)
	}

	return highScore
}

// assertGameEquals validates two games are equal
func assertGameEquals(t *testing.T, expected, actual *Game) {
	t.Helper()

	if expected.ID != actual.ID {
		t.Errorf("Expected game ID %s, got %s", expected.ID, actual.ID)
	}
	if expected.Title != actual.Title {
		t.Errorf("Expected title %s, got %s", expected.Title, actual.Title)
	}
	if expected.Engine != actual.Engine {
		t.Errorf("Expected engine %s, got %s", expected.Engine, actual.Engine)
	}
}

// assertStatusCodeAndContentType validates response status and content type
func assertStatusCodeAndContentType(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedContentType string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, expectedContentType) {
		t.Errorf("Expected content type to contain '%s', got '%s'", expectedContentType, contentType)
	}
}

// parseJSONArray parses JSON array response
func parseJSONArray(t *testing.T, body string, v interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("Failed to parse JSON array: %v. Body: %s", err, body)
	}
}
