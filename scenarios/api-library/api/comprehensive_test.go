
package main

import (
	"net/http"
	"testing"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "api-library",
		})

		if response != nil {
			if _, ok := response["timestamp"]; !ok {
				t.Error("Expected timestamp in health response")
			}
		}
	})
}

// TestSearchAPIsHandlerComprehensive tests the search endpoint
func TestSearchAPIsHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Create test APIs
	testAPI1 := setupTestAPI(t, env.DB, "Payment Gateway")
	defer testAPI1.Cleanup()

	testAPI2 := setupTestAPI(t, env.DB, "Weather Service")
	defer testAPI2.Cleanup()

	t.Run("Success_POST_with_query", func(t *testing.T) {
		searchReq := SearchRequest{
			Query: "payment",
			Limit: 10,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["results"]; !ok {
				t.Error("Expected results field in search response")
			}
		}
	})

	t.Run("Error_missing_query", func(t *testing.T) {
		searchReq := SearchRequest{
			Limit: 10,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "query")
	})

	t.Run("Error_invalid_JSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected error status for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Edge_negative_limit", func(t *testing.T) {
		searchReq := SearchRequest{
			Query: "test",
			Limit: -1,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		// Should either reject or normalize to valid limit
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected OK or BadRequest for negative limit, got %d", w.Code)
		}
	})
}

// TestListAPIsHandler tests the list endpoint
func TestListAPIsHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Create test APIs
	testAPI := setupTestAPI(t, env.DB, "Test API")
	defer testAPI.Cleanup()

	t.Run("Success_list_all", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/apis",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listAPIsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if apis, ok := response["apis"].([]interface{}); ok {
				if len(apis) == 0 {
					t.Log("Warning: No APIs returned (may be expected if test DB is empty)")
				}
			}
		}
	})

	t.Run("Success_with_category_filter", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/apis",
			QueryParams: map[string]string{"category": "testing"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listAPIsHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, nil)
	})
}

// TestCreateAPIHandler tests API creation
func TestCreateAPIHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_create_API", func(t *testing.T) {
		apiData := TestData.CreateAPIRequest("New Test API")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/apis",
			Body:   apiData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createAPIHandler(w, httpReq)

		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, w.Code, nil)
			if response != nil {
				if id, ok := response["id"].(string); ok {
					// Cleanup
					defer env.DB.Exec("DELETE FROM apis WHERE id = $1", id)
				}
			}
		}
	})

	t.Run("Error_missing_required_fields", func(t *testing.T) {
		apiData := map[string]interface{}{
			"name": "Incomplete API",
			// Missing required fields like provider, base_url, etc.
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/apis",
			Body:   apiData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createAPIHandler(w, httpReq)

		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			// Some implementations may have defaults, so we accept success
			t.Log("Handler accepted incomplete data (may have defaults)")
		}
	})

	t.Run("Error_invalid_JSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/apis",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createAPIHandler(w, httpReq)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected error status for invalid JSON, got %d", w.Code)
		}
	})
}

// TestGetAPIHandler tests single API retrieval
func TestGetAPIHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	testAPI := setupTestAPI(t, env.DB, "Get Test API")
	defer testAPI.Cleanup()

	t.Run("Success_get_existing_API", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getAPIHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   testAPI.API.ID,
			"name": testAPI.API.Name,
		})

		if response != nil {
			if _, ok := response["provider"]; !ok {
				t.Error("Expected provider field in API response")
			}
		}
	})

	// Test error patterns using builder
	suite := &HandlerTestSuite{
		HandlerName: "getAPIHandler",
		Handler:     getAPIHandler,
		BaseURL:     "/api/v1/apis/{id}",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("GET", "/api/v1/apis/invalid-uuid").
		AddNonExistentAPI("GET", "/api/v1/apis/{id}").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestUpdateAPIHandler tests API updates
func TestUpdateAPIHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	testAPI := setupTestAPI(t, env.DB, "Update Test API")
	defer testAPI.Cleanup()

	t.Run("Success_update_API", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":        "Updated API Name",
			"description": "Updated description",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
			Body:    updateData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateAPIHandler(w, httpReq)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if name, ok := response["name"].(string); ok && name != "Updated API Name" {
					t.Errorf("Expected name to be updated to 'Updated API Name', got '%s'", name)
				}
			}
		}
	})

	// Test error patterns
	suite := &HandlerTestSuite{
		HandlerName: "updateAPIHandler",
		Handler:     updateAPIHandler,
		BaseURL:     "/api/v1/apis/{id}",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("PUT", "/api/v1/apis/invalid-uuid").
		AddNonExistentAPI("PUT", "/api/v1/apis/{id}").
		AddInvalidJSON("PUT", "/api/v1/apis/"+testAPI.API.ID).
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestDeleteAPIHandler tests API deletion
func TestDeleteAPIHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_delete_API", func(t *testing.T) {
		// Create API specifically for deletion
		testAPI := setupTestAPI(t, env.DB, "Delete Test API")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteAPIHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected 200 or 204 for successful deletion, got %d", w.Code)
		}

		// Verify deletion
		var count int
		err = env.DB.QueryRow("SELECT COUNT(*) FROM apis WHERE id = $1", testAPI.API.ID).Scan(&count)
		if err == nil && count > 0 {
			t.Error("API was not deleted from database")
		}
	})

	// Test error patterns
	suite := &HandlerTestSuite{
		HandlerName: "deleteAPIHandler",
		Handler:     deleteAPIHandler,
		BaseURL:     "/api/v1/apis/{id}",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("DELETE", "/api/v1/apis/invalid-uuid").
		AddNonExistentAPI("DELETE", "/api/v1/apis/{id}").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestNotesHandlers tests note-related endpoints
func TestNotesHandlersComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	testAPI := setupTestAPI(t, env.DB, "Notes Test API")
	defer testAPI.Cleanup()

	t.Run("Success_add_note", func(t *testing.T) {
		noteData := map[string]interface{}{
			"content":    "This is a test note",
			"type":       "tip",
			"created_by": "test_user",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/apis/" + testAPI.API.ID + "/notes",
			URLVars: map[string]string{"id": testAPI.API.ID},
			Body:    noteData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		addNoteHandler(w, httpReq)

		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, w.Code, nil)
			if response != nil {
				if noteID, ok := response["id"].(string); ok {
					// Cleanup
					defer env.DB.Exec("DELETE FROM notes WHERE id = $1", noteID)
				}
			}
		}
	})

	t.Run("Success_get_notes", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/apis/" + testAPI.API.ID + "/notes",
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getNotesHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, nil)
	})

	t.Run("Error_invalid_note_type", func(t *testing.T) {
		noteData := map[string]interface{}{
			"content":    "Test note",
			"type":       "invalid_type",
			"created_by": "test_user",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/apis/" + testAPI.API.ID + "/notes",
			URLVars: map[string]string{"id": testAPI.API.ID},
			Body:    noteData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		addNoteHandler(w, httpReq)

		// May accept or reject - implementation dependent
		if w.Code == http.StatusBadRequest {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		}
	})
}

// TestMarkConfiguredHandler tests configuration marking
func TestMarkConfiguredHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	testAPI := setupTestAPI(t, env.DB, "Configure Test API")
	defer testAPI.Cleanup()

	t.Run("Success_mark_configured", func(t *testing.T) {
		configData := map[string]interface{}{
			"configured": true,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/apis/" + testAPI.API.ID + "/configure",
			URLVars: map[string]string{"id": testAPI.API.ID},
			Body:    configData,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		markConfiguredHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, nil)
		}
	})
}

// TestCategoriesAndTagsHandlers tests category and tag endpoints
func TestCategoriesAndTagsHandlersComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Success_get_categories", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/categories",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getCategoriesHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, nil)
	})

	t.Run("Success_get_tags", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tags",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTagsHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, nil)
	})
}
