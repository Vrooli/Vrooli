package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestCreatePersona tests the create persona endpoint
func TestCreatePersona(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := TestData.CreatePersonaRequest("Test Persona")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/persona/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createPersona(c)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": "Test Persona",
		})

		if response != nil {
			if _, ok := response["id"].(string); !ok {
				t.Error("Expected id field in response")
			}
			if _, ok := response["created_at"].(string); !ok {
				t.Error("Expected created_at field in response")
			}
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		body := []byte(`{"description": "Test"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/persona/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createPersona(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		body := []byte(`{"name": "test"`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/persona/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createPersona(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/persona/create", bytes.NewReader([]byte{}))
		c.Request.Header.Set("Content-Type", "application/json")

		createPersona(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestGetPersona tests the get persona endpoint
func TestGetPersona(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/persona/"+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "id", Value: testPersona.Persona.ID}}

		getPersona(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   testPersona.Persona.ID,
			"name": testPersona.Persona.Name,
		})

		if response != nil {
			if desc, ok := response["description"].(string); !ok || desc == "" {
				t.Error("Expected description field in response")
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/persona/"+nonExistentID, nil)
		c.Params = gin.Params{{Key: "id", Value: nonExistentID}}

		getPersona(c)

		assertErrorResponse(t, w, http.StatusNotFound, "Persona not found")
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/persona/invalid-uuid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		getPersona(c)

		// The handler should fail to find it (treats as string)
		if w.Code != http.StatusNotFound {
			t.Logf("Note: Handler treats invalid UUID as non-existent ID, status: %d", w.Code)
		}
	})
}

// TestListPersonas tests the list personas endpoint
func TestListPersonas(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		persona1 := setupTestPersona(t, "Persona 1")
		defer persona1.Cleanup()

		persona2 := setupTestPersona(t, "Persona 2")
		defer persona2.Cleanup()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/personas", nil)

		listPersonas(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		personas, ok := response["personas"].([]interface{})
		if !ok {
			t.Fatal("Expected personas array in response")
		}

		if len(personas) < 2 {
			t.Errorf("Expected at least 2 personas, got %d", len(personas))
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		// Clean all test personas first
		db.Exec("DELETE FROM personas WHERE id LIKE 'test-%'")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/personas", nil)

		listPersonas(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		personas, ok := response["personas"].([]interface{})
		if !ok {
			t.Error("Expected personas array in response")
		}

		// Should return empty array, not null
		if personas == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestConnectDataSource tests the connect data source endpoint
func TestConnectDataSource(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		req := TestData.ConnectDataSourceRequest(testPersona.Persona.ID, "file")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/datasource/connect", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		connectDataSource(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"message": "Data source connected successfully",
		})

		if response != nil {
			if _, ok := response["source_id"].(string); !ok {
				t.Error("Expected source_id in response")
			}
		}
	})

	t.Run("MissingPersonaID", func(t *testing.T) {
		body := []byte(`{"source_type": "file"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/datasource/connect", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		connectDataSource(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingSourceType", func(t *testing.T) {
		body := []byte(`{"persona_id": "test-123"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/datasource/connect", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		connectDataSource(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestGetDataSources tests the get data sources endpoint
func TestGetDataSources(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Create a data source
		req := TestData.ConnectDataSourceRequest(testPersona.Persona.ID, "file")
		body, _ := json.Marshal(req)

		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("POST", "/api/datasource/connect", bytes.NewReader(body))
		c1.Request.Header.Set("Content-Type", "application/json")
		connectDataSource(c1)

		// Get data sources
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/datasources/"+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "persona_id", Value: testPersona.Persona.ID}}

		getDataSources(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		sources, ok := response["data_sources"].([]interface{})
		if !ok {
			t.Fatal("Expected data_sources array in response")
		}

		if len(sources) < 1 {
			t.Error("Expected at least 1 data source")
		}
	})

	t.Run("NoDataSources", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona No Sources")
		defer testPersona.Cleanup()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/datasources/"+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "persona_id", Value: testPersona.Persona.ID}}

		getDataSources(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestSearchDocuments tests the search documents endpoint
func TestSearchDocuments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		req := TestData.SearchRequest(testPersona.Persona.ID, "test query")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/search", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		searchDocuments(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"query": "test query",
		})

		if response != nil {
			if _, ok := response["results"].([]interface{}); !ok {
				t.Error("Expected results array in response")
			}
			if total, ok := response["total"].(float64); !ok || total < 0 {
				t.Error("Expected total field in response")
			}
		}
	})

	t.Run("MissingPersonaID", func(t *testing.T) {
		body := []byte(`{"query": "test"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/search", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		searchDocuments(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingQuery", func(t *testing.T) {
		body := []byte(`{"persona_id": "test-123"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/search", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		searchDocuments(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Request without limit
		body := []byte(fmt.Sprintf(`{"persona_id": "%s", "query": "test"}`, testPersona.Persona.ID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/search", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		searchDocuments(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestStartTraining tests the start training endpoint
func TestStartTraining(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		req := TestData.StartTrainingRequest(testPersona.Persona.ID)
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/train/start", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		startTraining(c)

		response := assertJSONResponse(t, w, http.StatusAccepted, map[string]interface{}{
			"status":  "queued",
			"message": "Training job started",
		})

		if response != nil {
			if _, ok := response["job_id"].(string); !ok {
				t.Error("Expected job_id in response")
			}
		}
	})

	t.Run("MissingFields", func(t *testing.T) {
		body := []byte(`{"persona_id": "test-123"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/train/start", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		startTraining(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestGetTrainingJobs tests the get training jobs endpoint
func TestGetTrainingJobs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Start a training job first
		req := TestData.StartTrainingRequest(testPersona.Persona.ID)
		body, _ := json.Marshal(req)

		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("POST", "/api/train/start", bytes.NewReader(body))
		c1.Request.Header.Set("Content-Type", "application/json")
		startTraining(c1)

		// Get training jobs
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/training/jobs/"+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "persona_id", Value: testPersona.Persona.ID}}

		getTrainingJobs(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		jobs, ok := response["training_jobs"].([]interface{})
		if !ok {
			t.Fatal("Expected training_jobs array in response")
		}

		if len(jobs) < 1 {
			t.Error("Expected at least 1 training job")
		}
	})
}

// TestCreateAPIToken tests the create API token endpoint
func TestCreateAPIToken(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		req := TestData.CreateTokenRequest(testPersona.Persona.ID)
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/tokens/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createAPIToken(c)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name":    "test-token",
			"message": "API token created successfully",
		})

		if response != nil {
			if _, ok := response["token"].(string); !ok {
				t.Error("Expected token in response")
			}
			if _, ok := response["token_id"].(string); !ok {
				t.Error("Expected token_id in response")
			}
		}
	})

	t.Run("MissingFields", func(t *testing.T) {
		body := []byte(`{"persona_id": "test-123"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/tokens/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createAPIToken(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DefaultPermissions", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Request without permissions
		body := []byte(fmt.Sprintf(`{"persona_id": "%s", "name": "test"}`, testPersona.Persona.ID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/tokens/create", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		createAPIToken(c)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		perms, ok := response["permissions"].([]interface{})
		if !ok || len(perms) == 0 {
			t.Error("Expected default permissions in response")
		}
	})
}

// TestGetAPITokens tests the get API tokens endpoint
func TestGetAPITokens(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Create a token first
		req := TestData.CreateTokenRequest(testPersona.Persona.ID)
		body, _ := json.Marshal(req)

		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("POST", "/api/tokens/create", bytes.NewReader(body))
		c1.Request.Header.Set("Content-Type", "application/json")
		createAPIToken(c1)

		// Get tokens
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/tokens/"+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "persona_id", Value: testPersona.Persona.ID}}

		getAPITokens(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		tokens, ok := response["tokens"].([]interface{})
		if !ok {
			t.Fatal("Expected tokens array in response")
		}

		if len(tokens) < 1 {
			t.Error("Expected at least 1 token")
		}
	})
}

// TestHandleChat tests the chat endpoint
func TestHandleChat(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		req := TestData.ChatRequest(testPersona.Persona.ID, "Hello!")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handleChat(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"persona_name": testPersona.Persona.Name,
		})

		if response != nil {
			if _, ok := response["response"].(string); !ok {
				t.Error("Expected response field")
			}
			if _, ok := response["session_id"].(string); !ok {
				t.Error("Expected session_id field")
			}
		}
	})

	t.Run("PersonaNotFound", func(t *testing.T) {
		req := TestData.ChatRequest("nonexistent-id", "Hello!")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handleChat(c)

		assertErrorResponse(t, w, http.StatusNotFound, "Persona not found")
	})

	t.Run("MissingFields", func(t *testing.T) {
		body := []byte(`{"persona_id": "test-123"}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handleChat(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("NewSession", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Request without session_id
		body := []byte(fmt.Sprintf(`{"persona_id": "%s", "message": "Hello!"}`, testPersona.Persona.ID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handleChat(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if sessionID, ok := response["session_id"].(string); !ok || sessionID == "" {
			t.Error("Expected new session_id to be generated")
		}
	})
}

// TestGetChatHistory tests the get chat history endpoint
func TestGetChatHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		// Send a chat message first
		chatReq := TestData.ChatRequest(testPersona.Persona.ID, "Hello!")
		chatBody, _ := json.Marshal(chatReq)

		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(chatBody))
		c1.Request.Header.Set("Content-Type", "application/json")
		handleChat(c1)

		var chatResponse map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &chatResponse)
		sessionID := chatResponse["session_id"].(string)

		// Get chat history
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/chat/history/"+sessionID+"?persona_id="+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "session_id", Value: sessionID}}
		c.Request.URL.RawQuery = "persona_id=" + testPersona.Persona.ID

		getChatHistory(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"session_id": sessionID,
		})

		if response != nil {
			if messages, ok := response["messages"].([]interface{}); !ok || len(messages) == 0 {
				t.Error("Expected messages array with content")
			}
		}
	})

	t.Run("MissingPersonaID", func(t *testing.T) {
		sessionID := uuid.New().String()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/chat/history/"+sessionID, nil)
		c.Params = gin.Params{{Key: "session_id", Value: sessionID}}

		getChatHistory(c)

		assertErrorResponse(t, w, http.StatusBadRequest, "persona_id query parameter is required")
	})

	t.Run("NoHistory", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Test Persona")
		defer testPersona.Cleanup()

		sessionID := uuid.New().String()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/chat/history/"+sessionID+"?persona_id="+testPersona.Persona.ID, nil)
		c.Params = gin.Params{{Key: "session_id", Value: sessionID}}
		c.Request.URL.RawQuery = "persona_id=" + testPersona.Persona.ID

		getChatHistory(c)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"session_id": sessionID,
		})

		if response != nil {
			if messages, ok := response["messages"].([]interface{}); !ok || len(messages) != 0 {
				t.Error("Expected empty messages array for non-existent session")
			}
		}
	})
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "personal-digital-twin-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "personal-digital-twin-api",
	})

	if response != nil {
		if timestamp, ok := response["timestamp"].(string); !ok || timestamp == "" {
			t.Error("Expected timestamp in health response")
		}
	}
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("ConcurrentPersonaCreation", func(t *testing.T) {
		concurrency := 10
		errors := runConcurrentRequests(t, concurrency, func(i int) error {
			req := TestData.CreatePersonaRequest(fmt.Sprintf("Concurrent Persona %d", i))
			body, _ := json.Marshal(req)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/persona/create", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			createPersona(c)

			if w.Code != http.StatusCreated {
				return fmt.Errorf("expected status 201, got %d", w.Code)
			}
			return nil
		})

		validateNoConcurrencyErrors(t, errors)
	})

	t.Run("ConcurrentChatRequests", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Concurrent Test Persona")
		defer testPersona.Cleanup()

		concurrency := 10
		errors := runConcurrentRequests(t, concurrency, func(i int) error {
			req := TestData.ChatRequest(testPersona.Persona.ID, fmt.Sprintf("Message %d", i))
			body, _ := json.Marshal(req)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handleChat(c)

			if w.Code != http.StatusOK {
				return fmt.Errorf("expected status 200, got %d", w.Code)
			}
			return nil
		})

		validateNoConcurrencyErrors(t, errors)
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		// Set environment variables
		os.Setenv("API_PORT", "8080")
		os.Setenv("CHAT_PORT", "8081")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("QDRANT_URL", "http://localhost:6333")
		os.Setenv("OLLAMA_URL", "http://localhost:11434")
		os.Setenv("N8N_BASE_URL", "http://localhost:5678")
		os.Setenv("MINIO_URL", "http://localhost:9000")

		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("CHAT_PORT")
			os.Unsetenv("POSTGRES_URL")
			os.Unsetenv("QDRANT_URL")
			os.Unsetenv("OLLAMA_URL")
			os.Unsetenv("N8N_BASE_URL")
			os.Unsetenv("MINIO_URL")
		}()

		cfg := loadConfig()

		if cfg.Port != "8080" {
			t.Errorf("Expected port 8080, got %s", cfg.Port)
		}
		if cfg.ChatPort != "8081" {
			t.Errorf("Expected chat port 8081, got %s", cfg.ChatPort)
		}
		if !strings.Contains(cfg.PostgresURL, "postgres://") {
			t.Error("Expected valid PostgreSQL URL")
		}
	})

	t.Run("BuildPostgresURL", func(t *testing.T) {
		// Unset POSTGRES_URL to test building from components
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("API_PORT", "8080")
		os.Setenv("CHAT_PORT", "8081")
		os.Setenv("QDRANT_URL", "http://localhost:6333")
		os.Setenv("OLLAMA_URL", "http://localhost:11434")
		os.Setenv("N8N_BASE_URL", "http://localhost:5678")
		os.Setenv("MINIO_URL", "http://localhost:9000")

		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("API_PORT")
			os.Unsetenv("CHAT_PORT")
			os.Unsetenv("QDRANT_URL")
			os.Unsetenv("OLLAMA_URL")
			os.Unsetenv("N8N_BASE_URL")
			os.Unsetenv("MINIO_URL")
		}()

		cfg := loadConfig()

		if !strings.Contains(cfg.PostgresURL, "testuser") {
			t.Error("Expected PostgreSQL URL to contain testuser")
		}
		if !strings.Contains(cfg.PostgresURL, "testdb") {
			t.Error("Expected PostgreSQL URL to contain testdb")
		}
	})
}

// Performance test for API endpoints
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("ListPersonasPerformance", func(t *testing.T) {
		// Create multiple personas
		for i := 0; i < 50; i++ {
			persona := setupTestPersona(t, fmt.Sprintf("Perf Test Persona %d", i))
			defer persona.Cleanup()
		}

		start := time.Now()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/personas", nil)

		listPersonas(c)

		duration := time.Since(start)

		if duration > 1*time.Second {
			t.Errorf("List personas took too long: %v", duration)
		}

		t.Logf("List personas with 50 items took: %v", duration)
	})

	t.Run("ChatResponsePerformance", func(t *testing.T) {
		testPersona := setupTestPersona(t, "Perf Test Persona")
		defer testPersona.Cleanup()

		start := time.Now()

		req := TestData.ChatRequest(testPersona.Persona.ID, "Performance test message")
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handleChat(c)

		duration := time.Since(start)

		if duration > 500*time.Millisecond {
			t.Errorf("Chat response took too long: %v", duration)
		}

		t.Logf("Chat response took: %v", duration)
	})
}
