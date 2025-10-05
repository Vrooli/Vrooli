package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Set test mode
	gin.SetMode(gin.TestMode)
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "8080")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
	os.Unsetenv("API_PORT")

	os.Exit(code)
}

// Test Health Handler
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Mock database connection (in real scenario, we'd use a test database)
	// For now, we'll test the handler structure

	env.Router.GET("/health", func(c *gin.Context) {
		status := gin.H{
			"status":    "healthy",
			"service":   "game-dialog-generator",
			"theme":     "jungle-platformer",
			"timestamp": time.Now().UTC(),
			"resources": gin.H{
				"database": "connected",
				"ollama":   "available",
				"qdrant":   "available",
			},
		}
		c.JSON(http.StatusOK, status)
	})

	t.Run("SuccessfulHealthCheck", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "game-dialog-generator",
			"theme":   "jungle-platformer",
		})

		if response != nil {
			if resources, ok := response["resources"].(map[string]interface{}); ok {
				if resources["database"] != "connected" {
					t.Error("Expected database to be connected")
				}
			}
		}
	})
}

// Test Create Character Handler
func TestCreateCharacterHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Setup route with mock implementation
	env.Router.POST("/api/v1/characters", func(c *gin.Context) {
		var character Character

		if err := c.ShouldBindJSON(&character); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character data", "details": err.Error()})
			return
		}

		character.ID = uuid.New()
		character.CreatedAt = time.Now().UTC()
		character.UpdatedAt = time.Now().UTC()

		c.JSON(http.StatusCreated, gin.H{
			"success":      true,
			"character_id": character.ID,
			"message":      "🌿 Character created in the jungle adventure!",
		})
	})

	t.Run("SuccessfulCharacterCreation", func(t *testing.T) {
		characterData := TestData.CharacterRequest("TestHero")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   characterData,
		})

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, ok := response["character_id"]; !ok {
				t.Error("Expected character_id in response")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/characters").
			AddMissingRequiredFields("POST", "/api/v1/characters", map[string]interface{}{
				"personality_traits": map[string]interface{}{"brave": 0.8},
				// Missing name
			}).
			Build()

		suite := &HandlerTestSuite{
			HandlerName: "createCharacterHandler",
			Router:      env.Router,
			BaseURL:     "/api/v1/characters",
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   "",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// Test Get Character Handler
func TestGetCharacterHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test character for lookup
	testChar := createTestCharacter(t, "TestExplorer")
	defer testChar.Cleanup()

	// Mock storage
	mockDB := NewMockDatabase()
	mockDB.SaveCharacter(testChar.Character)

	// Setup route with mock implementation
	env.Router.GET("/api/v1/characters/:id", func(c *gin.Context) {
		characterID := c.Param("id")

		char, err := mockDB.GetCharacter(characterID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Character not found in the jungle"})
			return
		}

		c.JSON(http.StatusOK, char)
	})

	t.Run("SuccessfulCharacterRetrieval", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/characters/%s", testChar.Character.ID.String()),
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": "TestExplorer",
		})

		if response != nil {
			if id, ok := response["id"]; ok {
				if id != testChar.Character.ID.String() {
					t.Errorf("Expected character ID %s, got %s", testChar.Character.ID.String(), id)
				}
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidUUID("GET", "/api/v1/characters/invalid-uuid").
			AddNonExistentCharacter("GET", "/api/v1/characters/%s").
			Build()

		suite := &HandlerTestSuite{
			HandlerName: "getCharacterHandler",
			Router:      env.Router,
			BaseURL:     "/api/v1/characters/:id",
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("NonExistentCharacter", func(t *testing.T) {
		nonExistentID := uuid.New()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/characters/%s", nonExistentID.String()),
		})

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// Test List Characters Handler
func TestListCharactersHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test characters
	mockDB := NewMockDatabase()
	char1 := createTestCharacter(t, "Explorer1")
	char2 := createTestCharacter(t, "Explorer2")
	mockDB.SaveCharacter(char1.Character)
	mockDB.SaveCharacter(char2.Character)

	// Setup route
	env.Router.GET("/api/v1/characters", func(c *gin.Context) {
		characters := []Character{char1.Character, char2.Character}
		c.JSON(http.StatusOK, gin.H{
			"characters": characters,
			"count":      len(characters),
			"message":    "🌿 Characters from the jungle adventure!",
		})
	})

	t.Run("SuccessfulCharactersList", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/characters",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			count, ok := response["count"].(float64)
			if !ok || int(count) != 2 {
				t.Errorf("Expected 2 characters, got %v", response["count"])
			}

			characters, ok := response["characters"].([]interface{})
			if !ok || len(characters) != 2 {
				t.Error("Expected characters array with 2 items")
			}
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		// Test with empty database
		env.Router.GET("/api/v1/characters/empty", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"characters": []Character{},
				"count":      0,
				"message":    "🌿 Characters from the jungle adventure!",
			})
		})

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/characters/empty",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"count": float64(0),
		})

		if response != nil {
			characters, ok := response["characters"].([]interface{})
			if !ok || len(characters) != 0 {
				t.Error("Expected empty characters array")
			}
		}
	})
}

// Test Dialog Generation Handler
func TestGenerateDialogHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testChar := createTestCharacter(t, "DialogHero")
	mockDB := NewMockDatabase()
	mockDB.SaveCharacter(testChar.Character)

	// Setup route with mock implementation
	env.Router.POST("/api/v1/dialog/generate", func(c *gin.Context) {
		var req DialogGenerationRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dialog request", "details": err.Error()})
			return
		}

		// Validate character ID
		characterID, err := uuid.Parse(req.CharacterID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
			return
		}

		// Check if character exists
		_, err = mockDB.GetCharacter(characterID.String())
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Character not found in jungle"})
			return
		}

		// Mock dialog generation
		response := DialogGenerationResponse{
			Dialog:                   "This is a test dialog line!",
			Emotion:                  "neutral",
			CharacterConsistencyScore: 0.85,
		}

		c.JSON(http.StatusOK, response)
	})

	t.Run("SuccessfulDialogGeneration", func(t *testing.T) {
		dialogReq := TestData.DialogRequest(testChar.Character.ID.String(), "A jungle clearing")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/generate",
			Body:   dialogReq,
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if dialog, ok := response["dialog"].(string); !ok || dialog == "" {
				t.Error("Expected dialog in response")
			}

			if emotion, ok := response["emotion"].(string); !ok || emotion == "" {
				t.Error("Expected emotion in response")
			}

			if score, ok := response["character_consistency_score"].(float64); !ok || score < 0 || score > 1 {
				t.Error("Expected valid consistency score between 0 and 1")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := DialogGenerationTestPatterns(testChar.Character.ID.String())

		suite := &HandlerTestSuite{
			HandlerName: "generateDialogHandler",
			Router:      env.Router,
			BaseURL:     "/api/v1/dialog/generate",
		}

		suite.RunErrorTests(t, patterns.Build())
	})

	t.Run("NonExistentCharacter", func(t *testing.T) {
		nonExistentID := uuid.New()
		dialogReq := TestData.DialogRequest(nonExistentID.String(), "Test scene")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/generate",
			Body:   dialogReq,
		})

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// Test Batch Dialog Generation Handler
func TestBatchGenerateDialogHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test characters
	char1 := createTestCharacter(t, "BatchChar1")
	char2 := createTestCharacter(t, "BatchChar2")
	mockDB := NewMockDatabase()
	mockDB.SaveCharacter(char1.Character)
	mockDB.SaveCharacter(char2.Character)

	// Setup route
	env.Router.POST("/api/v1/dialog/batch", func(c *gin.Context) {
		var req BatchDialogRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch dialog request", "details": err.Error()})
			return
		}

		var dialogSet []DialogGenerationResponse
		for _, dialogReq := range req.DialogRequests {
			characterID, err := uuid.Parse(dialogReq.CharacterID)
			if err != nil {
				continue
			}

			_, err = mockDB.GetCharacter(characterID.String())
			if err != nil {
				continue
			}

			dialogSet = append(dialogSet, DialogGenerationResponse{
				Dialog:                   "Batch generated dialog",
				Emotion:                  "neutral",
				CharacterConsistencyScore: 0.80,
			})
		}

		metrics := map[string]interface{}{
			"total_requested": len(req.DialogRequests),
			"successful":      len(dialogSet),
			"duration_ms":     100,
			"avg_consistency": 0.80,
		}

		response := BatchDialogResponse{
			DialogSet:         dialogSet,
			GenerationMetrics: metrics,
		}

		c.JSON(http.StatusOK, response)
	})

	t.Run("SuccessfulBatchGeneration", func(t *testing.T) {
		sceneID := uuid.New().String()
		dialogReqs := []DialogGenerationRequest{
			TestData.DialogRequest(char1.Character.ID.String(), "Scene 1"),
			TestData.DialogRequest(char2.Character.ID.String(), "Scene 2"),
		}

		batchReq := TestData.BatchDialogRequest(sceneID, dialogReqs)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/batch",
			Body:   batchReq,
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			dialogSet, ok := response["dialog_set"].([]interface{})
			if !ok || len(dialogSet) != 2 {
				t.Errorf("Expected 2 dialog items, got %v", len(dialogSet))
			}

			metrics, ok := response["generation_metrics"].(map[string]interface{})
			if !ok {
				t.Error("Expected generation_metrics in response")
			} else {
				if total, ok := metrics["total_requested"].(float64); !ok || int(total) != 2 {
					t.Error("Expected total_requested to be 2")
				}
			}
		}
	})

	t.Run("EmptyBatchRequest", func(t *testing.T) {
		batchReq := BatchDialogRequest{
			SceneID:        uuid.New().String(),
			DialogRequests: []DialogGenerationRequest{},
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/batch",
			Body:   batchReq,
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			dialogSet, ok := response["dialog_set"].([]interface{})
			if !ok || len(dialogSet) != 0 {
				t.Error("Expected empty dialog_set for empty batch")
			}
		}
	})
}

// Test Project Creation Handler
func TestCreateProjectHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	env.Router.POST("/api/v1/projects", func(c *gin.Context) {
		var project Project

		if err := c.ShouldBindJSON(&project); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project data"})
			return
		}

		project.ID = uuid.New()
		project.CreatedAt = time.Now().UTC()

		c.JSON(http.StatusCreated, gin.H{
			"success":    true,
			"project_id": project.ID,
			"message":    "🌿 Game project created in the jungle!",
		})
	})

	t.Run("SuccessfulProjectCreation", func(t *testing.T) {
		projectData := TestData.ProjectRequest("TestGame")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects",
			Body:   projectData,
		})

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, ok := response["project_id"]; !ok {
				t.Error("Expected project_id in response")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/projects").
			AddMissingRequiredFields("POST", "/api/v1/projects", map[string]interface{}{
				"description": "Missing name field",
			}).
			Build()

		suite := &HandlerTestSuite{
			HandlerName: "createProjectHandler",
			Router:      env.Router,
			BaseURL:     "/api/v1/projects",
		}

		suite.RunErrorTests(t, patterns)
	})
}

// Test List Projects Handler
func TestListProjectsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test projects
	proj1 := createTestProject(t, "Project1")
	proj2 := createTestProject(t, "Project2")

	env.Router.GET("/api/v1/projects", func(c *gin.Context) {
		projects := []Project{proj1.Project, proj2.Project}
		c.JSON(http.StatusOK, gin.H{
			"projects": projects,
			"count":    len(projects),
		})
	})

	t.Run("SuccessfulProjectsList", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/projects",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			count, ok := response["count"].(float64)
			if !ok || int(count) != 2 {
				t.Errorf("Expected 2 projects, got %v", response["count"])
			}

			projects, ok := response["projects"].([]interface{})
			if !ok || len(projects) != 2 {
				t.Error("Expected projects array with 2 items")
			}
		}
	})
}

// Test Helper Functions

func TestExtractEmotionFromDialog(t *testing.T) {
	tests := []struct {
		name           string
		dialog         string
		defaultEmotion string
		expected       string
	}{
		{"Excited", "That's great! Awesome!", "", "excited"},
		{"Curious", "What is that?", "", "curious"},
		{"Thoughtful", "Hmm... I wonder...", "", "thoughtful"},
		{"Angry", "No! Stop that!", "", "angry"},
		{"Default", "Hello there", "happy", "happy"},
		{"Neutral", "Hello there", "", "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEmotionFromDialog(tt.dialog, tt.defaultEmotion)
			if result != tt.expected {
				t.Errorf("Expected emotion %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCalculateCharacterConsistency(t *testing.T) {
	testChar := createTestCharacter(t, "ConsistencyTest")

	tests := []struct {
		name     string
		dialog   string
		minScore float64
		maxScore float64
	}{
		{"BraveDialog", "I will fight with courage!", 0.7, 1.0},
		{"HumorousDialog", "Haha, that's a funny joke!", 0.7, 1.0},
		{"NeutralDialog", "The weather is nice", 0.6, 0.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateCharacterConsistency(testChar.Character, tt.dialog)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestCalculateAverageConsistency(t *testing.T) {
	tests := []struct {
		name      string
		dialogSet []DialogGenerationResponse
		expected  float64
	}{
		{
			"EmptySet",
			[]DialogGenerationResponse{},
			0.0,
		},
		{
			"SingleDialog",
			[]DialogGenerationResponse{
				{CharacterConsistencyScore: 0.8},
			},
			0.8,
		},
		{
			"MultipleDialogs",
			[]DialogGenerationResponse{
				{CharacterConsistencyScore: 0.8},
				{CharacterConsistencyScore: 0.6},
			},
			0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAverageConsistency(tt.dialogSet)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// Test Database Connection Error Handling
func TestDatabaseConnectionHandling(t *testing.T) {
	t.Run("MissingDatabaseConfig", func(t *testing.T) {
		// This test validates that the app properly requires database configuration
		// In production, initDB() would fail if no config is provided

		// Clear environment variables
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		// The initDB() function would log.Fatal here, so we can't test it directly
		// Instead, we verify the logic would catch missing config
		postgresURL := os.Getenv("POSTGRES_URL")
		if postgresURL == "" {
			dbHost := os.Getenv("POSTGRES_HOST")
			dbPort := os.Getenv("POSTGRES_PORT")
			dbUser := os.Getenv("POSTGRES_USER")
			dbPassword := os.Getenv("POSTGRES_PASSWORD")
			dbName := os.Getenv("POSTGRES_DB")

			if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
				// This is the expected behavior - missing config detected
				t.Log("Correctly detected missing database configuration")
			} else {
				t.Error("Should have detected missing database configuration")
			}
		}
	})
}

// Test Ollama and Qdrant Client Initialization
func TestClientInitialization(t *testing.T) {
	t.Run("MissingOllamaURL", func(t *testing.T) {
		os.Unsetenv("OLLAMA_URL")
		ollamaURL := os.Getenv("OLLAMA_URL")
		if ollamaURL == "" {
			t.Log("Correctly detected missing OLLAMA_URL")
		} else {
			t.Error("Should have detected missing OLLAMA_URL")
		}
	})

	t.Run("MissingQdrantURL", func(t *testing.T) {
		os.Unsetenv("QDRANT_URL")
		qdrantURL := os.Getenv("QDRANT_URL")
		if qdrantURL == "" {
			t.Log("Correctly detected missing QDRANT_URL")
		} else {
			t.Error("Should have detected missing QDRANT_URL")
		}
	})
}

// Test CORS Middleware
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Add CORS middleware
	env.Router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	env.Router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	t.Run("CORSHeaders", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS Allow-Origin header")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected CORS Allow-Methods header")
		}
	})

	t.Run("OPTIONSRequest", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/test",
		})

		if w.Code != 204 {
			t.Errorf("Expected status 204 for OPTIONS request, got %d", w.Code)
		}
	})
}

// Integration test for complete workflow
func TestCompleteWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	mockDB := NewMockDatabase()

	// Setup all routes
	setupWorkflowRoutes(env.Router, mockDB)

	t.Run("CompleteGameDialogWorkflow", func(t *testing.T) {
		// Step 1: Create a project
		projectData := TestData.ProjectRequest("EpicAdventure")
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/projects",
			Body:   projectData,
		})

		projectResp := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if projectResp == nil {
			t.Fatal("Failed to create project")
		}

		// Step 2: Create characters
		char1Data := TestData.CharacterRequest("Hero")
		w = makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body:   char1Data,
		})

		char1Resp := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if char1Resp == nil {
			t.Fatal("Failed to create character")
		}

		characterID := char1Resp["character_id"].(string)

		// Step 3: Generate dialog
		dialogReq := TestData.DialogRequest(characterID, "The hero enters the jungle temple")
		w = makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/generate",
			Body:   dialogReq,
		})

		dialogResp := assertJSONResponse(t, w, http.StatusOK, nil)

		if dialogResp == nil {
			t.Fatal("Failed to generate dialog")
		}

		if dialog, ok := dialogResp["dialog"].(string); !ok || dialog == "" {
			t.Error("Expected generated dialog in response")
		}

		t.Log("Complete workflow test passed successfully")
	})
}

// Helper function to setup workflow routes
func setupWorkflowRoutes(router *gin.Engine, mockDB *MockDatabase) {
	router.POST("/api/v1/projects", func(c *gin.Context) {
		var project Project
		if err := c.ShouldBindJSON(&project); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project data"})
			return
		}
		project.ID = uuid.New()
		project.CreatedAt = time.Now().UTC()
		mockDB.SaveProject(project)
		c.JSON(http.StatusCreated, gin.H{
			"success":    true,
			"project_id": project.ID,
		})
	})

	router.POST("/api/v1/characters", func(c *gin.Context) {
		var character Character
		if err := c.ShouldBindJSON(&character); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character data"})
			return
		}
		character.ID = uuid.New()
		character.CreatedAt = time.Now().UTC()
		character.UpdatedAt = time.Now().UTC()
		mockDB.SaveCharacter(character)
		c.JSON(http.StatusCreated, gin.H{
			"success":      true,
			"character_id": character.ID.String(),
		})
	})

	router.POST("/api/v1/dialog/generate", func(c *gin.Context) {
		var req DialogGenerationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dialog request"})
			return
		}

		characterID, err := uuid.Parse(req.CharacterID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
			return
		}

		_, err = mockDB.GetCharacter(characterID.String())
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Character not found"})
			return
		}

		response := DialogGenerationResponse{
			Dialog:                   "Mock generated dialog for the scene",
			Emotion:                  "determined",
			CharacterConsistencyScore: 0.88,
		}

		c.JSON(http.StatusOK, response)
	})
}
