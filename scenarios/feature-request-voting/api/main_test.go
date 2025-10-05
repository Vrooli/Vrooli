//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response != nil {
			if _, ok := response["timestamp"]; !ok {
				t.Error("Expected timestamp in health response")
			}
		}
	})
}

// TestListFeatureRequests tests listing feature requests
func TestListFeatureRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test scenario
		scenario := createTestScenario(t, ts.DB, "test-list-scenario")

		// Create feature requests
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Feature 1")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Feature 2")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars: map[string]string{"scenario_id": scenario.ID},
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			requests, ok := response["requests"].([]interface{})
			if !ok {
				t.Fatal("Expected requests array in response")
			}
			if len(requests) != 2 {
				t.Errorf("Expected 2 requests, got %d", len(requests))
			}
		}
	})

	t.Run("WithStatusFilter", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-filter-scenario")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Proposed Feature")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars:     map[string]string{"scenario_id": scenario.ID},
			QueryParams: map[string]string{"status": "proposed"},
		})

		assertJSONResponse(t, w, http.StatusOK, nil)
	})

	t.Run("WithSortOptions", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-sort-scenario")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Feature A")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Feature B")

		// Test sorting by date
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars:     map[string]string{"scenario_id": scenario.ID},
			QueryParams: map[string]string{"sort": "date"},
		})

		assertJSONResponse(t, w, http.StatusOK, nil)

		// Test sorting by priority
		w = makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars:     map[string]string{"scenario_id": scenario.ID},
			QueryParams: map[string]string{"sort": "priority"},
		})

		assertJSONResponse(t, w, http.StatusOK, nil)
	})

	t.Run("EmptyScenario", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-empty-scenario")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars: map[string]string{"scenario_id": scenario.ID},
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			requests := response["requests"].([]interface{})
			if len(requests) != 0 {
				t.Errorf("Expected 0 requests for empty scenario, got %d", len(requests))
			}
		}
	})
}

// TestCreateFeatureRequest tests creating feature requests
func TestCreateFeatureRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-create-scenario")

		payload := TestData.CreateFeatureRequestPayload(
			scenario.ID,
			"New Feature",
			"Feature description",
		)

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, ok := response["id"]; !ok {
				t.Error("Expected id in response")
			}
		}
	})

	t.Run("MissingTitle", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-missing-title")

		payload := map[string]interface{}{
			"scenario_id": scenario.ID,
			"description": "Description without title",
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("MissingDescription", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-missing-desc")

		payload := map[string]interface{}{
			"scenario_id": scenario.ID,
			"title":       "Title without description",
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("MissingScenarioID", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":       "Title without scenario",
			"description": "Description",
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   `{"invalid": "json"`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("WithTags", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-tags-scenario")

		payload := CreateFeatureRequestRequest{
			ScenarioID:  scenario.ID,
			Title:       "Feature with tags",
			Description: "Description",
			Tags:        []string{"urgent", "bug"},
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"success": true,
		})
	})
}

// TestGetFeatureRequest tests getting a single feature request
func TestGetFeatureRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-get-scenario")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Test Feature")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":    fr.ID,
			"title": fr.Title,
		})

		if response != nil {
			if _, ok := response["description"]; !ok {
				t.Error("Expected description in response")
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestUpdateFeatureRequest tests updating feature requests
func TestUpdateFeatureRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("UpdateTitle", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-update-title")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Original Title")

		newTitle := "Updated Title"
		payload := UpdateFeatureRequestRequest{
			Title: &newTitle,
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-update-status")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Status Test")

		newStatus := "shipped"
		payload := UpdateFeatureRequestRequest{
			Status: &newStatus,
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-update-multi")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Multi Update")

		newTitle := "New Title"
		newDesc := "New Description"
		newPriority := "high"

		payload := UpdateFeatureRequestRequest{
			Title:       &newTitle,
			Description: &newDesc,
			Priority:    &newPriority,
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("NoFieldsToUpdate", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-no-update")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "No Update")

		payload := UpdateFeatureRequestRequest{}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "No fields to update")
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		newTitle := "Updated"
		payload := UpdateFeatureRequestRequest{
			Title: &newTitle,
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
			Body:    payload,
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// TestDeleteFeatureRequest tests deleting feature requests
func TestDeleteFeatureRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-delete-scenario")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "To Delete")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		// Verify deletion
		w = makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
		})

		if w.Code != http.StatusNotFound {
			t.Error("Expected feature request to be deleted")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// TestVote tests the voting functionality
func TestVote(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("UpvoteSuccess", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-upvote-scenario")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Upvote Test")

		payload := TestData.VotePayload(1)

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"user_vote": float64(1),
		})

		if response != nil {
			voteCount, ok := response["vote_count"].(float64)
			if !ok || voteCount != 1 {
				t.Errorf("Expected vote_count to be 1, got %v", response["vote_count"])
			}
		}
	})

	t.Run("DownvoteSuccess", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-downvote-scenario")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Downvote Test")

		payload := TestData.VotePayload(-1)

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"user_vote": float64(-1),
		})
	})

	t.Run("InvalidVoteValue", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-invalid-vote")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Invalid Vote")

		payload := map[string]interface{}{
			"value": 5, // Invalid: must be 1 or -1
		}

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    payload,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "must be 1 or -1")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-vote-json")
		fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "JSON Test")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Body:    `{"invalid": "json"`,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestListScenarios tests listing scenarios
func TestListScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("Success", func(t *testing.T) {
		createTestScenario(t, ts.DB, "scenario-1")
		createTestScenario(t, ts.DB, "scenario-2")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var scenarios []interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
			t.Fatalf("Failed to parse scenarios response: %v", err)
		}

		if len(scenarios) < 2 {
			t.Errorf("Expected at least 2 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("Empty", func(t *testing.T) {
		// Clean all scenarios
		ts.DB.Exec("TRUNCATE TABLE scenarios CASCADE")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var scenarios []interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
			t.Fatalf("Failed to parse scenarios response: %v", err)
		}

		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(scenarios))
		}
	})
}

// TestGetScenario tests getting a single scenario
func TestGetScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	t.Run("ByID", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-get-by-id")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s", scenario.ID),
			URLVars: map[string]string{"id": scenario.ID},
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":           scenario.ID,
			"name":         scenario.Name,
			"display_name": scenario.DisplayName,
		})

		if response != nil {
			if _, ok := response["stats"]; !ok {
				t.Error("Expected stats in response")
			}
		}
	})

	t.Run("ByName", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-get-by-name")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s", scenario.Name),
			URLVars: map[string]string{"id": scenario.Name},
		})

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": scenario.Name,
		})
	})

	t.Run("WithStats", func(t *testing.T) {
		scenario := createTestScenario(t, ts.DB, "test-stats-scenario")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Request 1")
		createTestFeatureRequest(t, ts.DB, scenario.ID, "Request 2")

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s", scenario.ID),
			URLVars: map[string]string{"id": scenario.ID},
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			stats, ok := response["stats"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected stats object in response")
			}

			if totalRequests, ok := stats["total_requests"].(float64); !ok || totalRequests != 2 {
				t.Errorf("Expected total_requests to be 2, got %v", stats["total_requests"])
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}
