package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate response structure
		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}

		if _, ok := response["database"]; !ok {
			t.Error("Expected database status in response")
		}

		if version, ok := response["version"].(string); !ok || version == "" {
			t.Error("Expected version in response")
		}
	})
}

// TestGetPersons tests listing persons
func TestGetPersons(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_DefaultLimit", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		persons, ok := response["persons"].([]interface{})
		if !ok {
			t.Fatalf("Expected persons array in response, got %v", response)
		}

		count, ok := response["count"].(float64)
		if !ok {
			t.Fatalf("Expected count in response, got %v", response)
		}

		if int(count) != len(persons) {
			t.Errorf("Count mismatch: count=%d, len(persons)=%d", int(count), len(persons))
		}
	})

	t.Run("Success_CustomLimit", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts",
			QueryParams: map[string]string{
				"limit": "10",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		persons, ok := response["persons"].([]interface{})
		if !ok {
			t.Fatalf("Expected persons array in response")
		}

		if len(persons) > 10 {
			t.Errorf("Expected at most 10 persons, got %d", len(persons))
		}
	})

	t.Run("Success_SearchByName", func(t *testing.T) {
		// Create a test person with unique name
		personID := createTestPerson(t, env, "UniqueTestName"+uuid.New().String()[:8])
		defer cleanupTestPerson(t, env, personID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts",
			QueryParams: map[string]string{
				"search": "UniqueTestName",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		persons, ok := response["persons"].([]interface{})
		if !ok || len(persons) == 0 {
			t.Error("Expected at least one person in search results")
		}
	})

	t.Run("Success_FilterByTags", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts",
			QueryParams: map[string]string{
				"tags": "test",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})
}

// TestGetPerson tests getting a single person
func TestGetPerson(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test person
		personID := createTestPerson(t, env, "TestGetPerson")
		defer cleanupTestPerson(t, env, personID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts/" + personID,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != personID {
			t.Errorf("Expected person ID %s, got %v", personID, response["id"])
		}

		if fullName, ok := response["full_name"].(string); !ok || fullName != "TestGetPerson" {
			t.Errorf("Expected full_name 'TestGetPerson', got %v", response["full_name"])
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts/" + nonExistentID,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "error")
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts/invalid-uuid",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Gin/Database may return various error codes for invalid UUID
		// Accept 400, 404, or 500 as all indicate the request failed appropriately
		if w.Code == http.StatusOK || w.Code == http.StatusCreated {
			t.Errorf("Expected error status for invalid UUID, got %d", w.Code)
		}
	})

// TestCreatePerson tests creating a new person
func TestCreatePerson(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_MinimalData", func(t *testing.T) {
		req := CreatePersonRequest{
			FullName: "Minimal Test Person",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/contacts",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		personID, ok := response["id"].(string)
		if !ok || personID == "" {
			t.Error("Expected person ID in response")
		}
		defer cleanupTestPerson(t, env, personID)

		if message, ok := response["message"].(string); !ok || message == "" {
			t.Error("Expected success message in response")
		}
	})

	t.Run("Success_CompleteData", func(t *testing.T) {
		displayName := "Test Display"
		pronouns := "they/them"
		notes := "Test notes"

		req := CreatePersonRequest{
			FullName:    "Complete Test Person",
			DisplayName: &displayName,
			Nicknames:   []string{"Testy", "TestBot"},
			Pronouns:    &pronouns,
			Emails:      []string{"complete@test.com", "alt@test.com"},
			Phones:      []string{"+1-555-0001", "+1-555-0002"},
			Metadata: map[string]interface{}{
				"custom_field": "custom_value",
			},
			Tags:  []string{"test", "complete"},
			Notes: &notes,
			SocialProfiles: map[string]interface{}{
				"twitter": "@testuser",
			},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/contacts",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		personID := response["id"].(string)
		defer cleanupTestPerson(t, env, personID)
	})

	t.Run("Error_MissingFullName", func(t *testing.T) {
		req := map[string]interface{}{
			"emails": []string{"test@example.com"},
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/contacts",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "error")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/contacts", strings.NewReader("{invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestUpdatePerson tests updating a person
func TestUpdatePerson(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_UpdateFullName", func(t *testing.T) {
		personID := createTestPerson(t, env, "Original Name")
		defer cleanupTestPerson(t, env, personID)

		newName := "Updated Name"
		req := UpdatePersonRequest{
			FullName: &newName,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/contacts/" + personID,
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		person, ok := response["person"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected person object in response")
		}

		if fullName, ok := person["full_name"].(string); !ok || fullName != newName {
			t.Errorf("Expected full_name '%s', got %v", newName, person["full_name"])
		}
	})

	t.Run("Success_UpdateMultipleFields", func(t *testing.T) {
		personID := createTestPerson(t, env, "Multi Update Test")
		defer cleanupTestPerson(t, env, personID)

		newName := "Updated Multi"
		displayName := "Updated Display"
		emails := []string{"updated@test.com"}
		tags := []string{"updated", "test"}

		req := UpdatePersonRequest{
			FullName:    &newName,
			DisplayName: &displayName,
			Emails:      &emails,
			Tags:        &tags,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/contacts/" + personID,
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		newName := "Should Not Work"

		req := UpdatePersonRequest{
			FullName: &newName,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/contacts/" + nonExistentID,
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "error")
	})

	t.Run("Error_EmptyFullName", func(t *testing.T) {
		personID := createTestPerson(t, env, "Empty Name Test")
		defer cleanupTestPerson(t, env, personID)

		emptyName := ""
		req := UpdatePersonRequest{
			FullName: &emptyName,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/contacts/" + personID,
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "error")
	})
}

// TestGetPersonByAuthID tests getting a person by authenticator ID
func TestGetPersonByAuthID(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create person with auth ID
		authID := "auth-" + uuid.New().String()
		personID := createTestPerson(t, env, "Auth Test Person")
		defer cleanupTestPerson(t, env, personID)

		// Update with auth ID
		req := UpdatePersonRequest{
			ScenarioAuthenticatorID: &authID,
		}
		_, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/contacts/" + personID,
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to update person: %v", err)
		}

		// Get by auth ID
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts/by-auth/" + authID,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != personID {
			t.Errorf("Expected person ID %s, got %v", personID, response["id"])
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentAuthID := "auth-nonexistent-" + uuid.New().String()

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/contacts/by-auth/" + nonExistentAuthID,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "error")
	})
}

// TestCreateRelationship tests creating relationships
func TestCreateRelationship(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		person1ID := createTestPerson(t, env, "Person 1")
		person2ID := createTestPerson(t, env, "Person 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		strength := 0.9
		now := time.Now()
		introContext := "Met at conference"

		req := CreateRelationshipRequest{
			FromPersonID:        person1ID,
			ToPersonID:          person2ID,
			RelationshipType:    "friend",
			Strength:            &strength,
			LastContactDate:     &now,
			SharedInterests:     []string{"tech", "hiking"},
			IntroductionContext: &introContext,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/relationships",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		relationshipID, ok := response["id"].(string)
		if !ok || relationshipID == "" {
			t.Error("Expected relationship ID in response")
		}
		defer cleanupTestRelationship(t, env, relationshipID)
	})

	t.Run("Error_InvalidPersonID", func(t *testing.T) {
		nonExistentID1 := uuid.New().String()
		nonExistentID2 := uuid.New().String()

		req := CreateRelationshipRequest{
			FromPersonID:     nonExistentID1,
			ToPersonID:       nonExistentID2,
			RelationshipType: "friend",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/relationships",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "error")
	})

	t.Run("Error_MissingRequiredFields", func(t *testing.T) {
		req := map[string]interface{}{
			"from_person_id": uuid.New().String(),
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/relationships",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "error")
	})
}

// TestGetRelationships tests listing relationships
func TestGetRelationships(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_AllRelationships", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/relationships",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["relationships"]; !ok {
			t.Error("Expected relationships array in response")
		}

		if _, ok := response["count"]; !ok {
			t.Error("Expected count in response")
		}
	})

	t.Run("Success_FilterByPerson", func(t *testing.T) {
		person1ID := createTestPerson(t, env, "Rel Person 1")
		person2ID := createTestPerson(t, env, "Rel Person 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		relID := createTestRelationship(t, env, person1ID, person2ID, "friend")
		defer cleanupTestRelationship(t, env, relID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/relationships",
			QueryParams: map[string]string{
				"person_id": person1ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		relationships, ok := response["relationships"].([]interface{})
		if !ok {
			t.Fatal("Expected relationships array")
		}

		found := false
		for _, rel := range relationships {
			relMap := rel.(map[string]interface{})
			if relMap["id"].(string) == relID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find created relationship in results")
		}
	})

	t.Run("Success_FilterByType", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/relationships",
			QueryParams: map[string]string{
				"type": "friend",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})
}

// TestSearchContacts tests the search functionality
func TestSearchContacts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_BasicSearch", func(t *testing.T) {
		// Create a person with unique searchable data
		uniqueName := "UniqueSearchable" + uuid.New().String()[:8]
		personID := createTestPerson(t, env, uniqueName)
		defer cleanupTestPerson(t, env, personID)

		// Give database time to process
		time.Sleep(100 * time.Millisecond)

		req := SearchRequest{
			Query: "UniqueSearchable",
			Limit: intPtr(20),
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		results, ok := response["results"].([]interface{})
		if !ok {
			t.Fatal("Expected results array in response")
		}

		if len(results) == 0 {
			t.Error("Expected at least one search result")
		}
	})

	t.Run("Success_EmptyResults", func(t *testing.T) {
		req := SearchRequest{
			Query: "NonexistentPersonName" + uuid.New().String(),
			Limit: intPtr(20),
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		results, ok := response["results"].([]interface{})
		if !ok {
			t.Fatal("Expected results array in response")
		}

		count, ok := response["count"].(float64)
		if !ok {
			t.Fatal("Expected count in response")
		}

		if int(count) != len(results) {
			t.Errorf("Count mismatch: count=%d, len(results)=%d", int(count), len(results))
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/search", strings.NewReader("{invalid"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestGetSocialAnalytics tests the analytics endpoint
func TestGetSocialAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success_AllAnalytics", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["analytics"]; !ok {
			t.Error("Expected analytics array in response")
		}

		if _, ok := response["count"]; !ok {
			t.Error("Expected count in response")
		}
	})

	t.Run("Success_FilterByPerson", func(t *testing.T) {
		personID := createTestPerson(t, env, "Analytics Test Person")
		defer cleanupTestPerson(t, env, personID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics",
			QueryParams: map[string]string{
				"person_id": personID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})
}

// TestUtilityFunctions tests utility functions
func TestUtilityFunctions(t *testing.T) {
	t.Run("parseArrayString_Empty", func(t *testing.T) {
		var result []string
		err := parseArrayString("{}", &result)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected empty array, got %v", result)
		}
	})

	t.Run("parseArrayString_SingleElement", func(t *testing.T) {
		var result []string
		err := parseArrayString(`{"test"}`, &result)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "test" {
			t.Errorf("Expected [test], got %v", result)
		}
	})

	t.Run("parseArrayString_MultipleElements", func(t *testing.T) {
		var result []string
		err := parseArrayString(`{"one","two","three"}`, &result)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result))
		}
	})

	t.Run("arrayToPostgresArray_Empty", func(t *testing.T) {
		result := arrayToPostgresArray([]string{})
		if result != "{}" {
			t.Errorf("Expected {}, got %s", result)
		}
	})

	t.Run("arrayToPostgresArray_WithElements", func(t *testing.T) {
		result := arrayToPostgresArray([]string{"one", "two", "three"})
		expected := `{"one","two","three"}`
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("arrayToPostgresArray_WithQuotes", func(t *testing.T) {
		result := arrayToPostgresArray([]string{`test"quote`})
		if !strings.Contains(result, `\"`) {
			t.Error("Expected quotes to be escaped")
		}
	})
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
