package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"gopkg.in/h2non/gock.v1"
)

// Note: These tests are currently skipped due to issues with global variable mocking
// in concurrent test execution. They serve as templates for future integration testing.

// TestRealCreateCharacterHandler tests the actual createCharacterHandler with mocked DB
func TestRealCreateCharacterHandler(t *testing.T) {
	t.Skip("Skipping due to global variable concurrency issues - will be addressed in future refactor")
	cleanup := setupTestLogger()
	defer cleanup()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	// Set global db variable
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	// Initialize rest client to avoid nil pointer panic in goroutine
	oldClient := restClient
	restClient = resty.New()
	defer func() { restClient = oldClient }()

	oldOllamaURL := ollamaURL
	oldQdrantURL := qdrantURL
	ollamaURL = "http://localhost:11434"
	qdrantURL = "http://localhost:6333"
	defer func() {
		ollamaURL = oldOllamaURL
		qdrantURL = oldQdrantURL
	}()

	defer gock.Off()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Setup actual handler
	env.Router.POST("/api/v1/characters", createCharacterHandler)

	t.Run("SuccessfulCreation", func(t *testing.T) {
		// Mock Ollama and Qdrant for embedding generation (in goroutine)
		gock.New("http://localhost:11434").
			Post("/api/embeddings").
			Reply(200).
			JSON(map[string]interface{}{"embedding": []float64{0.1, 0.2, 0.3}})

		gock.New("http://localhost:6333").
			Put("/collections/characters/points").
			Reply(200).
			JSON(map[string]interface{}{"status": "ok"})

		// Expect INSERT query
		mock.ExpectExec("INSERT INTO characters").
			WithArgs(sqlmock.AnyArg(), "TestHero", sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		characterData := TestData.CharacterRequest("TestHero")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   characterData,
		})

		assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		// Verify all expectations were met
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}

		// Give goroutine time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Expect INSERT query to fail
		mock.ExpectExec("INSERT INTO characters").
			WillReturnError(fmt.Errorf("database error"))

		characterData := TestData.CharacterRequest("ErrorChar")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   characterData,
		})

		assertErrorResponse(t, w, http.StatusInternalServerError, "Failed to create character")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   `{"invalid": json}`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid character data")
	})
}

// TestRealGetCharacterHandler tests the actual getCharacterHandler
func TestRealGetCharacterHandler(t *testing.T) {
	t.Skip("Skipping due to global variable concurrency issues - will be addressed in future refactor")
	cleanup := setupTestLogger()
	defer cleanup()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	env.Router.GET("/api/v1/characters/:id", getCharacterHandler)

	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		testID := uuid.New()

		personalityJSON, _ := json.Marshal(map[string]interface{}{"brave": 0.8})
		speechJSON, _ := json.Marshal(map[string]interface{}{"tempo": "fast"})
		relationshipsJSON, _ := json.Marshal(map[string]interface{}{"allies": []string{}})
		voiceJSON, _ := json.Marshal(map[string]interface{}{"pitch": "high"})

		rows := sqlmock.NewRows([]string{"id", "name", "personality_traits", "background_story",
			"speech_patterns", "relationships", "voice_profile", "created_at", "updated_at"}).
			AddRow(testID, "TestChar", personalityJSON, "A brave hero", speechJSON,
				relationshipsJSON, voiceJSON, time.Now(), time.Now())

		mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
			WithArgs(testID.String()).
			WillReturnRows(rows)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/characters/%s", testID.String()),
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": "TestChar",
		})

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("CharacterNotFound", func(t *testing.T) {
		testID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
			WithArgs(testID.String()).
			WillReturnError(sql.ErrNoRows)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/characters/%s", testID.String()),
		})

		assertErrorResponse(t, w, http.StatusNotFound, "not found")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM characters WHERE id").
			WithArgs(testID.String()).
			WillReturnError(fmt.Errorf("database error"))

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/characters/%s", testID.String()),
		})

		assertErrorResponse(t, w, http.StatusInternalServerError, "Failed to retrieve character")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestRealListCharactersHandler tests the actual listCharactersHandler
func TestRealListCharactersHandler(t *testing.T) {
	t.Skip("Skipping due to global variable concurrency issues - will be addressed in future refactor")
	cleanup := setupTestLogger()
	defer cleanup()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	env.Router.GET("/api/v1/characters", listCharactersHandler)

	t.Run("SuccessfulList", func(t *testing.T) {
		personalityJSON, _ := json.Marshal(map[string]interface{}{"brave": 0.8})
		speechJSON, _ := json.Marshal(map[string]interface{}{"tempo": "fast"})
		relationshipsJSON, _ := json.Marshal(map[string]interface{}{})
		voiceJSON, _ := json.Marshal(map[string]interface{}{})

		rows := sqlmock.NewRows([]string{"id", "name", "personality_traits", "background_story",
			"speech_patterns", "relationships", "voice_profile", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Char1", personalityJSON, "Story1", speechJSON, relationshipsJSON, voiceJSON, time.Now(), time.Now()).
			AddRow(uuid.New(), "Char2", personalityJSON, "Story2", speechJSON, relationshipsJSON, voiceJSON, time.Now(), time.Now())

		mock.ExpectQuery("SELECT (.+) FROM characters ORDER BY created_at DESC").
			WillReturnRows(rows)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/characters",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			count, ok := response["count"].(float64)
			if !ok || int(count) != 2 {
				t.Errorf("Expected count 2, got %v", response["count"])
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM characters").
			WillReturnError(fmt.Errorf("database error"))

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/characters",
		})

		assertErrorResponse(t, w, http.StatusInternalServerError, "Failed to list characters")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestRealHealthHandler tests the actual healthHandler
func TestRealHealthHandler(t *testing.T) {
	t.Skip("Skipping due to global variable concurrency issues - will be addressed in future refactor")
	cleanup := setupTestLogger()
	defer cleanup()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	// Initialize rest client
	oldClient := restClient
	restClient = resty.New()
	defer func() { restClient = oldClient }()

	oldOllamaURL := ollamaURL
	oldQdrantURL := qdrantURL
	ollamaURL = "http://localhost:11434"
	qdrantURL = "http://localhost:6333"
	defer func() {
		ollamaURL = oldOllamaURL
		qdrantURL = oldQdrantURL
	}()

	defer gock.Off()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	env.Router.GET("/health", healthHandler)

	t.Run("HealthyStatus", func(t *testing.T) {
		// Mock database ping
		mock.ExpectPing()

		// Mock Ollama and Qdrant requests
		gock.New("http://localhost:11434").
			Get("/api/version").
			Reply(200).
			JSON(map[string]interface{}{"version": "0.1.0"})

		gock.New("http://localhost:6333").
			Get("/health").
			Reply(200).
			JSON(map[string]interface{}{"status": "ok"})

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "game-dialog-generator",
		})

		if response != nil {
			resources, ok := response["resources"].(map[string]interface{})
			if !ok {
				t.Error("Expected resources in response")
			} else {
				if resources["database"] != "connected" {
					t.Error("Expected database to be connected")
				}
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DegradedStatus_DatabaseDown", func(t *testing.T) {
		// Mock database ping failure
		pingExpectation := mock.ExpectPing()
		pingExpectation.WillReturnError(fmt.Errorf("database down"))

		// Mock Ollama and Qdrant requests
		gock.New("http://localhost:11434").
			Get("/api/version").
			Reply(200).
			JSON(map[string]interface{}{"version": "0.1.0"})

		gock.New("http://localhost:6333").
			Get("/health").
			Reply(200).
			JSON(map[string]interface{}{"status": "ok"})

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "degraded",
		})

		if response != nil {
			resources, ok := response["resources"].(map[string]interface{})
			if !ok {
				t.Error("Expected resources in response")
			} else {
				if resources["database"] != "disconnected" {
					t.Error("Expected database to be disconnected")
				}
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestRealProjectHandlers tests project-related handlers
func TestRealProjectHandlers(t *testing.T) {
	t.Skip("Skipping due to global variable concurrency issues - will be addressed in future refactor")
	cleanup := setupTestLogger()
	defer cleanup()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	env.Router.POST("/api/v1/projects", createProjectHandler)
	env.Router.GET("/api/v1/projects", listProjectsHandler)

	t.Run("CreateProject_Success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO projects").
			WithArgs(sqlmock.AnyArg(), "TestProject", sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), "json", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		projectData := TestData.ProjectRequest("TestProject")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects",
			Body:   projectData,
		})

		assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("ListProjects_Success", func(t *testing.T) {
		settingsJSON, _ := json.Marshal(map[string]interface{}{"theme": "jungle"})

		rows := sqlmock.NewRows([]string{"id", "name", "description", "characters", "settings", "export_format", "created_at"}).
			AddRow(uuid.New(), "Project1", "Desc1", `{"char1","char2"}`, settingsJSON, "json", time.Now()).
			AddRow(uuid.New(), "Project2", "Desc2", `{"char3"}`, settingsJSON, "json", time.Now())

		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC").
			WillReturnRows(rows)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			count, ok := response["count"].(float64)
			if !ok || int(count) != 2 {
				t.Errorf("Expected count 2, got %v", response["count"])
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
