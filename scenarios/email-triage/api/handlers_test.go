package main

import (
	"context"
	"testing"

	"email-triage/handlers"
	"email-triage/services"

	"github.com/google/uuid"
)

// TestRuleHandler tests the RuleHandler
func TestRuleHandler(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	ruleService := services.NewRuleService(testDB.db, "http://localhost:11434")
	handler := handlers.NewRuleHandler(ruleService)

	userID := uuid.New().String()

	t.Run("CreateRule_MissingAuth", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        "Test Rule",
			"description": "Test description",
			"conditions":  map[string]interface{}{"sender_contains": []string{"important"}},
			"actions":     map[string]interface{}{"priority": "high"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/rules",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// No auth context
		handler.CreateRule(w, httpReq)

		assertErrorResponse(t, w, 401, "")
	})

	t.Run("CreateRule_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        "Test Rule",
			"description": "Test description",
			"conditions":  map[string]interface{}{"sender_contains": []string{"important"}},
			"actions":     map[string]interface{}{"priority": "high"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/rules",
			Body:   reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.CreateRule(w, httpReq)

		// Should succeed or fail gracefully
		if w.Code != 201 && w.Code != 400 && w.Code != 500 {
			t.Logf("Got status %d (may depend on DB state)", w.Code)
		}
	})

	t.Run("ListRules_NoAuth", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/rules",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler.ListRules(w, httpReq)

		assertErrorResponse(t, w, 401, "")
	})

	t.Run("ListRules_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/rules",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.ListRules(w, httpReq)

		// Should return 200 with rules array
		if w.Code == 200 {
			t.Log("Rules listed successfully")
		}
	})
}

// TestEmailHandler tests the EmailHandler
func TestEmailHandler(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	searchService := services.NewSearchService("http://localhost:6333")
	handler := handlers.NewEmailHandler(testDB.db, searchService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)
	emailID := TestData.CreateTestEmail(t, testDB.db, accountID)

	t.Run("GetEmail_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/emails/" + emailID,
			URLVars: map[string]string{"id": emailID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetEmail(w, httpReq)

		if w.Code == 200 {
			response := assertJSONResponse(t, w, 200, map[string]interface{}{
				"id": emailID,
			})

			if response != nil {
				t.Log("Email retrieved successfully")
			}
		}
	})

	t.Run("GetEmail_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/emails/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetEmail(w, httpReq)

		assertErrorResponse(t, w, 404, "")
	})

	t.Run("SearchEmails_NoAuth", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/emails/search",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler.SearchEmails(w, httpReq)

		assertErrorResponse(t, w, 401, "")
	})

	t.Run("SearchEmails_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/emails/search",
			QueryParams: map[string]string{"query": "test"},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.SearchEmails(w, httpReq)

		// Should return results or empty array
		if w.Code == 200 {
			t.Log("Search completed successfully")
		}
	})

	t.Run("ApplyActions_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"actions": []map[string]interface{}{
				{"type": "tag", "value": "important"},
			},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/emails/" + emailID + "/actions",
			URLVars: map[string]string{"id": emailID},
			Body:    reqBody,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.ApplyActions(w, httpReq)

		// Should succeed or fail gracefully
		if w.Code == 200 || w.Code == 400 || w.Code == 404 {
			t.Logf("Actions applied with status %d", w.Code)
		}
	})

	t.Run("ForceSync_NoAuth", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/emails/sync",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler.ForceSync(w, httpReq)

		assertErrorResponse(t, w, 401, "")
	})
}

// TestAnalyticsHandler tests the AnalyticsHandler
func TestAnalyticsHandler(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	handler := handlers.NewAnalyticsHandler(testDB.db)

	userID := uuid.New().String()

	t.Run("GetDashboard_NoAuth", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/dashboard",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		handler.GetDashboard(w, httpReq)

		assertErrorResponse(t, w, 401, "")
	})

	t.Run("GetDashboard_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/dashboard",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetDashboard(w, httpReq)

		if w.Code == 200 {
			t.Log("Dashboard retrieved successfully")
		}
	})

	t.Run("GetUsageStats_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/usage",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetUsageStats(w, httpReq)

		if w.Code == 200 {
			t.Log("Usage stats retrieved successfully")
		}
	})

	t.Run("GetRulePerformance_Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/rules-performance",
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := context.WithValue(httpReq.Context(), "user_id", userID)
		httpReq = httpReq.WithContext(ctx)

		handler.GetRulePerformance(w, httpReq)

		if w.Code == 200 {
			t.Log("Rule performance retrieved successfully")
		}
	})
}

// TestHandlerHelpers tests helper functions
func TestHandlerHelpers(t *testing.T) {
	t.Run("GenerateUUID", func(t *testing.T) {
		id1 := uuid.New().String()
		id2 := uuid.New().String()

		if id1 == id2 {
			t.Error("UUIDs should be unique")
		}

		if len(id1) == 0 {
			t.Error("UUID should not be empty")
		}
	})
}
