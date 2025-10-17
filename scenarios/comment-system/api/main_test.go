// +build testing

package main

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Set test mode
	setupTestLogger()

	// Run tests
	m.Run()
}

// ============================================================================
// Database Tests
// ============================================================================

func TestNewDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		if testDB == nil {
			t.Skip("Database not available")
		}
		defer testDB.Cleanup()

		if testDB.DB == nil {
			t.Fatal("Expected database connection, got nil")
		}

		// Test ping
		if err := testDB.DB.conn.Ping(); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		config := &Config{
			DBHost: "invalid-host",
			DBPort: 9999,
			DBUser: "invalid",
			DBPass: "invalid",
			DBName: "invalid",
		}

		db, err := NewDatabase(config)
		if err == nil {
			db.Close()
			t.Error("Expected error with invalid database config, got nil")
		}
	})
}

func TestDatabaseGetComments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	scenarioName := "test-scenario"

	t.Run("EmptyDatabase", func(t *testing.T) {
		comments, totalCount, err := testDB.DB.GetComments(scenarioName, nil, 50, 0, "newest")
		if err != nil {
			t.Fatalf("Failed to get comments: %v", err)
		}

		if totalCount != 0 {
			t.Errorf("Expected 0 comments, got %d", totalCount)
		}

		if len(comments) != 0 {
			t.Errorf("Expected empty slice, got %d comments", len(comments))
		}
	})

	t.Run("WithComments", func(t *testing.T) {
		// Create test comments
		for i := 0; i < 3; i++ {
			createTestComment(t, testDB.DB, scenarioName, "Test comment "+string(rune('A'+i)))
		}

		comments, totalCount, err := testDB.DB.GetComments(scenarioName, nil, 50, 0, "newest")
		if err != nil {
			t.Fatalf("Failed to get comments: %v", err)
		}

		if totalCount != 3 {
			t.Errorf("Expected 3 comments, got %d", totalCount)
		}

		if len(comments) != 3 {
			t.Errorf("Expected 3 comments in slice, got %d", len(comments))
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// Clear and create 10 comments
		testDB.DB.conn.Exec("TRUNCATE TABLE comments CASCADE")
		for i := 0; i < 10; i++ {
			createTestComment(t, testDB.DB, scenarioName, "Test comment")
		}

		// Get first page
		comments, totalCount, err := testDB.DB.GetComments(scenarioName, nil, 5, 0, "newest")
		if err != nil {
			t.Fatalf("Failed to get comments: %v", err)
		}

		if totalCount != 10 {
			t.Errorf("Expected total count 10, got %d", totalCount)
		}

		if len(comments) != 5 {
			t.Errorf("Expected 5 comments in page, got %d", len(comments))
		}

		// Get second page
		comments2, _, err := testDB.DB.GetComments(scenarioName, nil, 5, 5, "newest")
		if err != nil {
			t.Fatalf("Failed to get comments page 2: %v", err)
		}

		if len(comments2) != 5 {
			t.Errorf("Expected 5 comments in page 2, got %d", len(comments2))
		}
	})

	t.Run("DifferentScenarios", func(t *testing.T) {
		testDB.DB.conn.Exec("TRUNCATE TABLE comments CASCADE")

		createTestComment(t, testDB.DB, "scenario-a", "Comment A")
		createTestComment(t, testDB.DB, "scenario-b", "Comment B")

		comments, totalCount, err := testDB.DB.GetComments("scenario-a", nil, 50, 0, "newest")
		if err != nil {
			t.Fatalf("Failed to get comments: %v", err)
		}

		if totalCount != 1 {
			t.Errorf("Expected 1 comment for scenario-a, got %d", totalCount)
		}

		if len(comments) > 0 && comments[0].ScenarioName != "scenario-a" {
			t.Errorf("Expected scenario-a, got %s", comments[0].ScenarioName)
		}
	})
}

func TestDatabaseCreateComment(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		comment := &Comment{
			ScenarioName: "test-scenario",
			Content:      "Test comment content",
			ContentType:  "markdown",
			Status:       "active",
			Metadata:     make(map[string]interface{}),
		}

		err := testDB.DB.CreateComment(comment)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		if comment.ID == uuid.Nil {
			t.Error("Expected comment ID to be set")
		}

		if comment.CreatedAt.IsZero() {
			t.Error("Expected created_at to be set")
		}

		if comment.Version != 1 {
			t.Errorf("Expected version 1, got %d", comment.Version)
		}
	})

	t.Run("WithAuthor", func(t *testing.T) {
		authorID := uuid.New()
		authorName := "Test Author"

		comment := &Comment{
			ScenarioName: "test-scenario",
			AuthorID:     &authorID,
			AuthorName:   &authorName,
			Content:      "Authored comment",
			ContentType:  "markdown",
			Status:       "active",
			Metadata:     make(map[string]interface{}),
		}

		err := testDB.DB.CreateComment(comment)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		if comment.AuthorID == nil || *comment.AuthorID != authorID {
			t.Error("Author ID not preserved")
		}
	})

	t.Run("WithMetadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"tags":     []string{"important", "question"},
			"priority": 5,
		}

		comment := &Comment{
			ScenarioName: "test-scenario",
			Content:      "Comment with metadata",
			ContentType:  "markdown",
			Status:       "active",
			Metadata:     metadata,
		}

		err := testDB.DB.CreateComment(comment)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
	})
}

func TestDatabaseScenarioConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	scenarioName := "test-scenario-config"

	t.Run("GetNonExistent_CreatesDefault", func(t *testing.T) {
		config, err := testDB.DB.GetScenarioConfig(scenarioName)
		if err != nil {
			t.Fatalf("Failed to get config: %v", err)
		}

		if config == nil {
			t.Fatal("Expected default config, got nil")
		}

		if config.ScenarioName != scenarioName {
			t.Errorf("Expected scenario name %s, got %s", scenarioName, config.ScenarioName)
		}

		if !config.AuthRequired {
			t.Error("Expected default AuthRequired to be true")
		}

		if config.AllowAnonymous {
			t.Error("Expected default AllowAnonymous to be false")
		}
	})

	t.Run("GetExisting", func(t *testing.T) {
		// Should return the same config created above
		config, err := testDB.DB.GetScenarioConfig(scenarioName)
		if err != nil {
			t.Fatalf("Failed to get existing config: %v", err)
		}

		if config.ScenarioName != scenarioName {
			t.Errorf("Expected scenario name %s, got %s", scenarioName, config.ScenarioName)
		}
	})

	t.Run("CreateDefaultConfig", func(t *testing.T) {
		newScenario := "new-scenario-config"
		config, err := testDB.DB.CreateDefaultConfig(newScenario)
		if err != nil {
			t.Fatalf("Failed to create default config: %v", err)
		}

		if config.ModerationLevel != "manual" {
			t.Errorf("Expected moderation level 'manual', got %s", config.ModerationLevel)
		}

		if config.ThemeConfig == nil {
			t.Error("Expected theme config to be initialized")
		}

		if config.NotificationSettings == nil {
			t.Error("Expected notification settings to be initialized")
		}
	})
}

// ============================================================================
// HTTP Handler Tests
// ============================================================================

func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Response missing timestamp field")
		}

		if _, ok := response["dependencies"]; !ok {
			t.Error("Response missing dependencies field")
		}
	})
}

func TestPostgresHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health/postgres",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}
	})
}

func TestGetComments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "test-scenario"

	t.Run("EmptyScenario", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName,
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if totalCount, ok := response["total_count"].(float64); !ok || totalCount != 0 {
			t.Errorf("Expected total_count 0, got %v", response["total_count"])
		}

		if hasMore, ok := response["has_more"].(bool); !ok || hasMore {
			t.Error("Expected has_more to be false")
		}
	})

	t.Run("WithComments", func(t *testing.T) {
		// Create test comments
		createTestComment(t, testDB.DB, scenarioName, "First comment")
		createTestComment(t, testDB.DB, scenarioName, "Second comment")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName,
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if totalCount, ok := response["total_count"].(float64); !ok || totalCount != 2 {
			t.Errorf("Expected total_count 2, got %v", response["total_count"])
		}

		if comments, ok := response["comments"].([]interface{}); !ok || len(comments) != 2 {
			t.Errorf("Expected 2 comments in response, got %v", response["comments"])
		}
	})

	t.Run("WithPaginationParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName + "?limit=1&offset=0",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if comments, ok := response["comments"].([]interface{}); !ok || len(comments) != 1 {
			t.Errorf("Expected 1 comment with limit=1, got %v", len(comments))
		}

		if hasMore, ok := response["has_more"].(bool); !ok || !hasMore {
			t.Error("Expected has_more to be true with more results")
		}
	})

	t.Run("InvalidLimitDefaults", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName + "?limit=-1",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still succeed with default limit
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("ExcessiveLimitCapped", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName + "?limit=500",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still succeed with capped limit
		assertJSONResponse(t, w, http.StatusOK)
	})
}

func TestCreateComment(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "test-scenario"

	// Create scenario config first
	testDB.DB.CreateDefaultConfig(scenarioName)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      "Test comment content",
				"content_type": "markdown",
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success to be true")
		}

		if comment, ok := response["comment"].(map[string]interface{}); ok {
			assertCommentFields(t, comment)

			if content, ok := comment["content"].(string); !ok || content != "Test comment content" {
				t.Errorf("Expected content 'Test comment content', got %v", comment["content"])
			}
		} else {
			t.Error("Response missing comment field")
		}
	})

	t.Run("EmptyContent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      "",
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Gin binding validation will reject empty content with its own error message
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("ContentTooLong", func(t *testing.T) {
		longContent := make([]byte, 10001)
		for i := range longContent {
			longContent[i] = 'a'
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      string(longContent),
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Comment content too long (max 10000 characters)")
	})

	t.Run("MissingContentField", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 400 due to binding validation
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DefaultContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      "Comment without content_type",
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		if comment, ok := response["comment"].(map[string]interface{}); ok {
			if contentType, ok := comment["content_type"].(string); !ok || contentType != "markdown" {
				t.Errorf("Expected default content_type 'markdown', got %v", comment["content_type"])
			}
		}
	})

	t.Run("WithMetadata", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      "Comment with metadata",
				"author_token": "test-token",
				"metadata": map[string]interface{}{
					"priority": 5,
					"tags":     []string{"important"},
				},
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		if comment, ok := response["comment"].(map[string]interface{}); ok {
			if metadata, ok := comment["metadata"].(map[string]interface{}); !ok {
				t.Error("Expected metadata field in comment")
			} else if len(metadata) == 0 {
				t.Error("Expected metadata to contain values")
			}
		}
	})
}

func TestUpdateComment(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)

	t.Run("InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/comments/invalid-uuid",
			Body: map[string]interface{}{
				"content":      "Updated content",
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid comment ID")
	})

	t.Run("NotImplemented", func(t *testing.T) {
		validUUID := uuid.New()

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/comments/" + validUUID.String(),
			Body: map[string]interface{}{
				"content":      "Updated content",
				"author_token": "test-token",
			},
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Currently returns 200 with "not implemented" message
		if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 200 or 401, got %d", w.Code)
		}
	})
}

func TestDeleteComment(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)

	t.Run("InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/comments/invalid-uuid",
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid comment ID")
	})

	t.Run("NotImplemented", func(t *testing.T) {
		validUUID := uuid.New()

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/comments/" + validUUID.String(),
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Currently returns 200 with "not implemented" message
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestGetConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "test-scenario-config"

	t.Run("Success_CreatesDefault", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/config/" + scenarioName,
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if config, ok := response["config"].(map[string]interface{}); ok {
			assertConfigFields(t, config)

			if name, ok := config["scenario_name"].(string); !ok || name != scenarioName {
				t.Errorf("Expected scenario_name %s, got %v", scenarioName, config["scenario_name"])
			}
		} else {
			t.Error("Response missing config field")
		}
	})

	t.Run("ExistingConfig", func(t *testing.T) {
		// Should return the same config created above
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/config/" + scenarioName,
		}

		w, err := makeHTTPRequest(app, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})
}

// ============================================================================
// Utility Function Tests
// ============================================================================

func TestGetEnv(t *testing.T) {
	t.Run("ExistingValue", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test-value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "default")
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got %s", result)
		}
	})

	t.Run("DefaultValue", func(t *testing.T) {
		result := getEnv("NON_EXISTENT_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("Expected 'default-value', got %s", result)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("ValidInteger", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 10)
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("InvalidInteger_UsesDefault", func(t *testing.T) {
		os.Setenv("TEST_INT", "not-a-number")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 10)
		if result != 10 {
			t.Errorf("Expected default 10, got %d", result)
		}
	})

	t.Run("NonExistent_UsesDefault", func(t *testing.T) {
		result := getEnvInt("NON_EXISTENT_INT", 99)
		if result != 99 {
			t.Errorf("Expected default 99, got %d", result)
		}
	})
}

// ============================================================================
// Error Pattern Tests
// ============================================================================

func TestCommentErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)

	// Create scenario config
	scenarioName := "test-error-patterns"
	testDB.DB.CreateDefaultConfig(scenarioName)

	// Build and run error scenarios
	builder := NewTestScenarioBuilder()
	builder.
		AddEmptyContent("/api/v1/comments/" + scenarioName).
		AddContentTooLong("/api/v1/comments/" + scenarioName).
		AddInvalidQueryParams("/api/v1/comments/" + scenarioName)

	builder.RunScenarios(t, app)
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestCommentLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "lifecycle-test"

	// Create config
	testDB.DB.CreateDefaultConfig(scenarioName)

	// Create comment
	createReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/comments/" + scenarioName,
		Body: map[string]interface{}{
			"content":      "Lifecycle test comment",
			"author_token": "test-token",
		},
	}

	w, err := makeHTTPRequest(app, createReq)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}

	response := assertJSONResponse(t, w, http.StatusCreated)

	var commentID string
	if comment, ok := response["comment"].(map[string]interface{}); ok {
		if id, ok := comment["id"].(string); ok {
			commentID = id
		}
	}

	if commentID == "" {
		t.Fatal("Failed to get comment ID from create response")
	}

	// Retrieve comments
	getReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/comments/" + scenarioName,
	}

	w, err = makeHTTPRequest(app, getReq)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	response = assertJSONResponse(t, w, http.StatusOK)

	if totalCount, ok := response["total_count"].(float64); !ok || totalCount != 1 {
		t.Errorf("Expected total_count 1 after create, got %v", response["total_count"])
	}
}

func TestConcurrentCommentCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "concurrent-test"

	testDB.DB.CreateDefaultConfig(scenarioName)

	// Create multiple comments concurrently
	numComments := 10
	done := make(chan bool, numComments)

	for i := 0; i < numComments; i++ {
		go func(index int) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/comments/" + scenarioName,
				Body: map[string]interface{}{
					"content":      "Concurrent comment " + string(rune('A'+index)),
					"author_token": "test-token",
				},
			}

			w, err := makeHTTPRequest(app, req)
			if err != nil {
				t.Errorf("Failed to create concurrent comment: %v", err)
			}

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", w.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numComments; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent comment creation")
		}
	}

	// Verify all were created
	comments, totalCount, err := testDB.DB.GetComments(scenarioName, nil, 100, 0, "newest")
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if totalCount != numComments {
		t.Errorf("Expected %d comments, got %d", numComments, totalCount)
	}

	if len(comments) != numComments {
		t.Errorf("Expected %d comments in slice, got %d", numComments, len(comments))
	}
}
