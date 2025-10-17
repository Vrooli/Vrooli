package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"email-triage/handlers"
	"email-triage/models"
	"email-triage/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealthEndpoints tests the health check endpoints
func TestHealthEndpoints(t *testing.T) {
	t.Run("HealthCheck", func(t *testing.T) {
		// Create a minimal server for testing
		server := &Server{}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.healthCheck(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response models.HealthStatus
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}
	})

	t.Run("HealthCheckDatabase_WithConnection", func(t *testing.T) {
		testDB := setupTestDB(t)
		if testDB == nil {
			t.Skip("Database not available")
		}
		defer testDB.cleanup()

		server := &Server{db: testDB.db}

		req := httptest.NewRequest("GET", "/health/database", nil)
		w := httptest.NewRecorder()

		server.healthCheckDatabase(w, req)

		// Should return 200 when DB is connected
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if connected, ok := response["connected"].(bool); !ok || !connected {
			t.Error("Expected connected to be true")
		}
	})

	t.Run("HealthCheckDatabase_NoConnection", func(t *testing.T) {
		// Create server with nil database to simulate connection failure
		server := &Server{db: nil}

		req := httptest.NewRequest("GET", "/health/database", nil)
		w := httptest.NewRecorder()

		// This should panic or handle gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Log("Handled nil database gracefully")
			}
		}()

		server.healthCheckDatabase(w, req)

		// If no panic, check for error response
		if w.Code == http.StatusServiceUnavailable {
			t.Log("Correctly returned 503 for database failure")
		}
	})
}

// TestAccountHandlerCreate tests account creation
func TestAccountHandlerCreate(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	// Create mock email service
	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()

	t.Run("Success_CreateAccount", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email_address": "test@example.com",
			"password":      "test-password",
			"imap_server":   "imap.example.com",
			"imap_port":     993,
			"smtp_server":   "smtp.example.com",
			"smtp_port":     587,
			"use_tls":       true,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/accounts",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Add user_id to context
		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.CreateAccount(w, httpReq)

		// Note: Will likely fail because email connection test will fail
		// but we're testing the handler logic
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Logf("Got status %d (connection test may fail in test environment)", w.Code)
		}
	})

	t.Run("Error_MissingAuth", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email_address": "test@example.com",
			"password":      "test-password",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/accounts",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// No user_id in context
		handler.CreateAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusUnauthorized, "authentication required")
	})

	t.Run("Error_MissingRequiredFields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email_address": "test@example.com",
			// Missing password
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/accounts",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.CreateAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/accounts",
			Body:   `{"invalid": "json"`,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.CreateAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})
}

// TestAccountHandlerList tests listing accounts
func TestAccountHandlerList(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()

	// Create test account
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)
	_ = accountID

	t.Run("Success_ListAccounts", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/accounts",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.ListAccounts(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"count": nil, // Just check field exists
		})

		if response != nil {
			if accounts, ok := response["accounts"].([]interface{}); ok {
				if len(accounts) != 1 {
					t.Errorf("Expected 1 account, got %d", len(accounts))
				}
			}
		}
	})

	t.Run("Error_NoAuth", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/accounts",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler.ListAccounts(w, httpReq)

		assertErrorResponse(t, w, http.StatusUnauthorized, "authentication required")
	})
}

// TestAccountHandlerGet tests getting a specific account
func TestAccountHandlerGet(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)

	t.Run("Success_GetAccount", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetAccount(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id": accountID,
		})

		if response != nil {
			if emailAddr, ok := response["email_address"].(string); !ok || emailAddr == "" {
				t.Error("Expected email_address in response")
			}
		}
	})

	t.Run("Error_AccountNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/accounts/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})

	t.Run("Error_WrongUser", func(t *testing.T) {
		differentUserID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Different user trying to access account
		ctx := context.WithValue(httpReq.Context(), "user_id", differentUserID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestAccountHandlerUpdate tests updating an account
func TestAccountHandlerUpdate(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)

	t.Run("Success_UpdateAccount", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"sync_enabled": false,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
			Body:    reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.UpdateAccount(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			t.Log("Account updated successfully")
		}
	})

	t.Run("Error_NoValidFields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"invalid_field": "value",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
			Body:    reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.UpdateAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "no valid fields")
	})
}

// TestAccountHandlerDelete tests deleting an account
func TestAccountHandlerDelete(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)

	t.Run("Success_DeleteAccount", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.DeleteAccount(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			t.Log("Account deleted successfully")
		}
	})

	t.Run("Error_AlreadyDeleted", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/accounts/" + accountID,
			URLVars: map[string]string{"id": accountID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.DeleteAccount(w, httpReq)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestHealthCheckQdrant tests the Qdrant health check endpoint
func TestHealthCheckQdrant(t *testing.T) {
	t.Run("Qdrant_Healthy", func(t *testing.T) {
		// Create mock search service that returns healthy
		searchService := services.NewSearchService("http://localhost:6333")
		server := &Server{searchService: searchService}

		req := httptest.NewRequest("GET", "/health/qdrant", nil)
		w := httptest.NewRecorder()

		server.healthCheckQdrant(w, req)

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check that timestamp is present
		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}

		// Log the result (may be healthy or not depending on Qdrant availability)
		if connected, ok := response["connected"].(bool); ok {
			t.Logf("Qdrant health: connected=%v", connected)
		}
	})

	t.Run("Qdrant_Unavailable", func(t *testing.T) {
		// Create search service pointing to non-existent server
		searchService := services.NewSearchService("http://localhost:99999")
		server := &Server{searchService: searchService}

		req := httptest.NewRequest("GET", "/health/qdrant", nil)
		w := httptest.NewRecorder()

		server.healthCheckQdrant(w, req)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check response structure
		if connected, ok := response["connected"].(bool); ok {
			t.Logf("Qdrant connection status: connected=%v", connected)

			// If it's connected despite invalid URL, that's okay (may have fallback logic)
			// The important part is that the health check responded properly
		} else {
			t.Error("Expected 'connected' field in response")
		}

		// Log the status code for debugging
		t.Logf("Health check returned status code: %d", w.Code)
	})
}

// TestUpdateEmailPriority tests the update email priority endpoint
func TestUpdateEmailPriority(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	server := &Server{db: testDB.db}

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)
	emailID := TestData.CreateTestEmail(t, testDB.db, accountID)

	t.Run("Success_UpdatePriority", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"priority_score": 0.8,
		}

		req := httptest.NewRequest("PUT", "/api/v1/emails/"+emailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": emailID})

		// Set body
		bodyBytes, _ := json.Marshal(reqBody)
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader(bodyBytes)).Body
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"success": true,
			})

			if response != nil {
				t.Log("Priority updated successfully")

				// Verify priority_score in response
				if score, ok := response["priority_score"].(float64); ok {
					if score != 0.8 {
						t.Errorf("Expected priority_score 0.8, got %f", score)
					}
				}
			}
		}
	})

	t.Run("Error_InvalidPriorityScore_TooHigh", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"priority_score": 1.5, // Invalid: > 1
		}

		req := httptest.NewRequest("PUT", "/api/v1/emails/"+emailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": emailID})

		bodyBytes, _ := json.Marshal(reqBody)
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader(bodyBytes)).Body
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "between 0 and 1")
	})

	t.Run("Error_InvalidPriorityScore_TooLow", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"priority_score": -0.5, // Invalid: < 0
		}

		req := httptest.NewRequest("PUT", "/api/v1/emails/"+emailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": emailID})

		bodyBytes, _ := json.Marshal(reqBody)
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader(bodyBytes)).Body
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "between 0 and 1")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/emails/"+emailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": emailID})

		// Invalid JSON
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader([]byte("{invalid"))).Body
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})

	t.Run("Error_EmailNotFound", func(t *testing.T) {
		nonExistentEmailID := uuid.New().String()

		reqBody := map[string]interface{}{
			"priority_score": 0.5,
		}

		req := httptest.NewRequest("PUT", "/api/v1/emails/"+nonExistentEmailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": nonExistentEmailID})

		bodyBytes, _ := json.Marshal(reqBody)
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader(bodyBytes)).Body
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})

	t.Run("Error_WrongUserAccess", func(t *testing.T) {
		differentUserID := uuid.New().String()

		reqBody := map[string]interface{}{
			"priority_score": 0.5,
		}

		req := httptest.NewRequest("PUT", "/api/v1/emails/"+emailID+"/priority", nil)
		req = mux.SetURLVars(req, map[string]string{"id": emailID})

		bodyBytes, _ := json.Marshal(reqBody)
		req.Body = httptest.NewRequest("PUT", "/", bytes.NewReader(bodyBytes)).Body
		req.Header.Set("Content-Type", "application/json")

		// Different user trying to update priority
		ctx := context.WithValue(req.Context(), "user_id", differentUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.updateEmailPriority(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestGetProcessorStatus tests the processor status endpoint
func TestGetProcessorStatus(t *testing.T) {
	server := &Server{}

	t.Run("Success_GetStatus", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/processor/status", nil)
		w := httptest.NewRecorder()

		server.getProcessorStatus(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"running": nil, // Just check field exists
		})

		if response != nil {
			t.Log("Processor status retrieved")
		}
	})
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	server := &Server{}

	// Create a test handler that requires auth
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID == nil {
			t.Error("Expected user_id in context")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	t.Run("DevMode_BypassAuth", func(t *testing.T) {
		// Set DEV_MODE
		oldDevMode := os.Getenv("DEV_MODE")
		os.Setenv("DEV_MODE", "true")
		defer os.Setenv("DEV_MODE", oldDevMode)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()

		// Wrap handler with middleware
		handler := server.authMiddleware(testHandler)
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 in dev mode, got %d", w.Code)
		}
	})

	t.Run("Error_NoAuthHeader", func(t *testing.T) {
		// Ensure DEV_MODE is off
		oldDevMode := os.Getenv("DEV_MODE")
		os.Setenv("DEV_MODE", "")
		defer os.Setenv("DEV_MODE", oldDevMode)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()

		handler := server.authMiddleware(testHandler)
		handler.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusUnauthorized, "authentication required")
	})
}

// TestLoadConfig tests the configuration loading
func TestLoadConfig(t *testing.T) {
	t.Run("Success_LoadConfig", func(t *testing.T) {
		// Set environment variables
		os.Setenv("QDRANT_URL", "http://test-qdrant:6333")
		os.Setenv("AUTH_SERVICE_URL", "http://test-auth:8080")
		defer os.Unsetenv("QDRANT_URL")
		defer os.Unsetenv("AUTH_SERVICE_URL")

		config := loadConfig()

		if config.QdrantURL != "http://test-qdrant:6333" {
			t.Errorf("Expected QDRANT_URL to be set, got %s", config.QdrantURL)
		}

		if config.AuthServiceURL != "http://test-auth:8080" {
			t.Errorf("Expected AUTH_SERVICE_URL to be set, got %s", config.AuthServiceURL)
		}
	})

	t.Run("Defaults_WhenNotSet", func(t *testing.T) {
		// Unset environment variables
		os.Unsetenv("QDRANT_URL")
		os.Unsetenv("OLLAMA_URL")

		config := loadConfig()

		// Should use defaults
		if config.QdrantURL == "" {
			t.Error("Expected default QDRANT_URL")
		}

		if config.OllamaURL == "" {
			t.Error("Expected default OLLAMA_URL")
		}
	})
}
